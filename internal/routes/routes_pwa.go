// setup:feature:demo

package routes

import (
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/web/views"

	"github.com/labstack/echo/v4"
)

func (ar *appRoutes) initPwaRoutes() {
	ar.e.GET("/pwa", handler.HandleComponent(views.PwaIndexPage()))
	ar.e.GET("/pwa/inspection", handler.HandleComponent(views.PwaSiteInspectionForm()))
	ar.e.GET("/pwa/report", handler.HandleComponent(views.PwaFieldReportForm()))
	ar.e.GET("/pwa/notes", handler.HandleComponent(views.PwaNotesForm()))

	ar.e.POST("/pwa/inspection", func(c echo.Context) error {
		return handler.RenderComponent(c, views.PwaFormSuccess("Site inspection saved."))
	})
	ar.e.POST("/pwa/report", func(c echo.Context) error {
		return handler.RenderComponent(c, views.PwaFormSuccess("Field report saved."))
	})
	ar.e.POST("/pwa/notes", func(c echo.Context) error {
		return handler.RenderComponent(c, views.PwaFormSuccess("Notes saved."))
	})
}
