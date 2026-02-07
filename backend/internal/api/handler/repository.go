package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"reflect"
	"time"

	"github.com/compasstechlab/dora-yaki/internal/api/middleware"
	"github.com/compasstechlab/dora-yaki/internal/datastore"
	"github.com/compasstechlab/dora-yaki/internal/domain/model"
	"github.com/compasstechlab/dora-yaki/internal/github"
	"github.com/compasstechlab/dora-yaki/internal/metrics"
	"github.com/compasstechlab/dora-yaki/internal/timeutil"
)

// RepositoryHandler handles repository-related API requests
type RepositoryHandler struct {
	ds        *datastore.Client
	gh        *github.Client
	collector *github.Collector
	logger    *slog.Logger
	cache     *middleware.ResponseCache
}

// NewRepositoryHandler creates a new RepositoryHandler
func NewRepositoryHandler(ds *datastore.Client, gh *github.Client, logger *slog.Logger, cache *middleware.ResponseCache) *RepositoryHandler {
	return &RepositoryHandler{
		ds:        ds,
		gh:        gh,
		collector: github.NewCollector(gh, logger),
		logger:    logger,
		cache:     cache,
	}
}

// AddRepositoryRequest request body for adding a repository
type AddRepositoryRequest struct {
	Owner string `json:"owner"`
	Name  string `json:"name"`
}

// BatchAddRepositoryRequest is a batch add request.
type BatchAddRepositoryRequest struct {
	Repositories []AddRepositoryRequest `json:"repositories"`
}

// BatchAddResult represents an individual result of a batch add.
type BatchAddResult struct {
	Owner      string            `json:"owner"`
	Name       string            `json:"name"`
	Success    bool              `json:"success"`
	Error      string            `json:"error,omitempty"`
	Repository *model.Repository `json:"repository,omitempty"`
}

// BatchAdd adds multiple repositories in a batch.
func (h *RepositoryHandler) BatchAdd(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req BatchAddRepositoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	const maxBatchSize = 100
	if len(req.Repositories) == 0 {
		http.Error(w, "repositories are required", http.StatusBadRequest)
		return
	}
	if len(req.Repositories) > maxBatchSize {
		http.Error(w, "too many repositories (max 100)", http.StatusBadRequest)
		return
	}

	results := make([]BatchAddResult, 0, len(req.Repositories))
	for _, repoReq := range req.Repositories {
		result := BatchAddResult{
			Owner: repoReq.Owner,
			Name:  repoReq.Name,
		}

		if repoReq.Owner == "" || repoReq.Name == "" {
			result.Error = "owner and name are required"
			results = append(results, result)
			continue
		}

		// Fetch repository info from GitHub
		repo, err := h.gh.GetRepository(ctx, repoReq.Owner, repoReq.Name)
		if err != nil {
			h.logger.Error("failed to get repository from GitHub", "error", err, "owner", repoReq.Owner, "name", repoReq.Name)
			result.Error = "repository not found on GitHub"
			results = append(results, result)
			continue
		}

		// Save to datastore
		if err := h.ds.SaveRepository(ctx, repo); err != nil {
			h.logger.Error("failed to save repository", "error", err, "owner", repoReq.Owner, "name", repoReq.Name)
			result.Error = "failed to save repository"
			results = append(results, result)
			continue
		}

		result.Success = true
		result.Repository = repo
		results = append(results, result)
	}

	respondJSON(w, http.StatusOK, results)
}

// List returns all repositories
func (h *RepositoryHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	repos, err := h.ds.ListRepositories(ctx)
	if err != nil {
		h.logger.Error("failed to list repositories", "error", err)
		http.Error(w, "failed to list repositories", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, repos)
}

// Add adds a new repository
func (h *RepositoryHandler) Add(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req AddRepositoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Owner == "" || req.Name == "" {
		http.Error(w, "owner and name are required", http.StatusBadRequest)
		return
	}

	// Fetch repository from GitHub
	repo, err := h.gh.GetRepository(ctx, req.Owner, req.Name)
	if err != nil {
		h.logger.Error("failed to get repository from GitHub", "error", err)
		http.Error(w, "repository not found on GitHub", http.StatusNotFound)
		return
	}

	// Save to datastore
	if err := h.ds.SaveRepository(ctx, repo); err != nil {
		h.logger.Error("failed to save repository", "error", err)
		http.Error(w, "failed to save repository", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusCreated, repo)
}

// Get returns a specific repository
func (h *RepositoryHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := getPathParam(r, "id")

	repo, err := h.ds.GetRepository(ctx, id)
	if err != nil {
		http.Error(w, "repository not found", http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, repo)
}

// Delete removes a repository
func (h *RepositoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := getPathParam(r, "id")

	if err := h.ds.DeleteRepository(ctx, id); err != nil {
		h.logger.Error("failed to delete repository", "error", err)
		http.Error(w, "failed to delete repository", http.StatusInternalServerError)
		return
	}

	// Invalidate cache after deletion
	if h.cache != nil {
		h.cache.Invalidate()
		h.logger.Info("response cache invalidated after delete")
	}

	w.WriteHeader(http.StatusNoContent)
}

// SyncResponse response body for sync operation
type SyncResponse struct {
	Repository   *model.Repository `json:"repository"`
	PullRequests int               `json:"pullRequests"`
	Reviews      int               `json:"reviews"`
	Deployments  int               `json:"deployments"`
	TeamMembers  int               `json:"teamMembers"`
	SyncedAt     time.Time         `json:"syncedAt"`
}

// Sync triggers a data sync for a repository
func (h *RepositoryHandler) Sync(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := getPathParam(r, "id")

	// Resolve owner/name from repository ID
	repo, err := h.ds.GetRepository(ctx, id)
	if err != nil {
		h.logger.Error("failed to get repository", "error", err, "id", id)
		http.Error(w, "repository not found", http.StatusNotFound)
		return
	}
	owner, name := repo.Owner, repo.Name

	// Get sync range parameter (defaults to "full")
	syncRange := r.URL.Query().Get("range")
	if syncRange == "" {
		syncRange = "full"
	}
	opts := github.CollectOptionsForRange(syncRange)

	// Collect data from GitHub
	data, err := h.collector.CollectAll(ctx, owner, name, opts)
	if err != nil {
		h.logger.Error("failed to sync repository", "error", err)
		http.Error(w, "failed to sync repository", http.StatusInternalServerError)
		return
	}

	// Save data to datastore
	h.logger.Info("saving collected data to datastore",
		"repository", data.Repository.FullName,
		"prs", len(data.PullRequests),
		"reviews", len(data.Reviews),
		"deployments", len(data.Deployments),
		"members", len(data.TeamMembers),
	)

	// Save each entity, logging errors but continuing on failure
	saveAndLog := func(fn func() error, entity string, count int) {
		if err := fn(); err != nil {
			h.logger.Error("failed to save "+entity, "error", err)
			return
		}
		h.logger.Info("saved "+entity, "count", count)
	}

	saveAndLog(func() error { return h.ds.SaveRepository(ctx, data.Repository) }, "repository", 1)
	saveAndLog(func() error { return h.ds.SavePullRequests(ctx, data.PullRequests) }, "pull requests", len(data.PullRequests))
	saveAndLog(func() error { return h.ds.SaveReviews(ctx, data.Reviews) }, "reviews", len(data.Reviews))
	saveAndLog(func() error { return h.ds.SaveDeployments(ctx, data.Deployments) }, "deployments", len(data.Deployments))
	saveAndLog(func() error { return h.ds.SaveTeamMembers(ctx, data.TeamMembers) }, "team members", len(data.TeamMembers))

	// Aggregate daily metrics
	h.logger.Info("aggregating daily metrics")
	aggregator := metrics.NewAggregator()
	endDate := timeutil.Now()
	startDate := opts.Since

	dailyMetrics := aggregator.AggregateRange(
		id,
		startDate,
		endDate,
		data.PullRequests,
		data.Reviews,
		data.Deployments,
	)

	if err := h.ds.SaveDailyMetricsBatch(ctx, dailyMetrics); err != nil {
		h.logger.Error("failed to save daily metrics", "error", err)
	}
	h.logger.Info("saved daily metrics", "count", len(dailyMetrics))

	// Update last sync timestamp
	now := time.Now()
	data.Repository.LastSyncedAt = &now
	if err := h.ds.SaveRepository(ctx, data.Repository); err != nil {
		h.logger.Error("failed to update last_synced_at", "error", err)
	}

	// Invalidate cache after sync
	if h.cache != nil {
		h.cache.Invalidate()
		h.logger.Info("response cache invalidated after sync")
	}

	response := &SyncResponse{
		Repository:   data.Repository,
		PullRequests: len(data.PullRequests),
		Reviews:      len(data.Reviews),
		Deployments:  len(data.Deployments),
		TeamMembers:  len(data.TeamMembers),
		SyncedAt:     time.Now(),
	}

	respondJSON(w, http.StatusOK, response)
}

// DateRanges returns data date ranges for all repositories.
func (h *RepositoryHandler) DateRanges(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	repos, err := h.ds.ListRepositories(ctx)
	if err != nil {
		h.logger.Error("failed to list repositories", "error", err)
		http.Error(w, "failed to list repositories", http.StatusInternalServerError)
		return
	}

	ranges := make([]*datastore.DataDateRange, 0, len(repos))
	for _, repo := range repos {
		dr, err := h.ds.GetDataDateRange(ctx, repo.ID)
		if err != nil {
			h.logger.Error("failed to get date range", "error", err, "repository", repo.FullName)
			continue
		}
		ranges = append(ranges, dr)
	}

	respondJSON(w, http.StatusOK, ranges)
}

// Helper functions

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data == nil {
		return
	}

	// Ensure nil slices are serialized as [] instead of null
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Slice && v.IsNil() {
		_, _ = w.Write([]byte("[]\n"))
		return
	}

	_ = json.NewEncoder(w).Encode(data)
}

func getPathParam(r *http.Request, name string) string {
	return r.PathValue(name)
}
