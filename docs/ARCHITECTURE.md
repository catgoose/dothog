# Architecture

This document describes how dothog processes requests, resolves navigation, and renders pages. For design principles and rationale, see [PHILOSOPHY.md](../PHILOSOPHY.md).

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
  ├─ Secure (security headers: X-Frame-Options, X-Content-Type-Options, etc.)
  ├─ Permissions-Policy + COOP headers
  ├─ Gzip (skipped when behind templ proxy; Caddy handles compression)
  ├─ Session (crooner/SCS — loads session, wraps LoadAndSave)
  ├─ Auth (crooner — OAuth/OIDC flow, login redirect)
  ├─ CSRF (double-submit token, per-request rotation on configured paths)
  ├─ Session Settings (loads shared settings row from SQLite → echo context)
  ├─ Link Relations (resolves LinksFor(path) → echo context + Link HTTP header)
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
4. Layout checks `settings.Layout` — if `domain.LayoutApp`, renders `AppNavLayout`; otherwise renders the classic `Index` layout.
5. Templ renders HTML and writes the response.

**Key files:**
- `internal/routes/routes.go` — `InitEcho()`, `InitRoutes()`
- `internal/routes/handler/handler.go` — `RenderBaseLayout`, `getLayoutCtx`, `renderDefaultLayout`

## Link Relations System

The link registry is the navigation topology of the application. All context bars, breadcrumbs, and the site map footer derive from it.

### Registration

Links are registered at startup in `routes_links.go` using three primitives:

| Primitive | Semantics | Example |
|-----------|-----------|---------|
| `Hub(center, title, spokes...)` | Parent→children. Center gets `rel="related"` to each spoke. Each spoke gets `rel="up"` to center. | `Hub("/demo", "Demo", Rel("/demo/inventory", "Inventory"))` |
| `Ring(name, members...)` | Symmetric peers. Every member gets `rel="related"` to every other member, grouped by ring name. | `Ring("Data", Rel("/demo/inventory", "Inventory"), Rel("/demo/catalog", "Catalog"))` |
| `Link(source, rel, target, title)` | Pairwise. `rel="related"` auto-creates the inverse. | `Link("/settings", "related", "/admin/config", "Admin Config")` |

A page can belong to multiple rings and one hub. The registry deduplicates automatically.

### Middleware Resolution

`LinkRelationsMiddleware` (`internal/routes/middleware/links.go`) runs on every request:

1. Calls `hypermedia.LinksFor(path)` to get all registered relations for the current path.
2. Sets the `Link` HTTP header (RFC 8288 format).
3. Stores links on the echo context for template rendering.

### Stored Links

Links can also be loaded from the database at startup via `hypermedia.LoadStoredLink()`. The demo DB stores link relations in a `stored_links` table, loaded during `InitRoutes()`.

**Key files:**
- `internal/routes/routes_links.go` — all Hub/Ring/Link declarations
- `internal/routes/hypermedia/links.go` — `Ring()`, `Hub()`, `Link()`, `LinksFor()`, registry internals
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

When a user navigates via a link that includes `?from=N`, the bitmask encodes which pages they came through. `hypermedia.ResolveFromMask(mask)` decodes the bitmask into breadcrumb entries. Origins are registered at startup via `hypermedia.RegisterFrom()`.

### Priority 2: `rel="up"` chain (declared hierarchy)

`hypermedia.BreadcrumbsFromLinks(path)` walks the `rel="up"` chain: current page → parent → grandparent → Home. This produces breadcrumbs like `Home > Demo > Inventory > Item Name`. Cycle detection prevents infinite loops.

### Priority 3: URL path segments (fallback)

`buildPathCrumbs(path, from, routes)` splits the URL into segments. Only segments with a registered GET route produce linked breadcrumbs. The terminal segment is always shown (unlinked).

### Page Labels

`handler.SetPageLabel(c, label)` overrides the terminal breadcrumb label. Detail page handlers use this to show the resource name (e.g., "Widget A" instead of "42").

### Boosted Navigation

`hx-boost` navigation sends full-page requests with the `HX-Boosted` header. Handlers check `hx.IsBoosted(c)` to decide whether to render a full layout or just a fragment.

## Session Settings

Session settings provide per-session preferences (theme, layout choice) stored in SQLite.

### Storage

- `domain.SessionSettings` struct: UUID, Theme, Layout, CreatedAt, UpdatedAt
- SQLite repository implementing `SessionSettingsProvider` interface
- All visitors share a single row (shared UUID) for the demo

### Middleware

`SessionSettingsMiddleware` (`internal/routes/middleware/session_settings.go`):
1. Loads settings by shared UUID via `repo.GetByUUID()`
2. Falls back to `domain.NewDefaultSettings()` on error or missing row
3. Auto-creates the row if it doesn't exist
4. Touches the row if last update was > 24 hours ago
5. Stores settings on the echo context via `c.Set()`

### Handlers

- `POST /settings/theme` — updates the theme (dark/light/etc.)
- `POST /settings/layout` — toggles between classic Index and AppNavLayout
- Both return updated page fragments for HTMX swap

## SSE System

Server-Sent Events provide real-time updates without polling.

### SSEBroker

`internal/ssebroker/ssebroker.go` implements topic-based pub/sub:

- `NewSSEBroker()` — creates a broker instance
- `Subscribe(topic)` — returns a read channel and unsubscribe function
- `Publish(topic, data)` — sends to all subscribers on a topic
- `HasSubscribers(topic)` — checks if anyone is listening (avoids rendering unused fragments)

### Wiring

1. `routes.go` creates a single `ssebroker.NewSSEBroker()` instance.
2. Route initializers receive the broker (e.g., `initPeopleRoutes(db, broker, actLog)`).
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
| `hypermedia.HTTPError` | `handler.HandleHypermediaError()` | Rich error with custom controls |
| `echo.HTTPError` | Echo framework | Converted to HTML with default controls |
| Generic `error` | Unhandled errors | 500 with generic controls |

### HTMX vs. Full Page

- **HTMX requests**: Error is delivered as an OOB swap to `#error-status` — a dismissible banner.
- **Full page requests**: Error renders as a complete HATEOAS error page with navigation controls.

### Error Controls

`hypermedia.ErrorControlsForStatus(statusCode, opts)` returns appropriate actions:

| Status | Controls |
|--------|----------|
| 400 | Go back, home |
| 401 | Login, home |
| 403 | Go back, home |
| 404 | Go back, home, search |
| 500+ | Retry, home, report issue |

The "Report Issue" button opens a modal that captures the request ID and log buffer for debugging.

### Error Trace Promotion

When `reqLogStore` is non-nil, the error handler promotes the per-request log buffer to the shared store. This allows the error report modal to retrieve the full request log by request ID.

## File Organization

```
internal/
├── config/          App configuration (env vars, feature flags)
├── demo/            Demo SQLite database, seed data, domain models
├── domain/          Core domain types (SessionSettings, etc.)
├── health/          Health check endpoint logic
├── logger/          Structured logging setup
├── routes/
│   ├── handler/     Layout rendering, breadcrumbs, error helpers
│   ├── htmx/        HTMX request helpers (IsBoosted, IsHTMX, etc.)
│   ├── hypermedia/  Link registry, controls, navigation, error types
│   ├── middleware/   Echo middleware (CSRF, session, links, errors, timing)
│   ├── params/      Request parameter parsing
│   ├── response/    Response builder (OOB swaps, retarget, etc.)
│   ├── routes.go           InitEcho, InitRoutes, NewAppRoutes
│   ├── routes_links.go     Hub/Ring/Link declarations
│   ├── routes_inventory.go Example: table with CRUD
│   └── routes_*.go         One file per feature area
├── setup/           Feature flag stripping, template setup logic
├── ssebroker/       Topic-based pub/sub for SSE
└── version/         Build version info

web/
├── assets/public/   Static assets (CSS, JS, images, fonts)
├── components/
│   └── core/        Reusable templ components
│       ├── context_bar.templ  Context bar (grouped related links)
│       ├── controls.templ     Hypermedia control buttons
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

- **Route files**: `routes_<feature>.go` — one file per feature area (inventory, people, kanban, etc.)
- **View files**: `<feature>.templ` — page-level templates matching route files
- **Component files**: `<component>.templ` in `web/components/core/` — reusable across pages
- **Feature gates**: `// setup:feature:TAG:start` / `// setup:feature:TAG:end` for block removal, `// setup:feature:TAG` for whole-file removal
