# Design Philosophy

<!--toc:start-->

- [Design Philosophy](#design-philosophy)
  - [Go](#go)
  - [Hypermedia-Driven Architecture](#hypermedia-driven-architecture)
  - [Uniform Interface](#uniform-interface)
  - [Self-Descriptive Methods](#self-descriptive-methods)
  - [Resource Identification](#resource-identification)
  - [Parent Routes Are Documents](#parent-routes-are-documents)
  - [Chesterton's Fence](#chestertons-fence)
  - [Server-Side State, Client-Side Rendering](#server-side-state-client-side-rendering)
  - [Content Negotiation](#content-negotiation)
    - [HAL: Content Negotiation for APIs](#hal-content-negotiation-for-apis)
  - [Mutations Redirect](#mutations-redirect)
  - [Postel's Law](#postels-law-be-conservative-in-what-you-send-liberal-in-what-you-accept)
  - [The Stack](#the-stack)
  - [Why Not an SPA?](#why-not-an-spa)
  - [Explicit SQL, Composable Helpers](#explicit-sql-composable-helpers)
  - [Schema as Code](#schema-as-code)
  - [Domain Patterns as Primitives](#domain-patterns-as-primitives)
  - [The Document Is the Resource](#the-document-is-the-resource)
  - [Cacheability](#cacheability)
  - [Locality of Behavior](#locality-of-behavior)
    - [The reach-up model](#the-reach-up-model)
    - [When client-side state is necessary](#when-client-side-state-is-necessary)
    - [Other LoB-aligned tools](#other-lob-aligned-tools)
  - [Errors Are Hypermedia](#errors-are-hypermedia)
    - [Global banner vs. inline errors](#global-banner-vs-inline-errors)
    - [Make reporting effortless or it won't happen](#make-reporting-effortless-or-it-wont-happen)
  - [Structured Observability](#structured-observability)
    - [Promote-on-error](#promote-on-error)
    - [Request and background context](#request-and-background-context)
  - [Web Standards Over Libraries](#web-standards-over-libraries)
    - [Link Relations as the Navigation Model](#link-relations-as-the-navigation-model)
    - [HTTP Headers as the API Surface](#http-headers-as-the-api-surface)
    - [Native HTML Over JavaScript](#native-html-over-javascript)
    - [CSS Over JavaScript](#css-over-javascript)
    - [Browser APIs Over Libraries](#browser-apis-over-libraries)
    - [Speculative Loading](#speculative-loading)
    - [The Principle](#the-principle)
  <!--toc:end-->

## Go

This project is written in Go and follows Go's values. These principles, inspired by the [Go Proverbs](https://go-proverbs.github.io/), inform every design decision.

**Clear is better than clever.** Code is read far more than it is written. A straightforward ten-line function that anyone can follow beats a three-line abstraction that requires context to decode. When you're tempted to be clever, write the obvious thing instead.

**A little copying is better than a little dependency.** If you need ten lines of utility code, copy them. Don't import a package with a transitive dependency tree, a maintenance burden, and a security surface — just to avoid writing a function. The Go standard library is the dependency. Everything else earns its place.

**The bigger the interface, the weaker the abstraction.** Good interfaces are small. `io.Reader` has one method and it powers the entire I/O ecosystem. In this project, interfaces are defined by the consumer, not the producer. A handler that needs to report issues accepts a `ReportHandler` with one method — it doesn't know or care whether the implementation sends an email, posts to Teams, or writes to a log. If your interface has more than three methods, you're describing an implementation, not a behavior.

**Make the zero value useful.** Types should work without explicit initialization. If a struct field is optional, its zero value should mean "use the default," not "panic at runtime." This is why `ErrorContext{}` renders a valid (if empty) error panel, and why `Control{}` renders as a secondary button. No constructors required for the common case.

**Errors are values.** Errors are not exceptions. They don't unwind the stack. They don't break control flow. They're returned, checked, wrapped with context, and handled at the appropriate level. `fmt.Errorf("open config: %w", err)` tells you what was happening when it failed. `panic` is for programmer bugs, not runtime conditions.

**Don't just check errors, handle them gracefully.** Checking `if err != nil { return err }` is correct but insufficient. Good error handling means adding context (`fmt.Errorf`), choosing the right recovery action (retry, degrade, report), and surfacing useful information to whoever or whatever is next in the chain — whether that's a user seeing an error banner or an operator reading structured logs.

**Design the architecture, name the components, document the details.** The architecture diagram and package names should tell you how the system works. Comments should tell you why a particular decision was made. Code that needs a comment to explain what it does should be rewritten until it doesn't.

**Don't panic.** `panic` means the program cannot continue. A missing database row is not that. A malformed user input is not that. A timeout from an upstream service is not that. Return an error, let the caller decide, and keep the server running.

## Hypermedia-Driven Architecture

This application is built on the principles of [REST as Roy Fielding defined it](https://roy.gbiv.com/pubs/dissertation/fielding_dissertation.pdf) — not the bastardized "REST API" that the industry settled on, but actual REST: **Representational State Transfer** through hypermedia. As Fielding himself clarified, [REST APIs must be hypertext-driven](https://roy.gbiv.com/untangled/2008/rest-apis-must-be-hypertext-driven).

The server returns HTML with embedded hypermedia controls that tell the client what actions are available and how to invoke them. The client doesn't need to know anything about the API surface ahead of time. Every transition is discoverable from the current representation. This is HATEOAS (Hypermedia As The Engine Of Application State).

**You do not need a single-page application.** The browser is already a hypermedia client. HTMX extends it to handle the cases where a full page reload is wasteful. The result is a simpler, more maintainable architecture that leverages what the web was designed to do.

This also preserves the [Principle of Least Astonishment](https://en.wikipedia.org/wiki/Principle_of_least_astonishment). Hypermedia respects the web's native interaction model: links navigate, forms submit, the back button goes back, the URL reflects where you are, refresh shows the current state. These aren't features — they're expectations that browsers have trained into users over three decades. SPAs break these expectations by default and then spend engineering effort rebuilding them (client-side routing, history management, scroll restoration, form state preservation). A hypermedia architecture gets them for free because it never took them away.

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

Writing `hx-get`, `hx-post`, `hx-target` directly on elements is perfectly fine — that's HTMX doing its job. But when the same arrangement of attributes shows up across multiple pages, that's a pattern asking to become a primitive. Breadcrumbs, modal menus, create-form actions, edit-form actions — these all started as raw `hx-*` attributes on elements, and each one eventually earned a factory function because the repetition made it worth encoding.

This isn't a rule. There's no enforcement layer. It's the natural progression of a tool-less machinist who starts making tools for himself: you do the work by hand until the hand-work repeats enough that building a jig saves time. `FormActions("create")` is a jig. It exists because we got tired of wiring the same three buttons with the same swap targets on every create form. If a new page needs something novel, write the `hx-*` attributes directly. If that novel thing shows up three more times, consider whether it's earned a place in the control vocabulary.

Because hypermedia drives the application, navigational chrome like breadcrumbs and action bars should either flow from the parent representation or be derivable from the navigation structure itself. The server already knows where the user is — the route, the resource, the hierarchy. Breadcrumbs are just that hierarchy rendered as links. Action bars are just the available transitions for the current resource state. These aren't independent pieces of UI that each handler assembles from scratch; they're projections of document state. A task's edit page knows it sits under `/tasks/{id}`, which sits under `/tasks` — the breadcrumb trail writes itself. The action bar knows whether the resource is in draft or published state and offers the transitions that make sense. Local modifications — an extra button for a specific workflow, a contextual link that only applies here — are fine, but the baseline should come from the resource's position in the navigation graph, not from per-handler boilerplate.

[Hyrum's Law](https://www.hyrumslaw.com/) applies here with unusual force: *with a sufficient number of users of an API, all observable behaviors of your system will be depended on by somebody.* In a hypermedia architecture, the HTML *is* the API. Every element ID, every CSS class, every `hx-target` selector, every swap mode is an observable behavior. Change `#task-detail` to `#task-content` and every OOB swap targeting it breaks. Rename a DaisyUI class and \_hyperscript selectors that reference it stop working. The explicit contract — Controls, factory functions, semantic classes — exists precisely to manage this. Changes flow through a single point rather than requiring a grep across templates. But outside the explicit contract, the implicit one is larger than you think.

## Self-Descriptive Methods

HTTP methods carry semantics. They're not just verbs to put on a request — they're a contract between client and server that constrains what the request can do before the handler runs a single line of code.

**GET is safe.** It has no side effects. It doesn't create, update, or delete anything. It's idempotent — calling it once is the same as calling it a hundred times. Because it's safe, browsers prefetch it, crawlers follow it, caches store it, and CSRF middleware skips it. A GET handler that mutates state isn't just a bad practice — it violates the self-descriptive constraint of REST. The method *is* part of the message.

**POST creates.** It's the only non-idempotent method. Submitting it twice may create two resources. This is why mutations redirect after completion — a refresh on a POST result would double-submit.

**PUT replaces.** It's idempotent — sending the same PUT twice produces the same state. The entire resource is overwritten with the request body.

**PATCH modifies partially.** It updates specific fields without replacing the whole resource.

**DELETE removes.** It's idempotent — deleting an already-deleted resource is a no-op, not an error.

The CSRF middleware encodes this distinction directly: safe methods (GET, HEAD, OPTIONS, TRACE) bypass token validation because they cannot change state. Unsafe methods require a valid token. This isn't an implementation detail — it's the HTTP specification applied as architecture.

## Resource Identification

URLs identify resources, not actions. `/tasks/42` is a resource. The HTTP method is the verb — GET retrieves it, PUT replaces it, DELETE removes it. The URL is the noun; the method is the sentence.

This means no verb-based URLs. `/api/getTask`, `/tasks/delete/42`, `/tasks/42/archive` — these encode the action in the address. The address should be stable regardless of what you're doing to the resource. `/tasks/42` is the same resource whether you're reading it, updating it, or deleting it.

Query parameters represent views of a collection. `/tasks?sort=name&dir=asc&page=2` isn't a different resource — it's the same collection, sorted and paginated. The base URL `/tasks` identifies the collection; the parameters select a projection of it.

Sub-resources nest hierarchically. `/people/7/edit` is the edit representation of person 7. `/tasks/42/notes` would be the notes belonging to task 42. The URL path mirrors the resource graph — you can read the hierarchy by reading the path segments.

When sub-resources have fixed, known identities — use semantic slugs, not opaque IDs. `/punchlist/2/checklist` is better than `/punchlist/2/step/3`. Numeric IDs are for database-assigned identifiers; named resources get named paths. If you can enumerate the set at development time (sections, phases, categories), those are slugs, not indices.

URLs should be bookmarkable, shareable, and stable. A user should be able to copy the address bar, send it to a colleague, and that colleague should see the same resource. This is a consequence of stateless requests and resource identification working together — the URL contains enough information to reconstruct the view.

## Parent Routes Are Documents

**If a route exists, it must serve a representation.** A parent route that redirects to a child isn't a resource — it's a broken link wearing a URL as a disguise. `/dashboard` should render a document, not bounce you to `/dashboard/overview`.

Redirecting parent routes breaks the hypermedia model in three ways. Breadcrumbs walk the `rel="up"` chain — if clicking "Dashboard" in the breadcrumb trail redirects you somewhere else, the chain points to a moving target instead of a stable document. Hub centers in the link registry declare `rel="related"` links to their children — a Hub that redirects isn't a resource, it's a hole in the navigation graph. And the URL lies — `/dashboard` appears in the address bar during the redirect but has no representation of its own.

A parent route should be one of three things:

- **An aggregate page** — a dashboard showing key metrics pulled from each child section. The parent earns its URL by summarizing what lives beneath it.
- **A discovery page** — lists and links to child sections, like `/demo` does. The parent is a table of contents for its sub-resources.
- **An informational page** — context about the section before the user navigates deeper. Useful for workflows with prerequisites or explanatory content.

The link relations system makes this natural. A Hub center with `rel="related"` links to all its children is a discovery page by definition — you declared the relationships, now render them. The breadcrumb system walks `rel="up"` from child to parent. If the parent redirects, `up` points to a redirect, not a document, and the breadcrumb chain breaks.

```go
// Wrong: the parent route is a redirect, not a document
r.GET("/dashboard", func(c echo.Context) error {
	return c.Redirect(http.StatusFound, "/dashboard/overview")
})

// Right: the parent route renders its own representation
r.GET("/dashboard", func(c echo.Context) error {
	// Aggregate data from child sections, render a real page
	return response.Builder.Component(c, dashboard.Index(metrics))
})
```

Parent routes are the backbone of the navigation graph. They must be stable, addressable documents — not trampolines.

## Chesterton's Fence

[Chesterton's Fence](https://en.wikipedia.org/wiki/G._K._Chesterton#Chesterton's_fence) is a principle of reform: *don't remove something until you understand why it was put there.* In the original formulation, if you encounter a fence across a road and can't see the reason for it, that's not an argument for removing it — it's an argument for finding out why it exists before you touch it.

In a hypermedia system, patterns exist for reasons that aren't always visible from the element alone. An `hx-target` that looks wrong might account for an OOB swap elsewhere on the page. A swap mode that seems redundant might prevent a layout shift during a transition. A control factory that seems over-engineered might encode a consistency guarantee across twenty pages. Before removing or simplifying a pattern, trace why it exists. The cost of understanding is low. The cost of breaking a silent invariant is high.

This applies at every level: the `From` query parameter on breadcrumbs, the `ErrorTarget` field on Controls, the `OOBTarget` on error contexts — each of these exists because a specific interaction broke without it. If you can't explain why it's there, you haven't yet earned the right to take it away.

## Server-Side State, Client-Side Rendering

State lives on the server. The client is a thin rendering layer. When state changes, the server sends new HTML. The browser's job is to display it and let the user interact with the controls embedded in it.

No client-side routing. No client-side state management. No `useState`, no Redux, no Zustand, no Pinia. The URL is the state. The HTML is the API.

Each request is stateless — it contains everything the server needs to process it. The server doesn't remember what happened last request. Session identity comes from a cookie, but session state is loaded fresh from the store on every request. Request context — the request ID, CSRF token, session settings — is established by middleware each time, not carried forward from a previous interaction. A handler cannot assume that another handler ran first, or that the client has seen a particular response. Every request stands alone.

## Content Negotiation

The same resource serves different representations based on client capabilities. This is [content negotiation](https://developer.mozilla.org/en-US/docs/Web/HTTP/Guides/Content_negotiation), and in a hypermedia architecture it's how the server maintains progressive enhancement without maintaining two codebases.

When a request carries the `HX-Request: true` header, the server knows the client is HTMX-capable. It returns a partial — just the fragment that changed, targeted at a specific element. When the same URL is requested via standard browser navigation (no `HX-Request` header), the server returns a full page: layout, navigation, content, the works. Same resource, same URL, different representations. The handler decides based on what the client can handle.

This is also how error responses work. An HTMX request that fails gets an out-of-band swap to the error banner — the page stays put, the error appears. A non-HTMX request that fails gets a full error page with navigation controls. Both surfaces carry the same information (error message, request ID, recovery actions), but the representation fits the client's rendering model.

The key constraint: every resource must be reachable without JavaScript. Links are `<a>` tags. Forms are `<form>` tags. HTMX enhances them with `hx-get`, `hx-post`, and targeted swaps, but if HTMX fails to load, the page still works — links navigate, forms submit, the server returns full pages. Progressive enhancement isn't a bonus feature. It's the baseline.

### HAL: Content Negotiation for APIs

Content negotiation extends beyond the HTML/fragment split. The same resource graph can serve `application/hal+json` — [HAL](https://datatracker.ietf.org/doc/html/draft-kelly-json-hal) adds `_links` and `_embedded` to JSON, giving API clients navigable relationships without out-of-band URL construction. A browser sees HTML with HTMX controls. A `curl` user sees HAL+JSON with link relations. Same resource, same relationships, different media type.

HAL gives JSON what `<a>` tags give HTML. It does not give JSON what `<form>` tags give HTML — and that gap is where the interesting architectural questions live. The `/hypermedia/hal` demo renders both representations side by side so the gap is visible. See [docs/HAL.md](docs/HAL.md) for the full Socratic inquiry into what HAL provides, what it doesn't, and what Fielding would say about both.

## Mutations Redirect

After a mutation — a POST, PUT, or DELETE that changes server state — the response should redirect the client to the new representation, not return it directly. This is the [POST/Redirect/GET](https://en.wikipedia.org/wiki/Post/Redirect/Get) pattern, and in HTMX it's implemented via the `HX-Redirect` response header with a `303 See Other` status.

The redirect does three things. First, it prevents double-submission: if the user refreshes the page, the browser re-sends a GET to the redirect target, not the original POST. Second, it keeps the URL bar honest — after creating a task, the browser shows `/tasks/42`, not `/tasks/create`. Third, it respects the browser's history stack — the back button takes you to a resource, not to a mutation endpoint.

This matters more in HTMX than it does in traditional forms because returning a partial from a POST *feels* natural — the swap just works. But the swap doesn't fix the URL, doesn't fix the history stack, and doesn't prevent double-submit. The redirect does. Reach for `response.Builder.Redirect()` after mutations, not `response.Builder.Component()`.

## Postel's Law: Be Conservative in What You Send, Liberal in What You Accept

[Postel's Law](https://en.wikipedia.org/wiki/Robustness_principle) (the Robustness Principle) was written for TCP but it governs every boundary in this application: **be conservative in what you send, be liberal in what you accept.**

### Forms: accept anything, validate on the server

HTML form inputs should be permissive. Don't prevent the user from typing. Don't mask, truncate, or reject input on the client side. Let the browser send whatever the user entered. The server validates it, and if it's wrong, the server sends back a representation of the error — inline, next to the field, with a clear message and the original input preserved so the user can fix it.

```html
<!-- Good: the browser accepts anything, the server decides -->
<input type="text" name="email" value="" />

<!-- Bad: client-side regex that rejects valid emails, confuses users,
     and duplicates validation that the server must do anyway -->
<input type="email" pattern="[a-z0-9._%+-]+@[a-z0-9.-]+\.[a-z]{2,}$"
       required oninvalid="this.setCustomValidity('Enter a valid email')" />
```

Use HTML's built-in `type`, `required`, `min`, `max`, and `maxlength` attributes for baseline guardrails — these are the browser doing its job without JavaScript. But don't pile on `pattern` attributes, custom `oninvalid` handlers, or JavaScript validation libraries that duplicate what the server already does. The server is the authority. The client is a convenience.

When validation fails, the server returns the form with errors rendered inline:

```html
<div class="form-control">
  <input type="text" name="email" value="not-an-email" class="input input-error" />
  <span class="label-text-alt text-error">Not a valid email address</span>
</div>
```

HTMX makes this seamless — `hx-post` submits the form, the server validates, and the response swaps in either the success state or the form with errors. No JavaScript validation library. No `useState` for error messages. No client-side schema that drifts from the server's rules. One source of truth for validation, one rendering path for errors.

### Responses: send exactly what the client needs

The server should be conservative — precise, minimal, correct — in what it sends. Every response is a complete, valid representation. No partial states that require the client to assemble meaning. No ambiguous status codes. No silent failures.

- Return the correct HTTP status code. A validation failure is 422, not 200 with an error field buried in JSON.
- Include hypermedia controls that tell the client what to do next. A successful create returns the new resource with edit/delete controls. A failed validation returns the form with errors and a retry control.
- Don't send more data than the representation requires. A list page doesn't need every field of every record — it needs the fields that appear in the list, plus controls for pagination and detail navigation.

### At every boundary

This principle applies beyond forms:

- **URL parameters**: accept unexpected query params gracefully (ignore them), but only emit well-documented params in links and controls.
- **SSE events**: accept reconnections and stale clients without error, but send only the OOB fragments that target elements actually on the current page.
- **Configuration**: accept missing env vars with sensible defaults, but document exactly which vars the app reads and what each one does.
- **API consumers**: if you expose `/health`, return a stable, documented schema. Don't break consumers who only read `status` by renaming it to `state`.

The robustness principle isn't about being sloppy. It's about building systems that compose well across trust boundaries. The server trusts nothing from the client but handles everything gracefully. The client trusts the server's response completely because the server is conservative about what it sends.

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

[Tesler's Law](https://en.wikipedia.org/wiki/Law_of_conservation_of_complexity) (the Law of Conservation of Complexity) says that every application has an inherent amount of complexity that cannot be removed or hidden — it can only be moved. SPAs don't reduce complexity. They move it to the client. Now the client manages routing, state, caching, authentication, error handling, and data synchronization — all concerns the server already handles. The total complexity is the same or greater. This project moves complexity to the server, where Go's type system, structured logging, and single-process model make it tractable. The complexity budget is fixed; the question is where it lives.

### But honestly

SPAs aren't wrong. They're wrong *here*. If the UI is the product — a design tool, a collaborative editor, a code IDE — the client needs to own state, and an SPA is the right architecture. CRDTs, offline-first sync, canvas rendering, shared cursors: these are inherently client-side concerns that no server round-trip can solve.

This project builds data-centric workflows — CRUD, dashboards, admin panels, form-heavy pages. That's hypermedia's home court. If it were building Figma, it would be React.

Where each falls apart:

```
  Hypermedia sucks at                    SPAs suck at
  ─────────────────────                  ────────────────────
  Rich interactive UIs                   Content sites — rebuilt the
    (spreadsheets, design tools)           browser for nothing

  Offline-first with                     CRUD apps — two state stores
    conflict resolution                    that drift apart

  Real-time collaborative               Bundle size, hydration,
    editing (CRDTs, OT)                    time-to-interactive

  Canvas / WebGL —                       SEO and accessibility as
    the rendering IS the client            afterthoughts bolted on

  Latency-sensitive interactions         The dependency treadmill —
    that need instant local feedback       node_modules, framework churn
```

Each side's weaknesses are the other side's strengths. Pick the architecture that matches your problem, not your identity.

## Explicit SQL, Composable Helpers

ORMs hide SQL behind method chains and magic. When something goes wrong — a slow query, a missing join, an unexpected NULL — you're debugging the ORM's generated SQL, not your own. This project takes the opposite approach: **write the SQL, but don't write it by hand every time.**

[Fraggle's dbrepo package](https://github.com/catgoose/fraggle) provides composable helpers that keep SQL visible:

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

Joel Spolsky's [Law of Leaky Abstractions](https://www.joelonsoftware.com/2002/11/11/the-law-of-leaky-abstractions/) explains why: *all non-trivial abstractions, to some degree, are leaky.* An ORM abstracts SQL behind method chains and objects. When it works, you don't think about SQL. When it leaks — a slow query, an unexpected N+1, a missing join, a dialect-specific feature you need today — you're debugging the ORM's generated SQL, not your own. You need to understand two systems: the abstraction and the thing it abstracts. The leaky abstraction made the easy case easier and the hard case harder.

This reasoning applies throughout the stack. HTMX abstracts HTTP, but `hx-swap`, `hx-target`, and response headers are HTTP concepts that leak through — and that's fine, because knowing HTTP is the point. DaisyUI abstracts Tailwind, but layout and spacing still require raw utilities. The project's philosophy isn't "no abstractions" — it's "choose abstractions whose leaks are useful rather than surprising." `SelectBuilder.Build()` leaks SQL intentionally — you can read it, copy it, run it. An ORM leaks SQL accidentally — you discover it in a slow query log at 2 AM.

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

SSE is essential to this model because it eliminates drift. Without it, the client's representation ages with every passing second — stale data masquerading as current state. With SSE, the server pushes new representations the moment state changes. There is no polling interval where drift can accumulate, no cache TTL to guess at, no "refresh to see changes" banner. The document on screen is the resource, continuously. This is what server-owned state actually means: not just that the server stores the data, but that the server is responsible for keeping every client's view current.

## Cacheability

[Cacheability](https://roy.gbiv.com/pubs/dissertation/fielding_dissertation.pdf) is one of Fielding's six REST constraints, and it's a correctness constraint, not just a performance optimization. Responses must declare themselves cacheable or not, because a stale cache violates the "document is the resource" principle — the user sees something that is no longer true.

Static assets — JavaScript, CSS, images, fonts — are fingerprinted by the build process. Their URLs change when their content changes. This makes them safe to cache aggressively: `Cache-Control: public, max-age=31536000, immutable`. The browser stores them forever and never asks again. When the content changes, it gets a new URL, and the old cached version is simply never requested.

Dynamic HTML representations are the opposite. They reflect current resource state, and current means *right now*. Caching a list page means a user might not see the item they just created. Caching a detail page means a user might see data that another user just updated. Dynamic responses carry no cache headers — every request goes to the server, and the server returns the truth. SSE handles the case where the truth changes while the client is watching.

SSE streams themselves explicitly disable caching (`Cache-Control: no-cache`) because the entire point of an event stream is that it's live. Caching an event stream is caching a contradiction.

## Locality of Behavior

The behaviour of a unit of code should be as obvious as possible by looking only at that unit of code. This is the [Locality of Behaviour](https://htmx.org/essays/locality-of-behaviour/) (LoB) principle, and it is the gravitational center of how this application handles interactivity.

Separation of Concerns told us to put HTML in one file, CSS in another, and JavaScript in a third. The result is spooky action at a distance — you look at a button and have no idea what it does without grepping three directories. LoB says: put the behavior on the element. When you read the markup, you know what it does. When you change the markup, you know what you're changing.

```html
<!-- LoB: the behavior is right here -->
<button hx-get="/tasks/1" hx-target="#task-detail" hx-swap="innerHTML">
  View Task
</button>

<!-- not LoB: the behavior is somewhere in a .js file, good luck finding it -->
<button id="view-task-btn" class="task-action">View Task</button>
```

### The reach-up model

Every tool in this stack exists because HTML alone couldn't express something. You start at HTML and only **reach up** when the current layer can't do what you need. Each step trades simplicity for capability.

This is [Gall's Law](https://en.wikipedia.org/wiki/John_Gall_(author)#Gall's_law) in practice: *a complex system that works is invariably found to have evolved from a simple system that worked. A complex system designed from scratch never works and cannot be patched up to make it work.* You didn't design a five-layer interactivity stack from scratch — you discovered it by writing HTML, hitting its limits, reaching for HTMX, hitting its limits, reaching for \_hyperscript, and so on. Each layer earned its place by solving a problem the layer below couldn't. The same applies to Controls: raw `hx-*` attributes came first. Factory functions came later, only when the repetition made them worth encoding.

[YAGNI](https://en.wikipedia.org/wiki/You_aren%27t_gonna_need_it) — You Aren't Gonna Need It — is the temporal cousin of Gall's Law. Gall's Law says complex systems must evolve from simple ones. YAGNI says don't build the complex one until you have evidence the simple one isn't enough. A feature without a user, an abstraction without a second use case, a configuration option nobody has asked to configure — these aren't investments, they're liabilities with carrying costs. Build what the problem requires today. When the problem changes, the code can change with it.

Two tracks rise from their foundations — Behavior (how things interact) and Presentation (how things look):

```
   Behavior                              Presentation

   ▲ reach up                            ▲ reach up
   │                                     │
   │  ┌──────────┐                       │  ┌──────────────┐
   │  │ .js files │  locality broken     │  │   raw CSS    │  escape hatch
   │  ├────────────────┐                 │  ├────────────────────┐
   │  │ inline <script> │  locality bent │  │     Tailwind       │  layout + spacing
   │  ├──────────────────────┐           │  ├──────────────────────────┐
   │  │       Alpine.js      │  state    │  │         DaisyUI          │  semantic
   │  ├────────────────────────────┐     │  ├────────────────────────────────┐
   │  │        _hyperscript        │     │  │             CSS                │  the cascade
   │  ├──────────────────────────────┐   │  └────────────────────────────────┘
   │  │            HTMX              │   │    start here ▲
   │  ├────────────────────────────────┐
   │  │             HTTP               │
   │  ├──────────────────────────────────┐
   │  │              HTML                │
   │  └──────────────────────────────────┘
   │    start here ▲
```

Map two dimensions — **where it runs** and **what it manages** — and six domains emerge:

```
                  State               Behavior              Presentation
           ┌──────────────────┬──────────────────────┬──────────────────────┐
           │                  │                      │                      │
  Server   │  Go + SQL        │  HTTP + HTMX         │  templ + DaisyUI     │
           │  source of truth │  hypermedia controls │  semantic components │
           │  resource state  │  resource transitions│  theme-aware markup  │
           │                  │                      │                      │
           ├──────────────────┼──────────────────────┼──────────────────────┤
           │                  │                      │                      │
  Client   │  Alpine.js       │  _hyperscript        │  Tailwind + CSS      │
           │  view state      │  DOM interactions    │  layout, spacing     │
           │  ephemeral       │  transitions, toggles│  visual adjustments  │
           │                  │                      │                      │
           └──────────────────┴──────────────────────┴──────────────────────┘
```

The left column is authority — server state is the single source of truth, client state is ephemeral view data. The middle column is interaction — the server drives transitions through hypermedia controls, the client handles what doesn't need the server. The right column is appearance — server-authored semantic markup adapts to themes, client-side utilities handle spatial layout.

Structure (HTML, templ, the DOM) is not a column — it's the medium all three concerns are expressed through. You don't "reach up" to structure; it's already there the moment you write `<div>`.

### The interactivity spectrum

This application has a formal structure — server-rendered HTML, typed hypermedia controls, uniform interfaces — but within that structure there are pockets of interactivity. A dismiss button. A copy-to-clipboard action. A tooltip that appears and fades. A modal that opens and closes. A theme switcher that updates the DOM and persists to the server.

These don't need a framework. They don't need a build step. They need a few lines of behavior attached to the element where the behavior happens.

You have three tools for this, in order of preference:

**1. HTMX attributes** — for server round-trips. `hx-get`, `hx-post`, `hx-target`, `hx-swap`. The server owns the state, HTMX asks for new representations. This is the primary tool.

**2. \_hyperscript** — for client-side behavior that doesn't need the server. DOM manipulation, transitions, clipboard, toggling visibility. This is the [preferred tool for LoB-adherent interactivity](#why-hyperscript-over-javascript). Always the first choice for client-side behavior.

**3. Inline `<script>` tags** — when \_hyperscript can't express what you need. Keep the script next to the element it relates to. This preserves locality — the behavior is still in the same template, visible in the same context. Always use JSDoc to document functions and parameters.

**4. JavaScript files** — the last resort. Sometimes you need shared utilities, third-party library initialization, or complex logic that doesn't belong inline. This is acceptable, but recognize it for what it is: you've broken locality. The behavior is no longer visible where it's used. Offset this by keeping files small, purpose-specific, and always documented with JSDoc.

The gradient is: **\_hyperscript → inline script → .js file**. Each step trades locality for capability. Take the smallest step you need.

### Why \_hyperscript over JavaScript

You could write JavaScript for every interactive behavior. It works. But it fragments behavior across elements and scripts, and each developer writes it differently. \_hyperscript keeps behavior on the element in a uniform, declarative syntax:

```html
<!-- best: hyperscript on the element, reads like intent -->
<button _="on click toggle .hidden on #panel then wait 2s then add .hidden to #panel">
  Show briefly
</button>

<!-- acceptable: inline script next to the element, still local -->
<button onclick="togglePanel(this)">Show briefly</button>
<script>
/** @param {HTMLButtonElement} btn - The button that triggered the toggle */
function togglePanel(btn) {
  const panel = document.getElementById('panel');
  panel.classList.toggle('hidden');
  setTimeout(() => panel.classList.add('hidden'), 2000);
}
</script>

<!-- avoid: behavior in a separate .js file, locality broken -->
<button id="toggle-btn">Show briefly</button>
<!-- the reader has no idea what this button does -->
```

The \_hyperscript version is self-contained. You read the element, you know the behavior. You delete the element, the behavior is gone. No orphaned event listeners. No dead functions in a utils file that nobody is sure are still used.

The inline script version is a step down — the behavior is still visible in the same template, but it's split across the element and the script tag. It's local enough. When you must go this route, **always use JSDoc**. Document every function, every parameter, every return value. JavaScript without JSDoc is a guessing game for the next reader.

```html
<script>
/**
 * Copy text to the clipboard and show a brief tooltip.
 * @param {HTMLElement} el - The element containing the text to copy
 * @param {string} selector - CSS selector for the tooltip to reveal
 * @returns {Promise<void>}
 */
async function copyAndFlash(el, selector) {
  await navigator.clipboard.writeText(el.textContent);
  const tip = el.querySelector(selector);
  tip.classList.remove('hidden');
  setTimeout(() => tip.classList.add('hidden'), 1500);
}
</script>
```

The .js file version is the last resort. The behavior is invisible at the point of use. You're relying on file names and conventions to connect element to behavior. Sometimes this is necessary — shared utilities, third-party initialization, complex algorithms. But keep these files small, single-purpose, and thoroughly documented with JSDoc.

More importantly, patterns emerge. When you write `on click toggle .hidden on #panel` enough times, you recognize it as a pattern. \_hyperscript lets you extract these into [behaviors](https://hyperscript.org/docs/#behaviors), [events](https://hyperscript.org/features/send/), and [listeners](https://hyperscript.org/features/on/) — all within \_hyperscript itself, not in a separate abstraction layer. The language scales from inline one-liners to reusable named behaviors without switching paradigms.

This matters because complexity is the [apex predator](https://grugbrain.dev/). Every time you cross a boundary — from HTML to JavaScript, from one file to another, from one paradigm to another — you pay a complexity tax. \_hyperscript keeps the tax low by keeping everything in one place, in one language, at one level of abstraction.

### DaisyUI: semantic classes as LoB for styling

The same principle applies to CSS. Tailwind gives you utility classes. DaisyUI gives you semantic component classes that encode design decisions:

```html
<!-- DaisyUI: intent is clear, theme-aware by default -->
<button class="btn btn-primary btn-sm">Save</button>
<dialog class="modal"><div class="modal-box">...</div></dialog>

<!-- raw Tailwind: you're reading implementation details, not intent -->
<button class="inline-flex items-center px-3 py-1.5 text-sm font-medium rounded
  bg-blue-600 text-white hover:bg-blue-700 focus:outline-none">Save</button>
```

DaisyUI's `btn-primary` adapts to the active theme. Raw Tailwind's `bg-blue-600` doesn't. When you read `modal-box`, you know it's a modal. When you read a wall of utility classes, you're reverse-engineering the design.

Use DaisyUI classes for components. Use Tailwind utilities for layout and spacing. The component tells you *what*, the utilities tell you *where*.

DaisyUI also inherits Tailwind's core build philosophy: **only ship the CSS you use.** Tailwind scans your markup and generates only the utility classes that actually appear. DaisyUI extends this — you choose which components to include, and unused component styles never enter the bundle. A project that uses `btn`, `modal`, and `badge` doesn't pay for `carousel`, `timeline`, or `drawer`. This is the opposite of monolithic CSS frameworks that ship everything and dare you to tree-shake what you don't need. The result is a small, predictable stylesheet where every rule traces back to an element in your templates.

This selectivity comes with a contract: **use DaisyUI's semantic color roles, not raw color values.** DaisyUI themes define `primary`, `secondary`, `accent`, `neutral`, `base-100/200/300`, `info`, `success`, `warning`, and `error`. Every DaisyUI component references these roles — `btn-primary` uses the theme's `primary`, `alert-error` uses the theme's `error`. If you reach for `bg-blue-600` or `text-red-500` instead, you've hard-coded a color that won't follow the theme. The theme selector switches all semantic colors at once; raw Tailwind colors don't participate. A button that uses `btn-primary` in the light theme is still correct in the dark theme, in the corporate theme, in any theme. A button that uses `bg-blue-600` is blue forever. Stick to the semantic roles and theming works for free.

### When client-side state is necessary

The interactivity spectrum above covers behavior — things that happen in response to events. But sometimes an element needs *state*: a dropdown that tracks whether it's open, a multi-step form that remembers which step you're on, a filter panel that holds transient selections before the user commits them to the server.

This is where the philosophy bends but doesn't break. The principle isn't "no client-side state." It's "no client-side state *that the server should own*." A modal's open/closed flag is not server state. A character count on a textarea is not server state. An accordion's expanded sections are not server state. These are *view states* — local, ephemeral, and meaningless outside the current DOM.

For these cases, [Alpine.js](https://alpinejs.dev/) is a natural companion. It keeps state on the element, declared inline, visible where it's used:

```html
<!-- Alpine: state and behavior are both on the element -->
<div x-data="{ open: false }">
  <button @click="open = !open">Toggle</button>
  <div x-show="open" x-transition>
    Panel content
  </div>
</div>
```

This is LoB-adherent. You read the element, you see the state, you see the behavior, you see the rendering logic. Delete the element, everything goes with it. No external store, no state management library, no subscription model.

The key distinction: Alpine manages *view state*. HTMX manages *resource state*. They don't compete — they govern different domains. A filter panel might use Alpine to track which checkboxes are selected (view state) and HTMX to submit the selection to the server and swap in filtered results (resource state). Each tool does what it's good at, and neither pretends to be the other.

We aren't enforcing Alpine.js as a requirement. The interactivity gradient still applies: if `_hyperscript` handles the interaction, use `_hyperscript`. If you need reactive client-side state that `_hyperscript` doesn't model well — conditional rendering, computed values, two-way bindings — Alpine is the tool that preserves locality. The point is not to prescribe a specific library but to provide only *discovered abstractions*: tools that emerge because they more perfectly align the UI, the user's mental model, and the data with each other. Alpine earns its place the same way `_hyperscript` does — by keeping behavior where you can see it.

### Other LoB-aligned tools

This project uses HTMX, `_hyperscript`, and DaisyUI. But the LoB principle is bigger than any one stack. Other projects worth knowing about:

- **[Alpine.js](https://alpinejs.dev/)** — Reactive client-side state declared inline via `x-data`, `x-show`, `x-bind`, and `@click`. Complements HTMX rather than replacing it — Alpine handles view state, HTMX handles resource state. Covered in detail [above](#when-client-side-state-is-necessary).

- **[Petite Vue](https://github.com/vuejs/petite-vue)** — A 6KB subset of Vue designed for progressive enhancement. Uses `v-scope` instead of a full Vue app mount. Similar to Alpine in spirit — inline reactive state on DOM elements — but with Vue's template syntax for teams already familiar with it.

- **[Tailwind CSS](https://tailwindcss.com/)** — Already in this project's stack, but worth calling out explicitly as a LoB tool. Utility classes on the element replace stylesheets in separate files. You read the element, you see how it looks. DaisyUI layers semantic meaning on top, but both are LoB for styling.

These tools share a conviction: **the reader should not have to leave the element to understand the element.** They differ in scope, syntax, and trade-offs, but they all reject the idea that behavior, state, and presentation should be scattered across separate files connected by naming conventions and hope.

### System space vs. user space errors

Not all errors are equal. Where an error originates determines how it should be handled.

**System space** errors are infrastructure failures — the database is down, a file can't be read, a service is unreachable. These are *exceptional*. They should not happen during normal operation. The system should log them with full context (request ID, stack trace, timing), alert operators, and present the user with a graceful degradation — an error banner with a Report Issue button, not a stack trace.

```go
db, err := database.Open(ctx, cfg.DBEngine)
if err != nil {
    // System space: this is exceptional. Log it, surface it, let ops handle it.
    log.Fatal("Failed to open database", "error", err)
}
```

**User space** errors are expected outcomes — validation failures, missing resources, permission denials. These are *contextual*. They're part of the normal flow. The user did something that didn't work, and the response should tell them what happened and what they can do about it. A 404 is not an exception — it's a representation of "this resource doesn't exist" with controls to navigate elsewhere.

```go
task, err := repo.FindByID(ctx, id)
if err != nil {
    // User space: this is contextual. Return a representation with recovery actions.
    return handler.HandleHypermediaError(c, 404, "Task not found", err,
        hypermedia.BackButton("Go back"),
        hypermedia.GoHomeButton("Home", "/", "#main"),
    )
}
```

The distinction: system space errors are for operators. User space errors are for users. Both produce structured, observable output. But system space reaches for the alarm while user space reaches for the navigation controls. Both adhere to the REST uniform interface — a result is always returned, never a raw exception, never a blank page, never silence.

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

### Global banner vs. inline errors

Errors render in two places depending on context:

**Global error banner** — sticky at the top of the viewport, delivered via OOB swap to `#error-status`. This is the default for unhandled errors and middleware-caught failures. The banner is closable and carries two controls: **Report Issue** and **Close**. No navigation buttons — the user hasn't left their page, so "Go Back" and "Go Home" don't make sense. They can dismiss it and continue, or report it and help us fix it.

**Inline errors** — rendered into a target near the element that triggered the error. A form submission that fails shows the error next to the form, not at the top of the page. The primary content stays put. No page navigation, no lost form state. Inline errors carry contextual controls (retry, fix the input, go back) and also a **Report Issue** button.

Both surfaces include Report Issue. This is deliberate.

### Make reporting effortless or it won't happen

Users don't report bugs. They close the tab, mutter something, and work around it. The ones who do report bugs send you "it doesn't work" with no context. You reply asking for details. They reply three days later with a screenshot of the wrong page. You've now spent more time on the email thread than the bug.

The fix is structural, not motivational: **put a Report Issue button on every error, everywhere, always.**

When the user clicks Report Issue, a modal opens. They can optionally describe what they were doing — or they can just hit Submit. Either way, the server already has everything it needs: the request ID, the full error chain, the route, the status code, all captured log entries, and request metadata. The `IssueReporter` implementation decides what to do with all of this — send an email, post to Teams, create a ticket, write to a log — but the data is complete regardless of what the user types.

```
User clicks Report Issue
  → Modal: "This will send error details to our support team" [Submit]
  → Server receives: request_id + description + full ErrorTrace from SQLite store
  → IssueReporter.Report() sends structured report to the right channel
  → Developer opens report, has request_id, full error chain, and every log entry
```

The request ID is the key. It's generated per-request by middleware, propagated through context into every log call, returned in the `X-Request-ID` header, and displayed in the error component the user is looking at. When the report arrives, you don't need to ask "what were you doing?" or "what page were you on?" — you look up the request ID in the error trace store and you see the entire request lifecycle: the full error chain, which handler ran, what queries executed, where it failed, and why — all persisted in SQLite with the captured slog entries.

This means the developer never leaves the admin UI or their log viewer. No email chain. No screenshot decoding. No "can you try again while I watch the logs?" The report contains a clean trace into the code, attached to the exact moment of failure. The admin UI at `/admin/error-traces` provides a sortable, filterable, paginated browser for all persisted error traces.

```go
// Every error surface includes Report Issue — global banner and inline alike
controls := []hypermedia.Control{
    hypermedia.ReportIssueButton(hypermedia.LabelReportIssue, requestID),
}
```

The `IssueReporter` interface is a single method, defined by the consumer. It receives the full `ErrorTrace` — not just log entries — so the implementation has access to the error chain, status code, route, method, user agent, remote IP, user ID, and all captured slog entries:

```go
type IssueReporter interface {
    Report(requestID string, description string, trace *promolog.ErrorTrace) error
}
```

Plug in whatever you want — email, Slack, Teams, Jira, a database table. The point is that the user's path from "something broke" to "report submitted" is two clicks and zero typing. If you make it harder than that, they won't do it, and you'll be debugging in the dark.

## Structured Observability

Every HTTP request gets a unique request ID generated by middleware, propagated through context, and returned in the `X-Request-ID` header. When an error reaches the user, the request ID is surfaced in the error component. When the same error hits the logs, it carries the same ID. A user reporting "something went wrong" can give you the request ID, and you can trace the exact request through structured logs and the persisted error trace.

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

### Promote-on-error

Not every request deserves to be remembered. The hot path — the 99% of requests that succeed — should not pay for observability infrastructure it doesn't need. This is why logging follows a **promote-on-error** pattern: buffer everything per-request, discard on success, persist on error. The successful request pays only the cost of appending to a local slice. The failed request gets a full forensic record.

The mechanics — per-request buffers, SQLite-backed trace storage, TTL cleanup, the `Promote` call and its `ErrorTrace` payload — live in [promolog](https://github.com/catgoose/promolog). Dothog wires promolog into its middleware stack:

1. **Correlation middleware** attaches a request ID and promolog buffer to each request's context
2. **The slog handler** (wrapped by `promolog.NewHandler`) captures every log record into the buffer
3. **Error handler middleware** calls `store.Promote()` when a request fails, persisting the full trace
4. **The SSE broker** listens for promoted traces via `store.SetOnPromote()` and broadcasts them to connected admin clients

The demo page at `/demo/logging` demonstrates the full flow with simulated support reports.

### Request and background context

User-initiated actions (HTTP requests) and background operations (workers, async jobs, scheduled tasks) both flow through the same structured logger, but they're tagged differently so you can filter by origin. Requests carry a `request_id`; background work carries a `context_id` and `context_description`.

Dothog builds on top of promolog's trace storage to provide:

- **Admin UI** at `/admin/error-traces` — sortable, filterable, paginated browser for all persisted error traces
- **Real-time monitoring** — SSE broadcasts new traces as they're promoted, so the admin dashboard updates live
- **Cross-layer correlation** — the same context flows from middleware through handlers into repository calls, so a single ID traces the full operation

## Web Standards Over Libraries

**Prefer web standards over JavaScript libraries.** The browser already has the capability. Every library you don't import is a dependency you don't maintain, a bundle you don't ship, and a behavior that works even when JavaScript fails.

### Link Relations as the Navigation Model

The app declares resource relationships using [IANA link relations](https://www.iana.org/assignments/link-relations/link-relations.xhtml) — the same standard that powers `<link rel="stylesheet">` and HTTP `Link` headers. Three composable primitives:

- **Ring** — peers that link to each other (admin pages, data pages)
- **Hub** — parent with children (discovery pages like `/demo`, `/admin`)
- **Link** — explicit pairwise relationship

The server registers these at startup. The middleware emits [RFC 8288](https://www.rfc-editor.org/rfc/rfc8288) `Link` HTTP headers on every response. The same registry drives the context bars, breadcrumbs, site map footer, and the live registry inspector — one data source, many views.

Breadcrumbs walk the `rel="up"` chain. Priority: `?from=` (user's journey) → `rel="up"` (declared hierarchy) → URL path segments (fallback). The document structure drives navigation, not URL parsing.

### HTTP Headers as the API Surface

Standard HTTP headers carry meaning that no application-level protocol needs to reinvent:

- `Link` headers (RFC 8288) — machine-readable relationships on every response
- `Vary: HX-Request` — cache correctness for HTMX partials vs full pages
- `Server-Timing` — server metrics visible in browser DevTools with zero client code
- Semantic status codes: `303 See Other` (POST-redirect-GET), `409 Conflict`, `422 Unprocessable Content`

These aren't implementation details. They're contracts that CDNs, crawlers, caches, and `curl` already understand. A `Link` header is useful to any HTTP client, not just your frontend.

### Native HTML Over JavaScript

HTML has interactive elements that most developers reach for libraries to implement:

- `<dialog>` for modals — focus trapping, Escape to close, backdrop styling
- `popover` for dismissable UI — click-outside dismiss, top-layer rendering
- `<details name="...">` for exclusive accordions — zero JS
- `<datalist>` for autocomplete — native, keyboard-friendly
- `<search>` for semantic search wrappers
- `<meter>` for gauges, `<progress>` for loading states
- `formaction`/`formmethod` for multi-action forms
- `inputmode`/`enterkeyhint` for mobile keyboard control
- `autocomplete` proper values for auto-fill

Each replaces a JavaScript pattern or adds semantics that screen readers and browsers already understand.

### CSS Over JavaScript

CSS can do what used to require runtime code:

- `content-visibility: auto` — skip rendering off-screen content (performance)
- `text-wrap: balance` on headings, `text-wrap: pretty` on paragraphs (typography)
- `accent-color` — brand native form controls in one line
- View Transitions — animated page navigation via CSS, zero JS
- `[x-cloak] { display: none }` — hide Alpine elements until initialized

### Browser APIs Over Libraries

The platform ships APIs that replace entire categories of npm packages:

- `navigator.sendBeacon()` — fire-and-forget logging without blocking navigation. Replaces analytics libraries.
- `BroadcastChannel` — cross-tab sync (theme changes propagate to all tabs). No polling, no extra SSE connections.
- `sessionStorage` — history breadcrumb trail. Per-tab, ephemeral, no server round-trip.
- `localStorage` — dismiss state for context bars. Persists across sessions.
- Service Worker — offline caching for PWA, gated behind production mode.

### Speculative Loading

Prefetch on hover, not on page load. The browser's [Speculation Rules API](https://developer.mozilla.org/en-US/docs/Web/API/Speculation_Rules_API) prefetches navigation targets when the user signals intent by hovering a link — spending bandwidth only on pages the user is likely to visit, not every link on the page. This fits hypermedia naturally: the server declares which pages exist, the browser speculatively loads them based on observed user behavior. Progressive enhancement keeps it safe — Chrome gets near-instant navigation, other browsers get normal speed with zero degradation. HTMX-enhanced elements are excluded from speculation because they perform partial swaps within the current document, not full-page navigations. This composes with the service worker precache: the SW handles offline essentials (static assets, critical routes), while Speculation Rules prefetches navigation targets on demand. Two layers, no overlap, no coordination needed.

### The Principle

The web platform is not a thin wrapper around JavaScript. It's a rich runtime with decades of standardized behavior. `<dialog>` handles focus management better than any modal library because the browser team spent years on edge cases. `content-visibility` is faster than any virtual scroll library because it operates at the rendering engine level. `Link` headers are understood by crawlers, CDNs, and `curl` — no client-side code needed.

When you reach for a library, you're saying "the platform can't do this." Usually it can. Check first.
