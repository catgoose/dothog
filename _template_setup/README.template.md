# {{APP_NAME}}

{{APP_NAME}} is a Go + HTMX application built on the reach-up architecture pattern.

It uses:

- Go + Echo (web framework)
- HTMX + templ (type-safe HTML templating)
- Tailwind CSS + DaisyUI (styling, 30+ themes)
- Air (live reload)
- Mage (build automation)
- Playwright (E2E testing)

## Features

{{FEATURE_TABLE}}

## Getting Started

### Prerequisites

- Go 1.24+
- Node.js 18+ (for Tailwind, Playwright)

### Setup

```bash
# Install dependencies
npm ci

# Start development (live reload)
go tool mage watch
```

This will:

- Start templ in watch mode, proxying to `https://localhost:{{APP_TLS_PORT}}` with a local HTTP proxy on `{{TEMPL_HTTP_PORT}}`
- Run Air to rebuild and restart the Go app
- Run Tailwind in watch mode

Access {{APP_NAME}} at: `https://localhost:{{APP_TLS_PORT}}`

### Ports

| Service | Port | URL |
| --- | --- | --- |
| Echo app (TLS) | {{APP_TLS_PORT}} | `https://localhost:{{APP_TLS_PORT}}` |
| templ HTTP proxy | {{TEMPL_HTTP_PORT}} | `http://localhost:{{TEMPL_HTTP_PORT}}` |
| Caddy TLS front | {{CADDY_TLS_PORT}} | `https://localhost:{{CADDY_TLS_PORT}}` |

### Environment Variables

Copy `.env.development` and adjust as needed. Key variables:

| Variable | Description | Default |
| --- | --- | --- |
| `SERVER_LISTEN_PORT` | Echo server port | {{APP_TLS_PORT}} |
| `LOG_LEVEL` | DEBUG, INFO, WARN, ERROR | INFO |
| `ENABLE_DATABASE` | Enable SQL backend | false |
| `APP_NAME` | Application display name | {{APP_NAME}} |

## Architecture

{{APP_NAME}} follows the **reach-up model** -- each layer reaches up to the layer above it rather than importing downward:

```
Behavior    (Alpine.js, Hyperscript)     -- client-side interactivity
    ^
Presentation (templ + DaisyUI)            -- type-safe HTML rendering
    ^
HTTP        (Echo + HTMX)                 -- routing, hypermedia exchanges
    ^
Data        (Go + SQL)                    -- repositories, schema DSL
```

Key patterns:

- **HATEOAS** -- hypermedia as the engine of application state; navigation and actions are link-driven
- **Server-rendered** -- templ components produce HTML; HTMX handles partial page updates
- **Feature-gated** -- code is organized by feature with clean separation for easy extension

{{FEATURE_SECTIONS}}

## Development

### Testing

```bash
go tool mage test             # Run all Go tests
go tool mage teste2e          # Run Playwright E2E tests
go tool mage testcoverage     # Coverage report
```

### Linting

```bash
go tool mage lint              # Run golangci-lint
go tool mage fixfieldalignment # Auto-fix struct field alignment
```

### Building

```bash
go tool mage build    # Full production build
go tool mage compile  # Compile Go binary only
```

### Mage Targets

All targets are run with `go tool mage <target>`:

| Target | Description |
| --- | --- |
| `watch` | Start dev mode with live reload |
| `build` | Full production build |
| `compile` | Compile Go binary |
| `test` / `testverbose` / `testcoverage` | Go tests |
| `teste2e` / `teste2eheaded` / `teste2eui` | Playwright E2E tests |
| `lint` / `lintwatch` | Lint |
| `updateassets` | Update frontend assets |
| `clean` | Remove build artifacts |
| `setup` | Run the template setup wizard |

## HTTPS Development Setup

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

## Module

- **Module path**: `{{MODULE_PATH}}`
- **Generated from**: {{TEMPLATE_REF}}
