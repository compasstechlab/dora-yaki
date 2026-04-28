package github

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/compasstechlab/dora-yaki/internal/auth"
	"github.com/compasstechlab/dora-yaki/internal/crypto"
	"github.com/compasstechlab/dora-yaki/internal/datastore"
	"github.com/compasstechlab/dora-yaki/internal/domain/model"
)

// refreshLead controls how soon before expiry an access token is proactively refreshed.
const refreshLead = 5 * time.Minute

// Sentinel errors returned by ClientFactory.ForUser.
var (
	// ErrTokenInvalid means the user's stored token has been marked invalid and cannot be used.
	ErrTokenInvalid = errors.New("user github token marked invalid")
	// ErrRefreshExpired means the refresh token itself has expired; user must re-login.
	ErrRefreshExpired = errors.New("user github refresh token expired")
	// ErrNoToken means the user has no stored token.
	ErrNoToken = errors.New("user has no github token")
)

// ClientFactory builds *github.Client instances scoped to a particular user,
// using their stored OAuth access_token (decrypted on demand). Refreshes the
// access_token if it is near expiry, and persists the rotated refresh_token.
type ClientFactory struct {
	ds        *datastore.Client
	encryptor crypto.Encryptor
	oauth     *auth.GitHubOAuthConfig
	logger    *slog.Logger
}

// NewClientFactory builds a ClientFactory.
func NewClientFactory(
	ds *datastore.Client,
	encryptor crypto.Encryptor,
	oauth *auth.GitHubOAuthConfig,
	logger *slog.Logger,
) *ClientFactory {
	return &ClientFactory{
		ds:        ds,
		encryptor: encryptor,
		oauth:     oauth,
		logger:    logger,
	}
}

// ForUser returns a Client authenticated as the given user. Refreshes the
// access token if it is within refreshLead of expiry. On unrecoverable token
// errors (refresh expired, refresh rejected) the user's token row is marked
// invalid and a typed sentinel is returned.
func (f *ClientFactory) ForUser(ctx context.Context, userID string) (*Client, error) {
	row, err := f.ds.GetUserGitHubToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("load user token: %w", err)
	}
	if row == nil {
		return nil, ErrNoToken
	}
	if row.MarkedInvalidAt != nil {
		return nil, ErrTokenInvalid
	}

	now := time.Now()
	if row.IsAccessTokenExpiringSoon(now, refreshLead) {
		if row.IsRefreshTokenExpired(now) {
			if err := f.ds.MarkUserTokenInvalid(ctx, userID); err != nil {
				f.logger.Warn("mark user token invalid", "userID", userID, "error", err)
			}
			return nil, ErrRefreshExpired
		}
		refreshed, refErr := f.refreshAndPersist(ctx, row)
		if refErr != nil {
			if err := f.ds.MarkUserTokenInvalid(ctx, userID); err != nil {
				f.logger.Warn("mark user token invalid", "userID", userID, "error", err)
			}
			return nil, fmt.Errorf("refresh token: %w", refErr)
		}
		row = refreshed
	}

	plain, err := f.encryptor.Decrypt(ctx, row.AccessTokenCipher)
	if err != nil {
		return nil, fmt.Errorf("decrypt access token: %w", err)
	}
	return NewClient(string(plain)), nil
}

// refreshAndPersist exchanges the stored refresh token for a new access token
// pair, encrypts the rotated values, and writes them back to Datastore.
func (f *ClientFactory) refreshAndPersist(ctx context.Context, row *model.UserGitHubToken) (*model.UserGitHubToken, error) {
	refreshPlain, err := f.encryptor.Decrypt(ctx, row.RefreshTokenCipher)
	if err != nil {
		return nil, fmt.Errorf("decrypt refresh token: %w", err)
	}
	tr, err := f.oauth.RefreshAccessToken(ctx, string(refreshPlain))
	if err != nil {
		return nil, err
	}
	now := time.Now()
	accessCipher, err := f.encryptor.Encrypt(ctx, []byte(tr.AccessToken))
	if err != nil {
		return nil, fmt.Errorf("encrypt new access token: %w", err)
	}
	var refreshCipher []byte
	if tr.RefreshToken != "" {
		refreshCipher, err = f.encryptor.Encrypt(ctx, []byte(tr.RefreshToken))
		if err != nil {
			return nil, fmt.Errorf("encrypt new refresh token: %w", err)
		}
	} else {
		// GitHub did not rotate the refresh token; keep existing ciphertext.
		refreshCipher = row.RefreshTokenCipher
	}

	row.AccessTokenCipher = accessCipher
	row.RefreshTokenCipher = refreshCipher
	if exp := tr.AccessTokenExpiresAt(now); !exp.IsZero() {
		row.AccessTokenExpiresAt = exp
	}
	if exp := tr.RefreshTokenExpiresAt(now); !exp.IsZero() {
		row.RefreshTokenExpiresAt = exp
	}
	row.LastRefreshedAt = &now
	row.UpdatedAt = now

	if err := f.ds.SaveUserGitHubToken(ctx, row); err != nil {
		return nil, fmt.Errorf("persist refreshed token: %w", err)
	}
	return row, nil
}
