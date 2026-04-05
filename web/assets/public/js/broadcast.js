// setup:feature:demo
/**
 * BroadcastChannel for cross-tab synchronization.
 * When one tab changes state (theme), all other tabs update.
 *
 * @fileoverview Creates a shared BroadcastChannel and exposes it as
 * window.appChannel so other scripts (theme-sse.js) can post messages
 * to all open tabs.
 */
(function() {
  if (!('BroadcastChannel' in window)) return;

  /** @type {BroadcastChannel} */
  const channel = new BroadcastChannel('{{BINARY_NAME}}');

  /**
   * @listens BroadcastChannel#message
   * @param {MessageEvent} event - Incoming cross-tab message.
   */
  channel.onmessage = function(event) {
    const msg = event.data;
    if (msg.type === 'theme-change') {
      document.documentElement.dataset.theme = msg.theme;
    }
  };

  window.appChannel = channel;
})();
