Use the fff MCP tools for all file search operations instead of default tools.

# Working with Dothog

This is a framework starter for Go + HTMX + Templ applications. It is a template — not a library, not a framework you depend on. You fork it via `mage setupTo` and it becomes your app.

## For AI Agents

If you are an AI agent working on a project derived from dothog:

### Reporting Issues Upstream

If you find a bug or limitation in the template itself (not your app-specific code), report it:

1. Check if it's already in `todo.md` in the dothog repo
2. If not, open an issue at the dothog GitHub repository
3. Include: what feature configuration you used (`SETUP_FEATURES=...`), the error, and the minimal reproduction

Common upstream issues:
- Feature stripping leaves broken references (missing `// setup:feature:*` markers)
- Tests reference types from stripped features
- Config fields used outside their feature gate
- Generated `_templ.go` files have stale imports after stripping

### What Lives Where

| Code | Where it lives | Import |
|------|---------------|--------|
| SQL dialects (SQLite, Postgres, MSSQL) | [catgoose/fraggle](https://github.com/catgoose/fraggle) | `dialect "github.com/catgoose/fraggle"` |
| Schema DSL, table traits, DDL generation | [catgoose/fraggle/schema](https://github.com/catgoose/fraggle) | `"github.com/catgoose/fraggle/schema"` |
| Query builders, SQL fragments, audit helpers | [catgoose/fraggle/dbrepo](https://github.com/catgoose/fraggle) | `"github.com/catgoose/fraggle/dbrepo"` |
| Error trace capture, promote-on-error | [catgoose/promolog](https://github.com/catgoose/promolog) | `"github.com/catgoose/promolog"` |
| OIDC authentication | [catgoose/crooner](https://github.com/catgoose/crooner) | `"github.com/catgoose/crooner"` |
| Environment config | [catgoose/dio](https://github.com/catgoose/dio) | `"github.com/catgoose/dio"` |
| App table definitions | `internal/database/schema/` | In-tree (re-exports fraggle/schema) |
| RepoManager, schema lifecycle, transactions | `internal/database/repository/` | In-tree (uses sqlx + dothog logger) |
| App routes, middleware, hypermedia | `internal/routes/` | In-tree |

### Creating a New App

```bash
SETUP_FEATURES=none mage setupTo /path/to/new-app "App Name"
```

Feature options: `auth`, `graph`, `database`, `mssql`, `postgres`, `sse`, `caddy`, `avatar`, `demo`, `session_settings`, `alpine`

Implicit (always included): `database`, `alpine`

### Key Patterns

- **Config**: Flat `AppConfig` struct, singleton via `config.GetConfig()`. Extend by adding fields and reading env vars in `buildConfig()`.
- **Database**: `dialect.OpenURL(ctx, dsn)` for app databases. `dialect.OpenSQLite(ctx, path)` for framework internals (e.g. error trace store).
- **Layout**: Override with `handler.SetLayout(fn)` to use your own page wrapper.
- **Auth**: Set `OIDC_ISSUER_URL` + `OIDC_CLIENT_ID` to enable. Works with any OIDC provider.
- **Feature gates**: `// setup:feature:X:start` / `// setup:feature:X:end` for blocks. `// setup:feature:X` as first line for whole files.

### Don't

- Don't commit changes back to the dothog repo for app-specific code
- Don't add dothog dependencies to derived apps — dothog is a starting point, not an upstream
- Don't use the old `DB_ENGINE` / `DB_PATH` env vars — use `DATABASE_URL` with a scheme
