package auth

import (
	"context"
	"errors"
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

type fakeVerifier struct {
	userID, login string
	err           error
	calls         int
}

func (f *fakeVerifier) VerifyAPIKey(_ context.Context, _ string) (string, string, error) {
	f.calls++
	return f.userID, f.login, f.err
}

func TestRequireAuthOrAPIKey(t *testing.T) {
	session, err := SignToken(testSecret, NewClaims("99", "alice"))
	if err != nil {
		t.Fatalf("SignToken: %v", err)
	}
	const apiKey = "dyk_3f9a1c2b7d4e5f60_secret" //nolint:gosec // G101: dummy fixture

	tests := []struct {
		name          string
		cookie        string
		bearer        string
		verifier      *fakeVerifier
		wantStatus    int
		wantUserID    string
		wantLogin     string
		wantVerifyRun bool
	}{
		{name: "valid cookie", cookie: session, verifier: &fakeVerifier{}, wantStatus: http.StatusOK, wantUserID: "99", wantLogin: "alice"},
		{name: "valid session bearer", bearer: session, verifier: &fakeVerifier{}, wantStatus: http.StatusOK, wantUserID: "99", wantLogin: "alice"},
		{name: "valid api key bearer", bearer: apiKey, verifier: &fakeVerifier{userID: "42", login: "bob"}, wantStatus: http.StatusOK, wantUserID: "42", wantLogin: "bob", wantVerifyRun: true},
		{name: "invalid api key", bearer: apiKey, verifier: &fakeVerifier{err: ErrInvalidAPIKey}, wantStatus: http.StatusUnauthorized, wantVerifyRun: true},
		{name: "not an api key", bearer: "garbage", verifier: &fakeVerifier{err: ErrNotAPIKey}, wantStatus: http.StatusUnauthorized, wantVerifyRun: true},
		{name: "verifier internal error", bearer: apiKey, verifier: &fakeVerifier{err: errors.New("datastore down")}, wantStatus: http.StatusInternalServerError, wantVerifyRun: true},
		{name: "no credentials", verifier: &fakeVerifier{}, wantStatus: http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotClaims *Claims
			handler := RequireAuthOrAPIKey(testSecret, tt.verifier)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotClaims, _ = FromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/api/x", nil)
			if tt.cookie != "" {
				req.AddCookie(&http.Cookie{Name: CookieName, Value: tt.cookie}) //nolint:gosec // G124: request-side test cookie
			}
			if tt.bearer != "" {
				req.Header.Set("Authorization", "Bearer "+tt.bearer)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if (tt.verifier.calls > 0) != tt.wantVerifyRun {
				t.Fatalf("verifier calls = %d, want called=%v", tt.verifier.calls, tt.wantVerifyRun)
			}
			if tt.wantStatus != http.StatusOK {
				return
			}
			if gotClaims == nil || gotClaims.UserID != tt.wantUserID || gotClaims.Login != tt.wantLogin {
				t.Fatalf("claims = %+v, want user %q login %q", gotClaims, tt.wantUserID, tt.wantLogin)
			}
		})
	}
}

func TestRequireAuth_RejectsAPIKeyBearer(t *testing.T) {
	handler := RequireAuth(testSecret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("session-only gate must not accept API keys")
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/x", nil)
	req.Header.Set("Authorization", "Bearer dyk_3f9a1c2b7d4e5f60_secret")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}
