package model

import "time"

// APIKeyPrefix : prefix of every personal API key token.
const APIKeyPrefix = "dyk_"

// APIKey : personal read-only API key. Only the SHA-256 hash of the secret is persisted.
type APIKey struct {
	ID         string     `json:"id" datastore:"id"`
	UserID     string     `json:"-" datastore:"user_id"`
	UserLogin  string     `json:"-" datastore:"user_login,noindex"`
	Name       string     `json:"name" datastore:"name,noindex"`
	SecretHash []byte     `json:"-" datastore:"secret_hash,noindex"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty" datastore:"expires_at,noindex"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty" datastore:"last_used_at,noindex"`
	CreatedAt  time.Time  `json:"createdAt" datastore:"created_at,noindex"`
}

// IsExpired : reports whether the key is no longer valid at now. A nil ExpiresAt never expires.
func (k *APIKey) IsExpired(now time.Time) bool {
	return k.ExpiresAt != nil && !now.Before(*k.ExpiresAt)
}

// DisplayPrefix : non-secret part of the token ("dyk_<id>") used to identify the key in UIs.
func (k *APIKey) DisplayPrefix() string {
	return APIKeyPrefix + k.ID
}
