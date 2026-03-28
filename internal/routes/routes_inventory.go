// setup:feature:demo

package routes

import (
	"fmt"
	"strconv"

	"catgoose/dothog/internal/demo"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/internal/routes/hypermedia"
	"catgoose/dothog/internal/routes/params"
	"catgoose/dothog/web/views"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
)

const inventoryBase = "/demo/inventory"

type inventoryRoutes struct{ db *demo.DB }

func (ar *appRoutes) initInventoryRoutes(db *demo.DB) {
	hypermedia.Ring("Data",
		hypermedia.Rel("/demo/inventory", "Inventory"),
		hypermedia.Rel("/demo/catalog", "Catalog"),
		hypermedia.Rel("/demo/bulk", "Bulk Ops"),
		hypermedia.Rel("/demo/people", "People"),
		hypermedia.Rel("/demo/vendors", "Vendors"),
	)

	d := &inventoryRoutes{db: db}
	ar.e.GET(inventoryBase, d.handleInventoryPage)
	ar.e.GET(inventoryBase+"/items", d.handleInventoryItems)
	// Static paths must be registered before parameterized ones.
	ar.e.GET(inventoryBase+"/items/new", d.handleNewItemForm)
	ar.e.GET(inventoryBase+"/items/new/cancel", d.handleNewItemCancel)
	ar.e.POST(inventoryBase+"/items", d.handleCreateItem)
	ar.e.GET(inventoryBase+"/items/:id", d.handleItemRow)
	ar.e.GET(inventoryBase+"/items/:id/edit", d.handleEditItemForm)
	ar.e.PUT(inventoryBase+"/items/:id", d.handleUpdateItem)
	ar.e.DELETE(inventoryBase+"/items/:id", d.handleDeleteItem)
}

func (d *inventoryRoutes) handleInventoryPage(c echo.Context) error {
	bar, container, err := d.buildInventoryContent(c)
	if err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to load inventory", err)
	}
	return handler.RenderBaseLayout(c, views.InventoryPage(bar, container))
}

func (d *inventoryRoutes) handleInventoryItems(c echo.Context) error {
	_, container, err := d.buildInventoryContent(c)
	if err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to load items", err)
	}
	setTableReplaceURL(c, inventoryBase)
	return handler.RenderComponent(c, container)
}

func (d *inventoryRoutes) handleNewItemForm(c echo.Context) error {
	filterQuery := filterQueryFromHXCurrentURL(c)
	saveURL := inventoryBase + "/items"
	if filterQuery != "" {
		saveURL = inventoryBase + "/items?" + filterQuery
	}
	return handler.RenderComponent(c, views.InventoryEditRow(demo.Item{}, true, saveURL, inventoryBase+"/items/new/cancel"))
}

func (d *inventoryRoutes) handleNewItemCancel(c echo.Context) error {
	return handler.RenderComponent(c, views.NewInventoryPlaceholder())
}

func parseItemFromForm(c echo.Context) demo.Item {
	price, _ := strconv.ParseFloat(c.FormValue("price"), 64)
	stock, _ := strconv.Atoi(c.FormValue("stock"))
	return demo.Item{
		Name:     c.FormValue("name"),
		Category: c.FormValue("category"),
		Price:    price,
		Stock:    stock,
		Active:   c.FormValue("active") == "true",
	}
}

func (d *inventoryRoutes) handleCreateItem(c echo.Context) error {
	item := parseItemFromForm(c)
	if _, err := d.db.CreateItem(c.Request().Context(), item); err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to create item", err)
	}
	_, container, err := d.buildInventoryContent(c)
	if err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to reload table", err)
	}
	setTableReplaceURL(c, inventoryBase)
	return handler.RenderComponent(c, container)
}

func (d *inventoryRoutes) handleItemRow(c echo.Context) error {
	id, err := params.ParseParamID(c, "id")
	if err != nil {
		return handler.HandleHypermediaError(c, 400, "Invalid item ID", err)
	}
	item, err := d.db.GetItem(c.Request().Context(), id)
	if err != nil {
		return handler.HandleHypermediaError(c, 404, "Item not found", err)
	}
	if c.Request().Header.Get("HX-Request") == "" {
		handler.SetPageLabel(c, item.Name)
		return handler.RenderBaseLayout(c, views.InventoryDetailPage(item))
	}
	return handler.RenderComponent(c, views.InventoryItemRow(item))
}

func (d *inventoryRoutes) handleEditItemForm(c echo.Context) error {
	id, err := params.ParseParamID(c, "id")
	if err != nil {
		return handler.HandleHypermediaError(c, 400, "Invalid item ID", err)
	}
	item, err := d.db.GetItem(c.Request().Context(), id)
	if err != nil {
		return handler.HandleHypermediaError(c, 404, "Item not found", err)
	}
	filterQuery := filterQueryFromHXCurrentURL(c)
	baseURL := fmt.Sprintf(inventoryBase+"/items/%d", id)
	saveURL := baseURL
	if filterQuery != "" {
		saveURL = baseURL + "?" + filterQuery
	}
	return handler.RenderComponent(c, views.InventoryEditRow(item, false, saveURL, baseURL))
}

func (d *inventoryRoutes) handleUpdateItem(c echo.Context) error {
	id, err := params.ParseParamID(c, "id")
	if err != nil {
		return handler.HandleHypermediaError(c, 400, "Invalid item ID", err)
	}
	item := parseItemFromForm(c)
	item.ID = id
	if err := d.db.UpdateItem(c.Request().Context(), item); err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to update item", err)
	}
	_, container, err := d.buildInventoryContent(c)
	if err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to reload table", err)
	}
	setTableReplaceURL(c, inventoryBase)
	return handler.RenderComponent(c, container)
}

func (d *inventoryRoutes) handleDeleteItem(c echo.Context) error {
	id, err := params.ParseParamID(c, "id")
	if err != nil {
		return handler.HandleHypermediaError(c, 400, "Invalid item ID", err)
	}
	if err := d.db.DeleteItem(c.Request().Context(), id); err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to delete item", err)
	}
	applyFilterFromCurrentURL(c)
	_, container, err := d.buildInventoryContent(c)
	if err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to reload table", err)
	}
	setTableReplaceURL(c, inventoryBase)
	return handler.RenderComponent(c, container)
}

func (d *inventoryRoutes) buildInventoryContent(c echo.Context) (hypermedia.FilterBar, templ.Component, error) {
	const perPage = 20
	tc, err := buildTableContent(c, d.db, parseTableParams(c, perPage),
		inventoryBase+"/items", "#inventory-table-container",
		hypermedia.TableCol{Label: "Actions"},
	)
	if err != nil {
		return hypermedia.FilterBar{}, nil, err
	}
	body := views.InventoryItemsBody(tc.Items)
	container := views.InventoryTableContainer(tc.Cols, body, tc.Info)
	return tc.Bar, container, nil
}
