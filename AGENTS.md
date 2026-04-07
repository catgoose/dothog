# Dothog Documentation Index

## Architecture & Philosophy
- [PHILOSOPHY.md](PHILOSOPHY.md) — Design principles, web standards strategy, hypermedia architecture
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — System architecture, request lifecycle, component interactions
- [MANIFESTO.md](MANIFESTO.md) — Project manifesto and guiding texts

## Development
- [docs/COMPONENTS.md](docs/COMPONENTS.md) — Component catalog, page patterns, how to build common views
- [docs/SETUP.md](docs/SETUP.md) — Scaffold guide, feature flags, setup wizard
- [docs/LINK_RELATIONS.md](docs/LINK_RELATIONS.md) — IANA link relations reference and cookbook

## Codebase
- `internal/routes/` — HTTP handlers and route registration
- `internal/routes/handler/` — Layout rendering, breadcrumb resolution
- `internal/routes/hypermedia/` — Link registry (Ring/Hub/Link), controls, navigation
- `internal/routes/middleware/` — Echo middleware (CSRF, session, links, server-timing)
- `web/views/` — Templ page templates
- `web/components/core/` — Reusable templ components (context bar, nav, table, filter, modal)
- `web/assets/public/` — Static assets (CSS, JS, images)
- `internal/demo/` — Demo SQLite database and seed data

## Key Concepts
- **Link Relations**: Ring (peers), Hub (parent→children), Link (pairwise). Drives context bars, breadcrumbs, site map.
- **Feature Flags**: `// setup:feature:TAG` markers. Stripped during `mage setup` for derived apps.
- **Layouts**: Index (classic top nav) and AppNavLayout (responsive mobile-first). Togglable in settings.
- **Breadcrumb Priority**: ?from= (user journey) → rel="up" (declared hierarchy) → URL path (fallback)

## Ecosystem Libraries

| Library | Purpose | Import |
|---------|---------|--------|
| [catgoose/chuck](https://github.com/catgoose/chuck) | SQL dialects, schema DSL, query builders | `"github.com/catgoose/chuck"` |
| [catgoose/promolog](https://github.com/catgoose/promolog) | Error trace capture, promote-on-error | `"github.com/catgoose/promolog"` |
| [catgoose/crooner](https://github.com/catgoose/crooner) | OIDC authentication | `"github.com/catgoose/crooner"` |
| [catgoose/dorman](https://github.com/catgoose/dorman) | Auth, CSRF, security headers | `"github.com/catgoose/dorman"` |
| [catgoose/linkwell](https://github.com/catgoose/linkwell) | HATEOAS controls, navigation, link registry | `"github.com/catgoose/linkwell"` |
| [catgoose/tavern](https://github.com/catgoose/tavern) | SSE pub/sub broker | `"github.com/catgoose/tavern"` |

## For AI Agents Working on Derived Apps

- **Config**: Flat `AppConfig` struct, singleton via `config.GetConfig()`. Extend by adding fields and reading env vars in `buildConfig()`.
- **Database**: `dialect.OpenURL(ctx, dsn)` for app databases. `dialect.OpenSQLite(ctx, path)` for framework internals.
- **Layout**: Override with `handler.SetLayout(fn)` to use your own page wrapper.
- **Auth**: Set `OIDC_ISSUER_URL` + `OIDC_CLIENT_ID` to enable. Works with any OIDC provider.
- **Feature gates**: `// setup:feature:X:start` / `// setup:feature:X:end` for blocks. `// setup:feature:X` as first line for whole files.
- Don't commit changes back to the dothog repo for app-specific code.
- Don't add dothog dependencies to derived apps — dothog is a starting point, not an upstream.

## Reporting Issues Upstream

If you find a bug in the template itself (not app-specific code), check if it's already tracked, then open an issue at the dothog GitHub repository. Include: feature configuration used, the error, and minimal reproduction.

Common upstream issues:
- Feature stripping leaves broken references (missing `// setup:feature:*` markers)
- Tests reference types from stripped features
- Config fields used outside their feature gate
- Generated `_templ.go` files have stale imports after stripping
