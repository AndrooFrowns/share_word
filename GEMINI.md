# ShareWord - Crossword Puzzle Platform

ShareWord is a collaborative crossword puzzle platform built with Go, SQLite, and Datastar. It enables users to create, import, solve, and share crossword puzzles in real-time.

## Project Overview

- **Backend**: Go (1.25.5) using `chi` for routing and `scs` for session management.
- **Database**: SQLite (via `modernc.org/sqlite`) with `sqlc` for type-safe query generation and `goose` for migrations.
- **Frontend**: `templ` for server-side component rendering and `datastar` for real-time reactivity via Server-Sent Events (SSE).
- **Real-time**: An embedded NATS server is used for internal pub/sub to synchronize puzzle states across clients.
- **Architecture**: A standard Go project structure with a clear separation between transport (HTTP), application logic (Service), and data access (DB).

## Directory Structure

- `cmd/`: Entry points for the application.
    - `server/`: The main web server.
    - `seed/`: Utility to seed the database with initial data.
    - `gen_puz/`: Utility for generating puzzle files.
- `internal/`: Private application code.
    - `app/`: Business logic and core services (Puzzle, Auth, Social).
    - `db/`: Database access layer, including SQL queries and generated code.
    - `transport/`: HTTP handlers, middleware, and router setup.
    - `web/`: Frontend assets, including `templ` components and static files (CSS/JS).
- `sql/schema/`: SQL migration files managed by `goose`.
- `notes/`: Documentation regarding project phases and design decisions.

## Building and Running

The project uses `just` as a task runner.

### Prerequisites
- Go 1.25.5+
- `sqlc` (for database code generation)
- `templ` (for HTML component generation)
- `air` (optional, for hot-reload development)

### Key Commands
- **Initialize/Generate**: `just generate` (runs `sqlc` and `templ`)
- **Migrate Database**: `just migrate-up`
- **Run Server**: `just run`
- **Development with Hot-Reload**: `just watch-run` (requires `air`)
- **Run Tests**: `just test`
- **Seed Database**: `just seed`

## Development Conventions

- **Database**: Always use `sqlc` for database interactions. Define queries in `internal/db/query.sql` and run `just generate`.
- **Frontend**: Components are built with `templ`. Reactive elements use `datastar` attributes (e.g., `data-on-click`, `data-sse`) to interact with the backend via SSE.
- **Real-time Updates**: Use the `Service.BroadcastUpdate` method to signal changes to a puzzle. This triggers SSE events that Datastar uses to update the UI without full page reloads.
- **Error Handling**: Prefer returning errors from services and handling them in the transport layer to return appropriate HTTP status codes.
- **Styling**: Custom CSS is organized in `internal/web/static/css/` and uses modern CSS variables.
