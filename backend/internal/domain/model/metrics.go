package model

import "time"

// FileExtensionMetrics holds aggregated change statistics per file extension.
type FileExtensionMetrics struct {
	Extension string `json:"extension"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Files     int    `json:"files"`
	PRCount   int    `json:"prCount"`
}

// CycleTimeMetrics represents cycle time analysis data
type CycleTimeMetrics struct {
	Period          string                 `json:"period"`
	StartDate       time.Time              `json:"startDate"`
	EndDate         time.Time              `json:"endDate"`
	TotalPRs        int                    `json:"totalPRs"`
	AvgCycleTime    float64                `json:"avgCycleTime"`    // hours
	AvgCodingTime   float64                `json:"avgCodingTime"`   // hours
	AvgPickupTime   float64                `json:"avgPickupTime"`   // hours
	AvgReviewTime   float64                `json:"avgReviewTime"`   // hours
	AvgMergeTime    float64                `json:"avgMergeTime"`    // hours
	MedianCycleTime float64                `json:"medianCycleTime"` // hours
	P90CycleTime    float64                `json:"p90CycleTime"`    // hours
	DailyBreakdown  []DailyMetrics         `json:"dailyBreakdown,omitempty"`
	ByAuthor        []AuthorMetrics        `json:"byAuthor,omitempty"`
	ByFileExtension []FileExtensionMetrics `json:"byFileExtension,omitempty"`
}

// AuthorMetrics represents metrics for a specific author
type AuthorMetrics struct {
	Author       string  `json:"author"`
	PRCount      int     `json:"prCount"`
	AvgCycleTime float64 `json:"avgCycleTime"`
	Additions    int     `json:"additions"`
	Deletions    int     `json:"deletions"`
}

// ReviewMetrics represents review analysis data
type ReviewMetrics struct {
	Period               string          `json:"period"`
	StartDate            time.Time       `json:"startDate"`
	EndDate              time.Time       `json:"endDate"`
	TotalReviews         int             `json:"totalReviews"`
	TotalComments        int             `json:"totalComments"`
	AvgReviewsPerPR      float64         `json:"avgReviewsPerPR"`
	AvgCommentsPerReview float64         `json:"avgCommentsPerReview"`
	AvgTimeToFirstReview float64         `json:"avgTimeToFirstReview"` // hours
	ApprovalRate         float64         `json:"approvalRate"`         // percentage
	ChangesRequestedRate float64         `json:"changesRequestedRate"` // percentage
	ByReviewer           []ReviewerStats `json:"byReviewer,omitempty"`
}

// ReviewerStats represents statistics for a specific reviewer
type ReviewerStats struct {
	Reviewer        string  `json:"reviewer"`
	ReviewCount     int     `json:"reviewCount"`
	CommentCount    int     `json:"commentCount"`
	AvgResponseTime float64 `json:"avgResponseTime"` // hours
	ApprovalRate    float64 `json:"approvalRate"`
}

// DORAMetrics represents DORA (DevOps Research and Assessment) metrics
type DORAMetrics struct {
	Period    string    `json:"period"`
	StartDate time.Time `json:"startDate"`
	EndDate   time.Time `json:"endDate"`

	// Deployment Frequency
	DeploymentCount     int     `json:"deploymentCount"`
	DeploymentFrequency string  `json:"deploymentFrequency"` // daily, weekly, monthly, yearly
	AvgDeploysPerDay    float64 `json:"avgDeploysPerDay"`

	// Lead Time for Changes
	AvgLeadTime    float64 `json:"avgLeadTime"` // hours
	MedianLeadTime float64 `json:"medianLeadTime"`
	P90LeadTime    float64 `json:"p90LeadTime"`

	// Change Failure Rate
	TotalChanges      int     `json:"totalChanges"`
	FailedChanges     int     `json:"failedChanges"`
	ChangeFailureRate float64 `json:"changeFailureRate"` // percentage

	// Mean Time to Recovery
	IncidentCount int     `json:"incidentCount"`
	AvgMTTR       float64 `json:"avgMTTR"` // hours
	MedianMTTR    float64 `json:"medianMTTR"`
}

// ProductivityScore represents the overall productivity score
type ProductivityScore struct {
	RepositoryID    string           `json:"repositoryId"`
	Period          string           `json:"period"`
	OverallScore    float64          `json:"overallScore"` // 0-100
	CycleTimeScore  float64          `json:"cycleTimeScore"`
	ReviewScore     float64          `json:"reviewScore"`
	DeploymentScore float64          `json:"deploymentScore"`
	QualityScore    float64          `json:"qualityScore"`
	TrendDirection  string           `json:"trendDirection"` // up, down, stable
	TrendPercentage float64          `json:"trendPercentage"`
	Recommendations []string         `json:"recommendations,omitempty"`
	ComponentScores []ComponentScore `json:"componentScores,omitempty"`
}

// ComponentScore represents a score component breakdown
type ComponentScore struct {
	Name        string  `json:"name"`
	Score       float64 `json:"score"`
	Weight      float64 `json:"weight"`
	Description string  `json:"description"`
}

// SprintPerformance represents sprint performance analysis
type SprintPerformance struct {
	SprintID   string    `json:"sprintId"`
	SprintName string    `json:"sprintName"`
	StartDate  time.Time `json:"startDate"`
	EndDate    time.Time `json:"endDate"`
	Status     string    `json:"status"` // planned, active, completed

	// Velocity
	PlannedItems   int     `json:"plannedItems"`
	CompletedItems int     `json:"completedItems"`
	CompletionRate float64 `json:"completionRate"`

	// PR Metrics
	PRsOpened int     `json:"prsOpened"`
	PRsMerged int     `json:"prsMerged"`
	AvgPRSize float64 `json:"avgPRSize"` // lines changed

	// Time Metrics
	AvgCycleTime  float64 `json:"avgCycleTime"`
	AvgReviewTime float64 `json:"avgReviewTime"`

	// Team Metrics
	ActiveContributors int `json:"activeContributors"`
	ReviewsSubmitted   int `json:"reviewsSubmitted"`

	// Comparison with previous sprint
	VelocityChange  float64 `json:"velocityChange"`
	CycleTimeChange float64 `json:"cycleTimeChange"`

	// Burndown data
	BurndownData []BurndownPoint `json:"burndownData,omitempty"`
}

// BurndownPoint represents a point in the burndown chart
type BurndownPoint struct {
	Date      time.Time `json:"date"`
	Planned   int       `json:"planned"`
	Remaining int       `json:"remaining"`
	Completed int       `json:"completed"`
}

// AIReport represents an AI-generated improvement report
type AIReport struct {
	RepositoryID    string           `json:"repositoryId"`
	GeneratedAt     time.Time        `json:"generatedAt"`
	Period          string           `json:"period"`
	Summary         string           `json:"summary"`
	Highlights      []string         `json:"highlights"`
	Concerns        []string         `json:"concerns"`
	Recommendations []Recommendation `json:"recommendations"`
	Predictions     []Prediction     `json:"predictions,omitempty"`
}

// Recommendation represents an improvement recommendation
type Recommendation struct {
	Category    string `json:"category"` // cycle_time, review, deployment, quality
	Priority    string `json:"priority"` // high, medium, low
	Title       string `json:"title"`
	Description string `json:"description"`
	Impact      string `json:"impact"`
	Effort      string `json:"effort"` // low, medium, high
}

// Prediction represents a metric prediction
type Prediction struct {
	Metric     string  `json:"metric"`
	Current    float64 `json:"current"`
	Predicted  float64 `json:"predicted"`
	Confidence float64 `json:"confidence"`
	Trend      string  `json:"trend"`
}
