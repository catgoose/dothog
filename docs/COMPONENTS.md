# Component Catalog

Practical patterns for building pages in dothog. Each section shows the route, handler, template, and wiring needed.

## Table with Filtering, Sorting, and Pagination

The inventory page (`/demo/inventory`) is the reference implementation.

### Route Structure

```go
const inventoryBase = "/demo/inventory"

ar.e.GET(inventoryBase, d.handleInventoryPage)           // Full page
ar.e.GET(inventoryBase+"/items", d.handleInventoryItems)  // Table fragment
ar.e.GET(inventoryBase+"/items/:id", d.handleItemRow)     // Single row
```

Two GET handlers serve the same content:
- `/demo/inventory` — returns the full page (layout + filter bar + table)
- `/demo/inventory/items` — returns just the table container (for HTMX swaps)

### Handler Pattern

```go
func (d *inventoryRoutes) handleInventoryPage(c echo.Context) error {
    bar, container, err := d.buildInventoryContent(c)
    if err != nil {
        return handler.HandleHypermediaError(c, 500, "Failed to load inventory", err)
    }
    return handler.RenderBaseLayout(c, views.InventoryPage(bar, container))
}

func (d *inventoryRoutes) handleInventoryItems(c echo.Context) error {
    bar, container, err := d.buildInventoryContent(c)
    if err != nil {
        return handler.HandleHypermediaError(c, 500, "Failed to load items", err)
    }
    if hx.IsBoosted(c) {
        return handler.RenderBaseLayout(c, views.InventoryPage(bar, container))
    }
    setTableReplaceURL(c, inventoryBase)
    return handler.RenderComponent(c, container)
}
```

The `buildInventoryContent` helper uses `parseTableParams(c, perPage)` to extract sort, filter, and page params from the query string, then `buildTableContent` to query the database and build the table component.

### Template Pattern

```templ
templ InventoryPage(bar hypermedia.FilterBar, tableContainer templ.Component) {
    <div class="p-4 space-y-4">
        @components.FilterBar(bar)
        @tableContainer
    </div>
}

templ InventoryTableContainer(cols []hypermedia.TableCol, body templ.Component, info hypermedia.PageInfo) {
    <div id="inventory-table-container">
        @components.Table(cols, body, info)
    </div>
}
```

The filter bar lives outside the table container so HTMX swaps don't replace it. The table container has a stable `id` that HTMX targets.

### HTMX Wiring

- `hx-get="/demo/inventory/items"` targets `#inventory-table-container`
- `hx-push-url="true"` makes filter/sort/page state bookmarkable
- Sort headers, filter inputs, and pagination links all target the same container
- `setTableReplaceURL(c, base)` sets `HX-Replace-URL` to keep the browser URL clean

## Detail Page

### Route Structure

```go
ar.e.GET(inventoryBase+"/items/:id", d.handleItemRow)
ar.e.GET(inventoryBase+"/items/:id/edit", d.handleEditItemForm)
ar.e.PUT(inventoryBase+"/items/:id", d.handleUpdateItem)
```

### Handler Pattern

```go
func (d *inventoryRoutes) handleItemRow(c echo.Context) error {
    id, err := params.ParseParamID(c, "id")
    if err != nil {
        return handler.HandleHypermediaError(c, 400, "Invalid item ID", err)
    }
    item, err := d.db.GetItem(c.Request().Context(), id)
    if err != nil {
        return handler.HandleHypermediaError(c, 404, "Item not found", err)
    }
    // Direct navigation or hx-boost: render full page
    if !hx.IsHTMX(c) || hx.IsBoosted(c) {
        handler.SetPageLabel(c, item.Name)
        return handler.RenderBaseLayout(c, views.InventoryDetailPage(item))
    }
    // HTMX partial: render just the row
    return handler.RenderComponent(c, views.InventoryItemRow(item))
}
```

`SetPageLabel(c, item.Name)` overrides the terminal breadcrumb — instead of showing the ID, it shows the item name.

### Breadcrumbs

Breadcrumbs resolve automatically via `rel="up"`:
- `/demo/inventory/items/42` has no explicit `rel="up"`, so breadcrumbs fall back to URL path: `Home > Demo > Inventory > Widget A`
- The Hub declaration `Hub("/demo", "Demo", Rel("/demo/inventory", "Inventory"))` gives `/demo/inventory` a `rel="up"` to `/demo`, so the chain resolves as: `Home > Demo > Inventory`

## Form with Validation

### Route Structure

```go
ar.e.GET(inventoryBase+"/items/new", d.handleNewItemForm)
ar.e.POST(inventoryBase+"/items", d.handleCreateItem)
```

### Handler Pattern

```go
func (d *inventoryRoutes) handleCreateItem(c echo.Context) error {
    item := parseItemFromForm(c)
    if _, err := d.db.CreateItem(c.Request().Context(), item); err != nil {
        return handler.HandleHypermediaError(c, 500, "Failed to create item", err)
    }
    // Reload the full table after creation
    _, container, err := d.buildInventoryContent(c)
    if err != nil {
        return handler.HandleHypermediaError(c, 500, "Failed to reload table", err)
    }
    setTableReplaceURL(c, inventoryBase)
    return handler.RenderComponent(c, container)
}
```

### Validation Pattern

For forms that need field-level validation:
1. POST handler validates input
2. On failure, return HTTP 422 with the form HTML containing error messages
3. HTMX swaps the form container with the error state
4. The user corrects and resubmits

### Template Pattern

Use `components.Controls()` with `hypermedia.Control` for action buttons:

```templ
@components.Controls([]hypermedia.Control{
    {
        Kind:    hypermedia.ControlKindHTMX,
        Label:   "+ Add Item",
        Variant: hypermedia.VariantPrimary,
        Swap:    hypermedia.SwapOuterHTML,
        HxRequest: hypermedia.HxGet("/demo/inventory/items/new", "#new-item-row"),
    },
})
```

## Modal

Modals use the HTML `<dialog>` element with `showModal()`.

### Handler

The GET handler returns a modal HTML fragment:

```go
ar.e.GET("/demo/people/:id/modal", d.handlePersonModal)
```

### Template Pattern

```templ
templ PersonModal(person demo.Person) {
    <dialog id="person-modal" class="modal">
        <div class="modal-box">
            <h3>{ person.Name }</h3>
            <!-- Modal content -->
            <form method="dialog">
                <button class="btn">Close</button>
            </form>
        </div>
        <form method="dialog" class="modal-backdrop">
            <button>close</button>
        </form>
    </dialog>
}
```

### HTMX Wiring

```html
<button
    hx-get="/demo/people/8/modal"
    hx-target="#modal-container"
    hx-on::load="this.querySelector('dialog')?.showModal()"
>
    View Details
</button>
<div id="modal-container"></div>
```

The modal component (`web/components/core/modal.templ`) provides reusable modal shells. The report-issue modal (`web/components/core/report_issue.templ`) is a concrete example.

## Inline Editing (Table Rows)

### Route Structure

```go
ar.e.GET(inventoryBase+"/items/:id/edit", d.handleEditItemForm)
ar.e.PUT(inventoryBase+"/items/:id", d.handleUpdateItem)
ar.e.DELETE(inventoryBase+"/items/:id", d.handleDeleteItem)
```

### Handler Pattern

```go
func (d *inventoryRoutes) handleEditItemForm(c echo.Context) error {
    id, err := params.ParseParamID(c, "id")
    // ... fetch item ...
    saveURL := fmt.Sprintf(inventoryBase+"/items/%d", id)
    return handler.RenderComponent(c, views.InventoryEditRow(item, false, saveURL, baseURL))
}
```

### Template Pattern

Row actions use `hypermedia.TableRowActions()`:

```templ
@components.Controls(hypermedia.TableRowActions(hypermedia.TableRowActionCfg{
    EditURL:     editURL,
    DeleteURL:   deleteURL,
    RowTarget:   rowTarget,       // "#item-row-42"
    TableTarget: "#inventory-table-container",
    ConfirmMsg:  "Delete this item?",
}))
```

Edit/delete swap the individual row. After mutation, the handler reloads and returns the full table container.

## Adding a New Page

Step-by-step guide for adding a new page to the application:

### 1. Create the route file

Create `internal/routes/myfeature.go`:

```go
package routes

import (
    "catgoose/dothog/internal/routes/handler"
    "catgoose/dothog/web/views"
    "github.com/labstack/echo/v4"
)

func (ar *appRoutes) initMyFeatureRoutes() {
    ar.e.GET("/demo/myfeature", func(c echo.Context) error {
        return handler.RenderBaseLayout(c, views.MyFeaturePage())
    })
}
```

### 2. Create the templ view

Create `web/views/myfeature.templ`:

```templ
package views

templ MyFeaturePage() {
    <div class="p-4">
        <h1 class="text-2xl font-bold">My Feature</h1>
        <!-- Page content -->
    </div>
}
```

### 3. Register routes

In `internal/routes/routes.go`, call the initializer inside `InitRoutes()`:

```go
ar.initMyFeatureRoutes()
```

### 4. Add to the link registry

Pick the seam that matches your feature's ownership:

- **Scaffold-facing feature** (always-on, no setup gate): co-locate the
  registration with the route initializer. Example from
  `internal/routes/examples.go`:

  ```go
  func (ar *AppRoutes) initMyFeatureRoutes() {
      linkwell.Hub("/myfeature", "My Feature",
          linkwell.Rel("/myfeature/child", "Child"),
      )
      ar.e.GET("/myfeature", handler.HandleComponent(views.MyFeaturePage()))
  }
  ```

- **Demo-only feature**: add to `internal/routes/links.go::initLinkRelations`
  (the file is `setup:feature:demo` so the call disappears when demo is
  stripped):

  ```go
  // Add as a spoke of an existing hub (Hub is append-only across calls)
  linkwell.Hub("/demo", "Demo",
      linkwell.Rel("/demo/myfeature", "My Feature"),
  )

  linkwell.Ring("Data",
      linkwell.Rel("/demo/myfeature", "My Feature"),
  )
  ```

The **curated top nav** in `internal/routes/handler/handler.go::appNavNavConfig`
is a separate concern from the link registry. If your feature wants a
top-nav entry, add it manually; the link registry alone does not drive top
nav.

### 5. Done

Context bars, breadcrumbs, and the footer site map update automatically
based on the link registry. No template changes needed for navigation.

## Page Composition Patterns

For page-level layout, use DaisyUI classes directly. These patterns appear across multiple apps and are the idiomatic way to compose pages.

### Card wrapper

Wrap content sections in a DaisyUI card with a title:

```templ
<div class="card bg-base-100 border border-base-300">
    <div class="card-body">
        <h2 class="card-title text-base">Settings</h2>
        <!-- card content -->
    </div>
</div>
```

### Stacked layout

Stack cards or sections vertically with consistent spacing:

```templ
<div class="space-y-6">
    <!-- cards, forms, or any block content -->
</div>
```

### Composed example

A typical settings or detail page:

```templ
templ MySettingsPage() {
    <div class="max-w-2xl mx-auto space-y-6">
        <h1 class="text-2xl font-bold">Settings</h1>
        <div class="card bg-base-100 border border-base-300">
            <div class="card-body">
                <h2 class="card-title text-base">Appearance</h2>
                <!-- theme picker, layout toggle -->
            </div>
        </div>
        <div class="card bg-base-100 border border-base-300">
            <div class="card-body">
                <h2 class="card-title text-base">Navigation</h2>
                <!-- context bar toggles -->
            </div>
        </div>
    </div>
}
```

These are too thin to warrant a reusable component — the HTML is self-explanatory and hiding it behind `Card(title)` or `StackedLayout()` adds indirection without reducing complexity. Write the markup directly.

## Multi-Region OOB Responses

Most HTMX updates are a single fragment swapped into the request's `hx-target` via `handler.RenderComponent`. When one action must also refresh other regions, the standard mechanism is out-of-band swaps: a component bakes its own `id` and `hx-swap-oob` attribute, and the response carries it alongside the main fragment.

`handler.RenderHypermedia` composes such a response from explicit, named regions without a bespoke per-endpoint composite template. It renders `Main` into the `hx-target`, then each `OOBFragment` in declared order:

```go
func (d *cartRoutes) handleAddToCart(c echo.Context) error {
    // ...mutate, then reload the regions this action changed...
    return handler.RenderHypermedia(c, handler.HypermediaResponse{
        Main: views.CartLineItems(items),
        Fragments: []handler.OOBFragment{
            {TargetID: "cart-count", Swap: "true", Component: views.CartCount(n)},
            {TargetID: "flash", Swap: "outerHTML", Component: views.Flash("Added to cart")},
        },
    })
}
```

Each component renders verbatim and owns its own `id` + `hx-swap-oob` markup (the standard OOB contract). `TargetID`/`Swap` restate that contract in code so the response shape is reviewable and testable; the builder never parses or rewrites the markup, so the author keeps each declared field and the component's own attribute in sync.

This complements self-OOB templ regions rather than replacing them. A single component that emits several `hx-swap-oob` blocks is still the right choice when those regions always update together; reach for `RenderHypermedia` when a handler assembles a varying set of regions from independent components.

## Resource & Identity Affordances

`web/components/core` (imported as `components`) exposes a small shared vocabulary so traversable resources and identity read the same across apps. Two rules hold throughout: an anchor always navigates (button styling never changes that), and secondary identity text renders outside the linked name so it never reads as its own link.

| Use | When |
|-----|------|
| plain `<a class="link …">` / DaisyUI `btn` | One-off link or button with no resource semantics. |
| `TextResourceLink` | Name a single entity/resource; visibly linked at rest. |
| `DestinationResourceLink` | Navigate to a sibling or destination view; visible link, never button-styled. |
| `ActionResourceLink` | Rare command-like navigation that should look like a button but stay an `<a>`. |
| `TileResourceLink` | A framed, clickable count/status tile wrapping caller children. |
| `IdentityDisplay` | Show an identity (avatar + name + optional caption) with no navigation. |
| `IdentityResourceLink` | Same as `IdentityDisplay`, but the primary name links; avatar and caption stay outside the anchor. |

Identity primitives take an `IdentityChip` (precomputed `DisplayName`, `Secondary`, `AvatarURL`, `Initials`, `Resolved`, and optional fallback class overrides) and an `IdentitySize` (`IdentitySizeCompact` for dense rows, `IdentitySizeBase` otherwise). The chip is deliberately app-neutral — callers project their own person/user model into it.

## Adding a New Component

### 1. Create the component

Create `web/components/core/mywidget.templ`:

```templ
package components

templ MyWidget(title string, items []string) {
    <div class="card bg-base-100 shadow">
        <div class="card-body">
            <h2 class="card-title">{ title }</h2>
            <ul>
                for _, item := range items {
                    <li>{ item }</li>
                }
            </ul>
        </div>
    </div>
}
```

### 2. Feature-gate if demo-only

If the component is only for the demo app, add the feature gate comment at the top of the file:

```go
// setup:feature:demo
package components
```

This causes `mage setup` to remove the entire file when the `demo` feature is not selected.

### 3. Use hypermedia controls

For action buttons, use `hypermedia.Control` instead of hardcoding URLs:

```templ
@components.Controls([]hypermedia.Control{
    {
        Kind:    hypermedia.ControlKindHTMX,
        Label:   "Refresh",
        Variant: hypermedia.VariantGhost,
        HxRequest: hypermedia.HxGet("/demo/myfeature/data", "#data-container"),
    },
})
```

This keeps URLs in the handler/registry layer and lets the component be reusable.
