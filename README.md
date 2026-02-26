# go-htmx-template

<!--toc:start-->
- [go-htmx-template](#go-htmx-template)
  - [Tech Stack](#tech-stack)
  - [Getting Started](#getting-started)
    - [Prerequisites](#prerequisites)
    - [Running the Project](#running-the-project)
    - [Project Structure](#project-structure)
  - [HTTPS Development Setup](#https-development-setup)
  - [Testing](#testing)
  - [Mage Targets](#mage-targets)
    - [Example Usage](#example-usage)
<!--toc:end-->

A template repository for building new Go + HTMX projects with modern tooling and best practices.

## Tech Stack

- [**Go**](https://go.dev/): Backend language
- [**Echo**](https://echo.labstack.com/): High performance, minimalist Go web framework
- [**HTMX**](https://htmx.org/): Modern frontend interactivity with minimal JavaScript
- [**templ**](https://templ.guide/): Type-safe HTML templating for Go
- [**Air**](https://github.com/air-verse/air): Live reloading for Go applications
- [**Mage**](https://magefile.org/): Make/rake-like build tool for Go
- [**Tailwind CSS**](https://tailwindcss.com/): Utility-first CSS framework
- [**DaisyUI**](https://daisyui.com/): Tailwind CSS component library
- [**Caddy**](https://caddyserver.com/): Optional reverse proxy with TLS termination for HTTPS development

## Getting Started

### Prerequisites

- Go 1.24+
- (Optional, recommended) [`gum`](https://github.com/charmbracelet/gum) for interactive setup:

  ```bash
  go install github.com/charmbracelet/gum@latest
  ```

### Running the Project

#### One-time template setup

If you are using this repo as a template for a new app:

```bash
# With gum installed (interactive wizard, preferred)
go tool mage setup

# Or specify values explicitly (works with or without gum)
go tool mage setup -n "My App" -m "github.com/me/my-app" -p 5124
```

With `gum` installed, the setup script will first ask whether to copy the template to a new directory; if you choose yes, you enter the target path, the tree is copied there excluding `.git`, and you can optionally run `git init` in the new directory. Setup then continues in that directory (or in the current directory if you declined). You are prompted for app name, module path, and base port; without `gum` or when passing flags, setup runs in the current directory with no copy prompt.

After setup: review `.env.dev`, run `go tool mage watch`, and if you used the copy-and-git-init flow, add a remote when ready: `git remote add origin <url>`.

#### Start dev server

To start the development server with live reload and built-in TLS support:

```sh
go tool mage watch
```

This will start the server with TLS on port `{{APP_TLS_PORT}}` and reload on code changes. The application automatically uses TLS in development mode.

### Project Structure

- `main.go` — Application entrypoint with context-based lifecycle management
- `internals/config/` — Configuration management with singleton pattern
- `internals/logger/` — Structured logging with environment-aware configuration
- `internals/routes/` — Route and handler definitions
- `internals/service/` — Business logic services (includes `graph/`: Microsoft Graph SDK for Go client and user cache)
- `web/views/` — Templ components (HTML views)
- `web/components/` — Reusable Templ components
- `web/assets/public/` — Static assets (JS, CSS, images)
- `tests/` — Test helpers and utilities
- `cmd/testwatcher/` — Go-based test watcher for development

Optional env vars (e.g. for Crooner auth or Microsoft Graph) are documented in `.env.sample`.

## HTTPS Development Setup

This project supports HTTPS development with built-in TLS support. The Go application runs with TLS in development mode, and Caddy provides additional reverse proxy capabilities if needed.

### Certificate Setup

When the Caddy feature is selected, `go tool mage setup` checks for existing `localhost.crt` and `localhost.key` in the project root. If they exist (e.g. already trusted by your OS), they are used as-is. If missing, setup asks whether to generate new self-signed certificates.

To regenerate certificates manually:

```bash
openssl req -x509 -newkey rsa:2048 -keyout localhost.key -out localhost.crt \
  -days 365 -nodes -subj "/CN=localhost" \
  -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"
```

To trust the certificate on your system:

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

### Running with HTTPS

**Option 1: Direct TLS (Recommended)**
```bash
go tool mage watch
```
This starts the Go application with built-in TLS support on port `{{APP_TLS_PORT}}`.

**Option 2: With Caddy Proxy**
1. **Start the Go application** (in one terminal):
   ```bash
   go tool mage watch
   ```

2. **Start Caddy with TLS termination** (in another terminal):
   ```bash
   go tool mage caddystart
   ```

3. **Access your application**:
   - Direct TLS (Echo): https://localhost:`{{APP_TLS_PORT}}`
   - Caddy proxy: https://localhost:`{{CADDY_TLS_PORT}}`
   - Templ HTTP proxy (internal): http://localhost:`{{TEMPL_HTTP_PORT}}`

### Troubleshooting

- **Certificate not trusted**: Follow the certificate setup instructions above
- **Templ proxy issues**: Make sure the certificate is trusted in your system
- **Port conflicts**: Ensure the chosen `{{APP_TLS_PORT}}`, `{{CADDY_TLS_PORT}}`, and `{{TEMPL_HTTP_PORT}}` are available

## Testing

This project includes a comprehensive test suite with multiple testing options:

### Running Tests

```bash
# Run all tests
go tool mage test

# Run tests with verbose output
go tool mage testverbose

# Run tests with coverage
go tool mage testcoverage

# Generate HTML coverage report
go tool mage testcoveragehtml

# Run benchmark tests
go tool mage testbenchmark

# Run tests with race detection
go tool mage testrace

# Run tests in watch mode (automatically runs on file changes)
go tool mage testwatch
```

### Test Coverage

- **Config Package**: 90.9% coverage (singleton pattern, environment variables)
- **Logger Package**: 76.9% coverage (initialization, log levels, thread safety)
- **Main Application**: Integration tests for startup, shutdown, and lifecycle

### Test Features

- **Singleton Testing**: Proper reset functions for testing
- **Environment Variables**: Isolated test environment
- **Context Management**: Lifecycle and cancellation testing
- **Thread Safety**: Concurrent access testing
- **Performance**: Benchmark tests for startup and request handling

## Mage Targets

This project uses [Mage](https://magefile.org/) for build automation. All commands can be run with `go tool mage <target>`.

| Command             | Category     | Description                                                                  |
| ------------------- | ------------ | ---------------------------------------------------------------------------- |
| `watch`             | Development  | Start development mode with live reload (Tailwind, Templ, Air)               |
| `air`               | Development  | Run Air live reload tool for Go development                                  |
| `templ`             | Development  | Run Templ in watch mode for template compilation                             |
| `templwatch`        | Development  | Run Templ in watch mode (alias for `templ`)                                  |
| `templgenerate`     | Development  | Generate Templ files once                                                    |
| `build`             | Build        | Clean, update assets, and build the project                                  |
| `compile`           | Build        | Build the Go project                                                         |
| `run`               | Build        | Execute the compiled binary                                                  |
| `copyfiles`         | Build        | Copy necessary files to build directory                                      |
| `updateassets`      | Assets       | Update all assets (Hyperscript, HTMX, DaisyUI, Tailwind)                     |
| `tailwind`          | Assets       | Run Tailwind CSS compilation                                                 |
| `tailwindwatch`     | Assets       | Run Tailwind in watch mode                                                   |
| `tailwindupload`    | Assets       | Update the Tailwind binary                                                   |
| `daisyupdate`       | Assets       | Update DaisyUI CSS                                                           |
| `htmxupdate`        | Assets       | Update HTMX related files                                                    |
| `hyperscriptupdate` | Assets       | Update Hyperscript file                                                      |
| `test`              | Testing      | Run all tests                                                                |
| `testverbose`       | Testing      | Run tests with verbose output                                                |
| `testcoverage`      | Testing      | Run tests with coverage report                                               |
| `testcoveragehtml`  | Testing      | Generate HTML coverage report                                                |
| `testbenchmark`     | Testing      | Run benchmark tests                                                          |
| `testrace`          | Testing      | Run tests with race detection                                                |
| `testwatch`         | Testing      | Run tests in watch mode using Go-based watcher                              |
| `lint`              | Code Quality | Run static analysis and style checks (golangci-lint, golint, fieldalignment) |
| `fixfieldalignment` | Code Quality | Automatically fix field alignment issues                                     |
| `lintwatch`         | Code Quality | Run Air with lint configuration for automatic linting on file changes        |
| `clean`             | Utility      | Remove build and debug files                                                 |
| `cleanbuild`        | Utility      | Remove the build directory                                                   |
| `cleandebug`        | Utility      | Remove debug binaries                                                        |
| `envcheck`          | Utility      | Verify the environment file exists                                           |
| `preparedirs`       | Utility      | Create necessary directories                                                 |
| `caddyinstall`      | HTTPS        | Install Caddy for local development                                         |
| `caddystart`        | HTTPS        | Start Caddy with TLS termination                                            |

### Example Usage

```sh
# Start development with live reload
go tool mage watch

# Build the project
go tool mage build

# Run tests
go tool mage test

# Run tests in watch mode
go tool mage testwatch

# Run linting
go tool mage lint

# Run linting in watch mode
go tool mage lintwatch

# Update all assets
go tool mage updateassets

# Clean build artifacts
go tool mage clean

# Set up HTTPS with Caddy (optional)
go tool mage caddyinstall
```
