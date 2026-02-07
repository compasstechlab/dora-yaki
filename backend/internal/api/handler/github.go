package handler

import (
	"log/slog"
	"net/http"

	"github.com/compasstechlab/dora-yaki/internal/github"
)

// GitHubHandler is a proxy handler for GitHub API.
type GitHubHandler struct {
	gh     *github.Client
	logger *slog.Logger
}

// NewGitHubHandler creates a new GitHubHandler
func NewGitHubHandler(gh *github.Client, logger *slog.Logger) *GitHubHandler {
	return &GitHubHandler{
		gh:     gh,
		logger: logger,
	}
}

// GetMe returns the authenticated user info and org list.
func (h *GitHubHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user, err := h.gh.GetAuthenticatedUser(ctx)
	if err != nil {
		h.logger.Error("failed to get authenticated user", "error", err)
		http.Error(w, "failed to get authenticated user", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, user)
}

// ListOwnerRepos returns repositories belonging to an org or user.
func (h *GitHubHandler) ListOwnerRepos(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	owner := r.PathValue("owner")

	if owner == "" {
		http.Error(w, "owner is required", http.StatusBadRequest)
		return
	}

	repoType := r.URL.Query().Get("type")
	opts := &github.OrgRepoListOptions{
		Type: repoType,
	}

	repos, err := h.gh.ListOwnerRepos(ctx, owner, opts)
	if err != nil {
		h.logger.Error("failed to list owner repos", "error", err, "owner", owner)
		http.Error(w, "failed to list repos", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, repos)
}
