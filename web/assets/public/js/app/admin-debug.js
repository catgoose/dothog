/**
 * @fileoverview Scaffold-baseline debug toggle controller.
 *
 * Backs the /admin/debug page and rehydrates persisted toggles on every
 * page load. The UI is wired with [data-debug-all] and [data-debug-key]
 * data attributes; this controller attaches delegated change listeners so
 * no inline event handlers are needed (CSP-safe under script-src 'self').
 */
(function () {
  'use strict';

  var KEY = 'app_debug';

  /**
   * @returns {Object<string, boolean>}
   */
  function load() {
    try {
      return JSON.parse(localStorage.getItem(KEY)) || {};
    } catch (e) {
      return {};
    }
  }

  /**
   * @param {Object<string, boolean>} state
   */
  function save(state) {
    localStorage.setItem(KEY, JSON.stringify(state));
  }

  var handlers = {
    'htmx-log': {
      on: function () {
        if (typeof htmx !== 'undefined') htmx.logAll();
      },
      off: function () {
        if (typeof htmx !== 'undefined') htmx.logNone();
      },
    },
    'htmx-events': {
      on: function () {
        window._htmxDbg = function (e) {
          console.debug(
            '%c[htmx:' + e.type.replace('htmx:', '') + ']',
            'color:#38bdf8;font-weight:bold',
            e.detail
          );
        };
        var evts = [
          'htmx:beforeRequest',
          'htmx:afterRequest',
          'htmx:beforeSwap',
          'htmx:afterSwap',
          'htmx:oobErrorNoTarget',
          'htmx:sseMessage',
          'htmx:sseError',
        ];
        evts.forEach(function (t) {
          document.body.addEventListener(t, window._htmxDbg);
        });
        window._htmxDbgEvts = evts;
      },
      off: function () {
        if (window._htmxDbg && window._htmxDbgEvts) {
          window._htmxDbgEvts.forEach(function (t) {
            document.body.removeEventListener(t, window._htmxDbg);
          });
          delete window._htmxDbg;
          delete window._htmxDbgEvts;
        }
      },
    },
    'hs-beep': {
      on: function () {
        window._hsDbg = function (e) {
          console.debug('%c[_hs:beep]', 'color:#a78bfa;font-weight:bold', e.detail);
        };
        document.body.addEventListener('hyperscript:beep', window._hsDbg);
      },
      off: function () {
        if (window._hsDbg) {
          document.body.removeEventListener('hyperscript:beep', window._hsDbg);
          delete window._hsDbg;
        }
      },
    },
    'alpine-events': {
      on: function () {
        window._alpineDbg = function (e) {
          console.debug('%c[alpine:' + e.type + ']', 'color:#34d399;font-weight:bold', e.detail);
        };
        document.addEventListener('alpine:initialized', window._alpineDbg);
        document.addEventListener('alpine:init', window._alpineDbg);
      },
      off: function () {
        if (window._alpineDbg) {
          document.removeEventListener('alpine:initialized', window._alpineDbg);
          document.removeEventListener('alpine:init', window._alpineDbg);
          delete window._alpineDbg;
        }
      },
    },
  };

  var checkboxMap = {
    'debug-htmx-log': 'htmx-log',
    'debug-htmx-events': 'htmx-events',
    'debug-hs': 'hs-beep',
    'debug-alpine': 'alpine-events',
  };

  /**
   * @param {string} key
   * @param {boolean} enabled
   */
  function toggle(key, enabled) {
    var state = load();
    state[key] = enabled;
    save(state);
    if (enabled) handlers[key].on();
    else handlers[key].off();
  }

  /**
   * @param {boolean} enabled
   */
  function toggleAll(enabled) {
    for (var id in checkboxMap) {
      var key = checkboxMap[id];
      var el = document.getElementById(id);
      if (el) el.checked = enabled;
      toggle(key, enabled);
    }
  }

  function hydrate() {
    var state = load();
    var allOn = true;
    for (var id in checkboxMap) {
      var key = checkboxMap[id];
      var el = document.getElementById(id);
      if (el && state[key]) {
        el.checked = true;
        handlers[key].on();
      } else {
        allOn = false;
      }
    }
    var allEl = document.querySelector('[data-debug-all]');
    if (allEl) allEl.checked = allOn;
  }

  document.addEventListener('change', function (event) {
    var target = event.target;
    if (!target || target.tagName !== 'INPUT') return;
    if (target.hasAttribute('data-debug-all')) {
      toggleAll(target.checked);
      return;
    }
    if (target.hasAttribute('data-debug-key')) {
      toggle(target.getAttribute('data-debug-key'), target.checked);
    }
  });

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', hydrate);
  } else {
    hydrate();
  }

  window._dbg = { toggle: toggle, toggleAll: toggleAll };
})();
