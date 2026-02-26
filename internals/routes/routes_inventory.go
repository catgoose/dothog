// setup:feature:demo

package routes

import (
	"fmt"
	"strconv"

	"catgoose/go-htmx-demo/internals/demo"
	"catgoose/go-htmx-demo/internals/routes/handler"
	"catgoose/go-htmx-demo/internals/routes/hypermedia"
	"catgoose/go-htmx-demo/web/views"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
)

const inventoryBase = "/tables/inventory"

type inventoryRoutes struct{ db *demo.DB }

func (ar *appRoutes) initInventoryRoutes(db *demo.DB) {
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

func (d *inventoryRoutes) handleCreateItem(c echo.Context) error {
	price, _ := strconv.ParseFloat(c.FormValue("price"), 64)
	stock, _ := strconv.Atoi(c.FormValue("stock"))
	item := demo.Item{
		Name:     c.FormValue("name"),
		Category: c.FormValue("category"),
		Price:    price,
		Stock:    stock,
		Active:   c.FormValue("active") == "true",
	}
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
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		return handler.HandleHypermediaError(c, 400, "Invalid item ID", fmt.Errorf("id=%q", c.Param("id")))
	}
	item, err := d.db.GetItem(c.Request().Context(), id)
	if err != nil {
		return handler.HandleHypermediaError(c, 404, "Item not found", err)
	}
	return handler.RenderComponent(c, views.InventoryItemRow(item))
}

func (d *inventoryRoutes) handleEditItemForm(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		return handler.HandleHypermediaError(c, 400, "Invalid item ID", fmt.Errorf("id=%q", c.Param("id")))
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
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		return handler.HandleHypermediaError(c, 400, "Invalid item ID", fmt.Errorf("id=%q", c.Param("id")))
	}
	price, _ := strconv.ParseFloat(c.FormValue("price"), 64)
	stock, _ := strconv.Atoi(c.FormValue("stock"))
	item := demo.Item{
		ID:       id,
		Name:     c.FormValue("name"),
		Category: c.FormValue("category"),
		Price:    price,
		Stock:    stock,
		Active:   c.FormValue("active") == "true",
	}
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
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		return handler.HandleHypermediaError(c, 400, "Invalid item ID", fmt.Errorf("id=%q", c.Param("id")))
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
	q := c.QueryParam("q")
	category := c.QueryParam("category")
	active := c.QueryParam("active")
	sort := c.QueryParam("sort")
	dir := c.QueryParam("dir")
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	const perPage = 20

	items, total, err := d.db.ListItems(c.Request().Context(), q, category, active, sort, dir, page, perPage)
	if err != nil {
		return hypermedia.FilterBar{}, nil, err
	}

	bar := hypermedia.NewFilterBar(inventoryBase+"/items", "#inventory-table-container",
		hypermedia.SearchField("q", "Search items\u2026", q),
		hypermedia.SelectField("category", "Category", category,
			hypermedia.SelectOptions(category,
				"", "All",
				"Electronics", "Electronics",
				"Clothing", "Clothing",
				"Food", "Food",
				"Books", "Books",
				"Sports", "Sports",
			)),
		hypermedia.CheckboxField("active", "Active only", active),
	)

	sortBase := stripParams(c.Request().URL, "sort", "dir")
	pageBase := stripParams(c.Request().URL, "page")

	cols := []hypermedia.TableCol{
		hypermedia.SortableCol("name", "Name", sort, dir, sortBase, "#inventory-table-container", "#filter-form"),
		hypermedia.SortableCol("category", "Category", sort, dir, sortBase, "#inventory-table-container", "#filter-form"),
		hypermedia.SortableCol("price", "Price", sort, dir, sortBase, "#inventory-table-container", "#filter-form"),
		hypermedia.SortableCol("stock", "Stock", sort, dir, sortBase, "#inventory-table-container", "#filter-form"),
		{Label: "Status"},
		{Label: "Actions"},
	}

	info := hypermedia.PageInfo{
		Page:       page,
		PerPage:    perPage,
		TotalItems: total,
		TotalPages: hypermedia.ComputeTotalPages(total, perPage),
		BaseURL:    pageBase,
		Target:     "#inventory-table-container",
		Include:    "#filter-form",
	}

	body := views.InventoryItemsBody(items)
	container := views.InventoryTableContainer(cols, body, info)
	return bar, container, nil
}
