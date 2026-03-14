package hypermedia

import "strings"

// NavItem is a server-computed navigation affordance.
// Active state is set by the handler (or SetActiveNavItem), not by JavaScript.
type NavItem struct {
	HTMXAttrs map[string]string
	Label     string
	Href      string
	Icon      string
	Children  []NavItem
	Active    bool
}

// Breadcrumb is one segment of a breadcrumb trail.
// Href empty means this is the current page (rendered as text, not an anchor).
type Breadcrumb struct {
	Label string
	Href  string
}

// BreadcrumbLabelHome is the default label for the root breadcrumb segment.
const BreadcrumbLabelHome = "Home"

// SetActiveNavItem performs exact-match active state setting.
// A parent item is marked active if any of its children match.
func SetActiveNavItem(items []NavItem, currentPath string) []NavItem {
	result := make([]NavItem, len(items))
	for i, item := range items {
		item.Children = SetActiveNavItem(item.Children, currentPath)
		childActive := false
		for _, child := range item.Children {
			if child.Active {
				childActive = true
				break
			}
		}
		item.Active = item.Href == currentPath || childActive
		result[i] = item
	}
	return result
}

// SetActiveNavItemPrefix performs longest-prefix match active state setting.
// Use for section-level nav: /users is active when path is /users/42/edit.
// A parent item is marked active if any of its children match.
func SetActiveNavItemPrefix(items []NavItem, currentPath string) []NavItem {
	result := make([]NavItem, len(items))
	for i, item := range items {
		item.Children = SetActiveNavItemPrefix(item.Children, currentPath)
		childActive := false
		for _, child := range item.Children {
			if child.Active {
				childActive = true
				break
			}
		}
		// Require href+separator to avoid "/" matching every path.
		isActive := item.Href != "" &&
			(currentPath == item.Href || strings.HasPrefix(currentPath, item.Href+"/"))
		item.Active = isActive || childActive
		result[i] = item
	}
	return result
}

// NavItemFromControl bridges a Control to a NavItem (Label, Href, Icon, HTMXAttrs).
func NavItemFromControl(ctrl Control) NavItem {
	return NavItem{
		Label:     ctrl.Label,
		Href:      ctrl.Href,
		Icon:      string(ctrl.Icon),
		HTMXAttrs: ctrl.HxRequest.Attrs(),
	}
}

// BreadcrumbsFromPath generates crumbs from a URL path.
// labels overrides auto-generated labels by segment index (0-based, not counting the Home crumb).
// The terminal segment always has an empty Href (rendered as plain text).
func BreadcrumbsFromPath(path string, labels map[int]string) []Breadcrumb {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return []Breadcrumb{{Label: BreadcrumbLabelHome, Href: "/"}}
	}
	segments := strings.Split(trimmed, "/")
	crumbs := make([]Breadcrumb, 0, len(segments)+1)
	crumbs = append(crumbs, Breadcrumb{Label: BreadcrumbLabelHome, Href: "/"})
	for i, seg := range segments {
		label := seg
		if l, ok := labels[i]; ok {
			label = l
		}
		href := "/" + strings.Join(segments[:i+1], "/")
		if i == len(segments)-1 {
			href = "" // terminal segment has no href
		}
		crumbs = append(crumbs, Breadcrumb{Label: label, Href: href})
	}
	return crumbs
}
