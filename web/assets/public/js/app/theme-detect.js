/**
 * @fileoverview Dark-mode fallback for full-page error shells.
 *
 * The standalone ErrorPageShell is rendered outside the SPA, so no session
 * theme cookie has been consulted yet. This script applies the system
 * preference once at load time to avoid flashing a light page to dark-mode
 * users. Loaded synchronously (no defer) so it runs before body styling.
 */
(function () {
  if (
    window.matchMedia &&
    window.matchMedia('(prefers-color-scheme: dark)').matches
  ) {
    document.documentElement.dataset.theme = 'dark';
  }
})();
