package main

import (
	"catgoose/go-htmx-demo/internals/config"
	// setup:feature:database:start
	"catgoose/go-htmx-demo/internals/database"
	"catgoose/go-htmx-demo/internals/database/dialect"
	dbrepo "catgoose/go-htmx-demo/internals/database/repository"
	// setup:feature:database:end
	log "catgoose/go-htmx-demo/internals/logger"
	"catgoose/go-htmx-demo/internals/routes"
	// setup:feature:avatar:start
	graphdb "catgoose/go-htmx-demo/internals/database"
	"catgoose/go-htmx-demo/internals/domain"
	"catgoose/go-htmx-demo/internals/service/graph"
	// setup:feature:avatar:end
	"context"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/catgoose/dio"
)

//go:embed web/assets/public/css/* web/assets/public/js/*
var staticAssets embed.FS

var staticFS = must(fs.Sub(staticAssets, "web/assets/public"))

func must(fs fs.FS, err error) fs.FS {
	if err != nil {
		panic(err)
	}
	return fs
}

func main() {
	log.Init()
	flag.Parse()
	envErr := dio.InitEnvironment(nil)
	// setup:feature:demo:start
	if envErr != nil {
		// No .env file -- apply standalone defaults so the demo binary
		// can run without any configuration.
		if os.Getenv("SERVER_LISTEN_PORT") == "" {
			os.Setenv("SERVER_LISTEN_PORT", "8080")
		}
		log.Info("No .env file found, using environment variables and defaults")
		envErr = nil
	}
	// setup:feature:demo:end
	if envErr != nil {
		log.Fatal("Failed to initialize environment", "error", envErr)
	}

	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatal("Failed to load configuration", "error", err)
	}

	appCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// setup:feature:database:start
	if cfg.EnableDatabase {
		db, err := database.Open(appCtx, cfg.DBEngine)
		if err != nil {
			log.Fatal("Failed to open database", "error", err)
		}
		defer func() {
			if closeErr := db.Close(); closeErr != nil {
				log.Info("Error closing database connection", "error", closeErr)
			}
		}()

		d, err := dialect.New(cfg.DBEngine)
		if err != nil {
			log.Fatal("Failed to create dialect", "error", err)
		}

		repoManager := dbrepo.NewManager(db, d)

		// InitRepo gates schema init. Destructive: drops existing tables and recreates them, wiping data. Only enable when intentionally resetting the database.
		if cfg.InitRepo {
			if err := repoManager.InitSchema(appCtx); err != nil {
				log.Fatal("Failed to initialize database schema", "error", err)
			}
		}
	}
	// setup:feature:database:end

	e, err := routes.InitEcho(appCtx, staticFS, cfg)
	if err != nil {
		log.Fatal("Failed to initialize Echo", "error", err)
	}

	ar := routes.NewAppRoutes(appCtx, e)
	if err := ar.InitRoutes(); err != nil {
		log.Fatal("Failed to setup routes", "error", err)
	}

	// setup:feature:avatar:start
	photoStore, err := graph.NewPhotoStore("web/assets/public/images")
	if err != nil {
		log.Fatal("Failed to create photo store", "error", err)
	}
	routes.RegisterAvatarRoutes(e, photoStore)

	tenantID, _ := dio.Env("AZURE_TENANT_ID")
	clientID, _ := dio.Env("AZURE_CLIENT_ID")
	clientSecret, _ := dio.Env("AZURE_CLIENT_SECRET")
	if tenantID != "" && clientID != "" && clientSecret != "" {
		graphClient, err := graph.NewGraphClient(tenantID, clientID, clientSecret)
		if err != nil {
			log.Fatal("Failed to create Graph client", "error", err)
		}
		sqliteDB, err := graphdb.OpenSQLiteInMemory()
		if err != nil {
			log.Fatal("Failed to open in-memory SQLite for user cache", "error", err)
		}
		defer sqliteDB.Close()
		userCache := graph.NewUserCache(sqliteDB)
		afterSync := func(ctx context.Context, users []domain.GraphUser) {
			if err := graph.SyncPhotos(ctx, graphClient, photoStore, users, false); err != nil {
				log.Error("Photo sync failed", "error", err)
			}
		}
		if err := graph.InitAndSyncUserCache(appCtx, userCache, cfg.AzureRefreshUsersHour, graphClient.FetchAllEnabledUsers, afterSync); err != nil {
			log.Fatal("Failed to initialize user cache", "error", err)
		}
	} else {
		log.Info("Azure credentials not configured; skipping user and photo sync")
	}
	// setup:feature:avatar:end

	go func() {
		if dio.Dev() {
			log.Info("Starting Echo server with TLS (development mode)", "port", cfg.ServerPort)
			if err := e.StartTLS(fmt.Sprintf(":%s", cfg.ServerPort), "localhost.crt", "localhost.key"); err != nil {
				if err != http.ErrServerClosed {
					log.Fatal("Failed to start Echo server with TLS", "error", err)
				}
			}
		} else {
			log.Info("Starting Echo server without TLS (production mode)", "port", cfg.ServerPort)
			if err := e.Start(fmt.Sprintf(":%s", cfg.ServerPort)); err != nil {
				if err != http.ErrServerClosed {
					log.Fatal("Failed to start Echo server", "error", err)
				}
			}
		}
	}()

	// Handle graceful shutdown (waiting for termination signal)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for shutdown signal
	<-sigChan
	log.Info("Shutting down gracefully...")

	// Cancel the application context to signal shutdown to all goroutines
	cancel()

	// Create a timeout context for graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Shutdown the Echo server
	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Info("Error during server shutdown", "error", err)
	}

	log.Info("Server shutdown complete")
}
