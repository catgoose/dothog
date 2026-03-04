// setup:feature:demo

package routes

import (
	"context"
	"strconv"

	"catgoose/go-htmx-demo/internals/demo"
	log "catgoose/go-htmx-demo/internals/logger"
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
	failedIDs := b.doBulkAction(c, func(ctx context.Context, id int) error {
		return b.db.DeleteItem(ctx, id)
	})
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
	failedIDs := b.doBulkAction(c, func(ctx context.Context, id int) error {
		item, err := b.db.GetItem(ctx, id)
		if err != nil {
			return err
		}
		item.Active = true
		return b.db.UpdateItem(ctx, item)
	})
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
	failedIDs := b.doBulkAction(c, func(ctx context.Context, id int) error {
		item, err := b.db.GetItem(ctx, id)
		if err != nil {
			return err
		}
		item.Active = false
		return b.db.UpdateItem(ctx, item)
	})
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

// doBulkAction parses form IDs and runs actionFn for each, returning any failed IDs.
func (b *bulkRoutes) doBulkAction(c echo.Context, actionFn func(ctx context.Context, id int) error) []int {
	_ = c.Request().ParseForm()
	var failedIDs []int
	ctx := c.Request().Context()
	for _, raw := range c.Request().Form["ids"] {
		if id, err := strconv.Atoi(raw); err == nil && id > 0 {
			if err := actionFn(ctx, id); err != nil {
				failedIDs = append(failedIDs, id)
			}
		}
	}
	return failedIDs
}

func (b *bulkRoutes) buildBulkContent(c echo.Context) (hypermedia.FilterBar, templ.Component, error) {
	const perPage = 20
	tc, err := buildTableContent(c, b.db, parseTableParams(c, perPage),
		bulkBase+"/items", "#bulk-table-container",
		hypermedia.TableCol{Label: "Status"},
	)
	if err != nil {
		return hypermedia.FilterBar{}, nil, err
	}
	body := views.BulkItemsBody(tc.Items)
	container := views.BulkTableContainer(tc.Cols, body, tc.Info)
	return tc.Bar, container, nil
}
