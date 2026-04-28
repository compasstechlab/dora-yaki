package app

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"sync"

	"github.com/compasstechlab/dora-yaki/internal/api"
	"github.com/compasstechlab/dora-yaki/internal/auth"
	"github.com/compasstechlab/dora-yaki/internal/config"
	"github.com/compasstechlab/dora-yaki/internal/crypto"
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
		if err := cfg.MustValidate(); err != nil {
			logger.Error("invalid configuration", "error", err)
			os.Exit(1)
		}

		// Initialize timezone
		timeutil.Init(cfg.Location())
		logger.Info("timezone initialized", "location", cfg.Location().String())

		logger.Info("initializing application",
			"environment", cfg.Environment,
		)

		// Initialize Datastore client
		if cfg.GCPProjectID == "" {
			logger.Error("GCP project ID not resolved (env/metadata); set GCP_PROJECT_ID")
			os.Exit(1)
		}
		logger.Info("using GCP project", "projectID", cfg.GCPProjectID)
		dsClient, err := datastore.NewClient(context.Background(), cfg.GCPProjectID)
		if err != nil {
			logger.Error("failed to create datastore client", "error", err)
			os.Exit(1)
		}

		// Initialize encryption.
		encryptor, err := crypto.NewFromConfig(context.Background(), crypto.FactoryConfig{
			KMSKey:    cfg.EncryptionKMSKey,
			KeyBase64: cfg.EncryptionKeyBase64,
		})
		if err != nil {
			logger.Error("failed to initialize encryptor", "error", err)
			os.Exit(1)
		}

		// Initialize OAuth wiring.
		oauthCfg := &auth.GitHubOAuthConfig{
			ClientID:     cfg.GitHubOAuthClientID,
			ClientSecret: cfg.GitHubOAuthClientSecret,
			RedirectURL:  cfg.OAuthRedirectURL,
		}
		jwtSecret := []byte(cfg.AuthJWTSecret)
		clientFactory := github.NewClientFactory(dsClient, encryptor, oauthCfg, logger)
		tokenPool := github.NewTokenPool(clientFactory, dsClient, logger)
		logger.Info("oauth login enabled")

		// Create router with all dependencies.
		r := api.NewRouterWithDeps(api.RouterDeps{
			DS:            dsClient,
			Logger:        logger,
			Config:        cfg,
			Encryptor:     encryptor,
			OAuthConfig:   oauthCfg,
			JWTSecret:     jwtSecret,
			ClientFactory: clientFactory,
			TokenPool:     tokenPool,
		})
		router = r.Handler()
	})
}

// RunHTTPServer is the Cloud Functions entry point.
func RunHTTPServer(w http.ResponseWriter, r *http.Request) {
	Init()
	router.ServeHTTP(w, r)
}
