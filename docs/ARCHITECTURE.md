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

Every HTTP request passes through Echo's middleware chain in this order:

```
Request
  │
  ├─ 103 Early Hints (preload CSS/JS via informational response)
  ├─ Server-Timing (measures total request duration → Server-Timing header)
  ├─ Correlation ID (promolog.CorrelationMiddleware → X-Request-ID)
  ├─ Request Logger (structured access log)
  ├─ Recover (panic recovery)
  ├─ Security Headers (dorman.SecurityHeaders — X-Frame-Options, HSTS, Permissions-Policy, etc.)
  ├─ Compression (zstd/brotli/gzip via httpcompression; skipped behind templ proxy)
  ├─ Session (crooner/SCS — loads session, wraps LoadAndSave)
  ├─ Auth (crooner — OAuth/OIDC flow, login redirect)
  ├─ CSRF (dorman.CSRFProtect — Sec-Fetch-Site fast-path, HMAC-SHA256 fallback)
  ├─ Session Settings (loads per-session preferences → request context)
  ├─ Link Relations (resolves linkwell.LinksFor(path) → echo context + Link HTTP header)
  ├─ Vary: HX-Request header
  │
  └─ Handler
       ├─ RenderBaseLayout(c, component) — full page
       └─ RenderComponent(c, component) — HTMX fragment
```

Middleware is registered in `InitEcho()` (`internal/routes/routes.go`). Each middleware is feature-gated with `// setup:feature:TAG` markers, so derived apps only include what they select during setup.

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

### Registration

Links are registered at startup in `links.go` using three primitives:

| Primitive | Semantics | Example |
|-----------|-----------|---------|
| `linkwell.Hub(center, title, spokes...)` | Parent→children. Center gets `rel="related"` to each spoke. Each spoke gets `rel="up"` to center. | `Hub("/demo", "Demo", Rel("/demo/inventory", "Inventory"))` |
| `linkwell.Ring(name, members...)` | Symmetric peers. Every member gets `rel="related"` to every other member, grouped by ring name. | `Ring("Data", Rel("/demo/inventory", "Inventory"), Rel("/demo/catalog", "Catalog"))` |
| `linkwell.Link(source, rel, target, title)` | Pairwise. `rel="related"` auto-creates the inverse. | `Link("/settings", "related", "/admin/config", "Admin Config")` |

A page can belong to multiple rings and one hub. The registry deduplicates automatically.

### Middleware Resolution

`LinkRelationsMiddleware` (`internal/routes/middleware/links.go`) runs on every request:

1. Calls `linkwell.LinksFor(path)` to get all registered relations for the current path (walks parent paths if no exact match).
2. Sets the `Link` HTTP header (RFC 8288 format).
3. Stores links on the echo context for template rendering.

### Stored Links

Links can also be loaded from the database at startup via `linkwell.LoadStoredLink()`. The demo DB stores link relations in a `stored_links` table, loaded during `InitRoutes()`.

**Key files:**
- `internal/routes/links.go` — all Hub/Ring/Link declarations
- `internal/routes/middleware/links.go` — `LinkRelationsMiddleware()`

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

## Error Handling

Errors are hypermedia responses with navigation controls, not dead ends.

### Error Handler

`middleware.NewHTTPErrorHandler()` (`internal/routes/middleware/error_handler.go`) is assigned to `e.HTTPErrorHandler`. It handles three error types:

| Type | Source | Handling |
|------|--------|----------|
| `linkwell.HTTPError` | `handler.HandleHypermediaError()` | Rich error with custom controls |
| `echo.HTTPError` | Echo framework | Converted to HTML with default controls |
| Generic `error` | Unhandled errors | 500 with generic controls |

### HTMX vs. Full Page

- **HTMX requests**: Error is delivered as an OOB swap to `#error-status` — a dismissible banner.
- **Full page requests**: Error renders as a complete HATEOAS error page with navigation controls.

### Error Controls

`linkwell.ErrorControlsForStatus(statusCode, opts)` returns appropriate actions:

| Status | Controls |
|--------|----------|
| 400 | Go back, home |
| 401 | Login, home |
| 403 | Go back, home |
| 404 | Go back, home, search |
| 500+ | Retry, home, report issue |

The "Report Issue" button opens a modal that captures the request ID and log buffer for debugging.

### Error Trace Promotion

When `reqLogStore` is non-nil, the error handler promotes the per-request log buffer to the shared store via `promolog.Storer.Promote()`. This allows the error report modal to retrieve the full request log by request ID.

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
├── logger/          Structured logging setup (slog + promolog)
├── repository/      Database repository implementations
├── routes/
│   ├── handler/     Layout rendering, breadcrumbs, error helpers
│   ├── middleware/   Echo middleware (session, links, errors, timing, correlation)
│   ├── params/      Request parameter parsing
│   ├── routes.go       InitEcho, InitRoutes, NewAppRoutes
│   ├── links.go        Hub/Ring/Link declarations
│   ├── inventory.go    Example: table with CRUD
│   └── *.go            One file per feature area
├── session/         Per-session settings middleware and types
├── setup/           Feature flag stripping, template setup logic
└── version/         Build version info

web/
├── assets/public/   Static assets (CSS, JS, images, fonts)
├── components/
│   └── core/        Reusable templ components
│       ├── context_bar.templ  Context bar (grouped related links)
│       ├── controls.templ     Hypermedia control buttons
│       ├── csrf.templ         CSRF meta tag + HTMX listener
│       ├── filter.templ       Filter bar for tables
│       ├── form.templ         Form controls with validation
│       ├── modal.templ        Dialog-based modals
│       ├── nav.templ          Navigation bar
│       ├── table.templ        Sortable table with pagination
│       └── sitemap.templ      Site map footer
└── views/           Page-level templ templates
    ├── inventory.templ   Example: table page
    ├── layout.templ      Index and AppNavLayout wrappers
    └── *.templ           One file per page
```

### Naming Conventions

- **Route files**: `<feature>.go` — one file per feature area (inventory, people, kanban, etc.)
- **View files**: `<feature>.templ` — page-level templates matching route files
- **Component files**: `<component>.templ` in `web/components/core/` — reusable across pages
- **Feature gates**: `// setup:feature:TAG:start` / `// setup:feature:TAG:end` for block removal, `// setup:feature:TAG` for whole-file removal
