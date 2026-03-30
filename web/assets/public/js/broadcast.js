// setup:feature:demo
/**
 * BroadcastChannel for cross-tab synchronization.
 * When one tab changes state (theme), all other tabs update.
 */
(function() {
  if (!('BroadcastChannel' in window)) return;

  var channel = new BroadcastChannel('{{BINARY_NAME}}');

  channel.onmessage = function(event) {
    var msg = event.data;
    if (msg.type === 'theme-change') {
      document.documentElement.dataset.theme = msg.theme;
    }
  };

  window.appChannel = channel;
})();
