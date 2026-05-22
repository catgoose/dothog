// setup:feature:demo

package views

import (
	"sort"
)

func adminStatusBadge(status string) string {
	switch status {
	case "healthy":
		return "healthy"
	case "degraded":
		return "degraded"
	default:
		return "unknown"
	}
}

func routeMethodBadge(method string) string {
	switch method {
	case "GET":
		return "badge-info"
	case "POST":
		return "badge-success"
	case "PUT", "PATCH":
		return "badge-warning"
	case "DELETE":
		return "badge-error"
	default:
		return "badge-ghost"
	}
}

func sseCountBadge(count int) string {
	if count > 0 {
		return "badge-success"
	}
	return "badge-ghost"
}

// IntervalMs reads Intervals[section]; missing keys yield fallback.
func (d AdminPanelData) IntervalMs(section string, fallback int) int {
	if v, ok := d.Intervals[section]; ok {
		return v
	}
	return fallback
}

func sortedTopics(counts map[string]int) []string {
	topics := make([]string, 0, len(counts))
	for t := range counts {
		topics = append(topics, t)
	}
	sort.Strings(topics)
	return topics
}
