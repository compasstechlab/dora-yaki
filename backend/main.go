package app

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"sync"

	"github.com/compasstechlab/dora-yaki/internal/api"
	"github.com/compasstechlab/dora-yaki/internal/config"
	"github.com/compasstechlab/dora-yaki/internal/datastore"
	"github.com/compasstechlab/dora-yaki/internal/github"
	"github.com/compasstechlab/dora-yaki/internal/timeutil"
)

var (
	router   http.Handler
	initOnce sync.Once
)

// Init initializes the application.
func Init() {
	initOnce.Do(func() {
		// Initialize logger
		logLevel := slog.LevelInfo
		if os.Getenv("DEBUG") == "true" {
			logLevel = slog.LevelDebug
		}
		logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: logLevel,
		}))
		slog.SetDefault(logger)

		// Load configuration
		cfg := config.Load()

		// Initialize timezone
		timeutil.Init(cfg.Location())
		logger.Info("timezone initialized", "location", cfg.Location().String())

		logger.Info("initializing application",
			"environment", cfg.Environment,
		)

		// Initialize GitHub client
		ghClient := github.NewClient(cfg.GitHubToken)

		// Initialize Datastore client
		var dsClient *datastore.Client
		if cfg.GCPProjectID != "" {
			logger.Info("using GCP project", "projectID", cfg.GCPProjectID)
			var err error
			dsClient, err = datastore.NewClient(context.Background(), cfg.GCPProjectID)
			if err != nil {
				logger.Error("failed to create datastore client", "error", err)
				os.Exit(1)
			}
		} else {
			logger.Warn("GCP project ID not resolved (env/metadata), running without datastore")
		}

		// Create router
		r := api.NewRouter(dsClient, ghClient, logger, cfg)
		router = r.Handler()
	})
}

// RunHTTPServer is the Cloud Functions entry point.
func RunHTTPServer(w http.ResponseWriter, r *http.Request) {
	Init()
	router.ServeHTTP(w, r)
}
