# Dothog Philosophy Audit

Audit of the dothog reference/demo app against its own `PHILOSOPHY.md`.
The dothog repo *is* the demo — this audit looks for drift in the reference
implementation itself. Findings are scoped to `main.go`, `internal/`, `web/`,
and `cmd/`.

---

## Violations (ordered by severity)

1. **`panic()` for runtime conditions** — `config/config.go:163`
   (`MustGetConfig`) panics on config load failure; `main.go:49` panics on
   embedded FS error. Philosophy §1 permits `panic` only for programmer
   bugs — a missing/bad config file is a runtime condition. Return a wrapped
   error and let the caller decide.

2. **Bare `return err` in setup paths** — `setup/setup.go` has ~20 bare
   `return err` (e.g. lines 155, 189, 254, 275, 310) where wrapping would
   tell the operator *which* schema / table / step failed.
   `demo/link_relations.go:31, 94` and `session/session.go:69` do the same.

3. **`UserRepository` interface has 5 methods** —
   `repository/interfaces.go:14-20` defines `CreateOrUpdate`, `GetByID`,
   `GetByAzureID`, `Update`, `UpdateLastLogin`. Philosophy §1: interfaces
   are defined by the consumer; >3 methods describes an implementation,
   not a behavior.

4. **Raw `form-control` markup outside composition primitives** —
   `web/views/calendar.templ:263`, `web/views/people.templ:122`,
   `web/views/settings.templ:115`,
   `web/views/hypermedia_controls.templ:259` use
   `<div class="form-control">…<input input-sm>` directly instead of
   `StackedLayout > Card > FormSection > FormField`. The scaffold rule (see
   `vopts/docs/app-alignment.md`) derives from philosophy §2 uniform
   interface — the reference repo should set the example.

5. **Demo entities use `int64` IDs, domain uses `int`** —
   `demo/calendar.go:36` (`ID int64`), `demo/recovery.go:37,43`
   (`seq int64`). Scaffold convention is `int`; demo should model it.

6. **No skip-to-main-content link** — `web/views/index.templ` starts with
   `#error-status` / menu content; there is no
   `<a href="#main" class="sr-only focus:not-sr-only">` as the first
   focusable element. Philosophy §13 lists this as a WCAG 2.2 baseline.

7. **`<form method="dialog">` in modal primitives** —
   `web/components/core/modal.templ:28, 38, 67, 76` uses `method="dialog"`
   on `<form>` to close modals. Philosophy §2/§12 prefers native button
   `formmethod="dialog"` or HTMX `hx-on::click="this.close()"` so the form
   element stays a real form.

8. **Inventory POST fallbacks are ambiguous** —
   `routes/routes_inventory.go:35, 37` registers parallel POST routes
   alongside PUT/DELETE routes. The handler resolution rule (HTMX request
   → PUT, no-JS form submission → POST) is not documented at the
   registration site. Philosophy §2 allows the pattern but requires
   clarity.

9. **Two parallel session-settings abstractions** —
   `repository/session_settings_repository.go` (ListAll/Upsert) and
   `internal/session/session.go` (`Provider{GetByUUID, Upsert, Touch}`)
   both encode the same concern with different shapes. Pick one.

10. **Inconsistent `IsBoosted` handling on detail routes** —
    `routes/routes_inventory.go:53-55` and
    `routes/routes_catalog.go:42-44` explicitly branch on
    `cheddar.IsBoosted(c)` for the full-layout path.
    `routes/routes_people.go` detail handler has no such guard. Vopts
    app-alignment.md calls this out as a silent-bug class.

11. **No `Accept`-header content negotiation** — all handlers branch on
    `HX-Request` only. Philosophy §2 ("Accept Header: The Full
    Negotiation Mechanism") names this as the goal state; the HAL demo
    already routes JSON on a separate URL instead of dispatching on
    `Accept`.

## Inconsistencies

1. **Error wrapping is correct everywhere except `setup/` and `demo/`** —
   the route handlers uniformly use `fmt.Errorf("…: %w", err)`, but
   `demo/link_relations.go:94` returns a bare error despite eight correct
   wrappings in the same file.

2. **Edit forms split between `hx-put` and POST** — most edit handlers
   use `hx-put` (e.g. `routes/routes_hypermedia.go:66`). Inventory
   registers both POST and PUT for the same logical operation on
   separate routes (`routes/routes_inventory.go:34-37`). Pick one path
   and document the fallback strategy at the handler.

3. **Interface method budgets** — `Pinger`, `Provider`, `IssueReporter`
   are 1-3 methods (good); `UserRepository` is 5 (breaks the rule).

## Notable alignments

- Error-as-hypermedia is correctly implemented end-to-end: middleware
  distinguishes HTMX vs full-page, renders OOB swaps, promotes traces
  via promolog, and carries request IDs throughout. This is the
  reference every app should follow.
- DaisyUI semantic classes are consistent: no raw `bg-blue-600` or
  `text-red-500` found anywhere in `web/views/`.
- No GET handler mutates state. HTTP verb hygiene is good.
- Progressive enhancement is real: `<noscript>` POST fallbacks in
  inventory (~lines 82-87) actually work without JS.
- Request tracing is wired correctly through promolog, the slog handler,
  and `X-Request-ID` response headers.

## Suggested follow-ups (concrete, scoped diffs)

1. **`setup/setup.go`**: wrap every bare `return err` with
   `fmt.Errorf("create schema %s: %w", name, err)` or the equivalent
   step description. (~20 sites.)

2. **`config/config.go:163`**: replace `panic` in `MustGetConfig` with a
   returned error; move the terminal panic (if any) to `main.go` where
   the caller actually decides whether the process can continue.

3. **`demo/calendar.go:36`, `demo/recovery.go:37,43`**: change
   `int64` → `int` to match domain convention.

4. **`web/views/index.templ`**: insert
   `<a href="#main" class="sr-only focus:not-sr-only focus:absolute
   focus:z-50 focus:p-2">Skip to main content</a>` as the first
   focusable element.

5. **`web/components/core/modal.templ`**: replace `<form method="dialog">`
   wrappers with button `formmethod="dialog"` or
   `hx-on::click="closest dialog then call it.close()"`.

6. **`repository/interfaces.go:14-20`**: split `UserRepository` — fold
   `UpdateLastLogin` into `Update`, or extract it onto a caller-defined
   single-method interface (philosophy §1 consumer-defined interfaces).

7. **`routes/routes_inventory.go:34-37`**: either remove the POST
   fallback and handle it via `formmethod`/`formaction` at the button
   level, or add a header comment explaining the POST-fallback dispatch
   rule.

8. **`routes/routes_people.go` detail handler**: add the same
   `cheddar.IsBoosted(c)` branch that inventory and catalog use.

9. **Consolidate session settings**: remove `session.Provider` and have
   route handlers depend on `repository.SessionSettingsStore` directly
   (or vice versa — one owner).

10. **HAL demo `Accept`-header dispatch**: document the current
    separate-URL strategy in `docs/HAL.md` as an incremental step, and
    land a same-URL `Accept: application/hal+json` dispatch on one
    resource (e.g. `/api/hal/tasks/:id`) as the reference pattern.
