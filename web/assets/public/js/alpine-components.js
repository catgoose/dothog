/**
 * Alpine.js CSP component registrations.
 *
 * The CSP build of Alpine (@alpinejs/csp) does not use eval(), so every
 * x-data component must be registered via Alpine.data() rather than using
 * inline object expressions.  This file is loaded BEFORE alpine.min.js
 * (both use defer, so execution order follows source order).
 *
 * Registration happens inside the "alpine:init" event, which the CSP build
 * fires before it walks the DOM.
 */
document.addEventListener('alpine:init', function () {

  // -- Alert toast (body in index.templ) --------------------------------
  Alpine.data('alertListener', function () {
    return {
      showAlert: function (event) {
        var t = document.createElement('div');
        t.className = 'toast toast-end toast-top z-50';
        var a = document.createElement('div');
        a.className = 'alert alert-info shadow-lg';
        a.textContent = event.detail;
        t.appendChild(a);
        document.body.appendChild(t);
        setTimeout(function () {
          t.style.transition = 'opacity 0.3s ease';
          t.style.opacity = '0';
          setTimeout(function () { t.remove(); }, 300);
        }, 3000);
      }
    };
  });

  // -- Error trace row (error_traces.templ) -----------------------------
  Alpine.data('traceRow', function () {
    return {
      expanded: false,
      loaded: false,
      toggle: function () {
        this.expanded = !this.expanded;
        if (this.expanded && !this.loaded) {
          this.loaded = true;
          // $el is the <tr> with x-on:click (the evaluation element),
          // which is also the element carrying hx-get and hx-trigger="expand".
          htmx.trigger(this.$el, 'expand');
        }
      },
      collapse: function () {
        this.expanded = false;
      }
    };
  });

  // -- Theme picker (settings_app.templ) ---------------------------------
  Alpine.data('themePicker', function () {
    return {
      current: '',
      init: function () {
        this.current = this.$el.dataset.currentTheme || 'dark';
      },
      setTheme: function (theme) {
        this.current = theme;
        document.documentElement.dataset.theme = theme;
        this._highlightSwatch();
        this._updatePreviewSwatch();
        this._syncSelect();
        var t = document.querySelector('meta[name="csrf-token"]');
        fetch('/settings/theme', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/x-www-form-urlencoded',
            'X-CSRF-Token': t ? t.content : ''
          },
          body: 'theme=' + theme
        });
        if (window.appChannel) {
          window.appChannel.postMessage({ type: 'theme-change', theme: theme });
        }
      },
      pickSwatch: function (event) {
        var btn = event.target.closest('.theme-swatch');
        if (!btn) return;
        var theme = btn.dataset.themeValue;
        if (theme) {
          this.setTheme(theme);
        }
      },
      pickFromSelect: function (event) {
        var theme = event.target.value;
        if (theme) {
          this.setTheme(theme);
        }
      },
      _highlightSwatch: function () {
        var current = this.current;
        this.$el.querySelectorAll('.theme-swatch').forEach(function (btn) {
          if (btn.dataset.themeValue === current) {
            btn.classList.add('ring-2', 'ring-primary', 'ring-offset-1');
          } else {
            btn.classList.remove('ring-2', 'ring-primary', 'ring-offset-1');
          }
        });
      },
      _updatePreviewSwatch: function () {
        var preview = this.$el.querySelector('.theme-preview-swatch');
        if (preview) {
          preview.dataset.theme = this.current;
        }
      },
      _syncSelect: function () {
        var select = this.$el.querySelector('select');
        if (select) {
          select.value = this.current;
        }
      }
    };
  });

  // -- Expandable (log entry attrs in error_traces.templ) ---------------
  Alpine.data('expandable', function () {
    return {
      open: false,
      toggle: function () { this.open = !this.open; }
    };
  });

  // -- Auto-open modal (report_email.templ) -----------------------------
  Alpine.data('autoModal', function () {
    return {
      init: function () {
        var el = this.$el;
        this.$nextTick(function () { el.showModal(); });
      }
    };
  });

  // -- Bulk select-all checkbox (bulk.templ) ----------------------------
  Alpine.data('bulkSelectAll', function () {
    return {
      toggleAll: function () {
        var checked = this.$el.querySelector('.select-all-check').checked;
        this.$el.closest('table').querySelectorAll('.row-check').forEach(function (cb) {
          cb.checked = checked;
        });
      }
    };
  });

  // -- Bulk row click-to-toggle (bulk.templ) ----------------------------
  Alpine.data('bulkRowToggle', function () {
    return {
      toggleRow: function (event) {
        if (event.target.tagName !== 'INPUT') {
          var cb = this.$el.querySelector('.row-check');
          cb.checked = !cb.checked;
        }
      }
    };
  });

  // -- Locale Intl formatters (hypermedia_components3.templ) -------------
  Alpine.data('intlRelativeTime', function () {
    return {
      formatted: '',
      init: function () {
        this.formatted = new Intl.RelativeTimeFormat(navigator.language, { numeric: 'auto' }).format(-2, 'hour');
      }
    };
  });

  Alpine.data('intlCurrency', function () {
    return {
      formatted: '',
      init: function () {
        this.formatted = new Intl.NumberFormat(navigator.language, { style: 'currency', currency: 'USD' }).format(1234.56);
      }
    };
  });

  Alpine.data('intlList', function () {
    return {
      formatted: '',
      init: function () {
        this.formatted = new Intl.ListFormat(navigator.language, { style: 'long', type: 'conjunction' }).format(['Alice', 'Bob', 'Charlie']);
      }
    };
  });

  Alpine.data('intlDate', function () {
    return {
      formatted: '',
      init: function () {
        this.formatted = new Intl.DateTimeFormat(navigator.language, { dateStyle: 'full' }).format(new Date());
      }
    };
  });

  // -- Range input live output (filter.templ) ----------------------------
  Alpine.data('rangeOutput', function () {
    return {
      updateOutput: function () {
        var input = this.$el.querySelector('input[type="range"]');
        var output = this.$el.querySelector('output');
        if (input && output) {
          output.textContent = input.value;
        }
      }
    };
  });

  // -- NavBar close-on-outside-click (nav.templ) ------------------------
  Alpine.data('navBar', function () {
    return {
      closeOthers: function (event) {
        var el = this.$el;
        el.querySelectorAll('details[open]').forEach(function (d) {
          if (!d.contains(event.target)) {
            d.open = false;
          }
        });
      }
    };
  });

  // -- NavMenu details exclusive toggle (nav.templ) ---------------------
  Alpine.data('navMenuDropdown', function () {
    return {
      closeOtherDropdowns: function () {
        var el = this.$el;
        if (el.open) {
          el.closest('ul.menu-horizontal').querySelectorAll('details').forEach(function (d) {
            if (d !== el) {
              d.open = false;
            }
          });
        }
      }
    };
  });

  // -- Error copy-to-clipboard (error_status.templ) ---------------------
  Alpine.data('errorCopy', function () {
    return {
      copyError: function () {
        navigator.clipboard.writeText(this.$el.dataset.errorJson);
        var tip = this.$refs.copyTip;
        if (tip) {
          tip.classList.remove('hidden');
          setTimeout(function () { tip.classList.add('hidden'); }, 1500);
        }
      }
    };
  });

  // -- Dismiss inline error (error_status.templ) ------------------------
  Alpine.data('dismissError', function () {
    return {
      dismiss: function () {
        var container = this.$el.closest('div[id]');
        if (container) {
          container.innerHTML = '';
        }
      }
    };
  });

  // -- Close parent dialog after HTMX request (modal.templ) -------------
  Alpine.data('modalSubmit', function () {
    return {
      closeDialog: function () {
        var dialog = this.$el.closest('dialog');
        if (dialog) {
          dialog.close();
        }
      }
    };
  });

  // -- Report issue toggle (report_issue.templ) -------------------------
  Alpine.data('reportIssueForm', function () {
    return {
      showMessageField: function () {
        var toggle = this.$refs.addToggle;
        var field = this.$refs.msgField;
        if (toggle) {
          toggle.remove();
        }
        if (field) {
          field.classList.remove('hidden');
          var textarea = field.querySelector('textarea');
          if (textarea) {
            textarea.focus();
          }
        }
      }
    };
  });

  // -- Existing global functions that need CSP registration -------------
  if (typeof offlineIndicator === 'function') {
    Alpine.data('offlineIndicator', offlineIndicator);
  }

});
