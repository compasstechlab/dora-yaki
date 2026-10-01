package auth

import (
	"context"
	"errors"
)

var (
	// ErrNotAPIKey : the bearer token is not an API key (e.g. a session token).
	ErrNotAPIKey = errors.New("not an api key")
	// ErrInvalidAPIKey : the API key is unknown, mismatched, expired, or its owner is no longer valid.
	ErrInvalidAPIKey = errors.New("invalid api key")
)

// APIKeyVerifier : resolves an API key token to its owner. Returns ErrNotAPIKey / ErrInvalidAPIKey on rejection.
type APIKeyVerifier interface {
	VerifyAPIKey(ctx context.Context, token string) (userID, login string, err error)
}
