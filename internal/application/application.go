// Package application contains application
package application

import (
	"github.com/lynxtaa/xiaomi-exporter/internal/application/handler"
	"github.com/lynxtaa/xiaomi-exporter/internal/config"
	"github.com/lynxtaa/xiaomi-exporter/internal/devicescanner"
)

// App is application
type App struct {
	ScanDevices *handler.ScanDevicesHandler
}

// New creates new application
func New(cfg *config.Config, scanner devicescanner.Scanner) *App {
	return &App{
		ScanDevices: handler.NewScanDevicesHandler(cfg, scanner),
	}
}
