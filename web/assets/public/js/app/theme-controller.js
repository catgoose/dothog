// setup:feature:session_settings
/**
 * @fileoverview Theme controller. Private to this file — no window export.
 *
 * Theme is server-owned session state. This controller is the browser-side
 * mirror: it applies the canonical theme to <html data-theme>, syncs every
 * mounted [data-theme-picker] in place, sends picker edits through an
 * explicit HTMX AJAX POST, and (when SSE is enabled) listens to /sse/theme
 * for live updates. External surfaces interact only through HTML
 * (data-theme-picker forms) and the app:theme-change CustomEvent.
 */
(function() {
  /** @type {string[]} */
  const activeSwatchClasses = ['ring-2', 'ring-primary', 'ring-offset-1'];

  /**
   * Sync a rendered picker fragment to the chosen theme.
   *
   * @param {Element} root - Theme picker root element.
   * @param {string} theme - Canonical theme value.
   */
  function syncPicker(root, theme) {
    root.dataset.currentTheme = theme;

    /** @type {HTMLSelectElement|null} */
    const select = root.querySelector('select[name="theme"]');
    if (select) {
      select.value = theme;
    }

    /** @type {HTMLElement|null} */
    const preview = root.querySelector('.theme-preview-swatch');
    if (preview) {
      preview.dataset.theme = theme;
    }

    root.querySelectorAll('.theme-swatch').forEach(function(btn) {
      const active = btn.getAttribute('data-theme-value') === theme;
      activeSwatchClasses.forEach(function(className) {
        btn.classList.toggle(className, active);
      });
    });
  }

  /**
   * Apply the canonical theme to the document and any mounted pickers.
   *
   * @param {string} theme - Canonical theme name.
   */
  function applyTheme(theme) {
    if (!theme) {
      return;
    }
    document.documentElement.dataset.theme = theme;

    /** @type {HTMLMetaElement|null} */
    const pageTheme = document.querySelector('meta[name="page-theme"]');
    if (pageTheme) {
      pageTheme.content = theme;
    }

    document.querySelectorAll('[data-theme-picker]').forEach(function(root) {
      syncPicker(root, theme);
    });
  }

  /**
   * Normalize the backing select value and send it through HTMX without a
   * browser navigation. When SSE is present the POST only mutates the
   * resource; without SSE the server returns the canonical picker fragment
   * to swap back into the form. The initiating tab also applies the chosen
   * theme immediately so the UI never waits on the round-trip to repaint.
   *
   * @param {HTMLElement|HTMLFormElement} control - The select, swatch button,
   *     or form submit source.
   */
  function submitTheme(control) {
    if (!control) {
      return;
    }

    /** @type {HTMLFormElement|null} */
    const form = control.tagName === 'FORM' ? /** @type {HTMLFormElement} */ (control) : control.closest('form');
    if (!form) {
      return;
    }

    /** @type {HTMLSelectElement|null} */
    const select = form.querySelector('select[name="theme"]');
    const theme = control.value || (select ? select.value : '');
    if (select && theme) {
      select.value = theme;
    }
    if (theme) {
      applyTheme(theme);
    }

    if (window.htmx && theme) {
      window.htmx.ajax('POST', form.getAttribute('action') || '/settings/theme', {
        source: form,
        target: form,
        swap: 'outerHTML',
        values: { theme: theme },
      });
      return;
    }

    form.submit();
  }

  document.addEventListener('htmx:afterSettle', function() {
    /** @type {HTMLMetaElement|null} */
    const pageTheme = document.querySelector('meta[name="page-theme"]');
    if (pageTheme && pageTheme.content) {
      applyTheme(pageTheme.content);
    }
  });

  document.body.addEventListener('app:theme-change', function(event) {
    if (event.detail && event.detail.theme) {
      applyTheme(event.detail.theme);
    }
  });

  // Delegated picker bindings. The picker form is just data-marked HTML;
  // these handlers turn its native submit/change/click into the async
  // theme-controller send path so derived apps don't need a separate
  // Alpine seam to wire them up.
  document.addEventListener('submit', function(event) {
    const form = event.target.closest('[data-theme-picker]');
    if (!form) return;
    event.preventDefault();
    submitTheme(form);
  });
  document.addEventListener('change', function(event) {
    const form = event.target.closest('[data-theme-picker]');
    if (!form) return;
    if (event.target.matches('select[name="theme"]')) {
      submitTheme(event.target);
    }
  });
  document.addEventListener('click', function(event) {
    const swatch = event.target.closest('[data-theme-picker] .theme-swatch');
    if (!swatch) return;
    event.preventDefault();
    submitTheme(swatch);
  });
})();

// setup:feature:sse:start
/**
 * Live theme receive path. Opens the only EventSource on /sse/theme and
 * republishes each server-sent change as one app-level CustomEvent
 * ("app:theme-change") — the controller above applies the canonical theme
 * to the document and any mounted pickers (no separate picker-sync event
 * is needed because applyTheme already updates every [data-theme-picker]
 * in place).
 *
 * @listens theme-change
 */
(function() {
  /** @type {EventSource} */
  const es = new EventSource("/sse/theme");
  es.addEventListener("theme-change", function(/** @type {MessageEvent} */ e) {
    document.body.dispatchEvent(new CustomEvent("app:theme-change", {
      bubbles: true,
      detail: { theme: e.data },
    }));
  });
})();
// setup:feature:sse:end
