package model

import "time"

// User represents an authenticated dora-yaki user, identified by their GitHub identity.
type User struct {
	ID          string    `json:"id" datastore:"id"`
	Login       string    `json:"login" datastore:"login"`
	Name        string    `json:"name" datastore:"name"`
	AvatarURL   string    `json:"avatarUrl" datastore:"avatar_url"`
	Email       string    `json:"email" datastore:"email"`
	CreatedAt   time.Time `json:"createdAt" datastore:"created_at"`
	UpdatedAt   time.Time `json:"updatedAt" datastore:"updated_at"`
	LastLoginAt time.Time `json:"lastLoginAt" datastore:"last_login_at"`
}

// UserGitHubToken stores a user's GitHub OAuth tokens, encrypted at rest.
// Plaintext tokens never appear in this struct; callers decrypt the *Cipher
// fields with a crypto.Encryptor when they need the raw value.
type UserGitHubToken struct {
	UserID                string     `json:"userId" datastore:"user_id"`
	AccessTokenCipher     []byte     `json:"-" datastore:"access_token_cipher,noindex"`
	RefreshTokenCipher    []byte     `json:"-" datastore:"refresh_token_cipher,noindex"`
	AccessTokenExpiresAt  time.Time  `json:"accessTokenExpiresAt" datastore:"access_token_expires_at"`
	RefreshTokenExpiresAt time.Time  `json:"refreshTokenExpiresAt" datastore:"refresh_token_expires_at"`
	Scopes                []string   `json:"scopes" datastore:"scopes"`
	MarkedInvalidAt       *time.Time `json:"markedInvalidAt,omitempty" datastore:"marked_invalid_at"`
	LastRefreshedAt       *time.Time `json:"lastRefreshedAt,omitempty" datastore:"last_refreshed_at"`
	LastUsedAt            *time.Time `json:"lastUsedAt,omitempty" datastore:"last_used_at"`
	CreatedAt             time.Time  `json:"createdAt" datastore:"created_at"`
	UpdatedAt             time.Time  `json:"updatedAt" datastore:"updated_at"`
}

// IsAccessTokenExpiringSoon reports whether the access token will expire within
// the given lead time. A zero AccessTokenExpiresAt means GitHub did not provide
// an expiry, so the token is treated as long-lived (returns false).
func (t *UserGitHubToken) IsAccessTokenExpiringSoon(now time.Time, lead time.Duration) bool {
	if t.AccessTokenExpiresAt.IsZero() {
		return false
	}
	return !t.AccessTokenExpiresAt.After(now.Add(lead))
}

// IsRefreshTokenExpired reports whether the refresh token can no longer be used.
func (t *UserGitHubToken) IsRefreshTokenExpired(now time.Time) bool {
	if t.RefreshTokenExpiresAt.IsZero() {
		return false
	}
	return !t.RefreshTokenExpiresAt.After(now)
}
