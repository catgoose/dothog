package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"catgoose/dothog/internal/config"
	"catgoose/dothog/internal/database"
	// setup:feature:database:start
	"catgoose/dothog/internal/dbschema"

	dialect "github.com/catgoose/chuck"
	// setup:feature:database:end

	// Framework SQLite stores (error traces, session settings, graph user cache)
	// always need the sqlite driver registered.
	_ "github.com/catgoose/chuck/driver/sqlite"
	// setup:feature:mssql:start
	_ "github.com/catgoose/chuck/driver/mssql"
	// setup:feature:mssql:end
	// setup:feature:postgres:start
	_ "github.com/catgoose/chuck/driver/postgres"
	// setup:feature:postgres:end

	// schema owns the chuck-backed schema materializer (schema.NewMaterializer runs
	// Init/Ensure/Validate). Repositories hold *sqlx.DB directly and reach
	// transactions via database.WithTransaction, so neither side leans on a
	// catch-all "manager as runtime access point" object.
	"catgoose/dothog/internal/database/schema"

	"catgoose/dothog/internal/logger"
	"catgoose/dothog/internal/routes"
	// setup:feature:database:start
	"github.com/jmoiron/sqlx"
	// setup:feature:database:end

	"github.com/catgoose/promolog"
	promologsqlite "github.com/catgoose/promolog/sqlite"

	// setup:feature:session_settings:start
	"catgoose/dothog/internal/repository"
	"catgoose/dothog/internal/session"
	// setup:feature:session_settings:end
	// setup:feature:graph:start
	"catgoose/dothog/internal/service/graph"

	// setup:feature:graph:end

	appenv "catgoose/dothog/internal/env"
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
	envErr := appenv.Init()
	// setup:feature:demo:start
	if envErr != nil {
		_ = os.Setenv("SERVER_LISTEN_PORT", appenv.GetDefault("SERVER_LISTEN_PORT", "3000"))
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

	// Error trace store
	traceDB, err := database.OpenSQLite(appCtx, "db/error_traces.db")
	if err != nil {
		logger.Fatal("Failed to open error traces database", "error", err)
	}
	defer func() {
		if closeErr := traceDB.Close(); closeErr != nil {
			logger.Info("Error closing error traces database", "error", closeErr)
		}
	}()
	reqLogStore := promologsqlite.NewStore(traceDB.DB)
	if err := reqLogStore.EnsureSchema(); err != nil {
		logger.Fatal("Failed to ensure error traces schema", "error", err)
	}
	reqLogStore.StartCleanup(appCtx, 90*24*time.Hour, 1*time.Hour)
	// setup:feature:demo:start
	routes.SeedErrorTraces(reqLogStore)
	// setup:feature:demo:end

	// setup:feature:database:start
	// App-data store: chuck-backed repository over cfg.DatabaseURL when set.
	// Framework-owned SQLite stores (session settings, error traces, graph user
	// cache) have their own openers below and are not registered here — keeping
	// the app-data layer isolated from internal runtime concerns.
	if cfg.DatabaseURL != "" {
		db, d, err := dialect.OpenURL(appCtx, cfg.DatabaseURL)
		if err != nil {
			logger.Fatal("Failed to open app database", "error", err)
		}
		defer func() {
			if closeErr := db.Close(); closeErr != nil {
				logger.Info("Error closing app database connection", "error", closeErr)
			}
		}()

		dbx := sqlx.NewDb(db, string(d.Engine()))
		// App-data schema lifecycle: tables live in internal/dbschema and flow
		// in via dbschema.Tables(). Concrete repositories take *sqlx.DB (dbx)
		// directly and use database.WithTransaction for multi-statement work.
		// Framework-owned tables (session settings, graph user cache, error
		// traces) are wired separately below and are not registered here.
		appSchema := schema.NewMaterializer(dbx, d, dbschema.Tables()...)

		// Derived apps that need a destructive schema reset call
		// appSchema.InitSchema(ctx) here behind their own gating; the template
		// never wires that path through config.
		if err := appSchema.EnsureSchema(appCtx); err != nil {
			logger.Fatal("Failed to ensure app database schema", "error", err)
		}

		if err := appSchema.ValidateSchema(appCtx); err != nil {
			logger.Fatal("App database schema validation failed", "error", err)
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
	settingsDialect, err := database.SQLiteDialect()
	if err != nil {
		logger.Fatal("Failed to create session settings dialect", "error", err)
	}
	settingsSchema := schema.NewMaterializer(settingsDB, settingsDialect, session.SettingsTable)
	if err := settingsSchema.EnsureSchema(appCtx); err != nil {
		logger.Fatal("Failed to ensure session settings schema", "error", err)
	}
	settingsRepo := repository.NewSessionSettingsRepository(settingsDB)
	// setup:feature:session_settings:end

	// InitEcho builds the shared Echo instance: static file serving, global
	// middleware, auth/session wiring, and the HTTP error handler. It does not
	// register the app's route table.
	e, err := routes.InitEcho(
		appCtx, staticFS, cfg,
		// setup:feature:session_settings:start
		settingsRepo,
		// setup:feature:session_settings:end
		reqLogStore,
	)
	if err != nil {
		logger.Fatal("Failed to initialize Echo", "error", err)
	}

	// NewAppRoutes owns the route layer on top of Echo: it receives the app's
	// stores/runtime dependencies and later registers endpoints via InitRoutes.
	ar := routes.NewAppRoutes(appCtx, e, routes.Repos{
		ReqLogStore: reqLogStore,
		// setup:feature:session_settings:start
		SessionSettings: settingsRepo,
		SessionStore:    settingsRepo,
		// setup:feature:session_settings:end
	})
	defer ar.Close()
	if err := ar.InitRoutes(); err != nil {
		logger.Fatal("Failed to setup routes", "error", err)
	}

	// setup:feature:graph:start
	// Graph-owned persistent directory cache: one on-disk SQLite file holds
	// the user directory snapshot and the avatar blobs together, so derived
	// apps can serve the last snapshot immediately on restart while the next
	// sync runs.
	directory, err := graph.OpenDirectory(appCtx, "db/graph_cache.db")
	if err != nil {
		logger.Fatal("Failed to open Graph directory", "error", err)
	}
	defer func() { _ = directory.Close() }()
	// setup:feature:avatar:start
	photoCache := directory.Photos()
	routes.RegisterAvatarRoutes(e, photoCache)
	// setup:feature:avatar:end

	tenantID, _ := appenv.Get("AZURE_TENANT_ID")
	clientID, _ := appenv.Get("AZURE_CLIENT_ID")
	clientSecret, _ := appenv.Get("AZURE_CLIENT_SECRET")
	if tenantID != "" && clientID != "" && clientSecret != "" {
		graphClient, err := graph.NewGraphClient(tenantID, clientID, clientSecret)
		if err != nil {
			logger.Fatal("Failed to create Graph client", "error", err)
		}
		// afterSync is nil when avatar is stripped; the gated assignment below
		// hooks photo sync in only when the avatar feature is kept. The split
		// declaration is intentional — setup:feature:avatar strips the
		// assignment but leaves the declaration so graph-without-avatar
		// scaffolds still compile.
		var afterSync func(ctx context.Context, users []graph.GraphUser) //nolint:staticcheck // S1021: split intentional, see comment above.
		// setup:feature:avatar:start
		afterSync = func(ctx context.Context, users []graph.GraphUser) {
			if err := graph.SyncPhotos(ctx, graphClient, photoCache, users, false); err != nil {
				logger.Error("Photo sync failed", "error", err)
			}
		}
		// setup:feature:avatar:end
		if err := graph.InitAndSyncDirectory(appCtx, directory, cfg.GraphUserCacheRefreshHour, graphClient.FetchAllEnabledUsers, afterSync); err != nil {
			logger.Fatal("Failed to initialize Graph directory", "error", err)
		}
	}
	// setup:feature:graph:end

	go func() {
		logger.Info("Starting Echo server", "port", cfg.ServerPort, "dev", appenv.Dev())
		if err := e.Start(fmt.Sprintf(":%s", cfg.ServerPort)); err != nil {
			if err != http.ErrServerClosed {
				logger.Fatal("Failed to start Echo server", "error", err)
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
