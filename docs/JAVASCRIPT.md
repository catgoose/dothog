# JavaScript in dothog

JavaScript is a first-class tool in this project, used at the correct level of
abstraction. This document covers practical conventions for writing and
organizing JS. For the theoretical foundation — locality of behavior, the
reach-up model, progressive enhancement — see [PHILOSOPHY.md](../PHILOSOPHY.md).

## The reach-up gradient

The project follows a gradient for client-side interactivity:

```
_hyperscript → inline <script> → .js file
```

Each step trades locality of behavior for capability. Take the smallest step
needed.

- **\_hyperscript** — First choice for client-side behavior. Keeps behavior on
  the element. Declarative, self-contained, no orphaned listeners.
- **Inline `<script>` tags** — When \_hyperscript can't express what you need.
  Keep next to the element it relates to. Always use JSDoc.
- **JavaScript files** — Shared utilities, library initialization, complex logic
  that doesn't belong inline. Acceptable — but recognize you've traded locality.
  Keep files small, purpose-specific, always documented with JSDoc.

## When .js files are the right choice

Not everything fits inline. External JS files are correct for:

- **Browser API bridges** — Service workers, IndexedDB, BroadcastChannel,
  EventSource. These are long-lived or need shared state across the page.
- **Library interop** — HTMX extensions, Alpine.js component registrations,
  morph strategies. These must run before the library initializes.
- **\_hyperscript call targets** — Functions exposed on `window` so \_hyperscript
  can call them (e.g. `window._ivUp`). The behavior is declared on the element
  via \_hyperscript; the JS file is the implementation.
- **Dev tooling** — Logging, debug toggles, diagnostic APIs. These are
  developer-facing, not user-facing.

## Window globals

Writing to `window` is acceptable when it serves as a bridge:

- `window._ivUp`, `window._ivDown`, `window._ivPost` — Called from \_hyperscript
  on interval slider elements.
- `window.appChannel` — Shared BroadcastChannel instance read by multiple files.
- `window.htmxLog`, `window.hsDebug` — Dev console APIs.

The rule: globals are bridge points between \_hyperscript/Alpine and JS, or dev
APIs. Never use `window` for app state.

## JSDoc

All JavaScript — inline or external — must use JSDoc. Document every function,
every parameter, every return value. JS without JSDoc is a guessing game.

- **Files**: Use `@fileoverview` or a top-level doc comment explaining purpose.
- **Functions**: `@param`, `@returns`, `@type` as needed.
- **IIFE modules**: Document the module purpose at the top.

Example from `interval-control.js`:

```js
/**
 * Interval control helpers called from hyperscript.
 *
 * Each interval slider wrapper (.iv-wrap) stores configuration in data-*
 * attributes and a `_ivUnit` expando for the current unit index.
 *
 * @module interval-control
 */
```

## Style rules

- Use `const` and `let`. No `var`.
- IIFE pattern `(function() { ... })()` for files that don't need to export
  globals — prevents pollution.
- Prefer browser APIs over libraries (`fetch` over axios, `BroadcastChannel`
  over custom pubsub, `EventSource` over polling).
- No frameworks. No bundlers. No transpilers. Scripts are served as-is.
- Feature-gated files use `// setup:feature:<name>` on line 1.

## CSP compliance

- No `unsafe-eval`. Alpine.js uses the CSP build (`@alpinejs/csp`).
- No `unsafe-inline` in production. Inline scripts in templates must either be
  externalized or covered by a CSP hash.
- External `.js` files loaded via `version.Asset()` are covered by
  `script-src 'self'`.

## File inventory

| File | Description |
|------|-------------|
| `alpine-components.js` | Alpine.js CSP-compatible component registrations |
| `beacon.js` | Client analytics via `navigator.sendBeacon` (demo feature) |
| `broadcast.js` | BroadcastChannel for cross-tab sync (demo feature) |
| `csrf-header.js` | Attaches CSRF token to HTMX requests |
| `debug-restore.js` | Restores debug toggles from localStorage |
| `dev-logging.js` | Development logging for HTMX and \_hyperscript |
| `htmx.alpine-morph.js` | HTMX extension using Alpine.morph for DOM updates |
| `interval-control.js` | Interval slider helpers called from \_hyperscript |
| `offline-indicator.js` | Alpine component for offline detection and sync (offline feature) |
| `sw.js` | Service worker for offline caching and mutation queue (offline feature) |
| `sw-cleanup.js` | Unregisters old service workers and clears caches (offline feature) |
| `sync.js` | IndexedDB-based offline write queue (offline feature) |
| `theme-sync.js` | Syncs server theme to `<html>` after hx-boost swaps |
| `theme-sse.js` | SSE listener for server-sent theme changes (SSE + demo feature) |
| `trusted-types.js` | Trusted Types policy for CSP compliance |
