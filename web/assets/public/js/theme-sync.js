/**
 * Sync the server-rendered theme to <html data-theme> after hx-boost
 * swaps. hx-boost replaces the <body> but preserves <html> attributes,
 * so the data-theme from the server response is lost without this.
 *
 * @listens htmx:afterSettle
 */
document.addEventListener('htmx:afterSettle', function() {
	const m = document.querySelector('meta[name="page-theme"]');
	if (m && m.content) document.documentElement.dataset.theme = m.content;
});
