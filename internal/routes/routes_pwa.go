// setup:feature:demo

package routes

import (
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/web/views"

	"github.com/labstack/echo/v4"
)

func (ar *AppRoutes) initPwaRoutes() {
	ar.e.GET("/platform/pwa", handler.HandleComponent(views.PwaIndexPage()))
	ar.e.GET("/platform/pwa/inspection", handler.HandleComponent(views.PwaSiteInspectionForm()))
	ar.e.GET("/platform/pwa/report", handler.HandleComponent(views.PwaFieldReportForm()))
	ar.e.GET("/platform/pwa/notes", handler.HandleComponent(views.PwaNotesForm()))
	ar.e.GET("/platform/pwa/info", handler.HandleComponent(views.PwaInfoPage()))

	ar.e.POST("/platform/pwa/inspection", func(c echo.Context) error {
		return handler.RenderComponent(c, views.PwaFormSuccess("Site inspection saved."))
	})
	ar.e.POST("/platform/pwa/report", func(c echo.Context) error {
		return handler.RenderComponent(c, views.PwaFormSuccess("Field report saved."))
	})
	ar.e.POST("/platform/pwa/notes", func(c echo.Context) error {
		return handler.RenderComponent(c, views.PwaFormSuccess("Notes saved."))
	})
}
