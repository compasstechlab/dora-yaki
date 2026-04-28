package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBuildAuthorizeURL(t *testing.T) {
	cfg := &GitHubOAuthConfig{
		ClientID:    "abc",
		RedirectURL: "https://example.com/api/auth/github/callback",
	}
	u := cfg.BuildAuthorizeURL("state-value")
	if !strings.HasPrefix(u, GitHubAuthorizeURL+"?") {
		t.Fatalf("authorize URL has wrong prefix: %s", u)
	}
	for _, want := range []string{
		"client_id=abc",
		"state=state-value",
		"redirect_uri=https%3A%2F%2Fexample.com%2Fapi%2Fauth%2Fgithub%2Fcallback",
		"scope=read%3Auser+user%3Aemail+repo",
	} {
		if !strings.Contains(u, want) {
			t.Errorf("authorize URL missing %q: %s", want, u)
		}
	}
}

func TestExchangeCode_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("Accept = %q, want application/json", got)
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1<<16)
		_ = r.ParseForm()
		if r.Form.Get("code") != "the-code" {
			t.Errorf("code form value = %q", r.Form.Get("code"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":             "ghu_access",
			"expires_in":               28800,
			"refresh_token":            "ghr_refresh",
			"refresh_token_expires_in": 15552000,
			"token_type":               "bearer",
			"scope":                    "repo,read:user",
		})
	}))
	defer srv.Close()

	cfg := &GitHubOAuthConfig{
		ClientID:      "id",
		ClientSecret:  "secret",
		RedirectURL:   "https://example/cb",
		TokenEndpoint: srv.URL,
	}
	tr, err := cfg.ExchangeCode(context.Background(), "the-code")
	if err != nil {
		t.Fatalf("ExchangeCode: %v", err)
	}
	if tr.AccessToken != "ghu_access" || tr.RefreshToken != "ghr_refresh" {
		t.Fatalf("token mismatch: %+v", tr)
	}
}

func TestExchangeCode_GitHubError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error":             "bad_verification_code",
			"error_description": "code expired",
		})
	}))
	defer srv.Close()

	cfg := &GitHubOAuthConfig{TokenEndpoint: srv.URL}
	if _, err := cfg.ExchangeCode(context.Background(), "x"); err == nil {
		t.Fatalf("expected error from GitHub error response")
	}
}

func TestRefreshAccessToken_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<16)
		_ = r.ParseForm()
		if r.Form.Get("grant_type") != "refresh_token" {
			t.Errorf("grant_type = %q", r.Form.Get("grant_type"))
		}
		if r.Form.Get("refresh_token") != "old-refresh" {
			t.Errorf("refresh_token = %q", r.Form.Get("refresh_token"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":             "new-access",
			"expires_in":               28800,
			"refresh_token":            "new-refresh",
			"refresh_token_expires_in": 15552000,
		})
	}))
	defer srv.Close()

	cfg := &GitHubOAuthConfig{TokenEndpoint: srv.URL}
	tr, err := cfg.RefreshAccessToken(context.Background(), "old-refresh")
	if err != nil {
		t.Fatalf("RefreshAccessToken: %v", err)
	}
	if tr.AccessToken != "new-access" || tr.RefreshToken != "new-refresh" {
		t.Fatalf("unexpected tokens: %+v", tr)
	}
}
