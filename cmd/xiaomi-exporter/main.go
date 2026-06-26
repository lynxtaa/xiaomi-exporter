// Package main runs the xiaomi-exporter.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lynxtaa/xiaomi-exporter/internal/application"
	"github.com/lynxtaa/xiaomi-exporter/internal/config"
	"github.com/lynxtaa/xiaomi-exporter/internal/devicescanner/xiaomi"
	httpserver "github.com/lynxtaa/xiaomi-exporter/internal/http"
	"github.com/lynxtaa/xiaomi-exporter/internal/reqid"
)

const (
	shutdownTimeout   = 10 * time.Second
	readHeaderTimeout = 10 * time.Second
)

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load(ctx)

	if err != nil {
		logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
		slog.SetDefault(logger)
		slog.Error("loading config", "error", err)
		return err
	}

	logger := slog.New(reqid.Handler(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	})))
	slog.SetDefault(logger)

	devices := make([]*xiaomi.Device, len(cfg.DeviceNames))
	for i := range cfg.DeviceNames {
		devices[i] = xiaomi.NewDevice(cfg.DeviceNames[i], cfg.DeviceMacs[i], cfg.DeviceBindKeys[i])
	}
	scanner := xiaomi.NewScanner(devices...)

	app := application.New(cfg, scanner)
	httpServer := httpserver.NewServer(app)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpServer.Handler(),
		ReadHeaderTimeout: readHeaderTimeout,
	}

	errCh := make(chan error, 2)
	go func() {
		slog.InfoContext(ctx, "http server listening", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	go func() {
		slog.InfoContext(ctx, "starting background BLE scanner")
		if err := app.ScanDevices.Handle(ctx); err != nil && !errors.Is(err, context.Canceled) {
			errCh <- fmt.Errorf("scanner: %w", err)
		}
	}()

	select {
	case err := <-errCh:
		slog.ErrorContext(ctx, "http server", "error", err)
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "xiaomi-exporter: %v\n", err)
		os.Exit(1)
	}
}
