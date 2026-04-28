package datastore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/datastore"

	"github.com/compasstechlab/dora-yaki/internal/domain/model"
)

// SaveRepositoryAccess upserts a (user, repo) access row.
func (c *Client) SaveRepositoryAccess(ctx context.Context, a *model.RepositoryAccess) error {
	key := datastore.NameKey(KindRepositoryAccess, model.RepositoryAccessKey(a.UserID, a.RepositoryID), nil)
	if _, err := c.client.Put(ctx, key, a); err != nil {
		return fmt.Errorf("save repository_access: %w", err)
	}
	return nil
}

// GetRepositoryAccess fetches a single (user, repo) access row.
// Returns (nil, nil) when absent.
func (c *Client) GetRepositoryAccess(ctx context.Context, userID, repoID string) (*model.RepositoryAccess, error) {
	key := datastore.NameKey(KindRepositoryAccess, model.RepositoryAccessKey(userID, repoID), nil)
	a := &model.RepositoryAccess{}
	if err := c.client.Get(ctx, key, a); err != nil {
		if errors.Is(err, datastore.ErrNoSuchEntity) {
			return nil, nil
		}
		return nil, fmt.Errorf("get repository_access: %w", err)
	}
	return a, nil
}

// ListRepositoryAccessByRepo returns every access row for a repository,
// filtered to candidates whose token is currently valid (is_token_candidate=true
// AND marked_invalid_at is nil), ordered by least-recently-used first.
func (c *Client) ListRepositoryAccessByRepo(ctx context.Context, repoID string) ([]*model.RepositoryAccess, error) {
	q := datastore.NewQuery(KindRepositoryAccess).
		FilterField("repository_id", "=", repoID).
		FilterField("is_token_candidate", "=", true)
	var rows []*model.RepositoryAccess
	if _, err := c.client.GetAll(ctx, q, &rows); err != nil {
		return nil, fmt.Errorf("list repository_access by repo: %w", err)
	}

	// Filter out invalid candidates and sort by LastUsedAt ascending in app code
	// to avoid composite-index requirements during early development. The volume
	// of candidates per repo is expected to be small (a handful of users).
	out := rows[:0]
	for _, r := range rows {
		if r.MarkedInvalidAt != nil {
			continue
		}
		out = append(out, r)
	}
	sortByLastUsedAsc(out)
	return out, nil
}

// ListRepositoryAccessByUser returns every access row for a user.
func (c *Client) ListRepositoryAccessByUser(ctx context.Context, userID string) ([]*model.RepositoryAccess, error) {
	q := datastore.NewQuery(KindRepositoryAccess).FilterField("user_id", "=", userID)
	var rows []*model.RepositoryAccess
	if _, err := c.client.GetAll(ctx, q, &rows); err != nil {
		return nil, fmt.Errorf("list repository_access by user: %w", err)
	}
	return rows, nil
}

// MarkRepositoryAccessCandidateInvalid clears IsTokenCandidate and stamps
// MarkedInvalidAt on the (user, repo) access row.
func (c *Client) MarkRepositoryAccessCandidateInvalid(ctx context.Context, userID, repoID string) error {
	key := datastore.NameKey(KindRepositoryAccess, model.RepositoryAccessKey(userID, repoID), nil)
	a := &model.RepositoryAccess{}
	if err := c.client.Get(ctx, key, a); err != nil {
		if errors.Is(err, datastore.ErrNoSuchEntity) {
			return nil
		}
		return fmt.Errorf("get repository_access: %w", err)
	}
	now := time.Now()
	a.IsTokenCandidate = false
	a.MarkedInvalidAt = &now
	if _, err := c.client.Put(ctx, key, a); err != nil {
		return fmt.Errorf("update repository_access: %w", err)
	}
	return nil
}

// UpdateRepositoryAccessLastUsed stamps LastUsedAt on a (user, repo) access row.
func (c *Client) UpdateRepositoryAccessLastUsed(ctx context.Context, userID, repoID string) error {
	key := datastore.NameKey(KindRepositoryAccess, model.RepositoryAccessKey(userID, repoID), nil)
	a := &model.RepositoryAccess{}
	if err := c.client.Get(ctx, key, a); err != nil {
		if errors.Is(err, datastore.ErrNoSuchEntity) {
			return nil
		}
		return fmt.Errorf("get repository_access: %w", err)
	}
	now := time.Now()
	a.LastUsedAt = &now
	if _, err := c.client.Put(ctx, key, a); err != nil {
		return fmt.Errorf("update repository_access: %w", err)
	}
	return nil
}

// ListAccessibleRepositoryIDsByUser returns the repository IDs this user is
// permitted to view (CanAccess=true).
func (c *Client) ListAccessibleRepositoryIDsByUser(ctx context.Context, userID string) ([]string, error) {
	q := datastore.NewQuery(KindRepositoryAccess).
		FilterField("user_id", "=", userID).
		FilterField("can_access", "=", true)
	var rows []*model.RepositoryAccess
	if _, err := c.client.GetAll(ctx, q, &rows); err != nil {
		return nil, fmt.Errorf("list accessible repos: %w", err)
	}
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.RepositoryID)
	}
	return ids, nil
}

// sortByLastUsedAsc orders rows by LastUsedAt ascending. Nil LastUsedAt sorts first.
func sortByLastUsedAsc(rows []*model.RepositoryAccess) {
	// Simple O(n^2) insertion sort is fine for the small slice sizes we expect
	// (handful of token candidates per repository).
	for i := 1; i < len(rows); i++ {
		for j := i; j > 0 && lastUsedLess(rows[j], rows[j-1]); j-- {
			rows[j], rows[j-1] = rows[j-1], rows[j]
		}
	}
}

func lastUsedLess(a, b *model.RepositoryAccess) bool {
	switch {
	case a.LastUsedAt == nil && b.LastUsedAt == nil:
		return false
	case a.LastUsedAt == nil:
		return true
	case b.LastUsedAt == nil:
		return false
	default:
		return a.LastUsedAt.Before(*b.LastUsedAt)
	}
}
