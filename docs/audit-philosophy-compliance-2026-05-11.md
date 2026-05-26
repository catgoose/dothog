# PHILOSOPHY.md Compliance Audit -- Dothog Demo Application

**Date:** 2026-05-11
**Scope:** Dothog demo application (internal/demo, internal/routes, web/views, web/components)

---

## Executive Summary

The dothog demo application shows strong overall compliance with PHILOSOPHY.md. 8 of 12 core principles fully pass. The main gaps are in accessibility (missing skip link and aria-live attributes), locality of behavior (Alpine.js + external JS files alongside _hyperscript), and a few HTTP method misuses in the hypermedia pattern routes.

---

## Principle-by-Principle Results

| # | Principle | Status | Key Issue |
|---|----------|--------|-----------|
| 1 | Parent routes are documents | **PASS** | `GET /` serves `ArchitecturePage()` directly -- no redirect |
| 2 | Mutations redirect (PRG) | **PARTIAL** | All mutations return 200+swap instead of 303 (deliberate HTMX tradeoff) |
| 3 | Resource identification | **PASS** | One minor verb-URL in noscript fallback (`/items/:id/delete`) |
| 4 | Self-descriptive methods | **PARTIAL** | `POST /patterns/state/like` and `/state/toggle` should be PATCH |
| 5 | Content negotiation | **PASS** | `Vary: HX-Request` set globally; `IsHTMX`/`IsBoosted` checks present |
| 6 | Locality of behavior | **PARTIAL** | Alpine.js + 11 external JS files; large inline script in `canvas.templ` |
| 7 | Accessibility | **PARTIAL** | Missing skip link; missing `aria-live` on swap targets |
| 8 | DaisyUI semantic colors | **PASS** | No raw Tailwind colors found in any template |
| 9 | Postel's law | **PASS** | No HTML `required` attributes; server-side validation only |
| 10 | Uniform interface (CSRF) | **PASS** | `dorman.CSRFProtect` exclusively; no `gorilla/csrf` |
| 11 | Explicit SQL | **PASS** | Named params, no ORM; minor `fmt.Sprintf` for table names |
| 12 | Errors are hypermedia | **PASS** | `HandleHypermediaError` used consistently throughout |

---

## Violations

### 1. Missing skip link (Accessibility -- Medium)

**File:** `web/views/index.templ`

No skip-to-content link (`<a href="#main-content" class="sr-only">Skip to main content</a>`) is present in the layout. The `<main>` element exists but has no `id` attribute for skip-link targeting.

### 2. Missing aria-live on HTMX swap targets (Accessibility -- Medium)

**File:** `web/views/index.templ` and various view templates

`#error-status` correctly has `aria-live="assertive"`. However, other HTMX swap targets like `#people-table-container`, `#inventory-table-container`, `#approvals-list`, `#person-profile-card` do not have `aria-live="polite"`. Screen readers won't announce content changes in these regions.

### 3. POST used where PATCH is correct (Self-Descriptive Methods -- Medium)

**File:** `internal/routes/routes_hypermedia.go`

- `POST /patterns/state/like` increments an existing counter -- should be `PATCH`
- `POST /patterns/state/toggle` flips existing toggle state -- should be `PATCH`
- `POST /interactions/inline-title` updates an existing title -- should be `PUT` or `PATCH`

### 4. Large inline script in canvas.templ (Locality of Behavior -- Medium)

**File:** `web/views/canvas.templ`

Contains a ~120-line inline `<script>` block with a full pixel-canvas controller (Bresenham line drawing, EventSource SSE, fetch calls, DOM manipulation). Also uses `onclick="selectColor(this.dataset.color, this)"` inline handler on palette buttons. This amount of JavaScript should be in an external file with JSDoc.

### 5. Alpine.js + external JS files (Locality of Behavior -- Medium)

**Files:** `web/views/index.templ`, `web/components/core/nav.templ`, `web/assets/public/js/`

The layout uses `x-data="alertListener"` and `x-on:show-alert.window`. The nav uses `x-data="navBar"` and `x-data="navMenuDropdown"`. There are 11+ external JS files including `alpine-components.js`, `broadcast.js`, `interval-control.js`, `sync.js`, etc. The philosophy prefers `_hyperscript` over external JS files.

Note: Alpine.js is endorsed for reactive view state management, and the CSP build requires `Alpine.data()` registration in JS files, creating inherent tension with locality-of-behavior.

### 6. Verb-based URL in noscript fallback (Resource Identification -- Low)

**File:** `internal/routes/routes_inventory.go`

`POST /apps/inventory/items/:id/delete` uses a verb path segment. Only used in `<noscript>` progressive enhancement fallback.

### 7. fmt.Sprintf for SQL table names (Explicit SQL -- Low)

**File:** `internal/demo/db.go`

`Reset()`, `GetSchemaInfo()`, and `columnCount()` use `fmt.Sprintf` to interpolate table names from `sqlite_master`. While the source is trusted, the pattern is fragile.

### 8. Inline style tag in people.templ (Low)

**File:** `web/views/people.templ`

`<style>body:has(#person-page) main { view-transition-name: none !important; }</style>` injects a global body style from inside a partial.

---

## Strengths

- **Exemplary parent route handling** -- `GET /` serves a document directly, all section roots serve documents
- **Strong HTTP method semantics** -- PUT for updates, PATCH for partials, DELETE for removals throughout
- **Global `Vary: HX-Request`** -- set in middleware, all apps benefit
- **Zero raw Tailwind colors** -- all DaisyUI semantic classes
- **No `required` attributes** -- server-side validation only
- **Consistent dorman CSRF** -- no gorilla/csrf dependency
- **Rich error handling** -- `HandleHypermediaError` with five distinct recovery scenarios, `ErrorContext` with controls
- **Noscript fallbacks** -- `web/views/inventory.templ` includes `<noscript>` forms for progressive enhancement

---

## Top 3 Actionable Items

1. **Add skip link and `aria-live` to HTMX swap targets** (Accessibility, Medium)
2. **Change `POST /patterns/state/like` and `/state/toggle` to `PATCH`** (HTTP methods, Medium, easy fix)
3. **Extract `canvas.templ` inline script to external JS with JSDoc** (Locality of behavior, Medium)
