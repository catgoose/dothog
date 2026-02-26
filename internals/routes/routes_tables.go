// setup:feature:demo
package routes

import (
	"net/url"

	hx "catgoose/go-htmx-demo/internals/routes/htmx"

	"github.com/labstack/echo/v4"
)

// filterQueryFromHXCurrentURL extracts the raw query string from the HX-Current-URL
// header that HTMX sends on every request. Returns "" if the header is absent or unparseable.
func filterQueryFromHXCurrentURL(c echo.Context) string {
	raw := c.Request().Header.Get("HX-Current-URL")
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return u.RawQuery
}

// setTableReplaceURL sets HX-Replace-Url to basePath?{currentQueryString} so the browser
// URL stays in sync with the active filters after any table-replacing response.
func setTableReplaceURL(c echo.Context, basePath string) {
	if !hx.IsHTMX(c) {
		return
	}
	pushURL := basePath
	if q := c.Request().URL.RawQuery; q != "" {
		pushURL += "?" + q
	}
	hx.ReplaceURL(c, pushURL)
}

// applyFilterFromCurrentURL reads HX-Current-URL and sets the request URL's query string
// so that buildXxxContent(c) can read filter params via c.QueryParam() on mutation requests
// (DELETE, PUT, POST) where no query params are present in the request URL.
func applyFilterFromCurrentURL(c echo.Context) {
	if rawQuery := filterQueryFromHXCurrentURL(c); rawQuery != "" {
		c.Request().URL.RawQuery = rawQuery
	}
}

// stripParams returns a copy of u with the named query params removed.
func stripParams(u *url.URL, params ...string) string {
	cp := *u
	q := cp.Query()
	for _, p := range params {
		q.Del(p)
	}
	cp.RawQuery = q.Encode()
	return cp.String()
}
