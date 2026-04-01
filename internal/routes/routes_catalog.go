// setup:feature:demo

package routes

import (
	"catgoose/dothog/internal/demo"
	"catgoose/dothog/internal/routes/handler"
	htmx "github.com/angelofallars/htmx-go"
	"github.com/catgoose/linkwell"
	"catgoose/dothog/internal/routes/params"
	"catgoose/dothog/web/views"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
)

const catalogBase = "/demo/catalog"

type catalogRoutes struct{ db *demo.DB }

func (ar *appRoutes) initCatalogRoutes(db *demo.DB) {
	cat := &catalogRoutes{db: db}
	ar.e.GET(catalogBase, cat.handleCatalogPage)
	ar.e.GET(catalogBase+"/items", cat.handleCatalogItems)
	ar.e.GET(catalogBase+"/items/:id", cat.handleCatalogDetailPage)
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
	bar, container, err := cat.buildCatalogContent(c)
	if err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to load items", err)
	}
	if htmx.IsBoosted(c.Request()) {
		return handler.RenderBaseLayout(c, views.CatalogPage(bar, container))
	}
	setTableReplaceURL(c, catalogBase)
	return handler.RenderComponent(c, container)
}

func (cat *catalogRoutes) handleCatalogDetailPage(c echo.Context) error {
	id, err := params.ParseParamID(c, "id")
	if err != nil {
		return handler.HandleHypermediaError(c, 400, "Invalid item ID", err)
	}
	item, err := cat.db.GetItem(c.Request().Context(), id)
	if err != nil {
		return handler.HandleHypermediaError(c, 404, "Item not found", err)
	}
	handler.SetPageLabel(c, item.Name)
	return handler.RenderBaseLayout(c, views.CatalogDetailPage(item))
}

func (cat *catalogRoutes) handleCatalogItemDetails(c echo.Context) error {
	id, err := params.ParseParamID(c, "id")
	if err != nil {
		return handler.HandleHypermediaError(c, 400, "Invalid item ID", err)
	}
	item, err := cat.db.GetItem(c.Request().Context(), id)
	if err != nil {
		return handler.HandleHypermediaError(c, 404, "Item not found", err)
	}
	return handler.RenderComponent(c, views.CatalogDetailContent(item))
}

func (cat *catalogRoutes) buildCatalogContent(c echo.Context) (linkwell.FilterBar, templ.Component, error) {
	const perPage = 20
	tc, err := buildTableContent(c, cat.db, parseTableParams(c, perPage),
		catalogBase+"/items", "#catalog-table-container",
		linkwell.TableCol{Label: "Details"},
	)
	if err != nil {
		return linkwell.FilterBar{}, nil, err
	}
	body := views.CatalogItemsBody(tc.Items)
	container := views.CatalogTableContainer(tc.Cols, body, tc.Info)
	return tc.Bar, container, nil
}
