package views

import "github.com/a-h/templ"

// debugOnChange tags a debug toggle input with the handler key the
// admin-debug.js delegated change listener uses to route toggles. The
// attribute is data-only so the markup stays CSP-safe under
// script-src 'self'.
func debugOnChange(key string) templ.Attributes {
	return templ.Attributes{
		"data-debug-key": key,
	}
}
