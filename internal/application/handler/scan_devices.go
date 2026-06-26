// Package handler provides use cases
package handler

import (
	"context"
	"time"

	"github.com/lynxtaa/xiaomi-exporter/internal/config"
	"github.com/lynxtaa/xiaomi-exporter/internal/devicescanner"
	"github.com/lynxtaa/xiaomi-exporter/internal/metrics"
)

// ScanDevicesHandler scans the configured sensors in the background and records every
// decoded reading into the Prometheus gauges as packets arrive.
type ScanDevicesHandler struct {
	scanner  devicescanner.Scanner
	interval time.Duration
	window   time.Duration
}

// NewScanDevicesHandler creates a ScanService from the configured devices.
func NewScanDevicesHandler(cfg *config.Config, scanner devicescanner.Scanner) *ScanDevicesHandler {
	return &ScanDevicesHandler{
		scanner:  scanner,
		interval: cfg.ScanInterval,
		window:   cfg.ScanWindow,
	}
}

// Handle scans in periodic windows until ctx is cancelled, updating gauges as
// readings arrive. Each gauge is only set once its value has actually been
// received, so missing metrics are never published as zero.
func (s *ScanDevicesHandler) Handle(ctx context.Context) error {
	return s.scanner.Run(ctx, s.interval, s.window, record)
}

func record(d devicescanner.Device, r devicescanner.Reading) {
	if r.Temperature != nil {
		metrics.TempGauge.WithLabelValues(d.Mac, d.Name).Set(*r.Temperature)
	}
	if r.Humidity != nil {
		metrics.HumGauge.WithLabelValues(d.Mac, d.Name).Set(*r.Humidity)
	}
}
