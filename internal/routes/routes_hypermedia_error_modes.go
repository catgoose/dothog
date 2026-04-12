// setup:feature:demo

package routes

import (
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/internal/session"
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
		theme := session.GetSettings(c.Request()).Theme
		return handler.RenderComponent(c, views.ErrorModes404(theme))
	})
	ar.e.GET(base+"/full-page/429", func(c echo.Context) error {
		theme := session.GetSettings(c.Request()).Theme
		return handler.RenderComponent(c, views.ErrorModes429(theme))
	})
	ar.e.GET(base+"/full-page/500", func(c echo.Context) error {
		theme := session.GetSettings(c.Request()).Theme
		return handler.RenderComponent(c, views.ErrorModes500(theme))
	})

	// Inline-full error demo triggers — return sized InlineFullErrorPanel.
	ar.e.GET(base+"/inline-full/sm", func(c echo.Context) error {
		return handler.RenderComponent(c, views.ErrorModesInlineFullResult("sm"))
	})
	ar.e.GET(base+"/inline-full/md", func(c echo.Context) error {
		return handler.RenderComponent(c, views.ErrorModesInlineFullResult("md"))
	})
	ar.e.GET(base+"/inline-full/lg", func(c echo.Context) error {
		return handler.RenderComponent(c, views.ErrorModesInlineFullResult("lg"))
	})
}
