# PHILOSOPHY.md Compliance Audit -- Dothog Demo App

**Date:** 2026-04-20
**Scope:** Demo application routes, handlers, templates, and components

---

## Summary

The dothog demo app demonstrates strong adherence to PHILOSOPHY.md across most principles. The error handling system is exemplary -- every error is a navigable hypermedia state with contextual recovery controls. The uniform interface, content negotiation, Postel's Law, explicit SQL, and server-side state principles are all well-implemented.

**13 specific violations found across 6 principle categories.**

| Category | Severity | Count |
|----------|----------|-------|
| Accessibility | High | 3 |
| Resource Identification | Medium | 3 |
| Self-Descriptive Methods | Medium | 2 |
| Explore First, Normalize Later | Medium | 3 |
| DaisyUI Semantic Colors | Low | 1 |
| Mutations Redirect | Low | 1 |

---

## Violations

### 1. Accessibility (High)

#### 1a. Missing `aria-live` on AppNavLayout error target

**File:** `web/views/app_nav_layout.templ`

The `<div id="error-status"></div>` swap target lacks `aria-live="assertive"`. Compare with `web/views/index.templ` where the equivalent element correctly has `aria-live="assertive"`. Screen readers will not announce error banners rendered into the AppNavLayout, which is the primary layout.

**Fix:** Add `aria-live="assertive"` and `role="alert"` to the `#error-status` div in `app_nav_layout.templ`.

#### 1b. Missing skip link

**Files:** `web/views/app_nav_layout.templ`, `web/views/index.templ`

Neither layout includes a "Skip to main content" link. The philosophy requires skip links for keyboard navigation. The `<main>` element exists, but there is no mechanism to skip past navigation.

**Fix:** Add `<a href="#main" class="sr-only focus:not-sr-only focus:absolute focus:z-50 focus:p-2">Skip to main content</a>` as the first focusable element in both layouts.

#### 1c. Missing `<header>` semantic landmark

**File:** `web/views/app_nav_layout.templ`

The sticky header group uses `<div class="app-header ...">` instead of `<header>`. The philosophy calls for semantic landmarks (`nav`, `main`, `header`, `footer`). `<nav>`, `<main>`, and `<footer>` are all present, but `<header>` is missing.

**Fix:** Replace the outer `<div class="app-header ...">` with `<header class="app-header ...">`.

---

### 2. Resource Identification (Medium)

#### 2a. `POST /admin/db/reinit` -- verb-based URL

**File:** `internal/routes/routes_admin.go`

The URL contains the verb "reinit". A more resource-oriented approach: `POST /admin/db` (to re-create the database resource) or `PUT /admin/db` (to replace it).

#### 2b. `GET /admin/system/check-update` -- verb-based URL

**File:** `internal/routes/routes_admin_core.go`

The URL contains the verb "check". Could be modeled as `GET /admin/system/update` (the update-availability resource).

#### 2c. `PUT /apps/bulk/items/activate` and `PUT /apps/bulk/items/deactivate` -- verb-based URLs

**File:** `internal/routes/routes_bulk.go`

These URLs contain verbs. A more resource-oriented approach: `PATCH /apps/bulk/items` with a body indicating the desired state change.

---

### 3. Self-Descriptive Methods (Medium)

#### 3a. `POST /realtime/canvas/reset` -- POST for idempotent operation

**File:** `internal/routes/routes_canvas.go`

Resetting the canvas is idempotent. Should be `DELETE /realtime/canvas` (clearing the resource) or `PUT /realtime/canvas` (replacing with empty state).

#### 3b. POST fallback routes duplicate PUT/DELETE semantics

**File:** `internal/routes/routes_inventory.go`

```go
ar.e.POST(inventoryBase+"/items/:id", d.handleUpdateItem)         // POST fallback for PUT
ar.e.POST(inventoryBase+"/items/:id/delete", d.handleDeleteItem)  // POST fallback for DELETE
```

The POST fallbacks at separate URLs muddy the self-descriptive method contract. The `/items/:id/delete` URL also introduces a verb. Note: the philosophy does acknowledge the Form Method Gap and says POST fallbacks are acceptable for progressive enhancement -- but the `/delete` verb segment is still a violation of Resource Identification.

---

### 4. Explore First, Normalize Later (Medium)

#### 4a. Warning triangle SVG duplicated 8+ times

**Files:** `web/components/core/error_controls.templ`, `error_status.templ`, `error_page.templ`, `controls.templ`

The warning triangle SVG icon (`M12 9v3.75m-9.303 3.376c-.866 1.5...`) is copy-pasted verbatim in at least 8 places across `reportButton`, `genericReportButton`, `inlineErrorActions`, `inlineFullMD`, `inlineFullLG`, `inlineFullXL`, etc. This exceeds the 3+ consolidation threshold.

**Fix:** Extract into a shared `warningTriangleIcon()` templ component.

#### 4b. Inline error button class string duplicated 12+ times

**File:** `web/components/core/error_status.templ`

The class string `"btn border-error/20 bg-error/10 text-error hover:bg-error/20"` appears in at least 12 places across `inlineFullXS`, `inlineFullSM`, `inlineFullMD`, `inlineFullLG`, `inlineFullXL`, `inlineFull2XL`, `inlineFull3XL`, and `inlineFullActions`.

**Fix:** Extract into a helper function like `inlineErrorBtnClass()`.

#### 4c. Clipboard copy button pattern approaching threshold

**Files:** `web/components/core/error_page.templ`, `error_status.templ`

The clipboard copy button with SVG icon and hyperscript appears in 2-3 places with minor variations. Approaching the consolidation threshold.

---

### 5. DaisyUI Semantic Colors (Low)

#### 5a. Inline CSS bypassing DaisyUI semantic classes

**File:** `web/views/settings.templ`

Contains an inline `<style>` block with raw CSS custom properties:

```css
.settings-tab { color: color-mix(in oklab, var(--color-base-content) 70%, transparent); }
.settings-tab:hover { background: var(--color-base-200); }
.settings-tab.active { background: color-mix(in oklab, var(--color-primary) 10%, transparent); }
```

While this uses DaisyUI CSS variables (not raw hex colors), it bypasses DaisyUI's semantic class system. The settings tabs could use DaisyUI's `tab`, `menu`, or `btn-ghost`/`btn-primary` classes.

---

### 6. Mutations Redirect (Low)

#### 6a. `POST /admin/db/reinit` returns partial instead of redirecting

**File:** `internal/routes/routes_admin.go`

This significant side-effecting operation (database reset) returns a `views.AdminDBStatus(...)` component rather than redirecting to `/admin/sqlite`. Given the magnitude of the operation, a redirect would better communicate the state transition.

---

## Principles with No Violations

- **Uniform Interface** -- `linkwell.Control` used consistently with factory functions
- **Locality of Behavior** -- Reach-up model followed well (HTMX -> _hyperscript -> Alpine inline)
- **Errors Are Hypermedia** -- Exemplary implementation with contextual recovery controls
- **Content Negotiation** -- HX-Request checks, Vary header, noscript fallback all present
- **Web Standards Over Libraries** -- Native `<dialog>`, `<details>`, `<datalist>` used
- **Postel's Law** -- Server-side validation only, no pattern/oninvalid attributes
- **Explicit SQL** -- chuck composable helpers, no ORM
- **Server-Side State** -- No client-side routing or state management
- **Parent Routes Are Documents** -- All parent routes serve representations
