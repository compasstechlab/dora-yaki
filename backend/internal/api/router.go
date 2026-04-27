package api

import (
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/compasstechlab/dora-yaki/internal/api/handler"
	"github.com/compasstechlab/dora-yaki/internal/api/middleware"
	"github.com/compasstechlab/dora-yaki/internal/auth"
	"github.com/compasstechlab/dora-yaki/internal/config"
	"github.com/compasstechlab/dora-yaki/internal/crypto"
	"github.com/compasstechlab/dora-yaki/internal/datastore"
	"github.com/compasstechlab/dora-yaki/internal/github"
)

// Router handles HTTP routing
type Router struct {
	mux        *http.ServeMux
	logger     *slog.Logger
	middleware func(http.Handler) http.Handler
	cache      *middleware.ResponseCache
}

// RouterDeps bundles the dependencies needed to construct the Router.
// All fields are required: OAuth login is mandatory.
type RouterDeps struct {
	DS            *datastore.Client
	Logger        *slog.Logger
	Config        *config.Config
	Encryptor     crypto.Encryptor
	OAuthConfig   *auth.GitHubOAuthConfig
	JWTSecret     []byte
	ClientFactory *github.ClientFactory
	TokenPool     *github.TokenPool
}

// NewRouterWithDeps creates a new Router with full auth wiring.
func NewRouterWithDeps(deps RouterDeps) *Router {
	cache := middleware.NewResponseCache(50*time.Minute, deps.DS, deps.Logger)

	r := &Router{
		mux:    http.NewServeMux(),
		logger: deps.Logger,
		cache:  cache,
	}

	r.middleware = middleware.Chain(
		middleware.Recovery(deps.Logger),
		middleware.Logger(deps.Logger),
		middleware.CORS(corsAllowedOrigins(deps.Config.FrontendURL)),
		middleware.RequestID(),
	)

	repoHandler := handler.NewRepositoryHandler(deps.DS, deps.ClientFactory, deps.TokenPool, deps.Logger, cache)
	metricsHandler := handler.NewMetricsHandler(deps.DS, deps.Logger)
	sprintHandler := handler.NewSprintHandler(deps.DS, deps.Logger)
	teamHandler := handler.NewTeamHandler(deps.DS, deps.Logger)
	githubHandler := handler.NewGitHubHandler(deps.ClientFactory, deps.Logger)
	botUserHandler := handler.NewBotUserHandler(deps.DS, deps.Logger)
	jobHandler := handler.NewJobHandler(deps.DS, deps.TokenPool, deps.Logger, cache, deps.Config)
	permissionCheckHandler := handler.NewPermissionCheckHandler(deps.DS, deps.ClientFactory, deps.Logger)
	authHandler := handler.NewAuthHandler(
		deps.DS,
		deps.Encryptor,
		deps.OAuthConfig,
		deps.ClientFactory,
		deps.JWTSecret,
		deps.Config.AuthCookieDomain,
		deps.Config.FrontendURL,
		deps.Logger,
	)

	r.registerRoutes(routeHandlers{
		repo:            repoHandler,
		metrics:         metricsHandler,
		sprint:          sprintHandler,
		team:            teamHandler,
		github:          githubHandler,
		botUser:         botUserHandler,
		job:             jobHandler,
		auth:            authHandler,
		permissionCheck: permissionCheckHandler,
	}, deps.JWTSecret, deps.Config.JobAuthKey)

	return r
}

func corsAllowedOrigins(frontendURL string) []string {
	if frontendURL == "" {
		return []string{"*"}
	}
	u, err := url.Parse(frontendURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return []string{"*"}
	}
	return []string{u.Scheme + "://" + u.Host}
}

type routeHandlers struct {
	repo            *handler.RepositoryHandler
	metrics         *handler.MetricsHandler
	sprint          *handler.SprintHandler
	team            *handler.TeamHandler
	github          *handler.GitHubHandler
	botUser         *handler.BotUserHandler
	job             *handler.JobHandler
	auth            *handler.AuthHandler
	permissionCheck *handler.PermissionCheckHandler
}

func (r *Router) registerRoutes(h routeHandlers, jwtSecret []byte, jobAuthKey string) {
	cached := r.cache.Middleware()
	gate := auth.RequireAuth(jwtSecret)
	jobGate := middleware.JobAuth(jobAuthKey)

	// Health check (always public).
	r.mux.HandleFunc("GET /health", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Cache invalidation (public for ops).
	r.mux.HandleFunc("POST /api/cache/invalidate", func(w http.ResponseWriter, req *http.Request) {
		r.cache.Invalidate()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Auth routes (public).
	r.mux.HandleFunc("GET /api/auth/github/login", h.auth.LoginRedirect)
	r.mux.HandleFunc("GET /api/auth/github/callback", h.auth.Callback)
	r.mux.HandleFunc("POST /api/auth/logout", h.auth.Logout)
	r.mux.HandleFunc("GET /api/auth/me", h.auth.Me)

	// Repository endpoints (gated).
	r.mux.Handle("GET /api/repositories", gate(cached(http.HandlerFunc(h.repo.List))))
	r.mux.Handle("POST /api/repositories", gate(http.HandlerFunc(h.repo.Add)))
	r.mux.Handle("GET /api/repositories/{id}", gate(http.HandlerFunc(h.repo.Get)))
	r.mux.Handle("DELETE /api/repositories/{id}", gate(http.HandlerFunc(h.repo.Delete)))
	r.mux.Handle("POST /api/repositories/batch", gate(http.HandlerFunc(h.repo.BatchAdd)))
	r.mux.Handle("POST /api/repositories/{id}/sync", gate(http.HandlerFunc(h.repo.Sync)))
	r.mux.Handle("GET /api/repositories/date-ranges", gate(cached(http.HandlerFunc(h.repo.DateRanges))))

	// GitHub proxy endpoints (gated).
	r.mux.Handle("GET /api/github/me", gate(http.HandlerFunc(h.github.GetMe)))
	r.mux.Handle("GET /api/github/owners/{owner}/repos", gate(http.HandlerFunc(h.github.ListOwnerRepos)))

	// Metrics endpoints (gated, cached).
	r.mux.Handle("GET /api/metrics/cycle-time", gate(cached(http.HandlerFunc(h.metrics.CycleTime))))
	r.mux.Handle("GET /api/metrics/reviews", gate(cached(http.HandlerFunc(h.metrics.Reviews))))
	r.mux.Handle("GET /api/metrics/dora", gate(cached(http.HandlerFunc(h.metrics.DORA))))
	r.mux.Handle("GET /api/metrics/productivity-score", gate(cached(http.HandlerFunc(h.metrics.ProductivityScore))))
	r.mux.Handle("GET /api/metrics/daily", gate(cached(http.HandlerFunc(h.metrics.DailyMetrics))))
	r.mux.Handle("GET /api/metrics/pull-requests", gate(cached(http.HandlerFunc(h.metrics.PullRequests))))

	// Sprint endpoints (gated).
	r.mux.Handle("GET /api/sprints", gate(http.HandlerFunc(h.sprint.List)))
	r.mux.Handle("POST /api/sprints", gate(http.HandlerFunc(h.sprint.Create)))
	r.mux.Handle("GET /api/sprints/{id}", gate(http.HandlerFunc(h.sprint.Get)))
	r.mux.Handle("GET /api/sprints/{id}/performance", gate(http.HandlerFunc(h.sprint.GetPerformance)))

	// Bot user endpoints (gated).
	r.mux.Handle("GET /api/bot-users", gate(http.HandlerFunc(h.botUser.List)))
	r.mux.Handle("POST /api/bot-users", gate(http.HandlerFunc(h.botUser.Add)))
	r.mux.Handle("DELETE /api/bot-users", gate(http.HandlerFunc(h.botUser.Delete)))

	// Job endpoints (scheduler/shared-key gated).
	r.mux.Handle("PUT /api/job/sync", jobGate(http.HandlerFunc(h.job.Sync)))
	r.mux.Handle("PUT /api/job/permission-check", jobGate(http.HandlerFunc(h.permissionCheck.Run)))

	// Team endpoints (gated, cached).
	r.mux.Handle("GET /api/team/members", gate(cached(http.HandlerFunc(h.team.ListMembers))))
	r.mux.Handle("GET /api/team/members/{id}/stats", gate(cached(http.HandlerFunc(h.team.GetMemberStats))))
	r.mux.Handle("GET /api/team/members/{id}/pull-requests", gate(cached(http.HandlerFunc(h.team.GetMemberPullRequests))))
	r.mux.Handle("GET /api/team/members/{id}/reviews", gate(cached(http.HandlerFunc(h.team.GetMemberReviews))))
}

// ServeHTTP implements http.Handler
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	handler := r.middleware(r.mux)
	handler.ServeHTTP(w, req)
}

// Handler returns the http.Handler
func (r *Router) Handler() http.Handler {
	return r.middleware(r.mux)
}
