// Package routes provides routing configuration and Echo server initialization.
package routes

import (
	"catgoose/dothog/internal/config"
	"catgoose/dothog/internal/logger"
	// setup:feature:demo:start
	"catgoose/dothog/internal/demo"
	// setup:feature:demo:end
	// setup:feature:sse:start
	ssebroker "github.com/catgoose/tavern"
	// setup:feature:sse:end
	"catgoose/dothog/internal/health"
	"catgoose/dothog/internal/version"
	"github.com/catgoose/promolog"
	"catgoose/dothog/internal/routes/handler"
	hypermedia "github.com/catgoose/linkwell"
	// setup:feature:session_settings:start
	"catgoose/dothog/internal/domain"
	// setup:feature:session_settings:end
	"catgoose/dothog/web/views"
	"catgoose/dothog/internal/routes/middleware"
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"os"
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
	// SetHealthDB sets the database connection for the /health endpoint to ping.
	SetHealthDB(db health.Pinger)
	// SetHealthStats sets a function that returns app-specific stats for /health.
	SetHealthStats(fn health.StatsFunc)
}

// setup:feature:session_settings:start

// SessionSettingsStore is the subset of session-settings operations that route
// handlers need: listing all rows and upserting a single row.
type SessionSettingsStore interface {
	ListAll(ctx context.Context) ([]domain.SessionSettings, error)
	Upsert(ctx context.Context, s *domain.SessionSettings) error
}

// setup:feature:session_settings:end

// appRoutes implements AppRoutes
type appRoutes struct {
	e             *echo.Echo
	ctx           context.Context
	reqLogStore   *promolog.Store
	issueReporter IssueReporter
	startTime     time.Time
	healthCfg     health.Config
	pollCount     int64 // atomic; demo counter for SSE polling
	// setup:feature:session_settings:start
	settingsRepo SessionSettingsStore
	// setup:feature:session_settings:end
	// setup:feature:sync:start
	versionChecker VersionChecker
	// setup:feature:sync:end
	// setup:feature:demo:start
	demoDB *demo.DB
	// setup:feature:demo:end
	// setup:feature:sse:start
	broker *ssebroker.SSEBroker
	// setup:feature:sse:end
}

// NewAppRoutes initializes routes.
// reqLogStore may be nil if request log capture is disabled.
// reporter may be nil; a default no-op reporter is used.
func NewAppRoutes(ctx context.Context, e *echo.Echo, reqLogStore *promolog.Store, reporter IssueReporter,
	// setup:feature:session_settings:start
	settingsRepo SessionSettingsStore,
	// setup:feature:session_settings:end
) AppRoutes {
	if reporter == nil {
		reporter = defaultReporter{}
	}
	startTime := time.Now()
	return &appRoutes{
		e:             e,
		ctx:           ctx,
		reqLogStore:   reqLogStore,
		issueReporter: reporter,
		startTime:     startTime,
		healthCfg: health.Config{
			Version:   version.Version,
			StartTime: startTime,
		},
		// setup:feature:session_settings:start
		settingsRepo: settingsRepo,
		// setup:feature:session_settings:end
	}
}

func (ar *appRoutes) InitRoutes() error {
	// Register known origins for ?from= breadcrumb resolution.
	// Home (bit 0) is pre-registered. Additional pages register here.
	// setup:feature:demo:start
	hypermedia.RegisterFrom(hypermedia.FromDashboard, hypermedia.Breadcrumb{Label: "Dashboard", Href: "/dashboard"})
	// setup:feature:demo:end

	cfg, err := config.GetConfig()
	if err != nil {
		return fmt.Errorf("handler init: %w", err)
	}
	ar.e.GET("/", handler.HandleComponent(views.HomePage(cfg.AppName)))
	// setup:feature:demo:start
	ar.e.GET("/", handler.HandleComponent(views.ArchitecturePage()))
	// setup:feature:demo:end
	// setup:feature:demo:start
	ar.initUserSettingsRoutes()
	// setup:feature:demo:end
	// setup:feature:session_settings:start
	ar.e.GET("/settings", func(c echo.Context) error {
		s := middleware.GetSessionSettings(c)
		return handler.RenderBaseLayout(c, views.AppSettingsPage(s.Theme))
	})
	// setup:feature:session_settings:end
	// setup:feature:demo:start
	ar.initLinkRelations()
	ar.e.GET("/welcome", handler.HandleComponent(views.WelcomePage()))
	ar.e.GET("/hypermedia", handler.HandleComponent(views.PatternsIndexPage()))
	ar.e.GET("/demo", handler.HandleComponent(views.DemoIndexPage()))
	// setup:feature:demo:end

	// Health check endpoint — returns structured ops metadata.
	// HEAD is used by the offline indicator to poll connectivity.
	healthHandler := func(c echo.Context) error {
		return c.JSON(http.StatusOK, health.Check(c.Request().Context(), ar.healthCfg))
	}
	ar.e.GET("/health", healthHandler)
	ar.e.HEAD("/health", healthHandler)

	ar.initReportIssueRoutes()

	// setup:feature:sync:start
	ar.initSyncRoutes()
	// setup:feature:sync:end

	ar.initAdminCoreRoutes()
	ar.initErrorTracesRoutes()

	// setup:feature:demo:start
	ar.initPwaRoutes()
	ar.initReportDemoRoutes()
	ar.initLoggingRoutes()
	ar.initControlsGalleryRoutes()
	ar.initComponentsRoutes()
	ar.initComponents2Routes()
	ar.initComponents3Routes()
	// setup:feature:demo:end

	// setup:feature:demo:start
	// setup:feature:sse:start
	ar.broker = ssebroker.NewSSEBroker()
	// setup:feature:sse:end
	// setup:feature:session_settings:start
	ar.initThemeRoutes(ar.broker)
	// setup:feature:session_settings:end
	ar.initHypermediaRoutes()
	ar.initHALRoutes()
	ar.initErrorsRoutes()
	// setup:feature:sse:start
	ar.initRealtimeRoutes(ar.broker)
	// setup:feature:sse:end

	db, err := demo.Open("db/demo.db")
	if err != nil {
		logger.WithContext(ar.ctx).Warn("Demo DB unavailable; /demo/* routes disabled", "error", err)
		return nil
	}
	// setup:feature:sync:start
	ar.versionChecker = NewSQLVersionChecker(db.RawDB())
	// setup:feature:sync:end
	ar.demoDB = db
	if stored, err := db.ListStoredLinks(); err == nil {
		for _, s := range stored {
			hypermedia.LoadStoredLink(s.Source, hypermedia.LinkRelation{
				Rel:   s.Rel,
				Href:  s.Target,
				Title: s.Title,
				Group: s.GroupName,
			})
		}
	}
	ar.initInventoryRoutes(db)
	ar.initCatalogRoutes(db)
	ar.initBulkRoutes(db)
	ar.initRepositoryRoutes(db)

	actLog := demo.NewActivityLog(200)
	board := demo.NewKanbanBoard()
	queue := demo.NewApprovalQueue()
	ar.initAdminSettingsRoutes(ar.broker)
	ar.initAdminRoutes(db, actLog, ar.broker)
	ar.initPeopleRoutes(db, ar.broker, actLog)
	ar.initKanbanRoutes(board, actLog, ar.broker)
	ar.initApprovalRoutes(queue, actLog, ar.broker)
	ar.initFeedRoutes(actLog, ar.broker)
	ar.initCanvasRoutes(demo.NewPixelCanvas(), ar.broker)
	ar.initSettingsRoutes(demo.NewSettingsStore())
	ar.initVendorContactRoutes(db, actLog, ar.broker)
	ar.initDashboardRoutes(db, board, queue, actLog)
	ar.initAdminErrorReportsRoutes(db)

	// setup:feature:demo:end
	ar.e.RouteNotFound("/*", handler.HandleNotFound)
	handler.InitRouteSet(ar.e, cfg.AppName)
	ar.healthCfg.Name = cfg.AppName
	return nil
}

func (ar *appRoutes) SetHealthDB(db health.Pinger) {
	ar.healthCfg.DB = db
}

func (ar *appRoutes) SetHealthStats(fn health.StatsFunc) {
	ar.healthCfg.Stats = fn
}

// InitEcho initializes Echo with global configurations
func InitEcho(ctx context.Context, staticFS fs.FS, cfg *config.AppConfig,
	// setup:feature:session_settings:start
	settingsRepo middleware.SessionSettingsProvider,
	// setup:feature:session_settings:end
	reqLogStore *promolog.Store,
) (*echo.Echo, error) {
	e := echo.New()

	// Preload critical assets. In production (direct H2), send 103 Early Hints
	// so the browser fetches CSS/JS while the server generates the response.
	// Behind the dev proxy chain (TEMPL_PROXY), 1xx responses get mangled, so
	// fall back to Link headers on the final response — the browser still gets
	// the preload hint, just slightly later.
	behindProxy := os.Getenv("TEMPL_PROXY") != ""
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			preloadLinks := []string{
				"</public/css/tailwind.css>; rel=preload; as=style",
				"</public/css/daisyui.css>; rel=preload; as=style",
				"</public/js/htmx.min.js>; rel=preload; as=script",
			}
			if !behindProxy && c.Request().ProtoMajor >= 2 {
				w := c.Response().Writer
				if flusher, ok := w.(http.Flusher); ok {
					for _, link := range preloadLinks {
						w.Header().Add("Link", link)
					}
					w.WriteHeader(http.StatusEarlyHints) // 103
					flusher.Flush()
				}
			} else {
				for _, link := range preloadLinks {
					c.Response().Header().Add("Link", link)
				}
			}
			return next(c)
		}
	})

	e.Use(middleware.ServerTimingMiddleware())
	e.Use(echo.WrapMiddleware(promolog.CorrelationMiddleware))
	e.Use(echoMiddleware.RequestLogger())
	e.Use(echoMiddleware.Recover())
	e.Use(echoMiddleware.Secure())
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Set("Permissions-Policy",
				"camera=(), microphone=(), geolocation=(), payment=(), usb=()")
			c.Response().Header().Set("Cross-Origin-Opener-Policy", "same-origin")
			return next(c)
		}
	})
	// Skip gzip when running behind the templ proxy (mage watch).
	// Echo's chunked gzip responses cause h2 framing errors through
	// the templ-proxy → Caddy chain. Caddy handles compression instead.
	if os.Getenv("TEMPL_PROXY") == "" {
		e.Use(echoMiddleware.Gzip())
	}

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

	e.HTTPErrorHandler = middleware.NewHTTPErrorHandler(reqLogStore)

	// setup:feature:session_settings:start
	if settingsRepo != nil {
		e.Use(middleware.SessionSettingsMiddleware(settingsRepo, nil))
	}
	// setup:feature:session_settings:end

	// setup:feature:demo:start
	e.Use(middleware.LinkRelationsMiddleware())
	// setup:feature:demo:end

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Set("Vary", "HX-Request")
			return next(c)
		}
	})

	static := e.Group("/public", func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			return next(c)
		}
	})
	static.StaticFS("/", staticFS)

	// setup:feature:offline:start
	// Serve the service worker from the root so it can control all pages.
	e.GET("/sw.js", func(c echo.Context) error {
		f, err := staticFS.Open("js/sw.js")
		if err != nil {
			return echo.NewHTTPError(http.StatusNotFound)
		}
		defer f.Close()
		c.Response().Header().Set("Content-Type", "application/javascript")
		c.Response().Header().Set("Service-Worker-Allowed", "/")
		return c.Stream(http.StatusOK, "application/javascript", f)
	})
	// setup:feature:offline:end

	return e, nil
}

