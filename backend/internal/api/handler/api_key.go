package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/compasstechlab/dora-yaki/internal/apikey"
	"github.com/compasstechlab/dora-yaki/internal/auth"
	"github.com/compasstechlab/dora-yaki/internal/domain/model"
)

const (
	maxAPIKeysPerUser       = 3 // expired keys count too; users delete them themselves
	defaultAPIKeyExpiryDays = 90
	maxAPIKeyNameLength     = 64 // in runes
)

// allowedAPIKeyExpiryDays : selectable key lifetimes. Keys never live longer than a year.
var allowedAPIKeyExpiryDays = []int{30, 90, 180, 365}

// apiKeyStore : persistence used by APIKeyHandler.
type apiKeyStore interface {
	SaveAPIKey(ctx context.Context, k *model.APIKey) error
	GetAPIKey(ctx context.Context, id string) (*model.APIKey, error)
	ListAPIKeysByUser(ctx context.Context, userID string) ([]*model.APIKey, error)
	DeleteAPIKey(ctx context.Context, id string) error
}

// APIKeyHandler : manages the caller's personal API keys.
type APIKeyHandler struct {
	store  apiKeyStore
	logger *slog.Logger
	now    func() time.Time
}

// NewAPIKeyHandler : creates an APIKeyHandler.
func NewAPIKeyHandler(store apiKeyStore, logger *slog.Logger) *APIKeyHandler {
	return &APIKeyHandler{store: store, logger: logger, now: time.Now}
}

// apiKeyResponse : public view of an API key.
type apiKeyResponse struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Prefix     string     `json:"prefix"`
	CreatedAt  time.Time  `json:"createdAt"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
}

// createAPIKeyResponse : response of 'POST /api/api-keys'.
type createAPIKeyResponse struct {
	APIKey apiKeyResponse `json:"apiKey"`
	Token  string         `json:"token"`
}

// createAPIKeyRequest : request of 'POST /api/api-keys'. ExpiresInDays=0 means the default.
type createAPIKeyRequest struct {
	Name          string `json:"name"`
	ExpiresInDays int    `json:"expiresInDays"`
}

func toAPIKeyResponse(k *model.APIKey) apiKeyResponse {
	return apiKeyResponse{
		ID:         k.ID,
		Name:       k.Name,
		Prefix:     k.DisplayPrefix(),
		CreatedAt:  k.CreatedAt,
		ExpiresAt:  k.ExpiresAt,
		LastUsedAt: k.LastUsedAt,
	}
}

// List : returns the caller's API keys, newest first.
func (h *APIKeyHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	keys, err := h.store.ListAPIKeysByUser(r.Context(), claims.UserID)
	if err != nil {
		h.logger.Error("failed to list api keys", "userID", claims.UserID, "error", err)
		http.Error(w, "failed to list api keys", http.StatusInternalServerError)
		return
	}
	resp := make([]apiKeyResponse, 0, len(keys))
	for _, k := range keys {
		resp = append(resp, toAPIKeyResponse(k))
	}
	respondJSON(w, http.StatusOK, resp)
}

// Create : issues a new API key for the caller.
func (h *APIKeyHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	req, msg := parseCreateAPIKeyRequest(r)
	if msg != "" {
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	existing, err := h.store.ListAPIKeysByUser(r.Context(), claims.UserID)
	if err != nil {
		h.logger.Error("failed to list api keys", "userID", claims.UserID, "error", err)
		http.Error(w, "failed to create api key", http.StatusInternalServerError)
		return
	}
	if len(existing) >= maxAPIKeysPerUser {
		http.Error(w, "api key limit reached", http.StatusConflict)
		return
	}

	token, id, hash, err := apikey.Generate()
	if err != nil {
		h.logger.Error("failed to generate api key", "userID", claims.UserID, "error", err)
		http.Error(w, "failed to create api key", http.StatusInternalServerError)
		return
	}
	now := h.now()
	expiresAt := now.AddDate(0, 0, req.ExpiresInDays)
	key := &model.APIKey{
		ID:         id,
		UserID:     claims.UserID,
		UserLogin:  claims.Login,
		Name:       req.Name,
		SecretHash: hash,
		ExpiresAt:  &expiresAt,
		CreatedAt:  now,
	}
	if err := h.store.SaveAPIKey(r.Context(), key); err != nil {
		h.logger.Error("failed to save api key", "apiKeyID", id, "userID", claims.UserID, "error", err)
		http.Error(w, "failed to create api key", http.StatusInternalServerError)
		return
	}

	h.logger.Info("api key created", "apiKeyID", id, "userID", claims.UserID)
	// [sec] The plaintext token is returned only in this response.
	w.Header().Set("Cache-Control", "no-store")
	respondJSON(w, http.StatusCreated, createAPIKeyResponse{APIKey: toAPIKeyResponse(key), Token: token})
}

// parseCreateAPIKeyRequest : decodes and validates the request. Returns a non-empty message on invalid input.
func parseCreateAPIKeyRequest(r *http.Request) (createAPIKeyRequest, string) {
	var req createAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, "invalid request body"
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || utf8.RuneCountInString(req.Name) > maxAPIKeyNameLength {
		return req, "name must be 1-64 characters"
	}
	if req.ExpiresInDays == 0 {
		req.ExpiresInDays = defaultAPIKeyExpiryDays
	}
	if !slices.Contains(allowedAPIKeyExpiryDays, req.ExpiresInDays) {
		return req, "expiresInDays must be one of 30, 90, 180, 365"
	}
	return req, ""
}

// Revoke : deletes one of the caller's API keys. Keys owned by others are reported as not found.
func (h *APIKeyHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id := getPathParam(r, "id")
	key, err := h.store.GetAPIKey(r.Context(), id)
	if err != nil {
		h.logger.Error("failed to get api key", "apiKeyID", id, "userID", claims.UserID, "error", err)
		http.Error(w, "failed to revoke api key", http.StatusInternalServerError)
		return
	}
	if key == nil || key.UserID != claims.UserID {
		http.Error(w, "api key not found", http.StatusNotFound)
		return
	}
	if err := h.store.DeleteAPIKey(r.Context(), id); err != nil {
		h.logger.Error("failed to delete api key", "apiKeyID", id, "userID", claims.UserID, "error", err)
		http.Error(w, "failed to revoke api key", http.StatusInternalServerError)
		return
	}
	h.logger.Info("api key revoked", "apiKeyID", id, "userID", claims.UserID)
	w.WriteHeader(http.StatusNoContent)
}
