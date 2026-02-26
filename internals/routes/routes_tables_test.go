package routes

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// filterQueryFromHXCurrentURL
// ---------------------------------------------------------------------------

func TestFilterQueryFromHXCurrentURL_HeaderWithQuery(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/tables/inventory", nil)
	req.Header.Set("HX-Current-URL", "https://example.com/tables?search=foo&page=2")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	result := filterQueryFromHXCurrentURL(c)

	require.Equal(t, "search=foo&page=2", result)
}

func TestFilterQueryFromHXCurrentURL_HeaderWithoutQuery(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/tables/inventory", nil)
	req.Header.Set("HX-Current-URL", "https://example.com/tables")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	result := filterQueryFromHXCurrentURL(c)

	require.Equal(t, "", result)
}

func TestFilterQueryFromHXCurrentURL_HeaderAbsent(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/tables/inventory", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	result := filterQueryFromHXCurrentURL(c)

	require.Equal(t, "", result)
}

func TestFilterQueryFromHXCurrentURL_UnparseableURL(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/tables/inventory", nil)
	req.Header.Set("HX-Current-URL", "://not-a-url")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	result := filterQueryFromHXCurrentURL(c)

	require.Equal(t, "", result)
}

// ---------------------------------------------------------------------------
// setTableReplaceURL
// ---------------------------------------------------------------------------

func TestSetTableReplaceURL_HTMXRequestWithQuery(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/tables/inventory?search=foo&page=2", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setTableReplaceURL(c, "/tables/inventory")

	require.Equal(t, "/tables/inventory?search=foo&page=2", rec.Header().Get("HX-Replace-Url"))
}

func TestSetTableReplaceURL_HTMXRequestWithoutQuery(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/tables/inventory", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setTableReplaceURL(c, "/tables/inventory")

	require.Equal(t, "/tables/inventory", rec.Header().Get("HX-Replace-Url"))
}

func TestSetTableReplaceURL_NonHTMXRequest(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/tables/inventory?search=foo", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setTableReplaceURL(c, "/tables/inventory")

	require.Empty(t, rec.Header().Get("HX-Replace-Url"))
}

// ---------------------------------------------------------------------------
// applyFilterFromCurrentURL
// ---------------------------------------------------------------------------

func TestApplyFilterFromCurrentURL_HasQuery(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/tables/inventory/1", nil)
	req.Header.Set("HX-Current-URL", "https://example.com/tables/inventory?search=widget&page=3")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	applyFilterFromCurrentURL(c)

	require.Equal(t, "search=widget&page=3", c.Request().URL.RawQuery)
}

func TestApplyFilterFromCurrentURL_NoHeader(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/tables/inventory/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	originalQuery := c.Request().URL.RawQuery

	applyFilterFromCurrentURL(c)

	require.Equal(t, originalQuery, c.Request().URL.RawQuery)
}

// ---------------------------------------------------------------------------
// stripParams
// ---------------------------------------------------------------------------

func TestStripParams_RemovesSingleParam(t *testing.T) {
	u, _ := url.Parse("https://example.com/tables?a=1&b=2&c=3")

	result := stripParams(u, "b")

	parsed, err := url.Parse(result)
	require.NoError(t, err)
	require.Equal(t, "1", parsed.Query().Get("a"))
	require.Empty(t, parsed.Query().Get("b"))
	require.Equal(t, "3", parsed.Query().Get("c"))
}

func TestStripParams_RemovesMultipleParams(t *testing.T) {
	u, _ := url.Parse("https://example.com/tables?a=1&b=2&c=3")

	result := stripParams(u, "a", "c")

	parsed, err := url.Parse(result)
	require.NoError(t, err)
	require.Empty(t, parsed.Query().Get("a"))
	require.Equal(t, "2", parsed.Query().Get("b"))
	require.Empty(t, parsed.Query().Get("c"))
}

func TestStripParams_NoParamsToStrip(t *testing.T) {
	u, _ := url.Parse("https://example.com/tables?a=1&b=2&c=3")

	result := stripParams(u, "x")

	parsed, err := url.Parse(result)
	require.NoError(t, err)
	require.Equal(t, "1", parsed.Query().Get("a"))
	require.Equal(t, "2", parsed.Query().Get("b"))
	require.Equal(t, "3", parsed.Query().Get("c"))
}

func TestStripParams_AllParamsStripped(t *testing.T) {
	u, _ := url.Parse("https://example.com/tables?a=1&b=2&c=3")

	result := stripParams(u, "a", "b", "c")

	parsed, err := url.Parse(result)
	require.NoError(t, err)
	require.Empty(t, parsed.RawQuery)
}
