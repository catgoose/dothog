// setup:feature:demo

package routes

import "net/http"

// dynamicGroupFromCookie returns a `broker.DynamicGroup` callback that reads
// the named cookie from the incoming request and runs the cookie value
// through fn to produce the per-subscriber topic list. When the cookie is
// missing or empty the callback returns defaultTopics, so callers can
// either fall back to nil (no subscription) or to a baseline group.
//
// Used by demo routes whose dynamic group membership is driven by a
// per-browser cookie (auction watch list, subscription group switcher).
func dynamicGroupFromCookie(name string, defaultTopics []string, fn func(value string) []string) func(r *http.Request) []string {
	return func(r *http.Request) []string {
		cookie, err := r.Cookie(name)
		if err != nil || cookie.Value == "" {
			return defaultTopics
		}
		return fn(cookie.Value)
	}
}
