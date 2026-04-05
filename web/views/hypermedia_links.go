// setup:feature:demo

package views

import (
	"fmt"
	"sort"

	"catgoose/dothog/internal/demo"
	"github.com/catgoose/linkwell"
)

func linksRowspan(links []linkwell.LinkRelation) string {
	return fmt.Sprintf("%d", len(links))
}

func linksCount(links map[string][]linkwell.LinkRelation) string {
	n := 0
	for _, v := range links {
		n += len(v)
	}
	return fmt.Sprintf("%d", n)
}

func pathsCount(links map[string][]linkwell.LinkRelation) string {
	return fmt.Sprintf("%d", len(links))
}

// storedLinkID returns the DB ID of a stored link matching source/rel/target,
// or 0 if the link is code-declared (not in the DB).
func storedLinkID(stored []demo.StoredLinkRelation, source, rel, target string) int {
	for _, s := range stored {
		if s.Source == source && s.Rel == rel && s.Target == target {
			return s.ID
		}
	}
	return 0
}

// linkDeleteURL returns the HTMX delete endpoint for a stored link.
func linkDeleteURL(id int) string {
	return fmt.Sprintf("/api/links/%d", id)
}

// sortedLinkPaths returns the keys of a link map in alphabetical order.
func sortedLinkPaths(links map[string][]linkwell.LinkRelation) []string {
	paths := make([]string, 0, len(links))
	for k := range links {
		paths = append(paths, k)
	}
	sort.Strings(paths)
	return paths
}

func codeLinkExample() string {
	return `linkwell.Link("/apps/inventory", "related", "/apps/people", "People")
// Result: inventory <-> people (bidirectional)`
}

func codeRingExample() string {
	return `linkwell.Ring(
    linkwell.Rel("/admin/health", "Health"),
    linkwell.Rel("/admin/error-traces", "Error Traces"),
    linkwell.Rel("/admin/sessions", "Sessions"),
    linkwell.Rel("/admin/settings", "Control Panel"),
)
// Result: 4 pages, each links to the other 3 = 12 directed links`
}

func codeHubExample() string {
	return `linkwell.Hub("/dashboard", "Dashboard",
    linkwell.Rel("/apps/inventory", "Inventory"),
    linkwell.Rel("/apps/people", "People"),
    linkwell.Rel("/apps/kanban", "Kanban"),
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
linkwell.Ring(
    linkwell.Rel("/admin/health", "Health"),
    linkwell.Rel("/admin/error-traces", "Error Traces"),
    linkwell.Rel("/admin/sessions", "Sessions"),
)
linkwell.Ring(
    linkwell.Rel("/admin/system", "System"),
    linkwell.Rel("/admin/health", "Health"),
    linkwell.Rel("/admin/error-traces", "Error Traces"),
)`
}
