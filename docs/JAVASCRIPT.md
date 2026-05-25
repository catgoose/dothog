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
  Keep next to the element it relates to. Use JSDoc when the script does
  anything non-trivial.
- **JavaScript files** — Shared utilities, library initialization, complex logic
  that doesn't belong inline. Acceptable — but recognize you've traded locality.
  Keep files small, purpose-specific, always documented with JSDoc.

## When .js files are the right choice

Not everything fits inline. External JS files are correct for:

- **Browser API bridges** — EventSource or similar
  long-lived browser APIs that need shared state across the page.
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
- `window.themeController` — Global theme bridge that owns canonical theme application
  plus the async HTMX send path for the theme picker.
- `window.htmxLog`, `window.hsDebug` — Dev console APIs.

The rule: globals are bridge points between \_hyperscript/Alpine and JS, or dev
APIs. Never use `window` for app state.

## JSDoc

Owned JavaScript should use JSDoc where it materially improves readability.
Start with file-level `@fileoverview` comments and add function/parameter docs
for public hooks, exported globals, event bridges, and other non-obvious code.

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
- Prefer browser APIs over libraries (`fetch` over axios, `EventSource`
  over polling).
- No frameworks. No bundlers. No transpilers. Scripts are served as-is.
- Feature-gated files use `// setup:feature:<name>` on line 1.

## CSP compliance

- No `unsafe-eval`. Alpine.js uses the CSP build (`@alpinejs/csp`).
- No inline event handlers (`onclick`, `onsubmit`, `onchange`) on
  scaffold-owned surfaces. Replace them with `_hyperscript` (`_=`),
  Alpine bindings on a registered component (`x-on:click="method()"`), or
  HTMX attributes. The theme picker and error-status clipboard already
  follow this convention.
- External `.js` files loaded via `version.Asset()` are covered by
  `script-src 'self'`.
- The `csp` setup feature ships the strict
  `script-src 'self'` policy described in
  [docs/SECURITY.md](SECURITY.md#content-security-policy-setupfeaturecsp)
  via the `CSP_HEADER` env var; the contract is config-driven so source-repo
  runs never accidentally emit it. The feature is mutually exclusive with
  `demo` (rejected by `setup.Run`) because demo views still carry inline
  `<script>` blocks and handlers that a strict policy would block.

## Asset layout

`web/assets/public/js/` is split into three groups so the path immediately
identifies what kind of file it is:

- **`vendor/`** — Upstream third-party libraries shipped as-is. Do not edit
  these files; replace them when bumping the upstream version.
- **`app/`** — Dothog-owned runtime assets. Some always load; others are
  gated by `// setup:feature:<tag>` markers in `web/views/index.templ`, so a
  derived app only ships the files for features it selected.
- **`dev/`** — Dothog-owned debug tooling. The files here exist for the
  developer experience, but their load conditions are file-specific —
  `debug-restore.js` rehydrates persisted debug flags on every page load,
  while `dev-logging.js` only loads when `devMode` is active.

### `vendor/` — third-party bundles

| File | Description |
|------|-------------|
| `alpine.min.js` | Alpine.js CSP build |
| `_hyperscript.min.js` | \_hyperscript runtime |
| `htmx.ext.sse.js` | HTMX SSE extension |
| `htmx.min.js` | HTMX core |
| `tavern.min.js` | Tavern client |

### `app/` — owned runtime

Shell behaviors that only react to local DOM events — toast spawning,
nav dropdown exclusivity, close-on-outside-click — live in `_hyperscript`
on the element that owns them. Alpine is reserved for coordinated client
state that spans multiple DOM regions. Feature-local Alpine components
ship in their own `*.alpine.js` file and register through
`dothog.alpine.register(name, factory)` exposed by `alpine-helper.js`, so
the helper can queue them until `alpine:init` fires. The helper exists for
CSP-safe named registrations; without CSP, inline `x-data="{...}"` would
also be viable.

| File | Feature gate | Description |
|------|--------------|-------------|
| `alpine-helper.js` | always | Exposes `window.dothog.alpine.register(name, factory)` and queues registrations until `alpine:init` |
| `csrf-header.js` | always | Attaches CSRF token to HTMX requests |
| `htmx.error-capabilities.js` | always | Advertises accepted error surfaces on HTMX requests |
| `trusted-types.js` | always | Trusted Types policy for CSP compliance |
| `interval-control.js` | `demo` | Interval slider helpers called from \_hyperscript |
| `theme-controller.js` | `session_settings` | Private theme controller. Owns the canonical `applyTheme`, syncs every `[data-theme-picker]` in place, sends picker edits via HTMX AJAX, and delegates `submit`/`change`/`click` on the document so the picker form is plain data-marked HTML. Carries an inner `sse:start/end` block that opens the single `/sse/theme` EventSource and republishes events as `app:theme-change`. No `window` export. |
| `admin-sessions.alpine.js` | `session_settings` | Page-scoped Alpine registration for `x-data="sessionsSelection"` on /admin/sessions. Owns the outer selection state so HTMX/SSE table refreshes don't destroy per-row checkbox state; reconciles on `htmx:afterSwap`. |

### `dev/` — debug tooling

| File | Load condition | Description |
|------|----------------|-------------|
| `debug-restore.js` | every page | Rehydrates persisted debug toggles from `localStorage` so flags survive reload |
| `dev-logging.js` | when `devMode` | Development logging for HTMX and \_hyperscript |
