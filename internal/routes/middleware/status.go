package middleware

import (
	"net/http"

	htmxpkg "github.com/catgoose/cheddar"
	hypermedia "github.com/catgoose/linkwell"
	"catgoose/dothog/internal/routes/response"
	corecomponents "catgoose/dothog/web/components/core"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
)

// Created returns a 201 builder. Chain PushURL, TriggerEvent, etc. as needed.
func Created(c echo.Context, cmp templ.Component) *response.Builder {
	return response.New(c).Status(http.StatusCreated).Component(cmp)
}

// Accepted returns a 202 builder.
func Accepted(c echo.Context, cmp templ.Component) *response.Builder {
	return response.New(c).Status(http.StatusAccepted).Component(cmp)
}

// NoContent sends a 204 response with HX-Reswap: none so HTMX performs no swap.
func NoContent(c echo.Context) error {
	c.Response().Header().Set(htmxpkg.HeaderReswap, string(htmxpkg.SwapNone))
	c.Response().WriteHeader(http.StatusNoContent)
	return nil
}

// SeeOther sends a 303 redirect via HX-Redirect for HTMX requests.
func SeeOther(c echo.Context, url string) error {
	c.Response().Header().Set(htmxpkg.HeaderRedirect, url)
	c.Response().WriteHeader(http.StatusSeeOther)
	return nil
}

// UnprocessableEntity returns a 422 builder pre-loaded with an error panel component.
// Append additional OOB swaps or triggers via the returned builder before calling Send().
func UnprocessableEntity(c echo.Context, msg string, controls ...hypermedia.Control) *response.Builder {
	ec := HypermediaError(c, http.StatusUnprocessableEntity, msg, nil, controls...)
	return response.New(c).
		Status(http.StatusUnprocessableEntity).
		Component(corecomponents.ErrorStatusFromContext(ec))
}
