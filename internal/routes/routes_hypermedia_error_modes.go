// setup:feature:demo

package routes

import (
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/web/views"

	"github.com/labstack/echo/v4"
)

func (ar *appRoutes) initErrorModesRoutes() {
	base := patternsBase + "/errors/modes"

	ar.e.GET(base, handler.HandleComponent(views.ErrorModesPage()))

	// Inline error demo — returns an InlineErrorPanel.
	ar.e.GET(base+"/inline", func(c echo.Context) error {
		return handler.RenderComponent(c, views.ErrorModesInlineResult())
	})

	// Full-page error demos with different action rows.
	ar.e.GET(base+"/full-page/404", func(c echo.Context) error {
		return handler.RenderComponent(c, views.ErrorModes404())
	})
	ar.e.GET(base+"/full-page/429", func(c echo.Context) error {
		return handler.RenderComponent(c, views.ErrorModes429())
	})
	ar.e.GET(base+"/full-page/500", func(c echo.Context) error {
		return handler.RenderComponent(c, views.ErrorModes500())
	})
}
