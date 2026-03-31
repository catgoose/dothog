# {{APP_NAME}}

A server-driven hypermedia web application built with Go, HTMX, and templ. Generated from [dothog](https://github.com/catgoose/dothog) using `mage setup`.

{{APP_NAME}} runs as a single binary with all assets embedded. No external runtime dependencies are required, though environment files (`.env.development`, `.env.sample`) are used to configure the application.

See [PHILOSOPHY.md](PHILOSOPHY.md) for the architectural principles behind the project.

## Setup Configuration

This project was generated with the following features enabled:

{{FEATURE_TABLE}}

For documentation on each configuration option, see [docs/SETUP.md](docs/SETUP.md).

## Tech Stack

{{TECH_STACK}}

## Quick Start

{{QUICK_START}}

## Architecture

{{APP_NAME}} follows a **reach-up model**: start at HTML and only reach for higher-abstraction tools when the current layer cannot express the intent.

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

## Project Structure

```
.
├── cmd/                    # CLI entry points
├── internal/
│   ├── config/             # Application configuration
│   ├── database/           # Database connections, schema, repository
│   ├── domain/             # Domain models
│   ├── routes/
│   │   ├── handler/        # Render helpers, error handling
│   │   ├── middleware/     # Correlation IDs, error handler
│   │   └── *.go            # Route handlers
│   └── service/            # Business logic, Graph client
├── web/
│   ├── assets/public/      # Static assets (CSS, JS, images)
│   ├── components/core/    # Reusable templ components
│   └── views/              # Page-level templ templates
├── e2e/                    # Playwright E2E tests
├── docs/                   # Documentation and MkDocs config
└── .github/workflows/      # CI/CD workflows
```

## Development

### Prerequisites

- Go 1.26+ (latest)
- Node.js 22+ (for Playwright E2E tests)

### Running the Dev Server

```bash
go tool mage watch
```

This starts templ in watch mode, Air for live reload, and Tailwind in watch mode.

Access the application at: `https://localhost:{{APP_TLS_PORT}}`

### HTTPS Development Setup

When Caddy is configured, the app uses self-signed certificates for local HTTPS.

**Linux (Ubuntu/Debian):**

```bash
sudo cp localhost.crt /usr/local/share/ca-certificates/
sudo update-ca-certificates
```

**macOS:**

1. Open Keychain Access
2. Drag `localhost.crt` to System keychain
3. Set Trust to Always Trust

**Windows:**

1. Right-click `localhost.crt` > Install Certificate
2. Choose Local Machine > Trusted Root Certification Authorities

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
go tool mage lint              # Run golangci-lint
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
| `envcheck` | Validate required environment variables |

## Environment Variables

See `.env.sample` for the full list. Key variables:

{{ENV_TABLE}}

{{FEATURE_SECTIONS}}

## Module

- **Module path**: `{{MODULE_PATH}}`
- **Generated from**: {{TEMPLATE_REF}}
