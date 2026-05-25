package routes

// SSE topic constants for the broker. These are app-specific channel names
// used by the real-time features (dashboard, canvas, feed, etc.).
const (
	TopicSystemStats   = "system-stats"
	TopicDashMetrics   = "dashboard-metrics"
	TopicPeopleUpdate  = "people-update"
	TopicActivityFeed  = "activity-feed"
	TopicErrorTraces   = "error-traces"
	// TopicThemeChange is the prefix; the broadcasting topic name is
	// session-scoped (see ThemeTopicForSession). A bare "theme-change"
	// channel would retheme every connected user when one session
	// changed, so /sse/theme subscribes each connection to its own
	// cookie-derived topic instead.
	TopicThemeChange   = "theme-change"
	TopicCanvasUpdate  = "canvas-update"
	TopicAdminPanel    = "admin-panel"
	TopicNumericalDash = "numerical-dash"
	TopicNotifications = "notifications"
	TopicObservatory   = "observatory"
	TopicAppLifeline   = "app-lifeline"

	// Tavern gallery lab topics.
	TopicTavernReplay      = "tavern/replay"
	TopicTavernBackpress   = "tavern/backpressure"
	TopicTavernPubRaw      = "tavern/pub/raw"
	TopicTavernPubDebounce = "tavern/pub/debounced"
	TopicTavernPubThrottle = "tavern/pub/throttled"
	TopicTavernPubChanged  = "tavern/pub/ifchanged"
	TopicTavernPubTTL      = "tavern/pub/ttl"
	TopicTavernHooksSource = "tavern/hooks/source"
	TopicTavernHooksInput  = "tavern/hooks/input"
	TopicTavernHooksDeriv  = "tavern/hooks/derived"
	TopicTavernHooksLog    = "tavern/hooks/log"
	TopicTavernHooksStats  = "tavern/hooks/stats"
)

// ThemeTopicForSession returns the per-session theme-change topic name. The
// public /sse/theme endpoint uses this to scope each connection to its own
// cookie-derived topic; POST /settings/theme publishes through the same
// helper so only the session that mutated its preference sees the update.
// Tabs that share a cookie share a topic and converge; other sessions
// never receive the event.
func ThemeTopicForSession(sessionUUID string) string {
	return TopicThemeChange + ":" + sessionUUID
}
