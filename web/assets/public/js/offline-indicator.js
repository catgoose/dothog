// setup:feature:offline
/**
 * Alpine.js data component for offline status detection.
 * Uses navigator.onLine and periodic /health pings to determine connectivity.
 * When transitioning from offline to online, flushes the sync queue.
 * @returns {AlpineComponent}
 */
function offlineIndicator() {
  return {
    online: navigator.onLine,
    pending: 0,
    _interval: null,
    _wasOffline: false,

    init() {
      window.addEventListener('online', () => {
        this.online = true;
        this.notifyServiceWorker(true);
        if (this._wasOffline) {
          this._wasOffline = false;
          this.syncIfNeeded();
        }
      });
      window.addEventListener('offline', () => {
        this.online = false;
        this._wasOffline = true;
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
        const wasOffline = !this.online;
        this.online = res.ok;
        this.notifyServiceWorker(res.ok);
        if (res.ok && wasOffline) {
          this.syncIfNeeded();
        }
      } catch {
        this.online = false;
        this.notifyServiceWorker(false);
      }
    },

    /**
     * Attempt to flush the offline queue when connectivity is restored.
     */
    async syncIfNeeded() {
      if (!this.online) return;
      try {
        const counts = await flushQueue();
        if (counts.applied > 0 || counts.conflicts > 0) {
          // Refresh the current page to show synced data
          const mainTarget = document.getElementById('main') || document.body;
          if (typeof htmx !== 'undefined') {
            htmx.ajax('GET', window.location.pathname, { target: mainTarget, swap: 'innerHTML' });
          }
        }
        if (counts.conflicts > 0) {
          // Trigger an HTMX event so the UI can show a conflict banner
          document.body.dispatchEvent(new CustomEvent('syncConflicts', {
            detail: { count: counts.conflicts },
          }));
        }
        // Update pending count
        const remaining = await getPendingCount();
        this.pending = remaining;
      } catch (err) {
        console.warn('Sync flush failed:', err);
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
