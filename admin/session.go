package admin

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const sessionCookieName = "mcpgw_session"

// createSessionCookie returns a signed, HttpOnly session cookie.
// The cookie value is base64(expiry_unix).<hex(hmac-sha256(secret, base64(expiry_unix)))>.
func createSessionCookie(secret string, ttl time.Duration, secure bool) *http.Cookie {
	expiry := time.Now().Add(ttl).Unix()
	payload := base64.URLEncoding.EncodeToString([]byte(strconv.FormatInt(expiry, 10)))
	sig := computeSessionHMAC(secret, payload)
	return &http.Cookie{
		Name:     sessionCookieName,
		Value:    payload + "." + sig,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(ttl.Seconds()),
	}
}

// validateSession returns true if the request carries a valid, unexpired session cookie.
func validateSession(r *http.Request, secret string) bool {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return false
	}
	parts := strings.SplitN(cookie.Value, ".", 2)
	if len(parts) != 2 {
		return false
	}
	payload, sig := parts[0], parts[1]
	if !hmac.Equal([]byte(computeSessionHMAC(secret, payload)), []byte(sig)) {
		return false
	}
	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return false
	}
	expiry, err := strconv.ParseInt(string(decoded), 10, 64)
	if err != nil {
		return false
	}
	return time.Now().Unix() < expiry
}

func computeSessionHMAC(secret, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}
