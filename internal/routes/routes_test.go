// setup:feature:demo
package routes

import (
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/web/views"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootRoute_ReturnsArchitecturePage(t *testing.T) {
	e := echo.New()
	e.GET("/", handler.HandleComponent(views.ArchitecturePage()))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "HATEOAS")
}
