// setup:feature:demo
/**
 * BroadcastChannel for cross-tab synchronization.
 * When one tab changes state (theme, auth), all other tabs update.
 * No server round-trip needed for tab-to-tab communication.
 */
(function() {
  if (!('BroadcastChannel' in window)) return;

  var channel = new BroadcastChannel('dothog');

  channel.onmessage = function(event) {
    var msg = event.data;

    // Theme sync: another tab changed the theme
    if (msg.type === 'theme-change') {
      document.documentElement.dataset.theme = msg.theme;
    }
  };

  // Expose for other scripts to use
  window.dothogChannel = channel;
})();
