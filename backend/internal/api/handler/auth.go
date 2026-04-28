package handler

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/compasstechlab/dora-yaki/internal/auth"
	"github.com/compasstechlab/dora-yaki/internal/crypto"
	"github.com/compasstechlab/dora-yaki/internal/datastore"
	"github.com/compasstechlab/dora-yaki/internal/domain/model"
	"github.com/compasstechlab/dora-yaki/internal/github"
)

// AuthHandler handles GitHub OAuth login, logout, and current-user endpoints.
type AuthHandler struct {
	ds            *datastore.Client
	encryptor     crypto.Encryptor
	oauth         *auth.GitHubOAuthConfig
	factory       *github.ClientFactory
	signingSecret []byte
	cookieDomain  string
	frontendURL   string
	logger        *slog.Logger
}

// NewAuthHandler builds an AuthHandler with explicit dependencies.
//
// `factory` is used post-login to seed RepositoryAccess rows for the freshly
// authenticated user. May be nil during the auth-only bootstrap phase.
func NewAuthHandler(
	ds *datastore.Client,
	encryptor crypto.Encryptor,
	oauth *auth.GitHubOAuthConfig,
	factory *github.ClientFactory,
	signingSecret []byte,
	cookieDomain string,
	frontendURL string,
	logger *slog.Logger,
) *AuthHandler {
	return &AuthHandler{
		ds:            ds,
		encryptor:     encryptor,
		oauth:         oauth,
		factory:       factory,
		signingSecret: signingSecret,
		cookieDomain:  cookieDomain,
		frontendURL:   frontendURL,
		logger:        logger,
	}
}

// LoginRedirect kicks off the GitHub OAuth code-grant flow.
//
// GET /api/auth/github/login?return_to=/path
func (h *AuthHandler) LoginRedirect(w http.ResponseWriter, r *http.Request) {
	returnTo := auth.SanitizeReturnTo(r.URL.Query().Get("return_to"))
	state, err := auth.EncodeState(h.signingSecret, returnTo)
	if err != nil {
		h.logger.Error("encode state", "error", err)
		h.redirectToLogin(w, r, "state_encode_failed")
		return
	}
	http.Redirect(w, r, h.oauth.BuildAuthorizeURL(state), http.StatusFound)
}

// Callback completes the OAuth dance: verifies state, exchanges the code,
// fetches the GitHub user, persists encrypted tokens, sets the session cookie,
// and redirects to the frontend.
//
// GET /api/auth/github/callback?code=...&state=...
func (h *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()
	code := q.Get("code")
	state := q.Get("state")
	if code == "" || state == "" {
		h.redirectToLogin(w, r, "missing_params")
		return
	}
	returnTo, err := auth.DecodeState(h.signingSecret, state)
	if err != nil {
		h.logger.Warn("invalid oauth state", "error", err)
		h.redirectToLogin(w, r, "invalid_state")
		return
	}

	tokens, err := h.oauth.ExchangeCode(ctx, code)
	if err != nil {
		h.logger.Error("exchange code", "error", err)
		h.redirectToLogin(w, r, "token_exchange_failed")
		return
	}

	ghClient := github.NewClient(tokens.AccessToken)
	ghUser, err := ghClient.GetAuthenticatedUser(ctx)
	if err != nil {
		h.logger.Error("fetch github user", "error", err)
		h.redirectToLogin(w, r, "user_fetch_failed")
		return
	}

	userID, err := h.findOrCreateUser(ctx, ghUser)
	if err != nil {
		h.logger.Error("upsert user", "error", err)
		h.redirectToLogin(w, r, "user_persist_failed")
		return
	}

	if err := h.saveEncryptedTokens(ctx, userID, tokens); err != nil {
		h.logger.Error("persist tokens", "error", err)
		h.redirectToLogin(w, r, "token_persist_failed")
		return
	}

	sessionToken, err := auth.SignToken(h.signingSecret, auth.NewClaims(userID, ghUser.Login))
	if err != nil {
		h.logger.Error("sign session token", "error", err)
		h.redirectToLogin(w, r, "session_sign_failed")
		return
	}
	auth.SetSessionCookie(w, r, sessionToken, h.cookieDomain, auth.TokenTTL)

	h.logger.Info("user logged in", "userID", userID, "login", ghUser.Login)
	go h.bootstrapRepositoryAccess(userID) //nolint:gosec,contextcheck // intentional: must outlive the request that triggered the redirect

	http.Redirect(w, r, h.absoluteFrontendURL(returnTo), http.StatusFound)
}

// bootstrapRepositoryAccess seeds RepositoryAccess rows for a newly logged-in
// user by probing every known repository with their token. Runs in the
// background so the OAuth redirect is not blocked.
func (h *AuthHandler) bootstrapRepositoryAccess(userID string) {
	if h.factory == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	repos, err := h.ds.ListRepositories(ctx)
	if err != nil {
		h.logger.Warn("bootstrap: list repos", "userID", userID, "error", err)
		return
	}
	if len(repos) == 0 {
		return
	}
	client, err := h.factory.ForUser(ctx, userID)
	if err != nil {
		h.logger.Warn("bootstrap: build client", "userID", userID, "error", err)
		return
	}

	now := time.Now()
	for _, repo := range repos {
		_, ferr := client.GetRepository(ctx, repo.Owner, repo.Name)
		canAccess := ferr == nil
		row := &model.RepositoryAccess{
			UserID:           userID,
			RepositoryID:     repo.ID,
			CanAccess:        canAccess,
			IsTokenCandidate: canAccess,
			CheckedAt:        now,
		}
		if err := h.ds.SaveRepositoryAccess(ctx, row); err != nil {
			h.logger.Warn("bootstrap: save access", "userID", userID, "repoID", repo.ID, "error", err)
		}
	}
	h.logger.Info("bootstrap: repository access seeded", "userID", userID, "repoCount", len(repos))
}

// Logout clears the session cookie. The encrypted token row is preserved so
// the user remains a sync candidate for repos they have access to.
//
// POST /api/auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	auth.ClearSessionCookie(w, r, h.cookieDomain)
	respondJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}

// Me returns the currently authenticated user, or 401 if no valid session.
//
// GET /api/auth/me
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ExtractClaims(r, h.signingSecret)
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	user, err := h.ds.GetUser(r.Context(), claims.UserID)
	if err != nil {
		h.logger.Error("get user", "error", err, "userID", claims.UserID)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	if user == nil {
		// Cookie valid but user gone — treat as unauthenticated.
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"id":        user.ID,
		"login":     user.Login,
		"name":      user.Name,
		"avatarUrl": user.AvatarURL,
	})
}

func (h *AuthHandler) findOrCreateUser(ctx context.Context, gh *github.GitHubUser) (string, error) {
	userID := strconv.FormatInt(gh.ID, 10)
	existing, err := h.ds.GetUser(ctx, userID)
	if err != nil {
		return "", err
	}
	now := time.Now()

	if existing == nil {
		user := &model.User{
			ID:          userID,
			Login:       gh.Login,
			Name:        gh.Name,
			AvatarURL:   gh.AvatarURL,
			Email:       gh.Email,
			CreatedAt:   now,
			UpdatedAt:   now,
			LastLoginAt: now,
		}
		if err := h.ds.SaveUser(ctx, user); err != nil {
			return "", err
		}
		return userID, nil
	}

	// Login is authoritative from GitHub; the optional fields only overwrite
	// stored values when GitHub returned a non-empty value (so we don't wipe
	// previously synced data when GitHub omits it).
	existing.Login = gh.Login
	if gh.Name != "" {
		existing.Name = gh.Name
	}
	if gh.AvatarURL != "" {
		existing.AvatarURL = gh.AvatarURL
	}
	if gh.Email != "" {
		existing.Email = gh.Email
	}
	existing.UpdatedAt = now
	existing.LastLoginAt = now
	if err := h.ds.SaveUser(ctx, existing); err != nil {
		return "", err
	}
	return userID, nil
}

func (h *AuthHandler) saveEncryptedTokens(ctx context.Context, userID string, tr *auth.TokenResponse) error {
	now := time.Now()

	accessCipher, err := h.encryptor.Encrypt(ctx, []byte(tr.AccessToken))
	if err != nil {
		return err
	}
	var refreshCipher []byte
	if tr.RefreshToken != "" {
		refreshCipher, err = h.encryptor.Encrypt(ctx, []byte(tr.RefreshToken))
		if err != nil {
			return err
		}
	}

	existing, err := h.ds.GetUserGitHubToken(ctx, userID)
	if err != nil {
		return err
	}

	row := &model.UserGitHubToken{
		UserID:                userID,
		AccessTokenCipher:     accessCipher,
		RefreshTokenCipher:    refreshCipher,
		AccessTokenExpiresAt:  tr.AccessTokenExpiresAt(now),
		RefreshTokenExpiresAt: tr.RefreshTokenExpiresAt(now),
		Scopes:                splitScopes(tr.Scope),
		MarkedInvalidAt:       nil,
		UpdatedAt:             now,
	}
	if existing != nil {
		row.CreatedAt = existing.CreatedAt
		row.LastUsedAt = existing.LastUsedAt
		row.LastRefreshedAt = existing.LastRefreshedAt
	} else {
		row.CreatedAt = now
	}
	return h.ds.SaveUserGitHubToken(ctx, row)
}

func splitScopes(scope string) []string {
	if scope == "" {
		return nil
	}
	parts := strings.Split(scope, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func (h *AuthHandler) absoluteFrontendURL(returnTo string) string {
	if h.frontendURL == "" {
		return returnTo
	}
	// If FrontendURL already has a path component, append returnTo to it.
	u, err := url.Parse(h.frontendURL)
	if err != nil || u.Scheme == "" {
		// FrontendURL is a relative path — concatenate.
		return strings.TrimRight(h.frontendURL, "/") + returnTo
	}
	u.Path = strings.TrimRight(u.Path, "/") + returnTo
	u.RawQuery = ""
	return u.String()
}

func (h *AuthHandler) redirectToLogin(w http.ResponseWriter, r *http.Request, errCode string) {
	target := h.absoluteFrontendURL("/login") + "?error=" + url.QueryEscape(errCode)
	http.Redirect(w, r, target, http.StatusFound)
}
