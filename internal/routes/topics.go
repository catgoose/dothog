package routes

// SSE topic constants for the broker. These are app-specific channel names
// used by the real-time features (dashboard, canvas, feed, etc.).
const (
	TopicSystemStats  = "system-stats"
	TopicDashMetrics = "dashboard-metrics"
	TopicPeopleUpdate = "people-update"
	TopicActivityFeed = "activity-feed"
	TopicErrorTraces  = "error-traces"
	TopicThemeChange  = "theme-change"
	TopicCanvasUpdate  = "canvas-update"
	TopicAdminPanel    = "admin-panel"
	TopicNumericalDash    = "numerical-dash"
	TopicNotifications    = "notifications"
	TopicObservatory      = "observatory"

	// Tavern gallery lab topics.
	TopicTavernReplay      = "tavern/replay"
	TopicTavernBackpress   = "tavern/backpressure"
	TopicTavernPubRaw      = "tavern/pub/raw"
	TopicTavernPubDebounce = "tavern/pub/debounced"
	TopicTavernPubThrottle = "tavern/pub/throttled"
	TopicTavernPubChanged  = "tavern/pub/ifchanged"
	TopicTavernPubTTL      = "tavern/pub/ttl"
	TopicTavernHooksSource = "tavern/hooks/source"
	TopicTavernHooksDeriv  = "tavern/hooks/derived"
	TopicTavernHooksLog    = "tavern/hooks/log"
	TopicTavernHooksStats  = "tavern/hooks/stats"
)
