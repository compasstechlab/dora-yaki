package metrics

import (
	"fmt"
	"time"

	"github.com/compasstechlab/dora-yaki/internal/domain/model"
	"github.com/compasstechlab/dora-yaki/internal/timeutil"
)

// Aggregator handles metrics aggregation
type Aggregator struct {
	calculator *Calculator
}

// NewAggregator creates a new Aggregator
func NewAggregator() *Aggregator {
	return &Aggregator{
		calculator: NewCalculator(),
	}
}

// AggregateDailyMetrics aggregates metrics for a specific date
func (a *Aggregator) AggregateDailyMetrics(
	repositoryID string,
	date time.Time,
	prs []*model.PullRequest,
	reviews []*model.Review,
	deployments []*model.Deployment,
) *model.DailyMetrics {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	// Filter data for this day
	var dayPRsOpened, dayPRsMerged, dayPRsClosed []*model.PullRequest
	for _, pr := range prs {
		if !pr.CreatedAt.Before(startOfDay) && pr.CreatedAt.Before(endOfDay) {
			dayPRsOpened = append(dayPRsOpened, pr)
		}
		if pr.MergedAt != nil && !pr.MergedAt.Before(startOfDay) && pr.MergedAt.Before(endOfDay) {
			dayPRsMerged = append(dayPRsMerged, pr)
		}
		if pr.ClosedAt != nil && !pr.ClosedAt.Before(startOfDay) && pr.ClosedAt.Before(endOfDay) {
			dayPRsClosed = append(dayPRsClosed, pr)
		}
	}

	var dayReviews []*model.Review
	for _, r := range reviews {
		if !r.SubmittedAt.Before(startOfDay) && r.SubmittedAt.Before(endOfDay) {
			dayReviews = append(dayReviews, r)
		}
	}

	var dayDeployments []*model.Deployment
	for _, d := range deployments {
		if !d.CreatedAt.Before(startOfDay) && d.CreatedAt.Before(endOfDay) {
			dayDeployments = append(dayDeployments, d)
		}
	}

	// Calculate cycle time for merged PRs
	cycleTimeMetrics := a.calculator.CalculateCycleTime(dayPRsMerged, startOfDay, endOfDay)

	// Calculate code changes
	totalAdditions, totalDeletions := 0, 0
	for _, pr := range dayPRsMerged {
		totalAdditions += pr.Additions
		totalDeletions += pr.Deletions
	}

	// Count active contributors
	contributors := make(map[string]bool)
	for _, pr := range dayPRsOpened {
		contributors[pr.Author] = true
	}
	for _, pr := range dayPRsMerged {
		contributors[pr.Author] = true
	}
	for _, r := range dayReviews {
		contributors[r.Reviewer] = true
	}

	// Calculate reviews per PR
	avgReviewsPerPR := 0.0
	if len(dayPRsMerged) > 0 {
		avgReviewsPerPR = float64(len(dayReviews)) / float64(len(dayPRsMerged))
	}

	return &model.DailyMetrics{
		ID:                 fmt.Sprintf("%s:%s", repositoryID, date.Format("2006-01-02")),
		RepositoryID:       repositoryID,
		Date:               startOfDay,
		AvgCycleTime:       cycleTimeMetrics.AvgCycleTime,
		AvgCodingTime:      cycleTimeMetrics.AvgCodingTime,
		AvgPickupTime:      cycleTimeMetrics.AvgPickupTime,
		AvgReviewTime:      cycleTimeMetrics.AvgReviewTime,
		AvgMergeTime:       cycleTimeMetrics.AvgMergeTime,
		PRsOpened:          len(dayPRsOpened),
		PRsMerged:          len(dayPRsMerged),
		PRsClosed:          len(dayPRsClosed),
		ReviewsSubmitted:   len(dayReviews),
		AvgReviewsPerPR:    avgReviewsPerPR,
		TotalAdditions:     totalAdditions,
		TotalDeletions:     totalDeletions,
		DeploymentCount:    len(dayDeployments),
		ActiveContributors: len(contributors),
	}
}

// AggregateRange aggregates metrics for a date range
func (a *Aggregator) AggregateRange(
	repositoryID string,
	startDate, endDate time.Time,
	prs []*model.PullRequest,
	reviews []*model.Review,
	deployments []*model.Deployment,
) []*model.DailyMetrics {
	var dailyMetrics []*model.DailyMetrics

	current := startDate
	for !current.After(endDate) {
		metrics := a.AggregateDailyMetrics(repositoryID, current, prs, reviews, deployments)
		dailyMetrics = append(dailyMetrics, metrics)
		current = current.AddDate(0, 0, 1)
	}

	return dailyMetrics
}

// CalculateSprintMetrics calculates metrics for a sprint
func (a *Aggregator) CalculateSprintMetrics(
	sprint *model.Sprint,
	prs []*model.PullRequest,
	reviews []*model.Review,
) *model.SprintPerformance {
	// Filter PRs within sprint date range
	var sprintPRsOpened, sprintPRsMerged []*model.PullRequest
	for _, pr := range prs {
		if !pr.CreatedAt.Before(sprint.StartDate) && !pr.CreatedAt.After(sprint.EndDate) {
			sprintPRsOpened = append(sprintPRsOpened, pr)
		}
		if pr.MergedAt != nil && !pr.MergedAt.Before(sprint.StartDate) && !pr.MergedAt.After(sprint.EndDate) {
			sprintPRsMerged = append(sprintPRsMerged, pr)
		}
	}

	// Filter reviews within sprint
	var sprintReviews []*model.Review
	for _, r := range reviews {
		if !r.SubmittedAt.Before(sprint.StartDate) && !r.SubmittedAt.After(sprint.EndDate) {
			sprintReviews = append(sprintReviews, r)
		}
	}

	// Calculate metrics
	cycleTimeMetrics := a.calculator.CalculateCycleTime(sprintPRsMerged, sprint.StartDate, sprint.EndDate)
	reviewMetrics := a.calculator.CalculateReviewMetrics(sprintReviews, prs, sprint.StartDate, sprint.EndDate)

	// Calculate average PR size
	totalLines := 0
	for _, pr := range sprintPRsMerged {
		totalLines += pr.Additions + pr.Deletions
	}
	avgPRSize := 0.0
	if len(sprintPRsMerged) > 0 {
		avgPRSize = float64(totalLines) / float64(len(sprintPRsMerged))
	}

	// Count active contributors
	contributors := make(map[string]bool)
	for _, pr := range sprintPRsOpened {
		contributors[pr.Author] = true
	}
	for _, r := range sprintReviews {
		contributors[r.Reviewer] = true
	}

	// Determine sprint status
	status := "planned"
	now := timeutil.Now()
	if now.After(sprint.StartDate) && now.Before(sprint.EndDate) {
		status = "active"
	} else if now.After(sprint.EndDate) {
		status = "completed"
	}

	// Generate burndown data
	burndownData := a.generateBurndown(sprint, sprintPRsMerged)

	return &model.SprintPerformance{
		SprintID:           sprint.ID,
		SprintName:         sprint.Name,
		StartDate:          sprint.StartDate,
		EndDate:            sprint.EndDate,
		Status:             status,
		PlannedItems:       len(sprintPRsOpened),
		CompletedItems:     len(sprintPRsMerged),
		CompletionRate:     calculateCompletionRate(len(sprintPRsOpened), len(sprintPRsMerged)),
		PRsOpened:          len(sprintPRsOpened),
		PRsMerged:          len(sprintPRsMerged),
		AvgPRSize:          avgPRSize,
		AvgCycleTime:       cycleTimeMetrics.AvgCycleTime,
		AvgReviewTime:      reviewMetrics.AvgTimeToFirstReview,
		ActiveContributors: len(contributors),
		ReviewsSubmitted:   len(sprintReviews),
		BurndownData:       burndownData,
	}
}

func (a *Aggregator) generateBurndown(sprint *model.Sprint, mergedPRs []*model.PullRequest) []model.BurndownPoint {
	var burndownData []model.BurndownPoint

	totalPlanned := len(mergedPRs) // Simplified: using merged PRs as planned
	current := sprint.StartDate

	for !current.After(sprint.EndDate) && !current.After(timeutil.Now()) {
		completed := 0
		for _, pr := range mergedPRs {
			if pr.MergedAt != nil && !pr.MergedAt.After(current) {
				completed++
			}
		}

		burndownData = append(burndownData, model.BurndownPoint{
			Date:      current,
			Planned:   totalPlanned,
			Remaining: totalPlanned - completed,
			Completed: completed,
		})

		current = current.AddDate(0, 0, 1)
	}

	return burndownData
}

func calculateCompletionRate(planned, completed int) float64 {
	if planned == 0 {
		return 0
	}
	return (float64(completed) / float64(planned)) * 100
}
