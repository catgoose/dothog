/**
 * @fileoverview Advertise accepted error surfaces on HTMX requests.
 *
 * Reads two declarative attributes from the triggering element (or its nearest
 * ancestor) and translates them into neutral request headers the server can
 * negotiate against without ever needing to know concrete DOM selectors.
 *
 *   data-error-accept   -> X-Error-Accept-Surfaces   (comma-separated)
 *   data-error-fallback -> X-Error-Fallback-Surface  (single value)
 *
 * Server side: see internal/routes/handler/error_capabilities.go.
 */
(function () {
  /**
   * Register the global htmx:configRequest hook once the body exists.
   * @listens htmx:configRequest
   */
  function register() {
    if (!document.body) return;
    document.body.addEventListener("htmx:configRequest", function (evt) {
      var elt = evt.detail && evt.detail.elt;
      if (!elt) return;
      var accept = closestAttr(elt, "data-error-accept");
      var fallback = closestAttr(elt, "data-error-fallback");
      if (accept) {
        evt.detail.headers["X-Error-Accept-Surfaces"] = accept;
      }
      if (fallback) {
        evt.detail.headers["X-Error-Fallback-Surface"] = fallback;
      }
    });
  }

  /**
   * Walk up from the triggering element until the named attribute is found.
   * @param {Element|null} el
   * @param {string} name
   * @returns {string|null}
   */
  function closestAttr(el, name) {
    while (el && el.nodeType === 1) {
      if (el.hasAttribute && el.hasAttribute(name)) return el.getAttribute(name);
      el = el.parentElement;
    }
    return null;
  }
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", register);
  } else {
    register();
  }
})();
