package routes

import (
	"catgoose/dothog/internal/config"
	"catgoose/dothog/internal/logger"
	// setup:feature:demo:start
	"catgoose/dothog/internal/demo"
	// setup:feature:demo:end
	// setup:feature:sse:start
	"github.com/catgoose/tavern"
	// setup:feature:sse:end
	"catgoose/dothog/internal/health"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/internal/version"
	"github.com/catgoose/promolog"
	// setup:feature:demo:start
	"github.com/catgoose/linkwell"
	// setup:feature:demo:end
	"catgoose/dothog/internal/routes/middleware"
	"catgoose/dothog/web/views"
	// setup:feature:session_settings:start
	"catgoose/dothog/internal/session"
	// setup:feature:session_settings:end
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/catgoose/dorman"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"time"
	// setup:feature:auth:start
	"github.com/catgoose/crooner"
	// setup:feature:auth:end
	"github.com/CAFxX/httpcompression"
	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
)

// Repos groups repository and store dependencies for the application routes.
// Generated apps add fields here as features are added.
type Repos struct {
	ReqLogStore   promolog.Storer
	IssueReporter IssueReporter
	// setup:feature:session_settings:start
	// SessionSettings is the management/admin seam for session_settings rows
	// (the /admin/sessions surface). See session.SettingsAdmin.
	SessionSettings session.SettingsAdmin
	// SessionStore is the read/write seam theme/layout handlers use to
	// persist updates onto the row hydrated by session.Middleware. Usually
	// the same concrete repo satisfies both this and SessionSettings.
	SessionStore session.SettingsProvider
	// setup:feature:session_settings:end
}

// AppRoutes owns the wired Echo server, runtime dependencies, and the SSE
// broker. Construct via NewAppRoutes, populate runtime deps via SetHealth*,
// and call Close at shutdown.
type AppRoutes struct {
	repos     Repos
	startTime time.Time
	ctx       context.Context
	e         *echo.Echo
	// setup:feature:demo:start
	demoDB *demo.DB
	// setup:feature:demo:end
	// setup:feature:sse:start
	broker *tavern.SSEBroker
	// setup:feature:sse:end
	healthCfg health.Config
}

// Close shuts down the SSE broker and releases resources.
func (ar *AppRoutes) Close() {
	// setup:feature:sse:start
	if ar.broker != nil {
		ar.broker.Close()
	}
	// setup:feature:sse:end
}

// setup:feature:sse:start

// initSSEBroker constructs ar.broker with the default keepalive/eviction
// policy and a slow-subscriber logger. Always-on when sse is selected, with
// or without demo content.
func (ar *AppRoutes) initSSEBroker() {
	ar.broker = tavern.NewSSEBroker(
		tavern.WithBufferSize(128),
		tavern.WithKeepalive(30*time.Second),
		tavern.WithSlowSubscriberEviction(100),
		tavern.WithSlowSubscriberCallback(func(topic string) {
			logger.Warn("Slow subscriber evicted", "topic", topic)
		}),
	)
	ar.broker.OnPublishDrop(func(topic string, count int) {
		logger.Debug("Message dropped", "topic", topic, "subscribers", count)
	})
}

// setup:feature:sse:end

// setup:feature:demo:start
// setup:feature:sse:start

// applyDemoSSEEnvelopes wraps demo dashboard topics so ScheduledPublisher
// raw HTML payloads ship with SSE event:/data: framing. Demo+SSE only —
// the wrapped topics are all demo dashboards/admin panels, so the wrap is
// pointless without demo content.
func (ar *AppRoutes) applyDemoSSEEnvelopes() {
	for _, topic := range []string{
		TopicDashMetrics,
		TopicNumericalDash,
		TopicAdminPanel,
		TopicSystemStats,
	} {
		topic := topic
		ar.broker.UseTopics(topic, func(next tavern.PublishFunc) tavern.PublishFunc {
			return func(t, msg string) {
				next(t, tavern.NewSSEMessage(topic, msg).String())
			}
		})
	}
}

// setup:feature:sse:end
// setup:feature:demo:end

// NewAppRoutes constructs the AppRoutes owner around an existing Echo
// instance. It stores route dependencies and defaults, but does not register
// endpoints yet; call InitRoutes to install the route table onto Echo. Nil
// IssueReporter falls back to a no-op reporter.
func NewAppRoutes(ctx context.Context, e *echo.Echo, repos Repos) *AppRoutes {
	if repos.IssueReporter == nil {
		repos.IssueReporter = defaultReporter{}
	}
	startTime := time.Now()
	return &AppRoutes{
		e:         e,
		ctx:       ctx,
		repos:     repos,
		startTime: startTime,
		healthCfg: health.Config{
			Version:   version.Version,
			StartTime: startTime,
		},
	}
}

// InitRoutes installs the application's route table onto the wrapped Echo and
// performs route-adjacent runtime setup such as broker creation and
// feature-gated route wiring. Returns an error only if config load fails;
// downstream demo features no-op when their data source is unavailable.
func (ar *AppRoutes) InitRoutes() error {
	// Register known origins for ?from= breadcrumb resolution.
	// Home (bit 0) is pre-registered. Additional pages register here.
	// setup:feature:demo:start
	linkwell.RegisterFrom(linkwell.FromDashboard, linkwell.Breadcrumb{Label: "Dashboard", Href: "/dashboard"})
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
		s := session.GetSettings(c.Request())
		return handler.RenderBaseLayout(c, views.AppSettingsPage(s.Theme))
	})
	// setup:feature:session_settings:end
	// setup:feature:demo:start
	ar.initLinkRelations()
	ar.e.GET("/welcome", handler.HandleComponent(views.WelcomePage()))
	ar.e.GET("/patterns", handler.HandleComponent(views.PatternsIndexPage()))
	ar.e.GET("/components", handler.HandleComponent(views.ComponentsIndexPage()))
	ar.e.GET("/realtime", handler.HandleComponent(views.RealtimeIndexPage()))
	ar.e.GET("/api", handler.HandleComponent(views.APIIndexPage()))
	ar.e.GET("/apps", handler.HandleComponent(views.ApplicationsIndexPage()))
	ar.e.GET("/platform", handler.HandleComponent(views.PlatformIndexPage()))
	// setup:feature:demo:end

	// Health check endpoint — returns structured ops metadata.
	healthHandler := func(c echo.Context) error {
		return c.JSON(http.StatusOK, health.Check(c.Request().Context(), ar.healthCfg))
	}
	ar.e.GET("/health", healthHandler)
	ar.e.HEAD("/health", healthHandler)

	ar.initReportIssueRoutes()

	ar.initAdminCoreRoutes()
	// setup:feature:session_settings:start
	ar.initAdminSessionsRoutes()
	// setup:feature:session_settings:end
	ar.initErrorTracesRoutes()

	// setup:feature:demo:start
	ar.initAdminDemoRoutes()
	ar.initReportDemoRoutes()
	ar.initControlsGalleryRoutes()
	ar.initComponentsRoutes()
	ar.initComponents2Routes()
	ar.initComponents3Routes()
	// setup:feature:demo:end

	// ── Core SSE infrastructure ─────────────────────────────────
	// Broker construction plus the scaffold-facing SSE routes derived apps
	// rely on: app-lifeline (consumed by the AppNav layout) and theme SSE
	// (cross-tab theme sync, paired with theme persistence). Anything else
	// SSE-flavored is demo content and lives below.
	// setup:feature:sse:start
	ar.initSSEBroker()
	ar.initLifelineRoutes(ar.broker)
	// setup:feature:sse:end
	// setup:feature:session_settings:start
	ar.initThemeRoutes()
	// setup:feature:sse:start
	ar.initThemeSSE(ar.broker)
	// setup:feature:sse:end
	// setup:feature:session_settings:end

	// ── Demo-only SSE showcase ──────────────────────────────────
	// Real-time playground, tavern gallery, and demo dashboards. Every
	// route here is jointly gated on demo+sse; the topic envelope wrappers
	// also live here because the wrapped topics are demo dashboards only.
	// setup:feature:demo:start
	// setup:feature:sse:start
	ar.applyDemoSSEEnvelopes()
	ar.initLoggingRoutes(ar.broker)
	ar.initRealtimeRoutes(ar.broker)
	ar.initNotificationsRoutes(ar.broker)
	ar.initDocRoutes(ar.broker)
	ar.initSensorRoutes(ar.broker)
	ar.initObservatoryRoutes(ar.broker)
	ar.initAuctionRoutes(ar.broker)
	ar.e.GET("/realtime/tavern", handler.HandleComponent(views.TavernIndexPage()))
	ar.initTavernReplayRoutes(ar.broker)
	ar.initTavernBackpressRoutes(ar.broker)
	ar.initTavernSubsRoutes(ar.broker)
	ar.initTavernPublishRoutes(ar.broker)
	ar.initTavernHooksRoutes(ar.broker)
	ar.initRecoveryRoutes(ar.broker)
	ar.initFailuresRoutes(ar.broker)
	ar.initTavernCalendarRoutes(ar.broker)
	ar.initTavernHotZoneRoutes(ar.broker)
	ar.initTavernToastRoutes(ar.broker)
	// setup:feature:sse:end
	// setup:feature:demo:end

	// setup:feature:demo:start
	ar.initHypermediaRoutes()
	ar.initHALRoutes()
	ar.initErrorsRoutes()
	ar.initErrorModesRoutes()

	db, err := demo.Open("db/demo.db")
	if err != nil {
		logger.WithContext(ar.ctx).Warn("Demo DB unavailable; app routes disabled", "error", err)
		return nil
	}
	ar.demoDB = db
	if stored, err := db.ListStoredLinks(); err == nil {
		for _, s := range stored {
			linkwell.LoadStoredLink(s.Source, linkwell.LinkRelation{
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
	// setup:feature:sse:start
	ar.initAdminSettingsRoutes(ar.broker)
	ar.initAdminRoutes(db, actLog, ar.broker)
	ar.initPeopleRoutes(db, ar.broker, actLog)
	ar.initKanbanRoutes(board, actLog, ar.broker)
	ar.initApprovalRoutes(queue, actLog, ar.broker)
	ar.initFeedRoutes(actLog, ar.broker)
	// setup:feature:sse:end
	ar.initCalendarRoutes()
	// setup:feature:sse:start
	ar.initCanvasRoutes(demo.NewPixelCanvas(), ar.broker)
	// setup:feature:sse:end
	ar.initSettingsRoutes(demo.NewSettingsStore())
	// setup:feature:sse:start
	ar.initVendorContactRoutes(db, actLog, ar.broker)
	// setup:feature:sse:end
	ar.initDashboardRoutes(db, board, queue, actLog)
	ar.initAdminErrorReportsRoutes(db)

	// setup:feature:demo:end
	ar.e.RouteNotFound("/*", handler.HandleNotFound)
	handler.InitRouteSet(ar.e, cfg.AppName)
	ar.healthCfg.Name = cfg.AppName
	return nil
}

// SetHealthDB injects the DB ping target used by GET /health; nil disables the DB row.
func (ar *AppRoutes) SetHealthDB(db health.Pinger) {
	ar.healthCfg.DB = db
}

// SetHealthStats injects the app-specific stats function rendered under GET /health.
func (ar *AppRoutes) SetHealthStats(fn health.StatsFunc) {
	ar.healthCfg.Stats = fn
}

// InitEcho assembles the project middleware chain (preload-link, correlation, recovery,
// server-timing, security headers, raw-writer + httpcompression, and feature-gated
// auth / session / link-relations); call once at startup, nil reqLogStore disables
// error-trace promotion.
func InitEcho(ctx context.Context, staticFS fs.FS, cfg *config.AppConfig,
	// setup:feature:session_settings:start
	settingsRepo session.SettingsProvider,
	// setup:feature:session_settings:end
	reqLogStore promolog.Storer,
) (*echo.Echo, error) {
	e := echo.New()

	// Preload critical assets. In production (direct H2), send 103 Early Hints
	// so the browser fetches CSS/JS while the server generates the response.
	// In `mage watch` the templ HTTP proxy (and optional Caddy in front of it)
	// is always in the chain — TEMPL_PROXY=true is set there — and 1xx
	// responses get mangled, so fall back to Link headers on the final
	// response. The browser still gets the preload hint, just slightly later.
	behindProxy := os.Getenv("TEMPL_PROXY") != ""
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			preloadLinks := []string{
				"<" + version.Asset("/public/css/tailwind.css") + ">; rel=preload; as=style; fetchpriority=high",
				"<" + version.Asset("/public/css/daisyui.css") + ">; rel=preload; as=style",
				"<" + version.Asset("/public/js/htmx.min.js") + ">; rel=preload; as=script; fetchpriority=high",
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

	e.Use(echoMiddleware.Recover())
	e.Use(middleware.ServerTimingMiddleware())
	e.Use(echo.WrapMiddleware(promolog.CorrelationMiddleware))
	e.Use(echoMiddleware.RequestLogger())
	e.Use(echo.WrapMiddleware(dorman.SecurityHeaders(dorman.SecurityHeadersConfig{
		PermissionsPolicy:       "camera=(), microphone=(), geolocation=(), payment=(), usb=()",
		CrossOriginOpenerPolicy: "same-origin",
	})))
	// Save the raw response writer before the compression middleware wraps it.
	// The error handler needs the unwrapped writer because httpcompression
	// finalizes (closes) its writer when the middleware chain unwinds, making
	// it unusable by the time Echo's HTTPErrorHandler runs.
	e.Use(middleware.RawWriterMiddleware())
	// Skip compression when running behind the templ proxy (mage watch).
	// Chunked compressed responses cause h2 framing errors through
	// the templ-proxy → Caddy chain. Caddy handles compression instead.
	if os.Getenv("TEMPL_PROXY") == "" {
		compress, err := httpcompression.DefaultAdapter()
		if err != nil {
			slog.Error("failed to create compression adapter", "error", err)
		} else {
			e.Use(echo.WrapMiddleware(compress))
		}
	}

	// setup:feature:auth:start
	if cfg != nil && cfg.AuthConfigured() {
		// Build crooner runtime locally from cfg's env-backed OIDC values.
		// AppConfig stays immutable; the SessionManager + AuthConfig live in
		// this function's scope.
		croonerParams := &crooner.AuthConfigParams{
			IssuerURL:         cfg.OIDCIssuerURL,
			ClientID:          cfg.OIDCClientID,
			ClientSecret:      cfg.OIDCClientSecret,
			RedirectURL:       cfg.OIDCRedirectURL,
			LoginURLRedirect:  cfg.OIDCLoginRedirectURL,
			LogoutURLRedirect: cfg.OIDCLogoutRedirectURL,
			AuthRoutes: &crooner.AuthRoutes{
				Login:    "/login",
				Logout:   "/logout",
				Callback: "/callback",
			},
		}
		sessionMgr, scsMgr, err := crooner.NewSCSManager(
			crooner.WithPersistentCookieName(cfg.SessionSecret, cfg.AppName),
			crooner.WithLifetime(24*time.Hour),
		)
		if err != nil {
			return nil, fmt.Errorf("crooner session manager: %w", err)
		}
		e.Use(echo.WrapMiddleware(scsMgr.LoadAndSave))
		croonerParams.SessionMgr = sessionMgr
		authCfg, err := crooner.NewAuthConfig(ctx, croonerParams)
		if err != nil {
			return nil, fmt.Errorf("crooner auth config: %w", err)
		}
		e.Use(echo.WrapMiddleware(authCfg.Middleware()))
		e.GET(croonerParams.AuthRoutes.Login, echo.WrapHandler(authCfg.LoginHandler()))
		e.GET(croonerParams.AuthRoutes.Logout, echo.WrapHandler(authCfg.LogoutHandler()))
		e.GET(croonerParams.AuthRoutes.Callback, echo.WrapHandler(authCfg.CallbackHandler()))
		// setup:feature:csrf:start
		if cfg.SessionSecret != "" {
			hash := sha256.Sum256([]byte(cfg.SessionSecret))
			csrfKey := hash[:]
			csrfMw := dorman.CSRFProtect(dorman.CSRFConfig{
				Key:              csrfKey,
				CookiePath:       "/",
				FieldName:        "csrf_token",
				RequestHeader:    "X-CSRF-Token",
				ExemptPaths:      cfg.CSRFExemptPaths,
				RotatePerRequest: cfg.CSRFRotatePerRequest,
				PerRequestPaths:  cfg.CSRFPerRequestPaths,
			})
			e.Use(echo.WrapMiddleware(csrfMw))
			// Inject token into echo context for templates
			e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
				return func(c echo.Context) error {
					c.Set("csrf_token", dorman.GetToken(c.Request()))
					return next(c)
				}
			})
		}
		// setup:feature:csrf:end
	}
	// setup:feature:auth:end

	e.HTTPErrorHandler = middleware.NewHTTPErrorHandler(reqLogStore)

	// setup:feature:session_settings:start
	if settingsRepo != nil {
		var sessCfg session.Config
		if cfg != nil && cfg.SessionSettingsCookieName != "" {
			sessCfg.CookieName = cfg.SessionSettingsCookieName
		}
		e.Use(echo.WrapMiddleware(session.Middleware(settingsRepo, nil, sessCfg)))
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
			if behindProxy {
				c.Response().Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			} else {
				c.Response().Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			return next(c)
		}
	})
	static.StaticFS("/", staticFS)

	return e, nil
}
