// setup:feature:demo
/**
 * Alpine.js component for history-based breadcrumbs.
 * Tracks the last N pages visited in sessionStorage and renders them
 * as a clickable trail showing "where you've been."
 * @returns {AlpineComponent}
 */
function historyBreadcrumbs() {
  var MAX = 4;
  var KEY = 'dothog_page_history';

  return {
    trail: [],
    hidden: localStorage.getItem('dothog_hide_history_crumbs') === 'true',
    init() {
      var history = JSON.parse(sessionStorage.getItem(KEY) || '[]');
      var current = window.location.pathname;

      // Remove current page if already in history (prevents duplicates on refresh)
      history = history.filter(function(h) { return h.path !== current; });

      // The trail is the history WITHOUT the current page
      this.trail = history.slice(-MAX);

      // Push current page onto history for next navigation
      history.push({ path: current, title: titleFromPath(current) });

      // Cap the history
      if (history.length > MAX + 1) {
        history = history.slice(-MAX - 1);
      }

      sessionStorage.setItem(KEY, JSON.stringify(history));
    },
    dismiss() {
      this.hidden = true;
      localStorage.setItem('dothog_hide_history_crumbs', 'true');
    }
  };
}

/**
 * Derive a readable title from a URL path.
 * "/demo/inventory" -> "Inventory"
 * "/admin/error-traces" -> "Error Traces"
 * "/" -> "Home"
 */
function titleFromPath(path) {
  if (path === '/') return 'Home';
  var segments = path.replace(/\/$/, '').split('/');
  var last = segments[segments.length - 1];
  return last.replace(/-/g, ' ').replace(/\b\w/g, function(c) { return c.toUpperCase(); });
}
