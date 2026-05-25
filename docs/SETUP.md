# Setup Guide

Dothog is a template. Running `mage setup` scaffolds a new application by rewriting module paths, configuring ports, and stripping features you don't need.

## Quick Start

```bash
# Interactive wizard (recommended)
go tool mage setup

# CLI flags
go tool mage setup -n "My App" -m "github.com/you/my-app" -p 12345
go tool mage setup -n "My App" --features auth,sse
go tool mage setup -n "My App" --features auth,mssql,sse   # MSSQL app data
go tool mage setup -n "My App" --features none  # bare HTMX app

# Cross-host generation (autodetects from runtime.GOOS when omitted)
go tool mage setup -n "My App" --platform linux
go tool mage setup -n "My App" --platform windows
```

## Host Platforms

Setup supports `linux` and `windows` as derived-app hosts. When `--platform`
is omitted, setup autodetects from `runtime.GOOS`; unsupported hosts (for
example `darwin`) are rejected with a clear error instead of producing a
silently broken scaffold. The platform decision shapes the static dev-tooling
configs that differ between the two hosts:

- `.air/server.toml` — `bin`/`cmd` reference `./tmp/main` on Linux,
  `./tmp/main.exe` on Windows; `cmd = "go tool mage airBuild"` on both so
  Air rebuilds stay repo-local and avoid shell-only operators.
- `.air/lint.toml` — `bin = "/bin/echo"` on Linux, `bin = "cmd"` plus
  `args_bin = ["/c", "exit"]` on Windows so Air has a real exec target.
- Generated README's "From Source" block shows `./<binary>` (Linux) or
  `.\<binary>.exe` (Windows).

`magefile.go` itself uses `runtime.GOOS` at runtime (Tailwind binary path,
dev binary extension, `Build`/`Compile`/`Run` output), so the same generated
file works on either host.

## What Setup Does

1. **Rewrites module path** — replaces `catgoose/dothog` with your module path in all `.go` files and `go.mod`
2. **Configures ports** — sets `APP_HTTP_PORT`, `TEMPL_HTTP_PORT` (base+1), `CADDY_TLS_PORT` (base+2) in `.env.development`, `magefile.go`, and Caddyfile when present
3. **Sets app name** — updates binary name in `magefile.go`, Dockerfile, logger, package.json
4. **Strips features** — removes code blocks and files for features you didn't select
5. **Generates README** — from `_template_setup/README.template.md` with your app name and ports
6. **Uses Caddy's internal CA** — local HTTPS comes from `tls internal` when Caddy is selected; no `openssl` prerequisite or scaffolded cert files
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

**CSRF feature** — wraps the CSRF block, which gates dorman.CSRFProtect middleware setup. Stripped when `csrf` is not selected.

### Available Features

| Tag | Label | Dependencies | Description |
|-----|-------|-------------|-------------|
| `session_settings` | Session Settings | — | Per-session theme/layout preferences in SQLite |
| `csrf` | CSRF Protection | — | CSRF middleware via gorilla/csrf |
| `auth` | Auth (Crooner) | — | OAuth/OIDC authentication via crooner |
| `graph` | Graph API | — | Microsoft Graph API integration |
| `avatar` | Avatar Photos | — | User avatar fetching (requires graph selected separately) |
| `mssql` | MSSQL (Microsoft SQL Server) | — | Chuck-backed app data with MSSQL support |
| `postgres` | PostgreSQL | — | Chuck-backed app data with PostgreSQL support |
| `sse` | SSE | caddy (hidden) | Server-Sent Events. Auto-includes the hidden Caddy HTTPS/H3 dev proxy. |
| `capacitor` | Capacitor | — | Capacitor mobile wrapper |
| `demo` | Demo Content | session_settings | Demo pages, SQLite seed data, example routes (auto-includes session_settings) |

### Implicit Features

`alpine` is always included and not presented in the wizard. `_hyperscript` is the default tool for client-side DOM behavior (loaded with HTMX); Alpine.js (CSP build) is kept available for coordinated view state and browser-API bridges, currently the theme picker. The CSP build eliminates `unsafe-eval` from Content Security Policy requirements; any remaining Alpine component is registered via `Alpine.data()` in `alpine-components.js`.

The Chuck-backed app-data layer still exists as an internal setup marker for stripping/bookkeeping, but it is not a user-facing feature choice. Select `mssql` or `postgres` when you want app-data support for one of those engines.

### Feature Dependencies

Dependencies are auto-resolved (closed by `setup.ExpandFeatureDeps` before stripping):
- Selecting `avatar` auto-includes `graph` — avatar code imports the Graph package directly.
- Selecting `mssql` or `postgres` auto-includes the hidden `database` tag — the chuck-backed app-data layer ships with the production engine.
- Selecting `sse` auto-includes the hidden `caddy` tag — the dev HTTPS/H3 front-proxy ships alongside the SSE broker. There is no supported "Caddy without SSE" setup shape.
- Selecting `demo` auto-includes `session_settings` — demo content reads session settings.

Web-standards behavior (Server-Timing, `Vary: HX-Request`, Permissions-Policy via dorman, 103 Early Hints) and the link-relations registry (context bars, breadcrumbs, footer site map, `Link` headers) are always-on baseline behavior owned by `internal/responsepolicy` and `linkwell` respectively; neither is a user-selectable feature. The registry has two seams: scaffold-facing relations are declared in route initializers (e.g. `internal/routes/examples.go`); demo-only relations live in `internal/routes/links.go`. Both write to the same global registry. The curated top nav (`internal/routes/handler/handler.go::appNavNavConfig`) is a separate concern from the registry — the nav lists discoverability anchors, not every page in the link graph.

## Interactive Wizard

The wizard (`mage_setup.go`) uses [huh](https://github.com/charmbracelet/huh) for the TUI:

1. **Target Directory** — choose where the derived app will be scaffolded
2. **App Configuration** — name, module path, base port
3. **Preset / Guided Flow** — pick a preset, the flat checklist, or the guided questions
4. **Optional App Data Support** — in guided mode, choose `mssql`, `postgres`, or neither
5. **Feature Questions / Checklist** — guided follow-up questions or the flat multi-select
6. **Replacement Prompt** — if the target directory already exists and is non-empty, setup asks whether to remove and replace it

Setup always scaffolds into the target directory rather than mutating the current template checkout. The copy excludes transient/template-only paths such as `.git`, `bin`, `build`, `tmp`, `_template_setup/`, `internal/setup/`, and `mage_setup.go`.

## CLI Flags

```
-n APP_NAME        Human-readable app name (required)
-m MODULE_PATH     Go module path (default: github.com/you/<app-name>)
-p BASE_PORT       5-digit base port < 60000
--features LIST    Comma-separated: auth,graph,avatar,mssql,postgres,sse,demo,session_settings,csrf,capacitor
                   "all" = keep everything, "none" = bare HTMX app
                   Hidden tags resolve via featureDeps: mssql/postgres imply database; sse implies the Caddy HTTPS/H3 dev proxy.
--force            Re-run setup on an already-customized module
```

## Post-Setup Cleanup

After setup, the wizard offers to remove template-specific files:
- `_template_setup/` — README template and setup assets
- `internal/setup/` — setup package (feature stripping logic)
- `mage_setup.go` — the setup mage target itself

These files are only needed for running setup. Derived apps don't need them.

## Derived App Layout

After `mage setup` with `--features auth,sse`:

- All `// setup:feature:demo` files are deleted
- All `// setup:feature:demo:start` ... `// setup:feature:demo:end` blocks are removed
- Remaining code compiles cleanly with `go mod tidy`
- The app has auth, SQLite, the SSE broker, the Caddy HTTPS/H3 dev proxy (pulled in by sse), and a clean starting point for your own routes

## Adding App Data

App-owned database tables are registered through `internal/dbschema`. The
package ships empty — derived apps extend `Tables()` with their own
`schema.TableDef`s and the runtime wires them into the chuck-backed
`schema.Materializer` automatically. The package itself only ships when the
derived app selects MSSQL or PostgreSQL during setup; scaffolds without
app-data support never see it. Framework-owned tables (session settings,
graph user cache, error traces) live in their own packages and are not
registered here.

1. Define a table next to the code that owns it (for example
   `internal/orders/schema.go`):

   ```go
   package orders

   import "<module>/internal/database/schema"

   var Table = schema.NewTable("Orders").
       Columns(
           schema.AutoIncrCol("Id"),
           schema.Col("CustomerId", schema.TypeInt()).NotNull(),
           schema.Col("Total", schema.TypeInt()).NotNull(),
       ).
       WithTimestamps()
   ```

2. Register the table in `internal/dbschema/dbschema.go`:

   ```go
   func Tables() []*schema.TableDef {
       return []*schema.TableDef{
           orders.Table,
       }
   }
   ```

3. Build a concrete repository that holds `*sqlx.DB` directly. Run
   single-statement queries via `GetContext` / `ExecContext` /
   `SelectContext`; use `database.WithTransaction(ctx, dbx, fn)` for
   multi-statement work. The chuck `dbrepo` package provides query-builder
   helpers (`NewSelect`, `NewWhere`, `SetClause`, `InsertInto`):

   ```go
   package orders

   import (
       "context"
       "database/sql"
       "fmt"

       "github.com/catgoose/chuck/dbrepo"
       "github.com/jmoiron/sqlx"
   )

   type Repository struct {
       db *sqlx.DB
   }

   func NewRepository(db *sqlx.DB) *Repository {
       return &Repository{db: db}
   }

   func (r *Repository) GetByID(ctx context.Context, id int) (*Order, error) {
       w := dbrepo.NewWhere().And("Id = @Id", sql.Named("Id", id))
       query, args := dbrepo.NewSelect("Orders", dbrepo.Columns("Id", "CustomerId", "Total")).Where(w).Build()
       var o Order
       if err := r.db.GetContext(ctx, &o, query, args...); err != nil {
           return nil, fmt.Errorf("get order: %w", err)
       }
       return &o, nil
   }
   ```

   For multi-statement work, take a `*sqlx.Tx`:

   ```go
   err := database.WithTransaction(ctx, dbx, func(ctx context.Context, tx *sqlx.Tx) error {
       if _, err := tx.ExecContext(ctx, "...", ...); err != nil {
           return err
       }
       return nil
   })
   ```

   `internal/repository/session_settings_repository.go` is the
   framework-owned reference implementation of this pattern.

`EnsureSchema` runs on every startup and is non-destructive. The template does
not wire any built-in env flag for destructive `InitSchema` resets; derived
apps that want that behavior must add their own gating logic.

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
