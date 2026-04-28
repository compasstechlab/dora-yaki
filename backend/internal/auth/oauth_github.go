package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// GitHubOAuthConfig holds the static OAuth App configuration plus an injectable
// HTTP client and token endpoint (for tests).
type GitHubOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	HTTPClient   *http.Client
	// TokenEndpoint overrides GitHubAccessTokenURL when non-empty. Used by tests.
	TokenEndpoint string
}

// GitHubAuthorizeURL is the GitHub OAuth authorization endpoint.
const GitHubAuthorizeURL = "https://github.com/login/oauth/authorize"

// GitHubAccessTokenURL is the OAuth token exchange endpoint.
const GitHubAccessTokenURL = "https://github.com/login/oauth/access_token" //nolint:gosec // public endpoint URL, not a credential

// DefaultScopes are the OAuth scopes requested when redirecting users.
// `repo` is needed to read private repository metadata; `read:user` and
// `user:email` provide the identity we link to the User entity.
var DefaultScopes = []string{"read:user", "user:email", "repo"}

// BuildAuthorizeURL constructs the URL that the browser should be redirected
// to for the authorization-code flow. The state must be unguessable and tied
// to the user's session/return path (see state.go).
func (c *GitHubOAuthConfig) BuildAuthorizeURL(state string) string {
	q := url.Values{}
	q.Set("client_id", c.ClientID)
	q.Set("redirect_uri", c.RedirectURL)
	q.Set("scope", strings.Join(DefaultScopes, " "))
	q.Set("state", state)
	q.Set("allow_signup", "false")
	return GitHubAuthorizeURL + "?" + q.Encode()
}

// TokenResponse is the JSON shape returned by GitHub's token endpoints.
type TokenResponse struct {
	AccessToken           string `json:"access_token"`
	ExpiresIn             int    `json:"expires_in"`
	RefreshToken          string `json:"refresh_token"`
	RefreshTokenExpiresIn int    `json:"refresh_token_expires_in"`
	TokenType             string `json:"token_type"`
	Scope                 string `json:"scope"`

	// Error fields are populated when GitHub rejects the request.
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	ErrorURI         string `json:"error_uri"`
}

// AccessTokenExpiresAt returns the absolute expiry of the access token, or zero
// if GitHub did not provide expires_in (legacy non-expiring tokens).
func (t *TokenResponse) AccessTokenExpiresAt(now time.Time) time.Time {
	if t.ExpiresIn <= 0 {
		return time.Time{}
	}
	return now.Add(time.Duration(t.ExpiresIn) * time.Second)
}

// RefreshTokenExpiresAt returns the absolute expiry of the refresh token, or
// zero if GitHub did not provide refresh_token_expires_in.
func (t *TokenResponse) RefreshTokenExpiresAt(now time.Time) time.Time {
	if t.RefreshTokenExpiresIn <= 0 {
		return time.Time{}
	}
	return now.Add(time.Duration(t.RefreshTokenExpiresIn) * time.Second)
}

// ExchangeCode swaps an authorization code for access + refresh tokens.
func (c *GitHubOAuthConfig) ExchangeCode(ctx context.Context, code string) (*TokenResponse, error) {
	body := url.Values{}
	body.Set("client_id", c.ClientID)
	body.Set("client_secret", c.ClientSecret)
	body.Set("code", code)
	body.Set("redirect_uri", c.RedirectURL)
	return c.postToken(ctx, body)
}

// RefreshAccessToken exchanges a refresh_token for a fresh access_token. GitHub
// rotates the refresh_token on each call, so callers must persist the new value.
func (c *GitHubOAuthConfig) RefreshAccessToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	body := url.Values{}
	body.Set("client_id", c.ClientID)
	body.Set("client_secret", c.ClientSecret)
	body.Set("grant_type", "refresh_token")
	body.Set("refresh_token", refreshToken)
	return c.postToken(ctx, body)
}

func (c *GitHubOAuthConfig) postToken(ctx context.Context, body url.Values) (*TokenResponse, error) {
	endpoint := c.tokenURL()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(body.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("post token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read token response: %w", err)
	}

	var tr TokenResponse
	if err := json.Unmarshal(raw, &tr); err != nil {
		return nil, fmt.Errorf("parse token response (status=%d): %w", resp.StatusCode, err)
	}
	if tr.Error != "" {
		return nil, fmt.Errorf("github oauth error: %s: %s", tr.Error, tr.ErrorDescription)
	}
	if tr.AccessToken == "" {
		return nil, fmt.Errorf("github oauth: empty access_token (status=%d)", resp.StatusCode)
	}
	return &tr, nil
}

func (c *GitHubOAuthConfig) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

// tokenURL returns the URL used for token exchange. Tests may override via
// TokenEndpoint; production always uses GitHubAccessTokenURL.
func (c *GitHubOAuthConfig) tokenURL() string {
	if c.TokenEndpoint != "" {
		return c.TokenEndpoint
	}
	return GitHubAccessTokenURL
}
