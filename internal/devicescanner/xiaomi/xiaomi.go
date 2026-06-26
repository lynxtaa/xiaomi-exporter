// Package xiaomi reads temperature and humidity from Xiaomi BLE sensors
// broadcasting encrypted MiBeacon advertisements (stock firmware).
package xiaomi

import (
	"context"
	"crypto/aes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/lynxtaa/xiaomi-exporter/internal/devicescanner"

	ccm "gitlab.com/go-extension/aes-ccm"
	"tinygo.org/x/bluetooth"
)

// Device describes a Xiaomi sensor on stock firmware (MiBeacon).
type Device struct {
	Name    string
	Mac     string
	BindKey string
}

// NewDevice creates a device; the MAC is normalized to upper case.
func NewDevice(name, mac, bindKey string) *Device {
	return &Device{
		Name:    name,
		Mac:     strings.ToUpper(mac),
		BindKey: bindKey,
	}
}

// Scanner contains data for scanning devices via Bluetooth
type Scanner struct {
	enableOnce     sync.Once
	enableErr      error
	miServiceUUID  bluetooth.UUID
	deviceRegistry map[string]*Device
}

var _ devicescanner.Scanner = (*Scanner)(nil)

// NewScanner returns a pointer to Scanner
func NewScanner(devices ...*Device) *Scanner {
	registry := make(map[string]*Device, len(devices))
	for _, d := range devices {
		registry[d.Mac] = d
	}

	return &Scanner{
		miServiceUUID:  bluetooth.New16BitUUID(0xFE95),
		deviceRegistry: registry,
	}
}

// Run scans in a window of length window every interval, invoking onReading for
// each decoded packet, until ctx is cancelled. The first window starts
// immediately. Scanning in short periodic windows keeps the sensors' battery
// drain negligible while still refreshing metrics every interval.
func (s *Scanner) Run(
	ctx context.Context,
	interval, window time.Duration,
	onReading devicescanner.ReadingFunc,
) error {
	adapter := bluetooth.DefaultAdapter

	s.enableOnce.Do(func() { s.enableErr = adapter.Enable() })
	if s.enableErr != nil {
		return fmt.Errorf("enable BLE adapter: %w", s.enableErr)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if err := s.scanWindow(ctx, adapter, window, onReading); err != nil {
			slog.ErrorContext(ctx, "scan window failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// scanWindow scans for at most window, dispatching every matching packet to onReading.
func (s *Scanner) scanWindow(
	ctx context.Context,
	adapter *bluetooth.Adapter,
	window time.Duration,
	onReading devicescanner.ReadingFunc,
) error {
	scanCtx, cancel := context.WithTimeout(ctx, window)
	defer cancel()

	go func() {
		<-scanCtx.Done()
		_ = adapter.StopScan()
	}()

	err := adapter.Scan(func(_ *bluetooth.Adapter, result bluetooth.ScanResult) {
		mac := strings.ToUpper(result.Address.String())
		d, ok := s.deviceRegistry[mac]
		if !ok {
			return
		}

		for _, sd := range result.ServiceData() {
			if sd.UUID != s.miServiceUUID {
				continue
			}
			r, err := decode(d, sd.Data)
			if err != nil {
				slog.DebugContext(ctx, "failed to decode payload", "mac", mac, "error", err)
				continue
			}
			slog.InfoContext(ctx, "metrics decoded", attrs(d, r)...)
			onReading(devicescanner.Device{Name: d.Name, Mac: d.Mac}, r)
		}
	})
	// Scan returns nil once StopScan is called (window elapsed or ctx cancelled).
	if err != nil && scanCtx.Err() == nil {
		return fmt.Errorf("scan: %w", err)
	}
	return nil
}

// decode decrypts one packet's MiBeacon (0xFE95) service data and parses the object.
func decode(d *Device, data []byte) (devicescanner.Reading, error) {
	// MiBeacon v5: FrameCtrl(2,LE) + ProductID(2,LE) + FrameCnt(1) + [MAC(6)] + [Capability(1)] + payload
	if len(data) < 5 {
		return devicescanner.Reading{}, fmt.Errorf("service data too short")
	}

	frameCtrl := binary.LittleEndian.Uint16(data[0:2])
	const (
		flagEncrypted  = 1 << 3
		flagMacInclude = 1 << 4
		flagCapInclude = 1 << 5
	)

	i := 5
	var mac []byte
	if frameCtrl&flagMacInclude != 0 {
		if len(data) < i+6 {
			return devicescanner.Reading{}, fmt.Errorf("payload too short for mac")
		}
		mac = data[i : i+6]
		i += 6
	}
	if frameCtrl&flagCapInclude != 0 {
		i++ // skip the capability byte
	}
	if len(data) < i {
		return devicescanner.Reading{}, fmt.Errorf("payload too short")
	}
	payload := data[i:]

	if frameCtrl&flagEncrypted == 0 {
		return parseObject(payload), nil
	}

	if mac == nil {
		return devicescanner.Reading{}, fmt.Errorf("encrypted frame without mac")
	}
	if len(payload) < 7 {
		return devicescanner.Reading{}, fmt.Errorf("encrypted payload too short")
	}

	key, err := hex.DecodeString(d.BindKey)
	if err != nil {
		return devicescanner.Reading{}, fmt.Errorf("invalid bind key format: %w", err)
	}

	cipherText := payload[:len(payload)-7]
	extCounter := payload[len(payload)-7 : len(payload)-4]
	mic := payload[len(payload)-4:]

	// Nonce (12 bytes): MAC(6) + ProductID(2) + FrameCnt(1) + ExtCounter(3)
	nonce := make([]byte, 0, 12)
	nonce = append(nonce, mac...)
	nonce = append(nonce, data[2:5]...)
	nonce = append(nonce, extCounter...)

	block, err := aes.NewCipher(key)
	if err != nil {
		return devicescanner.Reading{}, err
	}

	// MiBeacon: 12-byte nonce, 4-byte MIC (tag)
	aesccm, err := ccm.NewCCMWithSize(block, 12, 4)
	if err != nil {
		return devicescanner.Reading{}, err
	}

	cipherTextAndMic := make([]byte, 0, len(cipherText)+len(mic))
	cipherTextAndMic = append(cipherTextAndMic, cipherText...)
	cipherTextAndMic = append(cipherTextAndMic, mic...)

	// AAD for Xiaomi is always 0x11
	plainText, err := aesccm.Open(nil, nonce, cipherTextAndMic, []byte{0x11})
	if err != nil {
		return devicescanner.Reading{}, fmt.Errorf("decryption failed (wrong key?): %w", err)
	}

	return parseObject(plainText), nil
}

// parseObject parses a decrypted MiBeacon object: Type(2,LE) + Length(1) + Value.
func parseObject(payload []byte) devicescanner.Reading {
	if len(payload) < 3 {
		return devicescanner.Reading{}
	}

	objType := binary.LittleEndian.Uint16(payload[0:2])
	length := int(payload[2])
	if len(payload) < 3+length {
		return devicescanner.Reading{}
	}
	value := payload[3 : 3+length]

	var r devicescanner.Reading
	switch objType {
	case 0x1004: // temperature (signed, tenths of a degree)
		if len(value) >= 2 {
			t := float64(int16(binary.LittleEndian.Uint16(value))) / 10.0 //nolint:gosec
			r.Temperature = &t
		}
	case 0x1006: // humidity
		if len(value) >= 2 {
			h := float64(binary.LittleEndian.Uint16(value)) / 10.0
			r.Humidity = &h
		}
	}
	return r
}

func attrs(d *Device, r devicescanner.Reading) []any {
	a := []any{"name", d.Name, "mac", d.Mac}
	if r.Temperature != nil {
		a = append(a, "temp", *r.Temperature)
	}
	if r.Humidity != nil {
		a = append(a, "hum", *r.Humidity)
	}
	return a
}
