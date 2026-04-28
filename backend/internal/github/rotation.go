package github

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/compasstechlab/dora-yaki/internal/datastore"
)

const maxRotationAttempts = 3

// Sentinel errors for the rotation pool.
var (
	// ErrNoCandidate means no token candidate exists for the repository and no fallback is configured.
	ErrNoCandidate = errors.New("no token candidate for repository")
	// ErrAllCandidatesFailed means every candidate failed with an auth error.
	ErrAllCandidatesFailed = errors.New("all token candidates failed")
)

// TokenPool selects a per-user GitHub Client to use for syncing a particular
// repository, rotating to a different user when the current one's token is
// rejected. Backed by RepositoryAccess rows that mark which users are valid
// candidates.
type TokenPool struct {
	factory *ClientFactory
	ds      *datastore.Client
	logger  *slog.Logger
}

// NewTokenPool builds a TokenPool from the per-user ClientFactory.
func NewTokenPool(factory *ClientFactory, ds *datastore.Client, logger *slog.Logger) *TokenPool {
	return &TokenPool{factory: factory, ds: ds, logger: logger}
}

// candidateClient pairs a Client with the userID it belongs to (empty string
// when the client is the legacy fallback).
type candidateClient struct {
	client *Client
	userID string
}

// pickCandidates returns candidate clients for the repo, ordered LRU-first.
func (p *TokenPool) pickCandidates(ctx context.Context, repoID string) ([]candidateClient, error) {
	rows, err := p.ds.ListRepositoryAccessByRepo(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("list candidates: %w", err)
	}

	out := make([]candidateClient, 0, len(rows))
	for _, row := range rows {
		c, err := p.factory.ForUser(ctx, row.UserID)
		if err != nil {
			p.logger.Warn("skip candidate (token unavailable)",
				"userID", row.UserID, "repoID", repoID, "error", err,
			)
			continue
		}
		out = append(out, candidateClient{client: c, userID: row.UserID})
	}
	if len(out) == 0 {
		return nil, ErrNoCandidate
	}
	return out, nil
}

// RunWithRetry executes fn with a candidate Client. On auth error (401/403),
// marks the offending user invalid for this repo and retries with the next
// candidate, up to maxRotationAttempts.
//
// Returns the last error if all candidates fail, or any non-auth error from fn.
func (p *TokenPool) RunWithRetry(ctx context.Context, repoID string, fn func(*Client) error) error {
	candidates, err := p.pickCandidates(ctx, repoID)
	if err != nil {
		return err
	}

	var lastErr error
	limit := len(candidates)
	if limit > maxRotationAttempts {
		limit = maxRotationAttempts
	}

	for i := 0; i < limit; i++ {
		cand := candidates[i]
		err := fn(cand.client)
		if err == nil {
			if cand.userID != "" {
				if uerr := p.ds.UpdateRepositoryAccessLastUsed(ctx, cand.userID, repoID); uerr != nil {
					p.logger.Warn("update last_used", "userID", cand.userID, "repoID", repoID, "error", uerr)
				}
				if uerr := p.ds.UpdateUserTokenLastUsed(ctx, cand.userID); uerr != nil {
					p.logger.Warn("update token last_used", "userID", cand.userID, "error", uerr)
				}
			}
			return nil
		}
		if !IsAuthError(err) {
			return err
		}
		lastErr = err
		if cand.userID != "" {
			p.logger.Warn("token rejected; rotating", "userID", cand.userID, "repoID", repoID, "error", err)
			if uerr := p.ds.MarkRepositoryAccessCandidateInvalid(ctx, cand.userID, repoID); uerr != nil {
				p.logger.Warn("mark candidate invalid", "userID", cand.userID, "repoID", repoID, "error", uerr)
			}
			if uerr := p.ds.MarkUserTokenInvalid(ctx, cand.userID); uerr != nil {
				p.logger.Warn("mark user token invalid", "userID", cand.userID, "error", uerr)
			}
		}
	}

	if lastErr == nil {
		return ErrAllCandidatesFailed
	}
	return fmt.Errorf("%w: %v", ErrAllCandidatesFailed, lastErr)
}
