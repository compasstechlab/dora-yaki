package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireAuth_NoCookie(t *testing.T) {
	mw := RequireAuth(testSecret)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("inner handler should not run when unauthenticated")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/x", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestRequireAuth_ValidCookie(t *testing.T) {
	tok, err := SignToken(testSecret, NewClaims("99", "alice"))
	if err != nil {
		t.Fatalf("SignToken: %v", err)
	}
	mw := RequireAuth(testSecret)
	called := false
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		c, ok := FromContext(r.Context())
		if !ok || c.UserID != "99" {
			t.Fatalf("expected claims for user 99, got ok=%v c=%+v", ok, c)
		}
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/x", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: tok})
	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatalf("inner handler should have been invoked")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestRequireAuth_BearerHeader(t *testing.T) {
	tok, err := SignToken(testSecret, NewClaims("7", "bob"))
	if err != nil {
		t.Fatalf("SignToken: %v", err)
	}
	mw := RequireAuth(testSecret)
	called := false
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/x", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	handler.ServeHTTP(rec, req)

	if !called || rec.Code != http.StatusOK {
		t.Fatalf("expected authenticated handler to run, status=%d called=%v", rec.Code, called)
	}
}
