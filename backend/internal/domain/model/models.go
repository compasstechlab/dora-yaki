package model

import "time"

// Repository represents a GitHub repository
type Repository struct {
	ID             string     `json:"id" datastore:"id"`
	Owner          string     `json:"owner" datastore:"owner"`
	Name           string     `json:"name" datastore:"name"`
	FullName       string     `json:"fullName" datastore:"full_name"`
	Private        bool       `json:"private" datastore:"private"`
	CreatedAt      time.Time  `json:"createdAt" datastore:"created_at"`
	UpdatedAt      time.Time  `json:"updatedAt" datastore:"updated_at"`
	LastSyncedAt   *time.Time `json:"lastSyncedAt,omitempty" datastore:"last_synced_at"`
	ProcessStartAt *time.Time `json:"processStartAt,omitempty" datastore:"process_start_at"`
}

// FileExtStats holds change statistics per file extension.
type FileExtStats struct {
	Extension string `json:"extension" datastore:"extension"`
	Additions int    `json:"additions" datastore:"additions"`
	Deletions int    `json:"deletions" datastore:"deletions"`
	Files     int    `json:"files" datastore:"files"`
}

// PullRequest represents a GitHub pull request
type PullRequest struct {
	ID            string         `json:"id" datastore:"id"`
	RepositoryID  string         `json:"repositoryId" datastore:"repository_id"`
	Number        int            `json:"number" datastore:"number"`
	Title         string         `json:"title" datastore:"title,noindex"`
	Author        string         `json:"author" datastore:"author"`
	State         string         `json:"state" datastore:"state"`
	Draft         bool           `json:"draft" datastore:"draft"`
	CreatedAt     time.Time      `json:"createdAt" datastore:"created_at"`
	UpdatedAt     time.Time      `json:"updatedAt" datastore:"updated_at"`
	MergedAt      *time.Time     `json:"mergedAt,omitempty" datastore:"merged_at"`
	ClosedAt      *time.Time     `json:"closedAt,omitempty" datastore:"closed_at"`
	FirstCommitAt *time.Time     `json:"firstCommitAt,omitempty" datastore:"first_commit_at"`
	FirstReviewAt *time.Time     `json:"firstReviewAt,omitempty" datastore:"first_review_at"`
	ApprovedAt    *time.Time     `json:"approvedAt,omitempty" datastore:"approved_at"`
	Additions     int            `json:"additions" datastore:"additions"`
	Deletions     int            `json:"deletions" datastore:"deletions"`
	ChangedFiles  int            `json:"changedFiles" datastore:"changed_files"`
	CommitCount   int            `json:"commitCount" datastore:"commit_count"`
	FileExtStats  []FileExtStats `json:"fileExtStats,omitempty" datastore:"file_ext_stats,flatten"`
}

// CycleTimeHours returns the total cycle time of the PR in hours.
// PRの全体サイクルタイム（時間単位）を返す
func (pr *PullRequest) CycleTimeHours() float64 {
	if pr.MergedAt == nil {
		return 0
	}
	start := pr.CreatedAt
	if pr.FirstCommitAt != nil && pr.FirstCommitAt.Before(pr.CreatedAt) {
		start = *pr.FirstCommitAt
	}
	return pr.MergedAt.Sub(start).Hours()
}

// CodingTimeHours returns the coding time (first commit to PR creation) in hours.
// コーディング時間（時間単位）を返す
func (pr *PullRequest) CodingTimeHours() float64 {
	if pr.FirstCommitAt == nil {
		return 0
	}
	return pr.CreatedAt.Sub(*pr.FirstCommitAt).Hours()
}

// PickupTimeHours returns the time until first review in hours.
// レビュー開始までの待ち時間（時間単位）を返す
func (pr *PullRequest) PickupTimeHours() float64 {
	if pr.FirstReviewAt == nil {
		return 0
	}
	return pr.FirstReviewAt.Sub(pr.CreatedAt).Hours()
}

// ReviewTimeHours returns the review time (first review to approval) in hours.
// レビュー時間（時間単位）を返す
func (pr *PullRequest) ReviewTimeHours() float64 {
	if pr.FirstReviewAt == nil || pr.ApprovedAt == nil {
		return 0
	}
	return pr.ApprovedAt.Sub(*pr.FirstReviewAt).Hours()
}

// MergeTimeHours returns the time from approval to merge in hours.
// 承認からマージまでの時間（時間単位）を返す
func (pr *PullRequest) MergeTimeHours() float64 {
	if pr.ApprovedAt == nil || pr.MergedAt == nil {
		return 0
	}
	return pr.MergedAt.Sub(*pr.ApprovedAt).Hours()
}

// Review represents a GitHub pull request review
type Review struct {
	ID            string    `json:"id" datastore:"id"`
	PullRequestID string    `json:"pullRequestId" datastore:"pull_request_id"`
	RepositoryID  string    `json:"repositoryId" datastore:"repository_id"`
	Reviewer      string    `json:"reviewer" datastore:"reviewer"`
	State         string    `json:"state" datastore:"state"` // APPROVED, CHANGES_REQUESTED, COMMENTED, DISMISSED
	Body          string    `json:"body" datastore:"body,noindex"`
	SubmittedAt   time.Time `json:"submittedAt" datastore:"submitted_at"`
	CommentsCount int       `json:"commentsCount" datastore:"comments_count"`
}

// Deployment represents a deployment/release event
type Deployment struct {
	ID           string    `json:"id" datastore:"id"`
	RepositoryID string    `json:"repositoryId" datastore:"repository_id"`
	Environment  string    `json:"environment" datastore:"environment"`
	Ref          string    `json:"ref" datastore:"ref"`
	SHA          string    `json:"sha" datastore:"sha"`
	Status       string    `json:"status" datastore:"status"` // success, failure, pending
	CreatedAt    time.Time `json:"createdAt" datastore:"created_at"`
	DeployedAt   time.Time `json:"deployedAt" datastore:"deployed_at"`
}

// DailyMetrics represents aggregated metrics for a repository on a specific date
type DailyMetrics struct {
	ID           string    `json:"id" datastore:"id"` // repository_id:date
	RepositoryID string    `json:"repositoryId" datastore:"repository_id"`
	Date         time.Time `json:"date" datastore:"date"`

	// Cycle Time metrics (in hours)
	AvgCycleTime  float64 `json:"avgCycleTime" datastore:"avg_cycle_time"`
	AvgCodingTime float64 `json:"avgCodingTime" datastore:"avg_coding_time"`
	AvgPickupTime float64 `json:"avgPickupTime" datastore:"avg_pickup_time"`
	AvgReviewTime float64 `json:"avgReviewTime" datastore:"avg_review_time"`
	AvgMergeTime  float64 `json:"avgMergeTime" datastore:"avg_merge_time"`

	// PR Metrics
	PRsOpened int `json:"prsOpened" datastore:"prs_opened"`
	PRsMerged int `json:"prsMerged" datastore:"prs_merged"`
	PRsClosed int `json:"prsClosed" datastore:"prs_closed"`

	// Review Metrics
	ReviewsSubmitted int     `json:"reviewsSubmitted" datastore:"reviews_submitted"`
	AvgReviewsPerPR  float64 `json:"avgReviewsPerPR" datastore:"avg_reviews_per_pr"`

	// Code Metrics
	TotalAdditions int `json:"totalAdditions" datastore:"total_additions"`
	TotalDeletions int `json:"totalDeletions" datastore:"total_deletions"`

	// DORA Metrics
	DeploymentCount   int     `json:"deploymentCount" datastore:"deployment_count"`
	ChangeFailureRate float64 `json:"changeFailureRate" datastore:"change_failure_rate"`

	// Contributors
	ActiveContributors int `json:"activeContributors" datastore:"active_contributors"`
}

// TeamMember represents a team member
type TeamMember struct {
	ID        string    `json:"id" datastore:"id"`
	Login     string    `json:"login" datastore:"login"`
	Name      string    `json:"name" datastore:"name"`
	AvatarURL string    `json:"avatarUrl" datastore:"avatar_url"`
	CreatedAt time.Time `json:"createdAt" datastore:"created_at"`
}

// BotUser represents a custom registered bot user.
type BotUser struct {
	Username  string    `json:"username" datastore:"username"`
	CreatedAt time.Time `json:"createdAt" datastore:"created_at"`
}

// Sprint represents a development sprint
type Sprint struct {
	ID           string    `json:"id" datastore:"id"`
	RepositoryID string    `json:"repositoryId" datastore:"repository_id"`
	Name         string    `json:"name" datastore:"name"`
	StartDate    time.Time `json:"startDate" datastore:"start_date"`
	EndDate      time.Time `json:"endDate" datastore:"end_date"`
	Goals        string    `json:"goals" datastore:"goals,noindex"`
}

// SyncLock represents an exclusive lock for batch jobs.
// ジョブの排他ロックを表す。
type SyncLock struct {
	ID        string    `json:"id" datastore:"id"`
	LockedBy  string    `json:"lockedBy" datastore:"locked_by"`
	LockedAt  time.Time `json:"lockedAt" datastore:"locked_at"`
	ExpiresAt time.Time `json:"expiresAt" datastore:"expires_at"`
}

// SprintMetrics represents metrics for a sprint
type SprintMetrics struct {
	SprintID         string  `json:"sprintId" datastore:"sprint_id"`
	PlannedPRs       int     `json:"plannedPRs" datastore:"planned_prs"`
	CompletedPRs     int     `json:"completedPRs" datastore:"completed_prs"`
	CompletionRate   float64 `json:"completionRate" datastore:"completion_rate"`
	AvgCycleTime     float64 `json:"avgCycleTime" datastore:"avg_cycle_time"`
	TotalAdditions   int     `json:"totalAdditions" datastore:"total_additions"`
	TotalDeletions   int     `json:"totalDeletions" datastore:"total_deletions"`
	ReviewsSubmitted int     `json:"reviewsSubmitted" datastore:"reviews_submitted"`
}
