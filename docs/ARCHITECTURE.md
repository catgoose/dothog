# Architecture

This document describes how dothog processes requests, resolves navigation, and renders pages. For design principles and rationale, see [PHILOSOPHY.md](../PHILOSOPHY.md).

## Ecosystem Libraries

| Library | Purpose |
|---------|---------|
| [chuck](https://github.com/catgoose/chuck) | Multi-dialect SQL schema and query fragments |
| [promolog](https://github.com/catgoose/promolog) | Per-request log capture with promote-on-error |
| [crooner](https://github.com/catgoose/crooner) | OIDC/OAuth2 authentication and session management |
| [dorman](https://github.com/catgoose/dorman) | Authorization, CSRF protection, security headers |
| [linkwell](https://github.com/catgoose/linkwell) | HATEOAS link registry, navigation, hypermedia controls |
| [tavern](https://github.com/catgoose/tavern) | Thread-safe SSE pub/sub broker |

## Request Lifecycle

`InitEcho()` (`internal/routes/routes.go`) installs the pipeline in a single ordered chain. `internal/responsepolicy` owns the always-on web-standards layer at the front; auth/CSRF/session-settings/link-relations follow; the central error handler sits behind everything, ready to render any `*handler.SurfaceError`, `linkwell.HTTPError`, or `echo.HTTPError` that bubbles up.

```
Request
  │
  ├─ responsepolicy.Install               internal/responsepolicy
  │   ├─ Preload(Links, behindProxy)      103 Early Hints / Link header preload
  │   ├─ ServerTiming                     total request duration → Server-Timing
  │   ├─ dorman.SecurityHeaders           X-Frame-Options · HSTS · Permissions-Policy · COOP
  │   └─ Vary: HX-Request                 differentiate cached HTMX fragments vs. full pages
  │
  ├─ addCoreMiddleware                    internal/routes/routes.go
  │   ├─ Recover                          panic recovery → 500 through the central handler
  │   ├─ promolog.CorrelationMiddleware   X-Request-ID + per-request log buffer
  │   ├─ RequestLogger                    structured access log
  │   └─ RawWriterMiddleware              saves the pre-compression writer for the error handler
  │
  ├─ addCompressionMiddleware             zstd/brotli/gzip via httpcompression
  │                                       (skipped when TEMPL_PROXY is set)
  │
  ├─ addAuthMiddleware [feature:auth]
  │   ├─ crooner SCS LoadAndSave          OIDC session manager
  │   ├─ crooner authCfg.Middleware       OAuth/OIDC flow + login redirect
  │   └─ dorman.CSRFProtect  [feature:csrf]  HMAC-SHA256 double-submit + token in context
  │
  ├─ e.HTTPErrorHandler =
  │   handler.NewHTTPErrorHandler(reqLogStore)
  │                                       handles *SurfaceError, linkwell.HTTPError, echo.HTTPError;
  │                                       restores raw writer; promotes log buffer to trace store.
  │
  ├─ addSessionMiddleware [feature:session_settings]
  │                                       hydrates session.SessionSettings into request context
  │
  ├─ middleware.LinkRelationsMiddleware   linkwell.LinksFor(path) → Link header + echo context
  │
  └─ Handler
       ├─ handler.RenderBaseLayout(c, component)    full page through AppNavLayout
       ├─ handler.RenderComponent(c, component)     HTMX fragment
       ├─ handler.HandleComponent(component)        echo.HandlerFunc convenience wrapper
       └─ return handler.NewSurfaceError(c, …)      error surface chosen by route
```

`// setup:feature:TAG` markers gate the optional features; derived apps strip out blocks they didn't select. `responsepolicy` and `LinkRelationsMiddleware` are baseline behavior with no feature toggle — every derived app inherits them.

### Request Pipeline (ASCII view)

The same pipeline as a boxed companion to the README's Mermaid view, sized for inline reading.

```
                     Browser (HTML + HTMX)
                              │
                              │  HTTP request
                              ▼
   +----------------------------------------------------------------+
   |  Echo pipeline · InitEcho (internal/routes/routes.go)          |
   +----------------------------------------------------------------+
   |  responsepolicy.Install         internal/responsepolicy        |
   |    103/Link preload · Server-Timing · security · Vary          |
   +----------------------------------------------------------------+
   |  core middleware                                                |
   |    Recover · promolog correlation · access log                 |
   |    raw-writer save  (so the error handler can write post-comp) |
   +----------------------------------------------------------------+
   |  compression  (skipped when TEMPL_PROXY is set)                 |
   +----------------------------------------------------------------+
   |  crooner OIDC + SCS · dorman CSRF                               |
   |  session_settings hydration                                     |
   +----------------------------------------------------------------+
   |  e.HTTPErrorHandler = handler.NewHTTPErrorHandler               |
   |  LinkRelationsMiddleware  (always-on baseline)                  |
   |    linkwell.LinksFor(path) → Link header + echo context         |
   +----------------------------------------------------------------+
                              │
                              ▼
                +-------------------------------+
                |  handler (route function)     |
                +-------------------------------+
                       │                │
              success  │                │  *SurfaceError /
                       │                │  *linkwell.HTTPError
                       ▼                ▼
              +-------------+   +----------------------------------+
              | templ→HTML  |   | handler.NewHTTPErrorHandler       |
              +-------------+   |   trace promotion → promolog      |
                                |   raw-writer restore              |
                                |   capability negotiation (HTMX)   |
                                +----------------------------------+
                                                │
                                                ▼
                       banner · inline · inline-full · page · document
```

### Handler → Layout → Templ

1. Handler calls `handler.RenderBaseLayout(c, component)` for full pages or `handler.RenderComponent(c, component)` for HTMX fragments.
2. `RenderBaseLayout` checks for a custom layout (set via `handler.SetLayout()`). If none, it calls `renderDefaultLayout`.
3. `renderDefaultLayout` calls `getLayoutCtx(c)` to extract:
   - CSRF token (from `c.Get("csrf_token")`)
   - Theme (from session settings)
   - Current path
   - Breadcrumbs (resolved by priority — see below)
   - Link relations (from middleware)
   - Hub entries (for site map footer)
4. Layout renders `AppNavLayout` with the full navigation context.
5. Templ renders HTML and writes the response.

**Key files:**
- `internal/routes/routes.go` — `InitEcho()`, `InitRoutes()`
- `internal/routes/handler/handler.go` — `RenderBaseLayout`, `getLayoutCtx`, `renderDefaultLayout`

## Link Relations System

The link registry (`linkwell`) is the navigation topology of the application. All context bars, breadcrumbs, and the site map footer derive from it.

### Navigation Topology (ASCII view)

How the three seams write into one registry and how the registry drives the four runtime surfaces. The curated top nav is deliberately outside this loop — it is configured by hand, not by registration.

```
   +----------------------------------------------------------------+
   |  linkwell registry  (package-global, append-only Hub)          |
   |                                                                |
   |    Hub("/examples", "Examples", Rel("/examples/forms", …))     |
   |    Ring("Data", Rel("/demo/inventory"), Rel("/demo/catalog"))  |
   |    Link("/settings", "related", "/admin/config", "Admin Cfg")  |
   +----------------------------------------------------------------+
       ▲                          ▲                          ▲
       │ scaffold seam             │ demo seam                 │ stored seam
       │ initExamplesRoutes        │ initLinkRelations         │ demo.db
       │ (survives mage setup)     │ (// setup:feature:demo)   │ stored_links
       │                          │                          │
   +----------------------------------------------------------------+
   |  LinkRelationsMiddleware                                        |
   |    linkwell.LinksFor(path) → Link header + echo context         |
   +----------------------------------------------------------------+
       │
       ▼
   +-------------------------+   +-------------------------+
   |  LocalContextBar        |   |  Breadcrumbs            |
   |    ring siblings        |   |    ?from= bitmask  →    |
   |    hub-parent ↑ link    |   |    rel="up" chain   →    |
   |                          |   |    URL path fallback     |
   +-------------------------+   +-------------------------+
   +-------------------------+   +-------------------------+
   |  SiteMap (footer)        |   |  Curated top nav         |
   |    renders the whole     |   |    appNavNavConfig       |
   |    hub tree on every     |   |    manual; not derived   |
   |    page                   |   |    from the registry     |
   +-------------------------+   +-------------------------+
```

### Registration

Links are registered at startup using three primitives. Registration is
**split into two seams** — scaffold-facing relations and demo-only relations —
that both write to the same package-global linkwell registry. `linkwell.Hub`
is append-only, so both seams can extend a shared hub (e.g. `/admin`) without
stepping on each other.

| Primitive | Semantics | Example |
|-----------|-----------|---------|
| `linkwell.Hub(center, title, spokes...)` | Parent→children. Center gets `rel="related"` to each spoke. Each spoke gets `rel="up"` to center. | `Hub("/examples", "Examples", Rel("/examples/error-scenarios", "Error Scenarios"))` |
| `linkwell.Ring(name, members...)` | Symmetric peers. Every member gets `rel="related"` to every other member, grouped by ring name. | `Ring("Data", Rel("/demo/inventory", "Inventory"), Rel("/demo/catalog", "Catalog"))` |
| `linkwell.Link(source, rel, target, title)` | Pairwise. `rel="related"` auto-creates the inverse. | `Link("/settings", "related", "/admin/config", "Admin Config")` |

A page can belong to multiple rings and one hub. The registry deduplicates automatically.

### Seam ownership

- **Scaffold-facing relations** live in `internal/routes/examples.go::initExamplesRoutes` (and any future scaffold-owned route initializers). These survive `mage setup` regardless of feature selection.
- **Demo-only relations** live in `internal/routes/links.go::initLinkRelations`. The file is `// setup:feature:demo`; stripping demo removes the call.

The **curated top nav** in `internal/routes/handler/handler.go::appNavNavConfig` is a separate concern — it lists discoverability anchors, not every page in the link graph. Adding a top-nav entry does not register a link relation, and registering a link does not add a nav entry. Keep them aligned manually when both should reflect the same page.

### Middleware Resolution

`LinkRelationsMiddleware` (`internal/routes/middleware/links.go`) runs on every request:

1. Calls `linkwell.LinksFor(path)` to get all registered relations for the current path (walks parent paths if no exact match).
2. Sets the `Link` HTTP header (RFC 8288 format).
3. Stores links on the echo context for template rendering.

### Stored Links

Links can also be loaded from the database at startup via `linkwell.LoadStoredLink()`. The demo DB stores link relations in a `stored_links` table, loaded during `InitRoutes()`.

**Key files:**
- `internal/routes/examples.go` — scaffold-facing Hub/Rel declarations (always-on)
- `internal/routes/links.go` — demo-only Hub/Ring/Link declarations
- `internal/routes/middleware/links.go` — `LinkRelationsMiddleware()` (always-on baseline)

## Context Bar Resolution

The context bar shows related pages grouped by their ring membership. Resolution logic lives in `web/components/core/context_bar.templ`:

1. **Find the hub**: Check if the current page has `rel="up"` (spoke page) or outgoing `rel="related"` with a group name (hub center).
2. **Get spokes**: If hub center, use outgoing related links. If spoke, fetch the hub's related links.
3. **Resolve into rings**: Group spokes by their ring membership. Each ring becomes a named section in the context bar.
4. **Add parent link**: Spoke pages prepend a `↑ Hub Name` link to navigate up.
5. **Fallback**: Pages with no hub relationship fall back to simple grouping by `Group` field.

Hub center pages and spoke pages see the same grouped view — the difference is that spoke pages include the `↑` parent link.

## Breadcrumb System

Breadcrumbs are resolved in `getLayoutCtx()` (`internal/routes/handler/handler.go`) with three-tier priority:

### Priority 1: `?from=` bitmask (explicit navigation context)

When a user navigates via a link that includes `?from=N`, the bitmask encodes which pages they came through. `linkwell.ResolveFromMask(mask)` decodes the bitmask into breadcrumb entries. Origins are registered at startup via `linkwell.RegisterFrom()`.

### Priority 2: `rel="up"` chain (declared hierarchy)

`linkwell.BreadcrumbsFromLinks(path)` walks the `rel="up"` chain: current page → parent → grandparent → Home. This produces breadcrumbs like `Home > Demo > Inventory > Item Name`. Cycle detection prevents infinite loops.

### Priority 3: URL path segments (fallback)

`buildPathCrumbs(path, from, routes)` splits the URL into segments. Only segments with a registered GET route produce linked breadcrumbs. The terminal segment is always shown (unlinked).

### Page Labels

`handler.SetPageLabel(c, label)` overrides the terminal breadcrumb label. Detail page handlers use this to show the resource name (e.g., "Widget A" instead of "42").

### Boosted Navigation

`hx-boost` navigation sends full-page requests with the `HX-Boosted` header. Handlers check `htmx.IsBoosted(c.Request())` to decide whether to render a full layout or just a fragment.

## Session Settings

Session settings provide per-session preferences (theme, layout choice) stored in SQLite.

### Storage

- `session.SessionSettings` struct: UUID, Theme, Layout, CreatedAt, UpdatedAt
- SQLite repository implementing `session.Provider` interface
- All visitors share a single row (shared UUID) for the demo

### Middleware

`session.Middleware` (`internal/session/session.go`):
1. Loads settings by shared UUID via `repo.GetByUUID()`
2. Falls back to `session.NewDefaultSettings()` on error or missing row
3. Auto-creates the row if it doesn't exist
4. Touches the row if last update was > 24 hours ago
5. Stores settings on the request context

### Handlers

- `POST /settings/theme` — updates the theme (dark/light/etc.)
- `POST /settings/layout` — toggles between classic Index and AppNavLayout
- Both return updated page fragments for HTMX swap

## SSE System

Server-Sent Events provide real-time updates without polling.

### SSEBroker

`tavern.NewSSEBroker()` (from `github.com/catgoose/tavern`) implements topic-based pub/sub:

- `NewSSEBroker()` — creates a broker instance
- `Subscribe(topic)` — returns a read channel and unsubscribe function
- `SubscribeScoped(topic, scope)` — per-user/per-session subscriptions
- `Publish(topic, data)` — sends to all subscribers on a topic
- `PublishTo(topic, scope, data)` — sends to matching scoped subscribers only
- `HasSubscribers(topic)` — checks if anyone is listening
- `Close()` — shuts down all subscriptions (called via `defer ar.Close()` in main.go)

### Wiring

1. `routes.go` creates a single `tavern.NewSSEBroker()` instance.
2. Route initializers receive the broker.
3. SSE endpoints use `broker.Subscribe(topic)` and stream events.
4. Mutation handlers call `broker.Publish(topic, html)` to push OOB swap fragments.

### Client Side

```html
<div hx-ext="sse" sse-connect="/sse/people" sse-swap="people-updated">
  <!-- Content updated by SSE -->
</div>
```

The server sends named events. HTMX's SSE extension swaps the HTML fragment into the matching target.

## Response Policy

`internal/responsepolicy` is the explicit owner for the always-on web-standards response layer. `responsepolicy.Install(e, cfg)` runs first in `InitEcho` and registers, in order:

| Middleware | Purpose |
|------------|---------|
| `Preload(links, behindProxy)` | Sends 103 Early Hints with the supplied `Link: …; rel=preload` set (or falls back to the final-response `Link` header when behind a proxy that mangles 1xx). |
| `ServerTiming()` | Measures the request and adds a `Server-Timing` header for browser DevTools. |
| `dorman.SecurityHeaders(cfg.Security)` | X-Frame-Options, X-Content-Type-Options, Referrer-Policy, Permissions-Policy, Cross-Origin-Opener-Policy, optional HSTS / CSP. |
| `Vary("HX-Request")` | Ensures caches treat HTMX fragments and full pages as distinct representations. |

The behavior has no setup feature toggle — it's the framework baseline. Deployment-level knobs (preload list, CSP details, whether to skip 103 hints behind the templ proxy) come through `responsepolicy.Config`.

## HTMX Transport Seam

`internal/htmxutil` is a thin wrapper over [`github.com/angelofallars/htmx-go`](https://github.com/angelofallars/htmx-go). It is intentionally a wire-protocol layer, not a framework abstraction — it exposes only the request-detection and response-header mechanics dothog has chosen to standardize on. Application semantics (banner/page/document surfaces, report-issue meaning) live elsewhere.

| Function | Purpose |
|----------|---------|
| `IsHTMX(r)` / `IsBoosted(r)` | Detect HX-Request and HX-Boosted. |
| `CurrentURL(r)` / `CurrentRawQuery(r)` | Parse `HX-Current-URL` for filter/state preservation. |
| `htmxutil.New()` | Start a response builder. |
| `.ReplaceURL(url)` / `.Refresh()` | `HX-Replace-Url` / `HX-Refresh`. |
| `.Trigger(name)` / `.TriggerDetail(name, data)` | `HX-Trigger`. |
| `.TriggerAfterSwap(name)` / `.TriggerAfterSettle(name)` | After-swap / after-settle event timing. |
| `.Retarget(sel)` / `.Reselect(sel)` / `.Reswap(strat)` / `.ReswapNone()` | Late server-side overrides of the triggering element's hx-target / hx-swap. |
| `.Write(w)` | Flush the accumulated headers. |

Reach for `htmxutil` whenever a handler needs to set an HTMX response header. The scaffold-facing forms example (`internal/routes/forms.go`) is a working call site that uses `TriggerAfterSettle("focusFirstError")` to drive focus on the first invalid field after a validation rerender.

## Error Handling

Errors are hypermedia responses with navigation controls, not dead ends. The handler-owned surface contract is the single funnel; `handler.NewHTTPErrorHandler` is the only renderer.

### Error-Surface Negotiation (ASCII view)

How a route's chosen surface becomes the rendered surface, including the HTMX capability negotiation step.

```
   Route handler                       HTMX client (optional headers)
   ─────────────                       ──────────────────────────────
   return handler.NewSurfaceError(     X-Error-Accept-Surfaces:
       c, SurfaceInline, 422, …)         banner, inline, inline-full
                                        X-Error-Fallback-Surface:
        │                                 banner
        ▼
   +----------------------------------------------------------------+
   |  e.HTTPErrorHandler = handler.NewHTTPErrorHandler              |
   |    1. middleware.RestoreRawWriter(c)                            |
   |    2. promolog.Storer.Promote(...)  trace store                 |
   |    3. dispatch by error type:                                   |
   |         *SurfaceError         → renderSurfaceError              |
   |         *linkwell.HTTPError   → handleErrorWithContext          |
   |         *echo.HTTPError / err → handleError                     |
   +----------------------------------------------------------------+
        │
        │  htmxutil.IsHTMX(r)?
        ├── no  ─▶  render surface directly:
        │            Page     → handler.RenderBaseLayout (host chrome)
        │            Document → standalone HTML shell
        │            others   → write fragment to response
        │
        └── yes ─▶  ParseErrorCapabilities(c) → NegotiateSurface
                      +-----------------------------------------+
                      |  chosen ∈ Accept  → render at chosen    |
                      |  Fallback present → render at fallback  |
                      |  otherwise        → SurfaceBanner (OOB) |
                      +-----------------------------------------+
                                       │
                                       ▼
                +-------------+-----------------------------------+
                | banner      | OOB swap into #error-status        |
                | inline      | caller-targeted in-flow fragment   |
                | inline-full | caller-targeted container fragment |
                | page        | non-HTMX surface → collapses to    |
                |             |   banner OOB on HTMX requests       |
                | document    | non-HTMX surface → collapses to    |
                |             |   banner OOB on HTMX requests       |
                +-------------+-----------------------------------+
```

### Surface Taxonomy

`web/components/core/error_presentation.go` defines five `ErrorSurface` values:

| Surface | Where it renders | Typical use |
|---------|------------------|-------------|
| `SurfaceBanner` | OOB swap into `#error-status`, dismissable | Background failures, HTMX defaults |
| `SurfaceInline` | Caller-targeted fragment, in-flow | Form validation rerender (422) |
| `SurfaceInlineFull` | Caller-targeted fragment, container-sized | Wider in-page workflow conflicts |
| `SurfacePage` | Composed inside the host `AppNavLayout` | In-chrome resource pages (404 with nav + breadcrumbs) |
| `SurfaceDocument` | Standalone HTML shell, no host chrome | Auth/system boundary, last-resort renders |

### Surface-Carrying Errors

Routes return `*handler.SurfaceError` from `handler.NewSurfaceError(c, surface, status, title, detail, err, controls…)`. The constructor stamps route/request-ID and defaults the control set (500+ adds a Report Issue button). `NewSurfaceError(...).WithBody(component)` lets the route supply a richer in-chrome body (e.g. NotFoundPage's resource grid) without bypassing the central pipeline.

Three error types reach the central handler:

| Type | Source | Handling |
|------|--------|----------|
| `*handler.SurfaceError` | `handler.NewSurfaceError(...)` | Renders at the chosen surface (page/document compose layout; banner/inline write fragment) |
| `*linkwell.HTTPError` | `handler.HypermediaError(...)` and helpers | Banner OOB for HTMX, Document for non-HTMX |
| `*echo.HTTPError` / generic `error` | Echo framework / unhandled | Default banner-or-document path |

### Capability Negotiation

HTMX clients can advertise per-request which surfaces the triggering DOM context can honestly accept:

```
X-Error-Accept-Surfaces:  banner, inline, inline-full
X-Error-Fallback-Surface: banner
```

`handler.ParseErrorCapabilities(c)` reads them; `handler.NegotiateSurface(chosen, caps)` picks the actual surface. When the server's chosen surface is in the Accept list, it wins. Otherwise the client's Fallback applies; if there's no Fallback, the safe default is `SurfaceBanner`. `SurfacePage` and `SurfaceDocument` are non-HTMX by design — an HTMX render targeting them collapses to the banner OOB swap (documented degrade rule, not a silent error). Clients that send no capability headers keep the pre-negotiation behavior (`handleErrorWithContext` banner OOB).

### Error Controls

`linkwell.ErrorControlsForStatus(statusCode, opts)` returns the default control set:

| Status | Controls |
|--------|----------|
| 400 | Go back, home |
| 401 | Login, home |
| 403 | Go back, home |
| 404 | Go back, home, search |
| 500+ | Retry, home, **Report Issue** |

The "Report Issue" button opens a modal pre-populated with the request ID; the trace store (see below) supplies the captured request log.

### Error Trace Promotion

When `reqLogStore` is non-nil, the error handler promotes the per-request log buffer to the shared store via `promolog.Storer.Promote()`. This allows the error report modal to retrieve the full request log by request ID. The promotion runs before the response is rendered so even render failures leave a trace.

## Security

### Authentication (crooner)

OIDC/OAuth2 with PKCE flow. Crooner manages the login/callback/logout routes and puts identity on the request context.

### Authorization (dorman)

`dorman.RequireAuth` rejects unauthenticated requests (401). `dorman.RequireRole` / `dorman.RequireAnyRole` enforce role-based access (403). Identity is read from context via `dorman.GetIdentity(r)`.

### CSRF (dorman)

`dorman.CSRFProtect` implements double-submit cookie with HMAC-SHA256 and one-time-pad masking (BREACH protection). Token injected via `dorman.GetToken(r)` → `<meta name="csrf-token">` → HTMX configRequest listener.

### Security Headers (dorman)

`dorman.SecurityHeaders` sets X-Frame-Options, X-Content-Type-Options, Referrer-Policy, Permissions-Policy, Cross-Origin-Opener-Policy, and optionally HSTS and CSP.

## File Organization

```
internal/
├── config/          App configuration (env vars)
├── demo/            Demo SQLite database, seed data, domain models
├── domain/          Core domain types
├── env/             Environment loading (-env flag, .env.{mode} files)
├── health/          Health endpoint, runtime stats collection
├── htmxutil/        Local HTMX transport seam over htmx-go
├── logger/          Structured logging setup (slog + promolog)
├── repository/      Database repository implementations
├── responsepolicy/  Always-on web-standards layer (103 preload, Server-Timing, security, Vary)
├── routes/
│   ├── handler/         Surface contract: NewHTTPErrorHandler, NewSurfaceError,
│   │                    capability negotiation, layout rendering, breadcrumbs
│   ├── middleware/      Echo middleware (link-relations, raw-writer save)
│   ├── params/          Request parameter parsing
│   ├── routes.go        InitEcho, InitRoutes, NewAppRoutes (one chain, one place)
│   ├── examples.go      Scaffold-facing /examples hub + initExamplesRoutes
│   ├── error_scenarios.go Scaffold-facing /examples/error-scenarios teaching surface
│   ├── forms.go         Scaffold-facing /examples/forms (HTMX validation demo)
│   ├── links.go         Demo-only Hub/Ring/Link declarations
│   └── *.go             One file per feature area
├── session/         Per-session settings middleware and types
├── setup/           Feature flag stripping, template setup logic
└── version/         Build version info

web/
├── assets/public/   Static assets (CSS, JS, images, fonts)
├── components/
│   └── core/        Reusable templ components
│       ├── context_bar.templ        Local context bar (immediate ring siblings + hub parent)
│       ├── controls.templ           Hypermedia control buttons
│       ├── csrf.templ               CSRF meta tag + HTMX listener
│       ├── error_presentation.go    Surface taxonomy + presentation contract
│       ├── error_presentation.templ Renderer for the five error surfaces
│       ├── filter.templ             Filter bar for tables
│       ├── form.templ               Form controls with validation
│       ├── modal.templ              Dialog-based modals
│       ├── nav.templ                Navigation bar
│       ├── resource_card.templ      Discovery-page tile
│       ├── resource_grid.templ      Responsive grid wrapper
│       ├── sitemap.templ            Site map footer (renders the linkwell hub tree)
│       └── table.templ              Sortable table with pagination
└── views/           Page-level templ templates
    ├── examples_index.templ     /examples discovery page
    ├── error_scenarios.templ    /examples/error-scenarios pages
    ├── forms.templ              /examples/forms (server-driven HTMX validation)
    ├── inventory.templ          Demo: table page
    ├── layout.templ             AppNavLayout / RenderBaseLayout
    └── *.templ                  One file per page
```

### Naming Conventions

- **Route files**: `<feature>.go` — one file per feature area (inventory, people, kanban, etc.)
- **View files**: `<feature>.templ` — page-level templates matching route files
- **Component files**: `<component>.templ` in `web/components/core/` — reusable across pages
- **Feature gates**: `// setup:feature:TAG:start` / `// setup:feature:TAG:end` for block removal, `// setup:feature:TAG` for whole-file removal
