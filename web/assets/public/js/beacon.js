// setup:feature:demo
/**
 * Lightweight client analytics via navigator.sendBeacon.
 * Logs page views and HTMX navigation events without blocking the user.
 * Fire-and-forget — guaranteed to complete even during page unload.
 */
(function() {
  const endpoint = '/log/beacon';

  /**
   * Send an analytics event via navigator.sendBeacon.
   * @param {string} event - The event name (e.g. 'page_view', 'navigation').
   * @param {Object} [data={}] - Additional event-specific data.
   */
  function send(event, data) {
    const payload = JSON.stringify({
      event: event,
      path: window.location.pathname,
      referrer: document.referrer || '',
      timestamp: new Date().toISOString(),
      data: data || {}
    });
    navigator.sendBeacon(endpoint, new Blob([payload], { type: 'application/json' }));
  }

  // Log initial page load
  send('page_view');

  // Log HTMX navigations (hx-boost page transitions)
  document.body.addEventListener('htmx:pushedIntoHistory', function(e) {
    send('navigation', { to: e.detail.path });
  });

  // Log page unload with time spent
  const loadTime = Date.now();
  window.addEventListener('pagehide', function() {
    send('page_leave', { duration_ms: Date.now() - loadTime });
  });
})();
