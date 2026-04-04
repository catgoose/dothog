/**
 * Restore persisted debug toggles from localStorage on every page load.
 * The admin debug page writes { "htmx-log": true, ... } to app_debug.
 * Waits for DOMContentLoaded so htmx/hyperscript are available.
 */
document.addEventListener('DOMContentLoaded', function() {
	var state;
	try { state = JSON.parse(localStorage.getItem('app_debug')) || {}; } catch(e) { return; }
	if (state['htmx-log'] && typeof htmx !== 'undefined') { htmx.logAll(); }
	if (state['htmx-events'] && typeof htmx !== 'undefined') {
		window._htmxDbg = function(e) {
			console.debug('%c[htmx:' + e.type.replace('htmx:','') + ']', 'color:#38bdf8;font-weight:bold', e.detail);
		};
		var evts = ['htmx:beforeRequest','htmx:afterRequest','htmx:beforeSwap','htmx:afterSwap','htmx:oobErrorNoTarget','htmx:sseMessage','htmx:sseError'];
		evts.forEach(function(t) { document.body.addEventListener(t, window._htmxDbg); });
		window._htmxDbgEvts = evts;
	}
	if (state['hs-beep']) {
		window._hsDbg = function(e) { console.debug('%c[_hs:beep]', 'color:#a78bfa;font-weight:bold', e.detail); };
		document.body.addEventListener('hyperscript:beep', window._hsDbg);
	}
	if (state['alpine-events']) {
		window._alpineDbg = function(e) { console.debug('%c[alpine:' + e.type + ']', 'color:#34d399;font-weight:bold', e.detail); };
		document.addEventListener('alpine:initialized', window._alpineDbg);
		document.addEventListener('alpine:init', window._alpineDbg);
	}
});
