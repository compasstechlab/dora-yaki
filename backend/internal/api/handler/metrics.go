package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"time"

	"github.com/compasstechlab/dora-yaki/internal/datastore"
	"github.com/compasstechlab/dora-yaki/internal/domain/model"
	"github.com/compasstechlab/dora-yaki/internal/metrics"
	"github.com/compasstechlab/dora-yaki/internal/timeutil"
)

// MetricsHandler handles metrics-related API requests
type MetricsHandler struct {
	ds         *datastore.Client
	calculator *metrics.Calculator
	logger     *slog.Logger
}

// NewMetricsHandler creates a new MetricsHandler
func NewMetricsHandler(ds *datastore.Client, logger *slog.Logger) *MetricsHandler {
	return &MetricsHandler{
		ds:         ds,
		calculator: metrics.NewCalculator(),
		logger:     logger,
	}
}

// botFilter holds bot filtering settings.
type botFilter struct {
	excludeBots bool
	botsOnly    bool
}

// parseBotFilter parses bot filter settings from query parameters.
func parseBotFilter(r *http.Request) botFilter {
	q := r.URL.Query()
	botsOnly := q.Get("bots_only") == "true"
	// When bots_only is set, exclude_bots is ignored
	if botsOnly {
		return botFilter{excludeBots: false, botsOnly: true}
	}
	// Default to true when exclude_bots is not specified
	excludeBots := q.Get("exclude_bots") != "false"
	return botFilter{excludeBots: excludeBots, botsOnly: false}
}

// getBotUsernames retrieves custom bot username list from Datastore.
func (h *MetricsHandler) getBotUsernames(ctx context.Context) []string {
	usernames, err := h.ds.ListBotUsernames(ctx)
	if err != nil {
		h.logger.Warn("failed to get bot usernames", "error", err)
		return nil
	}
	return usernames
}

// parseDateRange parses date range from query params
func parseDateRange(r *http.Request) (time.Time, time.Time) {
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	endDate := timeutil.Now()
	startDate := endDate.AddDate(0, -1, 0) // Default: last month

	if startStr != "" {
		if t, err := timeutil.ParseDate(startStr); err == nil {
			startDate = t
		}
	}

	if endStr != "" {
		if t, err := timeutil.ParseDate(endStr); err == nil {
			endDate = t
		}
	}

	return startDate, endDate
}

// getRepositoryIDs retrieves multiple repository IDs. Returns all repositories if empty.
func (h *MetricsHandler) getRepositoryIDs(r *http.Request) ([]string, error) {
	ids := r.URL.Query()["repository"]
	if len(ids) > 0 {
		return ids, nil
	}
	// Return all registered repositories if not specified
	repos, err := h.ds.ListRepositories(r.Context())
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}
	all := make([]string, len(repos))
	for i, repo := range repos {
		all[i] = repo.ID
	}
	return all, nil
}

// collectPullRequests collects and merges PRs from multiple repositories.
func (h *MetricsHandler) collectPullRequests(ctx context.Context, repoIDs []string, start, end time.Time) ([]*model.PullRequest, error) {
	var result []*model.PullRequest
	for _, id := range repoIDs {
		prs, err := h.ds.ListPullRequestsByDateRange(ctx, id, start, end)
		if err != nil {
			h.logger.Warn("failed to list pull requests for repo", "repository", id, "error", err)
			continue
		}
		result = append(result, prs...)
	}
	return result, nil
}

// collectReviews collects and merges reviews from multiple repositories.
func (h *MetricsHandler) collectReviews(ctx context.Context, repoIDs []string, start, end time.Time) ([]*model.Review, error) {
	var result []*model.Review
	for _, id := range repoIDs {
		reviews, err := h.ds.ListReviewsByDateRange(ctx, id, start, end)
		if err != nil {
			h.logger.Warn("failed to list reviews for repo", "repository", id, "error", err)
			continue
		}
		result = append(result, reviews...)
	}
	return result, nil
}

// collectDeployments collects and merges deployments from multiple repositories.
func (h *MetricsHandler) collectDeployments(ctx context.Context, repoIDs []string, start, end time.Time) ([]*model.Deployment, error) {
	var result []*model.Deployment
	for _, id := range repoIDs {
		deployments, err := h.ds.ListDeployments(ctx, id, &datastore.QueryOptions{
			Since: start,
			Until: end,
		})
		if err != nil {
			h.logger.Warn("failed to list deployments for repo", "repository", id, "error", err)
			continue
		}
		result = append(result, deployments...)
	}
	return result, nil
}

// collectDailyMetrics collects daily metrics from multiple repositories and aggregates by date.
func (h *MetricsHandler) collectDailyMetrics(ctx context.Context, repoIDs []string, start, end time.Time) ([]*model.DailyMetrics, error) {
	// Group by date key
	grouped := make(map[string]*model.DailyMetrics)

	for _, id := range repoIDs {
		daily, err := h.ds.ListDailyMetrics(ctx, id, start, end)
		if err != nil {
			h.logger.Warn("failed to list daily metrics for repo", "repository", id, "error", err)
			continue
		}
		for _, dm := range daily {
			dateKey := dm.Date.Format("2006-01-02")
			agg, ok := grouped[dateKey]
			if !ok {
				// First entry: copy and store
				copied := *dm
				copied.RepositoryID = "" // Clear repository ID since data is aggregated
				grouped[dateKey] = &copied
				continue
			}
			// Cycle time: weighted average based on PRsMerged
			prevMerged := agg.PRsMerged
			newMerged := dm.PRsMerged
			totalMerged := prevMerged + newMerged
			if totalMerged > 0 {
				agg.AvgCycleTime = weightedAvg(agg.AvgCycleTime, prevMerged, dm.AvgCycleTime, newMerged)
				agg.AvgCodingTime = weightedAvg(agg.AvgCodingTime, prevMerged, dm.AvgCodingTime, newMerged)
				agg.AvgPickupTime = weightedAvg(agg.AvgPickupTime, prevMerged, dm.AvgPickupTime, newMerged)
				agg.AvgReviewTime = weightedAvg(agg.AvgReviewTime, prevMerged, dm.AvgReviewTime, newMerged)
				agg.AvgMergeTime = weightedAvg(agg.AvgMergeTime, prevMerged, dm.AvgMergeTime, newMerged)
			}

			// Counts: sum up
			agg.PRsOpened += dm.PRsOpened
			agg.PRsMerged += dm.PRsMerged
			agg.PRsClosed += dm.PRsClosed
			agg.ReviewsSubmitted += dm.ReviewsSubmitted
			agg.TotalAdditions += dm.TotalAdditions
			agg.TotalDeletions += dm.TotalDeletions
			agg.DeploymentCount += dm.DeploymentCount
			agg.ActiveContributors += dm.ActiveContributors

			// AvgReviewsPerPR: recalculate based on total PR count
			totalOpened := agg.PRsOpened
			if totalOpened > 0 {
				agg.AvgReviewsPerPR = float64(agg.ReviewsSubmitted) / float64(totalOpened)
			}
		}
	}

	// Convert map to slice and sort by date ascending
	result := make([]*model.DailyMetrics, 0, len(grouped))
	for _, dm := range grouped {
		result = append(result, dm)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Date.Before(result[j].Date)
	})
	return result, nil
}

// weightedAvg calculates a weighted average.
func weightedAvg(val1 float64, weight1 int, val2 float64, weight2 int) float64 {
	total := weight1 + weight2
	if total == 0 {
		return 0
	}
	return (val1*float64(weight1) + val2*float64(weight2)) / float64(total)
}

// CycleTime returns cycle time metrics
func (h *MetricsHandler) CycleTime(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	startDate, endDate := parseDateRange(r)
	bf := parseBotFilter(r)

	repoIDs, err := h.getRepositoryIDs(r)
	if err != nil {
		h.logger.Error("failed to get repository IDs", "error", err)
		http.Error(w, "failed to get repository IDs", http.StatusInternalServerError)
		return
	}

	prs, err := h.collectPullRequests(ctx, repoIDs, startDate, endDate)
	if err != nil {
		h.logger.Error("failed to collect pull requests", "error", err)
		http.Error(w, "failed to get metrics", http.StatusInternalServerError)
		return
	}

	// Apply bot filtering
	botUsernames := h.getBotUsernames(ctx)
	prs = model.FilterPullRequestsByBot(prs, botUsernames, bf.excludeBots, bf.botsOnly)

	// Calculate cycle time metrics
	cycleTimeMetrics := h.calculator.CalculateCycleTime(prs, startDate, endDate)

	// Get daily breakdown
	dailyMetrics, err := h.collectDailyMetrics(ctx, repoIDs, startDate, endDate)
	if err != nil {
		h.logger.Warn("failed to get daily metrics", "error", err)
	} else {
		cycleTimeMetrics.DailyBreakdown = make([]model.DailyMetrics, 0, len(dailyMetrics))
		for _, dm := range dailyMetrics {
			cycleTimeMetrics.DailyBreakdown = append(cycleTimeMetrics.DailyBreakdown, *dm)
		}
	}

	respondJSON(w, http.StatusOK, cycleTimeMetrics)
}

// Reviews returns review analysis metrics
func (h *MetricsHandler) Reviews(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	startDate, endDate := parseDateRange(r)
	bf := parseBotFilter(r)

	repoIDs, err := h.getRepositoryIDs(r)
	if err != nil {
		h.logger.Error("failed to get repository IDs", "error", err)
		http.Error(w, "failed to get repository IDs", http.StatusInternalServerError)
		return
	}

	reviews, err := h.collectReviews(ctx, repoIDs, startDate, endDate)
	if err != nil {
		h.logger.Error("failed to collect reviews", "error", err)
		http.Error(w, "failed to get metrics", http.StatusInternalServerError)
		return
	}

	prs, err := h.collectPullRequests(ctx, repoIDs, startDate, endDate)
	if err != nil {
		h.logger.Error("failed to collect pull requests", "error", err)
		http.Error(w, "failed to get metrics", http.StatusInternalServerError)
		return
	}

	// Apply bot filtering
	botUsernames := h.getBotUsernames(ctx)
	reviews = model.FilterReviewsByBot(reviews, botUsernames, bf.excludeBots, bf.botsOnly)
	prs = model.FilterPullRequestsByBot(prs, botUsernames, bf.excludeBots, bf.botsOnly)

	reviewMetrics := h.calculator.CalculateReviewMetrics(reviews, prs, startDate, endDate)
	respondJSON(w, http.StatusOK, reviewMetrics)
}

// DORA returns DORA metrics
func (h *MetricsHandler) DORA(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	startDate, endDate := parseDateRange(r)
	bf := parseBotFilter(r)

	repoIDs, err := h.getRepositoryIDs(r)
	if err != nil {
		h.logger.Error("failed to get repository IDs", "error", err)
		http.Error(w, "failed to get repository IDs", http.StatusInternalServerError)
		return
	}

	prs, err := h.collectPullRequests(ctx, repoIDs, startDate, endDate)
	if err != nil {
		h.logger.Error("failed to collect pull requests", "error", err)
		http.Error(w, "failed to get metrics", http.StatusInternalServerError)
		return
	}

	deployments, err := h.collectDeployments(ctx, repoIDs, startDate, endDate)
	if err != nil {
		h.logger.Error("failed to collect deployments", "error", err)
		http.Error(w, "failed to get metrics", http.StatusInternalServerError)
		return
	}

	// Apply bot filtering
	botUsernames := h.getBotUsernames(ctx)
	prs = model.FilterPullRequestsByBot(prs, botUsernames, bf.excludeBots, bf.botsOnly)

	doraMetrics := h.calculator.CalculateDORAMetrics(prs, deployments, startDate, endDate)
	respondJSON(w, http.StatusOK, doraMetrics)
}

// ProductivityScore returns the productivity score
func (h *MetricsHandler) ProductivityScore(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	startDate, endDate := parseDateRange(r)
	bf := parseBotFilter(r)

	repoIDs, err := h.getRepositoryIDs(r)
	if err != nil {
		h.logger.Error("failed to get repository IDs", "error", err)
		http.Error(w, "failed to get repository IDs", http.StatusInternalServerError)
		return
	}

	prs, err := h.collectPullRequests(ctx, repoIDs, startDate, endDate)
	if err != nil {
		h.logger.Error("failed to collect pull requests", "error", err)
		http.Error(w, "failed to get metrics", http.StatusInternalServerError)
		return
	}

	reviews, err := h.collectReviews(ctx, repoIDs, startDate, endDate)
	if err != nil {
		h.logger.Error("failed to collect reviews", "error", err)
		http.Error(w, "failed to get metrics", http.StatusInternalServerError)
		return
	}

	deployments, err := h.collectDeployments(ctx, repoIDs, startDate, endDate)
	if err != nil {
		h.logger.Error("failed to collect deployments", "error", err)
		http.Error(w, "failed to get metrics", http.StatusInternalServerError)
		return
	}

	// Apply bot filtering
	botUsernames := h.getBotUsernames(ctx)
	prs = model.FilterPullRequestsByBot(prs, botUsernames, bf.excludeBots, bf.botsOnly)
	reviews = model.FilterReviewsByBot(reviews, botUsernames, bf.excludeBots, bf.botsOnly)

	cycleTime := h.calculator.CalculateCycleTime(prs, startDate, endDate)
	reviewMetrics := h.calculator.CalculateReviewMetrics(reviews, prs, startDate, endDate)
	doraMetrics := h.calculator.CalculateDORAMetrics(prs, deployments, startDate, endDate)

	score := h.calculator.CalculateProductivityScore(cycleTime, reviewMetrics, doraMetrics)

	// Set "all" for multiple repositories
	if len(repoIDs) == 1 {
		score.RepositoryID = repoIDs[0]
	} else {
		score.RepositoryID = "all"
	}
	score.Period = "custom"

	respondJSON(w, http.StatusOK, score)
}

// DailyMetrics returns aggregated daily metrics
func (h *MetricsHandler) DailyMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	startDate, endDate := parseDateRange(r)

	repoIDs, err := h.getRepositoryIDs(r)
	if err != nil {
		h.logger.Error("failed to get repository IDs", "error", err)
		http.Error(w, "failed to get repository IDs", http.StatusInternalServerError)
		return
	}

	dailyMetrics, err := h.collectDailyMetrics(ctx, repoIDs, startDate, endDate)
	if err != nil {
		h.logger.Error("failed to collect daily metrics", "error", err)
		http.Error(w, "failed to get metrics", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, dailyMetrics)
}

// PullRequests returns a list of pull requests for given repositories.
func (h *MetricsHandler) PullRequests(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	startDate, endDate := parseDateRange(r)

	repoIDs, err := h.getRepositoryIDs(r)
	if err != nil {
		h.logger.Error("failed to get repository IDs", "error", err)
		http.Error(w, "failed to get repository IDs", http.StatusInternalServerError)
		return
	}

	// Build repository name map
	repos, _ := h.ds.ListRepositories(ctx)
	repoNameMap := make(map[string]string, len(repos))
	for _, repo := range repos {
		repoNameMap[repo.ID] = repo.FullName
	}

	prs, err := h.collectPullRequests(ctx, repoIDs, startDate, endDate)
	if err != nil {
		h.logger.Error("failed to collect pull requests", "error", err)
		http.Error(w, "failed to get pull requests", http.StatusInternalServerError)
		return
	}

	result := make([]MemberPullRequest, 0, len(prs))
	for _, pr := range prs {
		result = append(result, MemberPullRequest{
			Number:     pr.Number,
			Title:      pr.Title,
			Author:     pr.Author,
			State:      pr.State,
			CreatedAt:  pr.CreatedAt,
			MergedAt:   pr.MergedAt,
			Additions:  pr.Additions,
			Deletions:  pr.Deletions,
			CycleTime:  pr.CycleTimeHours(),
			CodingTime: pr.CodingTimeHours(),
			PickupTime: pr.PickupTimeHours(),
			ReviewTime: pr.ReviewTimeHours(),
			MergeTime:  pr.MergeTimeHours(),
			RepoName:   repoNameMap[pr.RepositoryID],
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	respondJSON(w, http.StatusOK, result)
}
