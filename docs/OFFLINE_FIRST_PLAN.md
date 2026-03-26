# Offline-First iOS App — Development Plan

## Goal

Ship the dothog demo as a native iOS app via Capacitor with offline-first capabilities. Users can read and write data without connectivity. Changes sync when back online. The server remains the authority.

## Core Principle

**One rendering path.** The server renders all HTML. The service worker caches it. HTMX doesn't know or care whether HTML came from the server or the cache — it makes a request, gets HTML, swaps it in. There are no offline-specific templates, no client-side rendering, no second set of pages to maintain.

```
Online:   HTMX request → Service Worker → Server → HTML → Cache + Render
Offline:  HTMX request → Service Worker → Cache  → HTML → Render
```

For reads, the cached HTML is the offline experience. For writes, the service worker intercepts the form submission, queues it to local SQLite, and returns a small "Saved (pending sync)" fragment. That fragment is the only HTML the service worker generates.

**Constraint:** Users can only access pages offline that they visited while online. The service worker replays the server's last response — it can't render pages the server never produced. This is an honest constraint: browse your assignments while you have signal, then work offline. If you need a page you haven't visited, you need connectivity.

## Phases

### Phase 1: Capacitor Shell (online only)

**Goal:** The dothog demo running in an iOS simulator as a native app. No offline support yet — just prove the webview + Go server architecture works.

**Steps:**

1. **Install Capacitor**
   - `npm install @capacitor/core @capacitor/cli`
   - `npx cap init "Dothog" "com.catgoose.dothog"` — generates `capacitor.config.ts`
   - Configure `server.url` to point at the Go dev server (`https://localhost:{port}`)

2. **Add iOS platform**
   - `npx cap add ios` — generates the `ios/` directory with an Xcode project
   - Open in Xcode: `npx cap open ios`

3. **Dev workflow**
   - Run the Go server locally
   - Run `npx cap run ios` to launch in simulator
   - The webview loads the Go server's HTML — HTMX, SSE, everything works
   - Add mage targets: `mage iosSync`, `mage iosOpen`, `mage iosRun`

4. **Validate**
   - All demo pages render correctly in the iOS webview
   - HTMX requests work (forms, partials, OOB swaps)
   - SSE streams connect and deliver live updates
   - Safe-area insets apply correctly (already in CSS)
   - Navigation feels native (no browser chrome, bottom nav works)

**Output:** A working iOS app that's a thin native wrapper around the existing web app.

---

### Phase 2: Service Worker + HTML Caching (read-only offline)

**Goal:** When the user loses connectivity, they can still view pages they've previously visited. No writes yet.

**Steps:**

1. **Create a service worker** (`web/assets/public/sw.js`)
   - Register it from the app layout template
   - Intercept all `GET` requests
   - Strategy: **network-first with cache fallback**
     - Try the server first
     - If the server responds, cache the response and return it
     - If the network fails, serve from cache
   - Pre-cache the shell (CSS, JS, HTMX, _hyperscript, DaisyUI) on install

2. **HTMX-aware caching**
   - Cache both full-page responses and HTMX partials
   - Key the cache by URL + `HX-Request` header presence (a partial and a full page for the same URL are different representations)
   - On cache hit for an HTMX request, return the cached partial — HTMX swaps it normally

3. **Connectivity detection**
   - Install `@capacitor/network` plugin
   - Listen for connectivity changes
   - Expose state to the UI via a CSS class on `<body>` or a small Alpine.js store
   - Show an offline indicator badge when disconnected

4. **Health check heartbeat**
   - Ping `/health` every 30 seconds (configurable)
   - Distinguish between offline (no network) and degraded (slow/intermittent)
   - The service worker uses this to decide between network-first and cache-first

5. **Uncached page handling**
   - If a user navigates to a page that isn't cached, show a friendly "not available offline" message with a back button
   - This is a hypermedia response — an error representation with controls, same as any other error

6. **Validate**
   - Load several pages while online (builds cache)
   - Enable airplane mode in simulator
   - Navigate to cached pages — they render correctly
   - Navigate to uncached pages — show the friendly offline message
   - HTMX partials from cache swap correctly
   - Offline indicator appears and disappears with connectivity

**Output:** Field workers can review their assignments, read inspection details, browse data — all offline. The biggest pain point (can't see anything without signal) is solved.

---

### Phase 3: Write Queue (offline writes)

**Goal:** Users can submit forms while offline. Writes queue locally and sync when back online.

**Steps:**

1. **Install SQLite plugin**
   - `npm install @capacitor-community/sqlite`
   - Create a `sync.js` module that manages the local database

2. **Sync queue table**
   ```sql
   CREATE TABLE sync_queue (
     id          INTEGER PRIMARY KEY AUTOINCREMENT,
     method      TEXT NOT NULL,       -- POST, PUT, DELETE
     url         TEXT NOT NULL,       -- /demo/repository/tasks/42
     body        TEXT NOT NULL,       -- form-urlencoded payload
     version     INTEGER,            -- row version at time of edit (from hidden field or data attr)
     created_at  TIMESTAMP NOT NULL,
     status      TEXT DEFAULT 'pending'  -- pending, syncing, synced, conflict
   );
   ```

3. **Service worker intercepts writes when offline**
   - For POST/PUT/DELETE when offline:
     - Save the form payload to `sync_queue`
     - Return a small HTML fragment: `<span class="badge badge-warning">Saved (pending sync)</span>`
     - HTMX swaps the fragment normally — the user sees confirmation
   - For POST/PUT/DELETE when online:
     - Pass through to the server normally — no interception

4. **Pending changes indicator**
   - Query `sync_queue WHERE status = 'pending'` count
   - Display in the offline indicator: "Offline — 3 changes pending"
   - Update after each write and after each sync

5. **Validate**
   - Go offline, submit a form, see "Saved (pending sync)"
   - Submit multiple forms, see the pending count increase
   - Pending changes persist across app restarts (SQLite is durable)
   - Come back online — pending indicator still shows (sync is Phase 4)

**Output:** Users can fill out forms and update records offline. Nothing syncs yet, but nothing is lost.

---

### Phase 4: Sync Protocol (online reconciliation)

**Goal:** When connectivity returns, queued writes flush to the server. Conflicts are detected.

**Steps:**

1. **`POST /sync` batch endpoint on the server**
   ```go
   type SyncOperation struct {
       Method  string `json:"method"`
       URL     string `json:"url"`
       Body    string `json:"body"`
       Version int    `json:"version"`
   }
   type SyncResult struct {
       Index       int    `json:"index"`
       Status      string `json:"status"` // applied, conflict, rejected
       NewVersion  int    `json:"new_version,omitempty"`
       CurrentData string `json:"current_data,omitempty"`
   }
   ```

2. **Server-side processing**
   - For each operation, check the version against the current row
   - Versions match: apply the mutation, increment version, return `applied`
   - Versions don't match: return `conflict` with current row data
   - Row was deleted: return `rejected`
   - Process sequentially within a transaction

3. **Client-side sync manager** (`sync.js`)
   - Trigger sync when connectivity is restored (network change event)
   - Also trigger on a timer when online (catch missed reconnects)
   - POST the pending queue to `/sync`
   - For `applied`: remove from queue, invalidate cache for that URL
   - For `conflict`: mark as conflict, surface to user (Phase 5)
   - For `rejected`: mark as rejected, notify user

4. **Post-sync refresh**
   - Invalidate the service worker cache for affected URLs
   - If the user is viewing a page with synced data, trigger an HTMX refresh
   - SSE pushes updates to other connected clients as usual

5. **Validate**
   - Queue several offline edits
   - Reconnect — watch them sync automatically
   - Verify the server has the correct data
   - Verify other connected clients see the updates via SSE
   - Force a version conflict — verify it's detected and flagged

**Output:** The full offline → online roundtrip works. Data created offline reaches the server. Conflicts are detected.

---

### Phase 5: Conflict Resolution + Push Notifications

**Goal:** Conflicts are surfaced as hypermedia and resolved through forms. Push notifications alert users.

**Steps:**

1. **Conflict as a resource**
   - `GET /conflicts` — list unresolved conflicts (admin or user view)
   - `GET /conflicts/:id` — single conflict showing both versions side by side
   - `POST /conflicts/:id/resolve` — submit the chosen resolution

2. **Conflict resolution form**
   - "Your version" vs "Current version" for each changed field
   - Controls: [Keep Mine] [Keep Theirs] [Merge field-by-field]
   - Standard HTMX form — the server applies the resolution
   - After resolution, SSE pushes the authoritative version to all clients

3. **Configurable strategy per table**
   - `last_write_wins` — auto-resolve, no user involvement (low-stakes)
   - `first_write_wins` — reject second write, user re-enters with current context (default)
   - `user_resolves` — present the conflict form (important data)
   - `admin_resolves` — escalate to admin queue (compliance/safety data)

4. **Push notifications**
   - Install `@capacitor/push-notifications`
   - After sync with conflicts: push "3 synced, 1 conflict needs attention"
   - After admin resolves a conflict: push to the original author
   - Server sends via APNs

5. **Validate**
   - Two devices edit the same record offline
   - Both sync — first applies, second gets a conflict
   - User sees the conflict form, resolves it
   - Both devices reflect the resolution

**Output:** Conflicts are a first-class hypermedia experience — a form with controls, not a technical error.

---

## Setup Integration

Feature flags for `mage setup`, layered:

- **`capacitor`** — scaffolds `capacitor.config.ts`, installs `@capacitor/core` and `@capacitor/cli` as npm deps, generates the `ios/` Xcode project, adds `mage ios*` targets to the magefile, adds `ios/` and Capacitor build artifacts to `.gitignore`
- **`offline`** — adds the service worker (`sw.js`), registers it in the app layout, installs `@capacitor/network`, adds the connectivity detection module and offline indicator component
- **`sync`** — installs `@capacitor-community/sqlite`, scaffolds `sync.js`, adds `routes_sync.go` with the `/sync` batch endpoint, adds `routes_conflicts.go` and `conflicts.templ` for conflict resolution, adds the sync queue schema

`capacitor` alone gives you an online-only native app. Add `offline` for read caching. Add `sync` for write queuing and reconciliation. `offline` implies `capacitor`. `sync` implies `offline`.

## Mage Targets

```
mage iosSetup     — install Capacitor + iOS platform
mage iosSync      — sync web assets to iOS project
mage iosRun       — build + run in simulator
mage iosBuild     — Fastlane production build
```

## Files Created (estimated)

```
capacitor.config.ts                 — Capacitor configuration
ios/                                — Xcode project (generated by Capacitor)
fastlane/Fastfile                   — iOS build/deploy automation
web/assets/public/sw.js             — Service worker (cache + offline write interception)
web/assets/public/js/sync.js        — Sync manager (queue, flush, conflict detection)
internal/routes/routes_sync.go      — /sync batch endpoint
internal/routes/routes_conflicts.go — Conflict CRUD handlers
web/views/conflicts.templ           — Conflict resolution UI
web/views/sync_status.templ         — Offline indicator, pending count
```

## Decisions Made

### Authentication

Auth is provider-agnostic. Crooner already abstracts the provider (Azure OIDC, Google,
local password, etc.). Offline mode only needs the cached session token — the service
worker serves it without validating against the auth server. On reconnect, if the token
expired, the first server request triggers re-auth through whatever provider is configured.
No offline-specific auth logic needed regardless of provider.

For company-owned devices managed with MDM (e.g., Airwatch), local device security is
handled at the OS level — the app doesn't need to add its own. For personal projects or
BYOD, the session token expiry and re-auth flow provide the security boundary.

### Offline Rendering

No client-side rendering. No offline-specific templates. The service worker replays
cached server HTML for reads and returns a minimal "pending sync" fragment for writes.
One rendering path, one set of templates, one server.

### Data Subset

The service worker cache is the working set — it contains whatever the user browsed while
online. No manual sync picker, no server-driven manifest. If coverage gaps emerge in real
projects, revisit proactive pre-fetching.

### Conflict Strategy

Per-project decision. Default: first-write-wins with re-prompt. Infrastructure is generic
(version checking, conflict detection). Resolution strategy is domain-specific and
configurable per table.

### iOS CI/CD

Fastlane in GitHub Actions (`runs-on: macos-latest`) or Azure DevOps (hosted macOS agents).
Handles code signing, provisioning profiles, TestFlight, and App Store submission.
Apple Developer account and provisioning profiles stored as CI secrets.

### Schema Migrations

Migrations happen online. Flow:

1. Client sync request includes local schema version
2. Server detects mismatch → returns `schema_upgrade_required`
3. Client flushes pending write queue first (old schema still valid)
4. Server sends new DDL
5. Client applies to local SQLite, reports new version
6. Normal sync resumes

**Rule:** Never migrate while pending writes exist. Flush first, migrate second.

## Open Questions

- **Conflict blocking**: Should conflicts block further syncs, or sync non-conflicting items independently? (Leaning: sync everything else, queue conflicts separately)
- **Offline write UX**: Should queued edits be reflected in cached pages, or show the original cached version with a "pending" badge? (Leaning: pending badge, don't modify cached HTML)
- **Multi-device**: Two devices from the same user edit offline — same version-based conflict detection applies, but UX for "you conflicted with yourself" may differ
- **Large file attachments**: Photos, signatures — separate upload pipeline with background upload, or same sync queue? (Leaning: separate pipeline, sync queue is for form data)
