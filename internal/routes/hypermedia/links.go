// setup:feature:demo
package hypermedia

import (
	"fmt"
	"strings"
	"sync"
)

// LinkRelation represents a relationship between two resources.
type LinkRelation struct {
	Rel   string // IANA link relation (e.g., "related", "collection", "up")
	Href  string // Target URL
	Title string // Human-readable label
}

// linkRegistry stores registered link relations keyed by source path.
var (
	linksMu  sync.RWMutex
	linksMap = make(map[string][]LinkRelation)
)

// Link registers a relationship from a source path to a target.
// For rel="related", the inverse is automatically registered (symmetric).
func Link(source, rel, target, title string) {
	linksMu.Lock()
	defer linksMu.Unlock()

	linksMap[source] = append(linksMap[source], LinkRelation{
		Rel:   rel,
		Href:  target,
		Title: title,
	})

	// rel="related" is symmetric — auto-create the inverse
	if rel == "related" {
		// Derive the inverse title from the source path
		// e.g., "/demo/inventory" -> "Inventory"
		inverseTitle := titleFromPath(source)
		linksMap[target] = append(linksMap[target], LinkRelation{
			Rel:   "related",
			Href:  source,
			Title: inverseTitle,
		})
	}
}

// LinksFor returns all registered link relations for a path.
// If rels is provided, only relations matching those types are returned.
func LinksFor(path string, rels ...string) []LinkRelation {
	linksMu.RLock()
	defer linksMu.RUnlock()

	all := linksMap[path]
	if len(rels) == 0 {
		result := make([]LinkRelation, len(all))
		copy(result, all)
		return result
	}

	relSet := make(map[string]bool, len(rels))
	for _, r := range rels {
		relSet[r] = true
	}

	var filtered []LinkRelation
	for _, l := range all {
		if relSet[l.Rel] {
			filtered = append(filtered, l)
		}
	}
	return filtered
}

// RelatedLinksFor returns only rel="related" links for a path,
// excluding the current path itself (for use in context bars).
func RelatedLinksFor(path string) []LinkRelation {
	links := LinksFor(path, "related")
	// Deduplicate by href (symmetric registration can create dupes)
	seen := make(map[string]bool)
	var unique []LinkRelation
	for _, l := range links {
		if l.Href == path || seen[l.Href] {
			continue
		}
		seen[l.Href] = true
		unique = append(unique, l)
	}
	return unique
}

// LinkHeader formats link relations as an RFC 8288 Link header value.
func LinkHeader(links []LinkRelation) string {
	if len(links) == 0 {
		return ""
	}
	parts := make([]string, len(links))
	for i, l := range links {
		parts[i] = fmt.Sprintf("<%s>; rel=\"%s\"; title=\"%s\"", l.Href, l.Rel, l.Title)
	}
	return strings.Join(parts, ", ")
}

// titleFromPath extracts a title from the last segment of a URL path.
// "/demo/inventory" -> "Inventory", "/admin/error-traces" -> "Error Traces"
func titleFromPath(path string) string {
	path = strings.TrimSuffix(path, "/")
	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		return path
	}
	seg := path[idx+1:]
	seg = strings.ReplaceAll(seg, "-", " ")
	// Title case
	words := strings.Fields(seg)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
