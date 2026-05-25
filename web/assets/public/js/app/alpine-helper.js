/**
 * @fileoverview Shared Alpine registration helper.
 *
 * Page-scoped *.alpine.js files call dothog.alpine.register(name, factory)
 * to expose a named Alpine.data component. The helper queues registrations
 * before alpine:init and registers directly after — so source order and
 * file granularity stop mattering for derived apps.
 *
 * Exists for CSP-safe named registrations (the @alpinejs/csp build forbids
 * inline x-data="{...}" expressions); without CSP, inline Alpine state
 * would also be viable. Shell-local DOM behaviors belong in _hyperscript
 * on their owning element, not here.
 */
(function () {
  if (window.dothog && window.dothog.alpine) {
    return;
  }
  window.dothog = window.dothog || {};

  /** @type {Array<[string, Function]>} */
  var pending = [];
  var ready = false;

  document.addEventListener('alpine:init', function () {
    ready = true;
    pending.forEach(function (entry) {
      window.Alpine.data(entry[0], entry[1]);
    });
    pending = [];
  });

  window.dothog.alpine = {
    /**
     * Register a named Alpine.data component. Safe to call before alpine:init
     * fires — pending registrations are flushed once Alpine boots.
     *
     * @param {string} name
     * @param {Function} factory
     */
    register: function (name, factory) {
      if (ready && window.Alpine) {
        window.Alpine.data(name, factory);
      } else {
        pending.push([name, factory]);
      }
    },
  };
})();
