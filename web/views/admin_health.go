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
