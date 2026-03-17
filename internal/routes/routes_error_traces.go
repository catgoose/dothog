package routes

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"catgoose/dothog/internal/requestlog"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/internal/routes/hypermedia"
	"catgoose/dothog/internal/routes/response"
	"catgoose/dothog/web/views"

	hx "catgoose/dothog/internal/routes/htmx"
	corecomponents "catgoose/dothog/web/components/core"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
)

const errorTracesBase = "/admin/error-traces"

func (ar *appRoutes) initErrorTracesRoutes() {
	if ar.reqLogStore == nil {
		return
	}
	ar.e.GET(errorTracesBase, ar.handleErrorTracesPage)
	ar.e.GET(errorTracesBase+"/list", ar.handleErrorTracesList)
	ar.e.GET(errorTracesBase+"/:requestID", ar.handleErrorTraceDetail)
	ar.e.DELETE(errorTracesBase+"/:requestID", ar.handleErrorTraceDelete)
}

func (ar *appRoutes) handleErrorTracesPage(c echo.Context) error {
	group, container, err := ar.buildErrorTracesContent(c)
	if err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to load error traces", err)
	}
	return handler.RenderBaseLayout(c, views.ErrorTracesPage(group.Bar, container))
}

func (ar *appRoutes) handleErrorTracesList(c echo.Context) error {
	group, container, err := ar.buildErrorTracesContent(c)
	if err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to load error traces", err)
	}
	b := response.New(c).
		Component(container).
		OOB(corecomponents.FilterGroupOOB(group))
	if hx.IsHTMX(c) {
		pushURL := errorTracesBase
		if q := c.Request().URL.RawQuery; q != "" {
			pushURL += "?" + q
		}
		hx.ReplaceURL(c, pushURL)
	}
	return b.Send()
}

func (ar *appRoutes) handleErrorTraceDetail(c echo.Context) error {
	requestID := c.Param("requestID")
	trace := ar.reqLogStore.Get(requestID)
	if trace == nil {
		return handler.HandleHypermediaError(c, 404, "Error trace not found", nil)
	}
	return handler.RenderComponent(c, views.ErrorTraceDetailContent(trace))
}

func (ar *appRoutes) handleErrorTraceDelete(c echo.Context) error {
	requestID := c.Param("requestID")
	if err := ar.reqLogStore.DeleteTrace(requestID); err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to delete trace", err)
	}
	// Re-apply current filters from HX-Current-URL
	if raw := c.Request().Header.Get("HX-Current-URL"); raw != "" {
		if u, err := url.Parse(raw); err == nil && u.RawQuery != "" {
			c.Request().URL.RawQuery = u.RawQuery
		}
	}
	group, container, err := ar.buildErrorTracesContent(c)
	if err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to reload traces", err)
	}
	return response.New(c).
		Component(container).
		OOB(corecomponents.FilterGroupOOB(group)).
		Send()
}

func (ar *appRoutes) buildErrorTracesContent(c echo.Context) (hypermedia.FilterGroup, templ.Component, error) {
	const perPage = 20
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	f := requestlog.TraceFilter{
		Q:       c.QueryParam("q"),
		Status:  c.QueryParam("status"),
		Method:  c.QueryParam("method"),
		Sort:    c.QueryParam("sort"),
		Dir:     c.QueryParam("dir"),
		Page:    page,
		PerPage: perPage,
	}

	traces, total, err := ar.reqLogStore.ListTraces(f)
	if err != nil {
		return hypermedia.FilterGroup{}, nil, err
	}

	avail, err := ar.reqLogStore.AvailableFilters(f)
	if err != nil {
		return hypermedia.FilterGroup{}, nil, err
	}

	target := "#error-traces-table-container"
	listURL := errorTracesBase + "/list"

	// Build status options from available codes, grouping into 4xx/5xx ranges.
	statusPairs := []string{"", "All"}
	has4xx, has5xx := false, false
	for _, code := range avail.StatusCodes {
		if code >= 400 && code < 500 {
			has4xx = true
		}
		if code >= 500 {
			has5xx = true
		}
	}
	if has4xx {
		statusPairs = append(statusPairs, "4xx", "4xx Client")
	}
	if has5xx {
		statusPairs = append(statusPairs, "5xx", "5xx Server")
	}
	for _, code := range avail.StatusCodes {
		s := strconv.Itoa(code)
		statusPairs = append(statusPairs, s, s)
	}

	// Build method options from available methods.
	methodPairs := []string{"", "All"}
	for _, m := range avail.Methods {
		methodPairs = append(methodPairs, m, m)
	}

	group := hypermedia.NewFilterGroup(listURL, target,
		hypermedia.SearchField("q", "Search routes, errors, IDs, users, IPs\u2026", f.Q),
		hypermedia.SelectField("status", "Status", f.Status,
			hypermedia.SelectOptions(f.Status, statusPairs...)),
		hypermedia.SelectField("method", "Method", f.Method,
			hypermedia.SelectOptions(f.Method, methodPairs...)),
	)

	sortBase := traceStripParams(c.Request().URL, "sort", "dir")
	cols := []hypermedia.TableCol{
		{Label: "", Width: "2rem"},
		hypermedia.SortableCol("CreatedAt", "Time", f.Sort, f.Dir, sortBase, target, "#filter-form"),
		hypermedia.SortableCol("StatusCode", "Status", f.Sort, f.Dir, sortBase, target, "#filter-form"),
		hypermedia.SortableCol("Method", "Method", f.Sort, f.Dir, sortBase, target, "#filter-form"),
		hypermedia.SortableCol("Route", "Route", f.Sort, f.Dir, sortBase, target, "#filter-form"),
		{Label: "Error"},
		{Label: "IP"},
	}

	pageBase := traceStripParams(c.Request().URL, "page")
	info := hypermedia.PageInfo{
		Page:       page,
		PerPage:    perPage,
		TotalItems: total,
		TotalPages: hypermedia.ComputeTotalPages(total, perPage),
		BaseURL:    pageBase,
		Target:     target,
		Include:    "#filter-form",
	}

	body := views.ErrorTracesBody(traces)
	container := views.ErrorTracesTableContainer(cols, body, info)
	return group, container, nil
}

// traceStripParams returns a copy of u with the named query params removed.
func traceStripParams(u *url.URL, params ...string) string {
	cp := *u
	q := cp.Query()
	for _, p := range params {
		q.Del(p)
	}
	cp.RawQuery = q.Encode()
	return cp.String()
}

// SeedErrorTraces inserts 1000 demo error traces spread over the past 3 years.
func SeedErrorTraces(store *requestlog.Store) {
	// Check if already seeded
	existing, _, _ := store.ListTraces(requestlog.TraceFilter{Page: 1, PerPage: 1})
	if len(existing) > 0 {
		return
	}

	type errorTemplate struct {
		ErrorChain string
		StatusCode int
		Route      string
		Method     string
		Entries    []requestlog.Entry
	}

	templates := []errorTemplate{
		{
			ErrorChain: "get user {id}: sql: no rows in result set",
			StatusCode: 404, Route: "/api/users/{id}", Method: "GET",
			Entries: []requestlog.Entry{
				{Level: "INFO", Message: "Request started", Attrs: "method=GET path=/api/users/{id}"},
				{Level: "INFO", Message: "Querying user by ID", Attrs: "table=users query=SELECT * FROM users WHERE id=? params={id} duration_ms=3"},
				{Level: "INFO", Message: "Database query completed", Attrs: "table=users rows_returned=0 duration_ms=3"},
				{Level: "ERROR", Message: "User not found", Attrs: "user_id={id} error=sql: no rows in result set"},
			},
		},
		{
			ErrorChain: "process order: validate inventory: insufficient stock for SKU-{id}",
			StatusCode: 422, Route: "/api/orders/{id}", Method: "POST",
			Entries: []requestlog.Entry{
				{Level: "INFO", Message: "Request started", Attrs: "method=POST path=/api/orders"},
				{Level: "INFO", Message: "Parsing order payload", Attrs: "content_type=application/json items=3 total_cents=14995"},
				{Level: "INFO", Message: "Validating inventory", Attrs: "sku=SKU-{id} requested_qty=10 warehouse=us-east-1"},
				{Level: "WARN", Message: "Low stock detected", Attrs: "sku=SKU-{id} available=2 requested=10 deficit=8"},
				{Level: "ERROR", Message: "Insufficient stock", Attrs: "sku=SKU-{id} available=2 requested=10 order_id=ORD-{id}"},
			},
		},
		{
			ErrorChain: "render dashboard: query metrics: context deadline exceeded",
			StatusCode: 504, Route: "/dashboard", Method: "GET",
			Entries: []requestlog.Entry{
				{Level: "INFO", Message: "Request started", Attrs: "method=GET path=/dashboard"},
				{Level: "INFO", Message: "Fetching metrics", Attrs: "range=7d source=prometheus endpoint=http://metrics:9090/api/v1/query_range"},
				{Level: "INFO", Message: "Query executing", Attrs: "query=rate(http_requests_total[5m]) elapsed_ms=2100"},
				{Level: "WARN", Message: "Query slow", Attrs: "elapsed_ms=4500 timeout=5000 query=rate(http_requests_total[5m])"},
				{Level: "ERROR", Message: "Context deadline exceeded", Attrs: "timeout=5s elapsed_ms=5001 source=prometheus"},
			},
		},
		{
			ErrorChain: "upload file: multipart: NextPart: unexpected EOF",
			StatusCode: 400, Route: "/api/files/upload", Method: "POST",
			Entries: []requestlog.Entry{
				{Level: "INFO", Message: "Request started", Attrs: "method=POST path=/api/files/upload content_length=1048576"},
				{Level: "INFO", Message: "Parsing multipart form", Attrs: "content_type=multipart/form-data boundary=----WebKitFormBoundary{hex} max_size=10485760"},
				{Level: "ERROR", Message: "Multipart parse failed", Attrs: "error=unexpected EOF bytes_read=524288 expected=1048576"},
			},
		},
		{
			ErrorChain: "save settings: database is locked",
			StatusCode: 500, Route: "/settings/theme", Method: "POST",
			Entries: []requestlog.Entry{
				{Level: "INFO", Message: "Request started", Attrs: "method=POST path=/settings/theme"},
				{Level: "INFO", Message: "Updating theme", Attrs: "theme=dark session=sess-{hex} table=SessionSettings"},
				{Level: "INFO", Message: "Acquiring write lock", Attrs: "table=SessionSettings timeout_ms=30000"},
				{Level: "ERROR", Message: "Database write failed", Attrs: "error=database is locked table=SessionSettings busy_timeout_ms=30000 retries=3"},
			},
		},
		{
			ErrorChain: "fetch report: connect: connection refused",
			StatusCode: 502, Route: "/api/reports/monthly", Method: "GET",
			Entries: []requestlog.Entry{
				{Level: "INFO", Message: "Request started", Attrs: "method=GET path=/api/reports/monthly"},
				{Level: "INFO", Message: "Calling reporting service", Attrs: "url=http://reports-svc:8080/v2/monthly timeout=10s"},
				{Level: "INFO", Message: "DNS resolved", Attrs: "host=reports-svc addr=10.0.5.42 duration_ms=2"},
				{Level: "ERROR", Message: "Upstream connection refused", Attrs: "host=reports-svc:8080 addr=10.0.5.42 error=connection refused dial_timeout=5s"},
			},
		},
		{
			ErrorChain: "authenticate: token expired",
			StatusCode: 401, Route: "/api/protected/data", Method: "GET",
			Entries: []requestlog.Entry{
				{Level: "INFO", Message: "Request started", Attrs: "method=GET path=/api/protected/data"},
				{Level: "INFO", Message: "Extracting bearer token", Attrs: "header=Authorization scheme=Bearer"},
				{Level: "WARN", Message: "Token validation failed", Attrs: "reason=expired exp=2026-03-13T23:59:59Z now=2026-03-14T00:01:12Z issuer=login.microsoftonline.com"},
				{Level: "ERROR", Message: "Authentication failed", Attrs: "error=token expired sub=usr-{id} aud=api://dothog"},
			},
		},
		{
			ErrorChain: "create item: UNIQUE constraint failed: items.name",
			StatusCode: 409, Route: "/demo/inventory/items", Method: "POST",
			Entries: []requestlog.Entry{
				{Level: "INFO", Message: "Request started", Attrs: "method=POST path=/demo/inventory/items"},
				{Level: "INFO", Message: "Parsing item form", Attrs: "name=Widget-{id} category=Electronics price=29.99"},
				{Level: "INFO", Message: "Creating item", Attrs: "table=items name=Widget-{id} category=Electronics"},
				{Level: "ERROR", Message: "Insert failed", Attrs: "error=UNIQUE constraint failed: items.name table=items name=Widget-{id} constraint=idx_items_name"},
			},
		},
		{
			ErrorChain: "authorize /admin/settings: role viewer cannot access admin resource",
			StatusCode: 403, Route: "/admin/settings", Method: "GET",
			Entries: []requestlog.Entry{
				{Level: "INFO", Message: "Request started", Attrs: "method=GET path=/admin/settings"},
				{Level: "INFO", Message: "Authenticating user", Attrs: "session_id=sess-{hex} method=bearer_token"},
				{Level: "INFO", Message: "User authenticated", Attrs: "user_id=usr-{id} email=user{id}@example.com roles=[viewer]"},
				{Level: "INFO", Message: "Checking authorization", Attrs: "resource=/admin/settings required_role=admin user_roles=[viewer]"},
				{Level: "WARN", Message: "Authorization denied", Attrs: "user_id=usr-{id} resource=/admin/settings reason=insufficient_role required=admin actual=viewer"},
			},
		},
		{
			ErrorChain: "update item {id}: optimistic lock: version mismatch",
			StatusCode: 409, Route: "/api/items/{id}", Method: "PUT",
			Entries: []requestlog.Entry{
				{Level: "INFO", Message: "Request started", Attrs: "method=PUT path=/api/items/{id}"},
				{Level: "INFO", Message: "Loading item for update", Attrs: "item_id={id} table=items"},
				{Level: "INFO", Message: "Comparing versions", Attrs: "item_id={id} client_version=3 server_version=4"},
				{Level: "ERROR", Message: "Version conflict detected", Attrs: "item_id={id} client_version=3 server_version=4 table=items"},
			},
		},
		{
			ErrorChain: "delete user {id}: foreign key constraint: user has active orders",
			StatusCode: 409, Route: "/api/users/{id}", Method: "DELETE",
			Entries: []requestlog.Entry{
				{Level: "INFO", Message: "Request started", Attrs: "method=DELETE path=/api/users/{id}"},
				{Level: "INFO", Message: "Checking user dependencies", Attrs: "user_id={id} tables=[orders,sessions,audit_log]"},
				{Level: "INFO", Message: "Found active references", Attrs: "user_id={id} orders=5 sessions=1 audit_entries=142"},
				{Level: "ERROR", Message: "Cannot delete: active orders exist", Attrs: "user_id={id} active_orders=5 constraint=fk_orders_user_id"},
			},
		},
		{
			ErrorChain: "parse JSON body: unexpected end of JSON input",
			StatusCode: 400, Route: "/api/webhooks/stripe", Method: "POST",
			Entries: []requestlog.Entry{
				{Level: "INFO", Message: "Request started", Attrs: "method=POST path=/api/webhooks/stripe"},
				{Level: "INFO", Message: "Receiving webhook", Attrs: "source=stripe event_type=payment_intent.succeeded content_length=2048"},
				{Level: "INFO", Message: "Verifying signature", Attrs: "header=Stripe-Signature algo=hmac-sha256"},
				{Level: "ERROR", Message: "Malformed JSON body", Attrs: "error=unexpected end of JSON input bytes_read=1024 content_length=2048"},
			},
		},
	}

	users := []string{
		"alice@example.com", "bob@example.com", "charlie@example.com",
		"dana@example.com", "eve@example.com", "frank@example.com",
		"grace@example.com", "hank@example.com", "",
	}
	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
		"Mozilla/5.0 (X11; Linux x86_64; rv:109.0)",
		"curl/8.1.2",
		"PostmanRuntime/7.32.3",
	}
	ips := []string{
		"192.168.1.100", "10.0.0.50", "172.16.0.25", "192.168.1.55",
		"10.0.0.12", "172.16.0.30", "192.168.1.200", "10.0.0.5",
	}

	now := time.Now()
	threeYears := 3 * 365 * 24 * time.Hour
	const count = 1000

	for i := 0; i < count; i++ {
		// Spread evenly over 3 years
		offset := threeYears * time.Duration(i) / count
		createdAt := now.Add(-threeYears + offset)

		tmpl := templates[i%len(templates)]
		id := i + 1

		idStr := strconv.Itoa(id)
		idHex := fmt.Sprintf("%08x", id)
		replacer := strings.NewReplacer("{id}", idStr, "{hex}", idHex)
		errorChain := replacer.Replace(tmpl.ErrorChain)
		route := replacer.Replace(tmpl.Route)

		// Format attrs with the current id so they have realistic values
		entries := make([]requestlog.Entry, len(tmpl.Entries))
		for j, e := range tmpl.Entries {
			entries[j] = requestlog.Entry{
				Level:   e.Level,
				Message: e.Message,
				Attrs:   replacer.Replace(e.Attrs),
			}
		}

		store.PromoteAt(requestlog.ErrorTrace{
			RequestID:  fmt.Sprintf("seed-%08x", i),
			ErrorChain: errorChain,
			StatusCode: tmpl.StatusCode,
			Route:      route,
			Method:     tmpl.Method,
			UserAgent:  userAgents[i%len(userAgents)],
			RemoteIP:   ips[i%len(ips)],
			UserID:     users[i%len(users)],
			Entries:    entries,
		}, createdAt)
	}
}
