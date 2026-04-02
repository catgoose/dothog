# Ecosystem

Dothog is built on six Go libraries, each with a single responsibility and zero overlap. All are under [github.com/catgoose](https://github.com/catgoose) and share the same conventions: standard `func(http.Handler) http.Handler` middleware signatures, interface-driven extensibility, and zero or minimal external dependencies.

## Libraries

### [chuck](https://github.com/catgoose/chuck) — SQL

Multi-dialect SQL schema builder and query fragments. One schema definition drives DDL generation, column lists, seed data, and schema snapshots across SQLite, PostgreSQL, and MSSQL. Composable query helpers (`WhereBuilder`, `SelectBuilder`) keep SQL visible without writing it by hand every time.

### [promolog](https://github.com/catgoose/promolog) — Logging

Per-request log capture with promote-on-error semantics. During normal requests, log records are buffered in memory and discarded. When a request fails, the entire buffer is promoted to a store for later inspection. Zero-dependency core; SQLite store in a separate submodule.

### [crooner](https://github.com/catgoose/crooner) — Authentication

OIDC/OAuth2 client with PKCE, session management, and pluggable backends. Handles the login/callback/logout flow and puts identity on the request context. Works with any OIDC-compliant provider (Azure AD, Google, Okta, Auth0, Keycloak).

### [porter](https://github.com/catgoose/porter) — Security

Authorization, CSRF protection, and security header middleware. `RequireAuth` and `RequireRole` enforce identity and role checks. `CSRFProtect` implements double-submit cookie with HMAC-SHA256 and BREACH protection. `SecurityHeaders` sets sensible defaults for X-Frame-Options, HSTS, Referrer-Policy, Permissions-Policy, and more.

### [linkwell](https://github.com/catgoose/linkwell) — Hypermedia

HATEOAS link registry, navigation primitives, and hypermedia controls. Declare page relationships at startup with `Hub`, `Ring`, and `Link`. Query them at request time for breadcrumbs, context bars, related links, and the site map. Pure-data control types (`Control`, `FilterBar`, `PageInfo`, `ModalConfig`) that templates render — no rendering logic in the library.

### [tavern](https://github.com/catgoose/tavern) — Real-time

Thread-safe, topic-based SSE pub/sub broker. Handlers publish events when state changes; SSE endpoints fan them out to connected browsers. Supports scoped subscriptions for per-user/per-session feeds. OOB fragment helpers for HTMX out-of-band swaps over SSE. Framework-agnostic `Component` interface compatible with templ.

## How They Fit Together

```
  HTTP Request
       │
       ▼
  ┌─────────────┐
  │  porter      │  Security headers, CSRF validation
  └──────┬──────┘
         │
  ┌──────▼──────┐
  │  crooner     │  OIDC authentication, session
  └──────┬──────┘
         │
  ┌──────▼──────┐
  │  porter      │  Role checks (RequireAuth, RequireRole)
  └──────┬──────┘
         │
  ┌──────▼──────┐
  │  handler     │
  │              │
  │  chuck ────┤  Schema, queries, dialect-aware SQL
  │  promolog ───┤  Per-request log buffer
  │  linkwell ───┤  Controls, navigation, breadcrumbs
  │  tavern  ────┤  SSE publish to connected browsers
  │              │
  └──────┬──────┘
         │
         ▼
  HTML Response
```

## Design Principles

Every library follows the [dothog design philosophy](PHILOSOPHY.md):

- **The server drives state.** Libraries provide data and middleware. Templates are downstream.
- **Zero or minimal dependencies.** promolog (core), linkwell, porter, and tavern have zero runtime dependencies. chuck has only database drivers. crooner has only OIDC/OAuth2 libraries.
- **Interfaces over implementations.** `promolog.Storer`, `session.Provider`, `porter.IdentityProvider` — implement the interface, bring your own backend.
- **Standard signatures.** All middleware uses `func(http.Handler) http.Handler`. No framework lock-in.
- **The struct is the interface.** linkwell's control types are pure data. Any template engine that can read Go struct fields can render them. No `Renderer` interface needed — the data flows one way.

## Independence

The six libraries are independent leaves. None imports another. Only dothog depends on all of them. You can use any library standalone without pulling in the rest of the ecosystem.
