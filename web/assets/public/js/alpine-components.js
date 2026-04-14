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
    var root = null;
    return {
      current: '',
      init: function () {
        root = this.$el;
        this.current = root.dataset.currentTheme || 'dark';
      },
      setTheme: function (theme) {
        this.current = theme;
        document.documentElement.dataset.theme = theme;
        // Update all visual indicators
        var select = root.querySelector('select');
        if (select) select.value = theme;
        var preview = root.querySelector('.theme-preview-swatch');
        if (preview) preview.dataset.theme = theme;
        root.querySelectorAll('.theme-swatch').forEach(function (btn) {
          if (btn.dataset.themeValue === theme) {
            btn.classList.add('ring-2', 'ring-primary', 'ring-offset-1');
          } else {
            btn.classList.remove('ring-2', 'ring-primary', 'ring-offset-1');
          }
        });
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
        if (theme) this.setTheme(theme);
      },
      pickFromSelect: function (event) {
        var theme = event.target.value;
        if (theme) this.setTheme(theme);
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

  // -- Existing global functions that need CSP registration -------------
  if (typeof offlineIndicator === 'function') {
    Alpine.data('offlineIndicator', offlineIndicator);
  }

});
