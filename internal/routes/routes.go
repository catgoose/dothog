// Package routes provides routing configuration and Echo server initialization.
package routes

import (
	"catgoose/dothog/internal/config"
	"catgoose/dothog/internal/logger"
	// setup:feature:demo:start
	"catgoose/dothog/internal/demo"
	// setup:feature:sse:start
	"catgoose/dothog/internal/ssebroker"
	// setup:feature:sse:end
	// setup:feature:demo:end
	"catgoose/dothog/internal/requestlog"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/internal/routes/hypermedia"
	// setup:feature:session_settings:start
	"catgoose/dothog/internal/repository"
	// setup:feature:session_settings:end
	corecomponents "catgoose/dothog/web/components/core"
	"catgoose/dothog/web/views"
	"catgoose/dothog/internal/routes/middleware"
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
	e             *echo.Echo
	ctx           context.Context
	reqLogStore   *requestlog.Store
	issueReporter IssueReporter
	startTime     time.Time
}

// NewAppRoutes initializes routes.
// reqLogStore may be nil if request log capture is disabled.
// reporter may be nil; a default no-op reporter is used.
func NewAppRoutes(ctx context.Context, e *echo.Echo, reqLogStore *requestlog.Store, reporter IssueReporter) AppRoutes {
	if reporter == nil {
		reporter = defaultReporter{}
	}
	return &appRoutes{
		e:             e,
		ctx:           ctx,
		reqLogStore:   reqLogStore,
		issueReporter: reporter,
		startTime:     time.Now(),
	}
}

func (ar *appRoutes) InitRoutes() error {
	// Register known origins for ?from= breadcrumb resolution.
	// Home (bit 0) is pre-registered. Additional pages register here.
	hypermedia.RegisterFrom(hypermedia.FromDashboard, hypermedia.Breadcrumb{Label: "Dashboard", Href: "/dashboard"})

	ar.e.GET("/", handler.HandleComponent(views.ArchitecturePage()))
	// setup:feature:session_settings:start
	ar.initUserSettingsRoutes()
	// setup:feature:session_settings:end
	// setup:feature:demo:start
	ar.e.GET("/welcome", handler.HandleComponent(views.WelcomePage()))
	// setup:feature:demo:end

	// Health check endpoint for Caddy
	ar.e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// Report issue endpoint — accepts a report, passes log entries to the
	// configured IssueReporter, and triggers a browser alert.
	reportHandler := func(c echo.Context) error {
		requestID := c.Param("requestID")
		description := c.FormValue("description")
		var trace *requestlog.ErrorTrace
		if ar.reqLogStore != nil && requestID != "" {
			trace = ar.reqLogStore.Get(requestID)
		}
		if err := ar.issueReporter.Report(requestID, description, trace); err != nil {
			logger.WithContext(c.Request().Context()).Error("Issue report failed",
				"reported_request_id", requestID, "error", err)
			c.Response().Header().Set("HX-Trigger", `{"showAlert":"Failed to submit report. Please try again."}`)
			c.Response().Header().Set("HX-Reswap", "none")
			return c.String(http.StatusInternalServerError, "")
		}
		c.Response().Header().Set("HX-Trigger", `{"showAlert":"Issue reported. Thank you for your feedback!"}`)
		c.Response().Header().Set("HX-Reswap", "none")
		return c.String(http.StatusOK, "")
	}
	ar.e.POST("/report-issue", reportHandler)
	ar.e.POST("/report-issue/:requestID", reportHandler)

	// Fetch the Report Issue modal for a request.
	ar.e.GET("/report-issue/:requestID", func(c echo.Context) error {
		requestID := c.Param("requestID")
		cfg := hypermedia.ReportIssueModal(requestID)
		return handler.RenderComponent(c, corecomponents.ReportIssueModal(cfg))
	})

	ar.initAdminCoreRoutes()
	ar.initErrorTracesRoutes()

	// setup:feature:demo:start
	ar.initReportDemoRoutes()
	ar.initLoggingRoutes()
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
	ar.initErrorsRoutes()
	// setup:feature:sse:start
	ar.initRealtimeRoutes(broker)
	// setup:feature:sse:end

	db, err := demo.Open("db/demo.db")
	if err != nil {
		logger.WithContext(ar.ctx).Warn("Demo DB unavailable; /demo/* routes disabled", "error", err)
		return nil
	}
	ar.initInventoryRoutes(db)
	ar.initCatalogRoutes(db)
	ar.initBulkRoutes(db)
	ar.initRepositoryRoutes(db)

	actLog := demo.NewActivityLog(200)
	board := demo.NewKanbanBoard()
	queue := demo.NewApprovalQueue()
	ar.initAdminRoutes(db, actLog, broker)
	ar.initPeopleRoutes(db, broker, actLog)
	ar.initKanbanRoutes(board, actLog, broker)
	ar.initApprovalRoutes(queue, actLog, broker)
	ar.initFeedRoutes(actLog, broker)
	ar.initSettingsRoutes(demo.NewSettingsStore())
	ar.initVendorContactRoutes(db, actLog, broker)
	ar.initDashboardRoutes(db, board, queue, actLog)
	// setup:feature:demo:end
	ar.e.RouteNotFound("/*", handler.HandleNotFound)
	cfg, err := config.GetConfig()
	if err != nil {
		return fmt.Errorf("handler init: %w", err)
	}
	handler.InitRouteSet(ar.e, cfg.AppName)
	return nil
}

// InitEcho initializes Echo with global configurations
func InitEcho(ctx context.Context, staticFS fs.FS, cfg *config.AppConfig,
	// setup:feature:session_settings:start
	settingsRepo repository.SessionSettingsRepository,
	// setup:feature:session_settings:end
	reqLogStore *requestlog.Store,
) (*echo.Echo, error) {
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

	e.Use(middleware.ErrorHandlerMiddleware(reqLogStore))

	// setup:feature:session_settings:start
	if settingsRepo != nil {
		e.Use(middleware.SessionSettingsMiddleware(settingsRepo))
		e.POST("/settings/theme", handleTheme(settingsRepo))
	}
	// setup:feature:session_settings:end

	e.StaticFS("/public", staticFS)

	return e, nil
}

// setup:feature:session_settings:start

// handleTheme updates the Theme session setting and redirects back.
func handleTheme(repo repository.SessionSettingsRepository) echo.HandlerFunc {
	return func(c echo.Context) error {
		theme := c.FormValue("theme")
		valid := false
		for _, t := range views.DaisyThemes {
			if t == theme {
				valid = true
				break
			}
		}
		if !valid {
			theme = "light"
		}
		settings := middleware.GetSessionSettings(c)
		settings.Theme = theme
		if repo != nil {
			_ = repo.Upsert(c.Request().Context(), settings)
		}
		referer := c.Request().Header.Get("Referer")
		if referer == "" {
			referer = "/"
		}
		return c.Redirect(http.StatusSeeOther, referer)
	}
}

// setup:feature:session_settings:end
