// setup:feature:demo
package views

import (
	"fmt"

	"catgoose/dothog/internal/routes/hypermedia"
)

func linksRowspan(links []hypermedia.LinkRelation) string {
	return fmt.Sprintf("%d", len(links))
}

func linksCount(links map[string][]hypermedia.LinkRelation) string {
	n := 0
	for _, v := range links {
		n += len(v)
	}
	return fmt.Sprintf("%d", n)
}

func pathsCount(links map[string][]hypermedia.LinkRelation) string {
	return fmt.Sprintf("%d", len(links))
}

func codeLinkExample() string {
	return `hypermedia.Link("/demo/inventory", "related", "/demo/people", "People")
// Result: inventory <-> people (bidirectional)`
}

func codeRingExample() string {
	return `hypermedia.Ring(
    hypermedia.Rel("/admin/health", "Health"),
    hypermedia.Rel("/admin/error-traces", "Error Traces"),
    hypermedia.Rel("/admin/sessions", "Sessions"),
    hypermedia.Rel("/admin/settings", "Control Panel"),
)
// Result: 4 pages, each links to the other 3 = 12 directed links`
}

func codeHubExample() string {
	return `hypermedia.Hub("/dashboard", "Dashboard",
    hypermedia.Rel("/demo/inventory", "Inventory"),
    hypermedia.Rel("/demo/people", "People"),
    hypermedia.Rel("/demo/kanban", "Kanban"),
)
// Result: dashboard -> inventory, people, kanban
//         inventory -> dashboard (only)
//         people    -> dashboard (only)
//         kanban    -> dashboard (only)`
}

func codeRelExample() string {
	return `type RelEntry struct {
    Path  string
    Title string
}

func Rel(path, title string) RelEntry {
    return RelEntry{Path: path, Title: title}
}`
}

func codeDedupExample() string {
	return `// These two calls share /admin/health and /admin/error-traces.
// hasLink prevents duplicate links on the shared pages.
hypermedia.Ring(
    hypermedia.Rel("/admin/health", "Health"),
    hypermedia.Rel("/admin/error-traces", "Error Traces"),
    hypermedia.Rel("/admin/sessions", "Sessions"),
)
hypermedia.Ring(
    hypermedia.Rel("/admin/system", "System"),
    hypermedia.Rel("/admin/health", "Health"),
    hypermedia.Rel("/admin/error-traces", "Error Traces"),
)`
}
