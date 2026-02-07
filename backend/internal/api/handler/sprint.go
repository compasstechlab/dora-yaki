package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/compasstechlab/dora-yaki/internal/datastore"
	"github.com/compasstechlab/dora-yaki/internal/domain/model"
	"github.com/compasstechlab/dora-yaki/internal/metrics"
)

// SprintHandler handles sprint-related API requests
type SprintHandler struct {
	ds         *datastore.Client
	aggregator *metrics.Aggregator
	logger     *slog.Logger
}

// NewSprintHandler creates a new SprintHandler
func NewSprintHandler(ds *datastore.Client, logger *slog.Logger) *SprintHandler {
	return &SprintHandler{
		ds:         ds,
		aggregator: metrics.NewAggregator(),
		logger:     logger,
	}
}

// CreateSprintRequest request body for creating a sprint
type CreateSprintRequest struct {
	RepositoryID string `json:"repositoryId"`
	Name         string `json:"name"`
	StartDate    string `json:"startDate"`
	EndDate      string `json:"endDate"`
	Goals        string `json:"goals"`
}

// List lists all sprints for a repository
func (h *SprintHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	repoID := r.URL.Query().Get("repository")

	if repoID == "" {
		http.Error(w, "repository parameter is required", http.StatusBadRequest)
		return
	}

	sprints, err := h.ds.ListSprints(ctx, repoID)
	if err != nil {
		h.logger.Error("failed to list sprints", "error", err)
		http.Error(w, "failed to list sprints", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, sprints)
}

// Create creates a new sprint
func (h *SprintHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateSprintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.RepositoryID == "" || req.Name == "" || req.StartDate == "" || req.EndDate == "" {
		http.Error(w, "repositoryId, name, startDate, and endDate are required", http.StatusBadRequest)
		return
	}

	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		http.Error(w, "invalid startDate format (use YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		http.Error(w, "invalid endDate format (use YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	sprint := &model.Sprint{
		ID:           generateSprintID(req.RepositoryID, req.Name),
		RepositoryID: req.RepositoryID,
		Name:         req.Name,
		StartDate:    startDate,
		EndDate:      endDate,
		Goals:        req.Goals,
	}

	if err := h.ds.SaveSprint(ctx, sprint); err != nil {
		h.logger.Error("failed to save sprint", "error", err)
		http.Error(w, "failed to create sprint", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusCreated, sprint)
}

// Get returns a specific sprint
func (h *SprintHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := getSprintID(r)

	sprint, err := h.ds.GetSprint(ctx, id)
	if err != nil {
		http.Error(w, "sprint not found", http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, sprint)
}

// GetPerformance returns performance metrics for a sprint
func (h *SprintHandler) GetPerformance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := getSprintID(r)

	sprint, err := h.ds.GetSprint(ctx, id)
	if err != nil {
		http.Error(w, "sprint not found", http.StatusNotFound)
		return
	}

	// Get PRs and reviews for the sprint period
	prs, err := h.ds.ListPullRequestsByDateRange(ctx, sprint.RepositoryID, sprint.StartDate, sprint.EndDate)
	if err != nil {
		h.logger.Error("failed to list pull requests", "error", err)
		http.Error(w, "failed to get sprint performance", http.StatusInternalServerError)
		return
	}

	reviews, err := h.ds.ListReviewsByDateRange(ctx, sprint.RepositoryID, sprint.StartDate, sprint.EndDate)
	if err != nil {
		h.logger.Error("failed to list reviews", "error", err)
		http.Error(w, "failed to get sprint performance", http.StatusInternalServerError)
		return
	}

	// Calculate sprint performance
	performance := h.aggregator.CalculateSprintMetrics(sprint, prs, reviews)

	respondJSON(w, http.StatusOK, performance)
}

func generateSprintID(repoID, name string) string {
	safeName := strings.ReplaceAll(name, " ", "-")
	return repoID + ":" + safeName
}

func getSprintID(r *http.Request) string {
	path := r.URL.Path
	parts := strings.Split(path, "/")

	// /api/sprints/{id} or /api/sprints/{id}/performance
	if len(parts) >= 4 {
		if parts[len(parts)-1] == "performance" {
			return parts[len(parts)-2]
		}
		return parts[len(parts)-1]
	}

	return ""
}
