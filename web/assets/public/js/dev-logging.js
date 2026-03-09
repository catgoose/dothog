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
    } catch (_) {
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
