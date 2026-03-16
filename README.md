# Base

An opinionated Go boilerplate for building server-rendered web apps. Designed to be AI-agent friendly — a coding agent can add, modify, or remove features with minimal context.

**Stack:** Go + SQLite + HTMX + Templ + Chi + Tailwind CSS

No CGO. No Node.js. No ORM. One binary, one database file.

## Quick Start

```bash
git clone https://github.com/omaklabs/base.git myapp
cd myapp
make build
./app serve
```

Visit `http://localhost:8080`.

## Commands

**Build:**
```
make build               Full build (generate + css + go build)
make dev                 Build and start the server
make css                 Compile Tailwind CSS
make css-watch           Watch Tailwind CSS for changes
make generate            Run templ generate + sqlc generate
make test                Run all tests
```

**App CLI:**
```
./app serve              Start the HTTP server
./app generate domain <name>    Scaffold a full domain
./app generate migration <name> Create a timestamped migration file
./app migrate up         Run pending migrations
./app migrate down       Rollback last migration
./app migrate status     Show migration status
./app routes             List all registered routes
./app seed               Run seed data
./app doctor             Run diagnostic checks
./app version            Show version info
```

## Project Structure

```
cmd/app/
  app.go              <- The ONE file you edit (module registration)
  serve.go, router.go, ...  <- Boilerplate (don't edit)

internal/
  server/              Shared infrastructure (Deps, helpers, error pages)
  middleware/           HTTP middleware (one per file)
  db/                  Database connection + SQLC-generated code
  jobs/                Background job queue
  email/               Email sending (dev logger / SMTP)
  ...                  Other infrastructure packages

templates/
  layouts/             Shared HTML layouts
  components/          Reusable UI components (buttons, cards, forms)

queries/               SQLC query files (one per domain)
migrations/            Goose SQL migrations
```

## Adding a Domain

```bash
./app generate domain post
sqlc generate
templ generate
go build ./...
```

This creates:
- `internal/post/` — handler, module, templates, tests
- `queries/posts.sql` — SQLC CRUD queries
- `migrations/<timestamp>_create_posts.sql` — table definition
- Updates `cmd/app/app.go` with the new module

## Removing a Domain

1. Remove the module line from `cmd/app/app.go`
2. Delete `internal/<domain>/`
3. Delete `queries/<domain>.sql`
4. Run `sqlc generate && go build ./...`

## Architecture

**Module pattern** — each domain exports a `server.Module` describing its routes, jobs, and seeds. Registration is a single line in `cmd/app/app.go`:

```go
var modules = []server.Module{
    posts.Module,
}
```

**Co-located templates** — Templ files live inside their domain package, not in a separate directory. Delete a domain directory and everything goes with it.

**Upgrade boundary** — `.omakase.yaml` declares which files are boilerplate (safe to overwrite) and which are user-owned (never touched during upgrades).

## Stack Choices

| Layer | Choice |
|-------|--------|
| Language | Go 1.25+ (no CGO) |
| Database | SQLite via `modernc.org/sqlite` |
| DB Access | SQLC (generated, type-safe queries) |
| Migrations | Goose (embedded SQL files) |
| Templates | Templ |
| Router | Chi |
| Interactivity | HTMX + Alpine.js |
| CSS | Tailwind CSS |
| Background Jobs | SQLite-backed queue |
| Sessions | Cookie + SQLite |

## For AI Agents

Read `CONVENTIONS.md` for the full conventions guide. It covers:
- Handler patterns (parse, validate, query, render)
- Naming conventions
- Error handling rules
- What you must never do
- Extension points

## License

MIT
