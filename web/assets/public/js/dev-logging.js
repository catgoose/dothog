/**
 * @fileoverview Dev-mode logging for HTMX and HyperScript.
 *
 * Provides category-based filtering for HTMX's verbose event logging and
 * console helpers for inspecting HyperScript/HTMX elements.  All flags
 * persist in `localStorage` so they survive page reloads.
 *
 * Usage (browser console):
 *   htmxLog.status()            — show current flags
 *   htmxLog.enable('requests')  — enable a category
 *   htmxLog.disable('swaps')    — disable a category
 *   htmxLog.enableAll()         — log everything (like htmx.logAll())
 *   htmxLog.disableAll()        — silence all logging
 *
 *   hsDebug.logAllHyperscriptElements()  — log all [_] elements
 *   hsDebug.logHxElements()              — log all hx-* elements
 *   hsDebug.logSelector('.my-class')     — log elements matching selector
 *
 * @see https://htmx.org/docs/#debugging
 */
(function () {
  "use strict";

  /** @const {string} localStorage key for persisted flags */
  var LS_KEY = "htmxLogFlags";

  /**
   * Default flag values.  All categories are off by default to keep the
   * console clean; enable what you need via `htmxLog.enable()`.
   *
   * @enum {boolean}
   */
  var DEFAULTS = {
    /** htmx:beforeRequest, afterRequest, responseError, sendError, timeout, abort */
    requests: false,
    /** htmx:beforeSwap, afterSwap, oobBeforeSwap, oobAfterSwap */
    swaps: false,
    /** htmx:trigger, sseMessage, load */
    events: false,
    /** htmx:historyRestore, pushUrl, replaceUrl */
    history: false,
    /** htmx:afterSettle, afterProcessNode, removedFromDOM */
    dom: false,
    /** Every htmx log call — equivalent to htmx.logAll() */
    all: false,
    /** _hyperscript runtime tracing (reserved for future use) */
    hyperscript: false,
  };

  /* ── Category matchers ──────────────────────────────────────────────── */

  /** @const {RegExp} Matches HTMX request lifecycle events */
  var REQUEST_RE =
    /^htmx:(beforeRequest|afterRequest|responseError|sendError|timeout|abort)/;

  /** @const {RegExp} Matches HTMX swap lifecycle events */
  var SWAP_RE = /^htmx:(beforeSwap|afterSwap|oob)/i;

  /** @const {RegExp} Matches HTMX trigger and SSE events */
  var TRIGGER_RE = /^htmx:(trigger|sseMessage|load$)/;

  /** @const {RegExp} Matches HTMX history events */
  var HISTORY_RE = /^htmx:(history|pushUrl|replaceUrl)/i;

  /** @const {RegExp} Matches HTMX DOM processing events */
  var DOM_RE = /^htmx:(afterSettle|afterProcessNode|removedFromDOM)/;

  /* ── Persistence ────────────────────────────────────────────────────── */

  /**
   * Load flags from localStorage, falling back to DEFAULTS.
   * @returns {Object<string, boolean>}
   */
  function load() {
    try {
      var stored = localStorage.getItem(LS_KEY);
      if (stored) {
        return Object.assign({}, DEFAULTS, JSON.parse(stored));
      }
    } catch (_e) {
      /* ignore corrupt data */
    }
    return Object.assign({}, DEFAULTS);
  }

  /**
   * Save flags to localStorage.
   * @param {Object<string, boolean>} flags
   */
  function save(flags) {
    localStorage.setItem(LS_KEY, JSON.stringify(flags));
  }

  /** @type {Object<string, boolean>} Active flag state */
  var flags = load();

  /* ── HTMX logger ────────────────────────────────────────────────────── */

  document.addEventListener("DOMContentLoaded", function () {
    if (typeof htmx !== "undefined") {
      /**
       * Custom HTMX logger that filters events by category.
       * @param {Element} elt   - The element that triggered the event
       * @param {string}  event - The HTMX event name
       * @param {*}       data  - Event detail payload
       */
      htmx.logger = function (elt, event, data) {
        if (flags.all) {
          console.log("[htmx]", event, elt, data);
          return;
        }
        if (flags.requests && REQUEST_RE.test(event)) {
          console.log("[htmx:req]", event, elt, data);
          return;
        }
        if (flags.swaps && SWAP_RE.test(event)) {
          console.log("[htmx:swap]", event, elt, data);
          return;
        }
        if (flags.events && TRIGGER_RE.test(event)) {
          console.log("[htmx:evt]", event, elt, data);
          return;
        }
        if (flags.history && HISTORY_RE.test(event)) {
          console.log("[htmx:hist]", event, elt, data);
          return;
        }
        if (flags.dom && DOM_RE.test(event)) {
          console.log("[htmx:dom]", event, elt, data);
          return;
        }
      };

      /* ── Always-on error event listeners ─────────────────────────────── */

      /**
       * Log detailed context when HTMX can't find a target for the response.
       * This fires when hx-target-error or hx-target resolves to no element.
       */
      document.body.addEventListener("htmx:targetError", function (e) {
        var d = e.detail;
        console.group("%c[htmx:targetError]", "color:#ef4444;font-weight:bold");
        console.error("Target not found:", d.target);
        console.log("Triggering element:", d.elt);
        console.log("Element hx-target:", d.elt && d.elt.getAttribute("hx-target"));
        console.log("Element hx-target-error:", d.elt && d.elt.getAttribute("hx-target-error"));
        console.log("Element hx-target-*:", d.elt && d.elt.getAttribute("hx-target-4*"));
        console.log("Closest hx-target-error:", d.elt && d.elt.closest("[hx-target-error]"));
        console.groupEnd();
      });

      /**
       * Log detailed context for HTMX response errors (non-2xx without swap).
       */
      document.body.addEventListener("htmx:responseError", function (e) {
        var d = e.detail;
        var xhr = d.xhr;
        console.group("%c[htmx:responseError]", "color:#ef4444;font-weight:bold");
        console.error("Status:", xhr.status, xhr.statusText);
        console.log("URL:", xhr.responseURL || d.pathInfo && d.pathInfo.requestPath);
        console.log("Triggering element:", d.elt);
        console.log("Target element:", d.target);
        if (xhr.responseText) {
          console.log("Response body (first 500 chars):", xhr.responseText.substring(0, 500));
        }
        console.groupEnd();
      });

      /**
       * Log network/connection failures.
       */
      document.body.addEventListener("htmx:sendError", function (e) {
        var d = e.detail;
        console.group("%c[htmx:sendError]", "color:#ef4444;font-weight:bold");
        console.error("Failed to send request");
        console.log("Triggering element:", d.elt);
        console.log("Target:", d.target);
        if (d.xhr) {
          console.log("URL:", d.xhr.responseURL);
        }
        console.groupEnd();
      });

      /**
       * Log swap errors (response received but swap failed).
       */
      document.body.addEventListener("htmx:swapError", function (e) {
        var d = e.detail;
        console.group("%c[htmx:swapError]", "color:#ef4444;font-weight:bold");
        console.error("Swap failed");
        console.log("Triggering element:", d.elt);
        console.log("Target:", d.target);
        if (d.xhr) {
          console.log("Status:", d.xhr.status);
          console.log("Response body (first 500 chars):", d.xhr.responseText && d.xhr.responseText.substring(0, 500));
        }
        console.groupEnd();
      });

      /**
       * Log beforeSwap details for error responses to help diagnose swap config.
       */
      document.body.addEventListener("htmx:beforeSwap", function (e) {
        var d = e.detail;
        if (d.xhr && d.xhr.status >= 400) {
          console.group("%c[htmx:beforeSwap] error response", "color:#f59e0b;font-weight:bold");
          console.log("Status:", d.xhr.status);
          console.log("isError:", d.isError);
          console.log("shouldSwap:", d.shouldSwap);
          console.log("Target:", d.target);
          console.log("Triggering element:", d.elt);
          console.log("serverResponse (first 300 chars):", d.serverResponse && d.serverResponse.substring(0, 300));
          console.groupEnd();
        }
      });
    }

    if (typeof _hyperscript !== "undefined" && _hyperscript.config) {
      _hyperscript.config.defaultTransition = "all 0.3s ease";
    }
  });

  /* ── Public API: htmxLog ────────────────────────────────────────────── */

  /**
   * Console API for toggling HTMX log categories at runtime.
   * @namespace htmxLog
   * @global
   */
  window.htmxLog = {
    /**
     * Enable a logging category.
     * @param {string} cat - Category name (requests|swaps|events|history|dom|all|hyperscript)
     */
    enable: function (cat) {
      flags[cat] = true;
      save(flags);
      console.log("[htmxLog] enabled:", cat);
    },

    /**
     * Disable a logging category.
     * @param {string} cat - Category name
     */
    disable: function (cat) {
      flags[cat] = false;
      save(flags);
      console.log("[htmxLog] disabled:", cat);
    },

    /** Enable all logging categories. */
    enableAll: function () {
      for (var k in flags) {
        flags[k] = true;
      }
      save(flags);
      console.log("[htmxLog] all enabled");
    },

    /** Disable all logging categories. */
    disableAll: function () {
      for (var k in flags) {
        flags[k] = false;
      }
      save(flags);
      console.log("[htmxLog] all disabled");
    },

    /** Print current flag state as a table. */
    status: function () {
      console.table(flags);
    },
  };

  /* ── Public API: hsDebug ────────────────────────────────────────────── */

  /**
   * Console helpers for inspecting HyperScript and HTMX elements.
   * @namespace hsDebug
   * @global
   */
  window.hsDebug = {
    /**
     * Log all elements matching a CSS selector along with their `_` attribute.
     * @param {string} sel - CSS selector
     */
    logSelector: function (sel) {
      document.querySelectorAll(sel).forEach(function (el) {
        console.log(el, el.getAttribute("_"));
      });
    },

    /** Log all elements with a HyperScript `_` attribute. */
    logAllHyperscriptElements: function () {
      this.logSelector("[_]");
    },

    /** Log all elements with HTMX action attributes (hx-get, hx-post, etc.). */
    logHxElements: function () {
      this.logSelector(
        "[hx-get],[hx-post],[hx-put],[hx-patch],[hx-delete]",
      );
    },
  };
})();
