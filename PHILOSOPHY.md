# Design Philosophy

<!--toc:start-->

- [Design Philosophy](#design-philosophy)
  - [Hypermedia-Driven Architecture](#hypermedia-driven-architecture)
  - [Uniform Interface](#uniform-interface)
  - [Server-Side State, Client-Side Rendering](#server-side-state-client-side-rendering)
  - [The Stack](#the-stack)
  - [Why Not an SPA?](#why-not-an-spa)
  - [Explicit SQL, Composable Helpers](#explicit-sql-composable-helpers)
  - [Schema as Code](#schema-as-code)
  - [Domain Patterns as Primitives](#domain-patterns-as-primitives)
  - [Errors Are Hypermedia](#errors-are-hypermedia)
  - [Structured Observability](#structured-observability)
  <!--toc:end-->

## Hypermedia-Driven Architecture

This application is built on the principles of [REST as Roy Fielding defined it](https://roy.gbiv.com/pubs/dissertation/fielding_dissertation.pdf) — not the bastardized "REST API" that the industry settled on, but actual REST: **Representational State Transfer** through hypermedia. As Fielding himself clarified, [REST APIs must be hypertext-driven](https://roy.gbiv.com/untangled/2008/rest-apis-must-be-hypertext-driven).

The server returns HTML with embedded hypermedia controls that tell the client what actions are available and how to invoke them. The client doesn't need to know anything about the API surface ahead of time. Every transition is discoverable from the current representation. This is HATEOAS (Hypermedia As The Engine Of Application State).

**You do not need a single-page application.** The browser is already a hypermedia client. HTMX extends it to handle the cases where a full page reload is wasteful. The result is a simpler, more maintainable architecture that leverages what the web was designed to do.

## Uniform Interface

Controls (buttons, links, form actions) share a uniform interface via the `hypermedia.Control` struct:

```go
type Control struct {
	Kind      ControlKind     // how to render: link, button, back, dismiss
	Label     string          // what the user sees
	Variant   ControlVariant  // visual emphasis: primary, ghost, danger
	HxRequest HxRequestConfig // method, url, target, include
	Confirm   string          // optional confirmation gate
	Swap      SwapMode        // HTMX swap strategy
	// ...
}
```

The server builds `[]Control` slices using factory functions (`FormActions`, `RowActions`, `ResourceActions`). The templ `Controls` component renders them. The client never hard-codes which buttons exist or what they do — the server tells it.

This means:

- **Cancel** always works the same way everywhere (link back to the cancel href)
- **Save** always works the same way everywhere (submit the closest form)
- **Delete** always works the same way everywhere (confirm, then fire the request)
- Error responses include their own recovery controls (retry, dismiss, go home)

New pages get correct behavior by composing existing control patterns rather than hand-wiring HTMX attributes.

## Server-Side State, Client-Side Rendering

State lives on the server. The client is a thin rendering layer. When state changes, the server sends new HTML. The browser's job is to display it and let the user interact with the controls embedded in it.

No client-side routing. No client-side state management. No `useState`, no Redux, no Zustand, no Pinia. The URL is the state. The HTML is the API.

## The Stack

| Layer         | Technology         | Why                                                           |
| ------------- | ------------------ | ------------------------------------------------------------- |
| Server        | Go + Echo          | Fast, typed, compiles to a single binary                      |
| Templates     | templ              | Type-safe HTML generation, composable components              |
| Hypermedia    | HTMX               | Extends HTML with AJAX, keeps the browser as the client       |
| Styling       | Tailwind + DaisyUI | Utility-first CSS, consistent design tokens                   |
| Offline       | Capacitor + SQLite | Native container for iOS, offline-first data                  |
| Interactivity | \_hyperscript      | Declarative client-side behavior (dismiss, back, transitions) |

## Why Not an SPA?

SPAs duplicate the server on the client. You end up maintaining two applications: one that manages data and one that manages UI state, routing, authentication, caching, and error handling — all over again, in JavaScript.

The web already has a UI runtime. It's called the browser. It handles navigation, caching, forms, links, history, accessibility, and progressive enhancement. An SPA throws all of that away and rebuilds it poorly.

This project proves you can build a real, production, offline-capable mobile application with server-rendered HTML, HTMX, and a thin native wrapper — no React, no Vue, no Angular, no virtual DOM, no build pipeline for the frontend, no node_modules black hole.

The complexity budget goes toward the problem domain, not the framework.

## Explicit SQL, Composable Helpers

ORMs hide SQL behind method chains and magic. When something goes wrong — a slow query, a missing join, an unexpected NULL — you're debugging the ORM's generated SQL, not your own. This project takes the opposite approach: **write the SQL, but don't write it by hand every time.**

The repository layer provides composable helpers that keep SQL visible:

```go
sb := NewSelect(TasksTable.Name, cols).
	Where(w).
	OrderByMap(sortBy+":"+sortDir, colMap, "SortOrder ASC").
	Paginate(perPage, offset).
	WithDialect(s.dialect)

query, args := sb.Build()
```

`SelectBuilder` and `WhereBuilder` compose query fragments. `SetClause`, `InsertInto`, and `Placeholders` generate SQL strings you can read. The generated SQL is predictable because it's just string concatenation with guard rails — not a query planner.

This means:

- You can **read the SQL** that runs against your database
- You can **copy it into a query tool** and run it directly
- You can **add database-specific hints** without fighting the abstraction
- You can **switch databases** by swapping the dialect, not rewriting queries

## Schema as Code

Table definitions live in Go, not migration files:

```go
var TasksTable = schema.NewTable("Tasks").
	Columns(
		schema.AutoIncrCol("ID"),
		schema.Col("Title", schema.TypeString(255)).NotNull(),
		schema.Col("Description", schema.TypeText()),
	).
	WithStatus("draft").
	WithVersion().
	WithArchive().
	WithSoftDelete().
	WithTimestamps()
```

Traits like `WithTimestamps()`, `WithSoftDelete()`, and `WithVersion()` add columns and behavior in one call. The table definition is the source of truth for DDL generation, seed data, column lists, and schema validation. One declaration drives `CREATE TABLE`, `INSERT`, `SELECT` column lists, and runtime validation — no drift between migration files and application code.

This mirrors the control composability on the frontend: `WithSoftDelete()` is to a table what `FormActions()` is to a form. Small, composable primitives that encode domain patterns.

## Domain Patterns as Primitives

Soft delete, optimistic locking, archival, entity replacement — these aren't framework features. They're small functions that set timestamps and check row counts:

```go
dbrepo.SetCreateTimestamps(&t.CreatedAt, &t.UpdatedAt)
dbrepo.InitVersion(&t.Version)
dbrepo.SetSoftDelete(&deletedAt)
dbrepo.IncrementVersion(&t.Version)
```

No base class. No embedded struct with 40 fields. No `Model` interface with 12 methods. Just functions that operate on the fields your struct actually has. If you need soft delete, call `SetSoftDelete`. If you don't, don't. The repository doesn't care either way.

The `WhereBuilder` encodes the read side of these patterns:

```go
w := NewWhere().
	NotDeleted().
	NotArchived().
	HasStatus("active").
	Search(search, "Title", "Description")
```

Each filter is a one-liner that adds a WHERE clause. They compose because they're just string builders with named parameters — not a type system trying to model SQL.

## The Document Is the Resource

In REST, what the client sees is a **representation** of a resource. Not a cached copy. Not a local snapshot that needs manual refreshing. The representation. If the resource changes, every client viewing it should see the new state.

SPAs get this wrong by default. The client fetches data, stores it locally, and renders from the local copy. Now you have two sources of truth — the server and the client's stale cache. You need invalidation strategies, optimistic updates, conflict resolution, and refetch logic. You've built a distributed system inside the browser and you're debugging consistency problems that shouldn't exist.

In a hypermedia architecture, the server owns the state and the server owns the representation. When a user's profile is mutated — by that user, by an admin, by a background job — the server knows. It can push the new representation to every client currently viewing that resource via SSE. The client doesn't poll. The client doesn't diff. HTMX listens for the event and swaps in the new HTML.

```
User A edits profile  →  Server updates resource  →  SSE to all viewers  →  HTMX swaps new HTML
```

The server decides **what changed**, **who needs to know**, and **what HTML to send**. The client's only job is to display what it receives. There is one source of truth. There is one rendering path. The document on screen is always the current representation of the resource.

This extends beyond live updates. HTMX response headers let the server direct client behavior after any mutation — redirect, refresh a section, push a new URL, trigger a client-side event. The server is in control because the server is the authority. The client and server aren't two systems negotiating over JSON — they're one system where the server speaks and the browser renders.

## Errors Are Hypermedia

Most applications treat errors as dead ends — a status code, a JSON blob, maybe a generic "something went wrong" page. In a hypermedia architecture, **errors are navigable states**. An error response is just another representation, and it should carry the same hypermedia controls as any other response.

The `ErrorContext` struct encodes this:

```go
type ErrorContext struct {
	Err        error
	Message    string    // what the user sees
	Route      string    // where it happened
	RequestID  string    // how to find it in the logs
	Controls   []Control // what the user can do about it
	OOBTarget  string    // where to render it (out-of-band)
	StatusCode int
	Closable   bool // can the user dismiss it
}
```

When a handler fails, it doesn't return a bare error — it returns an `ErrorContext` with recovery actions:

```go
return handler.HandleHypermediaError(c, 404, "Task not found", err,
	hypermedia.BackButton("Go back"),
	hypermedia.GoHomeButton("Home", "/", "#main"),
)
```

The error middleware detects this and renders it as HTML with embedded controls. The user sees a "Task not found" message with a "Go back" button and a "Home" link — not a stack trace, not a raw 404, not a blank page. The error tells them what happened and gives them a way out.

For inline operations (editing a row, submitting a form), errors render as out-of-band swaps — the primary content stays put and the error panel appears in its designated target. No page navigation, no lost form state.

## Structured Observability

Every HTTP request gets a unique request ID generated by middleware, propagated through context, and returned in the `X-Request-ID` header. When an error reaches the user, the request ID is surfaced in the error component. When the same error hits the logs, it carries the same ID. A user reporting "something went wrong" can give you the request ID, and you can trace the exact request through structured JSON logs.

Logging uses `slog` with context extraction. Every log call that passes through `logger.WithContext(ctx)` automatically picks up the request ID:

```go
logger.WithContext(c.Request().Context()).Error("Request error",
	"error", err, "status_code", statusCode, "message", message)
```

```json
{
  "level": "ERROR",
  "request_id": "a1b2c3...",
  "error": "sql: no rows",
  "status_code": 404,
  "message": "Task not found"
}
```

User-initiated actions (HTTP requests) are logged with `request_id` — the request lifecycle from arrival to response, including method, path, status, and latency. Background operations (workers, async jobs, scheduled tasks) use a separate `context_id` and `context_description` to distinguish them from user traffic. Both flow through the same structured logger, but they're tagged differently so you can filter by origin.

This means:

- **User reports an error** → look up the `request_id` in the logs, see exactly what happened
- **Background job fails** → the `context_id` and `context_description` tell you which job, what it was doing, and when
- **Correlate across layers** → the same context flows from middleware through handlers into repository calls, so a single ID traces the full operation
