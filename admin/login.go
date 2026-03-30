package admin

import (
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	"github.com/prasenjit-net/mcp-gateway/config"
)

type loginHandler struct {
	config *config.Config
}

// ServeLogin handles GET (render form) and POST (validate + set cookie) for /_auth/login.
func (h *loginHandler) ServeLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		h.handlePost(w, r)
		return
	}
	redirect := r.URL.Query().Get("redirect")
	if redirect == "" || !isSafeRedirect(redirect) {
		redirect = "/_ui/"
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	fmt.Fprint(w, loginPage(redirect, ""))
}

// ServeLogout clears the session cookie and redirects to the login page.
func (h *loginHandler) ServeLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
	http.Redirect(w, r, "/_auth/login", http.StatusSeeOther)
}

func (h *loginHandler) handlePost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	password := r.FormValue("password")
	redirect := r.FormValue("redirect")
	if redirect == "" || !isSafeRedirect(redirect) {
		redirect = "/_ui/"
	}

	if password != h.config.AdminPassword {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, loginPage(redirect, "Invalid password. Please try again."))
		return
	}

	secret := h.config.GatewaySecret
	if secret == "" {
		secret = h.config.AdminPassword
	}
	ttl := time.Duration(h.config.AdminSessionTTL) * time.Hour
	secure := strings.HasPrefix(r.Header.Get("X-Forwarded-Proto"), "https") ||
		r.TLS != nil
	http.SetCookie(w, createSessionCookie(secret, ttl, secure))
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

// isSafeRedirect ensures the redirect target is a relative path (prevents open redirect).
func isSafeRedirect(target string) bool {
	return strings.HasPrefix(target, "/") && !strings.HasPrefix(target, "//")
}

func loginPage(redirect, errMsg string) string {
	errHTML := ""
	if errMsg != "" {
		errHTML = fmt.Sprintf(`<div class="error">%s</div>`, html.EscapeString(errMsg))
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>MCP Gateway — Sign in</title>
  <style>
    *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
      background: #0f172a;
      color: #e2e8f0;
      display: flex;
      align-items: center;
      justify-content: center;
      min-height: 100vh;
    }
    .card {
      background: #1e293b;
      border: 1px solid #334155;
      border-radius: 12px;
      padding: 2.5rem;
      width: 100%%;
      max-width: 380px;
      box-shadow: 0 25px 50px -12px rgba(0,0,0,0.5);
    }
    .logo {
      text-align: center;
      margin-bottom: 2rem;
    }
    .logo svg {
      width: 48px; height: 48px;
      color: #6366f1;
    }
    h1 {
      text-align: center;
      font-size: 1.25rem;
      font-weight: 600;
      color: #f1f5f9;
      margin-bottom: 0.25rem;
    }
    .subtitle {
      text-align: center;
      font-size: 0.875rem;
      color: #94a3b8;
      margin-bottom: 2rem;
    }
    label {
      display: block;
      font-size: 0.875rem;
      font-weight: 500;
      color: #cbd5e1;
      margin-bottom: 0.5rem;
    }
    input[type="password"] {
      width: 100%%;
      padding: 0.625rem 0.875rem;
      background: #0f172a;
      border: 1px solid #334155;
      border-radius: 8px;
      color: #f1f5f9;
      font-size: 1rem;
      outline: none;
      transition: border-color .15s;
    }
    input[type="password"]:focus { border-color: #6366f1; }
    button {
      margin-top: 1.25rem;
      width: 100%%;
      padding: 0.75rem;
      background: #6366f1;
      color: #fff;
      border: none;
      border-radius: 8px;
      font-size: 1rem;
      font-weight: 600;
      cursor: pointer;
      transition: background .15s;
    }
    button:hover { background: #4f46e5; }
    .error {
      margin-top: 1rem;
      padding: 0.75rem 1rem;
      background: rgba(239,68,68,0.15);
      border: 1px solid rgba(239,68,68,0.4);
      border-radius: 8px;
      color: #fca5a5;
      font-size: 0.875rem;
    }
  </style>
</head>
<body>
  <div class="card">
    <div class="logo">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M12 2L2 7l10 5 10-5-10-5z"/>
        <path d="M2 17l10 5 10-5"/>
        <path d="M2 12l10 5 10-5"/>
      </svg>
    </div>
    <h1>MCP Gateway</h1>
    <p class="subtitle">Sign in to the admin console</p>
    <form method="POST" action="/_auth/login">
      <input type="hidden" name="redirect" value="%s">
      <div>
        <label for="password">Password</label>
        <input type="password" id="password" name="password" autofocus autocomplete="current-password" required>
      </div>
      <button type="submit">Sign in</button>
      %s
    </form>
  </div>
</body>
</html>`, html.EscapeString(redirect), errHTML)
}
