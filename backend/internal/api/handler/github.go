package handler

import (
	"log/slog"
	"net/http"

	"github.com/compasstechlab/dora-yaki/internal/auth"
	"github.com/compasstechlab/dora-yaki/internal/github"
)

// GitHubHandler is a proxy handler for GitHub API. It always uses a per-user client.
type GitHubHandler struct {
	factory *github.ClientFactory
	logger  *slog.Logger
}

// NewGitHubHandler creates a new GitHubHandler.
func NewGitHubHandler(factory *github.ClientFactory, logger *slog.Logger) *GitHubHandler {
	return &GitHubHandler{factory: factory, logger: logger}
}

// clientForRequest returns the GitHub client for the calling user.
// Returns an error if claims are missing or the token cannot be used.
func (h *GitHubHandler) clientForRequest(r *http.Request) (*github.Client, error) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		return nil, github.ErrNoToken
	}
	return h.factory.ForUser(r.Context(), claims.UserID)
}

// GetMe returns the authenticated user info and org list, scoped to the calling user's token.
func (h *GitHubHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	client, err := h.clientForRequest(r)
	if err != nil {
		h.logger.Warn("github client for request", "error", err)
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "github_token_invalid"})
		return
	}

	user, err := client.GetAuthenticatedUser(r.Context())
	if err != nil {
		h.logger.Error("failed to get authenticated user", "error", err)
		http.Error(w, "failed to get authenticated user", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, user)
}

// ListOwnerRepos returns repositories belonging to an org or user, using the calling user's token.
func (h *GitHubHandler) ListOwnerRepos(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("owner")
	if owner == "" {
		http.Error(w, "owner is required", http.StatusBadRequest)
		return
	}

	client, err := h.clientForRequest(r)
	if err != nil {
		h.logger.Warn("github client for request", "error", err)
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "github_token_invalid"})
		return
	}

	repoType := r.URL.Query().Get("type")
	opts := &github.OrgRepoListOptions{Type: repoType}

	repos, err := client.ListOwnerRepos(r.Context(), owner, opts)
	if err != nil {
		h.logger.Error("failed to list owner repos", "error", err, "owner", owner)
		http.Error(w, "failed to list repos", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, repos)
}
