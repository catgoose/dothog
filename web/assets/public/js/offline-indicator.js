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
      window.addEventListener('online', () => { this.online = true; });
      window.addEventListener('offline', () => { this.online = false; });

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
      } catch {
        this.online = false;
      }
    },
  };
}
