// setup:feature:session_settings
/**
 * @fileoverview Global theme state owner.
 *
 * Theme is app-wide state, not local component state. This controller applies
 * the canonical theme to <html data-theme>, keeps mounted theme pickers in
 * sync when asked, and sends theme changes through an explicit HTMX AJAX POST.
 * With SSE enabled, the canonical receive path is the theme-change stream.
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
   * resource; without SSE the server returns the canonical picker fragment.
   *
   * @param {HTMLElement|HTMLFormElement} control - The select, swatch button,
   *     or form submit source.
   * @returns {boolean} False to suppress the native default after submit.
   */
  function submitTheme(control) {
    if (!control) {
      return true;
    }

    /** @type {HTMLFormElement|null} */
    const form = control.tagName === 'FORM' ? /** @type {HTMLFormElement} */ (control) : control.closest('form');
    if (!form) {
      return true;
    }

    /** @type {HTMLSelectElement|null} */
    const select = form.querySelector('select[name="theme"]');
    const theme = control.value || (select ? select.value : '');
    if (select && theme) {
      select.value = theme;
    }

    if (window.htmx && theme) {
      window.htmx.ajax('POST', form.getAttribute('action') || '/settings/theme', {
        source: form,
        target: form,
        swap: 'outerHTML',
        values: { theme: theme },
      });
      return false;
    }

    form.submit();
    return false;
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

  window.themeController = {
    apply: applyTheme,
    submitTheme: submitTheme,
  };
})();
