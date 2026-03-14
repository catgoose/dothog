# {{APP_NAME}}

<!--toc:start-->

- [{{APP_NAME}}](#appname)
  - [Template Setup](#template-setup)
  - [Running the App](#running-the-app)
  - [Local HTTPS & Ports](#local-https-ports)
  - [HTTPS Development Setup](#https-development-setup)
  - [Middleware](#middleware)
  - [Database](#database)
  - [Server-Sent Events (SSE)](#server-sent-events-sse)
  - [Observability](#observability)
  - [Enabling Crooner (Auth)](#enabling-crooner-auth)
  - [Enabling Microsoft Graph](#enabling-microsoft-graph)
  - [Avatar Photos](#avatar-photos)
  - [Testing](#testing)
    - [Go Tests](#go-tests)
    - [E2E Tests (Playwright)](#e2e-tests-playwright)
    - [Linting](#linting)
  - [Updating Frontend Assets](#updating-frontend-assets)
  - [CI/CD Workflows](#cicd-workflows)
  - [Mage Targets](#mage-targets)
  - [Environment Variables](#environment-variables)
  <!--toc:end-->

{{APP_NAME}} is a Go + HTMX application generated from {{TEMPLATE_REF}}.

It uses:

- Go + Echo (web framework)
- HTMX + templ (type-safe HTML templating)
- Tailwind CSS + DaisyUI (styling, 30+ themes)
- Air (live reload)
- Mage (build automation)
- Playwright (E2E testing)
- Caddy (optional HTTPS dev proxy)

## Template Setup

This project was bootstrapped from {{TEMPLATE_REF}}. If you are starting from the template:

1. Run the setup wizard to stamp your app name, module path, and dev ports:

   ```bash
   go tool mage setup
   ```

   The wizard lets you optionally copy the template to a new directory (no `.git` is copied), run `git init` there, then complete app name, module path, and ports in that directory. With flags (e.g. `go tool mage setup -n "My App" -m "github.com/you/my-app" -p 5124`) setup runs non-interactively. After setup you can run cleanup (when prompted) to remove the `_template_setup` folder and `mage_setup.go`.

3. Review `.env.dev` (generated from `.env.sample`) and adjust as needed.
4. Start development:

   ```bash
   go tool mage watch
   ```

   If you used the copy-and-git-init flow, add a remote when ready: `git remote add origin <url>`.

## Running the App

With `.env.dev` in place (or equivalent env vars set):

```bash
go tool mage watch
```

This will:

- Start templ in watch mode, proxying to `https://localhost:{{APP_TLS_PORT}}` with a local HTTP proxy on `{{TEMPL_HTTP_PORT}}`.
- Run Air to rebuild and restart the Go app.
- Run Tailwind in watch mode.
- Optionally start Caddy (via `go tool mage caddystart`) using the templated `Caddyfile`.

Access {{APP_NAME}} at:

- Direct TLS: `https://localhost:{{APP_TLS_PORT}}`
- Via Caddy: `https://localhost:{{CADDY_TLS_PORT}}`

## Local HTTPS & Ports

The dev HTTPS stack for {{APP_NAME}}:

- Echo app (TLS): `https://localhost:{{APP_TLS_PORT}}`
- templ HTTP proxy (internal): `http://localhost:{{TEMPL_HTTP_PORT}}`
- Caddy TLS front: `https://localhost:{{CADDY_TLS_PORT}}`

Request flow:

- Browser → Caddy (`https://localhost:{{CADDY_TLS_PORT}}`)
- Caddy (TLS termination) → templ HTTP proxy (`http://localhost:{{TEMPL_HTTP_PORT}}`)
- templ HTTP proxy → Echo over TLS (`https://localhost:{{APP_TLS_PORT}}`)

## HTTPS Development Setup

When the Caddy feature is selected, `mage setup` checks for existing `localhost.crt` and
`localhost.key` in the project root. If they exist (e.g. already trusted by your OS), they
are used as-is. If missing, setup asks whether to generate new self-signed certificates.

Generated certificates need to be installed in your system trust store:

**Linux (Ubuntu/Debian):**

```bash
sudo cp localhost.crt /usr/local/share/ca-certificates/
sudo update-ca-certificates
```

**macOS:**

1. Open Keychain Access
2. Drag `localhost.crt` to Keychain Access → System
3. Double-click the certificate and set 'Trust' to 'Always Trust'

**Windows:**

1. Right-click `localhost.crt`
2. Select 'Install Certificate'
3. Choose 'Local Machine' and 'Trusted Root Certification Authorities'

To regenerate certificates manually:

```bash
openssl req -x509 -newkey rsa:2048 -keyout localhost.key -out localhost.crt \
	-days 365 -nodes -subj "/CN=localhost" \
	-addext "subjectAltName=DNS:localhost,IP:127.0.0.1"
```

## Middleware

The app includes several middleware components:

- **Correlation IDs** — Each request gets a unique ID for tracing through logs
- **CSRF protection** — Token-based CSRF with optional per-request rotation (`CSRF_ROTATE_PER_REQUEST`, `CSRF_PER_REQUEST_PATHS`)
- **Error handling** — Centralized error display
- **Form validation** — Request validation helpers
- **Session settings** — Per-user session preferences

## Database

Database support is **disabled by default** (`ENABLE_DATABASE=false` in `.env`). When enabled, the app supports SQLite and MS SQL Server.

Features include:

- Schema builder with traits (timestamps, soft delete, audit trails)
- Repository pattern with query builder (where/select)
- Health checks and validation
- Multi-dialect support (SQLite/MSSQL)

Configure via environment variables:

```bash
ENABLE_DATABASE=true
DB_ENGINE=sqlite # or sqlserver
DB_HOST=...
DB_DATABASE=...
DB_USER=...
DB_PASSWORD=...
```

## Server-Sent Events (SSE)

When the SSE feature is selected, the app includes a real-time event broker:

- Topic-based publish/subscribe
- HTMX SSE extension integration
- Automatic Caddy configuration for streaming

## Observability

- **Structured logging** — slog-based with configurable log level (`LOG_LEVEL`: DEBUG, INFO, WARN, ERROR)
- **Request log capture** — In-memory ring buffer (512 entries) for recent request inspection
- **Issue reporting** — `GET /report-issue/:requestID` endpoint with full request context
- **File logging** — Rotation via lumberjack

## Enabling Crooner (Auth)

Crooner (Azure AD / Entra ID auth) is **disabled by default** via the config variable `CroonerDisabled = true` in `internal/config/config.go`. To enable Crooner:

1. Set `CroonerDisabled = false` in config (or refactor to read from an env var).
2. Set the required env vars: `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`, `AZURE_TENANT_ID`, `AZURE_REDIRECT_URL`, `AZURE_LOGIN_REDIRECT_URL`, `AZURE_LOGOUT_REDIRECT_URL`, `SESSION_SECRET`. Optional: `APP_NAME` (defaults to `"app"`).

3. Protect routes using Crooner's session/claims helpers.

When Crooner is enabled, the app errors on startup if any of the required env vars are not set. When disabled, those vars are not required.

## Enabling Microsoft Graph

The template includes a Microsoft Graph client (using the [Microsoft Graph SDK for Go](https://github.com/microsoftgraph/msgraph-sdk-go)) and user cache under `internal/service/graph`. Graph is **off by default** until you:

1. Set Azure app-only credentials (same env vars as Crooner; used for Graph client credentials):

   ```bash
   AZURE_CLIENT_ID=...
   AZURE_CLIENT_SECRET=...
   AZURE_TENANT_ID=...
   AZURE_USER_REFRESH_HOUR=5
   ```

2. Wire the Graph client and user cache (e.g. `graph.NewGraphClient`, `InitAndSyncUserCache`) into your `main.go` or services.
3. Optionally enable photo download: `ENABLE_PHOTO_DOWNLOAD=true`.

See `.env.sample` for all optional env vars. If Azure/Graph vars are unset, Graph features stay inactive.

## Avatar Photos

When the `avatar` setup feature is selected, the app downloads user profile photos from Microsoft Graph and caches them on the filesystem.

- **Cache directory**: `web/assets/public/images` (override in `main.go` if needed)
- **Endpoint**: `GET /api/avatar/:azureID` — serves the cached photo, or 404
- **Schedule**: photos sync on the same schedule as the user cache refresh (`AZURE_USER_REFRESH_HOUR`). An initial sync runs on startup, then daily at the configured hour (production) or once (development).
- Photos are stored in a two-level directory (`images/ab/abc-123.jpg`) to avoid flat directory issues

## Testing

### Go Tests

```bash
go tool mage test             # Run all tests
go tool mage testverbose      # Verbose output
go tool mage testcoverage     # Coverage report
go tool mage testcoveragehtml # HTML coverage report
go tool mage testbenchmark    # Benchmarks
go tool mage testrace         # Race condition detection
go tool mage testwatch        # Auto-run tests on file changes
```

### E2E Tests (Playwright)

End-to-end tests live in `e2e/` and run against a live instance of the app using Playwright and Chromium.

```bash
npm ci                          # Install dependencies (first time)
npx playwright install chromium # Install browser (first time)
go tool mage teste2e            # Run E2E tests headless
go tool mage teste2eheaded      # Run with visible browser
go tool mage teste2eui          # Run with Playwright UI
```

### Linting

```bash
go tool mage lint              # Run golangci-lint, golint, fieldalignment
go tool mage fixfieldalignment # Auto-fix struct field alignment
go tool mage lintwatch         # Lint on file changes
```

## Updating Frontend Assets

Mage targets to pull the latest versions of frontend dependencies:

```bash
go tool mage updateassets      # Update all assets
go tool mage tailwindupdate    # Tailwind CLI binary
go tool mage htmxupdate        # HTMX + extensions (response-targets, SSE)
go tool mage hyperscriptupdate # Hyperscript
go tool mage daisyupdate       # DaisyUI CSS (includes 30+ themes)
```

## CI/CD Workflows

The template includes GitHub Actions workflows:

- **CI** (`ci.yml`) — Build, vet, and race-condition tests on push/PR
- **E2E** (`e2e.yml`) — Playwright end-to-end tests on push/PR
- **Docs** (`docs.yml`) — Generate gomarkdoc API docs and publish to GitHub Pages
- **Dependency Updates** (`main.yml`) — Weekly `go get -u`, verify build/tests, auto-commit
- **Release** (`release.yml`) — Semantic versioning with cross-compiled binaries (Linux/Windows)
- **Screenshots** (`screenshots.yml`) — Automated Playwright screenshot capture

## Mage Targets

All targets are run with `go tool mage <target>`:

| Target                                    | Description                                                       |
| ----------------------------------------- | ----------------------------------------------------------------- |
| `watch`                                   | Start dev mode with live reload (Tailwind, templ, Air)            |
| `build`                                   | Full production build (clean, tailwind, compile, copy files)      |
| `compile`                                 | Compile Go binary                                                 |
| `templ` / `templwatch`                    | Run templ / templ in watch mode                                   |
| `tailwind` / `tailwindwatch`              | Build / watch Tailwind CSS                                        |
| `air`                                     | Start Air live reload                                             |
| `test*`                                   | Test targets (see Testing section)                                |
| `teste2e` / `teste2eheaded` / `teste2eui` | Playwright E2E tests                                              |
| `lint` / `lintwatch`                      | Lint / lint on file changes                                       |
| `fixfieldalignment`                       | Auto-fix struct field alignment                                   |
| `updateassets`                            | Update all frontend assets (Tailwind, HTMX, DaisyUI, Hyperscript) |
| `caddyinstall`                            | Install Caddy for local HTTPS                                     |
| `caddystart`                              | Start Caddy with TLS termination                                  |
| `clean` / `cleanbuild` / `cleandebug`     | Remove build artifacts                                            |
| `setup`                                   | Run the template setup wizard                                     |
| `envcheck`                                | Validate required environment variables                           |

## Environment Variables

See `.env.sample` for the full list. Key variables:

| Variable                  | Description                                | Default    |
| ------------------------- | ------------------------------------------ | ---------- |
| `SERVER_LISTEN_PORT`      | Echo server port                           | (required) |
| `LOG_LEVEL`               | DEBUG, INFO, WARN, ERROR                   | INFO       |
| `ENABLE_DATABASE`         | Enable SQL backend                         | false      |
| `DB_ENGINE`               | sqlite or sqlserver                        | —          |
| `AZURE_CLIENT_ID`         | Azure AD app client ID                     | —          |
| `AZURE_CLIENT_SECRET`     | Azure AD app client secret                 | —          |
| `AZURE_TENANT_ID`         | Azure AD tenant ID                         | —          |
| `SESSION_SECRET`          | Session encryption key                     | —          |
| `CSRF_ROTATE_PER_REQUEST` | Rotate CSRF token per request              | false      |
| `CSRF_PER_REQUEST_PATHS`  | Comma-separated paths for per-request CSRF | —          |
| `AZURE_USER_REFRESH_HOUR` | Hour (0-23) for Graph user cache sync      | 5          |
| `ENABLE_PHOTO_DOWNLOAD`   | Download user photos from Graph            | false      |
