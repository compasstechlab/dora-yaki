package datastore

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/datastore"

	"github.com/compasstechlab/dora-yaki/internal/domain/model"
)

// SaveUser inserts or updates a User entity.
func (c *Client) SaveUser(ctx context.Context, user *model.User) error {
	key := datastore.NameKey(KindUser, user.ID, nil)
	if _, err := c.client.Put(ctx, key, user); err != nil {
		return fmt.Errorf("save user: %w", err)
	}
	return nil
}

// GetUser fetches a User by ID. Returns (nil, nil) if not found.
func (c *Client) GetUser(ctx context.Context, id string) (*model.User, error) {
	key := datastore.NameKey(KindUser, id, nil)
	user := &model.User{}
	if err := c.client.Get(ctx, key, user); err != nil {
		if errors.Is(err, datastore.ErrNoSuchEntity) {
			return nil, nil
		}
		return nil, fmt.Errorf("get user: %w", err)
	}
	return user, nil
}

// GetUserByLogin fetches the first User entity matching the given GitHub login.
// Returns (nil, nil) if not found.
func (c *Client) GetUserByLogin(ctx context.Context, login string) (*model.User, error) {
	q := datastore.NewQuery(KindUser).FilterField("login", "=", login).Limit(1)
	var users []*model.User
	if _, err := c.client.GetAll(ctx, q, &users); err != nil {
		return nil, fmt.Errorf("query user by login: %w", err)
	}
	if len(users) == 0 {
		return nil, nil
	}
	return users[0], nil
}

// ListUsers returns every User in the system.
func (c *Client) ListUsers(ctx context.Context) ([]*model.User, error) {
	var users []*model.User
	q := datastore.NewQuery(KindUser)
	if _, err := c.client.GetAll(ctx, q, &users); err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return users, nil
}
