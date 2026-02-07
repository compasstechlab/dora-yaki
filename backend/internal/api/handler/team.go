package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/compasstechlab/dora-yaki/internal/datastore"
	"github.com/compasstechlab/dora-yaki/internal/domain/model"
)

// TeamHandler handles team-related API requests
type TeamHandler struct {
	ds     *datastore.Client
	logger *slog.Logger
}

// NewTeamHandler creates a new TeamHandler
func NewTeamHandler(ds *datastore.Client, logger *slog.Logger) *TeamHandler {
	return &TeamHandler{
		ds:     ds,
		logger: logger,
	}
}

// getBotUsernames retrieves custom bot username list from Datastore.
func (h *TeamHandler) getBotUsernames(ctx context.Context) []string {
	usernames, err := h.ds.ListBotUsernames(ctx)
	if err != nil {
		h.logger.Warn("failed to get bot usernames", "error", err)
		return nil
	}
	return usernames
}

// MemberStats represents statistics for a team member
type MemberStats struct {
	Member                  *model.TeamMember            `json:"member"`
	PRsAuthored             int                          `json:"prsAuthored"`
	PRsMerged               int                          `json:"prsMerged"`
	ReviewsGiven            int                          `json:"reviewsGiven"`
	CommentsGiven           int                          `json:"commentsGiven"`
	AvgCycleTime            float64                      `json:"avgCycleTime"`
	AvgCodingTime           float64                      `json:"avgCodingTime"`
	AvgPickupTime           float64                      `json:"avgPickupTime"`
	AvgReviewTime           float64                      `json:"avgReviewTime"`
	AvgMergeTime            float64                      `json:"avgMergeTime"`
	ReviewsApproved         int                          `json:"reviewsApproved"`
	ReviewsChangesRequested int                          `json:"reviewsChangesRequested"`
	ReviewsCommented        int                          `json:"reviewsCommented"`
	ApprovalRate            float64                      `json:"approvalRate"`
	TotalAdditions          int                          `json:"totalAdditions"`
	TotalDeletions          int                          `json:"totalDeletions"`
	ByFileExtension         []model.FileExtensionMetrics `json:"byFileExtension,omitempty"`
}

// MemberReview is the response type for member review information.
type MemberReview struct {
	SubmittedAt time.Time `json:"submittedAt"`
	State       string    `json:"state"`
	RepoName    string    `json:"repoName"`
}

// MemberPullRequest is the response type for member pull request information.
type MemberPullRequest struct {
	Number     int        `json:"number"`
	Title      string     `json:"title"`
	Author     string     `json:"author,omitempty"`
	State      string     `json:"state"`
	CreatedAt  time.Time  `json:"createdAt"`
	MergedAt   *time.Time `json:"mergedAt,omitempty"`
	Additions  int        `json:"additions"`
	Deletions  int        `json:"deletions"`
	CycleTime  float64    `json:"cycleTime"`
	CodingTime float64    `json:"codingTime"`
	PickupTime float64    `json:"pickupTime"`
	ReviewTime float64    `json:"reviewTime"`
	MergeTime  float64    `json:"mergeTime"`
	RepoName   string     `json:"repoName"`
}

// ListMembers lists all team members
func (h *TeamHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	bf := parseBotFilter(r)

	members, err := h.ds.ListTeamMembers(ctx)
	if err != nil {
		h.logger.Error("failed to list team members", "error", err)
		http.Error(w, "failed to list team members", http.StatusInternalServerError)
		return
	}

	// Apply bot filtering
	botUsernames := h.getBotUsernames(ctx)
	members = model.FilterTeamMembersByBot(members, botUsernames, bf.excludeBots, bf.botsOnly)

	respondJSON(w, http.StatusOK, members)
}

// getRepositoryIDs retrieves multiple repository IDs. Returns all repositories if empty.
func (h *TeamHandler) getRepositoryIDs(r *http.Request) ([]string, error) {
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

// collectPullRequests collects PRs from multiple repositories.
func (h *TeamHandler) collectPullRequests(ctx context.Context, repoIDs []string, start, end time.Time) []*model.PullRequest {
	var result []*model.PullRequest
	for _, id := range repoIDs {
		prs, err := h.ds.ListPullRequestsByDateRange(ctx, id, start, end)
		if err != nil {
			h.logger.Warn("failed to list pull requests for repo", "repository", id, "error", err)
			continue
		}
		result = append(result, prs...)
	}
	return result
}

// collectReviews collects reviews from multiple repositories.
func (h *TeamHandler) collectReviews(ctx context.Context, repoIDs []string, start, end time.Time) []*model.Review {
	var result []*model.Review
	for _, id := range repoIDs {
		reviews, err := h.ds.ListReviewsByDateRange(ctx, id, start, end)
		if err != nil {
			h.logger.Warn("failed to list reviews for repo", "repository", id, "error", err)
			continue
		}
		result = append(result, reviews...)
	}
	return result
}

// GetMemberStats returns statistics for a specific team member
func (h *TeamHandler) GetMemberStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	memberID := getMemberID(r)
	startDate, endDate := parseDateRange(r)

	// Get multiple repository IDs
	repoIDs, err := h.getRepositoryIDs(r)
	if err != nil {
		h.logger.Error("failed to get repository IDs", "error", err)
		http.Error(w, "failed to get repository IDs", http.StatusInternalServerError)
		return
	}

	// Get member information
	members, err := h.ds.ListTeamMembers(ctx)
	if err != nil {
		h.logger.Error("failed to get team members", "error", err)
		http.Error(w, "failed to get member stats", http.StatusInternalServerError)
		return
	}

	var member *model.TeamMember
	for _, m := range members {
		if m.ID == memberID || m.Login == memberID {
			member = m
			break
		}
	}

	if member == nil {
		http.Error(w, "member not found", http.StatusNotFound)
		return
	}

	// Collect PRs and reviews from multiple repositories
	prs := h.collectPullRequests(ctx, repoIDs, startDate, endDate)
	reviews := h.collectReviews(ctx, repoIDs, startDate, endDate)

	stats := calculateMemberStats(member, prs, reviews)
	respondJSON(w, http.StatusOK, stats)
}

func getMemberID(r *http.Request) string {
	path := r.URL.Path
	parts := strings.Split(path, "/")

	// /api/team/members/{id}/stats or /api/team/members/{id}/pull-requests or /api/team/members/{id}/reviews
	if len(parts) >= 5 {
		suffix := parts[len(parts)-1]
		if suffix == "stats" || suffix == "pull-requests" || suffix == "reviews" {
			return parts[len(parts)-2]
		}
		return parts[len(parts)-1]
	}

	return ""
}

func calculateMemberStats(member *model.TeamMember, prs []*model.PullRequest, reviews []*model.Review) *MemberStats {
	stats := &MemberStats{
		Member: member,
	}

	// For cycle time breakdown aggregation
	var codingTimes, pickupTimes, reviewTimes, mergeTimes []float64
	var cycleTimesSum float64
	var cycleTimeCount int

	// For file extension aggregation
	type extAgg struct {
		additions int
		deletions int
		files     int
		prCount   int
	}
	extMap := make(map[string]*extAgg)

	for _, pr := range prs {
		if pr.Author == member.Login {
			stats.PRsAuthored++
			stats.TotalAdditions += pr.Additions
			stats.TotalDeletions += pr.Deletions

			if pr.MergedAt != nil {
				stats.PRsMerged++

				// Calculate cycle time
				if pr.FirstCommitAt != nil {
					cycleTime := pr.MergedAt.Sub(*pr.FirstCommitAt).Hours()
					cycleTimesSum += cycleTime
					cycleTimeCount++
				}

				// Cycle time breakdown
				if ct := pr.CodingTimeHours(); ct > 0 {
					codingTimes = append(codingTimes, ct)
				}
				if pt := pr.PickupTimeHours(); pt > 0 {
					pickupTimes = append(pickupTimes, pt)
				}
				if rt := pr.ReviewTimeHours(); rt > 0 {
					reviewTimes = append(reviewTimes, rt)
				}
				if mt := pr.MergeTimeHours(); mt > 0 {
					mergeTimes = append(mergeTimes, mt)
				}
			}

			// Aggregate file extension statistics
			seen := make(map[string]bool)
			for _, fs := range pr.FileExtStats {
				a, ok := extMap[fs.Extension]
				if !ok {
					a = &extAgg{}
					extMap[fs.Extension] = a
				}
				a.additions += fs.Additions
				a.deletions += fs.Deletions
				a.files += fs.Files
				if !seen[fs.Extension] {
					a.prCount++
					seen[fs.Extension] = true
				}
			}
		}
	}

	if cycleTimeCount > 0 {
		stats.AvgCycleTime = cycleTimesSum / float64(cycleTimeCount)
	}
	stats.AvgCodingTime = avgFloat(codingTimes)
	stats.AvgPickupTime = avgFloat(pickupTimes)
	stats.AvgReviewTime = avgFloat(reviewTimes)
	stats.AvgMergeTime = avgFloat(mergeTimes)

	// Convert file extension stats to slice
	if len(extMap) > 0 {
		stats.ByFileExtension = make([]model.FileExtensionMetrics, 0, len(extMap))
		for ext, a := range extMap {
			stats.ByFileExtension = append(stats.ByFileExtension, model.FileExtensionMetrics{
				Extension: ext,
				Additions: a.additions,
				Deletions: a.deletions,
				Files:     a.files,
				PRCount:   a.prCount,
			})
		}
		sort.Slice(stats.ByFileExtension, func(i, j int) bool {
			return (stats.ByFileExtension[i].Additions + stats.ByFileExtension[i].Deletions) >
				(stats.ByFileExtension[j].Additions + stats.ByFileExtension[j].Deletions)
		})
	}

	// Review statistics (including state-based counts)
	for _, review := range reviews {
		if review.Reviewer == member.Login {
			stats.ReviewsGiven++
			stats.CommentsGiven += review.CommentsCount
			switch review.State {
			case "APPROVED":
				stats.ReviewsApproved++
			case "CHANGES_REQUESTED":
				stats.ReviewsChangesRequested++
			case "COMMENTED":
				stats.ReviewsCommented++
			}
		}
	}

	// Approval rate
	if stats.ReviewsGiven > 0 {
		stats.ApprovalRate = float64(stats.ReviewsApproved) / float64(stats.ReviewsGiven) * 100
	}

	return stats
}

// GetMemberPullRequests returns a list of pull requests for a member.
func (h *TeamHandler) GetMemberPullRequests(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	memberID := getMemberID(r)
	startDate, endDate := parseDateRange(r)

	repoIDs, err := h.getRepositoryIDs(r)
	if err != nil {
		h.logger.Error("failed to get repository IDs", "error", err)
		http.Error(w, "failed to get repository IDs", http.StatusInternalServerError)
		return
	}

	// Get member information
	members, err := h.ds.ListTeamMembers(ctx)
	if err != nil {
		h.logger.Error("failed to get team members", "error", err)
		http.Error(w, "failed to get member", http.StatusInternalServerError)
		return
	}

	var member *model.TeamMember
	for _, m := range members {
		if m.ID == memberID || m.Login == memberID {
			member = m
			break
		}
	}
	if member == nil {
		http.Error(w, "member not found", http.StatusNotFound)
		return
	}

	// Build repository name map
	repos, _ := h.ds.ListRepositories(ctx)
	repoNameMap := make(map[string]string, len(repos))
	for _, repo := range repos {
		repoNameMap[repo.ID] = repo.FullName
	}

	// Collect and filter PRs
	prs := h.collectPullRequests(ctx, repoIDs, startDate, endDate)
	result := make([]MemberPullRequest, 0, len(prs))
	for _, pr := range prs {
		if pr.Author != member.Login {
			continue
		}
		result = append(result, MemberPullRequest{
			Number:     pr.Number,
			Title:      pr.Title,
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

	// Sort by creation date descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	respondJSON(w, http.StatusOK, result)
}

// GetMemberReviews returns a list of reviews for a member.
func (h *TeamHandler) GetMemberReviews(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	memberID := getMemberID(r)
	startDate, endDate := parseDateRange(r)

	repoIDs, err := h.getRepositoryIDs(r)
	if err != nil {
		h.logger.Error("failed to get repository IDs", "error", err)
		http.Error(w, "failed to get repository IDs", http.StatusInternalServerError)
		return
	}

	// Get member information
	members, err := h.ds.ListTeamMembers(ctx)
	if err != nil {
		h.logger.Error("failed to get team members", "error", err)
		http.Error(w, "failed to get member", http.StatusInternalServerError)
		return
	}

	var member *model.TeamMember
	for _, m := range members {
		if m.ID == memberID || m.Login == memberID {
			member = m
			break
		}
	}
	if member == nil {
		http.Error(w, "member not found", http.StatusNotFound)
		return
	}

	// Build repository name map
	repos, _ := h.ds.ListRepositories(ctx)
	repoNameMap := make(map[string]string, len(repos))
	for _, repo := range repos {
		repoNameMap[repo.ID] = repo.FullName
	}

	// Collect and filter reviews
	reviews := h.collectReviews(ctx, repoIDs, startDate, endDate)
	result := make([]MemberReview, 0, len(reviews))
	for _, review := range reviews {
		if review.Reviewer != member.Login {
			continue
		}
		result = append(result, MemberReview{
			SubmittedAt: review.SubmittedAt,
			State:       review.State,
			RepoName:    repoNameMap[review.RepositoryID],
		})
	}

	// Sort by submission date descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].SubmittedAt.After(result[j].SubmittedAt)
	})

	respondJSON(w, http.StatusOK, result)
}

func avgFloat(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	var sum float64
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}
