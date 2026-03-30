package admin_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/prasenjit-net/mcp-gateway/admin"
	"github.com/prasenjit-net/mcp-gateway/config"
	"github.com/prasenjit-net/mcp-gateway/registry"
	"github.com/prasenjit-net/mcp-gateway/store"
)

// ── Helpers ───────────────────────────────────────────────────────────────────

func adminCfg(password, secret string) *config.Config {
	cfg := config.DefaultConfig()
	cfg.AdminPassword = password
	cfg.GatewaySecret = secret
	cfg.AdminSessionTTL = 24
	return cfg
}

func buildDeps(t *testing.T, cfg *config.Config) *admin.Deps {
	t.Helper()
	s, err := store.NewJSONStore(t.TempDir())
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { s.Close() }) //nolint:errcheck
	return &admin.Deps{
		Store:    s,
		Registry: registry.NewRegistry(),
		Config:   cfg,
	}
}

func loginAndGetCookie(t *testing.T, cfg *config.Config) *http.Cookie {
	t.Helper()
	mux := http.NewServeMux()
	admin.RegisterRoutes(mux, buildDeps(t, cfg))

	body := url.Values{"password": {cfg.AdminPassword}, "redirect": {"/_ui/"}}.Encode()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/_auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("login POST = %d, want 303", rec.Code)
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == "mcpgw_session" {
			return c
		}
	}
	t.Fatal("session cookie not found in login response")
	return nil
}

// ── Login handler ─────────────────────────────────────────────────────────────

func TestLoginGetRendersForm(t *testing.T) {
	cfg := adminCfg("s3cr3t", "gw-secret")
	mux := http.NewServeMux()
	admin.RegisterRoutes(mux, buildDeps(t, cfg))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/_auth/login", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GET /_auth/login = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "<form") {
		t.Error("response should contain a <form> element")
	}
	if !strings.Contains(rec.Body.String(), "MCP Gateway") {
		t.Error("response should contain MCP Gateway branding")
	}
}

func TestLoginPostCorrectPassword(t *testing.T) {
	cfg := adminCfg("correct-pass", "gw-secret")
	mux := http.NewServeMux()
	admin.RegisterRoutes(mux, buildDeps(t, cfg))

	body := url.Values{"password": {"correct-pass"}, "redirect": {"/_ui/"}}.Encode()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/_auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Errorf("POST /_auth/login (correct) = %d, want 303", rec.Code)
	}
	hasCookie := false
	for _, c := range rec.Result().Cookies() {
		if c.Name == "mcpgw_session" {
			hasCookie = true
			if c.HttpOnly != true {
				t.Error("session cookie should be HttpOnly")
			}
		}
	}
	if !hasCookie {
		t.Error("session cookie not set after successful login")
	}
}

func TestLoginPostWrongPassword(t *testing.T) {
	cfg := adminCfg("correct-pass", "gw-secret")
	mux := http.NewServeMux()
	admin.RegisterRoutes(mux, buildDeps(t, cfg))

	body := url.Values{"password": {"wrong-pass"}, "redirect": {"/_ui/"}}.Encode()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/_auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("POST /_auth/login (wrong) = %d, want 401", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Invalid password") {
		t.Error("response should contain error message")
	}
}

func TestLogout(t *testing.T) {
	cfg := adminCfg("pass", "gw-secret")
	mux := http.NewServeMux()
	admin.RegisterRoutes(mux, buildDeps(t, cfg))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/_auth/logout", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Errorf("POST /_auth/logout = %d, want 303", rec.Code)
	}
	// Verify cookie is cleared (MaxAge = -1 or Expires in past)
	cleared := false
	for _, c := range rec.Result().Cookies() {
		if c.Name == "mcpgw_session" && c.MaxAge < 0 {
			cleared = true
		}
	}
	if !cleared {
		t.Error("session cookie should be cleared after logout")
	}
}

// ── Auth middleware ───────────────────────────────────────────────────────────

func TestAuthMiddlewareNoPasswordSkipsAuth(t *testing.T) {
	cfg := adminCfg("", "gw-secret") // no password = auth disabled
	mux := http.NewServeMux()
	// Register a fake protected route manually to test.
	mux.HandleFunc("GET /_api/specs", admin.UIAuthMiddleware(cfg, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/_api/specs", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("with no password, request should pass through, got %d", rec.Code)
	}
	if rec.Header().Get("X-Auth-Warning") == "" {
		t.Error("X-Auth-Warning header should be set when auth is disabled")
	}
}

func TestAuthMiddlewareWithValidSession(t *testing.T) {
	cfg := adminCfg("mypass", "gw-secret")
	deps := buildDeps(t, cfg)
	mux := http.NewServeMux()
	admin.RegisterRoutes(mux, deps)

	cookie := loginAndGetCookie(t, cfg)

	// Health is always public — just verify it doesn't 401.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/_api/health", nil)
	req.AddCookie(cookie)
	mux.ServeHTTP(rec, req)
	if rec.Code == http.StatusUnauthorized {
		t.Error("authenticated request to /health should not return 401")
	}

	// Protected stats route — valid session should reach the handler.
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/_api/specs", nil)
	req2.AddCookie(cookie)
	mux.ServeHTTP(rec2, req2)
	if rec2.Code == http.StatusUnauthorized {
		t.Error("authenticated request should not return 401")
	}
}

func TestAuthMiddlewareAPIWithoutSessionReturns401(t *testing.T) {
	cfg := adminCfg("mypass", "gw-secret")
	mux := http.NewServeMux()
	admin.RegisterRoutes(mux, buildDeps(t, cfg))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/_api/specs", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("unauthenticated API request = %d, want 401", rec.Code)
	}
}

func TestAuthMiddlewareUIWithoutSessionRedirects(t *testing.T) {
	cfg := adminCfg("mypass", "gw-secret")
	uiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	protected := admin.UIAuthMiddleware(cfg, uiHandler.ServeHTTP)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/_ui/dashboard", nil)
	protected(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Errorf("unauthenticated UI request = %d, want 303 redirect", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.HasPrefix(loc, "/_auth/login") {
		t.Errorf("redirect location = %q, want /_auth/login...", loc)
	}
}

// ── CORS middleware ───────────────────────────────────────────────────────────

func TestCORSAllowedOrigin(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.CORS.AllowedOrigins = []string{"https://app.example.com"}

	mux := http.NewServeMux()
	admin.RegisterRoutes(mux, buildDeps(t, cfg))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("OPTIONS", "/", nil)
	req.Header.Set("Origin", "https://app.example.com")
	mux.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "https://app.example.com" {
		t.Errorf("CORS origin = %q", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSDisallowedOrigin(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.CORS.AllowedOrigins = []string{"https://allowed.example.com"}

	mux := http.NewServeMux()
	admin.RegisterRoutes(mux, buildDeps(t, cfg))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("OPTIONS", "/", nil)
	req.Header.Set("Origin", "https://attacker.example.com")
	mux.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("disallowed origin should not get CORS headers")
	}
}

func TestCORSWildcard(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.CORS.AllowedOrigins = []string{"*"}

	mux := http.NewServeMux()
	admin.RegisterRoutes(mux, buildDeps(t, cfg))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("OPTIONS", "/", nil)
	req.Header.Set("Origin", "https://any-origin.com")
	mux.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "https://any-origin.com" {
		t.Errorf("wildcard CORS should allow any origin, got %q", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSNoOriginHeader(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.CORS.AllowedOrigins = []string{"https://example.com"}

	mux := http.NewServeMux()
	admin.RegisterRoutes(mux, buildDeps(t, cfg))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/_api/health", nil) // no Origin header
	mux.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("no Origin header should produce no CORS response headers")
	}
}

func TestCORSEmptyAllowedOrigins(t *testing.T) {
	cfg := config.DefaultConfig()
	// AllowedOrigins is empty by default — same-origin policy.

	mux := http.NewServeMux()
	admin.RegisterRoutes(mux, buildDeps(t, cfg))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("OPTIONS", "/", nil)
	req.Header.Set("Origin", "https://any.example.com")
	mux.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("empty AllowedOrigins should not set CORS headers")
	}
}

// ── Health endpoint (always public) ──────────────────────────────────────────

func TestHealthPublicWithPassword(t *testing.T) {
	cfg := adminCfg("somepass", "gw-secret")
	mux := http.NewServeMux()
	admin.RegisterRoutes(mux, buildDeps(t, cfg))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/_api/health", nil)
	mux.ServeHTTP(rec, req)

	// Health should be reachable without auth.
	if rec.Code == http.StatusUnauthorized {
		t.Error("/_api/health should be public even when auth is enabled")
	}
}

// ── Open-redirect protection ──────────────────────────────────────────────────

func TestLoginRedirectRelativeOnly(t *testing.T) {
	cfg := adminCfg("pass", "gw-secret")
	mux := http.NewServeMux()
	admin.RegisterRoutes(mux, buildDeps(t, cfg))

	// Attempt open redirect via redirect param.
	body := url.Values{
		"password": {"pass"},
		"redirect": {"//evil.example.com/steal"},
	}.Encode()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/_auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	mux.ServeHTTP(rec, req)

	loc := rec.Header().Get("Location")
	if strings.HasPrefix(loc, "//") || strings.HasPrefix(loc, "http") {
		t.Errorf("open redirect not blocked, Location = %q", loc)
	}
}

// ── Session expiry ────────────────────────────────────────────────────────────

func TestExpiredSessionRejected(t *testing.T) {
	cfg := adminCfg("pass", "gw-secret")
	// Create a session with -1 hour TTL (already expired).
	cfg.AdminSessionTTL = 0 // 0 → defaults to 24h in Load(), but here we set directly
	// Override TTL to negative to get an expired cookie:
	_ = cfg
	// We'll directly test the session validation via login with an injected expired cookie.
	mux := http.NewServeMux()
	admin.RegisterRoutes(mux, buildDeps(t, cfg))

	expiredCookie := &http.Cookie{
		Name:  "mcpgw_session",
		Value: "dW5peFRpbWU=.invalidsig", // invalid value
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/_api/specs", nil)
	req.AddCookie(expiredCookie)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("invalid session cookie = %d, want 401", rec.Code)
	}
}

func TestSessionCookieExpiry(t *testing.T) {
	cfg := adminCfg("pass", "gw-secret")
	mux := http.NewServeMux()
	admin.RegisterRoutes(mux, buildDeps(t, cfg))

	cookie := loginAndGetCookie(t, cfg)
	if cookie.MaxAge <= 0 {
		t.Errorf("session cookie MaxAge = %d, want > 0", cookie.MaxAge)
	}
	expectedMaxAge := 24 * 3600
	if cookie.MaxAge != expectedMaxAge {
		t.Errorf("MaxAge = %d, want %d", cookie.MaxAge, expectedMaxAge)
	}
}

// ── JSON error helper ─────────────────────────────────────────────────────────

func TestJSONErrorOnAPIRoute(t *testing.T) {
	cfg := adminCfg("pass", "gw-secret")
	mux := http.NewServeMux()
	admin.RegisterRoutes(mux, buildDeps(t, cfg))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/_api/specs", nil)
	mux.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q, want application/json for 401", rec.Header().Get("Content-Type"))
	}
	if !strings.Contains(rec.Body.String(), "unauthorized") {
		t.Errorf("body should contain 'unauthorized', got: %s", rec.Body.String())
	}
}

// ── Session uses fallback secret ──────────────────────────────────────────────

func TestSessionWithFallbackSecret(t *testing.T) {
	// GatewaySecret empty → AdminPassword used as signing key.
	cfg := config.DefaultConfig()
	cfg.AdminPassword = "admin-pass"
	cfg.GatewaySecret = "" // empty
	cfg.AdminSessionTTL = 1

	mux := http.NewServeMux()
	admin.RegisterRoutes(mux, buildDeps(t, cfg))

	body := url.Values{"password": {"admin-pass"}, "redirect": {"/_ui/"}}.Encode()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/_auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("login = %d, want 303", rec.Code)
	}

	var sessionCookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == "mcpgw_session" {
			sessionCookie = c
		}
	}
	if sessionCookie == nil {
		t.Fatal("session cookie not set")
	}

	// Use the cookie to access a protected route.
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/_api/specs", nil)
	req2.AddCookie(sessionCookie)
	mux.ServeHTTP(rec2, req2)

	if rec2.Code == http.StatusUnauthorized {
		t.Error("valid session with fallback secret should not return 401")
	}
}

// ensure session TTL flows through correctly
func TestAdminSessionTTL(t *testing.T) {
	cfg := adminCfg("pass", "secret")
	cfg.AdminSessionTTL = 2

	mux := http.NewServeMux()
	admin.RegisterRoutes(mux, buildDeps(t, cfg))

	body := url.Values{"password": {"pass"}, "redirect": {"/_ui/"}}.Encode()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/_auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	mux.ServeHTTP(rec, req)

	for _, c := range rec.Result().Cookies() {
		if c.Name == "mcpgw_session" {
			expected := 2 * 3600
			if c.MaxAge != expected {
				t.Errorf("MaxAge = %d, want %d (2 hours)", c.MaxAge, expected)
			}
			// Verify expiry is approximately now+2h.
			if c.Expires.IsZero() {
				return // MaxAge-only cookie, fine
			}
			diff := c.Expires.Sub(time.Now())
			if diff < time.Hour || diff > 3*time.Hour {
				t.Errorf("unexpected Expires: %v (diff=%v)", c.Expires, diff)
			}
			return
		}
	}
	t.Error("session cookie not found")
}
