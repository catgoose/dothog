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
	for _, size := range []string{"xs", "sm", "md", "lg", "xl", "2xl", "3xl"} {
		size := size // capture
		ar.e.GET(base+"/inline-full/"+size, func(c echo.Context) error {
			if c.QueryParam("retry") == "true" {
				return handler.RenderComponent(c, views.ErrorModesInlineFullRetryResult(size))
			}
			return handler.RenderComponent(c, views.ErrorModesInlineFullResult(size))
		})
	}

	// Unified contract demo triggers — all use RenderError() with different surfaces.
	ar.e.GET(base+"/contract/banner", func(c echo.Context) error {
		demos := views.ContractDemoPresentations()
		return handler.RenderComponent(c, views.ErrorModesContractResult(demos[0]))
	})
	ar.e.GET(base+"/contract/inline", func(c echo.Context) error {
		demos := views.ContractDemoPresentations()
		return handler.RenderComponent(c, views.ErrorModesContractResult(demos[1]))
	})
	ar.e.GET(base+"/contract/inline-full", func(c echo.Context) error {
		demos := views.ContractDemoPresentations()
		return handler.RenderComponent(c, views.ErrorModesContractResult(demos[2]))
	})
	ar.e.GET(base+"/contract/full-page", func(c echo.Context) error {
		theme := session.GetSettings(c.Request()).Theme
		p := views.ContractFullPagePresentation(theme)
		return handler.RenderComponent(c, views.ErrorModesContractFullPage(p))
	})
}
