# {{APP_NAME}}

{{APP_NAME}} is a Go + HTMX application generated from {{TEMPLATE_REF}}.

It uses:

- Go + Echo
- HTMX + templ
- Air for live reload
- Tailwind + DaisyUI for styling
- Caddy (optional) for HTTPS dev proxy

## Template Setup

This project was bootstrapped from {{TEMPLATE_REF}}. If you are starting from the template:

1. (Optional but recommended) Install [`gum`](https://github.com/charmbracelet/gum) for a nicer interactive wizard:

   ```bash
   go install github.com/charmbracelet/gum@latest
   ```

2. Run the setup script once to stamp your app name, module path, and dev ports:

   ```bash
   go tool mage setup
   # Or call the script directly:
   ./_template_setup/setup-template.sh
   ```

   With `gum` installed, you can optionally copy the template to a new directory (no `.git` is copied), run `git init` there, then complete app name, module path, and ports in that directory. With flags (e.g. `go tool mage setup -n "My App" -m "github.com/you/my-app" -p 5124`) or without `gum`, setup runs in the current directory. After setup you can run cleanup (when prompted) to remove the `_template_setup` folder and `mage_setup.go`.

3. Review `.env.dev` (generated from `.env.example`) and adjust as needed.
4. Start development:

   ```bash
   go tool mage watch
   ```

   If you used the copy-and-git-init flow, add a remote when ready: `git remote add origin <url>`.

## Local HTTPS & Ports

The dev HTTPS stack for {{APP_NAME}} mirrors the Buckets / PTO-Calendar pattern:

- Echo app (TLS): `https://localhost:{{APP_TLS_PORT}}`
- templ HTTP proxy (internal): `http://localhost:{{TEMPL_HTTP_PORT}}`
- Caddy TLS front: `https://localhost:{{CADDY_TLS_PORT}}`

Request flow:

- Browser → Caddy (`https://localhost:{{CADDY_TLS_PORT}}`)
- Caddy (TLS termination) → templ HTTP proxy (`http://localhost:{{TEMPL_HTTP_PORT}}`)
- templ HTTP proxy → Echo over TLS (`https://localhost:{{APP_TLS_PORT}}`)

## HTTPS Development Setup

When the Caddy feature is selected, `mage setup` checks for existing `localhost.crt` and
`localhost.key` in the project root.  If they exist (e.g. already trusted by your OS), they
are used as-is.  If missing, setup asks whether to generate new self-signed certificates.

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

## Enabling Crooner (Auth)

Crooner (Azure AD / Entra ID auth) is **disabled by default** via the config variable `CroonerDisabled = true` in `internals/config/config.go`. To enable Crooner:

1. Set `CroonerDisabled = false` in config (or refactor to read from an env var).
2. Set the required env vars: `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`, `AZURE_TENANT_ID`, `AZURE_REDIRECT_URL`, `AZURE_LOGIN_REDIRECT_URL`, `AZURE_LOGOUT_REDIRECT_URL`, `SESSION_SECRET`. Optional: `APP_NAME` (defaults to `"app"`).

3. Protect routes using Crooner's session/claims helpers.

When Crooner is enabled, the app errors on startup if any of the required env vars are not set. When disabled, those vars are not required.

## Enabling Microsoft Graph

The template includes a Microsoft Graph client (using the [Microsoft Graph SDK for Go](https://github.com/microsoftgraph/msgraph-sdk-go)) and user cache under `internals/service/graph`. Graph is **off by default** until you:

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

Common commands:

```bash
go tool mage test
go tool mage testverbose
go tool mage testcoverage
go tool mage testcoveragehtml
go tool mage testbenchmark
go tool mage testrace
go tool mage testwatch
```

## Mage Targets

Key Mage targets (run with `go tool mage <target>`):

- `watch`      — Start dev mode with live reload (Tailwind, templ, Air)
- `templ`      — Run templ in watch mode
- `tailwind`   — Build Tailwind CSS
- `test*`      — Test targets (see above)
- `caddystart` — Start Caddy with TLS termination using the templated `Caddyfile`
- `setup`      — Run the template setup script (for initial app configuration)
