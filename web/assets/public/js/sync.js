/**
 * Sync manager for offline write queuing.
 * Uses IndexedDB for local storage — works in service workers, main thread,
 * and regular browsers without Capacitor.
 */

const DB_NAME = 'dothog_sync';
const QUEUE_STORE = 'sync_queue';

/**
 * Open the sync database.
 * @returns {Promise<IDBDatabase>}
 */
function openSyncDB() {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open(DB_NAME, 1);
    req.onupgradeneeded = () => {
      const db = req.result;
      if (!db.objectStoreNames.contains(QUEUE_STORE)) {
        const store = db.createObjectStore(QUEUE_STORE, {
          keyPath: 'id',
          autoIncrement: true,
        });
        store.createIndex('status', 'status', { unique: false });
      }
    };
    req.onsuccess = () => resolve(req.result);
    req.onerror = () => reject(req.error);
  });
}

/**
 * Queue an offline write operation.
 * @param {Object} op
 * @param {string} op.method - HTTP method (POST, PUT, DELETE)
 * @param {string} op.url - Request URL
 * @param {string} op.body - Form-encoded body
 * @param {string} op.contentType - Content-Type header value
 * @param {number|null} op.version - Row version for conflict detection
 * @returns {Promise<void>}
 */
async function queueWrite(op) {
  const db = await openSyncDB();
  return new Promise((resolve, reject) => {
    const tx = db.transaction(QUEUE_STORE, 'readwrite');
    tx.objectStore(QUEUE_STORE).add({
      method: op.method,
      url: op.url,
      body: op.body,
      contentType: op.contentType || 'application/x-www-form-urlencoded',
      version: op.version || null,
      createdAt: new Date().toISOString(),
      status: 'pending',
    });
    tx.oncomplete = () => resolve();
    tx.onerror = () => reject(tx.error);
  });
}

/**
 * Get the count of pending writes in the queue.
 * @returns {Promise<number>}
 */
async function getPendingCount() {
  const db = await openSyncDB();
  return new Promise((resolve, reject) => {
    const tx = db.transaction(QUEUE_STORE, 'readonly');
    const idx = tx.objectStore(QUEUE_STORE).index('status');
    const req = idx.count(IDBKeyRange.only('pending'));
    req.onsuccess = () => resolve(req.result);
    req.onerror = () => reject(req.error);
  });
}

/**
 * Get all pending operations for sync.
 * @returns {Promise<Array>}
 */
async function getPendingOperations() {
  const db = await openSyncDB();
  return new Promise((resolve, reject) => {
    const tx = db.transaction(QUEUE_STORE, 'readonly');
    const idx = tx.objectStore(QUEUE_STORE).index('status');
    const req = idx.getAll(IDBKeyRange.only('pending'));
    req.onsuccess = () => resolve(req.result);
    req.onerror = () => reject(req.error);
  });
}

/**
 * Update the status of a queued operation.
 * @param {number} id - Queue entry ID
 * @param {string} status - New status (syncing, synced, conflict, rejected)
 * @returns {Promise<void>}
 */
async function updateStatus(id, status) {
  const db = await openSyncDB();
  return new Promise((resolve, reject) => {
    const tx = db.transaction(QUEUE_STORE, 'readwrite');
    const store = tx.objectStore(QUEUE_STORE);
    const req = store.get(id);
    req.onsuccess = () => {
      const entry = req.result;
      if (entry) {
        entry.status = status;
        store.put(entry);
      }
      tx.oncomplete = () => resolve();
    };
    tx.onerror = () => reject(tx.error);
  });
}

/**
 * Remove all synced entries from the queue.
 * @returns {Promise<void>}
 */
async function clearSynced() {
  const db = await openSyncDB();
  return new Promise((resolve, reject) => {
    const tx = db.transaction(QUEUE_STORE, 'readwrite');
    const store = tx.objectStore(QUEUE_STORE);
    const idx = store.index('status');
    const req = idx.openCursor(IDBKeyRange.only('synced'));
    req.onsuccess = () => {
      const cursor = req.result;
      if (cursor) {
        cursor.delete();
        cursor.continue();
      }
    };
    tx.oncomplete = () => resolve();
    tx.onerror = () => reject(tx.error);
  });
}
