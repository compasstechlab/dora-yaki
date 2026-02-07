package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/compasstechlab/dora-yaki/internal/api/middleware"
	"github.com/compasstechlab/dora-yaki/internal/config"
	"github.com/compasstechlab/dora-yaki/internal/datastore"
	"github.com/compasstechlab/dora-yaki/internal/domain/model"
	"github.com/compasstechlab/dora-yaki/internal/github"
	"github.com/compasstechlab/dora-yaki/internal/metrics"
	"github.com/compasstechlab/dora-yaki/internal/timeutil"
)

const (
	syncLockID = "sync-job"
	// processStartGuard is the minimum elapsed time from ProcessStartAt (prevents premature re-execution).
	processStartGuard = 10 * time.Minute
)

// JobHandler handles batch job API requests.
// バッチジョブ系APIを処理する。
type JobHandler struct {
	ds        *datastore.Client
	gh        *github.Client
	collector *github.Collector
	logger    *slog.Logger
	cache     *middleware.ResponseCache
	cfg       *config.Config
}

// NewJobHandler creates a new JobHandler.
func NewJobHandler(ds *datastore.Client, gh *github.Client, logger *slog.Logger, cache *middleware.ResponseCache, cfg *config.Config) *JobHandler {
	return &JobHandler{
		ds:        ds,
		gh:        gh,
		collector: github.NewCollector(gh, logger),
		logger:    logger,
		cache:     cache,
		cfg:       cfg,
	}
}

// jobSyncRequest is a request parsed from both query parameters and JSON body.
// クエリパラメータと JSON body 両方から読み取るリクエスト。
type jobSyncRequest struct {
	Range      string `json:"range"`
	Interval   int    `json:"interval"`    // Sync interval in minutes (0 = use config value)
	Repo       string `json:"repo"`        // Target repository name (owner/name or name)
	NoLock     bool   `json:"nolock"`      // Skip Datastore lock mechanism
	Force      bool   `json:"force"`       // Disable ProcessStartAt validation when repo is specified
	ClearCache bool   `json:"clear_cache"` // Invalidate response cache after sync (default: false)
}

// JobSyncResponse is the sync job response.
// 同期ジョブのレスポンス。
type JobSyncResponse struct {
	Status       string           `json:"status"`
	Message      string           `json:"message"`
	TotalRepos   int              `json:"totalRepos"`
	SyncedRepos  int              `json:"syncedRepos"`
	SkippedRepos int              `json:"skippedRepos"`
	Results      []RepoSyncResult `json:"results,omitempty"`
	StartedAt    time.Time        `json:"startedAt"`
	FinishedAt   time.Time        `json:"finishedAt"`
	DurationSec  float64          `json:"durationSec"`
}

// RepoSyncResult represents the sync result for an individual repository.
// 個別リポジトリの同期結果。
type RepoSyncResult struct {
	RepositoryID string `json:"repositoryId"`
	FullName     string `json:"fullName"`
	Success      bool   `json:"success"`
	Error        string `json:"error,omitempty"`
	PullRequests int    `json:"pullRequests"`
	Reviews      int    `json:"reviews"`
	Deployments  int    `json:"deployments"`
}

// parseSyncRequest parses parameters from both query parameters and JSON body.
// Priority: JSON body > query params > default values
func parseSyncRequest(r *http.Request) jobSyncRequest {
	q := r.URL.Query()
	interval, _ := strconv.Atoi(q.Get("interval"))

	nolock, _ := strconv.ParseBool(q.Get("nolock"))
	force, _ := strconv.ParseBool(q.Get("force"))
	clearCache, _ := strconv.ParseBool(q.Get("clear_cache"))

	req := jobSyncRequest{
		Range:      q.Get("range"),
		Interval:   interval,
		Repo:       q.Get("repo"),
		NoLock:     nolock,
		Force:      force,
		ClearCache: clearCache,
	}

	// Override with JSON body if present
	if r.Body != nil && r.ContentLength != 0 {
		var body jobSyncRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
			if body.Range != "" {
				req.Range = body.Range
			}
			if body.Interval > 0 {
				req.Interval = body.Interval
			}
			if body.Repo != "" {
				req.Repo = body.Repo
			}
			if body.NoLock {
				req.NoLock = true
			}
			if body.Force {
				req.Force = true
			}
			if body.ClearCache {
				req.ClearCache = true
			}
		}
	}

	if req.Range == "" {
		req.Range = "day"
	}
	return req
}

// Sync synchronizes a single repository that matches the criteria.
// Cloud Scheduler からの定期実行を想定。
func (h *JobHandler) Sync(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	startedAt := time.Now()

	// Parse request parameters (query + JSON body)
	req := parseSyncRequest(r)

	// Generate instance ID
	instanceID := fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano())

	h.logger.Info("sync job started",
		"instanceID", instanceID,
		"range", req.Range,
		"interval", req.Interval,
		"repo", req.Repo,
		"nolock", req.NoLock,
		"force", req.Force,
		"clear_cache", req.ClearCache,
	)

	// Acquire exclusive lock (skip if nolock=true)
	if !req.NoLock {
		if err := h.ds.AcquireSyncLock(ctx, syncLockID, instanceID, h.cfg.SyncLockTTL()); err != nil {
			h.logger.Warn("sync job skipped: lock already held", "error", err)
			respondJSON(w, http.StatusConflict, map[string]string{
				"status":  "skipped",
				"message": fmt.Sprintf("sync job already running: %s", err.Error()),
			})
			return
		}
		defer func() {
			if err := h.ds.ReleaseSyncLock(ctx, syncLockID, instanceID); err != nil {
				h.logger.Error("failed to release sync lock", "error", err)
			}
		}()
	}

	// Get all repositories
	repos, err := h.ds.ListRepositories(ctx)
	if err != nil {
		h.logger.Error("failed to list repositories", "error", err)
		http.Error(w, "failed to list repositories", http.StatusInternalServerError)
		return
	}

	// Select one sync target
	target := h.pickSyncTarget(repos, req)

	if target == nil {
		finishedAt := time.Now()
		h.logger.Info("sync job completed: no eligible repository found")
		respondJSON(w, http.StatusOK, &JobSyncResponse{
			Status:       "completed",
			Message:      "no eligible repository found",
			TotalRepos:   len(repos),
			SyncedRepos:  0,
			SkippedRepos: len(repos),
			StartedAt:    startedAt,
			FinishedAt:   finishedAt,
			DurationSec:  finishedAt.Sub(startedAt).Seconds(),
		})
		return
	}

	// Update ProcessStartAt (sync start marker)
	now := time.Now()
	target.ProcessStartAt = &now
	if err := h.ds.SaveRepository(ctx, target); err != nil {
		h.logger.Error("failed to update process_start_at", "error", err)
	}

	// Execute sync
	result := h.syncSingleRepo(ctx, target, req.Range)

	// Invalidate cache only when explicitly requested
	if req.ClearCache && result.Success && h.cache != nil {
		h.cache.Invalidate()
		h.logger.Info("response cache invalidated after job sync")
	}

	finishedAt := time.Now()
	syncedCount := 0
	if result.Success {
		syncedCount = 1
	}

	response := &JobSyncResponse{
		Status:       "completed",
		Message:      fmt.Sprintf("synced %d/%d repositories", syncedCount, len(repos)),
		TotalRepos:   len(repos),
		SyncedRepos:  syncedCount,
		SkippedRepos: len(repos) - 1,
		Results:      []RepoSyncResult{result},
		StartedAt:    startedAt,
		FinishedAt:   finishedAt,
		DurationSec:  finishedAt.Sub(startedAt).Seconds(),
	}

	h.logger.Info("sync job completed",
		"totalRepos", len(repos),
		"syncedRepo", target.FullName,
		"success", result.Success,
		"durationSec", response.DurationSec,
	)

	respondJSON(w, http.StatusOK, response)
}

// pickSyncTarget selects one repository to sync.
//
// When repo is specified:
//   - Find matching repository by FullName or Name
//   - Skip ProcessStartAt check if force=true
//   - Interval check always applies
//
// When repo is not specified:
//   - From repos where interval has passed AND ProcessStartAt is >10min ago
//   - Select the one with the oldest LastSyncedAt
func (h *JobHandler) pickSyncTarget(repos []*model.Repository, req jobSyncRequest) *model.Repository {
	// Use request interval if specified, otherwise use config value
	syncInterval := h.cfg.SyncInterval()
	if req.Interval > 0 {
		syncInterval = time.Duration(req.Interval) * time.Minute
	}
	now := timeutil.Now()

	// When specific repo is requested
	if req.Repo != "" {
		for _, repo := range repos {
			if !matchRepoName(repo, req.Repo) {
				continue
			}

			// Check interval
			if repo.LastSyncedAt != nil && now.Sub(*repo.LastSyncedAt) < syncInterval {
				h.logger.Info("skipping repository: recently synced",
					"repository", repo.FullName,
					"lastSyncedAt", repo.LastSyncedAt,
					"interval", syncInterval,
				)
				return nil
			}

			// Check ProcessStartAt if force=false
			if !req.Force {
				if repo.ProcessStartAt != nil && now.Sub(*repo.ProcessStartAt) < processStartGuard {
					h.logger.Info("skipping repository: process recently started",
						"repository", repo.FullName,
						"processStartAt", repo.ProcessStartAt,
					)
					return nil
				}
			}

			return repo
		}
		h.logger.Warn("specified repository not found", "repo", req.Repo)
		return nil
	}

	// No repo specified: select the oldest eligible repository
	var target *model.Repository
	var oldestSync time.Time

	for _, repo := range repos {
		// Check interval
		if repo.LastSyncedAt != nil && now.Sub(*repo.LastSyncedAt) < syncInterval {
			continue
		}

		// ProcessStartAt check: skip if less than 10min elapsed
		if repo.ProcessStartAt != nil && now.Sub(*repo.ProcessStartAt) < processStartGuard {
			h.logger.Info("skipping repository: process recently started",
				"repository", repo.FullName,
				"processStartAt", repo.ProcessStartAt,
			)
			continue
		}

		// Select repository with the oldest LastSyncedAt
		repoSync := time.Time{} // zero value = never synced = highest priority
		if repo.LastSyncedAt != nil {
			repoSync = *repo.LastSyncedAt
		}

		if target == nil || repoSync.Before(oldestSync) {
			target = repo
			oldestSync = repoSync
		}
	}

	return target
}

// matchRepoName checks if the repository matches the given name.
// Matches by FullName (owner/name) or Name exact match.
func matchRepoName(repo *model.Repository, name string) bool {
	return repo.FullName == name || repo.Name == name
}

// syncSingleRepo executes sync for a single repository.
// 単一リポジトリの同期を実行する。
func (h *JobHandler) syncSingleRepo(ctx context.Context, repo *model.Repository, syncRange string) RepoSyncResult {
	result := RepoSyncResult{
		RepositoryID: repo.ID,
		FullName:     repo.FullName,
	}

	opts := github.CollectOptionsForRange(syncRange)

	// Collect data from GitHub
	data, err := h.collector.CollectAll(ctx, repo.Owner, repo.Name, opts)
	if err != nil {
		h.logger.Error("failed to sync repository",
			"repository", repo.FullName,
			"error", err,
		)
		result.Error = err.Error()
		return result
	}

	// Save to Datastore
	if err := h.ds.SaveRepository(ctx, data.Repository); err != nil {
		h.logger.Error("failed to save repository", "error", err)
	}
	if err := h.ds.SavePullRequests(ctx, data.PullRequests); err != nil {
		h.logger.Error("failed to save pull requests", "error", err)
	}
	if err := h.ds.SaveReviews(ctx, data.Reviews); err != nil {
		h.logger.Error("failed to save reviews", "error", err)
	}
	if err := h.ds.SaveDeployments(ctx, data.Deployments); err != nil {
		h.logger.Error("failed to save deployments", "error", err)
	}
	if err := h.ds.SaveTeamMembers(ctx, data.TeamMembers); err != nil {
		h.logger.Error("failed to save team members", "error", err)
	}

	// Aggregate daily metrics
	aggregator := metrics.NewAggregator()
	endDate := timeutil.Now()
	startDate := opts.Since
	dailyMetrics := aggregator.AggregateRange(
		repo.ID,
		startDate,
		endDate,
		data.PullRequests,
		data.Reviews,
		data.Deployments,
	)
	if err := h.ds.SaveDailyMetricsBatch(ctx, dailyMetrics); err != nil {
		h.logger.Error("failed to save daily metrics", "error", err)
	}

	// Update LastSyncedAt
	now := time.Now()
	data.Repository.LastSyncedAt = &now
	if err := h.ds.SaveRepository(ctx, data.Repository); err != nil {
		h.logger.Error("failed to update last_synced_at", "error", err)
	}

	result.Success = true
	result.PullRequests = len(data.PullRequests)
	result.Reviews = len(data.Reviews)
	result.Deployments = len(data.Deployments)

	h.logger.Info("repository sync completed",
		"repository", repo.FullName,
		"pullRequests", result.PullRequests,
		"reviews", result.Reviews,
		"deployments", result.Deployments,
	)

	return result
}
