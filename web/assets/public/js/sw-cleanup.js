if ('serviceWorker' in navigator) {
	navigator.serviceWorker.getRegistrations().then(function(regs) {
		regs.forEach(function(r) { r.unregister(); });
	});
	caches.keys().then(function(keys) {
		keys.forEach(function(k) { caches.delete(k); });
	});
}
