package views

func healthBadgeClass(status string) string {
	switch status {
	case "healthy":
		return "badge-success"
	case "degraded":
		return "badge-warning"
	default:
		return "badge-error"
	}
}

func dbBadgeClass(status string) string {
	if status == "connected" {
		return "badge-success"
	}
	return "badge-error"
}

func intervalOr(m map[string]int, key string, fallback int) int {
	if v, ok := m[key]; ok {
		return v
	}
	return fallback
}
