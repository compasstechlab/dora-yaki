package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/compasstechlab/dora-yaki/internal/datastore"
	"github.com/compasstechlab/dora-yaki/internal/domain/model"
)

// BotUserHandler handles custom bot user management.
type BotUserHandler struct {
	ds     *datastore.Client
	logger *slog.Logger
}

// NewBotUserHandler creates a new BotUserHandler
func NewBotUserHandler(ds *datastore.Client, logger *slog.Logger) *BotUserHandler {
	return &BotUserHandler{
		ds:     ds,
		logger: logger,
	}
}

// addBotUserRequest is a request to add a bot user.
type addBotUserRequest struct {
	Username string `json:"username"`
}

// List returns the list of custom bot users.
func (h *BotUserHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	botUsers, err := h.ds.ListBotUsers(ctx)
	if err != nil {
		h.logger.Error("failed to list bot users", "error", err)
		http.Error(w, "failed to list bot users", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, botUsers)
}

// Add adds a custom bot user.
func (h *BotUserHandler) Add(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req addBotUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" {
		http.Error(w, "username is required", http.StatusBadRequest)
		return
	}

	botUser := &model.BotUser{
		Username:  req.Username,
		CreatedAt: time.Now(),
	}

	if err := h.ds.SaveBotUser(ctx, botUser); err != nil {
		h.logger.Error("failed to save bot user", "error", err)
		http.Error(w, "failed to save bot user", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusCreated, botUser)
}

// Delete removes a custom bot user.
func (h *BotUserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	username := r.URL.Query().Get("username")

	if username == "" {
		http.Error(w, "username query parameter is required", http.StatusBadRequest)
		return
	}

	if err := h.ds.DeleteBotUser(ctx, username); err != nil {
		h.logger.Error("failed to delete bot user", "error", err)
		http.Error(w, "failed to delete bot user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
