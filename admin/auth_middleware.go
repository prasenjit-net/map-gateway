package admin

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/prasenjit-net/mcp-gateway/config"
)

// newAuthMiddleware returns a middleware that enforces admin authentication.
// When AdminPassword is empty (dev mode) authentication is skipped and a
// warning is surfaced via the X-Auth-Warning response header.
// For unauthenticated requests:
//   - /_api/* returns 401 JSON
//   - all other paths redirect to /_auth/login?redirect=<path>
func newAuthMiddleware(cfg *config.Config) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return authHandler(cfg, next)
	}
}

// UIAuthMiddleware wraps a UI handler with admin authentication.
// This is exported so cmd/serve.go can protect the /_ui/ route.
func UIAuthMiddleware(cfg *config.Config, next http.HandlerFunc) http.HandlerFunc {
	return authHandler(cfg, next)
}

func authHandler(cfg *config.Config, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cfg.AdminPassword == "" {
			// Auth disabled — dev mode.
			w.Header().Set("X-Auth-Warning", "admin authentication is disabled; set ADMIN_PASSWORD")
			next(w, r)
			return
		}
		secret := cfg.GatewaySecret
		if secret == "" {
			secret = cfg.AdminPassword // fallback signing key when no gateway secret set
		}
		if validateSession(r, secret) {
			next(w, r)
			return
		}
		// Reject or redirect.
		if strings.HasPrefix(r.URL.Path, "/_api/") {
			jsonError(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		loginURL := "/_auth/login?redirect=" + url.QueryEscape(r.URL.RequestURI())
		http.Redirect(w, r, loginURL, http.StatusSeeOther)
	}
}
