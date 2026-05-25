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

});
