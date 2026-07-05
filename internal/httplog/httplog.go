// Package httplog provides HTTP middleware that logs incoming requests via slog.
package httplog

import (
	"log/slog"
	"net/http"
)

// Middleware logs each incoming request.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.InfoContext(r.Context(), "incoming request",
			"method", r.Method,
			"path", r.URL.Path,
			"query", r.URL.Query(),
			"headers", r.Header,
			"remoteAddr", r.RemoteAddr,
		)
		next.ServeHTTP(w, r)
	})
}
