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
	"catgoose/dothog/internal/responsepolicy"
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

// Deps groups repository and store dependencies for the application routes.
// Generated apps add fields here as features are added. main.go owns
// construction of every field here so InitRoutes never discovers runtime
// state on its own (no config.GetConfig, no demo.Open).
type Deps struct {
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
	// setup:feature:demo:start
	// DemoDB is the optional demo SQLite store. Nil means "demo content
	// disabled" — main.go warns and continues when db/demo.db is missing,
	// so demo routes that need a backing DB no-op silently.
	DemoDB *demo.DB
	// setup:feature:demo:end
	// AppName is the display label main.go pulls from AppConfig once at
	// startup. Used for the home page title, the AppNav layout, and the
	// /health name field.
	AppName string
}

// AppRoutes owns the wired Echo server, runtime dependencies, and the SSE
// broker. Construct via NewAppRoutes, populate runtime deps via SetHealth*,
// and call Close at shutdown.
type AppRoutes struct {
	deps      Deps
	startTime time.Time
	ctx       context.Context
	e         *echo.Echo
	// healthIntervals returns the live admin section intervals shown on the
	// /admin/health page. Demo's admin settings hooks this up in
	// initAdminSettingsRoutes; with demo stripped it stays nil and the
	// scaffold renders the page with no interval data.
	healthIntervals func() map[string]int
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
func NewAppRoutes(ctx context.Context, e *echo.Echo, deps Deps) *AppRoutes {
	if deps.IssueReporter == nil {
		deps.IssueReporter = defaultReporter{}
	}
	startTime := time.Now()
	ar := &AppRoutes{
		e:         e,
		ctx:       ctx,
		deps:      deps,
		startTime: startTime,
		healthCfg: health.Config{
			Name:      deps.AppName,
			Version:   version.Version,
			StartTime: startTime,
		},
	}
	// setup:feature:demo:start
	ar.demoDB = deps.DemoDB
	// setup:feature:demo:end
	return ar
}

// InitRoutes is the high-level coordinator that wires the application's
// route table onto the embedded Echo. Each phase is its own helper so the
// scaffold/admin/SSE/demo boundaries stay legible at a glance; behavior is
// identical to running every helper inline. Runtime inputs (app name, demo
// DB) are injected via Deps in NewAppRoutes — InitRoutes does not call
// config.GetConfig or demo.Open itself.
func (ar *AppRoutes) InitRoutes() error {
	ar.initScaffoldRoutes()
	ar.initHealthRoutes()
	ar.initScaffoldAdminRoutes()
	ar.initExamplesRoutes()
	ar.initSSERoutes()
	// setup:feature:demo:start
	ar.initDemoSSEShowcase()
	ar.initDemoExtras()
	ar.initDemoDBRoutes()
	// setup:feature:demo:end
	ar.finalizeRoutes()
	return nil
}

// initScaffoldRoutes wires the home page, the session_settings page, and
// the demo-feature navigation links derived apps reach from the AppNav
// layout. Single registrar so derived apps know exactly where GET / lives.
func (ar *AppRoutes) initScaffoldRoutes() {
	// Register known origins for ?from= breadcrumb resolution. Home (bit 0)
	// is pre-registered; additional pages register here.
	// setup:feature:demo:start
	linkwell.RegisterFrom(linkwell.FromDashboard, linkwell.Breadcrumb{Label: "Dashboard", Href: "/dashboard"})
	// setup:feature:demo:end

	// Single root handler — derived apps register GET / exactly once. The
	// scaffold default serves the HomePage; the demo feature overrides the
	// handler via the gated assignment below so there is no Echo route
	// override at registration time. The default assignment looks ineffectual
	// in the template (with demo present) but becomes the only assignment
	// once demo is stripped.
	homeHandler := handler.HandleComponent(views.HomePage(ar.deps.AppName)) //nolint:ineffassign,staticcheck // default-then-demo-override is intentional under feature stripping.
	// setup:feature:demo:start
	homeHandler = handler.HandleComponent(views.ArchitecturePage())
	// setup:feature:demo:end
	ar.e.GET("/", homeHandler)
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
}

// initHealthRoutes wires the structured GET/HEAD /health endpoint backed by
// healthCfg seeded in NewAppRoutes.
func (ar *AppRoutes) initHealthRoutes() {
	healthHandler := func(c echo.Context) error {
		return c.JSON(http.StatusOK, health.Check(c.Request().Context(), ar.healthCfg))
	}
	ar.e.GET("/health", healthHandler)
	ar.e.HEAD("/health", healthHandler)
}

// initScaffoldAdminRoutes wires report-issue, admin core, the optional
// session_settings admin, error traces, and the demo admin showcase pages.
func (ar *AppRoutes) initScaffoldAdminRoutes() {
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
}

// initSSERoutes builds the SSE broker and wires the scaffold-facing SSE
// routes derived apps always need (app-lifeline + session-owned theme sync).
// Everything demo-flavored that piggybacks on the broker lives in
// initDemoSSEShowcase.
func (ar *AppRoutes) initSSERoutes() {
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
}

// setup:feature:demo:start

// initDemoSSEShowcase wires the demo+sse playground: realtime, tavern
// gallery, observatory/auction/sensors, and the demo dashboards. Jointly
// gated on demo+sse — when sse is stripped the body collapses to nothing.
// The topic envelope wrappers live here too because the wrapped topics are
// demo dashboards only.
func (ar *AppRoutes) initDemoSSEShowcase() {
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
}

// initDemoExtras wires the demo-only routes that don't need either the
// demo DB or the SSE broker: hypermedia, HAL, errors, error modes.
func (ar *AppRoutes) initDemoExtras() {
	ar.initHypermediaRoutes()
	ar.initHALRoutes()
	ar.initErrorsRoutes()
	ar.initErrorModesRoutes()
}

// initDemoDBRoutes wires every demo route whose handlers touch the demo
// SQLite store. main.go owns demo.Open and logs the warn-and-continue when
// db/demo.db is missing; this method returns silently when ar.demoDB is
// nil, letting the shared finalization tail in InitRoutes still run.
func (ar *AppRoutes) initDemoDBRoutes() {
	db := ar.demoDB
	if db == nil {
		return
	}
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
}

// setup:feature:demo:end

// finalizeRoutes installs the route-not-found handler and primes the
// breadcrumb route-set with the full GET table. Runs last so InitRouteSet
// sees every registered route.
func (ar *AppRoutes) finalizeRoutes() {
	ar.e.RouteNotFound("/*", handler.HandleNotFound)
	handler.InitRouteSet(ar.e, ar.deps.AppName)
}

// SetHealthDB injects the DB ping target used by GET /health; nil disables the DB row.
func (ar *AppRoutes) SetHealthDB(db health.Pinger) {
	ar.healthCfg.DB = db
}

// SetHealthStats injects the app-specific stats function rendered under GET /health.
func (ar *AppRoutes) SetHealthStats(fn health.StatsFunc) {
	ar.healthCfg.Stats = fn
}

// InitEcho assembles the project middleware chain. The web-standards response
// policy (103/preload, Server-Timing, security headers, Vary: HX-Request) is
// installed first via internal/responsepolicy as one explicit owner; the rest
// of the chain — Recover/correlation/logging, raw-writer, compression,
// feature-gated auth/session/link-relations, and the HTTPErrorHandler — stays
// here in InitEcho. Order matters: response-policy precedes everything so 103
// Early Hints flushes before any other handler writes a header, the raw-writer
// save must precede compression, and the static handler runs last.
func InitEcho(ctx context.Context, staticFS fs.FS, cfg *config.AppConfig,
	// setup:feature:session_settings:start
	settingsRepo session.SettingsProvider,
	// setup:feature:session_settings:end
	reqLogStore promolog.Storer,
) (*echo.Echo, error) {
	e := echo.New()
	behindProxy := os.Getenv("TEMPL_PROXY") != ""

	responsepolicy.Install(e, responsepolicy.Config{
		BehindProxy: behindProxy,
		PreloadLinks: []string{
			"<" + version.Asset("/public/css/tailwind.css") + ">; rel=preload; as=style; fetchpriority=high",
			"<" + version.Asset("/public/css/daisyui.css") + ">; rel=preload; as=style",
			"<" + version.Asset("/public/js/htmx.min.js") + ">; rel=preload; as=script; fetchpriority=high",
		},
		Security: dorman.SecurityHeadersConfig{
			PermissionsPolicy:       "camera=(), microphone=(), geolocation=(), payment=(), usb=()",
			CrossOriginOpenerPolicy: "same-origin",
		},
	})

	addCoreMiddleware(e)
	addCompressionMiddleware(e)
	if err := addAuthMiddleware(ctx, e, cfg); err != nil {
		return nil, err
	}
	e.HTTPErrorHandler = handler.NewHTTPErrorHandler(reqLogStore)
	// setup:feature:session_settings:start
	addSessionMiddleware(e, settingsRepo, cfg)
	// setup:feature:session_settings:end
	// LinkRelationsMiddleware is scaffold-facing infrastructure: it powers
	// breadcrumbs and the LocalContextBar via the link registry, which is
	// always-on baseline behavior. Demo and scaffold seams both register
	// relations into the same registry; the middleware reads whatever was
	// registered.
	e.Use(middleware.LinkRelationsMiddleware())
	addStaticHandler(e, staticFS, behindProxy)

	return e, nil
}

// addCoreMiddleware installs Recover, correlation, request logging, and the
// raw-writer save. The web-standards response policy ran earlier via
// responsepolicy.Install; RawWriter must run before compression so the error
// handler can swap back to the raw writer after httpcompression finalises its
// wrapper.
func addCoreMiddleware(e *echo.Echo) {
	e.Use(echoMiddleware.Recover())
	e.Use(echo.WrapMiddleware(promolog.CorrelationMiddleware))
	e.Use(echoMiddleware.RequestLogger())
	e.Use(middleware.RawWriterMiddleware())
}

// addCompressionMiddleware installs httpcompression unless TEMPL_PROXY is
// set. Chunked compressed responses cause h2 framing errors through the
// templ-proxy → Caddy chain in mage watch; Caddy handles compression
// instead in that mode.
func addCompressionMiddleware(e *echo.Echo) {
	if os.Getenv("TEMPL_PROXY") != "" {
		return
	}
	compress, err := httpcompression.DefaultAdapter()
	if err != nil {
		slog.Error("failed to create compression adapter", "error", err)
		return
	}
	e.Use(echo.WrapMiddleware(compress))
}

// addAuthMiddleware wires the crooner SCS session manager, OIDC handlers,
// and the CSRF protector. No-op when auth is not configured. Returns any
// crooner construction errors so InitEcho can surface them at startup.
func addAuthMiddleware(ctx context.Context, e *echo.Echo, cfg *config.AppConfig) error {
	// setup:feature:auth:start
	if cfg == nil || !cfg.AuthConfigured() {
		return nil
	}
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
		return fmt.Errorf("crooner session manager: %w", err)
	}
	e.Use(echo.WrapMiddleware(scsMgr.LoadAndSave))
	croonerParams.SessionMgr = sessionMgr
	authCfg, err := crooner.NewAuthConfig(ctx, croonerParams)
	if err != nil {
		return fmt.Errorf("crooner auth config: %w", err)
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
	// setup:feature:auth:end
	return nil
}

// setup:feature:session_settings:start

// addSessionMiddleware installs the session_settings middleware so request
// handlers can read the hydrated row via session.GetSettings. Nil
// settingsRepo skips the middleware so handlers see no session row.
func addSessionMiddleware(e *echo.Echo, settingsRepo session.SettingsProvider, cfg *config.AppConfig) {
	if settingsRepo == nil {
		return
	}
	var sessCfg session.Config
	if cfg != nil && cfg.SessionSettingsCookieName != "" {
		sessCfg.CookieName = cfg.SessionSettingsCookieName
	}
	e.Use(echo.WrapMiddleware(session.Middleware(settingsRepo, nil, sessCfg)))
}

// setup:feature:session_settings:end

// addStaticHandler mounts the embedded static FS under /public with a
// Cache-Control policy that flips between immutable production caching and
// no-cache during mage watch (the templ proxy serves rebuilt assets, so the
// browser must not cache).
func addStaticHandler(e *echo.Echo, staticFS fs.FS, behindProxy bool) {
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
}
