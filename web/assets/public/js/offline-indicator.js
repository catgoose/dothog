/**
 * Alpine.js data component for offline status detection.
 * Uses navigator.onLine and periodic /health pings to determine connectivity.
 * @returns {AlpineComponent}
 */
function offlineIndicator() {
  return {
    online: navigator.onLine,
    pending: 0,
    _interval: null,

    init() {
      window.addEventListener('online', () => {
        this.online = true;
        this.notifyServiceWorker(true);
      });
      window.addEventListener('offline', () => {
        this.online = false;
        this.notifyServiceWorker(false);
      });

      // Listen for pending count updates from the service worker
      navigator.serviceWorker?.addEventListener('message', (event) => {
        if (event.data?.type === 'PENDING_COUNT') {
          this.pending = event.data.count;
        }
      });

      // Heartbeat: verify actual server reachability (navigator.onLine can be wrong)
      this._interval = setInterval(() => this.checkHealth(), 30000);
    },

    destroy() {
      if (this._interval) {
        clearInterval(this._interval);
      }
    },

    /**
     * Ping /health to verify server reachability.
     * navigator.onLine only checks network interface, not actual connectivity.
     */
    async checkHealth() {
      try {
        const res = await fetch('/health', { method: 'HEAD', cache: 'no-store' });
        this.online = res.ok;
        this.notifyServiceWorker(res.ok);
      } catch {
        this.online = false;
        this.notifyServiceWorker(false);
      }
    },

    /**
     * Notify the service worker of connectivity changes.
     * @param {boolean} online
     */
    notifyServiceWorker(online) {
      navigator.serviceWorker?.controller?.postMessage({
        type: 'SET_ONLINE_STATUS',
        online,
      });
    },
  };
}
