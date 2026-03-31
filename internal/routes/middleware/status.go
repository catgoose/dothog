package middleware

import (
	"net/http"

	"github.com/catgoose/flighty"
	"github.com/catgoose/linkwell"
	corecomponents "catgoose/dothog/web/components/core"

	"github.com/labstack/echo/v4"
)

// UnprocessableEntity returns a 422 builder pre-loaded with an error panel component.
// Append additional OOB swaps or triggers via the returned builder before calling Send().
func UnprocessableEntity(c echo.Context, msg string, controls ...linkwell.Control) *flighty.Builder {
	ec := HypermediaError(c, http.StatusUnprocessableEntity, msg, nil, controls...)
	return flighty.New(c).
		Status(http.StatusUnprocessableEntity).
		Component(corecomponents.ErrorStatusFromContext(ec))
}
