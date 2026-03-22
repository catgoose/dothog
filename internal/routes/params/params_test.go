package params

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newContext(path string, query string) echo.Context {
	e := echo.New()
	var url string
	if query != "" {
		url = path + "?" + query
	} else {
		url = path
	}
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec)
}

func TestParsePaginationParams_Defaults(t *testing.T) {
	c := newContext("/", "")
	pp := ParsePaginationParams(c, 10, 100)
	assert.Equal(t, 1, pp.Page)
	assert.Equal(t, 10, pp.Limit)
	assert.Equal(t, 0, pp.Offset)
}

func TestParsePaginationParams_WithPageAndLimit(t *testing.T) {
	c := newContext("/", "page=3&limit=25")
	pp := ParsePaginationParams(c, 10, 100)
	assert.Equal(t, 3, pp.Page)
	assert.Equal(t, 25, pp.Limit)
	assert.Equal(t, 50, pp.Offset)
}

func TestParsePaginationParams_InvalidPage_UsesDefault(t *testing.T) {
	c := newContext("/", "page=-1&page=abc")
	pp := ParsePaginationParams(c, 10, 100)
	assert.Equal(t, 1, pp.Page)
}

func TestParsePaginationParams_LimitOverMax_UsesDefault(t *testing.T) {
	c := newContext("/", "limit=500")
	pp := ParsePaginationParams(c, 10, 100)
	assert.Equal(t, 10, pp.Limit)
}

func TestCalculateTotalPages(t *testing.T) {
	assert.Equal(t, 0, CalculateTotalPages(0, 10))
	assert.Equal(t, 1, CalculateTotalPages(5, 10))
	assert.Equal(t, 1, CalculateTotalPages(10, 10))
	assert.Equal(t, 2, CalculateTotalPages(11, 10))
	assert.Equal(t, 4, CalculateTotalPages(40, 10))
	assert.Equal(t, 0, CalculateTotalPages(10, 0))
}

func TestParseParamID_Valid(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/users/42", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("42")

	id, err := ParseParamID(c, "id")
	require.NoError(t, err)
	assert.Equal(t, 42, id)
}

func TestParseParamID_NotFound(t *testing.T) {
	c := newContext("/users", "")
	_, err := ParseParamID(c, "id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "id parameter not found")
}

func TestParseParamID_Invalid(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/users/abc", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("abc")

	_, err := ParseParamID(c, "id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid id")
}

func TestParseParamID_Zero(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/users/0", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("0")

	_, err := ParseParamID(c, "id")
	require.Error(t, err)
}

func TestParseParamID_Negative(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/users/-1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("-1")

	_, err := ParseParamID(c, "id")
	require.Error(t, err)
}

func TestParseFilterParams(t *testing.T) {
	c := newContext("/", "search=foo&status=active&type=short&sort=name:asc&page=2&limit=20&year=2024")
	fp := ParseFilterParams(c, 10, 100)
	assert.Equal(t, "foo", fp.Search)
	assert.Equal(t, "active", fp.Status)
	assert.Equal(t, "short", fp.DurationType)
	assert.Equal(t, "name:asc", fp.Sort)
	assert.Equal(t, 2024, fp.Year)
	assert.Equal(t, 2, fp.Pagination.Page)
	assert.Equal(t, 20, fp.Pagination.Limit)
	assert.Equal(t, 20, fp.Pagination.Offset)
}

func TestResolveYearWithDefault_YearParamExists(t *testing.T) {
	c := newContext("/", "year=2023")
	year := ResolveYearWithDefault(c, 2023, []int{2022, 2023, 2024})
	assert.Equal(t, 2023, year)
}

func TestResolveYearWithDefault_NoParam_UsesFirstAvailable(t *testing.T) {
	c := newContext("/", "")
	year := ResolveYearWithDefault(c, 0, []int{2022, 2023, 2024})
	assert.Equal(t, 2022, year)
}

func TestResolveYearWithDefault_YearProvided(t *testing.T) {
	c := newContext("/", "")
	year := ResolveYearWithDefault(c, 2024, []int{2022, 2023, 2024})
	assert.Equal(t, 2024, year)
}

func TestResolveYearWithDefault_EmptyAvailable(t *testing.T) {
	c := newContext("/", "")
	year := ResolveYearWithDefault(c, 0, nil)
	assert.Equal(t, 0, year)
}
