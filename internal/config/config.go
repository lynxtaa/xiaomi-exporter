// Package config loads typed configuration from environment variables.
package config

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

var macAddrR = regexp.MustCompile(`^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$`)

// Config contains app configuration. Device fields are parallel comma-separated
// lists: the Nth name, MAC and bind key describe the same device.
type Config struct {
	DeviceNames    []string   `env:"DEVICE_NAMES,notEmpty"`
	DeviceMacs     []string   `env:"DEVICE_MACS,notEmpty"`
	DeviceBindKeys []string   `env:"DEVICE_BIND_KEYS,notEmpty"`
	HTTPAddr       string     `env:"HTTP_ADDR" envDefault:":8080"`
	LogLevel       slog.Level `env:"LOG_LEVEL" envDefault:"info"`

	// ScanInterval is how often a scan window starts.
	ScanInterval time.Duration `env:"SCAN_INTERVAL" envDefault:"5m"`
	// ScanWindow is how long each scan keeps the radio on. Short windows keep
	// the sensors' battery drain negligible; gauges retain values between them.
	ScanWindow time.Duration `env:"SCAN_WINDOW" envDefault:"90s"`
}

// Load parses environment variables (and an optional .env file) into a new T.
func Load(ctx context.Context) (*Config, error) {
	if err := godotenv.Load(); err != nil {
		slog.DebugContext(ctx, "No .env file found, relying on system environment variables")
	}

	cfg := &Config{}
	err := env.Parse(cfg)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	return cfg, validate(cfg)
}

func validate(cfg *Config) error {
	if len(cfg.DeviceNames) != len(cfg.DeviceMacs) || len(cfg.DeviceNames) != len(cfg.DeviceBindKeys) {
		return fmt.Errorf(
			"DEVICE_NAMES, DEVICE_MACS and DEVICE_BIND_KEYS must have equal lengths (%d/%d/%d)",
			len(cfg.DeviceNames), len(cfg.DeviceMacs), len(cfg.DeviceBindKeys),
		)
	}
	for _, mac := range cfg.DeviceMacs {
		if !macAddrR.MatchString(mac) {
			return fmt.Errorf("invalid MAC address %q", mac)
		}
	}
	return nil
}
