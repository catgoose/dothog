// setup:feature:demo

package routes

import (
	"strconv"

	log "catgoose/go-htmx-demo/internals/logger"
	"catgoose/go-htmx-demo/internals/demo"
	"catgoose/go-htmx-demo/internals/routes/handler"
	"catgoose/go-htmx-demo/internals/routes/hypermedia"
	"catgoose/go-htmx-demo/web/views"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
)

const bulkBase = "/tables/bulk"

type bulkRoutes struct{ db *demo.DB }

func (ar *appRoutes) initBulkRoutes(db *demo.DB) {
	b := &bulkRoutes{db: db}
	ar.e.GET(bulkBase, b.handleBulkPage)
	ar.e.GET(bulkBase+"/items", b.handleBulkItems)
	ar.e.DELETE(bulkBase+"/items", b.handleBulkDeleteItems)
	ar.e.PUT(bulkBase+"/items/activate", b.handleBulkActivateItems)
	ar.e.PUT(bulkBase+"/items/deactivate", b.handleBulkDeactivateItems)
}

func (b *bulkRoutes) handleBulkPage(c echo.Context) error {
	bar, container, err := b.buildBulkContent(c)
	if err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to load bulk table", err)
	}
	return handler.RenderBaseLayout(c, views.BulkPage(bar, container))
}

func (b *bulkRoutes) handleBulkItems(c echo.Context) error {
	_, container, err := b.buildBulkContent(c)
	if err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to load items", err)
	}
	setTableReplaceURL(c, bulkBase)
	return handler.RenderComponent(c, container)
}

func (b *bulkRoutes) handleBulkDeleteItems(c echo.Context) error {
	_ = c.Request().ParseForm()
	var failedIDs []int
	for _, raw := range c.Request().Form["ids"] {
		if id, err := strconv.Atoi(raw); err == nil && id > 0 {
			if err := b.db.DeleteItem(c.Request().Context(), id); err != nil {
				failedIDs = append(failedIDs, id)
			}
		}
	}
	if len(failedIDs) > 0 {
		log.WithContext(c.Request().Context()).Warn("Bulk delete: failed items", "ids", failedIDs)
	}
	applyFilterFromCurrentURL(c)
	_, container, err := b.buildBulkContent(c)
	if err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to reload table", err)
	}
	setTableReplaceURL(c, bulkBase)
	return handler.RenderComponent(c, container)
}

func (b *bulkRoutes) handleBulkActivateItems(c echo.Context) error {
	_ = c.Request().ParseForm()
	var failedIDs []int
	for _, raw := range c.Request().Form["ids"] {
		if id, err := strconv.Atoi(raw); err == nil && id > 0 {
			if item, err := b.db.GetItem(c.Request().Context(), id); err == nil {
				item.Active = true
				if err := b.db.UpdateItem(c.Request().Context(), item); err != nil {
					failedIDs = append(failedIDs, id)
				}
			} else {
				failedIDs = append(failedIDs, id)
			}
		}
	}
	if len(failedIDs) > 0 {
		log.WithContext(c.Request().Context()).Warn("Bulk activate: failed items", "ids", failedIDs)
	}
	applyFilterFromCurrentURL(c)
	_, container, err := b.buildBulkContent(c)
	if err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to reload table", err)
	}
	setTableReplaceURL(c, bulkBase)
	return handler.RenderComponent(c, container)
}

func (b *bulkRoutes) handleBulkDeactivateItems(c echo.Context) error {
	_ = c.Request().ParseForm()
	var failedIDs []int
	for _, raw := range c.Request().Form["ids"] {
		if id, err := strconv.Atoi(raw); err == nil && id > 0 {
			if item, err := b.db.GetItem(c.Request().Context(), id); err == nil {
				item.Active = false
				if err := b.db.UpdateItem(c.Request().Context(), item); err != nil {
					failedIDs = append(failedIDs, id)
				}
			} else {
				failedIDs = append(failedIDs, id)
			}
		}
	}
	if len(failedIDs) > 0 {
		log.WithContext(c.Request().Context()).Warn("Bulk deactivate: failed items", "ids", failedIDs)
	}
	applyFilterFromCurrentURL(c)
	_, container, err := b.buildBulkContent(c)
	if err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to reload table", err)
	}
	setTableReplaceURL(c, bulkBase)
	return handler.RenderComponent(c, container)
}

func (b *bulkRoutes) buildBulkContent(c echo.Context) (hypermedia.FilterBar, templ.Component, error) {
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

	items, total, err := b.db.ListItems(c.Request().Context(), q, category, active, sort, dir, page, perPage)
	if err != nil {
		return hypermedia.FilterBar{}, nil, err
	}

	bar := hypermedia.NewFilterBar(bulkBase+"/items", "#bulk-table-container",
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
		hypermedia.SortableCol("name", "Name", sort, dir, sortBase, "#bulk-table-container", "#filter-form"),
		hypermedia.SortableCol("category", "Category", sort, dir, sortBase, "#bulk-table-container", "#filter-form"),
		hypermedia.SortableCol("price", "Price", sort, dir, sortBase, "#bulk-table-container", "#filter-form"),
		hypermedia.SortableCol("stock", "Stock", sort, dir, sortBase, "#bulk-table-container", "#filter-form"),
		{Label: "Status"},
	}

	info := hypermedia.PageInfo{
		Page:       page,
		PerPage:    perPage,
		TotalItems: total,
		TotalPages: hypermedia.ComputeTotalPages(total, perPage),
		BaseURL:    pageBase,
		Target:     "#bulk-table-container",
		Include:    "#filter-form",
	}

	body := views.BulkItemsBody(items)
	container := views.BulkTableContainer(cols, body, info)
	return bar, container, nil
}
