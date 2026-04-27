package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/compasstechlab/dora-yaki/internal/datastore"
	"github.com/compasstechlab/dora-yaki/internal/domain/model"
	"github.com/compasstechlab/dora-yaki/internal/github"
)

const (
	permissionCheckLockID  = "permission-check"
	permissionCheckLockTTL = 30 * time.Minute
	permissionCheckWorkers = 5
)

// PermissionCheckHandler refreshes RepositoryAccess for every (user, repo)
// pair. Designed to be invoked daily by Cloud Scheduler.
type PermissionCheckHandler struct {
	ds      *datastore.Client
	factory *github.ClientFactory
	logger  *slog.Logger
}

// NewPermissionCheckHandler builds a PermissionCheckHandler.
func NewPermissionCheckHandler(
	ds *datastore.Client,
	factory *github.ClientFactory,
	logger *slog.Logger,
) *PermissionCheckHandler {
	return &PermissionCheckHandler{ds: ds, factory: factory, logger: logger}
}

// PermissionCheckResponse summarizes the result of a permission-check run.
type PermissionCheckResponse struct {
	Status      string    `json:"status"`
	Message     string    `json:"message"`
	UsersTotal  int       `json:"usersTotal"`
	ReposTotal  int       `json:"reposTotal"`
	Granted     int       `json:"granted"`
	Revoked     int       `json:"revoked"`
	StartedAt   time.Time `json:"startedAt"`
	FinishedAt  time.Time `json:"finishedAt"`
	DurationSec float64   `json:"durationSec"`
}

// Run is the HTTP entry point for the permission check job.
//
// PUT /api/job/permission-check
func (h *PermissionCheckHandler) Run(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	startedAt := time.Now()

	if h.factory == nil {
		respondJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "oauth not enabled",
		})
		return
	}

	instanceID := fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano())
	if err := h.ds.AcquireSyncLock(ctx, permissionCheckLockID, instanceID, permissionCheckLockTTL); err != nil {
		h.logger.Warn("permission-check skipped: lock held", "error", err)
		respondJSON(w, http.StatusConflict, map[string]string{
			"status":  "skipped",
			"message": err.Error(),
		})
		return
	}
	defer func() {
		if err := h.ds.ReleaseSyncLock(ctx, permissionCheckLockID, instanceID); err != nil {
			h.logger.Error("release permission-check lock", "error", err)
		}
	}()

	users, err := h.ds.ListUsers(ctx)
	if err != nil {
		h.logger.Error("list users", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	repos, err := h.ds.ListRepositories(ctx)
	if err != nil {
		h.logger.Error("list repositories", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	now := time.Now()
	var granted, revoked int
	var counterMu sync.Mutex

	for _, user := range users {
		userID := user.ID
		client, ferr := h.factory.ForUser(ctx, userID)
		if ferr != nil {
			// Token unusable — mark every existing access row as revoked.
			h.logger.Info("permission-check: user token unavailable; revoking", "userID", userID, "error", ferr)
			h.revokeAllForUser(ctx, userID, repos, now, &counterMu, &revoked)
			continue
		}

		// Probe each repository in parallel up to permissionCheckWorkers.
		var wg sync.WaitGroup
		jobs := make(chan *model.Repository)
		for i := 0; i < permissionCheckWorkers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for repo := range jobs {
					_, ferr := client.GetRepository(ctx, repo.Owner, repo.Name)
					row := &model.RepositoryAccess{
						UserID:           userID,
						RepositoryID:     repo.ID,
						CanAccess:        ferr == nil,
						IsTokenCandidate: ferr == nil,
						CheckedAt:        now,
					}
					if err := h.ds.SaveRepositoryAccess(ctx, row); err != nil {
						h.logger.Warn("save access", "userID", userID, "repoID", repo.ID, "error", err)
						continue
					}
					counterMu.Lock()
					if row.CanAccess {
						granted++
					} else {
						revoked++
					}
					counterMu.Unlock()
				}
			}()
		}
		for _, repo := range repos {
			jobs <- repo
		}
		close(jobs)
		wg.Wait()
	}

	finishedAt := time.Now()
	resp := &PermissionCheckResponse{
		Status:      "completed",
		Message:     fmt.Sprintf("checked %d users x %d repos", len(users), len(repos)),
		UsersTotal:  len(users),
		ReposTotal:  len(repos),
		Granted:     granted,
		Revoked:     revoked,
		StartedAt:   startedAt,
		FinishedAt:  finishedAt,
		DurationSec: finishedAt.Sub(startedAt).Seconds(),
	}
	h.logger.Info("permission-check completed",
		"users", len(users),
		"repos", len(repos),
		"granted", granted,
		"revoked", revoked,
		"durationSec", resp.DurationSec,
	)
	respondJSON(w, http.StatusOK, resp)
}

// revokeAllForUser marks every (user, repo) row as can_access=false.
func (h *PermissionCheckHandler) revokeAllForUser(
	ctx context.Context,
	userID string,
	repos []*model.Repository,
	now time.Time,
	counterMu *sync.Mutex,
	revoked *int,
) {
	for _, repo := range repos {
		row := &model.RepositoryAccess{
			UserID:           userID,
			RepositoryID:     repo.ID,
			CanAccess:        false,
			IsTokenCandidate: false,
			CheckedAt:        now,
		}
		if err := h.ds.SaveRepositoryAccess(ctx, row); err != nil {
			h.logger.Warn("save access", "userID", userID, "repoID", repo.ID, "error", err)
			continue
		}
		counterMu.Lock()
		*revoked++
		counterMu.Unlock()
	}
}
