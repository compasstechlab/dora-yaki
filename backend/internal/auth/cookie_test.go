package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSetSessionCookieUsesSameSiteNoneForHTTPS(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://api.example.com/api/auth/github/callback", nil)
	rec := httptest.NewRecorder()

	SetSessionCookie(rec, req, "token", "", TokenTTL)

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	if cookies[0].SameSite != http.SameSiteNoneMode {
		t.Fatalf("expected SameSite=None, got %v", cookies[0].SameSite)
	}
	if !cookies[0].Secure {
		t.Fatal("expected Secure cookie for HTTPS request")
	}
}

func TestSetSessionCookieUsesLaxForHTTP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://localhost:7202/api/auth/github/callback", nil)
	rec := httptest.NewRecorder()

	SetSessionCookie(rec, req, "token", "", TokenTTL)

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	if cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatalf("expected SameSite=Lax, got %v", cookies[0].SameSite)
	}
	if cookies[0].Secure {
		t.Fatal("expected non-Secure cookie for HTTP request")
	}
}
