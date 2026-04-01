// setup:feature:demo
package routes

import (
	"net/url"
	"strconv"

	"catgoose/dothog/internal/demo"
	"github.com/catgoose/linkwell"

	"github.com/labstack/echo/v4"
)

// tableParams holds the common parsed query parameters for table pages.
type tableParams struct {
	Q        string
	Category string
	Active   string
	Sort     string
	Dir      string
	Page     int
	PerPage  int
}

// parseTableParams extracts standard filter/sort/pagination params from the request.
func parseTableParams(c echo.Context, perPage int) tableParams {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	return tableParams{
		Q:        c.QueryParam("q"),
		Category: c.QueryParam("category"),
		Active:   c.QueryParam("active"),
		Sort:     c.QueryParam("sort"),
		Dir:      c.QueryParam("dir"),
		Page:     page,
		PerPage:  perPage,
	}
}

// tableContent holds the shared components built from tableParams.
type tableContent struct {
	Items []demo.Item
	Total int
	Bar   linkwell.FilterBar
	Cols  []linkwell.TableCol
	Info  linkwell.PageInfo
}

// buildTableContent queries the DB and builds the filter bar, sortable columns, and pagination info.
// extraCols are appended after the standard sortable columns.
func buildTableContent(c echo.Context, db *demo.DB, p tableParams, itemsURL, target string, extraCols ...linkwell.TableCol) (tableContent, error) {
	items, total, err := db.ListItems(c.Request().Context(), p.Q, p.Category, p.Active, p.Sort, p.Dir, p.Page, p.PerPage)
	if err != nil {
		return tableContent{}, err
	}

	bar := linkwell.NewFilterBar(itemsURL, target,
		linkwell.SearchField("q", "Search items\u2026", p.Q),
		linkwell.SelectField("category", "Category", p.Category,
			linkwell.SelectOptions(p.Category, itemCategoryPairs()...)),
		linkwell.CheckboxField("active", "Active only", p.Active),
	)

	sortBase := buildSortBase(c)
	cols := []linkwell.TableCol{
		linkwell.SortableCol("name", "Name", p.Sort, p.Dir, sortBase, target, "#filter-form"),
		linkwell.SortableCol("category", "Category", p.Sort, p.Dir, sortBase, target, "#filter-form"),
		linkwell.SortableCol("price", "Price", p.Sort, p.Dir, sortBase, target, "#filter-form"),
		linkwell.SortableCol("stock", "Stock", p.Sort, p.Dir, sortBase, target, "#filter-form"),
		{Label: "Status"},
	}
	cols = append(cols, extraCols...)

	info := buildPageInfo(c, p.Page, p.PerPage, total, target)

	return tableContent{Items: items, Total: total, Bar: bar, Cols: cols, Info: info}, nil
}

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
	if c.Request().Header.Get("HX-Request") != "true" {
		return
	}
	pushURL := basePath
	if q := c.Request().URL.RawQuery; q != "" {
		pushURL += "?" + q
	}
	c.Response().Header().Set("HX-Replace-Url", pushURL)
}

// applyFilterFromCurrentURL reads HX-Current-URL and sets the request URL's query string
// so that buildXxxContent(c) can read filter params via c.QueryParam() on mutation requests
// (DELETE, PUT, POST) where no query params are present in the request URL.
func applyFilterFromCurrentURL(c echo.Context) {
	if rawQuery := filterQueryFromHXCurrentURL(c); rawQuery != "" {
		c.Request().URL.RawQuery = rawQuery
	}
}

// buildPageInfo constructs a PageInfo from common pagination parameters.
func buildPageInfo(c echo.Context, page, perPage, total int, target string) linkwell.PageInfo {
	pageBase := stripParams(c.Request().URL, "page")
	return linkwell.PageInfo{
		Page:       page,
		PerPage:    perPage,
		TotalItems: total,
		TotalPages: linkwell.ComputeTotalPages(total, perPage),
		BaseURL:    pageBase,
		Target:     target,
		Include:    "#filter-form",
	}
}

// buildSortBase returns the current URL with sort/dir params stripped.
func buildSortBase(c echo.Context) string {
	return stripParams(c.Request().URL, "sort", "dir")
}

// itemCategoryPairs returns value/label pairs for the item category filter.
func itemCategoryPairs() []string {
	pairs := []string{"", "All"}
	for _, c := range demo.ItemCategories {
		pairs = append(pairs, c, c)
	}
	return pairs
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
