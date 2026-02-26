# go-htmx-demo

Go + HTMX + Templ demo with batteries included.

## Stack

- **Go** — backend server with Echo v4
- **HTMX** — hypermedia-driven frontend interactions
- **Templ** — type-safe HTML templating for Go
- **Tailwind CSS + DaisyUI** — utility-first styling
- **SQLite** — embedded database via modernc.org driver
- **SSE** — real-time updates via Server-Sent Events

## Package Documentation

Browse the auto-generated API documentation for each package in the sidebar.

### Core

| Package | Description |
|---------|-------------|
| [config](packages/config.md) | Application configuration management |
| [logger](packages/logger.md) | Structured logging with slog |
| [shared](packages/shared.md) | Common utilities and types |

### Routing

| Package | Description |
|---------|-------------|
| [routes](packages/routes.md) | Echo server and route initialization |
| [handler](packages/routes-handler.md) | HTTP handler utilities and rendering |
| [htmx](packages/routes-htmx.md) | HX-* header constants and helpers |
| [hypermedia](packages/routes-hypermedia.md) | HATEOAS controls, filters, tables, pagination |
| [middleware](packages/routes-middleware.md) | Application middleware |
| [params](packages/routes-params.md) | Request parameter parsing |
| [response](packages/routes-response.md) | Fluent HTMX response builder |

### Data

| Package | Description |
|---------|-------------|
| [database](packages/database.md) | Database connection management |
| [repository](packages/database-repository.md) | Data access layer |
| [demo](packages/demo.md) | Self-contained SQLite demo database |

### Realtime

| Package | Description |
|---------|-------------|
| [ssebroker](packages/ssebroker.md) | Topic-based pub/sub SSE broker |
