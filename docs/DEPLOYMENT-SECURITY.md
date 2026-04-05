# Deployment Security

How security works across the full deployment stack, from the browser to the application. This covers what the infrastructure provides vs what the app provides, how they layer together, and how to configure for different environments.

## Architecture

```
Browser ──HTTPS──> nginx (443) ──HTTP──> app (:port)
```

Two layers, each with a role:

| Layer | Handles | Examples |
|-------|---------|----------|
| **nginx** | TLS termination, reverse proxy, SSE buffering, H3/QUIC | SSL with wildcard certificate, HTTP/2, HTTP/3, proxy headers |
| **App (porter)** | Application security, session management, CSRF, per-app CSP | Security headers, CSRF tokens, auth middleware, correlation IDs |

## What each layer does

### nginx (origin proxy)

Configured in `deploy/cloud-init.yml`. Each app gets a server block.

- Listens on port 443 with a wildcard certificate
- HTTP/2 enabled (`listen 443 ssl http2`) for multiplexed requests
- Port 80 kept for health checks and HTTP-to-HTTPS redirect
- SSE endpoints get dedicated config: buffering off, 300s timeouts, HTTP/1.1 upstream
- `X-Forwarded-Proto`, `X-Real-IP`, `X-Forwarded-For` headers passed through

### App (porter middleware)

Configured in each app's `routes.go`. This is what ships with the scaffold.

| Middleware | What it does |
|-----------|-------------|
| `porter.SecurityHeaders()` | Permissions-Policy, COOP, X-Frame-Options, X-Content-Type-Options |
| `porter.CSRFProtect()` | CSRF tokens with Sec-Fetch-Site fast-path, double-submit cookies |
| Compression | Brotli + Gzip via httpcompression |
| Correlation IDs | Per-request trace IDs via promolog |
| `crooner` session/auth | OIDC, PKCE, session management (when auth feature enabled) |

## Corporate / Self-Hosted Deployment

For corporate networks, local deployments, and self-hosted setups behind your own infrastructure.

### Wildcard certificate setup (corporate CA, Let's Encrypt)

Configure your wildcard certificate on nginx:

```nginx
# /etc/nginx/snippets/ssl.conf
ssl_certificate /etc/nginx/ssl/your-wildcard.crt;
ssl_certificate_key /etc/nginx/ssl/your-wildcard.key;
ssl_protocols TLSv1.2 TLSv1.3;
ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305;
ssl_prefer_server_ciphers off;
```

If your cert is RSA (most corporate CAs), use broader ciphers:

```nginx
ssl_ciphers HIGH:!aNULL:!MD5;
```

Everything else stays the same. The app doesn't care who terminates TLS -- it sees `X-Forwarded-Proto: https` from nginx either way.

### Features that require a CDN / edge proxy

If deploying without a CDN or edge proxy, here's how to cover common security features:

| Feature | Without edge proxy | Replacement |
|---------|-------------------|-------------|
| HSTS | Not set at edge | Enable in porter: `porter.DefaultHSTSConfig()` |
| CSP (baseline) | No edge fallback | Set in porter (already done per-app) |
| Rate limiting | None | Add Go middleware (`golang.org/x/time/rate`) or nginx `limit_req` |
| Early Hints (edge cache) | No edge replay | App still sends 103s if nginx uses H2 upstream, but no edge caching |
| ECH | Not available | N/A -- requires a supporting edge proxy |
| X-Robots-Tag | Not set | Add in porter or nginx if needed |
| DDoS protection | None | Firewall rules, fail2ban, or upstream provider |
| Bot management | None | Consider Caddy with crowdsec or similar |

### Corporate/Azure deployment checklist

When deploying behind a corporate reverse proxy or Azure Application Gateway:

1. **TLS**: Use your corporate wildcard cert on nginx (or let the corporate proxy terminate TLS and proxy HTTP to nginx on port 80)
2. **HSTS**: Enable in porter if the corporate proxy doesn't set it
3. **CSP**: Already set by porter per-app -- no change needed
4. **CSRF**: Already handled by porter -- no change needed
5. **Rate limiting**: Add nginx `limit_req` zone:
   ```nginx
   # In http block
   limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;
   
   # In location block
   location /api/ {
       limit_req zone=api burst=20 nodelay;
       proxy_pass http://127.0.0.1:PORT;
   }
   ```
6. **Auth**: Enable the auth feature (`mage setup` with OIDC) -- crooner handles Azure AD/Entra ID natively
7. **Session cookie Secure flag**: Set via environment variable when serving over HTTPS
8. **X-Forwarded headers**: Ensure your corporate proxy passes `X-Forwarded-Proto`, `X-Real-IP`, and `X-Forwarded-For` -- porter and crooner rely on these

### Corporate nginx config with H3 and SSE (tavern)

This config is designed for deploying dothog apps on a corporate Linux server with your own wildcard certificate, HTTP/3 (QUIC), and full SSE support for tavern's real-time features.

**Requirements:**
- nginx 1.25.0+ (HTTP/3 support merged into mainline)
- Ubuntu 24.04: `apt install nginx` gets you 1.26+ with QUIC
- Your corporate wildcard cert and key (PEM format)
- UDP port 443 open in your firewall (QUIC uses UDP, not TCP)

```nginx
# /etc/nginx/nginx.conf — add to the http block
# Rate limiting zone (shared across all server blocks)
limit_req_zone $binary_remote_addr zone=app:10m rate=10r/s;

# Connection limiting (prevents a single IP from hogging connections)
limit_conn_zone $binary_remote_addr zone=addr:10m;
```

**Per-app server config:**

```nginx
# /etc/nginx/sites-available/myapp
server {
    # HTTP/1.1 and HTTP/2 over TLS
    listen 443 ssl;
    http2 on;

    # HTTP/3 (QUIC) over UDP
    listen 443 quic reuseport;

    server_name myapp.corp.example.com;

    # TLS — your corporate wildcard cert
    ssl_certificate     /etc/nginx/ssl/corp-wildcard.crt;
    ssl_certificate_key /etc/nginx/ssl/corp-wildcard.key;
    ssl_protocols       TLSv1.2 TLSv1.3;
    ssl_ciphers         HIGH:!aNULL:!MD5;

    # Advertise HTTP/3 support to browsers
    # Browsers discover H3 via this header and upgrade on subsequent requests
    add_header Alt-Svc 'h3=":443"; ma=86400' always;

    # HSTS — set at the nginx layer
    add_header Strict-Transport-Security "max-age=63072000; includeSubDomains; preload" always;

    # Rate limiting on POST (protects demo/mutation endpoints)
    limit_req zone=app burst=20 nodelay;
    limit_conn addr 50;

    # --- SSE (tavern) ---
    # Tavern uses /sse/ for Server-Sent Events. These are long-lived HTTP
    # connections that stream events to the browser. Without these settings,
    # nginx buffers the response and the browser sees nothing until the
    # connection closes.
    location /sse/ {
        proxy_pass http://127.0.0.1:PORT;

        # Proxy headers
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # SSE requires all buffering disabled
        proxy_buffering off;            # Don't buffer the response body
        proxy_cache off;                # Don't cache SSE streams
        proxy_request_buffering off;    # Don't buffer the request body
        chunked_transfer_encoding off;  # SSE uses its own framing

        # Disable compression — SSE data is small and frequent,
        # compression adds latency for no benefit
        gzip off;

        # HTTP/1.1 upstream with persistent connections
        # SSE doesn't work over HTTP/2 upstream because nginx doesn't
        # support streaming responses over its H2 upstream implementation
        proxy_http_version 1.1;
        proxy_set_header Connection "";

        # Tell nginx not to buffer at the application level
        proxy_set_header X-Accel-Buffering "no";

        # Timeouts — tavern sends keepalive pings every 30s,
        # so 300s handles up to 10 missed pings before dropping
        proxy_read_timeout 300s;
        proxy_send_timeout 300s;
        proxy_connect_timeout 60s;
    }

    # --- Everything else ---
    location / {
        proxy_pass http://127.0.0.1:PORT;

        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}

# Redirect HTTP to HTTPS
server {
    listen 80;
    server_name myapp.corp.example.com;
    return 301 https://$host$request_uri;
}
```

**Important notes:**

- **`reuseport`**: Only use on the **first** server block that binds to port 443 with `quic`. If you have multiple apps, only the first gets `reuseport`. Subsequent server blocks use `listen 443 quic;` without it.
- **`http2 on`**: nginx 1.25+ uses `http2 on` as a directive instead of `listen 443 ssl http2`. Both forms work in 1.25-1.26, but the directive form is forward-compatible.
- **UDP firewall**: HTTP/3 uses UDP 443. If your corporate firewall only allows TCP 443, browsers will fall back to HTTP/2 over TCP automatically. H3 is a progressive enhancement.
- **SSE over H3**: Works transparently. The browser opens the SSE connection over whichever protocol it negotiated (H2 or H3). The nginx-to-app connection is always HTTP/1.1 regardless.
- **Tavern keepalive**: Tavern sends `:keepalive` comments every 30 seconds by default (`WithKeepalive(30*time.Second)`). This keeps the SSE connection alive through proxies that would otherwise close idle connections. The 300s `proxy_read_timeout` is intentionally higher to tolerate temporary network hiccups without dropping clients.
- **Last-Event-ID resumption**: Tavern supports `SubscribeFromID` which uses the `Last-Event-ID` header. This works through nginx with no extra config — the header passes through naturally. If a client reconnects after a dropped connection, it resumes from where it left off.

**Multi-app setup:**

For multiple apps behind one nginx instance, extract the shared config into snippets:

```nginx
# /etc/nginx/snippets/ssl.conf
ssl_certificate     /etc/nginx/ssl/corp-wildcard.crt;
ssl_certificate_key /etc/nginx/ssl/corp-wildcard.key;
ssl_protocols       TLSv1.2 TLSv1.3;
ssl_ciphers         HIGH:!aNULL:!MD5;

# /etc/nginx/snippets/proxy-headers.conf
proxy_set_header Host $host;
proxy_set_header X-Real-IP $remote_addr;
proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
proxy_set_header X-Forwarded-Proto $scheme;

# /etc/nginx/snippets/sse.conf
proxy_read_timeout 300s;
proxy_send_timeout 300s;
proxy_connect_timeout 60s;
proxy_buffering off;
proxy_cache off;
proxy_request_buffering off;
chunked_transfer_encoding off;
gzip off;
proxy_http_version 1.1;
proxy_set_header Connection "";
proxy_set_header X-Accel-Buffering "no";
```

Then each app config becomes:

```nginx
server {
    listen 443 ssl;
    http2 on;
    listen 443 quic;  # no reuseport on secondary server blocks
    server_name shopmaint.corp.example.com;
    include snippets/ssl.conf;
    add_header Alt-Svc 'h3=":443"; ma=86400' always;
    add_header Strict-Transport-Security "max-age=63072000; includeSubDomains; preload" always;

    location /sse/ {
        proxy_pass http://127.0.0.1:33000;
        include snippets/proxy-headers.conf;
        include snippets/sse.conf;
    }
    location / {
        proxy_pass http://127.0.0.1:33000;
        include snippets/proxy-headers.conf;
    }
}
```
