// setup:feature:offline
/**
 * Service worker for offline-first HTML caching.
 * Strategy: network-first with cache fallback.
 *
 * - GET requests: try server first, cache response, fall back to cache
 * - POST/PUT/DELETE: queue to IndexedDB when offline, pass through when online
 * - Static assets (/public/): cache-first (they're immutable with long max-age)
 */

importScripts('/public/js/sync.js');

const CACHE_NAME = '{{BINARY_NAME}}-v1';

// Connectivity state — updated by the main thread via postMessage
self._isOnline = true;

/**
 * Handle messages from the main thread.
 * SET_ONLINE_STATUS updates the connectivity flag used by fetch interception.
 * GET_PENDING_COUNT responds with the current queue size.
 */
self.addEventListener('message', (event) => {
  if (event.data && event.data.type === 'SET_ONLINE_STATUS') {
    self._isOnline = event.data.online;
  }
  if (event.data && event.data.type === 'GET_PENDING_COUNT') {
    getPendingCount().then((count) => {
      event.source.postMessage({ type: 'PENDING_COUNT', count });
    });
  }
});

// Static assets to pre-cache on install
const PRECACHE_URLS = [
  '/public/css/app-layout.css',
  '/public/js/htmx.min.js',
  '/public/js/_hyperscript.min.js',
  '/public/js/alpine.min.js',
  '/public/js/alpine.morph.min.js',
  '/public/js/htmx.alpine-morph.js',
  '/public/js/context-bar.js',
  '/public/js/history-breadcrumbs.js',
  '/public/js/beacon.js',
  '/public/js/broadcast.js',
  '/pwa',
  '/pwa/inspection',
  '/pwa/report',
  '/pwa/notes',
];

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME).then((cache) => cache.addAll(PRECACHE_URLS))
  );
  self.skipWaiting();
});

self.addEventListener('activate', (event) => {
  // Clean up old cache versions
  event.waitUntil(
    caches.keys().then((keys) =>
      Promise.all(keys.filter((k) => k !== CACHE_NAME).map((k) => caches.delete(k)))
    )
  );
  self.clients.claim();
});

/**
 * Intercept all fetch requests.
 * - Non-GET mutations when offline: queue to IndexedDB, return synthetic response.
 * - Static assets (/public/): cache-first (immutable).
 * - HTML/HTMX requests: network-first with cache fallback.
 */
self.addEventListener('fetch', (event) => {
  const { request } = event;

  // Mutations: queue offline, pass through online
  if (request.method !== 'GET') {
    if (!self._isOnline) {
      event.respondWith(handleOfflineWrite(request));
    }
    return;
  }

  // Static assets: cache-first (they have immutable Cache-Control headers)
  if (request.url.includes('/public/')) {
    event.respondWith(cacheFirst(request));
    return;
  }

  // HTML/HTMX requests: network-first with cache fallback
  event.respondWith(networkFirst(request));
});

/**
 * Cache-first strategy for static assets.
 * @param {Request} request
 * @returns {Promise<Response>}
 */
async function cacheFirst(request) {
  const cached = await caches.match(request);
  if (cached) {
    return cached;
  }
  const response = await fetch(request);
  if (response.ok) {
    const cache = await caches.open(CACHE_NAME);
    cache.put(request, response.clone());
  }
  return response;
}

/**
 * Network-first strategy for HTML pages and HTMX partials.
 * Caches successful responses. Falls back to cache when offline.
 * Uses the full URL + HX-Request header as the cache key to separate
 * full-page and partial representations of the same resource.
 * @param {Request} request
 * @returns {Promise<Response>}
 */
async function networkFirst(request) {
  // Build a cache key that distinguishes HTMX partials from full pages
  const cacheKey = buildCacheKey(request);

  try {
    const response = await fetch(request);
    if (response.ok) {
      const cache = await caches.open(CACHE_NAME);
      cache.put(cacheKey, response.clone());
    }
    return response;
  } catch (err) {
    // Network failed — try cache
    const cached = await caches.match(cacheKey);
    if (cached) {
      return cached;
    }

    // Nothing in cache — try the PWA landing page as offline fallback
    const pwaFallback = await caches.match('/pwa');
    if (pwaFallback) {
      return pwaFallback;
    }

    // Last resort: synthetic offline response
    return new Response(offlineHTML(), {
      status: 503,
      headers: { 'Content-Type': 'text/html' },
    });
  }
}

/**
 * Build a cache key that includes the HX-Request header so that
 * HTMX partials and full-page responses are cached separately.
 * @param {Request} request
 * @returns {Request}
 */
function buildCacheKey(request) {
  const isHTMX = request.headers.get('HX-Request') === 'true';
  if (isHTMX) {
    const url = new URL(request.url);
    url.searchParams.set('_htmx', '1');
    return new Request(url.toString(), { method: 'GET' });
  }
  return request;
}

/**
 * Handle a mutation request when offline.
 * Queues the write to IndexedDB and returns a synthetic "pending" response.
 * @param {Request} request
 * @returns {Promise<Response>}
 */
async function handleOfflineWrite(request) {
  const body = await request.text();

  await queueWrite({
    method: request.method,
    url: new URL(request.url).pathname + new URL(request.url).search,
    body: body,
    contentType: request.headers.get('Content-Type'),
  });

  const count = await getPendingCount();

  // Notify all clients of the new pending count
  const clients = await self.clients.matchAll();
  for (const client of clients) {
    client.postMessage({ type: 'PENDING_COUNT', count });
  }

  // Return a synthetic HTML response that HTMX can swap
  const html = `<div class="badge badge-warning badge-sm gap-1">
  <svg xmlns="http://www.w3.org/2000/svg" class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/>
  </svg>
  Saved (pending sync)
</div>`;

  return new Response(html, {
    status: 200,
    headers: {
      'Content-Type': 'text/html',
      'HX-Trigger': JSON.stringify({ pendingSync: count }),
      'HX-Reswap': 'innerHTML',
    },
  });
}

/**
 * Minimal offline fallback HTML.
 * @returns {string}
 */
function offlineHTML() {
  return `<!doctype html>
<html data-theme="dark">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Offline</title>
  <style>
    body { font-family: system-ui, sans-serif; display: flex; align-items: center; justify-content: center; min-height: 100vh; margin: 0; background: #1d232a; color: #a6adba; }
    .container { text-align: center; padding: 2rem; }
    h1 { font-size: 1.5rem; margin-bottom: 0.5rem; }
    p { color: #6b7280; }
    button { margin-top: 1rem; padding: 0.5rem 1.5rem; border-radius: 0.5rem; border: 1px solid #3d4451; background: #2a303c; color: #a6adba; cursor: pointer; }
    button:hover { background: #3d4451; }
  </style>
</head>
<body>
  <div class="container">
    <h1>You're offline</h1>
    <p>This page isn't available offline. Connect to the network and try again.</p>
    <button onclick="history.back()">Go Back</button>
  </div>
</body>
</html>`;
}
