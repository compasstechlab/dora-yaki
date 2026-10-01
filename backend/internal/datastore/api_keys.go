package datastore

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"cloud.google.com/go/datastore"

	"github.com/compasstechlab/dora-yaki/internal/domain/model"
)

// SaveAPIKey : upserts an API key row.
func (c *Client) SaveAPIKey(ctx context.Context, k *model.APIKey) error {
	key := datastore.NameKey(KindAPIKey, k.ID, nil)
	if _, err := c.client.Put(ctx, key, k); err != nil {
		return fmt.Errorf("save api_key: %w", err)
	}
	return nil
}

// GetAPIKey : loads an API key by id. Returns (nil, nil) if absent.
func (c *Client) GetAPIKey(ctx context.Context, id string) (*model.APIKey, error) {
	key := datastore.NameKey(KindAPIKey, id, nil)
	k := &model.APIKey{}
	if err := c.client.Get(ctx, key, k); err != nil {
		if errors.Is(err, datastore.ErrNoSuchEntity) {
			return nil, nil
		}
		return nil, fmt.Errorf("get api_key: %w", err)
	}
	return k, nil
}

// ListAPIKeysByUser : returns a user's API keys, newest first. Sorted in memory to avoid a composite index.
func (c *Client) ListAPIKeysByUser(ctx context.Context, userID string) ([]*model.APIKey, error) {
	var keys []*model.APIKey
	q := datastore.NewQuery(KindAPIKey).FilterField("user_id", "=", userID)
	if _, err := c.client.GetAll(ctx, q, &keys); err != nil {
		return nil, fmt.Errorf("list api_keys: %w", err)
	}
	slices.SortFunc(keys, func(a, b *model.APIKey) int {
		return b.CreatedAt.Compare(a.CreatedAt)
	})
	return keys, nil
}

// DeleteAPIKey : deletes an API key by id.
func (c *Client) DeleteAPIKey(ctx context.Context, id string) error {
	key := datastore.NameKey(KindAPIKey, id, nil)
	if err := c.client.Delete(ctx, key); err != nil {
		return fmt.Errorf("delete api_key: %w", err)
	}
	return nil
}

// UpdateAPIKeyLastUsed : stamps LastUsedAt on an API key. Runs in a transaction so a concurrent revoke is never undone.
func (c *Client) UpdateAPIKeyLastUsed(ctx context.Context, id string, at time.Time) error {
	key := datastore.NameKey(KindAPIKey, id, nil)
	_, err := c.client.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		k := &model.APIKey{}
		if err := tx.Get(key, k); err != nil {
			if errors.Is(err, datastore.ErrNoSuchEntity) {
				return nil
			}
			return fmt.Errorf("get api_key: %w", err)
		}
		k.LastUsedAt = &at
		if _, err := tx.Put(key, k); err != nil {
			return fmt.Errorf("update api_key: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("update api_key last used: %w", err)
	}
	return nil
}
