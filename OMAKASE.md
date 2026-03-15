# Omakase Go -- AI Agent Guide

Read this file in full before making any changes. It is the single source of truth for how this codebase works, what you are allowed to do, and how to do it.

---

## 1. Stack Rules -- Never Deviate

| Layer | Choice | Constraint |
|-------|--------|------------|
| Language | Go 1.25+ | No CGO. No libraries that require CGO. |
| Database | SQLite via `modernc.org/sqlite` | No Postgres, no MySQL, no other engines. |
| DB access | SQLC | No GORM, no raw `sql.Query` in handlers. |
| Migrations | Goose (plain SQL files) | Embedded via `migrations/embed.go`. |
| Templates | Templ (`github.com/a-h/templ`) | No `html/template`, no string concatenation. |
| Router | Chi (`github.com/go-chi/chi/v5`) | No Gin, no Echo, no Fiber. |
| Interactivity | HTMX + Alpine.js | No React, no Vue, no Svelte. |
| CSS | Tailwind CSS | No other CSS frameworks. Component variants via `data-*` attributes. |
| Background jobs | SQLite-backed job queue (`internal/jobs`) | No Redis, no external queue. |
| CSRF | `gorilla/csrf` | Token injected into context via middleware. |
| Sessions | Cookie + SQLite (`internal/auth`) | Strategy-agnostic session management. |
| Build tooling | Go only | No Node.js, no npm, no JavaScript build tools. |

---

## 2. Architecture Overview

### File Ownership

The codebase separates **boilerplate** (infrastructure, upgradeable) from **user** (domain logic, never overwritten). See `.omakase.yaml` for the full manifest.

- **Boilerplate files** (`cmd/app/*.go` except `app.go`, `internal/server/`, `internal/middleware/`, etc.) — safe to overwrite during upgrades.
- **User files** (`cmd/app/app.go`, `internal/notes/`, `internal/posts/`, `queries/`, user migrations) — never touched by upgrades.
- **`cmd/app/app.go`** — the ONE file in `cmd/app/` that users edit. Contains the `modules` list.

### Module Pattern

Each domain exports a `server.Module` var describing its routes, jobs, seeds. The boilerplate iterates the `modules` list in `app.go` for all registration.

```go
// internal/notes/module.go
var Module = server.Module{
    Name:  "notes",
    Path:  "/notes",
    Mount: Mount,
}
```

```go
// cmd/app/app.go — the ONLY file users edit in cmd/app/
var modules = []server.Module{
    notes.Module,
}
```

### Co-Located Templates

Page templates live inside their domain package, not in a separate `templates/pages/` directory. Shared layouts and components stay in `templates/`.

```
internal/notes/
├── handler.go          ← handlers
├── handler_test.go     ← tests
├── module.go           ← Module var
├── notes_list.templ    ← page templates (co-located)
├── notes_show.templ
└── notes_form.templ

templates/
├── layouts/            ← shared (boilerplate)
├── components/         ← shared (boilerplate)
└── icons/              ← shared (boilerplate)
```

---

## 3. Package Responsibility Map

| Package | Owns | Never |
|---------|------|-------|
| `internal/config` | Env loading (`Load()`), `.env` file parsing (`LoadDotEnv()`), config schema (`schema.go`), redacted config view (`view.go`) | Store runtime state here. |
| `internal/db` | SQLC-generated code (`*.sql.go`), DB connection (`connection.go`), generated models (`models.go`) | Edit generated files (`*.sql.go`, `models.go`) by hand. They are overwritten by `sqlc generate`. |
| `internal/auth` | Session create / validate / delete, expired session cleanup | Add authentication strategy here. This package is strategy-agnostic -- the agent adds login/signup logic in domain handlers. |
| `internal/middleware` | HTTP middleware chain. Each middleware lives in its own file: `request_id.go`, `request_logger.go`, `recovery.go`, `body_limit.go`, `session.go`, `csrf_context.go`, `flash.go`, `internal_key.go` | Combine multiple middlewares into one file. |
| `internal/server` | Shared HTTP infrastructure: `Deps` struct (dependency injection), `Module` type (domain registration), `IsHTMX()`, `RenderError()`, `RenderNotFound()`, `HandleNotFound()`, `HandleWelcome()`. Co-located templates: `welcome.templ`, `error_*.templ`. | Import domain packages from here. Server is a leaf dependency — domains import it, not the other way around. |
| `internal/notes` | Notes domain: `Module` var, `Mount()` + all CRUD handlers, co-located page templates. Reference domain for the agent to copy. | Put non-notes logic here. One package per domain. |
| `internal/jobs` | Job queue (`queue.go`), scheduler (`scheduler.go`). `Queue.Register()`, `Queue.Enqueue()`, `Queue.Process()`. `Scheduler.Add()`, `Scheduler.Start()`. | Process work inline in handlers. Always enqueue a job instead. |
| `internal/email` | `Mailer` interface (`email.go`), `DevMailer` (logs, no SMTP), `SMTPMailer` (production), `Store` (DB persistence) | Call SMTP directly. Always go through the `Mailer` interface. |
| `internal/logger` | Structured JSON logger with SSE fan-out. Methods: `Info()`, `Warn()`, `Error()`. | Use `fmt.Println` or `log.Printf` in application code. Always use the logger. |
| `internal/api` | Internal platform API mounted at `/internal/*`. Endpoints for config, health, jobs, emails, logs. | Expose these endpoints to end users. They are protected by `middleware.InternalAPIKey`. |
| `internal/flash` | Cookie-based flash messages. `flash.Set(w, message, variant)`, `flash.Get(r)`, `flash.GetAndClear(w, r)`. | Read flash directly in templates. The `FlashContext` middleware puts it into the view context. |
| `internal/view` | Template context carriers: CSRF token (`WithCSRFToken` / `CSRFToken`), flash messages (`WithFlash` / `GetFlash`). | Put business logic here. This is a thin context-passing layer only. |
| `internal/validate` | Form validation. `validate.New()` creates a `Validator`. Methods: `Required()`, `MinLength()`, `MaxLength()`, `Email()`, `Matches()`, `Equals()`, `In()`, `Unique()`, `Check()`. Returns `Errors` map. | Validate inside DB queries. Validate in handlers before querying. |
| `internal/pagination` | Page calculation. `pagination.New(page, perPage, total)` or `pagination.FromRequest(r, total)`. Returns `Pagination` struct with `Offset`, `Limit`, `HasPrev`, `HasNext`, `Pages()`. | Implement pagination math in handlers. Use `FromRequest`. |
| `internal/storage` | File storage interface (`Storage`): `Put()`, `Get()`, `Delete()`, `Exists()`, `URL()`. Local filesystem implementation (`local.go`). Upload parsing (`upload.go`): `ParseUpload()`, `GenerateKey()`. | Use `os.*` directly for file operations. Always go through the `Storage` interface. |
| `internal/testutil` | Test helpers: `SetupTestDB(t)` returns `(*sql.DB, *db.Queries, cleanup)`. `CreateTestUser(t, q, email)` returns `db.User`. | Write raw test setup. Always use these helpers. |
| `templates/layouts` | Layout wrappers (`base.templ`, `app.templ`). Shared by all domains. | Put domain-specific markup here. |
| `templates/components` | Reusable UI components: `button.templ`, `card.templ`, `csrf.templ`, `flash.templ`, `input.templ`, `pagination.templ`. | Use BEM classes. Use `data-*` attributes for variants, sizes, and states. |
| `templates/icons` | Icon system. `lucide_gen.go` (generated, do not edit). `icon.go` has `Register()` for custom icons and `Custom()` to render them. | Edit `lucide_gen.go` by hand. It is generated by `cmd/icongen`. |
| `migrations/` | SQL migration files. Boilerplate base uses `NNN_` prefix; user migrations use timestamp `YYYYMMDDHHMMSS_` prefix. Embedded via `embed.go`. | Create migrations without both `-- +goose Up` AND `-- +goose Down` sections. |
| `queries/` | SQLC query files (`.sql`). One file per domain. | Write raw SQL in Go files. Write queries here, then run `sqlc generate`. |
| `cmd/app` | CLI entry point (`main.go`), server startup (`serve.go`), router construction (`router.go`), module registration (`app.go`), migration commands (`migrate.go`), route listing (`routes.go`), seed data (`seed.go`), diagnostics (`doctor.go`), version (`version.go`). Only `app.go` is user-edited; all other files are boilerplate. | Put business logic here. This package wires dependencies and delegates to `internal/`. |
| `cmd/icongen` | Icon code generator. Reads `lucideIcons` slice and writes `templates/icons/lucide_gen.go`. | Edit the generated output. Edit the source `lucideIcons` slice in this file, then run `go run ./cmd/icongen`. |
| `assets/` | Static assets (CSS, JS, images). Embedded and served at `/assets/*`. | Add build tooling for assets. |

---

## 4. How to Add a New Domain (Step-by-Step Checklist)

Follow every step in order. Do not skip any.

1. **Write migration** -- Create `migrations/YYYYMMDDHHMMSS_create_<domain>.sql` with `-- +goose Up` and `-- +goose Down` sections. Use a timestamp prefix (e.g., `20260315143000`). Add indexes on all foreign key columns.

2. **Write queries** -- Create `queries/<domain>.sql` with SQLC annotations. Use `-- name: MethodName :one`, `:many`, `:exec`, or `:execresult`. One file per domain.

3. **Run sqlc generate** -- Execute `sqlc generate` to produce Go code in `internal/db/`. Never edit the generated files.

4. **Create domain package** -- Create `internal/<domain>/` with these files:

   - `handler.go` -- Define `Mount(r chi.Router, deps *server.Deps)` and unexported handler factory functions. Handlers receive `*server.Deps` and follow the four-step pattern (parse -> validate -> query -> render). Use `server.RenderError(w, r, err, deps.IsDev)` for errors.
   - `module.go` -- Export a `Module` var of type `server.Module` with `Name`, `Path`, and `Mount`.

5. **Write templates** -- Create page templates **inside the domain package** (co-located):
   - `internal/<domain>/<domain>_list.templ` -- list/index view
   - `internal/<domain>/<domain>_show.templ` -- single item view
   - `internal/<domain>/<domain>_form.templ` -- create/edit form
   Use `package <domain>` (not `package pages`). Import shared components from `templates/components/` and layouts from `templates/layouts/`. Call templates directly from handlers (same package, no import needed).

6. **Run templ generate** -- Execute `templ generate` to compile templates.

7. **Register module** -- Add one line to `cmd/app/app.go`:
   ```go
   var modules = []server.Module{
       notes.Module,
       <domain>.Module,  // ← add here
   }
   ```

8. **Write tests** -- Create `internal/<domain>/handler_test.go` with table-driven tests. Use `testutil.SetupTestDB(t)` and `httptest`. Create a `setupRouter(deps)` helper that wires routes without auth middleware.

9. **Verify** -- Run `go build ./...` and `go test ./...`. Both must pass.

---

## 5. How to Remove a Domain

1. **Remove from modules** -- Delete the line from `cmd/app/app.go`.
2. **Delete package** -- Delete `internal/<domain>/` (handlers, templates, module, tests — all in one directory).
3. **Delete queries** -- Delete `queries/<domain>.sql`, then run `sqlc generate`.
4. **Handle migration** -- Create a new down migration or use `migrate reset` (dev only).
5. **Verify** -- Run `go build ./...` and `go test ./...`.

---

## 6. How to Add Configuration for a Feature

1. **Add fields** to the `Config` struct in `internal/config/config.go`. Add env var loading in the `Load()` function using the `envOr()` helper.

2. **Add to redacted view** -- Update `Redacted()` in `internal/config/view.go` to include the new fields. Mark sensitive values.

3. **Register schema group** -- Add a new `SchemaGroup` in `internal/config/schema.go` via the `init()` function, or call `config.RegisterGroup()` from an `init()` in the feature package.

4. **Update tests** -- Add test cases in `internal/config/config_test.go`.

---

## 7. How to Add a Background Job

1. **Define the handler function** inside your domain package:
   ```go
   func handleMyJob(ctx context.Context, payload []byte) error {
       var p MyPayload
       if err := json.Unmarshal(payload, &p); err != nil {
           return fmt.Errorf("unmarshalling payload: %w", err)
       }
       // ... do work ...
       return nil
   }
   ```

2. **Register via Module** -- Add to the `Jobs` field in your domain's `Module` var:
   ```go
   var Module = server.Module{
       Name:  "notes",
       Path:  "/notes",
       Mount: Mount,
       Jobs: []server.Job{
           {Type: "my_job", Handler: handleMyJob},
       },
   }
   ```

3. **Enqueue from handlers:**
   ```go
   deps.Queue.Enqueue(r.Context(), "my_job", MyPayload{Field: "value"})
   ```

4. **For recurring jobs**, add to the `Schedules` field:
   ```go
   Schedules: []jobs.Schedule{
       jobs.Daily("my_job", nil),
       jobs.EveryMinutes(30, "cleanup", nil),
   },
   ```

---

## 8. How to Send Email

1. **Build the message:**
   ```go
   msg := email.Message{
       To:      []string{"user@example.com"},
       From:    cfg.Mail.From,
       Subject: "Welcome",
       HTML:    "<h1>Welcome</h1>",
       Text:    "Welcome",
   }
   ```

2. **Send via the mailer:**
   ```go
   if err := deps.Mailer.Send(ctx, msg); err != nil {
       return fmt.Errorf("sending welcome email: %w", err)
   }
   ```

3. **What happens automatically:**
   - Email is stored in the database (via `email.Store`) with status `pending`.
   - In development (`ENV=development`): `DevMailer` logs the email and marks it `sent` immediately. No SMTP.
   - In production: `SMTPMailer` sends via SMTP, then marks `sent` or `failed`.

---

## 9. How to Add Icons

1. **Use an existing Lucide icon:** Call `@icons.IconName("size")` in a templ file, where `IconName` is the PascalCase version of the Lucide name and `size` is a data-size value (e.g., `"sm"`, `"md"`, `"lg"`).
   ```
   @icons.Search("md")
   @icons.ArrowRight("sm")
   @icons.CheckCircle("lg")
   ```

2. **Add a new Lucide icon:** Add a new entry to the `lucideIcons` slice in `cmd/icongen/main.go` with the kebab-case name and SVG inner content. Then run:
   ```
   go run ./cmd/icongen
   ```
   This regenerates `templates/icons/lucide_gen.go`.

3. **Add a custom (non-Lucide) icon:** Call `icons.Register("name", svgInnerContent)` during app initialization, then use `@icons.Custom("name", "md")` in templates.

---

## 10. Conventions

### Error Handling

- Always wrap errors with context: `fmt.Errorf("doing thing: %w", err)`
- Never ignore errors with `_` (except for logging writes and `crypto/rand.Read` in request ID generation).
- User errors (validation failures) -> re-render the form with errors, HTTP 422.
- Not found -> `server.RenderNotFound(w, r)`, HTTP 404.
- Server errors -> `server.RenderError(w, r, err, deps.IsDev)`, HTTP 500. Log the error. Never expose internal details to users.
- Auth required -> `http.Redirect(w, r, "/login", http.StatusSeeOther)`.

### Testing

- Always table-driven with `t.Run(tt.name, func(t *testing.T) { ... })`.
- Use `testutil.SetupTestDB(t)` for any test that touches the database. It returns `(*sql.DB, *db.Queries, cleanup)`. Defer the cleanup function.
- Use `testutil.CreateTestUser(t, q, "alice@example.com")` for user fixtures.
- Use `httptest.NewRecorder()` and `httptest.NewRequest()` for handler tests.
- Test file naming: `<file>_test.go` in the same package (e.g., `internal/notes/handler_test.go`).

### Naming

| Thing | Convention | Example |
|-------|-----------|---------|
| Go files | `snake_case.go` | `body_limit.go`, `request_logger.go` |
| Exported functions | `PascalCase` | `CreateSession`, `FromRequest` |
| Unexported functions | `camelCase` | `envOr`, `generateToken` |
| Handler methods | `handle<Action>` | `handleListPosts`, `handleCreatePost`, `handleShowPost` |
| Domain packages | One package per domain | `internal/posts/handler.go`, `internal/users/handler.go` |
| Migrations | `YYYYMMDDHHMMSS_description.sql` | `20260315143000_create_posts.sql` |
| Templates (pages) | Co-located in domain package | `internal/posts/posts_list.templ`, `posts_show.templ`, `posts_form.templ` |
| Query files | `<domain>.sql` | `queries/posts.sql`, `queries/users.sql` |
| URL patterns | RESTful | `GET /posts`, `GET /posts/{id}`, `POST /posts`, `PUT /posts/{id}`, `DELETE /posts/{id}` |

### Handler Pattern

Every handler follows the same four-step structure:

```go
func handleCreatePost(deps *server.Deps) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 1. Parse input
        r.ParseForm()
        title := r.FormValue("title")
        body := r.FormValue("body")

        // 2. Validate
        v := validate.New()
        v.Required("title", title)
        v.MinLength("body", body, 10)
        if v.HasErrors() {
            w.WriteHeader(http.StatusUnprocessableEntity)
            PostForm(title, body, v.Errors()).Render(r.Context(), w)
            return
        }

        // 3. Query DB
        post, err := deps.Queries.CreatePost(r.Context(), db.CreatePostParams{
            Title: title,
            Body:  body,
        })
        if err != nil {
            server.RenderError(w, r, err, deps.IsDev)
            return
        }

        // 4. Respond
        flash.Set(w, "Post created", "success")
        http.Redirect(w, r, "/posts/"+strconv.FormatInt(post.ID, 10), http.StatusSeeOther)
    }
}
```

### HTMX Pattern

```go
func handleListPosts(deps *server.Deps) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        posts, _ := deps.Queries.ListPosts(r.Context(), ...)

        if server.IsHTMX(r) {
            // HTMX request: render only the partial
            PostListPartial(posts).Render(r.Context(), w)
            return
        }

        // Full page request: render layout + page
        PostList(posts).Render(r.Context(), w)
    }
}
```

- Use `hx-get`, `hx-post`, `hx-target`, `hx-swap` on elements.
- Pagination uses `hx-get` with page query parameter, targeting the list container.
- `server.IsHTMX(r)` checks for the `HX-Request` header (defined in `internal/server/helpers.go`).
- Templates are in the same package as handlers — call them directly without import prefix.

### Template Pattern

- Use `@components.CSRFField()` inside every `<form>` (retrieves token from context via `view.CSRFToken`).
- Use `@components.FlashMessage()` in layouts (retrieves flash from context via `view.GetFlash`).
- Use `data-variant`, `data-size`, and other `data-*` attributes for component styling. Never BEM classes.
- Use `@icons.IconName("size")` for icons.
- Pass validation errors as `validate.Errors` to form templates for inline error display.
- Page templates use `package <domain>` (not `package pages`) and live in the domain directory.

---

## 11. What You Must Never Do

- Use GORM or any ORM.
- Write raw SQL strings in handler files. All queries go in `queries/*.sql` and are accessed via SQLC-generated methods.
- Use CGO or any library that requires CGO.
- Add Node.js, npm, or any JavaScript build tooling.
- Use `html/template`. Always use templ.
- Use BEM class variants. Use `data-*` attributes for component variants, sizes, and states.
- Use goroutines in handlers. Use `internal/jobs` for async work.
- Skip CSRF tokens on form submissions. Every POST/PUT/DELETE form must include `@components.CSRFField()`.
- Edit SQLC-generated files (`internal/db/*.sql.go`, `internal/db/models.go`) by hand. They are overwritten by `sqlc generate`.
- Edit `templates/icons/lucide_gen.go` by hand. It is generated by `cmd/icongen`.
- Use `fmt.Println` or `log.Printf` for application logging. Use the structured logger (`logger.Info`, `logger.Warn`, `logger.Error`).
- Store business logic in `cmd/app`. Put it in `internal/`.
- Use global mutable state. Inject dependencies via constructors and struct fields (exception: `var Module` in domain packages).
- Inline styles or non-Tailwind CSS classes.
- Use `os.*` directly for file storage. Use the `storage.Storage` interface.
- Edit boilerplate-owned files in `cmd/app/` (except `app.go`). Register domains via the `modules` list in `app.go` only.

---

## 12. CLI Commands

```
./app serve              Start the HTTP server (default command)
./app migrate up         Run all pending migrations (default action)
./app migrate down       Rollback the last migration
./app migrate status     Show migration status
./app migrate reset      Drop all tables and re-migrate (dev only, requires ENV=development)
./app routes             List all registered routes
./app seed               Run seed data
./app doctor             Run diagnostic checks
./app version            Show version info
./app help               Show usage
```

---

## 13. Extension Points

These are the exact locations where new code is added:

| What | Where | How |
|------|-------|-----|
| New domain | `cmd/app/app.go` | Add `<domain>.Module` to the `modules` slice. |
| New routes | `internal/<domain>/handler.go` | Define `Mount()` in the domain package; routes are auto-mounted via Module. |
| New middleware | `cmd/app/router.go` | Add `router.Use(...)` in `buildRouter()` (boilerplate change). |
| New job handlers | `internal/<domain>/module.go` | Add to the `Jobs` field of the domain's Module. |
| New scheduled tasks | `internal/<domain>/module.go` | Add to the `Schedules` field of the domain's Module. |
| New seed data | `internal/<domain>/module.go` | Add to the `Seeds` field of the domain's Module. |
| New config fields | `internal/config/config.go` | Add to `Config` struct and `Load()`. Update `view.go` and `schema.go`. |
| New Lucide icons | `cmd/icongen/main.go` | Add to `lucideIcons` slice, then run `go run ./cmd/icongen`. |
| New page templates | `internal/<domain>/` | Create `<domain>_<view>.templ` in the domain package, run `templ generate`. |
| New components | `templates/components/` | Create `<name>.templ`, run `templ generate`. |
| New query files | `queries/` | Create `<domain>.sql`, run `sqlc generate`. |
| New migrations | `migrations/` | Create `YYYYMMDDHHMMSS_description.sql` with `-- +goose Up` and `-- +goose Down`. |

---

## 14. Pre-Commit Checklist

Before committing any change, always run:

```
go build ./...       # Must compile
go test ./...        # Must pass
```

If you added or changed `.templ` files, first run:
```
templ generate
```

If you added or changed `queries/*.sql` files, first run:
```
sqlc generate
```

If you added icons to `cmd/icongen/main.go`, first run:
```
go run ./cmd/icongen
```
