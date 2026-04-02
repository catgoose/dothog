# Setup Guide

Dothog is a template. Running `mage setup` scaffolds a new application by rewriting module paths, configuring ports, and stripping features you don't need.

## Quick Start

```bash
# Interactive wizard (recommended)
go tool mage setup

# CLI flags
go tool mage setup -n "My App" -m "github.com/you/my-app" -p 12345
go tool mage setup -n "My App" --features auth,database,sse,caddy
go tool mage setup -n "My App" --features none  # bare HTMX app
```

## What Setup Does

1. **Rewrites module path** — replaces `catgoose/dothog` with your module path in all `.go` files and `go.mod`
2. **Configures ports** — sets `APP_TLS_PORT`, `TEMPL_HTTP_PORT` (base+1), `CADDY_TLS_PORT` (base+2) in `.env.development`, Caddyfile, air config
3. **Sets app name** — updates binary name in `magefile.go`, Dockerfile, logger, package.json
4. **Strips features** — removes code blocks and files for features you didn't select
5. **Generates README** — from `_template_setup/README.template.md` with your app name and ports
6. **Ensures certificates** — generates self-signed TLS certs if Caddy is selected
7. **Runs `go mod tidy`** — cleans up unused dependencies after stripping

## Feature Flags

Code is tagged with `// setup:feature:TAG` markers. During setup, unselected features are stripped.

### Marker Syntax

**Whole-file removal** — first line of the file:
```go
// setup:feature:demo
package views
```
If `demo` is not selected, the entire file is deleted.

**Block removal** — wraps a code section:
```go
// setup:feature:sse:start
broker := ssebroker.NewSSEBroker()
ar.initRealtimeRoutes(broker)
// setup:feature:sse:end
```
If `sse` is not selected, everything between `:start` and `:end` (inclusive) is removed.

**CSRF feature** — wraps the CSRF block, which gates porter.CSRFProtect middleware setup. Stripped when `csrf` is not selected.

### Available Features

| Tag | Label | Dependencies | Description |
|-----|-------|-------------|-------------|
| `session_settings` | Session Settings | — | Per-session theme/layout preferences in SQLite |
| `csrf` | CSRF Protection | — | CSRF middleware via gorilla/csrf |
| `auth` | Auth (Crooner) | — | OAuth/OIDC authentication via crooner |
| `graph` | Graph API | — | Microsoft Graph API integration |
| `avatar` | Avatar Photos | — | User avatar fetching (requires graph selected separately) |
| `database` | Database | — | Implicit; SQLite is always included |
| `mssql` | MSSQL dialect | — | Microsoft SQL Server production dialect |
| `postgres` | PostgreSQL dialect | — | PostgreSQL production dialect |
| `sse` | SSE | — | Server-Sent Events (requires caddy selected separately) |
| `caddy` | Caddy (HTTPS) | — | Caddy reverse proxy with TLS |
| `link_relations` | Link Relations | — | Context bars, breadcrumbs, site map |
| `web_standards` | Web Standards | — | Server-Timing, Vary, Permissions-Policy, Early Hints |
| `browser_apis` | Browser APIs | sse | sendBeacon and BroadcastChannel support (auto-includes sse) |
| `capacitor` | Capacitor | — | Capacitor mobile wrapper |
| `offline` | Offline | — | Offline-first with service worker and write queue |
| `sync` | Sync | offline | SQLite sync between client and server (auto-includes offline) |
| `pwa` | PWA | offline, sync | Progressive Web App — offline + sync + mobile (auto-includes offline and sync) |
| `demo` | Demo Content | session_settings | Demo pages, SQLite seed data, example routes (auto-includes session_settings) |

### Implicit Features

`database` and `alpine` are always included and not presented in the wizard. SQLite is the base database engine; Alpine.js is the standard client-side state layer.

### Feature Dependencies

Dependencies are auto-resolved:
- Selecting `browser_apis` auto-includes `sse`
- Selecting `sync` auto-includes `offline`
- Selecting `pwa` auto-includes `offline` and `sync`
- Selecting `demo` auto-includes `session_settings`

## Interactive Wizard

The wizard (`mage_setup.go`) uses [huh](https://github.com/charmbracelet/huh) for the TUI:

1. **App Configuration** — name, module path, base port
2. **Feature Selection** — multi-select with preselection (all except demo/alpine)
3. **Force Confirm** — shown only if module is already customized
4. **Final Confirmation** — summary of all selections

### Copy-first Mode

The wizard asks whether to copy the template to a new directory before setup. This is the recommended approach — it leaves the original template untouched for future use. The copy:
- Excludes `.git`, `bin`, `build`, `tmp`
- Removes setup-only files (`_template_setup/`, `internal/setup/`, `mage_setup.go`)
- Optionally runs `git init` in the new directory

## CLI Flags

```
-n APP_NAME        Human-readable app name (required)
-m MODULE_PATH     Go module path (default: github.com/you/<app-name>)
-p BASE_PORT       5-digit base port < 60000
--features LIST    Comma-separated: auth,graph,avatar,database,sse,caddy,demo
                   "all" = keep everything, "none" = bare HTMX app
--no-caddy         Deprecated alias for omitting caddy from --features
--force            Re-run setup on an already-customized module
```

## Post-Setup Cleanup

After setup, the wizard offers to remove template-specific files:
- `_template_setup/` — README template and setup assets
- `internal/setup/` — setup package (feature stripping logic)
- `mage_setup.go` — the setup mage target itself

These files are only needed for running setup. Derived apps don't need them.

## Derived App Layout

After `mage setup` with `--features auth,database,caddy`:

- All `// setup:feature:demo` files are deleted
- All `// setup:feature:demo:start` ... `// setup:feature:demo:end` blocks are removed
- All `// setup:feature:sse:start` ... `// setup:feature:sse:end` blocks are removed
- Remaining code compiles cleanly with `go mod tidy`
- The app has auth, SQLite, Caddy, and a clean starting point for your own routes

## Custom Layouts

Derived apps can override the default layout:

```go
handler.SetLayout(handler.AppNavLayoutFunc(hypermedia.NavConfig{
    AppName: "My App",
    Items: []hypermedia.NavItem{
        {Label: "Home", Href: "/", Icon: "..."},
        {Label: "Dashboard", Href: "/dashboard", Icon: "..."},
    },
    MaxVisible: 5,
}))
```

Or provide a fully custom layout function:

```go
handler.SetLayout(func(c echo.Context, cmp templ.Component) error {
    // Your custom layout logic
    return handler.RenderComponent(c, myLayout(cmp))
})
```
