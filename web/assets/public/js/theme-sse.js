// setup:feature:sse
// setup:feature:session_settings:start
/**
 * @fileoverview Listen for server-sent theme-change events and apply the
 * canonical theme through the shared theme controller.
 *
 * This helper only matters when both SSE and session_settings are present:
 * the server emits theme-change events over EventSource, and the client
 * applies them to the global document state plus any mounted theme pickers.
 *
 * @listens theme-change
 */
(function() {
	/** @type {EventSource} */
	const es = new EventSource("/sse/theme");
	es.addEventListener("theme-change", function(/** @type {MessageEvent} */ e) {
		document.body.dispatchEvent(new CustomEvent("app:theme-change", {
			bubbles: true,
			detail: { theme: e.data },
		}));
		document.body.dispatchEvent(new CustomEvent("app:theme-picker-sync", {
			bubbles: true,
			detail: { theme: e.data },
		}));
	});
})();
// setup:feature:session_settings:end
