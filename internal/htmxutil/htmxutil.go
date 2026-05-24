// Package htmxutil is the repo's local HTMX transport seam, backed by
// github.com/angelofallars/htmx-go. It exposes only the request-detection and
// response-header mechanics dothog has chosen to standardize on; application
// semantics (banner/page/document surfaces, report-issue meaning, business
// workflow) intentionally stay out so the seam is a thin wire-protocol layer,
// not a framework abstraction.
package htmxutil

import (
	"net/http"
	"net/url"

	htmx "github.com/angelofallars/htmx-go"
)

// IsHTMX reports whether the request originated from htmx (HX-Request: true).
func IsHTMX(r *http.Request) bool { return htmx.IsHTMX(r) }

// IsBoosted reports whether the request was issued via hx-boost rather than a
// targeted swap. Boosted requests want full-page semantics; they shouldn't be
// served as fragments.
func IsBoosted(r *http.Request) bool { return htmx.IsBoosted(r) }

// CurrentURL parses the HX-Current-URL header that htmx attaches to every
// request. Returns nil + false when the header is missing or malformed so the
// caller can short-circuit without panicking on partial state.
func CurrentURL(r *http.Request) (*url.URL, bool) {
	raw, ok := htmx.GetCurrentURL(r)
	if !ok || raw == "" {
		return nil, false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, false
	}
	return u, true
}

// CurrentRawQuery returns just the query string portion of HX-Current-URL.
// Convenient for the common "preserve table filter params across mutation
// requests" pattern.
func CurrentRawQuery(r *http.Request) string {
	u, ok := CurrentURL(r)
	if !ok {
		return ""
	}
	return u.RawQuery
}

// Response is the local builder seam for HTMX response headers. Wraps an
// htmx-go Response so future callers can't reach for surface area that isn't
// part of dothog's chosen vocabulary.
type Response struct {
	r htmx.Response
}

// New starts a Response builder. Chain helpers, then call Write.
func New() *Response { return &Response{r: htmx.NewResponse()} }

// ReplaceURL adds HX-Replace-Url so the browser address bar updates in-place
// without a navigation.
func (r *Response) ReplaceURL(replaceURL string) *Response {
	r.r = r.r.ReplaceURL(replaceURL)
	return r
}

// Refresh adds HX-Refresh: true so the client performs a full reload of the
// current page once the response lands.
func (r *Response) Refresh() *Response {
	r.r = r.r.Refresh(true)
	return r
}

// Trigger adds HX-Trigger with a plain event name (no detail payload).
func (r *Response) Trigger(name string) *Response {
	r.r = r.r.AddTrigger(htmx.Trigger(name))
	return r
}

// TriggerDetail adds HX-Trigger with one named event whose detail payload is
// marshaled to JSON. Use for events that want a value attached (alert text,
// preset params, etc.).
func (r *Response) TriggerDetail(name string, detail any) *Response {
	r.r = r.r.AddTrigger(htmx.TriggerObject(name, detail))
	return r
}

// ReswapNone adds HX-Reswap: none so htmx skips the default swap. Use for
// "the response is a side effect, don't touch the DOM" flows.
func (r *Response) ReswapNone() *Response {
	r.r = r.r.Reswap(htmx.SwapNone)
	return r
}

// Write flushes the accumulated headers onto w.
func (r *Response) Write(w http.ResponseWriter) error { return r.r.Write(w) }
