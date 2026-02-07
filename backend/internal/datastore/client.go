package datastore

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/datastore"

	"github.com/compasstechlab/dora-yaki/internal/domain/model"
)

// Client wraps the Cloud Datastore client
type Client struct {
	client    *datastore.Client
	projectID string
}

// Kind names for Datastore entities
const (
	KindRepository   = "Repository"
	KindPullRequest  = "PullRequest"
	KindReview       = "Review"
	KindDeployment   = "Deployment"
	KindDailyMetrics = "DailyMetrics"
	KindTeamMember   = "TeamMember"
	KindSprint       = "Sprint"
	KindMetricsCache = "MetricsCache"
	KindBotUser      = "BotUser"
	KindSyncLock     = "SyncLock"
)

// NewClient creates a new Datastore client
func NewClient(ctx context.Context, projectID string) (*Client, error) {
	client, err := datastore.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create datastore client: %w", err)
	}

	return &Client{
		client:    client,
		projectID: projectID,
	}, nil
}

// Close closes the Datastore client
func (c *Client) Close() error {
	return c.client.Close()
}

// Repository operations

// SaveRepository saves a repository to Datastore
func (c *Client) SaveRepository(ctx context.Context, repo *model.Repository) error {
	key := datastore.NameKey(KindRepository, repo.ID, nil)
	_, err := c.client.Put(ctx, key, repo)
	return err
}

// GetRepository gets a repository by ID
func (c *Client) GetRepository(ctx context.Context, id string) (*model.Repository, error) {
	key := datastore.NameKey(KindRepository, id, nil)
	repo := &model.Repository{}
	if err := c.client.Get(ctx, key, repo); err != nil {
		return nil, err
	}
	return repo, nil
}

// ListRepositories lists all repositories
func (c *Client) ListRepositories(ctx context.Context) ([]*model.Repository, error) {
	var repos []*model.Repository
	query := datastore.NewQuery(KindRepository).Order("-updated_at")
	_, err := c.client.GetAll(ctx, query, &repos)
	return repos, err
}

// DeleteRepository deletes a repository
func (c *Client) DeleteRepository(ctx context.Context, id string) error {
	key := datastore.NameKey(KindRepository, id, nil)
	return c.client.Delete(ctx, key)
}

// Pull Request operations

// SavePullRequests saves multiple pull requests
func (c *Client) SavePullRequests(ctx context.Context, prs []*model.PullRequest) error {
	keys := make([]*datastore.Key, len(prs))
	for i, pr := range prs {
		keys[i] = datastore.NameKey(KindPullRequest, pr.ID, nil)
	}

	_, err := c.client.PutMulti(ctx, keys, prs)
	return err
}

// GetPullRequest gets a pull request by ID
func (c *Client) GetPullRequest(ctx context.Context, id string) (*model.PullRequest, error) {
	key := datastore.NameKey(KindPullRequest, id, nil)
	pr := &model.PullRequest{}
	if err := c.client.Get(ctx, key, pr); err != nil {
		return nil, err
	}
	return pr, nil
}

// ListPullRequests lists pull requests for a repository
func (c *Client) ListPullRequests(ctx context.Context, repositoryID string, opts *QueryOptions) ([]*model.PullRequest, error) {
	var prs []*model.PullRequest
	query := datastore.NewQuery(KindPullRequest).
		FilterField("repository_id", "=", repositoryID).
		Order("-created_at")

	if opts != nil {
		if !opts.Since.IsZero() {
			query = query.FilterField("updated_at", ">=", opts.Since)
		}
		if opts.Limit > 0 {
			query = query.Limit(opts.Limit)
		}
	}

	_, err := c.client.GetAll(ctx, query, &prs)
	return prs, err
}

// ListPullRequestsByDateRange lists PRs within a date range
func (c *Client) ListPullRequestsByDateRange(ctx context.Context, repositoryID string, startDate, endDate time.Time) ([]*model.PullRequest, error) {
	var prs []*model.PullRequest
	query := datastore.NewQuery(KindPullRequest).
		FilterField("repository_id", "=", repositoryID).
		FilterField("created_at", ">=", startDate).
		FilterField("created_at", "<=", endDate)

	_, err := c.client.GetAll(ctx, query, &prs)
	return prs, err
}

// Review operations

// SaveReviews saves multiple reviews
func (c *Client) SaveReviews(ctx context.Context, reviews []*model.Review) error {
	if len(reviews) == 0 {
		return nil
	}

	keys := make([]*datastore.Key, len(reviews))
	for i, r := range reviews {
		keys[i] = datastore.NameKey(KindReview, r.ID, nil)
	}

	_, err := c.client.PutMulti(ctx, keys, reviews)
	return err
}

// ListReviews lists reviews for a repository
func (c *Client) ListReviews(ctx context.Context, repositoryID string, opts *QueryOptions) ([]*model.Review, error) {
	var reviews []*model.Review
	query := datastore.NewQuery(KindReview).
		FilterField("repository_id", "=", repositoryID).
		Order("-submitted_at")

	if opts != nil {
		if !opts.Since.IsZero() {
			query = query.FilterField("submitted_at", ">=", opts.Since)
		}
		if opts.Limit > 0 {
			query = query.Limit(opts.Limit)
		}
	}

	_, err := c.client.GetAll(ctx, query, &reviews)
	return reviews, err
}

// ListReviewsByDateRange lists reviews within a date range
func (c *Client) ListReviewsByDateRange(ctx context.Context, repositoryID string, startDate, endDate time.Time) ([]*model.Review, error) {
	var reviews []*model.Review
	query := datastore.NewQuery(KindReview).
		FilterField("repository_id", "=", repositoryID).
		FilterField("submitted_at", ">=", startDate).
		FilterField("submitted_at", "<=", endDate)

	_, err := c.client.GetAll(ctx, query, &reviews)
	return reviews, err
}

// Deployment operations

// SaveDeployments saves multiple deployments
func (c *Client) SaveDeployments(ctx context.Context, deployments []*model.Deployment) error {
	if len(deployments) == 0 {
		return nil
	}

	keys := make([]*datastore.Key, len(deployments))
	for i, d := range deployments {
		keys[i] = datastore.NameKey(KindDeployment, d.ID, nil)
	}

	_, err := c.client.PutMulti(ctx, keys, deployments)
	return err
}

// ListDeployments lists deployments for a repository
func (c *Client) ListDeployments(ctx context.Context, repositoryID string, opts *QueryOptions) ([]*model.Deployment, error) {
	var deployments []*model.Deployment
	query := datastore.NewQuery(KindDeployment).
		FilterField("repository_id", "=", repositoryID).
		Order("-created_at")

	if opts != nil {
		if !opts.Since.IsZero() {
			query = query.FilterField("created_at", ">=", opts.Since)
		}
		if opts.Limit > 0 {
			query = query.Limit(opts.Limit)
		}
	}

	_, err := c.client.GetAll(ctx, query, &deployments)
	return deployments, err
}

// Daily Metrics operations

// SaveDailyMetrics saves daily metrics
func (c *Client) SaveDailyMetrics(ctx context.Context, metrics *model.DailyMetrics) error {
	key := datastore.NameKey(KindDailyMetrics, metrics.ID, nil)
	_, err := c.client.Put(ctx, key, metrics)
	return err
}

// SaveDailyMetricsBatch saves multiple daily metrics
func (c *Client) SaveDailyMetricsBatch(ctx context.Context, metricsList []*model.DailyMetrics) error {
	if len(metricsList) == 0 {
		return nil
	}

	keys := make([]*datastore.Key, len(metricsList))
	for i, m := range metricsList {
		keys[i] = datastore.NameKey(KindDailyMetrics, m.ID, nil)
	}

	_, err := c.client.PutMulti(ctx, keys, metricsList)
	return err
}

// ListDailyMetrics lists daily metrics for a repository
func (c *Client) ListDailyMetrics(ctx context.Context, repositoryID string, startDate, endDate time.Time) ([]*model.DailyMetrics, error) {
	var metrics []*model.DailyMetrics
	query := datastore.NewQuery(KindDailyMetrics).
		FilterField("repository_id", "=", repositoryID).
		FilterField("date", ">=", startDate).
		FilterField("date", "<=", endDate).
		Order("date")

	_, err := c.client.GetAll(ctx, query, &metrics)
	return metrics, err
}

// Team Member operations

// SaveTeamMembers saves team members
func (c *Client) SaveTeamMembers(ctx context.Context, members []*model.TeamMember) error {
	if len(members) == 0 {
		return nil
	}

	keys := make([]*datastore.Key, len(members))
	for i, m := range members {
		keys[i] = datastore.NameKey(KindTeamMember, m.ID, nil)
	}

	_, err := c.client.PutMulti(ctx, keys, members)
	return err
}

// ListTeamMembers lists all team members
func (c *Client) ListTeamMembers(ctx context.Context) ([]*model.TeamMember, error) {
	var members []*model.TeamMember
	query := datastore.NewQuery(KindTeamMember).Order("login")
	_, err := c.client.GetAll(ctx, query, &members)
	return members, err
}

// Sprint operations

// SaveSprint saves a sprint
func (c *Client) SaveSprint(ctx context.Context, sprint *model.Sprint) error {
	key := datastore.NameKey(KindSprint, sprint.ID, nil)
	_, err := c.client.Put(ctx, key, sprint)
	return err
}

// GetSprint gets a sprint by ID
func (c *Client) GetSprint(ctx context.Context, id string) (*model.Sprint, error) {
	key := datastore.NameKey(KindSprint, id, nil)
	sprint := &model.Sprint{}
	if err := c.client.Get(ctx, key, sprint); err != nil {
		return nil, err
	}
	return sprint, nil
}

// ListSprints lists sprints for a repository
func (c *Client) ListSprints(ctx context.Context, repositoryID string) ([]*model.Sprint, error) {
	var sprints []*model.Sprint
	query := datastore.NewQuery(KindSprint).
		FilterField("repository_id", "=", repositoryID).
		Order("-start_date")

	_, err := c.client.GetAll(ctx, query, &sprints)
	return sprints, err
}

// QueryOptions options for queries
type QueryOptions struct {
	Since  time.Time
	Until  time.Time
	Limit  int
	Offset int
}

// MetricsCacheEntry is a cache entry stored in Datastore.
// Key format: "{endpoint}:{reposHash}:{start}:{end}".
// e.g. "metrics/cycle-time:all:2026-01-06:2026-02-06"
// e.g. "metrics/cycle-time:a1b2c3:2026-01-06:2026-02-06"
// e.g. "team/members/14109108/stats:all:2026-01-06:2026-02-06"
type MetricsCacheEntry struct {
	Key       string    `datastore:"key"`
	Body      []byte    `datastore:"body,noindex"`
	CreatedAt time.Time `datastore:"created_at"`
	TTLSec    int       `datastore:"ttl_sec"`
}

// GetMetricsCache retrieves cache from Datastore. Returns nil if expired.
func (c *Client) GetMetricsCache(ctx context.Context, cacheKey string) ([]byte, error) {
	key := datastore.NameKey(KindMetricsCache, cacheKey, nil)
	entry := &MetricsCacheEntry{}
	if err := c.client.Get(ctx, key, entry); err != nil {
		return nil, err
	}

	// Check TTL expiration
	if time.Since(entry.CreatedAt) > time.Duration(entry.TTLSec)*time.Second {
		return nil, fmt.Errorf("cache expired")
	}

	return entry.Body, nil
}

// PutMetricsCache stores cache in Datastore.
func (c *Client) PutMetricsCache(ctx context.Context, cacheKey string, body []byte, ttlSec int) error {
	key := datastore.NameKey(KindMetricsCache, cacheKey, nil)
	entry := &MetricsCacheEntry{
		Key:       cacheKey,
		Body:      body,
		CreatedAt: time.Now(),
		TTLSec:    ttlSec,
	}
	_, err := c.client.Put(ctx, key, entry)
	return err
}

// DataDateRange represents the date range of stored data for a repository.
type DataDateRange struct {
	RepositoryID string     `json:"repositoryId"`
	OldestDate   *time.Time `json:"oldestDate,omitempty"`
	NewestDate   *time.Time `json:"newestDate,omitempty"`
	PRCount      int        `json:"prCount"`
}

// GetDataDateRange retrieves the oldest and newest PR dates for a repository.
// Uses only descending order to avoid requiring additional composite indexes.
func (c *Client) GetDataDateRange(ctx context.Context, repositoryID string) (*DataDateRange, error) {
	result := &DataDateRange{RepositoryID: repositoryID}

	// Get all PR created_at dates using projection query (lightweight)
	type prDate struct {
		CreatedAt time.Time `datastore:"created_at"`
	}
	var dates []prDate
	q := datastore.NewQuery(KindPullRequest).
		FilterField("repository_id", "=", repositoryID).
		Order("-created_at").
		Project("created_at")
	if _, err := c.client.GetAll(ctx, q, &dates); err != nil {
		return nil, fmt.Errorf("failed to get PR dates: %w", err)
	}

	result.PRCount = len(dates)
	if len(dates) == 0 {
		return result, nil
	}

	// Descending order: first = newest, last = oldest
	newest := dates[0].CreatedAt
	oldest := dates[len(dates)-1].CreatedAt
	result.NewestDate = &newest
	result.OldestDate = &oldest

	return result, nil
}

// BotUser operations

// SaveBotUser saves a custom bot user.
func (c *Client) SaveBotUser(ctx context.Context, botUser *model.BotUser) error {
	key := datastore.NameKey(KindBotUser, botUser.Username, nil)
	_, err := c.client.Put(ctx, key, botUser)
	return err
}

// ListBotUsers retrieves the list of custom bot users.
func (c *Client) ListBotUsers(ctx context.Context) ([]*model.BotUser, error) {
	var botUsers []*model.BotUser
	query := datastore.NewQuery(KindBotUser).Order("username")
	_, err := c.client.GetAll(ctx, query, &botUsers)
	return botUsers, err
}

// DeleteBotUser deletes a custom bot user.
func (c *Client) DeleteBotUser(ctx context.Context, username string) error {
	key := datastore.NameKey(KindBotUser, username, nil)
	return c.client.Delete(ctx, key)
}

// ListBotUsernames retrieves a list of custom bot usernames.
func (c *Client) ListBotUsernames(ctx context.Context) ([]string, error) {
	botUsers, err := c.ListBotUsers(ctx)
	if err != nil {
		return nil, err
	}
	usernames := make([]string, len(botUsers))
	for i, bu := range botUsers {
		usernames[i] = bu.Username
	}
	return usernames, nil
}

// SyncLock operations

// AcquireSyncLock acquires an exclusive lock using a transaction.
// 既存ロックが有効期限内の場合はエラーを返す。
func (c *Client) AcquireSyncLock(ctx context.Context, lockID, lockedBy string, ttl time.Duration) error {
	key := datastore.NameKey(KindSyncLock, lockID, nil)

	_, err := c.client.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		var existing model.SyncLock
		if err := tx.Get(key, &existing); err == nil {
			// Lock exists and is still valid; acquisition fails
			if time.Now().Before(existing.ExpiresAt) {
				return fmt.Errorf("lock already held by %s until %s", existing.LockedBy, existing.ExpiresAt.Format(time.RFC3339))
			}
		}

		// Write new lock
		now := time.Now()
		lock := &model.SyncLock{
			ID:        lockID,
			LockedBy:  lockedBy,
			LockedAt:  now,
			ExpiresAt: now.Add(ttl),
		}
		_, err := tx.Put(key, lock)
		return err
	})

	return err
}

// ReleaseSyncLock deletes the lock if lockedBy matches within a transaction.
// トランザクション内で lockedBy が一致するロックを削除する。
func (c *Client) ReleaseSyncLock(ctx context.Context, lockID, lockedBy string) error {
	key := datastore.NameKey(KindSyncLock, lockID, nil)

	_, err := c.client.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		var existing model.SyncLock
		if err := tx.Get(key, &existing); err != nil {
			// No lock exists; nothing to do
			return nil
		}

		// Not our lock; do not release
		if existing.LockedBy != lockedBy {
			return nil
		}

		return tx.Delete(key)
	})

	return err
}

// GetSyncLock retrieves lock information (for debugging/monitoring).
// ロック情報を取得する（デバッグ・監視用）。
func (c *Client) GetSyncLock(ctx context.Context, lockID string) (*model.SyncLock, error) {
	key := datastore.NameKey(KindSyncLock, lockID, nil)
	lock := &model.SyncLock{}
	if err := c.client.Get(ctx, key, lock); err != nil {
		return nil, err
	}
	return lock, nil
}

// DeleteAllMetricsCache deletes all metrics cache entries.
func (c *Client) DeleteAllMetricsCache(ctx context.Context) error {
	query := datastore.NewQuery(KindMetricsCache).KeysOnly()
	keys, err := c.client.GetAll(ctx, query, nil)
	if err != nil {
		return fmt.Errorf("failed to list cache keys: %w", err)
	}
	if len(keys) == 0 {
		return nil
	}
	return c.client.DeleteMulti(ctx, keys)
}
