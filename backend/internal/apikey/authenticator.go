package apikey

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/compasstechlab/dora-yaki/internal/auth"
	"github.com/compasstechlab/dora-yaki/internal/domain/model"
)

// lastUsedUpdateInterval : minimum interval between LastUsedAt writes, to limit Datastore writes.
const lastUsedUpdateInterval = 15 * time.Minute

// Store : persistence needed to verify API keys.
type Store interface {
	GetAPIKey(ctx context.Context, id string) (*model.APIKey, error)
	UpdateAPIKeyLastUsed(ctx context.Context, id string, at time.Time) error
	GetUserGitHubToken(ctx context.Context, userID string) (*model.UserGitHubToken, error)
}

// Authenticator : verifies API key tokens against the Store.
type Authenticator struct {
	store  Store
	logger *slog.Logger
	now    func() time.Time
}

var _ auth.APIKeyVerifier = (*Authenticator)(nil)

// NewAuthenticator : creates an Authenticator.
func NewAuthenticator(store Store, logger *slog.Logger) *Authenticator {
	return &Authenticator{store: store, logger: logger, now: time.Now}
}

// VerifyAPIKey : resolves a token to its owner. Store failures are returned as non-sentinel errors.
func (a *Authenticator) VerifyAPIKey(ctx context.Context, token string) (userID, login string, err error) {
	id, secret, ok := Parse(token)
	if !ok {
		if strings.HasPrefix(token, Prefix) {
			return "", "", auth.ErrInvalidAPIKey
		}
		return "", "", auth.ErrNotAPIKey
	}

	key, err := a.store.GetAPIKey(ctx, id)
	if err != nil {
		return "", "", fmt.Errorf("get api key: %w", err)
	}
	now := a.now()
	// [sec] Do not reveal which check failed (avoid an existence oracle).
	if key == nil || !SecretMatches(secret, key.SecretHash) || key.IsExpired(now) {
		return "", "", auth.ErrInvalidAPIKey
	}

	owner, err := a.store.GetUserGitHubToken(ctx, key.UserID)
	if err != nil {
		return "", "", fmt.Errorf("get owner token: %w", err)
	}
	if owner == nil || owner.MarkedInvalidAt != nil {
		return "", "", auth.ErrInvalidAPIKey
	}

	a.touchLastUsed(ctx, key, now)
	return key.UserID, key.UserLogin, nil
}

// touchLastUsed : stamps LastUsedAt at most once per lastUsedUpdateInterval. Failures are only logged.
func (a *Authenticator) touchLastUsed(ctx context.Context, key *model.APIKey, now time.Time) {
	if key.LastUsedAt != nil && now.Sub(*key.LastUsedAt) < lastUsedUpdateInterval {
		return
	}
	if err := a.store.UpdateAPIKeyLastUsed(ctx, key.ID, now); err != nil {
		a.logger.Warn("failed to update api key last used", "apiKeyID", key.ID, "error", err)
	}
}
