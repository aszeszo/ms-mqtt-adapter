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
		prefix := r.Header.Get("X-Ingress-Path")

		if prefix != "" {
			r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
			if r.URL.Path == "" {
				r.URL.Path = "/"
			}
		}

		// Always log to help debug ingress issues
		logger.Debug("HTTP request",
			"original_path", originalPath,
			"ingress_prefix", prefix,
			"rewritten_path", r.URL.Path,
			"method", r.Method,
			"has_ingress", prefix != "")

		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Note: ingressMiddleware already logs requests with ingress details
		// This is a fallback for non-ingress requests or additional logging
		next.ServeHTTP(w, r)
	})
}
