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

Create `internal/routes/routes_myfeature.go`:

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

In `internal/routes/routes_links.go`, add the page to a hub and/or ring:

```go
// Add as a spoke of the Demo hub
hypermedia.Hub("/demo", "Demo",
    // ... existing spokes ...
    hypermedia.Rel("/demo/myfeature", "My Feature"),
)

// Add to a ring for peer navigation
hypermedia.Ring("Data",
    // ... existing members ...
    hypermedia.Rel("/demo/myfeature", "My Feature"),
)
```

### 5. Done

Context bars, breadcrumbs, and the site map footer update automatically based on the link registry. No template changes needed for navigation.

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
