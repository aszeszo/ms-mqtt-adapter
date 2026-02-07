package api

import (
	"log/slog"
	"net/http"
	"strings"
)

// ingressMiddleware strips the X-Ingress-Path prefix from the request URL
// so that the router sees clean paths.
func ingressMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		originalPath := r.URL.Path
		if prefix := r.Header.Get("X-Ingress-Path"); prefix != "" {
			r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
			if r.URL.Path == "" {
				r.URL.Path = "/"
			}
			logger.Debug("Ingress path rewrite",
				"original_path", originalPath,
				"ingress_prefix", prefix,
				"rewritten_path", r.URL.Path)
		}
		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Debug("HTTP request", "method", r.Method, "path", r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
