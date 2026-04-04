/**
 * Listen for server-sent theme-change events and apply them to the
 * document so all open browsers stay in sync.
 * @listens theme-change
 */
(function() {
	/** @type {EventSource} */
	var es = new EventSource("/sse/theme");
	es.addEventListener("theme-change", function(/** @type {MessageEvent} */ e) {
		document.documentElement.dataset.theme = e.data;
		if (window.appChannel) window.appChannel.postMessage({type:'theme-change',theme:e.data});
	});
})();
