package routes

// SSE topic constants for the broker. These are app-specific channel names
// used by the real-time features (dashboard, canvas, feed, etc.).
const (
	TopicSystemStats  = "system-stats"
	TopicDashMetrics  = "dashboard-metrics"
	TopicDashServices = "dashboard-services"
	TopicDashEvents   = "dashboard-events"
	TopicPeopleUpdate = "people-update"
	TopicActivityFeed = "activity-feed"
	TopicErrorTraces  = "error-traces"
	TopicThemeChange  = "theme-change"
	TopicCanvasUpdate  = "canvas-update"
	TopicAdminPanel    = "admin-panel"
	TopicNumericalDash = "numerical-dash"
)
