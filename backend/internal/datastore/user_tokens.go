package datastore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/datastore"

	"github.com/compasstechlab/dora-yaki/internal/domain/model"
)

// SaveUserGitHubToken upserts the encrypted token row for a user.
func (c *Client) SaveUserGitHubToken(ctx context.Context, t *model.UserGitHubToken) error {
	key := datastore.NameKey(KindUserGitHubToken, t.UserID, nil)
	if _, err := c.client.Put(ctx, key, t); err != nil {
		return fmt.Errorf("save user_github_token: %w", err)
	}
	return nil
}

// GetUserGitHubToken loads the encrypted token row for a user. Returns (nil, nil) if absent.
func (c *Client) GetUserGitHubToken(ctx context.Context, userID string) (*model.UserGitHubToken, error) {
	key := datastore.NameKey(KindUserGitHubToken, userID, nil)
	t := &model.UserGitHubToken{}
	if err := c.client.Get(ctx, key, t); err != nil {
		if errors.Is(err, datastore.ErrNoSuchEntity) {
			return nil, nil
		}
		return nil, fmt.Errorf("get user_github_token: %w", err)
	}
	return t, nil
}

// MarkUserTokenInvalid sets MarkedInvalidAt=now on a user's token row, signaling
// that the next API attempt should treat the user as unauthenticated.
func (c *Client) MarkUserTokenInvalid(ctx context.Context, userID string) error {
	key := datastore.NameKey(KindUserGitHubToken, userID, nil)
	t := &model.UserGitHubToken{}
	if err := c.client.Get(ctx, key, t); err != nil {
		if errors.Is(err, datastore.ErrNoSuchEntity) {
			return nil
		}
		return fmt.Errorf("get user_github_token: %w", err)
	}
	now := time.Now()
	t.MarkedInvalidAt = &now
	t.UpdatedAt = now
	if _, err := c.client.Put(ctx, key, t); err != nil {
		return fmt.Errorf("update user_github_token: %w", err)
	}
	return nil
}

// UpdateUserTokenLastUsed stamps LastUsedAt on a user's token row.
func (c *Client) UpdateUserTokenLastUsed(ctx context.Context, userID string) error {
	key := datastore.NameKey(KindUserGitHubToken, userID, nil)
	t := &model.UserGitHubToken{}
	if err := c.client.Get(ctx, key, t); err != nil {
		if errors.Is(err, datastore.ErrNoSuchEntity) {
			return nil
		}
		return fmt.Errorf("get user_github_token: %w", err)
	}
	now := time.Now()
	t.LastUsedAt = &now
	t.UpdatedAt = now
	if _, err := c.client.Put(ctx, key, t); err != nil {
		return fmt.Errorf("update user_github_token: %w", err)
	}
	return nil
}
