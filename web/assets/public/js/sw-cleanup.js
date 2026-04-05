/**
 * @fileoverview Cleanup script that removes stale service workers and caches
 * from previous deployments. Runs on every page load to ensure the offline
 * feature can be cleanly enabled/disabled without leftover registrations or
 * cached assets interfering. Intentionally minimal — no error handling or
 * logging needed.
 */
if ('serviceWorker' in navigator) {
	navigator.serviceWorker.getRegistrations().then(function(regs) {
		regs.forEach(function(r) { r.unregister(); });
	});
	caches.keys().then(function(keys) {
		keys.forEach(function(k) { caches.delete(k); });
	});
}
