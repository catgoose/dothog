// Package routes provides routing configuration and Echo server initialization.
package routes

import (
	"catgoose/go-htmx-demo/internals/config"
	// setup:feature:demo:start
	"catgoose/go-htmx-demo/internals/demo"
	log "catgoose/go-htmx-demo/internals/logger"
	// setup:feature:sse:start
	"catgoose/go-htmx-demo/internals/ssebroker"
	// setup:feature:sse:end
	// setup:feature:demo:end
	"catgoose/go-htmx-demo/internals/routes/handler"
	"catgoose/go-htmx-demo/web/views"
	"catgoose/go-htmx-demo/internals/routes/middleware"
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"time"
	// setup:feature:auth:start
	"github.com/catgoose/crooner"
	// setup:feature:auth:end
	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
)

// AppRoutes defines the interface for app routes
type AppRoutes interface {
	InitRoutes() error
}

// appRoutes implements AppRoutes
type appRoutes struct {
	e   *echo.Echo
	ctx context.Context
}

// NewAppRoutes initializes routes
func NewAppRoutes(ctx context.Context, e *echo.Echo) AppRoutes {
	return &appRoutes{
		e:   e,
		ctx: ctx,
	}
}

func (ar *appRoutes) InitRoutes() error {
	ar.e.GET("/", handler.HandleComponent(views.HomePage()))

	// Health check endpoint for Caddy
	ar.e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// setup:feature:demo:start
	ar.initControlsGalleryRoutes()
	ar.initComponentsRoutes()
	ar.initComponents2Routes()
	ar.initComponents3Routes()
	// setup:feature:demo:end

	// setup:feature:demo:start
	// setup:feature:sse:start
	broker := ssebroker.NewSSEBroker()
	// setup:feature:sse:end
	ar.initHypermediaRoutes()
	// setup:feature:sse:start
	ar.initRealtimeRoutes(broker)
	// setup:feature:sse:end

	db, err := demo.Open("demo.db")
	if err != nil {
		log.Warn("Demo DB unavailable; /tables/* routes disabled", "error", err)
		return nil
	}
	ar.initInventoryRoutes(db)
	ar.initCatalogRoutes(db)
	ar.initBulkRoutes(db)

	adminDB, err := demo.OpenMemoryDB()
	if err != nil {
		log.Warn("Admin in-memory DB unavailable; /admin routes disabled", "error", err)
	} else {
		ar.initAdminRoutes(adminDB)
	}
	// setup:feature:demo:end
	return nil
}

// InitEcho initializes Echo with global configurations
func InitEcho(ctx context.Context, staticFS fs.FS, cfg *config.AppConfig) (*echo.Echo, error) {
	e := echo.New()

	e.Use(middleware.RequestIDMiddleware())
	e.Use(echoMiddleware.RequestLogger())
	e.Use(echoMiddleware.Recover())

	// setup:feature:auth:start
	if cfg != nil && !cfg.CroonerDisabled && cfg.CroonerConfig != nil {
		sessionMgr, scsMgr, err := crooner.NewSCSManager(
			crooner.WithPersistentCookieName(cfg.SessionSecret, cfg.AppName),
			crooner.WithLifetime(24*time.Hour),
		)
		if err != nil {
			return nil, fmt.Errorf("crooner session manager: %w", err)
		}
		e.Use(echo.WrapMiddleware(scsMgr.LoadAndSave))
		cfg.SessionMgr = sessionMgr
		cfg.CroonerConfig.SessionMgr = sessionMgr
		if err := crooner.NewAuthConfig(ctx, e, cfg.CroonerConfig); err != nil {
			return nil, fmt.Errorf("crooner auth config: %w", err)
		}
		// setup:feature:csrf:start
		if cfg.SessionMgr != nil {
			e.Use(middleware.CSRF(cfg.SessionMgr, middleware.CSRFConfig{
				RotatePerRequest: cfg.CSRFRotatePerRequest,
				PerRequestPaths:  cfg.CSRFPerRequestPaths,
				ExemptPaths:      cfg.CSRFExemptPaths,
			}))
		}
		// setup:feature:csrf:end
	}
	// setup:feature:auth:end

	e.Use(middleware.ErrorHandlerMiddleware())

	e.StaticFS("/public", staticFS)

	return e, nil
}
