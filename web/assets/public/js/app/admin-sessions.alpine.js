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
 * Change events bubble from per-row checkboxes, row theme selects, and the
 * [data-session-select-all] header checkbox into one onChange handler, so the
 * swapped table fragment never needs Alpine re-initialization.
 */
(function () {
  function rowCheckboxes(root) {
    return Array.prototype.slice.call(
      root.querySelectorAll('input[type="checkbox"][data-session-select-uuid]')
    );
  }

  function themeSelects(root) {
    return Array.prototype.slice.call(
      root.querySelectorAll('select[data-theme-session-uuid]')
    );
  }

  function saveButtons(root) {
    return Array.prototype.slice.call(
      root.querySelectorAll('[data-session-theme-save]')
    );
  }

  function syncCheckboxes(root, selected) {
    rowCheckboxes(root).forEach(function (cb) {
      cb.checked = selected.indexOf(cb.dataset.sessionSelectUuid) !== -1;
    });
    var selectAll = root.querySelector('[data-session-select-all]');
    if (selectAll) {
      var boxes = rowCheckboxes(root);
      selectAll.checked =
        boxes.length > 0 &&
        boxes.every(function (cb) {
          return selected.indexOf(cb.dataset.sessionSelectUuid) !== -1;
        });
    }
  }

  function syncDraftControls(root, drafts) {
    themeSelects(root).forEach(function (select) {
      var uuid = select.dataset.themeSessionUuid;
      var draft = drafts[uuid];
      if (draft && select.value !== draft) {
        select.value = draft;
      }
    });
    saveButtons(root).forEach(function (button) {
      button.hidden = !drafts[button.dataset.sessionThemeSave];
    });
  }

  window.dothog.alpine.register('sessionsSelection', function () {
    return {
      /** @type {string[]} */
      selected: [],

      /** @type {Record<string, string>} */
      drafts: {},

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
       * @param {Event} event - Bubbled change event from a row checkbox, the
       * select-all checkbox, or a row theme select.
       */
      onChange: function (event) {
        var target = event.target;
        if (!target) {
          return;
        }
        if (target.tagName === 'SELECT' && target.hasAttribute('data-theme-session-uuid')) {
          var themeUUID = target.dataset.themeSessionUuid;
          var canonicalTheme = target.dataset.canonicalTheme || '';
          if (target.value === canonicalTheme) {
            delete this.drafts[themeUUID];
          } else {
            this.drafts[themeUUID] = target.value;
          }
          syncDraftControls(this.$root, this.drafts);
          return;
        }
        if (target.tagName !== 'INPUT') {
          return;
        }
        if (target.hasAttribute('data-session-select-all')) {
          var boxes = rowCheckboxes(this.$root);
          if (target.checked) {
            this.selected = boxes.map(function (cb) {
              return cb.dataset.sessionSelectUuid;
            });
          } else {
            this.selected = [];
          }
          syncCheckboxes(this.$root, this.selected);
          return;
        }
        if (target.hasAttribute('data-session-select-uuid')) {
          var uuid = target.dataset.sessionSelectUuid;
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
        var self = this;
        var present = {};
        rowCheckboxes(this.$root).forEach(function (cb) {
          present[cb.dataset.sessionSelectUuid] = true;
        });
        this.selected = this.selected.filter(function (u) {
          return present[u];
        });

        var nextDrafts = {};
        themeSelects(this.$root).forEach(function (select) {
          var uuid = select.dataset.themeSessionUuid;
          var draft = self.drafts[uuid];
          if (!draft) {
            return;
          }
          if (draft !== (select.dataset.canonicalTheme || '')) {
            nextDrafts[uuid] = draft;
          }
        });
        this.drafts = nextDrafts;

        syncCheckboxes(this.$root, this.selected);
        syncDraftControls(this.$root, this.drafts);
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
        if (!window.confirm('Delete the selected session rows? The browser cookie will be re-issued on next request.')) {
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
