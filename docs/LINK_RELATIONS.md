# Link Relations Reference

This document maps IANA link relations to concrete use cases in a dothog application. Some are implemented today; others are patterns waiting for a use case.

## Currently Implemented

### `rel="related"` — Peer Navigation

**Pattern:** Ring, Hub center→spoke
**Drives:** Context bars (full + local), site map footer, nav dropdowns
**Registration:** `Ring("Data", ...)`, `Hub("/demo", "Demo", ...)`, `Link(source, "related", target, title)`

Every page that belongs to a group gets `rel="related"` links to its peers. The context bar renders them. The site map footer lists them. The Link HTTP header exposes them to any HTTP client.

### `rel="up"` — Parent Navigation

**Pattern:** Hub spoke→center
**Drives:** Breadcrumbs, context bar `↑` parent link
**Registration:** Automatic from `Hub()` — every spoke gets `rel="up"` to its hub center

Breadcrumbs walk the `rel="up"` chain: current page → parent → grandparent → Home. Priority: `?from=` (explicit journey) → `rel="up"` (declared hierarchy) → URL path segments (fallback).

### `rel="create-form"` — New Resource Button

**Pattern:** `Link("/demo/inventory", "create-form", "/demo/inventory/items/new", "New Item")`
**Registered but not yet driving UI.**

**Potential:** A list page component reads its `create-form` link and renders a "New" button automatically. No template hardcoding — the button appears because the relationship exists.

```go
// The list page handler wouldn't need to pass a "createURL" — the template reads it from the registry
createLinks := hypermedia.LinksFor(currentPath, "create-form")
if len(createLinks) > 0 {
    // Render "New" button pointing to createLinks[0].Href
}
```

## Not Yet Implemented — Concrete Use Cases

### `rel="edit-form"` — Edit Button on Detail Pages

**Use case:** A detail page (e.g., `/demo/people/8`) declares `edit-form` pointing to `/demo/people/8/edit`. The detail page component reads this and renders an "Edit" button automatically.

```go
hypermedia.Link("/demo/people/:id", "edit-form", "/demo/people/:id/edit", "Edit")
```

**Challenge:** Dynamic paths with IDs. The registry uses static paths, but detail pages have dynamic segments. Options:
- Register at request time (middleware adds the link for the current resource)
- Use a pattern-based registry that matches `/demo/people/*`
- Keep edit buttons template-hardcoded (they're context-specific anyway)

### `rel="collection"` — Back to List

**Use case:** A detail page declares `collection` pointing to its list page. The detail page renders a "Back to People" link automatically.

```go
// On /demo/people/8:
hypermedia.Link("/demo/people/8", "collection", "/demo/people", "People")
```

Similar to `rel="up"` but semantically different — `up` is hierarchical (parent), `collection` is the set this item belongs to. In practice they often point to the same place.

### `rel="search"` — Search Endpoint

**Use case:** A resource declares its search/filter endpoint. The filter bar component auto-wires to it.

```go
hypermedia.Link("/demo/inventory", "search", "/demo/inventory/items", "Search Inventory")
```

**Potential:** The filter bar reads `rel="search"` to know where to send queries. No hardcoded `hx-get` URL in the template.

### `rel="first"` / `rel="last"` / `rel="next"` / `rel="prev"` — Pagination

**Use case:** Table pagination links come from Link headers instead of custom pagination structs.

```http
Link: </demo/inventory?page=2>; rel="next",
      </demo/inventory?page=5>; rel="last",
      </demo/inventory?page=1>; rel="first"
```

**Potential:** The table pagination component reads these from the response headers. The server sets them based on the current page and total count. GitHub's API uses this exact pattern.

**Challenge:** These are per-response (dynamic), not per-route (static). The current registry is static. Would need either:
- Handler-level link emission (set Link headers per response)
- A dynamic LinkSource that computes pagination links from request params

### `rel="monitor"` — SSE Endpoint

**Use case:** A resource declares its real-time update endpoint. Components auto-connect.

```go
hypermedia.Link("/demo/people", "monitor", "/sse/people", "People Updates")
```

**Potential:** An SSE component reads `rel="monitor"` to know which endpoint to connect to. No hardcoded SSE URL in the template. Adding SSE to a page means registering one link, not editing the template.

### `rel="alternate"` — Content Negotiation

**Use case:** A page declares its alternate representations.

```go
hypermedia.Link("/demo/inventory", "alternate", "/api/inventory", "JSON")
```

**Potential:** An "Export" or "API" button appears on pages that have an alternate representation. The button reads from the registry.

### `rel="describedby"` — Documentation

**Use case:** A resource links to its documentation or schema.

```go
hypermedia.Link("/demo/inventory", "describedby", "/hypermedia/crud", "CRUD Patterns")
```

**Potential:** A help icon or "How this works" link appears on pages that have documentation. Context-sensitive help driven by the registry.

### `rel="canonical"` — Bookmarkable URL

**Use case:** When HTMX loads a partial, the response includes the canonical full-page URL.

```http
Link: </demo/inventory>; rel="canonical"
```

**Potential:** The browser can identify the "real" URL for bookmarking even when the current URL came from an HTMX partial swap.

### `rel="author"` — Audit Trail

**Use case:** A resource links to who created or last modified it.

```go
// Dynamic — set per response based on the resource's creator
hypermedia.Link("/demo/inventory/items/42", "author", "/demo/people/7", "Jane Smith")
```

**Potential:** A "Created by" link on detail pages, driven by the registry.

## Static vs Dynamic Relations

The current registry is **static** — relationships are declared at startup and don't change per request. This works for:
- `related` (peer navigation)
- `up` (hierarchy)
- `create-form` (the create URL for a resource type)
- `search` (the search endpoint for a resource)
- `monitor` (the SSE endpoint for a resource)
- `describedby` (documentation links)

Some relations are inherently **dynamic** — they depend on the current resource or request:
- `edit-form` (includes the resource ID)
- `collection` (same — the detail page's collection)
- `first/next/prev/last` (pagination state)
- `canonical` (depends on how the page was reached)
- `author` (depends on who created the resource)
- `item` (each row in a list links to its detail)

Dynamic relations would need either per-response Link header emission or a request-scoped link source. The static registry handles the navigation topology; dynamic relations handle per-resource metadata.

## The Pattern

Every UI element that currently hardcodes a URL could potentially read it from the link registry:

| UI Element | Current | With Link Relations |
|-----------|---------|-------------------|
| "New" button | Template hardcodes create URL | Reads `rel="create-form"` |
| "Edit" button | Template hardcodes edit URL | Reads `rel="edit-form"` |
| "Back to list" | Template hardcodes list URL | Reads `rel="collection"` |
| Pagination | Custom struct with page URLs | Reads `rel="next/prev/first/last"` from Link headers |
| SSE connection | Template hardcodes SSE URL | Reads `rel="monitor"` |
| Help link | Template hardcodes docs URL | Reads `rel="describedby"` |
| Search bar target | Template hardcodes search URL | Reads `rel="search"` |

The principle: **the server declares what's possible, the client discovers it from the response.** This is HATEOAS applied to every UI control, not just navigation.

## Cookbook

### Adding a New Section

End-to-end example: adding a new page to the Demo hub with ring membership.

```go
// 1. Add Hub spoke in routes_links.go
hypermedia.Hub("/demo", "Demo",
    // ... existing spokes ...
    hypermedia.Rel("/demo/newpage", "New Page"),
)

// 2. Add to a Ring (or create a new one)
hypermedia.Ring("MyGroup",
    hypermedia.Rel("/demo/newpage", "New Page"),
    hypermedia.Rel("/demo/otherpage", "Other Page"),
)

// 3. Register routes
ar.e.GET("/demo/newpage", handler.HandleComponent(views.NewPage()))

// 4. Done — context bars, breadcrumbs, and site map update automatically
```

What happens:
- `/demo/newpage` gets `rel="up"` to `/demo` (from the Hub)
- `/demo/newpage` gets `rel="related"` to `/demo/otherpage` (from the Ring)
- `/demo` context bar shows "New Page" grouped under its ring
- Breadcrumbs on `/demo/newpage`: Home > Demo > New Page
- Site map footer includes "New Page" under the Demo hub

### Understanding Context Bar Resolution

The context bar resolution logic (`web/components/core/context_bar.templ`) follows these rules:

**Hub center pages** (e.g., `/demo`):
- Have outgoing `rel="related"` links to all spokes
- Context bar groups spokes by their ring membership
- Each ring becomes a named section

**Spoke pages** (e.g., `/demo/inventory`):
- Have `rel="up"` pointing to the hub center
- Fetch the hub's related links to find all sibling spokes
- Group siblings by ring membership (same as hub center view)
- Prepend a `↑ Demo` parent link at the top

**Ring-only pages** (no hub):
- Have `rel="related"` links but no `rel="up"`
- Fall back to simple grouping by the `Group` field on each link
- Show ring siblings without a parent link

### Creating a New Ring

Rings are symmetric peer groups. Every member links to every other member.

```go
hypermedia.Ring("Monitoring",
    hypermedia.Rel("/admin/health", "Health"),
    hypermedia.Rel("/admin/error-traces", "Error Traces"),
    hypermedia.Rel("/admin/sessions", "Sessions"),
)
```

A page can belong to multiple rings. On `/admin/health`, the context bar shows:
- Links from the "Monitoring" ring (Error Traces, Sessions)
- Links from any other rings `/admin/health` belongs to (e.g., "System")

### Pairwise Links

For one-off relationships that don't fit a ring or hub:

```go
hypermedia.Link("/settings", "related", "/admin/config", "Admin Config")
```

`rel="related"` auto-creates the inverse: `/admin/config` also links to `/settings`.

### Action Relations

Register semantic relationships between list pages and their create forms:

```go
hypermedia.Link("/demo/inventory", "create-form", "/demo/inventory/items/new", "New Item")
```

These are registered but not yet auto-rendered. Templates currently hardcode the "New" button, but the registry enables future auto-discovery.
