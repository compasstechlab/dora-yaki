package github

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/go-github/v82/github"

	"github.com/compasstechlab/dora-yaki/internal/domain/model"
	"github.com/compasstechlab/dora-yaki/internal/timeutil"
)

// Collector handles collecting metrics data from GitHub
type Collector struct {
	client *Client
	logger *slog.Logger
}

// NewCollector creates a new Collector
func NewCollector(client *Client, logger *slog.Logger) *Collector {
	return &Collector{
		client: client,
		logger: logger,
	}
}

// CollectOptions options for data collection
type CollectOptions struct {
	Since    time.Time
	Until    time.Time
	State    string // all, open, closed
	PerPage  int
	MaxPages int
}

// DefaultCollectOptions returns default collection options
func DefaultCollectOptions() *CollectOptions {
	return &CollectOptions{
		Since:    timeutil.Now().AddDate(0, -3, 0), // 3 months ago
		Until:    timeutil.Now(),
		State:    "all",
		PerPage:  100,
		MaxPages: 10,
	}
}

// CollectOptionsForRange returns CollectOptions based on the specified sync range.
func CollectOptionsForRange(syncRange string) *CollectOptions {
	now := timeutil.Now()
	opts := &CollectOptions{
		Until:   now,
		State:   "all",
		PerPage: 100,
	}

	switch syncRange {
	case "day":
		opts.Since = now.AddDate(0, 0, -1)
		opts.MaxPages = 3
	case "week":
		opts.Since = now.AddDate(0, 0, -7)
		opts.MaxPages = 5
	case "month":
		opts.Since = now.AddDate(0, -1, 0)
		opts.MaxPages = 10
	default: // "full"
		opts.Since = now.AddDate(0, -3, 0)
		opts.MaxPages = 10
	}

	return opts
}

// CollectedData holds all collected data for a repository
type CollectedData struct {
	Repository   *model.Repository
	PullRequests []*model.PullRequest
	Reviews      []*model.Review
	Deployments  []*model.Deployment
	TeamMembers  []*model.TeamMember
}

// CollectAll collects all data for a repository
func (c *Collector) CollectAll(ctx context.Context, owner, repo string, opts *CollectOptions) (*CollectedData, error) {
	if opts == nil {
		opts = DefaultCollectOptions()
	}

	c.logger.Info("starting data collection",
		"owner", owner,
		"repo", repo,
		"since", opts.Since,
		"until", opts.Until,
	)

	data := &CollectedData{}

	// Collect repository info
	repoInfo, err := c.client.GetRepository(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to collect repository: %w", err)
	}
	data.Repository = repoInfo
	repoID := repoInfo.ID // Use numeric ID for subsequent collection
	c.logger.Info("repository info collected", "repoID", repoID, "fullName", repoInfo.FullName)

	// Collect pull requests
	prs, err := c.CollectPullRequests(ctx, owner, repo, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to collect pull requests: %w", err)
	}
	data.PullRequests = prs

	// Collect reviews for each PR
	reviews, err := c.CollectReviews(ctx, owner, repo, prs, repoID)
	if err != nil {
		c.logger.Warn("failed to collect some reviews", "error", err)
	}
	data.Reviews = reviews

	// Collect deployments
	deployments, err := c.CollectDeployments(ctx, owner, repo, opts, repoID)
	if err != nil {
		c.logger.Warn("failed to collect deployments", "error", err)
	}
	data.Deployments = deployments

	// Collect team members
	c.logger.Info("collecting contributors", "owner", owner, "repo", repo)
	members, err := c.client.ListContributors(ctx, owner, repo)
	if err != nil {
		c.logger.Warn("failed to collect contributors", "error", err)
	}
	data.TeamMembers = members

	c.logger.Info("data collection completed",
		"prs", len(data.PullRequests),
		"reviews", len(data.Reviews),
		"deployments", len(data.Deployments),
		"members", len(data.TeamMembers),
	)

	return data, nil
}

// CollectPullRequests collects pull requests from GitHub
func (c *Collector) CollectPullRequests(ctx context.Context, owner, repo string, opts *CollectOptions) ([]*model.PullRequest, error) {
	c.logger.Info("collecting pull requests",
		"owner", owner, "repo", repo,
		"state", opts.State, "maxPages", opts.MaxPages,
	)

	var allPRs []*model.PullRequest
	const progressInterval = 20

	for page := 1; page <= opts.MaxPages; page++ {
		listOpts := &PullRequestListOptions{
			State:     opts.State,
			Sort:      "updated",
			Direction: "desc",
			Page:      page,
			PerPage:   opts.PerPage,
		}

		prs, err := c.client.ListPullRequests(ctx, owner, repo, listOpts)
		if err != nil {
			return nil, err
		}

		if len(prs) == 0 {
			break
		}

		c.logger.Info("fetched pull requests page",
			"page", page, "count", len(prs), "totalSoFar", len(allPRs),
		)

		// Filter by date range and enrich with additional data
		for _, pr := range prs {
			if pr.UpdatedAt.Before(opts.Since) {
				c.logger.Info("reached date boundary, stopping PR collection",
					"total", len(allPRs), "boundaryPR", pr.Number,
				)
				return allPRs, nil
			}

			// Fetch PR details to supplement stats (not available from List API)
			prDetail, err := c.client.GetPullRequest(ctx, owner, repo, pr.Number)
			if err != nil {
				c.logger.Warn("failed to get pull request details",
					"pr", pr.Number,
					"error", err,
				)
			} else {
				pr.Additions = prDetail.Additions
				pr.Deletions = prDetail.Deletions
				pr.ChangedFiles = prDetail.ChangedFiles
				pr.CommitCount = prDetail.CommitCount
			}

			// Fetch file stats by extension
			files, err := c.client.ListPullRequestFiles(ctx, owner, repo, pr.Number)
			if err != nil {
				c.logger.Warn("failed to list pull request files",
					"pr", pr.Number,
					"error", err,
				)
			} else {
				pr.FileExtStats = aggregateFileExtStats(files)
			}

			// Enrich PR with first commit time
			firstCommitTime, err := c.client.GetFirstCommitTime(ctx, owner, repo, pr.Number)
			if err != nil {
				c.logger.Warn("failed to get first commit time",
					"pr", pr.Number,
					"error", err,
				)
			} else {
				pr.FirstCommitAt = firstCommitTime
			}

			allPRs = append(allPRs, pr)

			// Progress log
			if len(allPRs)%progressInterval == 0 {
				c.logger.Info("PR collection progress",
					"collected", len(allPRs),
					"latestPR", pr.Number,
					"author", pr.Author,
				)
			}
		}

		if len(prs) < opts.PerPage {
			break
		}
	}

	c.logger.Info("pull request collection finished", "total", len(allPRs))
	return allPRs, nil
}

// CollectReviews collects reviews for pull requests
func (c *Collector) CollectReviews(ctx context.Context, owner, repo string, prs []*model.PullRequest, repositoryID string) ([]*model.Review, error) {
	c.logger.Info("collecting reviews", "targetPRs", len(prs))

	var allReviews []*model.Review
	const progressInterval = 20

	for i, pr := range prs {
		reviews, err := c.client.ListPullRequestReviews(ctx, owner, repo, pr.Number, repositoryID)
		if err != nil {
			c.logger.Warn("failed to collect reviews for PR",
				"pr", pr.Number,
				"error", err,
			)
			continue
		}

		// Enrich PR with first review time
		for _, review := range reviews {
			if pr.FirstReviewAt == nil || review.SubmittedAt.Before(*pr.FirstReviewAt) {
				pr.FirstReviewAt = &review.SubmittedAt
			}

			// Track first approval time
			if review.State == "APPROVED" {
				if pr.ApprovedAt == nil || review.SubmittedAt.Before(*pr.ApprovedAt) {
					pr.ApprovedAt = &review.SubmittedAt
				}
			}
		}

		// Get comment counts for reviews
		comments, err := c.client.ListReviewComments(ctx, owner, repo, pr.Number)
		if err != nil {
			c.logger.Warn("failed to collect review comments",
				"pr", pr.Number,
				"error", err,
			)
		} else {
			// Count comments per reviewer
			commentCounts := make(map[string]int)
			for _, comment := range comments {
				commentCounts[comment.GetUser().GetLogin()]++
			}

			for j, review := range reviews {
				reviews[j].CommentsCount = commentCounts[review.Reviewer]
			}
		}

		allReviews = append(allReviews, reviews...)

		// Progress log
		processed := i + 1
		if processed%progressInterval == 0 || processed == len(prs) {
			c.logger.Info("review collection progress",
				"processedPRs", processed,
				"totalPRs", len(prs),
				"reviewsCollected", len(allReviews),
			)
		}
	}

	c.logger.Info("review collection finished",
		"totalReviews", len(allReviews), "processedPRs", len(prs),
	)
	return allReviews, nil
}

// CollectDeployments collects deployment data
func (c *Collector) CollectDeployments(ctx context.Context, owner, repo string, opts *CollectOptions, repositoryID string) ([]*model.Deployment, error) {
	c.logger.Info("collecting deployments", "owner", owner, "repo", repo)

	var allDeployments []*model.Deployment

	for page := 1; page <= opts.MaxPages; page++ {
		listOpts := &DeploymentListOptions{
			Page:    page,
			PerPage: opts.PerPage,
		}

		deployments, err := c.client.ListDeployments(ctx, owner, repo, listOpts, repositoryID)
		if err != nil {
			return nil, err
		}

		if len(deployments) == 0 {
			break
		}

		c.logger.Info("fetched deployments page",
			"page", page, "count", len(deployments), "totalSoFar", len(allDeployments),
		)

		for _, d := range deployments {
			if d.CreatedAt.Before(opts.Since) {
				c.logger.Info("reached date boundary, stopping deployment collection",
					"total", len(allDeployments),
				)
				return allDeployments, nil
			}
			allDeployments = append(allDeployments, d)
		}

		if len(deployments) < opts.PerPage {
			break
		}
	}

	c.logger.Info("deployment collection finished", "total", len(allDeployments))
	return allDeployments, nil
}

// SyncRepository syncs data for a specific repository
func (c *Collector) SyncRepository(ctx context.Context, owner, repo string, lastSyncTime *time.Time) (*CollectedData, error) {
	opts := DefaultCollectOptions()

	if lastSyncTime != nil {
		opts.Since = *lastSyncTime
	}

	return c.CollectAll(ctx, owner, repo, opts)
}

// aggregateFileExtStats aggregates change stats by file extension.
func aggregateFileExtStats(files []*github.CommitFile) []model.FileExtStats {
	statsMap := make(map[string]*model.FileExtStats)

	for _, f := range files {
		filename := f.GetFilename()
		ext := strings.ToLower(filepath.Ext(filename))
		if ext == "" {
			ext = "(no ext)"
		}

		s, ok := statsMap[ext]
		if !ok {
			s = &model.FileExtStats{Extension: ext}
			statsMap[ext] = s
		}
		s.Additions += f.GetAdditions()
		s.Deletions += f.GetDeletions()
		s.Files++
	}

	result := make([]model.FileExtStats, 0, len(statsMap))
	for _, s := range statsMap {
		result = append(result, *s)
	}

	// Sort by number of changed lines descending
	sort.Slice(result, func(i, j int) bool {
		return (result[i].Additions + result[i].Deletions) > (result[j].Additions + result[j].Deletions)
	})

	return result
}
