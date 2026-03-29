# PHILOSOPHY.md Compliance Audit

Audit date: 2026-03-29

## V1: Init-time panic in demo seeder

**File:** `internal/demo/seed.go`
**Status:** Deferred
**Rationale:** The panic occurs only during demo data seeding at startup. If seeding
fails, the application cannot serve demo pages and crashing is the correct behavior.
This aligns with the Go proverb "Don't panic" exception for truly unrecoverable
programmer-level errors at init time.

## V2: Inline HTMX partial swaps bypass PRG (kanban, approvals, repository)

**Status:** Not a violation
**Rationale:** Inline HTMX partial swaps (editing a row in-place, moving a card
between columns) are a valid hypermedia interaction pattern distinct from PRG. The
server controls the next state and returns the updated representation directly to the
target element. PRG applies to navigation mutations (creating a resource with its own
URL). Inline swaps are for non-navigation mutations where the user stays on the same
view. PHILOSOPHY.md updated to document this exception.

## V3: Inline HTMX partial swaps in CRUD demo

**Status:** Not a violation
**Rationale:** Same as V2. The CRUD demo's inline edit/create/toggle/delete pattern
uses correct HTTP verbs (POST for create, PUT for full update, PATCH for toggle,
DELETE for remove) and returns updated representations directly. This is proper
hypermedia.

## V4: Verb URL `/demo/kanban/tasks/:id/move`

**File:** `internal/routes/routes_kanban.go`, `web/views/kanban.templ`
**Status:** Resolved
**Fix:** Route changed from `PATCH /demo/kanban/tasks/:id/move` to
`PATCH /demo/kanban/tasks/:id`. Status is now read from the request body via
`FormValue("status")` instead of a query parameter. Templates updated to use
`hx-vals` to send status in the request body.

## V5: Verb URL `/demo/approvals/:id/:action`

**File:** `internal/routes/routes_approvals.go`, `web/views/approvals.templ`
**Status:** Resolved
**Fix:** Route changed from `POST /demo/approvals/:id/:action` to
`PATCH /demo/approvals/:id`. Action is now read from the request body via
`FormValue("action")`. Templates updated to use `hx-patch` with `hx-vals` to send
the action in the request body.

## V6: Verb URLs for repository restore/archive

**File:** `internal/routes/routes_repository.go`, `web/views/repository.templ`
**Status:** Resolved
**Fix:** Three separate routes (`POST .../restore`, `POST .../archive`,
`POST .../unarchive`) consolidated into a single `PATCH /demo/repository/tasks/:id`
endpoint. The action is read from the request body via `FormValue("action")`.
Templates updated to use `hx-patch` with `hx-vals`.

## V7: Verb URL for repository unarchive

**Status:** Resolved (merged with V6)

## V8: Alpine `x-on:click` in control buttons

**File:** `web/components/core/controls.templ`, `web/components/core/error_controls.templ`
**Status:** Resolved
**Fix:** Converted `backButton`, `homeButton`, and `genericDismiss` in `controls.templ`
from Alpine `x-on:click` to _hyperscript `_="on click ..."`. Also converted
`errorBackButton` in `error_controls.templ`. The `dismissButton` in
`error_controls.templ` already used _hyperscript and was not changed.

## V9: DELETE handler returns 200 instead of 204

**File:** `internal/routes/routes_hypermedia.go`
**Status:** Resolved
**Fix:** Changed `c.NoContent(200)` to `c.NoContent(http.StatusNoContent)` (204) in
`handleCRUDDelete`. Updated the delete button's swap strategy from
`hx-swap="outerHTML"` to `hx-swap="delete"` so the row is removed from the DOM when
the server returns 204 with no body. Tests updated to expect 204.
