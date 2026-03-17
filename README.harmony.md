# Harmony

A Go + HTMX application template for building server-driven hypermedia applications.

Harmony runs as a single binary with all assets embedded. No external runtime dependencies, no configuration files required to start.

See [PHILOSOPHY.md](PHILOSOPHY.md) for the architectural principles behind the project.

<!--toc:start-->

- [Harmony](#harmony)
  - [Quick Start](#quick-start)
    - [From Release Binary](#from-release-binary)
    - [From Docker](#from-docker)
    - [From Source](#from-source)
  - [Template Setup](#template-setup)
  - [Features](#features)
  - [Tech Stack](#tech-stack)
  - [Architecture](#architecture)
  - [Hypermedia Patterns](#hypermedia-patterns)
    - [HATEOAS Error Recovery](#hateoas-error-recovery)
    - [Inline CRUD](#inline-crud)
    - [Real-time SSE](#real-time-sse)
    - [Data Views](#data-views)
    - [State and Interaction Patterns](#state-and-interaction-patterns)
  - [Schema Builder](#schema-builder)
    - [Table Traits](#table-traits)
    - [Table Types](#table-types)
    - [Column Types](#column-types)
    - [Domain Structs](#domain-structs)
    - [Repository Helpers](#repository-helpers)
    - [Where Builder](#where-builder)
    - [Select Builder](#select-builder)
    - [Seed Data](#seed-data)
    - [Schema Lifecycle](#schema-lifecycle)
  - [Project Structure](#project-structure)
  - [Development](#development)
    - [Prerequisites](#prerequisites)
    - [Running the Dev Server](#running-the-dev-server)
    - [HTTPS Development Setup](#https-development-setup)
  - [Testing](#testing)
  - [Mage Targets](#mage-targets)
  - [CI/CD Workflows](#cicd-workflows)
  - [Environment Variables](#environment-variables)
  <!--toc:end-->

## Quick Start

### From Release Binary

Download the latest release for your platform from the [Releases](../../releases) page:

```bash
# Linux
chmod +x harmony-linux-amd64
./harmony-linux-amd64

# Windows
harmony-windows-amd64.exe
```

The server starts on `http://localhost:3000` by default. Override with:

```bash
SERVER_LISTEN_PORT=8080 ./harmony-linux-amd64
```

### From Docker

```bash
docker pull ghcr.io/catgoose/harmony:latest
docker run -p 3000:3000 ghcr.io/catgoose/harmony:latest
```

Or build it yourself:

```bash
docker build -t harmony .
docker run -p 3000:3000 harmony
```

### From Source

```bash
git clone https://github.com/catgoose/harmony.git
cd harmony
go build -o harmony .
./harmony
```

## Template Setup

Harmony is designed to be used as a project template. After cloning:

1. Run the setup wizard to configure your app name, module path, and dev ports:

   ```bash
   go tool mage setup
   ```

   Supports flags for non-interactive use: `go tool mage setup -n "My App" -m "github.com/you/my-app" -p 5124`

2. Review `.env.dev` (generated from `.env.sample`) and adjust as needed.

4. Start development:

   ```bash
   go tool mage watch
   ```

## Features

- **HATEOAS Error Recovery** -- Server-driven error responses with embedded retry, fix, and alternative-action controls
- **Inline CRUD** -- Create, edit, toggle, and delete table rows in place without page reloads
- **SSE Real-time Dashboard** -- Live system stats via Server-Sent Events with OOB swaps
- **Interactive Data Views** -- Sorting, filtering, debounced search, pagination, and bulk operations
- **State Patterns** -- Counters, toggles, auto-load, lazy reveal, live preview, append-without-replace
- **Infinite Scroll** -- Sentinel-driven auto-loading with `hx-trigger="revealed"`
- **Optimistic UI** -- Immediate visual feedback via HyperScript with server reconciliation
- **Undo / Soft Delete** -- Delete with OOB undo toast, auto-dismiss timer, and one-click restore
- **Hypermedia Controls Gallery** -- Buttons, modals, dismiss, confirmation dialogs, and form patterns
- **Schema Builder** -- Composable DDL API with traits, query builder, and multi-dialect support (SQLite/MSSQL)

## Tech Stack

| Component | Purpose |
| --- | --- |
| [Go](https://go.dev/) | Application server, single binary output |
| [Echo](https://echo.labstack.com/) | HTTP routing and middleware |
| [HTMX](https://htmx.org/) | Hypermedia interactions |
| [templ](https://templ.guide/) | Type-safe HTML templating |
| [Tailwind CSS](https://tailwindcss.com/) | Utility-first styling |
| [DaisyUI](https://daisyui.com/) | Semantic component classes with 30+ themes |
| [Hyperscript](https://hyperscript.org/) | Client-side DOM interactions |
| [SQLite](https://www.sqlite.org/) | Embedded database |
| [Air](https://github.com/air-verse/air) | Live reload for development |
| [Mage](https://magefile.org/) | Build automation (Go-based) |

## Architecture

Harmony follows a **reach-up model**: start at HTML and only reach for higher-abstraction tools when the current layer cannot express the intent. Behavior and presentation stay on the element (locality of behavior).

```
                  State               Behavior              Presentation
           +------------------+----------------------+----------------------+
  Server   |  Go + SQL        |  HTTP + HTMX         |  templ + DaisyUI     |
           |  source of truth |  hypermedia controls |  semantic components |
           +------------------+----------------------+----------------------+
  Client   |  Alpine.js       |  _hyperscript        |  Tailwind + CSS      |
           |  view state      |  DOM interactions    |  layout, spacing     |
           +------------------+----------------------+----------------------+
```

The handler layer is the thickest: it knows the domain, selects templates, and assembles hypermedia controls. The service layer handles business logic. The repository layer is a thin SQL interface. Each layer does less than the one above it.

## Hypermedia Patterns

Harmony implements [HATEOAS](https://htmx.org/essays/hateoas/) -- the server drives application state by embedding hypermedia controls directly in responses.

### HATEOAS Error Recovery

Error responses include embedded recovery controls so the user always has a path forward:

| Scenario | Status | Recovery Controls |
| --- | --- | --- |
| Transient failure | 500 | Retry button re-issues the request |
| Validation error | 422 | Fix & Resubmit fetches a pre-filled correction form |
| Conflict | 409 | Update Existing (PUT) or Create as Copy (POST) |
| Stale data | 412 | Load Fresh Data or Force Save with confirmation |
| Cascade constraint | 409 | Reassign dependent items or Force Delete with confirmation |

Each error panel is built from a `Control` struct that maps to HTMX attributes (`hx-get`, `hx-post`, `hx-target`, `hx-confirm`), so recovery actions are standard hypermedia controls.

### Inline CRUD

Full create/read/update/delete without page reloads:

- **Create** -- `hx-post` appends a new row via `hx-swap="outerHTML"`
- **Edit** -- `hx-get` swaps a display row for an inline edit form
- **Save** -- `hx-put` with `hx-include="closest tr"` sends form data, server returns the read-only row
- **Toggle** -- `hx-patch` flips a single field and returns the updated row
- **Delete** -- `hx-delete` with `hx-confirm` removes the row from the DOM

### Real-time SSE

Server-Sent Events stream live data to the browser:

- Topic-based pub/sub broker with non-blocking publish and subscriber cleanup
- Background goroutines emit system stats, metrics, service health, and events
- Single SSE events carry multiple `hx-swap-oob` elements for composite updates
- Throttle control adjusts the event interval via query parameter

### Data Views

Interactive data views backed by SQLite:

- **Sort** -- Column headers toggle ASC/DESC via `hx-get` with sort parameters
- **Filter** -- Search input with `hx-trigger="input changed delay:400ms"` for debounced queries
- **Paginate** -- Server-computed pagination with `hx-include` to preserve filter state across pages
- **Bulk actions** -- Checkboxes with `hx-include` send selected IDs to bulk endpoints

### State and Interaction Patterns

- **Like counter** -- `hx-post` increments server state, returns button + count as a fragment
- **Toggle** -- POST flips boolean state, returns updated badge and button label
- **Auto-load** -- `hx-trigger="load"` fires GET immediately after DOM insertion
- **Lazy reveal** -- `hx-trigger="intersect once"` loads content when scrolled into view
- **Live preview** -- `hx-trigger="keyup changed delay:500ms"` for debounced server-side rendering
- **Append** -- `hx-swap="beforeend"` appends without replacing existing content
- **Modal** -- `hx-get` fetches a `<dialog>` fragment, `hx-on::load` opens it
- **Dismiss** -- HyperScript handles client-only UI like fade-out without a server round-trip

## Schema Builder

Composable DDL API for defining tables with common SQL patterns as chainable traits.

### Table Traits

```go
NewTable("Tasks").
    Columns(
        AutoIncrCol("ID"),
        Col("Title", TypeString(255)).NotNull(),
    ).
    WithUUID().
    WithStatus("draft").
    WithSortOrder().
    WithParent().
    WithNotes().
    WithExpiry().
    WithVersion().
    WithArchive().
    WithReplacement().
    WithTimestamps().
    WithSoftDelete().
    WithAuditTrail()
```

| Method | Column(s) | DDL | Mutable |
| --- | --- | --- | --- |
| `WithVersion()` | Version | `INTEGER NOT NULL DEFAULT 1` | Yes |
| `WithSortOrder()` | SortOrder | `INTEGER NOT NULL DEFAULT 0` | Yes |
| `WithStatus(default)` | Status | `VARCHAR(50) NOT NULL DEFAULT '{default}'` | Yes |
| `WithNotes()` | Notes | `TEXT` (nullable) | Yes |
| `WithUUID()` | UUID | `VARCHAR(36) NOT NULL UNIQUE` | No |
| `WithParent()` | ParentID | `INTEGER` (nullable) | Yes |
| `WithExpiry()` | ExpiresAt | `TIMESTAMP` (nullable) | Yes |
| `WithArchive()` | ArchivedAt | `TIMESTAMP` (nullable) | Yes |
| `WithReplacement()` | ReplacedByID | `INTEGER` (nullable) | Yes |
| `WithTimestamps()` | CreatedAt, UpdatedAt | `TIMESTAMP NOT NULL DEFAULT NOW()` | UpdatedAt only |
| `WithSoftDelete()` | DeletedAt | `TIMESTAMP` (nullable) | Yes |
| `WithAuditTrail()` | CreatedBy, UpdatedBy, DeletedBy | `VARCHAR(255)` | UpdatedBy, DeletedBy only |

### Table Types

Pre-built constructors for common patterns:

```go
NewLookupTable("Tags", "Type", "Label")           // Categorized reference data
NewLookupJoinTable("ItemTags")                     // Owner-to-lookup M2M
NewMappingTable("UserRoles", "UserID", "RoleID")   // Generic M2M join
NewConfigTable("Settings", "Key", "Value")         // Key-value settings
NewEventTable("AuditLog",                          // Append-only log
    Col("EventType", TypeVarchar(100)).NotNull(),
    Col("Payload", TypeText()),
)
NewQueueTable("JobQueue", "Payload")               // Job/outbox with retry
```

### Column Types

| Function | SQLite | MSSQL |
| --- | --- | --- |
| `TypeInt()` | `INTEGER` | `INT` |
| `TypeText()` | `TEXT` | `NVARCHAR(MAX)` |
| `TypeString(n)` | `TEXT` | `NVARCHAR(n)` |
| `TypeVarchar(n)` | `TEXT` | `VARCHAR(n)` |
| `TypeTimestamp()` | `TIMESTAMP` | `DATETIME` |
| `TypeAutoIncrement()` | `INTEGER PRIMARY KEY AUTOINCREMENT` | `INT PRIMARY KEY IDENTITY(1,1)` |
| `TypeLiteral(s)` | `s` | `s` |

### Domain Structs

Embeddable structs for domain models:

```go
type Task struct {
    ID                 int    `db:"ID"`
    Title              string `db:"Title"`
    domain.UUID
    domain.Status
    domain.SortOrder
    domain.Timestamps
    domain.SoftDelete
    domain.AuditTrail
}
```

### Repository Helpers

```go
repository.SetCreateTimestamps(&m.CreatedAt, &m.UpdatedAt)
repository.SetUpdateTimestamp(&m.UpdatedAt)
repository.SetSoftDelete(&deletedAt)
repository.SetDeleteAudit(&deletedAt, &deletedBy, "admin")
repository.InitVersion(&m.Version)
repository.IncrementVersion(&m.Version)
repository.SetSortOrder(&m.SortOrder, 5)
repository.SetStatus(&m.Status, "active")
repository.SetExpiry(&m.ExpiresAt, time.Now().Add(24*time.Hour))
repository.SetArchive(&m.ArchivedAt)
repository.SetReplacement(&m.ReplacedByID, newID)
repository.SetCreateAudit(&m.CreatedBy, &m.UpdatedBy, "user1")
repository.SetUpdateAudit(&m.UpdatedBy, "user2")
```

### Where Builder

Composable query filters:

```go
w := repository.NewWhere().
    NotDeleted().
    NotExpired().
    NotArchived().
    NotReplaced().
    HasStatus("active").
    HasVersion(3).
    IsRoot().
    HasParent(42)
```

### Select Builder

```go
query, args := repository.NewSelect("Tasks", "ID", "Title", "Status").
    Where(w).
    OrderBy("CreatedAt DESC").
    Paginate(25, 0).
    WithDialect(d).
    Build()

countQuery, countArgs := repository.NewSelect("Tasks", "ID", "Title", "Status").
    Where(w).
    CountQuery()
```

### Seed Data

Declare initial rows as part of schema definition. Seed is idempotent (`INSERT OR IGNORE`):

```go
settings := NewConfigTable("Settings", "Key", "Value").
    WithSeedRows(
        schema.SeedRow{"Key": "'app.name'", "Value": "'My App'"},
        schema.SeedRow{"Key": "'app.theme'", "Value": "'dark'"},
    )
```

### Schema Lifecycle

| Method | When | Behavior |
| --- | --- | --- |
| `InitSchema` | Development / tests | Drops and recreates all registered tables |
| `EnsureSchema` | Production startup | `CREATE TABLE IF NOT EXISTS` + `CREATE INDEX IF NOT EXISTS` |
| `SeedSchema` | Production startup | `INSERT OR IGNORE` for declared seed rows |
| `ValidateSchema` | Production startup | Read-only check that tables and columns exist |

## Project Structure

```
.
├── cmd/                    # CLI entry points
├── internal/
│   ├── config/             # Application configuration
│   ├── routes/
│   │   ├── handler/        # Render helpers, error handling
│   │   ├── hypermedia/     # Control structs, HATEOAS builders
│   │   ├── middleware/     # CSRF, correlation IDs, error handler, sessions
│   │   ├── response/       # OOB response builder
│   │   └── *.go            # Route handlers
│   └── service/            # Business logic, Graph client, SSE broker
├── web/
│   ├── assets/public/      # Static assets (CSS, JS, images)
│   ├── components/core/    # Reusable templ components
│   └── views/              # Page-level templ templates
├── e2e/                    # Playwright E2E tests
├── docs/                   # Documentation and MkDocs config
├── scripts/                # Build and utility scripts
└── .github/workflows/      # CI/CD workflows
```

## Development

### Prerequisites

- Go 1.24+ (latest)
- Node.js 22+ (for Playwright E2E tests)
- [Mage](https://magefile.org/) (`go install github.com/magefile/mage@latest`)

### Running the Dev Server

```bash
go tool mage watch
```

This starts templ in watch mode, Air for live reload, and Tailwind in watch mode.

Access the application at:

- Direct TLS: `https://localhost:{APP_TLS_PORT}`
- Via Caddy: `https://localhost:{CADDY_TLS_PORT}`

### HTTPS Development Setup

The setup script checks for existing `localhost.crt` and `localhost.key`. If missing, it offers to generate self-signed certificates.

To generate manually:

```bash
openssl req -x509 -newkey rsa:2048 -keyout localhost.key -out localhost.crt \
    -days 365 -nodes -subj "/CN=localhost" \
    -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"
```

Install the certificate in your system trust store for browser acceptance.

## Testing

```bash
# Go tests
go tool mage test              # Run all tests
go tool mage testverbose       # Verbose output
go tool mage testcoverage      # Coverage report
go tool mage testrace          # Race condition detection
go tool mage testwatch         # Auto-run on file changes

# E2E tests (Playwright)
npm ci                          # Install dependencies
npx playwright install chromium # Install browser
go tool mage teste2e            # Run headless
go tool mage teste2eheaded      # Run with visible browser
go tool mage teste2eui          # Run with Playwright UI

# Linting
go tool mage lint              # Run golangci-lint, golint, fieldalignment
go tool mage lintwatch         # Lint on file changes
```

## Mage Targets

All targets are run with `go tool mage <target>`:

| Target | Description |
| --- | --- |
| `watch` | Start dev mode with live reload (Tailwind, templ, Air) |
| `build` | Full production build |
| `compile` | Compile Go binary |
| `templ` / `templwatch` | Run templ / templ in watch mode |
| `tailwind` / `tailwindwatch` | Build / watch Tailwind CSS |
| `air` | Start Air live reload |
| `test*` | Test targets (see Testing section) |
| `teste2e` / `teste2eheaded` / `teste2eui` | Playwright E2E tests |
| `lint` / `lintwatch` | Lint / lint on file changes |
| `updateassets` | Update all frontend assets |
| `setup` | Run the template setup wizard |
| `envcheck` | Validate required environment variables |

## CI/CD Workflows

- **CI** (`ci.yml`) -- Build, vet, and race-condition tests on push/PR
- **E2E** (`e2e.yml`) -- Playwright end-to-end tests on push/PR
- **Docs** (`docs.yml`) -- Generate API docs and publish to GitHub Pages
- **Dependency Updates** (`main.yml`) -- Weekly `go get -u`, verify build/tests
- **Release** (`release.yml`) -- Semantic versioning with cross-compiled binaries (Linux/Windows)
- **Screenshots** (`screenshots.yml`) -- Automated Playwright screenshot capture

## Environment Variables

See `.env.sample` for the full list. Key variables:

| Variable | Description | Default |
| --- | --- | --- |
| `SERVER_LISTEN_PORT` | Echo server port | (required) |
| `LOG_LEVEL` | DEBUG, INFO, WARN, ERROR | INFO |
| `ENABLE_DATABASE` | Enable SQL backend | false |
| `DB_ENGINE` | sqlite or sqlserver | -- |
| `AZURE_CLIENT_ID` | Azure AD app client ID | -- |
| `AZURE_CLIENT_SECRET` | Azure AD app client secret | -- |
| `AZURE_TENANT_ID` | Azure AD tenant ID | -- |
| `SESSION_SECRET` | Session encryption key | -- |
| `CSRF_ROTATE_PER_REQUEST` | Rotate CSRF token per request | false |
| `AZURE_USER_REFRESH_HOUR` | Hour (0-23) for Graph user cache sync | 5 |
| `ENABLE_PHOTO_DOWNLOAD` | Download user photos from Graph | false |
