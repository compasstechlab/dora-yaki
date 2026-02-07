package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/compasstechlab/dora-yaki/internal/api/handler"
	"github.com/compasstechlab/dora-yaki/internal/api/middleware"
	"github.com/compasstechlab/dora-yaki/internal/config"
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

// NewRouter creates a new Router
func NewRouter(ds *datastore.Client, gh *github.Client, logger *slog.Logger, cfg *config.Config) *Router {
	// Create a 3-tier response cache with 50-minute TTL
	cache := middleware.NewResponseCache(50*time.Minute, ds, logger)

	r := &Router{
		mux:    http.NewServeMux(),
		logger: logger,
		cache:  cache,
	}

	// Setup middleware chain
	r.middleware = middleware.Chain(
		middleware.Recovery(logger),
		middleware.Logger(logger),
		middleware.CORS([]string{"*"}),
		middleware.RequestID(),
	)

	// Initialize handlers
	repoHandler := handler.NewRepositoryHandler(ds, gh, logger, cache)
	metricsHandler := handler.NewMetricsHandler(ds, logger)
	sprintHandler := handler.NewSprintHandler(ds, logger)
	teamHandler := handler.NewTeamHandler(ds, logger)
	githubHandler := handler.NewGitHubHandler(gh, logger)
	botUserHandler := handler.NewBotUserHandler(ds, logger)
	jobHandler := handler.NewJobHandler(ds, gh, logger, cache, cfg)

	// Register routes
	r.registerRoutes(repoHandler, metricsHandler, sprintHandler, teamHandler, githubHandler, botUserHandler, jobHandler)

	return r
}

func (r *Router) registerRoutes(
	repoHandler *handler.RepositoryHandler,
	metricsHandler *handler.MetricsHandler,
	sprintHandler *handler.SprintHandler,
	teamHandler *handler.TeamHandler,
	githubHandler *handler.GitHubHandler,
	botUserHandler *handler.BotUserHandler,
	jobHandler *handler.JobHandler,
) {
	// Cache middleware
	cached := r.cache.Middleware()

	// Health check
	r.mux.HandleFunc("GET /health", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Cache invalidation endpoint
	r.mux.HandleFunc("POST /api/cache/invalidate", func(w http.ResponseWriter, req *http.Request) {
		r.cache.Invalidate()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Repository endpoints (list is cached)
	r.mux.Handle("GET /api/repositories", cached(http.HandlerFunc(repoHandler.List)))
	r.mux.HandleFunc("POST /api/repositories", repoHandler.Add)
	r.mux.HandleFunc("GET /api/repositories/{id}", repoHandler.Get)
	r.mux.HandleFunc("DELETE /api/repositories/{id}", repoHandler.Delete)
	r.mux.HandleFunc("POST /api/repositories/batch", repoHandler.BatchAdd)
	r.mux.HandleFunc("POST /api/repositories/{id}/sync", repoHandler.Sync)
	r.mux.Handle("GET /api/repositories/date-ranges", cached(http.HandlerFunc(repoHandler.DateRanges)))

	// GitHub proxy endpoints
	r.mux.HandleFunc("GET /api/github/me", githubHandler.GetMe)
	r.mux.HandleFunc("GET /api/github/owners/{owner}/repos", githubHandler.ListOwnerRepos)

	// Metrics endpoints (cached)
	r.mux.Handle("GET /api/metrics/cycle-time", cached(http.HandlerFunc(metricsHandler.CycleTime)))
	r.mux.Handle("GET /api/metrics/reviews", cached(http.HandlerFunc(metricsHandler.Reviews)))
	r.mux.Handle("GET /api/metrics/dora", cached(http.HandlerFunc(metricsHandler.DORA)))
	r.mux.Handle("GET /api/metrics/productivity-score", cached(http.HandlerFunc(metricsHandler.ProductivityScore)))
	r.mux.Handle("GET /api/metrics/daily", cached(http.HandlerFunc(metricsHandler.DailyMetrics)))
	r.mux.Handle("GET /api/metrics/pull-requests", cached(http.HandlerFunc(metricsHandler.PullRequests)))

	// Sprint endpoints
	r.mux.HandleFunc("GET /api/sprints", sprintHandler.List)
	r.mux.HandleFunc("POST /api/sprints", sprintHandler.Create)
	r.mux.HandleFunc("GET /api/sprints/{id}", sprintHandler.Get)
	r.mux.HandleFunc("GET /api/sprints/{id}/performance", sprintHandler.GetPerformance)

	// Bot user endpoints
	r.mux.HandleFunc("GET /api/bot-users", botUserHandler.List)
	r.mux.HandleFunc("POST /api/bot-users", botUserHandler.Add)
	r.mux.HandleFunc("DELETE /api/bot-users", botUserHandler.Delete)

	// Job endpoints
	r.mux.HandleFunc("PUT /api/job/sync", jobHandler.Sync)

	// Team endpoints (cached)
	r.mux.Handle("GET /api/team/members", cached(http.HandlerFunc(teamHandler.ListMembers)))
	r.mux.Handle("GET /api/team/members/{id}/stats", cached(http.HandlerFunc(teamHandler.GetMemberStats)))
	r.mux.Handle("GET /api/team/members/{id}/pull-requests", cached(http.HandlerFunc(teamHandler.GetMemberPullRequests)))
	r.mux.Handle("GET /api/team/members/{id}/reviews", cached(http.HandlerFunc(teamHandler.GetMemberReviews)))
}

// ServeHTTP implements http.Handler
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Apply middleware and serve
	handler := r.middleware(r.mux)
	handler.ServeHTTP(w, req)
}

// Handler returns the http.Handler
func (r *Router) Handler() http.Handler {
	return r.middleware(r.mux)
}
