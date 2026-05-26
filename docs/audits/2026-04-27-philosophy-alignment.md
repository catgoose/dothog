# Philosophy Alignment Audit — Dothog Demo — 2026-04-27

Audit of the dothog demo application against
[PHILOSOPHY.md](https://github.com/catgoose/dothog/blob/main/PHILOSOPHY.md).

## Result: Pass

The dothog demo application demonstrates exemplary conformance with every
principle in PHILOSOPHY.md. No violations found.

## Conformance Details

### HTTP Method Semantics

Full HTTP method vocabulary correctly applied:

- `GET` — safe, idempotent reads for all page and component loads
- `POST` — creates (with POST fallbacks for browser compatibility)
- `PUT` — full resource updates (e.g., `PUT /items/:id`)
- `PATCH` — partial updates (e.g., `PATCH /items/:id/toggle`)
- `DELETE` — resource removal (with `POST /items/:id/delete` fallback)

Route registration in `internal/routes/routes_inventory.go`:

```go
ar.e.GET(inventoryBase+"/items", d.handleInventoryItems)
ar.e.POST(inventoryBase+"/items", d.handleCreateItem)
ar.e.PUT(inventoryBase+"/items/:id", d.handleUpdateItem)
ar.e.POST(inventoryBase+"/items/:id", d.handleUpdateItem)     // POST fallback
ar.e.DELETE(inventoryBase+"/items/:id", d.handleDeleteItem)
ar.e.POST(inventoryBase+"/items/:id/delete", d.handleDeleteItem) // POST fallback
```

POST fallbacks for PUT and DELETE honor the _Form Method Gap_ section of the
philosophy — HTMX clients use correct methods; `<noscript>` forms use POST.

### Parent Routes Are Documents

All parent routes render full page documents:

- `/apps/inventory` — renders table with filter bar
- `/apps/people` — renders list with card details
- `/patterns/crud` — renders CRUD demo page
- `/api/links` — renders links editor
- `/demo` — renders discovery page for all demo sections

No parent route redirects to a child.

### Uniform Interface — Controls

Controls are used extensively via the `linkwell.Control` struct and factory
functions:

- `FormActions()`, `RowActions()`, `ResourceActions()` for CRUD forms
- `ErrorControlsForStatus()` for status-code-appropriate error recovery
- `DismissButton()`, `ReportIssueButton()`, `RetryButton()`, `BackButton()`
- Error responses in `routes_hypermedia_errors.go` compose controls:

```go
ec := linkwell.ErrorContext{
    Controls: []linkwell.Control{
        linkwell.DismissButton(linkwell.LabelDismiss),
        linkwell.ReportIssueButton(linkwell.LabelReportIssue, requestID),
        linkwell.RetryButton("Retry", linkwell.HxMethodGet, base+"/flaky", "#errors-retry-result"),
    },
}
```

### Content Negotiation

HX-Request header checking is applied throughout:

```go
if htmx.IsHTMX(c.Request()) && !htmx.IsBoosted(c.Request()) {
    return handler.RenderComponent(c, views.InventoryItemRow(item))
}
return handler.RenderBaseLayout(c, views.InventoryDetailPage(item))
```

Boosted links receive full layout; explicit `hx-get` requests receive
fragments. Same resource, different representations.

### Errors Are Hypermedia

Full error-as-hypermedia pipeline implemented:

- `ErrorContext` struct with `Controls`, `OOBTarget`, `RequestID`, `Closable`
- Form validation returns `422 Unprocessable Content` with the form and
  inline errors — not a redirect, not a generic error page
- OOB error banner for global errors with Report Issue and Close controls
- Flaky endpoint demo with Retry control
- `HandleHypermediaError()` in handler adds status-code-appropriate default
  controls when none are provided

### Server-Side State

No client-side state management detected:

- No JavaScript framework imports
- No `useState`, Redux, Zustand, or any client state library
- Form state transmitted via URL parameters (filters, pagination)
- Session settings stored server-side
- UI state maintained via response HTML attributes

### Link Relations

linkwell integration drives navigation:

- Hub/Ring/Link declarations at startup
- RFC 8288 `Link` headers emitted by middleware on every response
- Breadcrumbs walk `rel="up"` chain with priority: `?from=` > `rel="up"` > URL path
- Context bars, site map footer, and registry inspector share one data source

### Explicit SQL and Schema as Code

- Chuck's `SelectBuilder`, `WhereBuilder`, `SetClause` for composable SQL
- Table definitions via `schema.NewTable().Columns(...)` with trait functions
  (`WithTimestamps()`, `WithSoftDelete()`, `WithVersion()`)
- Domain patterns as functions (`SetSoftDelete()`, `IncrementVersion()`)

### Accessibility

Semantic landmarks present throughout core components:

- `<nav role="navigation" aria-label="App navigation">` in context bar
- `<main role="main">` for primary content
- `<table role="table">` with `<thead role="rowgroup">` for data grids
- `aria-label` on navigation elements
- Form labels with proper associations

### Locality of Behavior

- HTMX attributes on elements (`hx-get`, `hx-post`, `hx-target`, `hx-swap`)
- `_hyperscript` for client-side DOM behavior (dismiss, transitions)
- DaisyUI semantic classes for styling intent
- No behavior-in-separate-.js-file patterns detected

### CSS Over JavaScript

- Tailwind CSS + DaisyUI for all styling
- No inline styles or custom JavaScript for DOM manipulation
- Theme-aware via DaisyUI semantic color roles
- Class toggles via response HTML, not JS state

## Cross-Reference

The dothog demo is audited as the upstream reference implementation. The
companion audit of all vopts applications is at:
[catgoose/vopts docs/audits/2026-04-27-philosophy-alignment.md](https://github.com/catgoose/vopts/blob/audit/philosophy-alignment-2026-04-27/docs/audits/2026-04-27-philosophy-alignment.md)
