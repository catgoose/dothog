# Security

What dothog provides out of the box and where it expects you to bring your own.

## Ships with scaffold

These are active in every generated app by default.

### HTTP Security Headers

| Header | Value | Notes |
|--------|-------|-------|
| Permissions-Policy | `camera=(), microphone=(), geolocation=(), payment=(), usb=()` | Restrictive by default |
| Cross-Origin-Opener-Policy | `same-origin` | Prevents cross-origin window references |
| Vary | `HX-Request` | Cache separation for HTMX partial vs full-page responses |
| Cache-Control (static) | `public, max-age=31536000, immutable` | Long-lived cache for fingerprinted assets |

Provided via `dorman.SecurityHeaders()` middleware.

### Alpine.js CSP Build

Uses `@alpinejs/csp` which eliminates `eval()` and `new Function()`. All components registered via `Alpine.data()`. Downstream apps do not need `unsafe-eval` in their Content Security Policy.

### Template Auto-Escaping

Templ (the Go template engine) HTML-escapes all interpolated values by default. This prevents reflected XSS in server-rendered content.

### Parameterized Database Queries

All database access uses `jmoiron/sqlx` with parameterized placeholders. No raw SQL string concatenation from user input.

### SQLite Hardening

- WAL mode enabled for concurrent read access
- 30-second busy timeout for lock contention
- Connection pool: 1 writer, configurable idle/lifetime limits
- Prepared statements with deferred close

### Structured Error Handling

- Custom error handler renders safe error pages (no stack traces exposed to users)
- Per-request correlation IDs for tracing
- Error traces persisted to SQLite with 90-day retention
- Separate rendering for HTMX partial vs full-page errors
- Panic recovery middleware prevents crashes from leaking details

### Request Logging

- Structured request logging via promolog
- Correlation middleware for distributed tracing
- Server-Timing header for performance observability

### Graceful Shutdown

- OS signal handling (SIGINT, SIGTERM)
- 10-second drain timeout
- Closes database connections and SSE broker cleanly

### Dependency Integrity

- `go.sum` checksums for all module dependencies
- Minimal dependency surface -- security-critical code in catgoose-maintained libraries (dorman, crooner)

## Feature-gated (opt-in via setup)

These are available in the scaffold but only included when the feature is enabled during `mage setup`.

### CSRF Protection (`setup:feature:csrf`)

- Token generation and rotation via `dorman.CSRFProtect()`
- Sec-Fetch-Site fast-path: modern browsers (94%+ coverage) skip token validation entirely when `Sec-Fetch-Site: same-origin` is present
- Double-submit cookie pattern with configurable key from `SESSION_SECRET`
- Automatic HTMX header injection via `htmx:configRequest` listener
- Meta tag `<meta name="csrf-token">` for client-side access
- Per-request rotation and exempt path configuration
- Header: `X-CSRF-Token`, field: `csrf_token`

### Authentication (`setup:feature:auth`)

- OIDC/OAuth2 via crooner library
- PKCE flow support
- Session management with 24-hour lifetime
- Login, logout, callback routes registered automatically
- Azure AD Graph integration for user/photo sync (optional)

### Session Settings (`setup:feature:session_settings`)

- Persistent session preferences in SQLite
- HttpOnly cookies (no JavaScript access)
- SameSite=Lax (CSRF protection at cookie level)
- Cryptographically random UUID session IDs (crypto/rand)
- Touch-based refresh (re-validates after 24 hours)

### Offline / PWA (`setup:feature:offline`)

- Service worker with network-first strategy for HTML
- Cache-first for immutable static assets
- Mutation queue (POST/PUT/DELETE stored in IndexedDB when offline)
- Versioned cache naming for clean updates

## Not provided

These are deliberate gaps -- things the scaffold does not implement. Depending on your deployment, you may need to add them.

### Content-Security-Policy header

No CSP header is set by default. The Alpine CSP build means you can set a strict policy without `unsafe-eval`, but you need to configure the header yourself (via dorman or your reverse proxy). A recommended starting point:

```
script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; font-src 'self';
```

### Strict-Transport-Security (HSTS)

Disabled by default in dorman (can break dev environments without TLS). Enable with `dorman.DefaultHSTSConfig()` when serving over HTTPS, or handle at the reverse proxy layer.

### X-Frame-Options / frame-ancestors

Set to `SAMEORIGIN` by default via `dorman.SecurityHeaders()`. Override with CSP `frame-ancestors` if you need stricter control.

### X-Content-Type-Options

Set to `nosniff` by default via `dorman.SecurityHeaders()`. No action needed.

### Rate Limiting

No rate limiting middleware. For public-facing apps, add rate limiting at the reverse proxy layer or integrate a Go middleware (e.g., `golang.org/x/time/rate`).

### Authorization / RBAC

No role-based access control. Authentication (who are you?) is provided via crooner, but authorization (what can you do?) is left to the application. Dorman is designed for this but the scaffold does not wire up permission checks -- your app defines its own authorization rules.

### Input Validation

Beyond template escaping and parameterized queries, there is no request body validation middleware. Validate inputs in your handlers. Consider a validation library if your app has complex form processing.

### Request Body Size Limits

No explicit request body size limits. Echo's default is 4MB. For file upload endpoints, configure `echo.BodyLimit()`.

### TLS at the Application

Dev mode supports `StartTLS` with local certificates. Production assumes deployment behind a TLS-terminating proxy. The app itself does not manage certificates or ACME.

### Session Cookie Secure Flag

The `Secure` flag is set automatically in production (`!appenv.Dev()`). In development mode it is disabled so local HTTP works. No action needed for production deployments behind TLS.

### Encryption at Rest

SQLite databases are not encrypted. If you need encryption at rest, use SQLCipher or encrypt at the filesystem level.

### Dependency Vulnerability Scanning

No automated vulnerability scanning in CI. Consider adding `govulncheck` or Dependabot to your pipeline.

### Audit Logging

No audit trail for user actions or database modifications. Implement application-level audit logging if required for compliance.
