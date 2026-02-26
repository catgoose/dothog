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

const catalogBase = "/tables/catalog"

type catalogRoutes struct{ db *demo.DB }

func (ar *appRoutes) initCatalogRoutes(db *demo.DB) {
	cat := &catalogRoutes{db: db}
	ar.e.GET(catalogBase, cat.handleCatalogPage)
	ar.e.GET(catalogBase+"/items", cat.handleCatalogItems)
	ar.e.GET(catalogBase+"/items/:id/details", cat.handleCatalogItemDetails)
}

func (cat *catalogRoutes) handleCatalogPage(c echo.Context) error {
	bar, container, err := cat.buildCatalogContent(c)
	if err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to load catalog", err)
	}
	return handler.RenderBaseLayout(c, views.CatalogPage(bar, container))
}

func (cat *catalogRoutes) handleCatalogItems(c echo.Context) error {
	_, container, err := cat.buildCatalogContent(c)
	if err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to load items", err)
	}
	setTableReplaceURL(c, catalogBase)
	return handler.RenderComponent(c, container)
}

func (cat *catalogRoutes) handleCatalogItemDetails(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		return handler.HandleHypermediaError(c, 400, "Invalid item ID", fmt.Errorf("id=%q", c.Param("id")))
	}
	item, err := cat.db.GetItem(c.Request().Context(), id)
	if err != nil {
		return handler.HandleHypermediaError(c, 404, "Item not found", err)
	}
	return handler.RenderComponent(c, views.CatalogDetailContent(item))
}

func (cat *catalogRoutes) buildCatalogContent(c echo.Context) (hypermedia.FilterBar, templ.Component, error) {
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

	items, total, err := cat.db.ListItems(c.Request().Context(), q, category, active, sort, dir, page, perPage)
	if err != nil {
		return hypermedia.FilterBar{}, nil, err
	}

	bar := hypermedia.NewFilterBar(catalogBase+"/items", "#catalog-table-container",
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
		hypermedia.SortableCol("name", "Name", sort, dir, sortBase, "#catalog-table-container", "#filter-form"),
		hypermedia.SortableCol("category", "Category", sort, dir, sortBase, "#catalog-table-container", "#filter-form"),
		hypermedia.SortableCol("price", "Price", sort, dir, sortBase, "#catalog-table-container", "#filter-form"),
		hypermedia.SortableCol("stock", "Stock", sort, dir, sortBase, "#catalog-table-container", "#filter-form"),
		{Label: "Status"},
		{Label: "Details"},
	}

	info := hypermedia.PageInfo{
		Page:       page,
		PerPage:    perPage,
		TotalItems: total,
		TotalPages: hypermedia.ComputeTotalPages(total, perPage),
		BaseURL:    pageBase,
		Target:     "#catalog-table-container",
		Include:    "#filter-form",
	}

	body := views.CatalogItemsBody(items)
	container := views.CatalogTableContainer(cols, body, info)
	return bar, container, nil
}
