package handler

import (
	corecomponents "catgoose/dothog/web/components/core"

	"github.com/labstack/echo/v4"
)

// RenderErrorSurface renders an ErrorPresentation through the route-owned
// contract. SurfacePage composes the inner error content inside the host
// layout (so chrome — nav, breadcrumbs, theme — wraps the error). Every
// other surface (banner, inline, inline-full, document) is self-contained
// and writes straight to the response.
func RenderErrorSurface(c echo.Context, p corecomponents.ErrorPresentation) error {
	if p.Surface == corecomponents.SurfacePage {
		return RenderBaseLayout(c, corecomponents.RenderError(p))
	}
	return RenderComponent(c, corecomponents.RenderError(p))
}
