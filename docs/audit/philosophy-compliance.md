# PHILOSOPHY.md Compliance Audit -- Dothog Demo Application

**Date:** 2026-03-29
**Scope:** Demo application (`internal/demo/`, `internal/routes/`, `web/views/`, `web/components/`)
**Audited against:** `PHILOSOPHY.md` (22 principles)

## Summary

The dothog demo application demonstrates **strong overall compliance** with its philosophy document. The architecture is genuinely hypermedia-driven, the `Control` struct uniform interface is well-implemented, error handling is exemplary, SQL is explicit, and traits/composable helpers replace ORM abstractions effectively. The violations found are relatively minor.

**Compliance score: ~87%** (9 violations across 22 principles, most low-severity)

---

## Violations Found

### V1: `panic` in `must()` helper (Low)
- **File:** `main.go:47-52`
- **Principle:** #20 -- No panic for runtime conditions
- **Description:** `func must(fs fs.FS, err error) fs.FS` panics if embedded filesystem fails. While init-time only, the philosophy draws a hard line on `panic`.
- **Fix:** Replace with `log.Fatal("failed to sub static FS: ", err)` or document as acceptable init-time invariant.

### V2: POST handlers return components instead of redirecting (Medium)
- **File:** `internal/routes/routes_inventory.go:68-75` (`handleCreateItem`), `routes_repository.go` (`handleCreateTask`), `routes_hypermedia.go` (`handleCRUDCreate`)
- **Principle:** #8 -- Mutations redirect (POST/Redirect/GET)
- **Description:** POST handlers respond with rendered components (200 OK) rather than `HX-Redirect` + 303. This is idiomatic HTMX for inline table updates but deviates from PRG.
- **Fix:** Either align code with strict PRG, or clarify in PHILOSOPHY.md that inline HTMX partial updates are an acceptable exception to PRG for non-navigation operations.

### V3: PUT handlers return components instead of redirecting (Medium)
- **File:** `internal/routes/routes_people.go:104` (`handlePersonUpdate`), `routes_inventory.go` (`handleUpdateItem`), `routes_vendors_contacts.go` (`handleContactUpdate`)
- **Principle:** #8 -- Mutations redirect
- **Description:** Same pattern as V2 for PUT operations.
- **Fix:** Same as V2.

### V4: Verb-based URL `/demo/kanban/tasks/:id/move` (Low)
- **File:** `internal/routes/routes_kanban.go:31`
- **Principle:** #4 -- Resource identification (no verb-based URLs)
- **Fix:** Change to `PATCH /demo/kanban/tasks/:id` with target status in request body.

### V5: Verb-based URL `/demo/approvals/:id/:action` (Low)
- **File:** `internal/routes/routes_approvals.go:31`
- **Principle:** #4 -- Resource identification
- **Fix:** Change to `PATCH /demo/approvals/:id` with action in request body.

### V6: Verb-based URLs for repository tasks (Low)
- **File:** `internal/routes/routes_repository.go:37-39`
- **URLs:** `POST .../restore`, `POST .../archive`, `POST .../unarchive`
- **Principle:** #4 -- Resource identification
- **Fix:** Use `PATCH /demo/repository/tasks/:id` with desired state in body.

### V7: POST used for state modifications instead of PATCH (Medium)
- **File:** `internal/routes/routes_repository.go:37-39`
- **Principle:** #3 -- Self-descriptive HTTP methods (POST creates, PATCH modifies)
- **Description:** `POST .../restore`, `POST .../archive`, `POST .../unarchive` use POST for what are semantically partial modifications.
- **Fix:** Use `PATCH` instead of `POST` for these operations.

### V8: Alpine.js used where _hyperscript preferred (Low)
- **File:** `web/components/core/controls.templ` (`backButton`, `homeButton`, `dismissButton`)
- **Principle:** #13 -- Locality of behavior (_hyperscript for client behavior)
- **Description:** Uses `x-on:click` (Alpine) instead of `_hyperscript`. The philosophy allows Alpine for view state but prefers _hyperscript.
- **Fix:** Migrate to `_hyperscript` or document Alpine as acceptable for simple event handlers.

### V9: DELETE returns 200 instead of 204 (Very Low)
- **File:** `internal/routes/routes_hypermedia.go:172` (`handleCRUDDelete`)
- **Principle:** #3 -- Self-descriptive HTTP methods
- **Description:** `c.NoContent(200)` should be `c.NoContent(http.StatusNoContent)` (204).
- **Fix:** Change to 204. Verify HTMX handles 204 correctly with `hx-swap="delete"`.

---

## Areas of Strong Compliance

| Principle | Assessment |
|-----------|------------|
| #1 Hypermedia-driven (HTMX, not SPA) | **Excellent** -- Zero SPA patterns, all server-rendered |
| #2 Uniform interface (`hypermedia.Control`) | **Excellent** -- `Control` struct with factory functions used everywhere |
| #5 Parent routes are documents | **Good** -- `/demo`, `/hypermedia`, `/dashboard` all serve documents |
| #6 Server-side state | **Excellent** -- All state server-side (SQLite, in-memory stores, SSE) |
| #7 Content negotiation (HX-Request) | **Good** -- `Vary: HX-Request` set globally, partials vs full pages |
| #9 Postel's Law (server validates) | **Good** -- No client-side validation, inline error panels |
| #10 Explicit SQL, composable helpers | **Excellent** -- `SelectBuilder`, `WhereBuilder`, `Columns()`, no ORM |
| #11 Schema as code with traits | **Excellent** -- `NewTable().WithStatus().WithSortOrder().WithVersion()...` |
| #12 Domain patterns as primitives | **Good** -- Standalone functions, no base classes |
| #14 Errors are hypermedia | **Excellent** -- `ErrorContext` with controls, request ID, report button |
| #15 Structured observability | **Good** -- Request IDs, `promolog.Store`, promote-on-error |
| #16 DaisyUI semantic classes | **Excellent** -- `btn btn-primary`, `alert alert-error`, no raw colors |
| #17 Native HTML over JS | **Good** -- `<dialog>`, native `<select>`, `<input>` elements |
| #19 Link relations | **Excellent** -- `rel="up"` chains, `BreadcrumbsFromLinks()`, link registry |
| #22 Go principles | **Good** -- Clear code, small interfaces, error wrapping |
