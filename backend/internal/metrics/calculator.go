package metrics

import (
	"sort"
	"time"

	"github.com/compasstechlab/dora-yaki/internal/domain/model"
)

// Calculator handles metrics calculations
type Calculator struct{}

// NewCalculator creates a new Calculator
func NewCalculator() *Calculator {
	return &Calculator{}
}

// CalculateCycleTime calculates cycle time metrics for pull requests
func (c *Calculator) CalculateCycleTime(prs []*model.PullRequest, startDate, endDate time.Time) *model.CycleTimeMetrics {
	// Filter merged PRs within the date range
	var mergedPRs []*model.PullRequest
	for _, pr := range prs {
		if pr.MergedAt != nil && !pr.MergedAt.Before(startDate) && !pr.MergedAt.After(endDate) {
			mergedPRs = append(mergedPRs, pr)
		}
	}

	if len(mergedPRs) == 0 {
		return &model.CycleTimeMetrics{
			Period:    "custom",
			StartDate: startDate,
			EndDate:   endDate,
			TotalPRs:  0,
		}
	}

	var cycleTimes, codingTimes, pickupTimes, reviewTimes, mergeTimes []float64

	authorMetricsMap := make(map[string]*model.AuthorMetrics)

	for _, pr := range mergedPRs {
		// Calculate individual times (in hours)
		cycleTime := pr.CycleTimeHours()
		codingTime := pr.CodingTimeHours()
		pickupTime := pr.PickupTimeHours()
		reviewTime := pr.ReviewTimeHours()
		mergeTime := pr.MergeTimeHours()

		if cycleTime > 0 {
			cycleTimes = append(cycleTimes, cycleTime)
		}
		if codingTime > 0 {
			codingTimes = append(codingTimes, codingTime)
		}
		if pickupTime > 0 {
			pickupTimes = append(pickupTimes, pickupTime)
		}
		if reviewTime > 0 {
			reviewTimes = append(reviewTimes, reviewTime)
		}
		if mergeTime > 0 {
			mergeTimes = append(mergeTimes, mergeTime)
		}

		// Aggregate by author
		if _, ok := authorMetricsMap[pr.Author]; !ok {
			authorMetricsMap[pr.Author] = &model.AuthorMetrics{
				Author: pr.Author,
			}
		}
		authorMetricsMap[pr.Author].PRCount++
		authorMetricsMap[pr.Author].Additions += pr.Additions
		authorMetricsMap[pr.Author].Deletions += pr.Deletions
		if cycleTime > 0 {
			authorMetricsMap[pr.Author].AvgCycleTime += cycleTime
		}
	}

	// Calculate averages for authors
	authorMetrics := make([]model.AuthorMetrics, 0, len(authorMetricsMap))
	for _, am := range authorMetricsMap {
		if am.PRCount > 0 {
			am.AvgCycleTime /= float64(am.PRCount)
		}
		authorMetrics = append(authorMetrics, *am)
	}

	// Sort by PR count
	sort.Slice(authorMetrics, func(i, j int) bool {
		return authorMetrics[i].PRCount > authorMetrics[j].PRCount
	})

	// Aggregate change stats by file extension
	byFileExtension := c.aggregateFileExtMetrics(mergedPRs)

	return &model.CycleTimeMetrics{
		Period:          "custom",
		StartDate:       startDate,
		EndDate:         endDate,
		TotalPRs:        len(mergedPRs),
		AvgCycleTime:    average(cycleTimes),
		AvgCodingTime:   average(codingTimes),
		AvgPickupTime:   average(pickupTimes),
		AvgReviewTime:   average(reviewTimes),
		AvgMergeTime:    average(mergeTimes),
		MedianCycleTime: median(cycleTimes),
		P90CycleTime:    percentile(cycleTimes, 90),
		ByAuthor:        authorMetrics,
		ByFileExtension: byFileExtension,
	}
}

// CalculateReviewMetrics calculates review analysis metrics
func (c *Calculator) CalculateReviewMetrics(reviews []*model.Review, prs []*model.PullRequest, startDate, endDate time.Time) *model.ReviewMetrics {
	// Filter reviews within date range
	var filteredReviews []*model.Review
	for _, review := range reviews {
		if !review.SubmittedAt.Before(startDate) && !review.SubmittedAt.After(endDate) {
			filteredReviews = append(filteredReviews, review)
		}
	}

	if len(filteredReviews) == 0 {
		return &model.ReviewMetrics{
			Period:    "custom",
			StartDate: startDate,
			EndDate:   endDate,
		}
	}

	// Count totals
	totalComments := 0
	approvedCount := 0
	changesRequestedCount := 0
	reviewerStatsMap := make(map[string]*model.ReviewerStats)

	for _, review := range filteredReviews {
		totalComments += review.CommentsCount

		switch review.State {
		case "APPROVED":
			approvedCount++
		case "CHANGES_REQUESTED":
			changesRequestedCount++
		}

		// Aggregate by reviewer
		if _, ok := reviewerStatsMap[review.Reviewer]; !ok {
			reviewerStatsMap[review.Reviewer] = &model.ReviewerStats{
				Reviewer: review.Reviewer,
			}
		}
		reviewerStatsMap[review.Reviewer].ReviewCount++
		reviewerStatsMap[review.Reviewer].CommentCount += review.CommentsCount
		if review.State == "APPROVED" {
			reviewerStatsMap[review.Reviewer].ApprovalRate++
		}
	}

	// Calculate reviewer stats
	reviewerStats := make([]model.ReviewerStats, 0, len(reviewerStatsMap))
	for _, rs := range reviewerStatsMap {
		if rs.ReviewCount > 0 {
			rs.ApprovalRate = (rs.ApprovalRate / float64(rs.ReviewCount)) * 100
		}
		reviewerStats = append(reviewerStats, *rs)
	}

	sort.Slice(reviewerStats, func(i, j int) bool {
		return reviewerStats[i].ReviewCount > reviewerStats[j].ReviewCount
	})

	// Calculate time to first review
	var timeToFirstReviews []float64
	for _, pr := range prs {
		if pr.FirstReviewAt != nil {
			ttfr := pr.FirstReviewAt.Sub(pr.CreatedAt).Hours()
			if ttfr > 0 {
				timeToFirstReviews = append(timeToFirstReviews, ttfr)
			}
		}
	}

	// Calculate reviews per PR
	prReviewCount := make(map[string]int)
	for _, review := range filteredReviews {
		prReviewCount[review.PullRequestID]++
	}

	reviewsPerPR := make([]float64, 0, len(prReviewCount))
	for _, count := range prReviewCount {
		reviewsPerPR = append(reviewsPerPR, float64(count))
	}

	totalReviews := len(filteredReviews)
	approvalRate := 0.0
	changesRequestedRate := 0.0
	if totalReviews > 0 {
		approvalRate = (float64(approvedCount) / float64(totalReviews)) * 100
		changesRequestedRate = (float64(changesRequestedCount) / float64(totalReviews)) * 100
	}

	return &model.ReviewMetrics{
		Period:               "custom",
		StartDate:            startDate,
		EndDate:              endDate,
		TotalReviews:         totalReviews,
		TotalComments:        totalComments,
		AvgReviewsPerPR:      average(reviewsPerPR),
		AvgCommentsPerReview: float64(totalComments) / float64(max(totalReviews, 1)),
		AvgTimeToFirstReview: average(timeToFirstReviews),
		ApprovalRate:         approvalRate,
		ChangesRequestedRate: changesRequestedRate,
		ByReviewer:           reviewerStats,
	}
}

// CalculateDORAMetrics calculates DORA metrics
func (c *Calculator) CalculateDORAMetrics(prs []*model.PullRequest, deployments []*model.Deployment, startDate, endDate time.Time) *model.DORAMetrics {
	// Calculate deployment frequency
	var filteredDeployments []*model.Deployment
	for _, d := range deployments {
		if !d.CreatedAt.Before(startDate) && !d.CreatedAt.After(endDate) {
			filteredDeployments = append(filteredDeployments, d)
		}
	}

	days := endDate.Sub(startDate).Hours() / 24
	if days == 0 {
		days = 1
	}

	deploymentCount := len(filteredDeployments)
	avgDeploysPerDay := float64(deploymentCount) / days

	// Determine deployment frequency category
	var deploymentFrequency string
	switch {
	case avgDeploysPerDay >= 1:
		deploymentFrequency = "daily"
	case avgDeploysPerDay >= 1.0/7:
		deploymentFrequency = "weekly"
	case avgDeploysPerDay >= 1.0/30:
		deploymentFrequency = "monthly"
	default:
		deploymentFrequency = "yearly"
	}

	// Calculate lead time for changes (PR creation to merge)
	var leadTimes []float64
	var mergedPRs []*model.PullRequest
	for _, pr := range prs {
		if pr.MergedAt != nil && !pr.MergedAt.Before(startDate) && !pr.MergedAt.After(endDate) {
			mergedPRs = append(mergedPRs, pr)
			leadTime := pr.MergedAt.Sub(pr.CreatedAt).Hours()
			if leadTime > 0 {
				leadTimes = append(leadTimes, leadTime)
			}
		}
	}

	// Calculate change failure rate (simplified: based on reverted PRs or bug fixes)
	failedChanges := 0
	for _, pr := range mergedPRs {
		// Simple heuristic: if a PR title contains "revert", "fix", or "hotfix", consider it a failed change
		// In a real implementation, this would be more sophisticated
		_ = pr // placeholder for actual failure detection logic
	}

	totalChanges := len(mergedPRs)
	changeFailureRate := 0.0
	if totalChanges > 0 {
		changeFailureRate = (float64(failedChanges) / float64(totalChanges)) * 100
	}

	return &model.DORAMetrics{
		Period:              "custom",
		StartDate:           startDate,
		EndDate:             endDate,
		DeploymentCount:     deploymentCount,
		DeploymentFrequency: deploymentFrequency,
		AvgDeploysPerDay:    avgDeploysPerDay,
		AvgLeadTime:         average(leadTimes),
		MedianLeadTime:      median(leadTimes),
		P90LeadTime:         percentile(leadTimes, 90),
		TotalChanges:        totalChanges,
		FailedChanges:       failedChanges,
		ChangeFailureRate:   changeFailureRate,
	}
}

// CalculateProductivityScore calculates the overall productivity score
func (c *Calculator) CalculateProductivityScore(
	cycleTime *model.CycleTimeMetrics,
	reviews *model.ReviewMetrics,
	dora *model.DORAMetrics,
) *model.ProductivityScore {
	// Weight configuration
	cycleTimeWeight := 0.30
	reviewWeight := 0.25
	deploymentWeight := 0.25
	qualityWeight := 0.20

	// Calculate component scores (0-100)
	cycleTimeScore := c.scoreCycleTime(cycleTime.AvgCycleTime)
	reviewScore := c.scoreReview(reviews)
	deploymentScore := c.scoreDeployment(dora)
	qualityScore := c.scoreQuality(dora.ChangeFailureRate)

	// Calculate overall score
	overallScore := cycleTimeScore*cycleTimeWeight +
		reviewScore*reviewWeight +
		deploymentScore*deploymentWeight +
		qualityScore*qualityWeight

	// Generate recommendations
	var recommendations []string
	if cycleTimeScore < 60 {
		recommendations = append(recommendations, "Consider breaking down PRs into smaller, more manageable pieces")
	}
	if reviewScore < 60 {
		recommendations = append(recommendations, "Review response time could be improved - consider setting review SLAs")
	}
	if deploymentScore < 60 {
		recommendations = append(recommendations, "Increase deployment frequency through automation and CI/CD improvements")
	}
	if qualityScore < 60 {
		recommendations = append(recommendations, "Focus on reducing change failure rate through better testing")
	}

	return &model.ProductivityScore{
		OverallScore:    overallScore,
		CycleTimeScore:  cycleTimeScore,
		ReviewScore:     reviewScore,
		DeploymentScore: deploymentScore,
		QualityScore:    qualityScore,
		TrendDirection:  "stable",
		Recommendations: recommendations,
		ComponentScores: []model.ComponentScore{
			{Name: "Cycle Time", Score: cycleTimeScore, Weight: cycleTimeWeight, Description: "Time from first commit to merge"},
			{Name: "Review Efficiency", Score: reviewScore, Weight: reviewWeight, Description: "Code review speed and quality"},
			{Name: "Deployment Frequency", Score: deploymentScore, Weight: deploymentWeight, Description: "How often code is deployed"},
			{Name: "Change Quality", Score: qualityScore, Weight: qualityWeight, Description: "Success rate of changes"},
		},
	}
}

// aggregateFileExtMetrics aggregates file extension stats from merged PRs.
func (c *Calculator) aggregateFileExtMetrics(prs []*model.PullRequest) []model.FileExtensionMetrics {
	type extAgg struct {
		additions int
		deletions int
		files     int
		prCount   int
	}
	m := make(map[string]*extAgg)

	for _, pr := range prs {
		// Track extensions in this PR (for PR count)
		seen := make(map[string]bool)
		for _, fs := range pr.FileExtStats {
			ext := fs.Extension
			a, ok := m[ext]
			if !ok {
				a = &extAgg{}
				m[ext] = a
			}
			a.additions += fs.Additions
			a.deletions += fs.Deletions
			a.files += fs.Files
			if !seen[ext] {
				a.prCount++
				seen[ext] = true
			}
		}
	}

	result := make([]model.FileExtensionMetrics, 0, len(m))
	for ext, a := range m {
		result = append(result, model.FileExtensionMetrics{
			Extension: ext,
			Additions: a.additions,
			Deletions: a.deletions,
			Files:     a.files,
			PRCount:   a.prCount,
		})
	}

	// Sort by number of changed lines descending
	sort.Slice(result, func(i, j int) bool {
		return (result[i].Additions + result[i].Deletions) > (result[j].Additions + result[j].Deletions)
	})

	return result
}

// Scoring helper functions

func (c *Calculator) scoreCycleTime(avgHours float64) float64 {
	// Scoring based on industry benchmarks
	// Elite: < 24h (1 day)
	// High: < 168h (1 week)
	// Medium: < 720h (1 month)
	// Low: >= 720h
	switch {
	case avgHours <= 24:
		return 100
	case avgHours <= 72:
		return 80
	case avgHours <= 168:
		return 60
	case avgHours <= 336:
		return 40
	default:
		return 20
	}
}

func (c *Calculator) scoreReview(metrics *model.ReviewMetrics) float64 {
	score := 50.0

	// Factor in time to first review (target: < 4h)
	if metrics.AvgTimeToFirstReview <= 4 {
		score += 25
	} else if metrics.AvgTimeToFirstReview <= 8 {
		score += 15
	} else if metrics.AvgTimeToFirstReview <= 24 {
		score += 5
	}

	// Factor in reviews per PR (target: 1-3)
	if metrics.AvgReviewsPerPR >= 1 && metrics.AvgReviewsPerPR <= 3 {
		score += 25
	} else if metrics.AvgReviewsPerPR > 0 {
		score += 10
	}

	return min(score, 100)
}

func (c *Calculator) scoreDeployment(metrics *model.DORAMetrics) float64 {
	switch metrics.DeploymentFrequency {
	case "daily":
		return 100
	case "weekly":
		return 75
	case "monthly":
		return 50
	default:
		return 25
	}
}

func (c *Calculator) scoreQuality(changeFailureRate float64) float64 {
	// Target: < 15% change failure rate
	switch {
	case changeFailureRate <= 5:
		return 100
	case changeFailureRate <= 10:
		return 80
	case changeFailureRate <= 15:
		return 60
	case changeFailureRate <= 30:
		return 40
	default:
		return 20
	}
}

// Statistical helper functions

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}

func percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	index := (p / 100) * float64(len(sorted)-1)
	lower := int(index)
	upper := lower + 1

	if upper >= len(sorted) {
		return sorted[lower]
	}

	weight := index - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}
