package auth

import (
	"net/http"
	"time"
)

// SetSessionCookie writes the signed session token into a secure HttpOnly cookie.
// Secure flag is enabled when the request was over TLS or X-Forwarded-Proto: https.
func SetSessionCookie(w http.ResponseWriter, r *http.Request, token, domain string, ttl time.Duration) {
	secure := isTLSRequest(r)
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		Domain:   domain,
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		SameSite: sameSiteMode(secure),
		Secure:   secure,
	})
}

// ClearSessionCookie expires the session cookie immediately.
func ClearSessionCookie(w http.ResponseWriter, r *http.Request, domain string) {
	secure := isTLSRequest(r)
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		Domain:   domain,
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: sameSiteMode(secure),
		Secure:   secure,
	})
}

func sameSiteMode(secure bool) http.SameSite {
	if secure {
		return http.SameSiteNoneMode
	}
	return http.SameSiteLaxMode
}

func isTLSRequest(r *http.Request) bool {
	if r == nil {
		return false
	}
	if r.TLS != nil {
		return true
	}
	return r.Header.Get("X-Forwarded-Proto") == "https"
}
