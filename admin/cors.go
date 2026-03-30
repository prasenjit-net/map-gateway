package admin

import (
	"net/http"
	"strings"

	"github.com/prasenjit-net/mcp-gateway/config"
)

// newCORSMiddleware returns a middleware factory that applies CORS headers
// based on the configured allowed origins. If no origins are configured,
// CORS headers are not set (same-origin only).
func newCORSMiddleware(cfg *config.Config) func(http.HandlerFunc) http.HandlerFunc {
	methods := strings.Join(cfg.CORS.AllowedMethods, ", ")
	headers := strings.Join(cfg.CORS.AllowedHeaders, ", ")

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && len(cfg.CORS.AllowedOrigins) > 0 {
				if isOriginAllowed(origin, cfg.CORS.AllowedOrigins) {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Methods", methods)
					w.Header().Set("Access-Control-Allow-Headers", headers)
					w.Header().Add("Vary", "Origin")
				}
			}
			next(w, r)
		}
	}
}

// newCORSPreflightHandler returns a handler for OPTIONS preflight requests.
func newCORSPreflightHandler(cfg *config.Config) http.HandlerFunc {
	methods := strings.Join(cfg.CORS.AllowedMethods, ", ")
	headers := strings.Join(cfg.CORS.AllowedHeaders, ", ")

	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && len(cfg.CORS.AllowedOrigins) > 0 {
			if isOriginAllowed(origin, cfg.CORS.AllowedOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", methods)
				w.Header().Set("Access-Control-Allow-Headers", headers)
				w.Header().Add("Vary", "Origin")
			}
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func isOriginAllowed(origin string, allowed []string) bool {
	for _, a := range allowed {
		if a == "*" || a == origin {
			return true
		}
	}
	return false
}
