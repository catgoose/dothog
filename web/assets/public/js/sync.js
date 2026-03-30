// setup:feature:sync
/**
 * Sync manager for offline write queuing.
 * Uses IndexedDB for local storage — works in service workers, main thread,
 * and regular browsers without Capacitor.
 */

const DB_NAME = '{{BINARY_NAME}}_sync';
const QUEUE_STORE = 'sync_queue';

/** @type {IDBDatabase|null} */
let _syncDB = null;

/**
 * Open the sync database, reusing an existing connection if available.
 * @returns {Promise<IDBDatabase>}
 */
function openSyncDB() {
  if (_syncDB) {
    return Promise.resolve(_syncDB);
  }
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
    req.onsuccess = () => {
      _syncDB = req.result;
      _syncDB.onclose = () => { _syncDB = null; };
      resolve(_syncDB);
    };
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

/**
 * Flush all pending operations to the server's /sync endpoint.
 * Called when connectivity is restored.
 * @returns {Promise<{applied: number, conflicts: number, rejected: number}>}
 */
async function flushQueue() {
  const ops = await getPendingOperations();
  if (ops.length === 0) {
    return { applied: 0, conflicts: 0, rejected: 0 };
  }

  const payload = {
    operations: ops.map((op) => ({
      method: op.method,
      url: op.url,
      body: op.body,
      content_type: op.contentType,
      version: op.version,
      queued_at: op.createdAt,
    })),
    schema_version: 1,
  };

  /** @type {Record<string, string>} */
  const headers = { 'Content-Type': 'application/json' };
  if (typeof document !== 'undefined') {
    /** @type {HTMLMetaElement|null} */
    const csrfMeta = document.querySelector('meta[name="csrf-token"]');
    if (csrfMeta) {
      headers['X-CSRF-Token'] = csrfMeta.content;
    }
  }

  const res = await fetch('/sync', {
    method: 'POST',
    headers: headers,
    body: JSON.stringify(payload),
  });

  if (!res.ok) {
    throw new Error(`Sync failed: ${res.status}`);
  }

  const data = await res.json();
  const counts = { applied: 0, conflicts: 0, rejected: 0 };

  for (const result of data.results) {
    const op = ops[result.index];
    if (!op) continue;

    switch (result.status) {
      case 'applied':
        await updateStatus(op.id, 'synced');
        counts.applied++;
        break;
      case 'conflict':
        await updateStatus(op.id, 'conflict');
        counts.conflicts++;
        break;
      case 'rejected':
        await updateStatus(op.id, 'rejected');
        counts.rejected++;
        break;
      default:
        await updateStatus(op.id, 'error');
        break;
    }
  }

  await clearSynced();
  return counts;
}
