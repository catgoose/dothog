# PHILOSOPHY.md Compliance Audit -- Dothog Demo

Audit date: 2026-05-04

The dothog demo application audited against [PHILOSOPHY.md](../PHILOSOPHY.md) principles.

## Summary

| Severity | Count | Principle Areas |
|----------|-------|-----------------|
| HIGH | 0 | -- |
| MEDIUM | 4 | Locality of Behavior, Postel's Law, Content Negotiation, Accessibility |
| LOW | 5 | DaisyUI classes, LoB, Route registration, Accessibility (x2) |

**Overall Assessment:** The demo application is strongly aligned with PHILOSOPHY.md. The architecture consistently applies hypermedia controls via `linkwell.Control`, uses noun-based resource URLs, explicit SQL, server-side validation, semantic DaisyUI classes, and proper content negotiation. No high-severity violations found.

## Compliant Areas (Passed)

- **Resource Identification** (Principle #1) -- All URLs are noun-based: `/apps/inventory/items/:id`, `/apps/people/:id`, `/apps/kanban/tasks/:id`, `/platform/settings/:id`, `/apps/vendors/:id/contacts`. No verb-based URLs found.
- **Self-Descriptive Methods** (Principle #3) -- Correct HTTP methods throughout. POST fallbacks for PUT/DELETE are documented as progressive enhancement (`<noscript>` forms).
- **Uniform Interface** (Principle #6) -- Exemplary use of `linkwell.Control` structs and factory functions (`TableRowActions`, `ResourceActions`, `RowFormActions`, `RetryButton`, `DismissButton`, `ReportIssueButton`).
- **Explicit SQL** (Principle #11) -- All queries use `database/sql` with named parameters or `chuck/dbrepo` composable helpers (NewSelect, NewWhere, InsertInto, SetClause). No ORM.
- **Structured Observability** (Principle #17) -- Request IDs threaded through all error responses. Promote-on-error logging via promolog. `ReportIssueButton` receives request ID for correlation.
- **Schema as Code** (Principle #14) -- Table definitions use `chuck/schema` with traits.
- **Domain Patterns as Primitives** (Principle #15) -- Small composable functions, no base classes.

## Violations

### MEDIUM: Canvas Inline Script Pushes LoB Boundary (Principle #8)

- **File:** `web/views/canvas.templ`
- **Details:** ~120-line inline `<script>` block with pixel-painting controller, Bresenham's line algorithm, SSE handling, and fetch calls. Per the reach-up model (HTML -> HTMX -> _hyperscript -> Alpine -> inline script -> .js file), this is at the boundary of what should remain inline.
- **Fix:** Consider extracting the `selectColor` function and palette interaction to `_hyperscript` or Alpine. The SSE and canvas rendering legitimately need JS.

### MEDIUM: Calendar `required` Attribute (Principle #10 Postel's Law)

- **File:** `web/views/calendar.templ` -- `calendarAddEventForm`
- **Code:** `<input type="text" name="title" ... required />`
- **Details:** The `required` attribute enforces client-side validation. PHILOSOPHY.md states: forms should be permissive, server validates. The server already checks `if title == ""` in `routes_calendar.go`.
- **Fix:** Remove the `required` attribute.

### MEDIUM: Calendar Plain-Text Error Responses (Principle #5 Content Negotiation)

- **File:** `internal/routes/routes_calendar.go` -- `handleAddEvent`, `handleDeleteEvent`, `handleDay`
- **Code:** `return c.String(http.StatusBadRequest, "missing or invalid d")`
- **Details:** Returns plain text without checking HX-Request for partial vs full responses. Should return hypermedia error components.
- **Fix:** Replace `c.String(...)` calls with `handler.HandleHypermediaError(c, 400, "missing or invalid d", nil)`.

### MEDIUM: Clickable Table Rows Without Keyboard Support (Principle #12 Accessibility)

- **File:** `web/views/people.templ` -- `PersonRow`
- **Code:** `<tr class="hover:bg-base-200/50 cursor-pointer" hx-get={...} hx-target="body" hx-push-url="true">`
- **Details:** Entire row is clickable via `hx-get` but has no `role="link"`, `tabindex="0"`, or keyboard handler. Keyboard users cannot focus or activate these rows.
- **Fix:** Add `tabindex="0"` and `role="link"`, and extend trigger: `hx-trigger="click, keydown[key=='Enter']"`. Or wrap row content in an anchor within the first `<td>`.

### LOW: Dynamic Class Construction May Break Tailwind Purge (Principle #9)

- **File:** `web/views/dashboard.templ` -- `statBadge`
- **Code:** `"text-" + color` constructs classes dynamically (e.g., `text-primary`, `text-warning`)
- **Fix:** Use explicit full class strings or safelist the dynamically constructed classes in Tailwind config.

### LOW: Calendar Global Function Instead of _hyperscript (Principle #8 LoB)

- **File:** `web/views/calendar.templ` -- `calendarLegend`
- **Code:** Global `markCalendarSelected(btn)` function referenced via `hx-on::after-request`
- **Fix:** Replace with `_hyperscript` on the button element for element-scoped behavior.

### LOW: Root Route Double-Registration (Principle #2)

- **File:** `internal/routes/routes.go`
- **Code:** `ar.e.GET("/", handler.HandleComponent(views.HomePage(...)))` followed by `ar.e.GET("/", handler.HandleComponent(views.ArchitecturePage()))` in the demo block
- **Details:** First registration is dead code, silently overridden by the demo feature block.
- **Fix:** Remove the first registration or gate with an `else` branch.

### LOW: Missing `aria-live` on Error Status Container (Principle #12)

- **File:** `web/views/app_nav_layout.templ`
- **Element:** `<div id="error-status">` -- OOB swap target for error banners
- **Fix:** Add `aria-live="polite"` (or `aria-live="assertive"` with `role="alert"` for errors).

### LOW: Kanban Arrow Buttons Missing `aria-label` (Principle #12)

- **File:** `web/views/kanban.templ` -- `KanbanCard`
- **Details:** Move buttons use arrow symbols without explicit `aria-label`. Screen readers would read the raw unicode characters.
- **Fix:** Add `aria-label="Move to In Progress"` (etc.) to each move button.
