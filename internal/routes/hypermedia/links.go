// setup:feature:demo
package hypermedia

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// LinkRelation represents a relationship between two resources.
type LinkRelation struct {
	Rel   string // IANA link relation (e.g., "related", "collection", "up")
	Href  string // Target URL
	Title string // Human-readable label
	Group string // Optional group name (e.g., ring name) for UI grouping
}

// linkRegistry stores registered link relations keyed by source path.
var (
	linksMu  sync.RWMutex
	linksMap = make(map[string][]LinkRelation)

	hubsMu  sync.RWMutex
	hubsMap = make(map[string]string) // path -> title
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
		inverseTitle := TitleFromPath(source)
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

// RelEntry is a path+title pair for use with Ring and Hub.
type RelEntry struct {
	Path  string
	Title string
}

// Rel creates a RelEntry for use with Ring and Hub.
func Rel(path, title string) RelEntry {
	return RelEntry{Path: path, Title: title}
}

// Ring registers symmetric rel="related" links between all members.
// Every member links to every other member. The name is used for UI grouping.
func Ring(name string, members ...RelEntry) {
	linksMu.Lock()
	defer linksMu.Unlock()

	for i, a := range members {
		for j, b := range members {
			if i == j {
				continue
			}
			if !hasLink(linksMap[a.Path], b.Path, "related") {
				linksMap[a.Path] = append(linksMap[a.Path], LinkRelation{
					Rel:   "related",
					Href:  b.Path,
					Title: b.Title,
					Group: name,
				})
			}
		}
	}
}

// Hub registers a center page that links to all spokes, and each spoke
// links back to the center only. Spokes do not link to each other.
// The centerTitle is used as the group name for all links.
func Hub(centerPath, centerTitle string, spokes ...RelEntry) {
	hubsMu.Lock()
	hubsMap[centerPath] = centerTitle
	hubsMu.Unlock()

	linksMu.Lock()
	defer linksMu.Unlock()

	for _, spoke := range spokes {
		// Center -> spoke
		if !hasLink(linksMap[centerPath], spoke.Path, "related") {
			linksMap[centerPath] = append(linksMap[centerPath], LinkRelation{
				Rel:   "related",
				Href:  spoke.Path,
				Title: spoke.Title,
				Group: centerTitle,
			})
		}
		// Spoke -> center (uses rel="up" to indicate parent)
		if !hasLink(linksMap[spoke.Path], centerPath, "up") {
			linksMap[spoke.Path] = append(linksMap[spoke.Path], LinkRelation{
				Rel:   "up",
				Href:  centerPath,
				Title: centerTitle,
				Group: centerTitle,
			})
		}
	}
}

// hasLink checks if a link with the given href and rel already exists.
func hasLink(links []LinkRelation, href, rel string) bool {
	for _, l := range links {
		if l.Href == href && l.Rel == rel {
			return true
		}
	}
	return false
}

// AllLinks returns all registered link relations grouped by source path.
// Used for admin/debug inspection.
func AllLinks() map[string][]LinkRelation {
	linksMu.RLock()
	defer linksMu.RUnlock()

	result := make(map[string][]LinkRelation, len(linksMap))
	for k, v := range linksMap {
		copied := make([]LinkRelation, len(v))
		copy(copied, v)
		result[k] = copied
	}
	return result
}

// SortedPaths returns all registered source paths in sorted order.
func SortedPaths(links map[string][]LinkRelation) []string {
	paths := make([]string, 0, len(links))
	for k := range links {
		paths = append(paths, k)
	}
	sort.Strings(paths)
	return paths
}

// HubEntry represents a hub center with its spoke links for site map rendering.
type HubEntry struct {
	Path   string
	Title  string
	Spokes []LinkRelation
}

// Hubs returns all registered hub centers with their spokes, sorted by path.
// Spokes within each hub are sorted alphabetically by title.
func Hubs() []HubEntry {
	hubsMu.RLock()
	paths := make([]string, 0, len(hubsMap))
	titles := make(map[string]string, len(hubsMap))
	for p, t := range hubsMap {
		paths = append(paths, p)
		titles[p] = t
	}
	hubsMu.RUnlock()

	sort.Strings(paths)

	entries := make([]HubEntry, 0, len(paths))
	for _, p := range paths {
		spokes := LinksFor(p, "related")
		sort.Slice(spokes, func(i, j int) bool {
			return spokes[i].Title < spokes[j].Title
		})
		entries = append(entries, HubEntry{
			Path:   p,
			Title:  titles[p],
			Spokes: spokes,
		})
	}
	return entries
}

// TitleFromPath extracts a title from the last segment of a URL path.
// "/demo/inventory" -> "Inventory", "/admin/error-traces" -> "Error Traces"
func TitleFromPath(path string) string {
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

