// Package http handles server creation
package http

import (
	"net/http"

	"github.com/lynxtaa/xiaomi-exporter/internal/application"
	"github.com/lynxtaa/xiaomi-exporter/internal/httplog"
	"github.com/lynxtaa/xiaomi-exporter/internal/metrics"
	"github.com/lynxtaa/xiaomi-exporter/internal/reqid"
)

// Server represents the HTTP server.
type Server struct {
	mux         *http.ServeMux
	application *application.App
}

// NewServer creates a new HTTP server with all routes configured.
func NewServer(application *application.App) *Server {
	s := &Server{
		mux:         http.NewServeMux(),
		application: application,
	}

	// Scanning happens in the background; /metrics just serves the latest gauges.
	s.mux.Handle("GET /metrics", metrics.Handler())

	return s
}

// Handler returns the HTTP Handler with middleware applied.
func (s *Server) Handler() http.Handler {
	handler := http.Handler(s.mux)
	handler = httplog.Middleware(handler)
	handler = reqid.Middleware(handler)

	return handler
}
