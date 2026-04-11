package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"

	"github.com/ok2ju/oversite/backend/internal/auth"
	"github.com/ok2ju/oversite/backend/internal/config"
	"github.com/ok2ju/oversite/backend/internal/faceit"
	"github.com/ok2ju/oversite/backend/internal/handler"
	"github.com/ok2ju/oversite/backend/internal/storage"
	"github.com/ok2ju/oversite/backend/internal/store"
	ws "github.com/ok2ju/oversite/backend/internal/websocket"
	"github.com/ok2ju/oversite/backend/internal/worker"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "oversite",
		Short: "Oversite CS2 demo viewer backend",
	}

	rootCmd.AddCommand(
		serveCmd(),
		wsCmd(),
		workerCmd(),
		migrateCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func serveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the API server",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			setupLogger(cfg.LogLevel)

			// Database
			db, err := sql.Open("postgres", cfg.DatabaseURL)
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}
			defer func() { _ = db.Close() }()

			// Redis
			redisOpts, err := redis.ParseURL(cfg.RedisURL)
			if err != nil {
				return fmt.Errorf("parsing redis URL: %w", err)
			}
			redisClient := redis.NewClient(redisOpts)
			defer func() { _ = redisClient.Close() }()

			// Domain services
			queries := store.New(db)

			// OAuth
			oauthCfg := auth.FaceitOAuthConfig{
				ClientID:     cfg.FaceitClientID,
				ClientSecret: cfg.FaceitClientSecret,
				RedirectURI:  cfg.FaceitRedirectURI,
				AuthURL:      "https://accounts.faceit.com/accounts",
				TokenURL:     "https://api.faceit.com/auth/v1/oauth/token",
				UserInfoURL:  "https://api.faceit.com/auth/v1/resources/userinfo",
			}

			stateStore := auth.NewRedisStateStore(redisClient)
			sessionStore := auth.NewRedisSessionStore(redisClient)
			faceitClient := auth.NewFaceitClient(oauthCfg)
			oauthSvc := auth.NewOAuthService(oauthCfg, stateStore, queries, faceitClient)
			secure := cfg.Environment == "production"
			queue := worker.NewRedisQueue(redisClient)
			authHandler := handler.NewAuthHandler(oauthSvc, sessionStore, secure, queue)

			// MinIO object storage
			minioClient, err := storage.NewMinIOClient(
				cfg.MinioEndpoint, cfg.MinioAccessKey, cfg.MinioSecretKey, cfg.MinioUseSSL,
			)
			if err != nil {
				return fmt.Errorf("creating minio client: %w", err)
			}
			if err := minioClient.EnsureBucket(context.Background(), cfg.MinioBucket); err != nil {
				return fmt.Errorf("ensuring minio bucket: %w", err)
			}

			demoHandler := handler.NewDemoHandler(queries, minioClient, queue, cfg.MinioBucket)
			demoImporter := faceit.NewDemoImporter(queries, minioClient, queue, &http.Client{Timeout: 5 * time.Minute}, cfg.MinioBucket)
			faceitHandler := handler.NewFaceitHandler(queue, queries, demoImporter)
			tickHandler := handler.NewTickHandler(queries, queries)
			rosterHandler := handler.NewRosterHandler(queries, queries)
			eventHandler := handler.NewEventHandler(queries, queries)
			heatmapHandler := handler.NewHeatmapHandler(queries, queries)
			roundHandler := handler.NewRoundHandler(queries, queries)

			// Health checks with real dependencies
			health := handler.NewHealthHandler(
				&handler.DBChecker{DB: db},
				stateStore,
				&handler.MinIOChecker{Endpoint: cfg.MinioEndpoint, UseSSL: cfg.MinioUseSSL},
			)
			router := handler.NewRouter(health, authHandler, demoHandler, faceitHandler, tickHandler, rosterHandler, eventHandler, heatmapHandler, roundHandler, sessionStore)

			slog.Info("starting API server", "port", cfg.Port, "env", cfg.Environment)
			return http.ListenAndServe(":"+cfg.Port, router)
		},
	}
}

func wsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ws",
		Short: "Start the WebSocket server",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadWS()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			setupLogger(cfg.LogLevel)

			// Database
			db, err := sql.Open("postgres", cfg.DatabaseURL)
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}
			defer func() { _ = db.Close() }()

			// Redis
			redisOpts, err := redis.ParseURL(cfg.RedisURL)
			if err != nil {
				return fmt.Errorf("parsing redis URL: %w", err)
			}
			redisClient := redis.NewClient(redisOpts)
			defer func() { _ = redisClient.Close() }()

			sessionStore := auth.NewRedisSessionStore(redisClient)

			// Yjs relay with DB-backed state persistence.
			queries := store.New(db)
			stateStore := ws.NewPgStateStore(queries)
			relay := ws.NewYjsRelay(stateStore, cfg.YjsAutoSaveInterval)

			hub := ws.NewHubWithRelay(relay)
			go hub.Run()

			server := ws.NewServer(hub, sessionStore)

			// Health checks — WS server needs Redis and DB.
			health := handler.NewHealthHandler(&handler.DBChecker{DB: db}, &handler.RedisChecker{Client: redisClient}, nil)
			router := server.Router(health)

			slog.Info("starting WebSocket server", "port", cfg.WSPort, "env", cfg.Environment)
			return http.ListenAndServe(":"+cfg.WSPort, router)
		},
	}
}

func workerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "worker",
		Short: "Start the background worker",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadWorker()
			if err != nil {
				return fmt.Errorf("loading worker config: %w", err)
			}

			setupLogger(cfg.LogLevel)

			// Database
			db, err := sql.Open("postgres", cfg.DatabaseURL)
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}
			defer func() { _ = db.Close() }()

			// Redis
			redisOpts, err := redis.ParseURL(cfg.RedisURL)
			if err != nil {
				return fmt.Errorf("parsing redis URL: %w", err)
			}
			redisClient := redis.NewClient(redisOpts)
			defer func() { _ = redisClient.Close() }()

			// Faceit API client
			faceitAPI := faceit.NewClient(&http.Client{}, redisClient, faceit.ClientConfig{
				APIKey:  cfg.FaceitAPIKey,
				BaseURL: cfg.FaceitAPIBaseURL,
			})

			// MinIO object storage
			minioClient, err := storage.NewMinIOClient(
				cfg.MinioEndpoint, cfg.MinioAccessKey, cfg.MinioSecretKey, cfg.MinioUseSSL,
			)
			if err != nil {
				return fmt.Errorf("creating minio client: %w", err)
			}
			if err := minioClient.EnsureBucket(context.Background(), cfg.MinioBucket); err != nil {
				return fmt.Errorf("ensuring minio bucket: %w", err)
			}

			// Services
			queries := store.New(db)
			queue := worker.NewRedisQueueWithTimeout(redisClient, cfg.WorkerBlockTimeout)
			syncSvc := faceit.NewSyncService(faceitAPI, queries)
			if cfg.FaceitAutoImport {
				demoImporter := faceit.NewDemoImporter(queries, minioClient, queue, &http.Client{Timeout: 5 * time.Minute}, cfg.MinioBucket)
				syncSvc = syncSvc.WithAutoImport(demoImporter)
			}

			// Worker
			faceitHandler := worker.NewFaceitSyncHandler(syncSvc)
			w := worker.NewWorker(queue, worker.FaceitSyncStream, "workers", "worker-1", faceitHandler).
				WithMaxRetry(cfg.WorkerMaxRetry).
				WithStaleThreshold(cfg.WorkerStaleThreshold).
				WithClaimInterval(cfg.WorkerClaimInterval)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if err := w.Start(ctx); err != nil {
				return fmt.Errorf("starting worker: %w", err)
			}

			slog.Info("worker running", "stream", worker.FaceitSyncStream, "env", cfg.Environment)

			// Wait for shutdown signal
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			<-sigCh

			slog.Info("shutdown signal received, stopping worker...")
			w.Stop()
			return nil
		},
	}
}

func migrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Run database migrations",
	}

	var migrationsPath string
	cmd.PersistentFlags().StringVar(&migrationsPath, "path", "", "path to migrations directory (default: auto-detected relative to binary)")

	cmd.AddCommand(
		&cobra.Command{
			Use:   "up",
			Short: "Run all pending migrations",
			RunE: func(cmd *cobra.Command, args []string) error {
				m, err := newMigrate(migrationsPath)
				if err != nil {
					return err
				}
				defer func() { _, _ = m.Close() }()

				slog.Info("running migrations up")
				if err := m.Up(); err != nil {
					if errors.Is(err, migrate.ErrNoChange) {
						slog.Info("no new migrations to apply")
						return nil
					}
					return fmt.Errorf("migrate up: %w", err)
				}
				slog.Info("migrations applied successfully")
				return nil
			},
		},
		&cobra.Command{
			Use:   "down",
			Short: "Rollback the last migration",
			RunE: func(cmd *cobra.Command, args []string) error {
				m, err := newMigrate(migrationsPath)
				if err != nil {
					return err
				}
				defer func() { _, _ = m.Close() }()

				slog.Info("rolling back last migration")
				if err := m.Steps(-1); err != nil {
					if errors.Is(err, migrate.ErrNoChange) {
						slog.Info("no migrations to rollback")
						return nil
					}
					return fmt.Errorf("migrate down: %w", err)
				}
				slog.Info("migration rolled back successfully")
				return nil
			},
		},
	)
	return cmd
}

// newMigrate creates a golang-migrate instance configured with DATABASE_URL
// and the migrations directory. It reads DATABASE_URL directly from the
// environment so that running migrations doesn't require all other env vars
// (Redis, MinIO, etc.) to be set.
func newMigrate(migrationsPath string) (*migrate.Migrate, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required")
	}

	sourceURL, err := resolveMigrationsSource(migrationsPath)
	if err != nil {
		return nil, err
	}

	m, err := migrate.New(sourceURL, dbURL)
	if err != nil {
		return nil, fmt.Errorf("creating migrate instance: %w", err)
	}

	m.Log = &slogMigrateLogger{}
	return m, nil
}

// resolveMigrationsSource determines the file:// source URL for migrations.
// If an explicit path is provided via --path flag, it uses that.
// Otherwise, it looks for a migrations/ directory relative to the binary.
func resolveMigrationsSource(explicit string) (string, error) {
	if explicit != "" {
		abs, err := filepath.Abs(explicit)
		if err != nil {
			return "", fmt.Errorf("resolving migrations path: %w", err)
		}
		return "file://" + abs, nil
	}

	// Default: migrations/ directory relative to the working directory.
	// This works when running from the backend/ directory or via Docker.
	abs, err := filepath.Abs("migrations")
	if err != nil {
		return "", fmt.Errorf("resolving default migrations path: %w", err)
	}
	return "file://" + abs, nil
}

// slogMigrateLogger adapts slog to the migrate.Logger interface.
type slogMigrateLogger struct{}

func (l *slogMigrateLogger) Printf(format string, v ...interface{}) {
	slog.Info(fmt.Sprintf(format, v...))
}

func (l *slogMigrateLogger) Verbose() bool {
	return false
}

func setupLogger(level string) {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: logLevel}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, opts))
	slog.SetDefault(logger)
}
