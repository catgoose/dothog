package main

import (
	"catgoose/dothog/internal/config"
	dialect "github.com/catgoose/fraggle"
	_ "github.com/catgoose/fraggle/driver/postgres"
	_ "github.com/catgoose/fraggle/driver/sqlite"
	// setup:feature:session_settings:start
	"catgoose/dothog/internal/database"
	// setup:feature:session_settings:end
	// setup:feature:database:start
	dbrepo "catgoose/dothog/internal/database/repository"
	"catgoose/dothog/internal/database/schema"
	"github.com/jmoiron/sqlx"
	// setup:feature:database:end
	"catgoose/dothog/internal/logger"
	"github.com/catgoose/promolog"
	"catgoose/dothog/internal/routes"
	// setup:feature:session_settings:start
	"catgoose/dothog/internal/repository"
	// setup:feature:session_settings:end
	// setup:feature:avatar:start
	graphdb "catgoose/dothog/internal/database"
	"catgoose/dothog/internal/domain"
	"catgoose/dothog/internal/service/graph"
	// setup:feature:avatar:end
	"context"
	"embed"
	"flag"
	"log/slog"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/catgoose/dio"
)

//go:embed web/assets/public/css/* web/assets/public/js/* all:web/assets/public/images
var staticAssets embed.FS

var staticFS = must(fs.Sub(staticAssets, "web/assets/public"))

func must(fs fs.FS, err error) fs.FS {
	if err != nil {
		panic(err)
	}
	return fs
}

func main() {
	logger.SetHandlerWrapper(func(h slog.Handler) slog.Handler {
		return promolog.NewHandler(h)
	})
	logger.Init()
	flag.Parse()
	dioOpts := &dio.Options{}
	if env := os.Getenv("ENV"); env != "" {
		dioOpts.Env = env
	}
	envErr := dio.InitEnvironment(dioOpts)
	// setup:feature:demo:start
	if envErr != nil {
		// No .env file -- apply standalone defaults so the demo binary
		// can run without any configuration.
		os.Setenv("SERVER_LISTEN_PORT", dio.EnvWithDefault("SERVER_LISTEN_PORT", "3000"))
	}
	// setup:feature:demo:end
	if envErr != nil {
		logger.Info("Environment file not found (using OS env vars)", "error", envErr)
	}

	cfg, err := config.GetConfig()
	if err != nil {
		logger.Fatal("Failed to load configuration", "error", err)
	}

	appCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Error trace store — persists error request logs to SQLite for debugging.
	traceDB, _, err := dialect.OpenSQLite(appCtx, "db/error_traces.db")
	if err != nil {
		logger.Fatal("Failed to open error traces database", "error", err)
	}
	defer func() {
		if closeErr := traceDB.Close(); closeErr != nil {
			logger.Info("Error closing error traces database", "error", closeErr)
		}
	}()
	reqLogStore := promolog.NewStore(traceDB)
	if err := reqLogStore.InitSchema(); err != nil {
		logger.Fatal("Failed to init error traces schema", "error", err)
	}
	reqLogStore.StartCleanup(appCtx, 90*24*time.Hour, 1*time.Hour)
	// setup:feature:demo:start
	routes.SeedErrorTraces(reqLogStore)
	// setup:feature:demo:end

	// setup:feature:database:start
	if cfg.EnableDatabase {
		db, d, err := dialect.OpenURL(appCtx, cfg.DatabaseURL)
		if err != nil {
			logger.Fatal("Failed to open database", "error", err)
		}
		defer func() {
			if closeErr := db.Close(); closeErr != nil {
				logger.Info("Error closing database connection", "error", closeErr)
			}
		}()

		dbx := sqlx.NewDb(db, string(d.Engine()))
		repoManager := dbrepo.NewManager(dbx, d,
			// setup:feature:session_settings:start
			schema.SessionSettingsTable,
			// setup:feature:session_settings:end
			// setup:feature:graph:start
			schema.UsersTable,
			// setup:feature:graph:end
		)

		// InitRepo gates schema init. Destructive: drops existing tables and recreates them, wiping data. Only enable when intentionally resetting the database.
		if cfg.InitRepo {
			if err := repoManager.InitSchema(appCtx); err != nil {
				logger.Fatal("Failed to initialize database schema", "error", err)
			}
		}

		if err := repoManager.EnsureSchema(appCtx); err != nil {
			logger.Fatal("Failed to ensure database schema", "error", err)
		}

		if err := repoManager.ValidateSchema(appCtx); err != nil {
			logger.Fatal("Database schema validation failed", "error", err)
		}

	}
	// setup:feature:database:end

	// setup:feature:session_settings:start
	settingsDB, err := database.OpenSQLite(appCtx, "db/session_settings.db")
	if err != nil {
		logger.Fatal("Failed to open session settings database", "error", err)
	}
	defer func() {
		if closeErr := settingsDB.Close(); closeErr != nil {
			logger.Info("Error closing session settings database", "error", closeErr)
		}
	}()
	settingsDialect, err := dialect.New(dialect.SQLite)
	if err != nil {
		logger.Fatal("Failed to create session settings dialect", "error", err)
	}
	settingsManager := dbrepo.NewManager(settingsDB, settingsDialect, schema.SessionSettingsTable)
	if err := settingsManager.EnsureSchema(appCtx); err != nil {
		logger.Fatal("Failed to ensure session settings schema", "error", err)
	}
	settingsRepo := repository.NewSessionSettingsRepository(settingsManager)
	// setup:feature:session_settings:end

	e, err := routes.InitEcho(appCtx, staticFS, cfg,
		// setup:feature:session_settings:start
		settingsRepo,
		// setup:feature:session_settings:end
		reqLogStore,
	)
	if err != nil {
		logger.Fatal("Failed to initialize Echo", "error", err)
	}

	ar := routes.NewAppRoutes(appCtx, e, reqLogStore, nil,
		// setup:feature:session_settings:start
		settingsRepo,
		// setup:feature:session_settings:end
	)
	if err := ar.InitRoutes(); err != nil {
		logger.Fatal("Failed to setup routes", "error", err)
	}

	// setup:feature:avatar:start
	photoStore, err := graph.NewPhotoStore("web/assets/public/images")
	if err != nil {
		logger.Fatal("Failed to create photo store", "error", err)
	}
	routes.RegisterAvatarRoutes(e, photoStore)

	tenantID, _ := dio.Env("AZURE_TENANT_ID")
	clientID, _ := dio.Env("AZURE_CLIENT_ID")
	clientSecret, _ := dio.Env("AZURE_CLIENT_SECRET")
	if tenantID != "" && clientID != "" && clientSecret != "" {
		graphClient, err := graph.NewGraphClient(tenantID, clientID, clientSecret)
		if err != nil {
			logger.Fatal("Failed to create Graph client", "error", err)
		}
		sqliteDB, err := graphdb.OpenSQLiteInMemory()
		if err != nil {
			logger.Fatal("Failed to open in-memory SQLite for user cache", "error", err)
		}
		defer func() { _ = sqliteDB.Close() }()
		userCache := graph.NewUserCache(sqliteDB)
		afterSync := func(ctx context.Context, users []domain.GraphUser) {
			if err := graph.SyncPhotos(ctx, graphClient, photoStore, users, false); err != nil {
				logger.Error("Photo sync failed", "error", err)
			}
		}
		if err := graph.InitAndSyncUserCache(appCtx, userCache, cfg.GraphUserCacheRefreshHour, graphClient.FetchAllEnabledUsers, afterSync); err != nil {
			logger.Fatal("Failed to initialize user cache", "error", err)
		}
	} else {
		logger.Info("Azure credentials not configured; skipping user and photo sync")
	}
	// setup:feature:avatar:end

	go func() {
		if dio.Dev() {
			logger.Info("Starting Echo server with TLS (development mode)", "port", cfg.ServerPort)
			if err := e.StartTLS(fmt.Sprintf(":%s", cfg.ServerPort), "localhost.crt", "localhost.key"); err != nil {
				if err != http.ErrServerClosed {
					logger.Fatal("Failed to start Echo server with TLS", "error", err)
				}
			}
		} else {
			logger.Info("Starting Echo server without TLS (production mode)", "port", cfg.ServerPort)
			if err := e.Start(fmt.Sprintf(":%s", cfg.ServerPort)); err != nil {
				if err != http.ErrServerClosed {
					logger.Fatal("Failed to start Echo server", "error", err)
				}
			}
		}
	}()

	// Handle graceful shutdown (waiting for termination signal)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for shutdown signal
	<-sigChan
	logger.Info("Shutting down gracefully...")

	// Cancel the application context to signal shutdown to all goroutines
	cancel()

	// Create a timeout context for graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Shutdown the Echo server
	if err := e.Shutdown(shutdownCtx); err != nil {
		logger.Info("Error during server shutdown", "error", err)
	}

	logger.Info("Server shutdown complete")
}
