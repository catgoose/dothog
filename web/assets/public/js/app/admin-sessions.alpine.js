// setup:feature:session_settings
/**
 * @fileoverview Page-scoped Alpine registration for the admin sessions
 * bulk-selection surface.
 *
 * Owns x-data="sessionsSelection" on the AdminSessionsPage wrapper. Selection
 * state stays on the outer element so it survives HTMX/SSE-driven swaps of
 * the inner #admin-sessions-table fragment; init() registers an
 * htmx:afterSwap listener that calls reconcile() to drop UUIDs no longer
 * present in the refreshed table.
 *
 * Change events bubble from per-row [data-session-uuid] checkboxes and the
 * [data-session-select-all] header checkbox into one onChange handler, so the
 * swapped table fragment never needs Alpine re-initialization.
 */
(function () {
  function uuidsIn(root) {
    return Array.prototype.slice.call(
      root.querySelectorAll('[data-session-uuid]')
    );
  }

  function syncCheckboxes(root, selected) {
    uuidsIn(root).forEach(function (cb) {
      cb.checked = selected.indexOf(cb.dataset.sessionUuid) !== -1;
    });
    var selectAll = root.querySelector('[data-session-select-all]');
    if (selectAll) {
      var boxes = uuidsIn(root);
      selectAll.checked =
        boxes.length > 0 &&
        boxes.every(function (cb) {
          return selected.indexOf(cb.dataset.sessionUuid) !== -1;
        });
    }
  }

  window.dothog.alpine.register('sessionsSelection', function () {
    return {
      /** @type {string[]} */
      selected: [],

      init: function () {
        var self = this;
        this.$root.addEventListener('htmx:afterSwap', function () {
          self.reconcile();
        });
      },

      /**
       * @returns {string} Status label for the bulk-action bar.
       */
      countLabel: function () {
        var n = this.selected.length;
        if (n === 0) return 'No rows selected';
        if (n === 1) return '1 row selected';
        return n + ' rows selected';
      },

      /**
       * @returns {boolean} True when there are zero selected rows.
       */
      isNoneSelected: function () {
        return this.selected.length === 0;
      },

      /**
       * @param {Event} event - Bubbled change event from a checkbox.
       */
      onChange: function (event) {
        var target = event.target;
        if (!target || target.tagName !== 'INPUT') {
          return;
        }
        if (target.hasAttribute('data-session-select-all')) {
          var boxes = uuidsIn(this.$root);
          if (target.checked) {
            this.selected = boxes.map(function (cb) {
              return cb.dataset.sessionUuid;
            });
          } else {
            this.selected = [];
          }
          syncCheckboxes(this.$root, this.selected);
          return;
        }
        if (target.hasAttribute('data-session-uuid')) {
          var uuid = target.dataset.sessionUuid;
          if (target.checked) {
            if (this.selected.indexOf(uuid) === -1) {
              this.selected.push(uuid);
            }
          } else {
            this.selected = this.selected.filter(function (u) {
              return u !== uuid;
            });
          }
          syncCheckboxes(this.$root, this.selected);
        }
      },

      /**
       * Drop selected UUIDs whose rows are no longer present after a
       * server-driven table swap, then resync DOM checkbox state.
       */
      reconcile: function () {
        var present = {};
        uuidsIn(this.$root).forEach(function (cb) {
          present[cb.dataset.sessionUuid] = true;
        });
        this.selected = this.selected.filter(function (u) {
          return present[u];
        });
        syncCheckboxes(this.$root, this.selected);
      },

      /**
       * POST the current selection to /admin/sessions/delete; the server
       * returns the refreshed table fragment and reconcile() clears the
       * survivors via the htmx:afterSwap hook.
       */
      batchDelete: function () {
        if (this.selected.length === 0) {
          return;
        }
        var payload = this.selected.slice().join(',');
        this.selected = [];
        window.htmx.ajax('POST', '/admin/sessions/delete', {
          target: '#admin-sessions-table',
          swap: 'innerHTML',
          values: { uuids: payload },
        });
      },
    };
  });
})();
