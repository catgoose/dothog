# PHILOSOPHY.md Compliance Audit — dothog Demo App

**Date:** 2026-03-30
**Scope:** `internal/demo/`, `internal/routes/`, `web/views/`, `main.go`

---

## Summary

| Severity | Count | Principle Area |
|----------|-------|----------------|
| Medium | 2 | Resource Identification (verb-based URLs) |
| Medium | 1 | HTTP Method Semantics (POST vs PATCH) |
| Medium | 8 | Mutations Redirect (missing HX-Request guard) |
| Medium | 5 | Schema as Code (raw DDL vs builder) |
| Low | 1 | Content Negotiation (dashboard) |
| Low | 1 | Postel's Law / Security (unparameterized table names) |

---

## Violations

### 1. Verb-based URLs — Resource Identification

**File:** `internal/routes/routes_canvas.go`
**Routes:** `POST /demo/canvas/place`, `POST /demo/canvas/reset`

Both `/place` and `/reset` are verbs describing actions rather than resources. The philosophy states: "URLs identify resources, not actions."

**Suggested fix:**
- `POST /demo/canvas/place` → `POST /demo/canvas/pixels` or `PUT /demo/canvas/pixels/:x/:y`
- `POST /demo/canvas/reset` → `DELETE /demo/canvas` or `PUT /demo/canvas` with empty body

---

### 2. Verb-based URLs — Error Report Status Transitions

**File:** `internal/routes/routes_admin_error_reports.go`
**Routes:** `POST /admin/error-reports/:id/resolve`, `POST /admin/error-reports/:id/dismiss`

The `/resolve` and `/dismiss` suffixes are verbs. These represent state transitions, not resources. Compare with approvals (`PATCH /demo/approvals/:id` with `action` form value) and kanban (`PATCH /demo/kanban/tasks/:id` with `status` form value) which correctly model this.

**Suggested fix:** `PATCH /admin/error-reports/:id` with a `status` form field (`status=resolved` or `status=dismissed`).

---

### 3. POST for Status Transitions Should Use PATCH

**File:** `internal/routes/routes_admin_error_reports.go`
**Routes:** `POST /admin/error-reports/:id/resolve`, `POST /admin/error-reports/:id/dismiss`

POST is defined as "creates a new resource." The philosophy explicitly reserves PATCH for partial modifications. The approvals and kanban routes correctly use PATCH for the same kind of operation.

**Suggested fix:** Change to `PATCH /admin/error-reports/:id` with a status field in the form body.

---

### 4. Missing HX-Request Guards on Mutation Handlers — Mutations Redirect

Multiple PUT/POST/DELETE handlers return inline partial swaps without checking `HX-Request`. A non-HTMX client (e.g., `curl`, standard form submission) would receive a bare HTML fragment instead of a proper redirect.

**Affected handlers:**

| File | Handler | Route |
|------|---------|-------|
| `routes_people.go` | `handlePersonUpdate` | `PUT /demo/people/:id` |
| `routes_inventory.go` | `handleCreateItem` | `POST /demo/inventory/items` |
| `routes_inventory.go` | `handleUpdateItem` | `PUT /demo/inventory/items/:id` |
| `routes_vendors_contacts.go` | `handleContactUpdate` | `PUT /demo/vendors/contacts/:id` |
| `routes_settings.go` | `handleSettingsSave` | `PUT /demo/settings/:id` |
| `routes_canvas.go` | `handlePlace` | `POST /demo/canvas/place` |
| `routes_hypermedia.go` | `handleInteractionsSubmit` | `POST /demo/hypermedia/interactions/submit` |
| `routes_hypermedia.go` | `handleInteractionsComment` | `POST /demo/hypermedia/interactions/comment` |
| `routes_hypermedia.go` | `handleLinksCreate` | `POST /hypermedia/links` |
| `routes_hypermedia.go` | `handleLinksDelete` | `DELETE /hypermedia/links/:id` |

**Suggested fix:** Add content negotiation — for HTMX requests, the inline partial swap is acceptable under the stated exception. For non-HTMX requests, respond with `303 See Other` redirecting to the parent resource.

---

### 5. Raw DDL Instead of Schema-as-Code — Schema as Code

**File:** `internal/demo/db.go`, `internal/demo/people.go`, `internal/demo/vendor_contact.go`, `internal/demo/error_reports.go`, `internal/demo/link_relations.go`

The items, people, vendors, contacts, error_reports, and link_relations tables are all defined using raw `CREATE TABLE IF NOT EXISTS` DDL strings. In contrast, the tasks table (`internal/demo/tasks.go`) correctly uses the schema-as-code pattern with `schema.NewTable("Tasks").Columns(...)` and traits like `WithTimestamps()`, `WithSoftDelete()`, etc.

**Suggested fix:** Migrate the remaining table definitions to use the `schema.NewTable()` builder pattern with appropriate traits, matching the tasks table.

---

### 6. Dashboard Content Negotiation (Low)

**File:** `internal/routes/routes_dashboard.go`

The `/dashboard` route always renders a full base layout. It does not check the `HX-Request` header to return a partial when navigating via HTMX boosted links. Compare with `routes_inventory.go` which correctly checks `hx.IsBoosted(c)`.

**Suggested fix:** If boosted links target `/dashboard`, add a partial response path. Otherwise, this is acceptable.

---

### 7. Unparameterized Table Names in Seed Code (Low)

**File:** `internal/demo/seed.go` — `CopyTable`, `DropMainTable`, `ExecSQL`, `listTableNames`

Table names are interpolated into SQL via `fmt.Sprintf`. While these names come from `sqlite_master` (not user input), the `ExecSQL` function exposes raw SQL execution.

**Suggested fix:** Validate table names against `^[a-zA-Z_][a-zA-Z0-9_]*$` pattern. Consider whether `ExecSQL` needs to exist.

---

## What Is Done Well

- Kanban and approvals correctly use `PATCH` for state transitions with resource-oriented URLs
- Explicit SQL throughout with named parameters — no ORM
- Tasks table uses schema-as-code with traits, serving as a good reference
- Error handling consistently uses `HandleHypermediaError` with controls and recovery options
- Structured observability with request IDs, slog, and promote-on-error trace store
- Static assets served with `Cache-Control: public, max-age=31536000, immutable`; SSE streams set `Cache-Control: no-cache`
- `Vary: HX-Request` header set globally in middleware
- Hypermedia controls use uniform `Control` struct with factory functions
- Server-side state throughout; no client-side state management
- Web standards preferred (SSE over WebSockets, native form submissions, HTML semantics)
