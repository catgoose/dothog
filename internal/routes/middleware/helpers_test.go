package middleware

import (
	"net/http/httptest"

	"github.com/labstack/echo/v4"
)

// newTestContext returns a fresh echo.Context wired to a recorder for the
// remaining middleware unit tests (links, server-timing).
func newTestContext(method, path string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	return c, rec
}
