# Omakase Go Boilerplate — Build Plan

## What this is

The starting point for every app Omakase generates. When a user creates a new
project, Omakase clones this repo and builds on top of it.

The philosophy: one binary, one file, one server, zero runtime dependencies.
Ship a Go binary + SQLite database to a $3.29 VPS and you're done.

---

## Repo

```
github.com/omakase-dev/go-boilerplate
Public repo. MIT license.
Pinned to a specific commit tag in production.
Agents always clone from the pinned tag, never from main directly.
```

---

## Stack (fixed, never negotiated)

```
Go 1.22+
SQLite (modernc.org/sqlite — pure Go, no CGO)
SQLC (type-safe SQL → Go structs, no ORM)
Goose v3 (pressly/goose — migrations, supports embed.FS for single-binary)
Templ (type-safe HTML templates, compiled to Go)
HTMX 2.0.4 (pinned version)
Alpine.js 3.14 (pinned version — client-side micro-interactions)
Tailwind CSS v4 (via standalone CLI, no Node)
Chi router (lightweight, stdlib-compatible)
gorilla/csrf (CSRF protection for form submissions)
Air (live reload in development)
```

---

## Project structure

```
.
├── cmd/
│   └── server/
│       └── main.go          ← entry point, wires everything together
├── internal/
│   ├── api/                 ← internal management API (platform endpoints)
│   ├── auth/                ← session management (strategy-agnostic)
│   ├── config/              ← env var loading
│   ├── db/                  ← SQLC generated code + db connection
│   ├── handlers/            ← HTTP handlers (one file per domain)
│   ├── logger/              ← structured JSON logging + ring buffer
│   ├── middleware/          ← session, logging, recovery, request ID, internal key
│   ├── models/              ← domain types (beyond SQLC structs)
│   └── jobs/                ← background job definitions
├── migrations/              ← SQL migration files (Goose)
├── queries/                 ← SQL query files (SQLC input)
├── templates/               ← Templ template files
│   ├── layouts/
│   ├── components/
│   ├── icons/              ← SVG icons as Templ components
│   └── pages/
├── assets/
│   ├── css/
│   │   └── app.css          ← Tailwind source
│   ├── js/
│   │   └── app.js           ← JS entry point (minimal)
│   └── static/              ← fonts, images, favicon
├── public/                  ← compiled assets (git ignored)
├── .air.toml                ← live reload config
├── .sqlc.yaml               ← SQLC config
├── .gitignore               ← ignores public/, bin/, data/, tmp/
├── Makefile                 ← dev commands
├── Dockerfile               ← production image
├── OMAKASE.md               ← rules for the coding agent
└── README.md
```

---

## Why these specific choices

### modernc.org/sqlite (not mattn/go-sqlite3)
No CGO. Pure Go. Cross-compiles trivially. Single binary without needing
a C compiler on the build machine. Critical for Omakase's build pipeline.

### SQLC (not GORM, not sqlx)
Write SQL, get type-safe Go structs back. No ORM magic. Agents write
plain SQL queries that map directly to Go code. Easy to verify, easy to test.
The SQL is readable, the Go is generated — agents only touch SQL.

### Templ (not html/template)
Type-safe, compiled templates. Template errors are compile errors — not
runtime panics. Agents generate Templ files, `templ generate` produces Go.
Components are Go functions — composable, testable, no magic.

### Goose v3 / pressly/goose (not Atlas, not golang-migrate)
Simple SQL migration files. Up and down. `goose up`. That's it.
No DSL. Agents write SQL migrations just like Rails migrations but in SQL.
Uses `embed.FS` so migrations are compiled into the binary — no loose files in prod.

### HTMX + Alpine.js (not React, not Vue)
HTMX handles server-driven interactivity — swap HTML fragments, no SPA needed.
Alpine.js handles client-side micro-interactions — dropdowns, modals, tabs,
toggles, confirm dialogs. Without Alpine, agents write inconsistent vanilla JS
for every UI interaction. Alpine is ~17KB, no build step, and pairs naturally
with HTMX. HTMX owns the server round-trip, Alpine owns the DOM state.

### Tailwind standalone CLI (not Node)
No package.json. No node_modules. Tailwind CLI is a single binary.
Download it, run it, done. The generated app has zero Node dependency.

---

## Core files to build

### 1. main.go — the wire-up

```go
// cmd/server/main.go
package main

func main() {
    cfg  := config.Load()       // from env vars
    db   := db.Connect(cfg)     // SQLite connection + pragmas
    jobs := jobs.NewQueue(db)   // background job queue (SQLite-backed)

    // Run embedded migrations on startup
    goose.SetBaseFS(migrations.FS)
    goose.Up(db, ".")

    router := chi.NewRouter()
    router.Use(middleware.RequestID)
    router.Use(middleware.Logger)
    router.Use(middleware.Recovery)
    router.Use(csrf.Protect([]byte(cfg.CSRFKey)))  // CSRF protection
    router.Use(middleware.Session(db))              // sets ctx user from session cookie

    // Health check — no auth, no middleware
    router.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("ok"))
    })

    // Internal management API — protected by shared secret
    // Used by the Omakase platform to manage jobs, stream logs, check health
    router.Route("/internal", func(r chi.Router) {
        r.Use(middleware.InternalAPIKey(cfg.InternalAPIKey))
        internal.Mount(r, db, jobs, logger)
    })

    // Mount app handlers
    handlers.Mount(router, db, jobs)

    // Serve embedded assets
    router.Handle("/assets/*", http.StripPrefix("/assets/",
        http.FileServer(http.FS(assets.Files))))

    // Graceful shutdown
    srv := &http.Server{Addr: cfg.Addr, Handler: router}
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

    go func() {
        log.Printf("listening on %s", cfg.Addr)
        srv.ListenAndServe()
    }()

    <-ctx.Done()
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    srv.Shutdown(shutdownCtx)
}
```

### 2. db/connection.go — SQLite setup

```go
// internal/db/connection.go
package db

func Connect(cfg config.Config) *sql.DB {
    db, err := sql.Open("sqlite", cfg.DatabasePath)

    // Critical SQLite pragmas — set once, never negotiated
    pragmas := []string{
        "PRAGMA journal_mode=WAL",      // concurrent reads
        "PRAGMA synchronous=NORMAL",    // safe + fast
        "PRAGMA foreign_keys=ON",       // enforce FKs
        "PRAGMA busy_timeout=5000",     // wait 5s on lock
        "PRAGMA cache_size=-20000",     // 20MB cache
        "PRAGMA temp_store=MEMORY",     // temp tables in RAM
    }

    for _, pragma := range pragmas {
        db.Exec(pragma)
    }

    return db
}
```

### 3. config/config.go — env var loading

```go
// internal/config/config.go
package config

type Config struct {
    Addr           string // e.g. ":8080"
    DatabasePath   string // e.g. "./data/app.db"
    CSRFKey        string // 32-byte key for gorilla/csrf
    InternalAPIKey string // shared secret for /internal/* endpoints
    Env            string // "development" | "production"
}

func Load() Config {
    return Config{
        Addr:           envOr("ADDR", ":8080"),
        DatabasePath:   envOr("DATABASE_PATH", "./data/app.db"),
        CSRFKey:        envOr("CSRF_KEY", "change-me-in-production-32bytes!"),
        InternalAPIKey: envOr("INTERNAL_API_KEY", "change-me-in-production"),
        Env:            envOr("ENV", "development"),
    }
}

func envOr(key, fallback string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return fallback
}
```

### 4. assets/embed.go — embedded static files

```go
// assets/embed.go
package assets

import "embed"

//go:embed static/* js/* css/*
var Files embed.FS
```

### 5. migrations/embed.go — embedded migrations

```go
// migrations/embed.go
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
```

### 6. auth/session.go — session management (strategy-agnostic)

Pure Go session layer. No opinion on HOW the user authenticates — only manages
the session AFTER authentication succeeds. The agent adds the auth strategy
(email/password, OAuth, magic link, etc.) on top of this.

```go
// internal/auth/session.go
package auth

type Session struct {
    ID        string
    UserID    int64
    Token     string
    ExpiresAt time.Time
    CreatedAt time.Time
}

func CreateSession(db *sql.DB, userID int64) (*Session, error)
func ValidateSession(db *sql.DB, token string) (*Session, error)
func DeleteSession(db *sql.DB, token string) error
```

The agent picks the auth strategy and builds on top of this:
- Email/password → agent adds HashPassword/CheckPassword + login/register handlers
- OAuth → agent adds OAuth flow that ends with CreateSession()
- Magic link → agent adds email flow that ends with CreateSession()
- API keys → agent adds key validation that ends with CreateSession()

### 7. middleware/session.go — request authentication

```go
// internal/middleware/session.go
func Session(db *sql.DB) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token, _ := r.Cookie("session_token")
            if token != nil {
                session, err := auth.ValidateSession(db, token.Value)
                if err == nil {
                    user, _ := queries.GetUser(r.Context(), db, session.UserID)
                    ctx := context.WithValue(r.Context(), ctxUserKey, user)
                    r = r.WithContext(ctx)
                }
            }
            next.ServeHTTP(w, r)
        })
    }
}

func CurrentUser(r *http.Request) *db.User {
    user, _ := r.Context().Value(ctxUserKey).(*db.User)
    return user
}

func RequireAuth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if CurrentUser(r) == nil {
            http.Redirect(w, r, "/login", http.StatusSeeOther)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

### 8. jobs/queue.go — background jobs (SQLite-backed)

No Redis. No external queue. SQLite is the queue.

```go
// internal/jobs/queue.go
package jobs

type Job struct {
    ID         int64
    Type       string
    Payload    []byte
    Status     string    // pending | running | done | failed
    Attempts   int
    RunAt      time.Time
    CreatedAt  time.Time
}

type Queue struct {
    db *sql.DB
}

func (q *Queue) Enqueue(jobType string, payload any) error
func (q *Queue) Process(ctx context.Context) error  // runs in goroutine
func (q *Queue) Register(jobType string, handler JobHandler)

// Management methods — used by internal API
func (q *Queue) List(status string, limit, offset int) ([]Job, error)
func (q *Queue) Get(id int64) (*Job, error)
func (q *Queue) Retry(id int64) error    // resets status to pending, increments attempts
func (q *Queue) Cancel(id int64) error   // sets status to cancelled
func (q *Queue) Stats() (map[string]int, error)  // {"pending": 5, "failed": 2, ...}
```

Workers run in goroutines started from main.go. No separate process needed.
Everything in one binary.

### 9. internal/logger.go — structured logging with ring buffer

Structured JSON logging to stdout + ring buffer for SSE streaming.
The platform connects to `/internal/logs` and gets real-time logs.

```go
// internal/logger/logger.go
package logger

type Logger struct {
    buffer *RingBuffer  // holds last N log entries in memory
}

type Entry struct {
    Level   string    `json:"level"`    // info | warn | error
    Msg     string    `json:"msg"`
    Fields  map[string]any `json:"fields,omitempty"`
    Time    time.Time `json:"ts"`
}

func New(bufferSize int) *Logger              // default 1000 entries
func (l *Logger) Info(msg string, fields ...any)
func (l *Logger) Warn(msg string, fields ...any)
func (l *Logger) Error(msg string, fields ...any)
func (l *Logger) Subscribe() <-chan Entry     // SSE consumers call this
func (l *Logger) Unsubscribe(ch <-chan Entry)
```

Every log call: writes JSON to stdout AND pushes to the ring buffer.
SSE subscribers receive entries from the buffer channel in real-time.
When a subscriber disconnects, it's cleaned up automatically.

### 10. internal/api.go — platform management API

Internal endpoints for the Omakase platform. Protected by `X-Internal-Key` header.
These are NOT user-facing — the platform calls them to manage the deployed app.

```go
// internal/api/api.go
package api

func Mount(r chi.Router, db *sql.DB, jobs *jobs.Queue, log *logger.Logger) {
    r.Get("/health", handleHealth(db))       // detailed health: db, uptime, memory, Go version
    r.Get("/jobs", handleListJobs(jobs))      // ?status=failed&limit=20&offset=0
    r.Get("/jobs/{id}", handleGetJob(jobs))   // job detail + error message
    r.Post("/jobs/{id}/retry", handleRetryJob(jobs))
    r.Delete("/jobs/{id}", handleCancelJob(jobs))
    r.Get("/jobs/stats", handleJobStats(jobs))  // {"pending": 5, "running": 1, "failed": 2}
    r.Get("/logs", handleLogStream(log))      // SSE: real-time structured log stream
}
```

```go
// internal/middleware/internal_key.go
func InternalAPIKey(key string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if r.Header.Get("X-Internal-Key") != key {
                http.Error(w, "unauthorized", http.StatusUnauthorized)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

SSE log streaming endpoint:

```go
// internal/api/logs.go
func handleLogStream(log *logger.Logger) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/event-stream")
        w.Header().Set("Cache-Control", "no-cache")
        w.Header().Set("Connection", "keep-alive")

        ch := log.Subscribe()
        defer log.Unsubscribe(ch)

        for {
            select {
            case entry := <-ch:
                data, _ := json.Marshal(entry)
                fmt.Fprintf(w, "data: %s\n\n", data)
                w.(http.Flusher).Flush()
            case <-r.Context().Done():
                return
            }
        }
    }
}
```

### 11. Base migrations

```sql
-- migrations/001_create_users.sql
-- +goose Up
CREATE TABLE users (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    email      TEXT NOT NULL UNIQUE,
    name       TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_users_email ON users(email);
-- NOTE: Auth-strategy columns (password_hash, oauth_provider, etc.)
-- are added by the agent based on the chosen auth strategy.

-- +goose Down
DROP TABLE IF EXISTS users;

-- migrations/002_create_sessions.sql
-- +goose Up
CREATE TABLE sessions (
    id         TEXT PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token      TEXT NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_sessions_token ON sessions(token);
CREATE INDEX idx_sessions_user_id ON sessions(user_id);

-- +goose Down
DROP TABLE IF EXISTS sessions;

-- migrations/003_create_jobs.sql
-- +goose Up
CREATE TABLE jobs (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    type       TEXT NOT NULL,
    payload    TEXT NOT NULL DEFAULT '{}',
    status     TEXT NOT NULL DEFAULT 'pending',
    attempts   INTEGER NOT NULL DEFAULT 0,
    run_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_jobs_status_run_at ON jobs(status, run_at);

-- +goose Down
DROP TABLE IF EXISTS jobs;
```

Convention: every migration must include both `+goose Up` and `+goose Down`.

### 12. Base Templ templates

```go
// templates/layouts/base.templ
package layouts

templ Base(title string) {
    <!DOCTYPE html>
    <html lang="en" class="h-full">
    <head>
        <meta charset="UTF-8"/>
        <meta name="viewport" content="width=device-width, initial-scale=1.0"/>
        <title>{ title }</title>
        <link rel="stylesheet" href="/assets/css/app.css"/>
        <script src="/assets/js/htmx.min.js"></script>
        <script src="/assets/js/alpine.min.js" defer></script>
        <script src="/assets/js/app.js" defer></script>
    </head>
    <body class="h-full bg-background text-text antialiased"
          hx-boost="true">
        { children... }
    </body>
    </html>
}
```

Note: Templ component files (button.templ, input.templ, card.templ, flash.templ)
are thin wrappers around HTML + data attributes. The styling lives in CSS (see
Design System section above). Templ components just map Go args to data attributes.
No conditional class logic, no `templ.KV()` for styling.

### 13. Handler pattern

Every handler follows the same pattern. Agents extend this.

```go
// internal/handlers/posts.go
package handlers

type PostsHandler struct {
    db      *sql.DB
    queries *db.Queries
}

func (h *PostsHandler) Index(w http.ResponseWriter, r *http.Request) {
    posts, err := h.queries.ListPosts(r.Context(), currentUser(r).ID)
    if err != nil {
        renderError(w, r, err)
        return
    }

    // HTMX partial or full page
    if isHTMX(r) {
        pages.PostsList(posts).Render(r.Context(), w)
        return
    }
    layouts.App(pages.PostsIndex(posts)).Render(r.Context(), w)
}

func (h *PostsHandler) Create(w http.ResponseWriter, r *http.Request) {
    r.ParseForm()
    title := r.FormValue("title")
    body  := r.FormValue("body")

    if title == "" {
        renderFormError(w, r, "title is required")
        return
    }

    post, err := h.queries.CreatePost(r.Context(), db.CreatePostParams{
        UserID: currentUser(r).ID,
        Title:  title,
        Body:   body,
    })
    if err != nil {
        renderError(w, r, err)
        return
    }

    setFlash(w, "Post created")
    http.Redirect(w, r, "/posts/"+post.ID, http.StatusSeeOther)
}
```

### 14. Makefile — dev commands

```makefile
# Makefile
.PHONY: dev build generate lint test

dev:
	air  # live reload via .air.toml

build:
	templ generate
	tailwindcss -i assets/css/app.css -o public/css/app.css --minify
	go build -o bin/server ./cmd/server

generate:
	templ generate
	sqlc generate

lint:
	golangci-lint run

test:
	go test ./...
```

Note: Migrations run automatically on startup via embedded Goose.
No separate `make migrate` needed — the binary is self-contained.

### 15. Dockerfile — production image

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN templ generate
RUN tailwindcss -i assets/css/app.css -o public/css/app.css --minify
RUN go build -o bin/server ./cmd/server

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/bin/server .
EXPOSE 8080
CMD ["./server"]
```

Single binary. Assets and migrations are embedded via `//go:embed`.
No loose files to copy. No runtime. No Ruby. No gems. Nothing.

---

## OMAKASE.md (rules for the coding agent)

```markdown
# Omakase Go App Rules

## Stack — never deviate
- Go 1.22+
- SQLite via modernc.org/sqlite (no CGO, no Postgres, no MySQL)
- SQLC for database access (no GORM, no raw sql.Query strings in handlers)
- Goose for migrations (plain SQL files only)
- Templ for templates (no html/template, no string concatenation)
- HTMX for server interactivity + Alpine.js for client-side UI (no React, no Vue, no Svelte)
- Tailwind CSS (no other CSS frameworks)
- Chi router (no Gin, no Echo, no Fiber)
- Jobs queue via SQLite jobs table (no Redis, no external queue)

## File conventions
- One handler file per domain: handlers/posts.go, handlers/users.go
- Queries in queries/ as .sql files, never inline in Go
- Templates in templates/ as .templ files, never html/template
- Migrations named: 001_description.sql, 002_description.sql
- Component styling in assets/css/app.css using data-* attribute selectors
- Component variants via data-variant, sizes via data-size, states via data-*
- No global variables except db connection and job queue

## What you must do
- Write SQL in queries/*.sql files, run sqlc generate after
- Write migrations in migrations/*.sql with +goose Up/Down comments
- Run templ generate after adding or changing .templ files
- Add indexes to all foreign key columns
- Use transactions for multi-step writes
- Check errors — never use _ for error returns
- Include CSRF token in all forms (gorilla/csrf provides the template helper)
- Run go test ./... before calling git_commit

## What you must never do
- Use GORM or any ORM
- Write raw SQL strings in handler files
- Use CGO or any library that requires CGO
- Add Node.js, npm, or any JavaScript build tooling
- Use html/template (always use templ)
- Inline styles or non-Tailwind CSS classes
- BEM class variants (use data-* attributes for component variants/sizes/states)
- Use goroutines in handlers (use jobs for async work)
- Skip CSRF tokens on form submissions

## Handler pattern
Every handler: parse input → validate → query db → render template
If HTMX request (isHTMX(r)): render partial only
If full request: render layout + page

## Error handling
User errors (validation): render form with error message, 422
Not found: render 404 page
Server errors: log error, render 500 page, never expose details
Auth required: redirect to /login

## When done
Run: go build ./... (must pass)
Run: go test ./... (must pass)
Then call git_commit with summary
```

---

## Design system (Tailwind v4 tokens + data-attribute components)

Components use `data-*` attributes for variants, sizes, and states.
No BEM. No conditional class logic in Templ. The agent writes self-documenting
HTML attributes and gets sensible defaults when attributes are omitted.

```css
/* assets/css/app.css */
@import "tailwindcss";

@theme {
  /* Semantic colors — agents think "primary" not "orange-500" */
  --color-background:    #09090b;
  --color-surface:       #18181b;
  --color-surface-hover: #27272a;
  --color-border:        #27272a;
  --color-ring:          #f97316;

  --color-text:          #f4f4f5;
  --color-text-muted:    #a1a1aa;
  --color-text-faint:    #71717a;

  --color-primary-500:   #f97316;
  --color-primary-600:   #ea580c;
  --color-primary-700:   #c2410c;

  --color-error-500:     #ef4444;
  --color-error-600:     #dc2626;
  --color-error-700:     #b91c1c;

  --color-success-500:   #22c55e;
  --color-success-600:   #16a34a;

  --color-warning-500:   #eab308;
  --color-warning-600:   #ca8a04;

  /* Radius */
  --radius-sm:   4px;
  --radius-md:   8px;
  --radius-lg:   12px;
  --radius-xl:   16px;
}
```

### Component CSS pattern

Every component follows the same structure: base class → size via `data-size` →
variant via `data-variant` → state modifiers via `data-*`. Omitted attributes
fall back to defaults via `:not([data-attr])`.

```css
/* ============================================================
 * Button Component
 * Usage: <button class="btn" data-variant="primary" data-size="md">Click</button>
 *
 * Variants: primary (default), secondary, destructive, ghost, outline
 * Sizes: sm, md (default), lg
 * Modifiers: data-type="icon", data-loading="true", data-full-width="true"
 * ============================================================ */

/* --- Base --- */
.btn {
  @apply inline-flex items-center justify-center gap-2 whitespace-nowrap;
  @apply rounded-lg font-medium transition-colors duration-150;
  @apply focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2;
  @apply disabled:opacity-50 disabled:pointer-events-none;
  @apply cursor-pointer select-none;
}

.btn > svg, .btn > i { @apply pointer-events-none shrink-0; }

/* --- Sizes --- */
.btn[data-size="sm"]                  { @apply h-9 px-3 text-sm; }
.btn[data-size="md"], .btn:not([data-size]) { @apply h-10 px-4 text-sm; }
.btn[data-size="lg"]                  { @apply h-11 px-6 text-base; }

/* Icon-only buttons */
.btn[data-size="sm"][data-type="icon"]                              { @apply w-9 h-9 p-0; }
.btn[data-size="md"][data-type="icon"], .btn[data-type="icon"]:not([data-size]) { @apply w-10 h-10 p-0; }
.btn[data-size="lg"][data-type="icon"]                              { @apply w-11 h-11 p-0; }

/* Icon sizing */
.btn[data-size="sm"] > svg, .btn[data-size="sm"] > i               { @apply w-4 h-4; }
.btn[data-size="md"] > svg, .btn:not([data-size]) > svg,
.btn[data-size="md"] > i,  .btn:not([data-size]) > i               { @apply w-5 h-5; }
.btn[data-size="lg"] > svg, .btn[data-size="lg"] > i               { @apply w-5 h-5; }

/* --- Variants --- */
.btn[data-variant="primary"], .btn:not([data-variant]) {
  @apply bg-primary-500 text-white hover:bg-primary-600 active:bg-primary-700;
}
.btn[data-variant="secondary"] {
  @apply border border-border bg-surface text-text-muted hover:bg-surface-hover active:bg-zinc-700;
}
.btn[data-variant="destructive"] {
  @apply bg-error-500 text-white hover:bg-error-600 active:bg-error-700;
}
.btn[data-variant="ghost"] {
  @apply bg-transparent text-text-muted hover:bg-surface-hover active:bg-zinc-700;
}
.btn[data-variant="outline"] {
  @apply border border-primary-500 bg-transparent text-primary-500 hover:bg-primary-500/10 active:bg-primary-500/20;
}

/* --- States --- */
.btn[data-loading="true"]     { @apply cursor-wait opacity-70; }
.btn[data-full-width="true"]  { @apply w-full; }


/* ============================================================
 * Input Component
 * Usage: <input class="input" data-size="md" />
 *        <input class="input" data-error="true" />
 *
 * Sizes: sm, md (default), lg
 * Modifiers: data-error="true"
 * ============================================================ */

.input {
  @apply w-full bg-surface border border-border rounded-lg text-text outline-none
         transition-colors focus:border-ring;
}

.input[data-size="sm"]                      { @apply h-9 px-3 text-sm; }
.input[data-size="md"], .input:not([data-size]) { @apply h-10 px-3 text-sm; }
.input[data-size="lg"]                      { @apply h-11 px-4 text-base; }

.input[data-error="true"]                   { @apply border-error-500 focus:border-error-500; }


/* ============================================================
 * Card Component
 * Usage: <div class="card">...</div>
 *        <div class="card" data-padding="none">...</div>
 * ============================================================ */

.card { @apply bg-surface border border-border rounded-xl p-6; }
.card[data-padding="none"] { @apply p-0; }
.card[data-padding="sm"]   { @apply p-4; }


/* ============================================================
 * Badge Component
 * Usage: <span class="badge" data-variant="success">Active</span>
 *
 * Variants: default, success, error, warning
 * ============================================================ */

.badge { @apply inline-flex px-2 py-0.5 rounded-md text-xs font-medium; }

.badge:not([data-variant]),
.badge[data-variant="default"] { @apply bg-zinc-800 text-text-muted; }
.badge[data-variant="success"] { @apply bg-success-500/10 text-success-500; }
.badge[data-variant="error"]   { @apply bg-error-500/10 text-error-500; }
.badge[data-variant="warning"] { @apply bg-warning-500/10 text-warning-500; }


/* ============================================================
 * Flash / Toast Component
 * Usage: <div class="flash" data-variant="success">...</div>
 *
 * Variants: success (default), error, warning, info
 * Typically used with Alpine.js for auto-dismiss (see flash.templ)
 * ============================================================ */

.flash {
  @apply px-4 py-3 rounded-lg text-sm font-medium border;
}

.flash:not([data-variant]),
.flash[data-variant="success"] { @apply bg-success-500/10 border-success-500/20 text-success-500; }
.flash[data-variant="error"]   { @apply bg-error-500/10 border-error-500/20 text-error-500; }
.flash[data-variant="warning"] { @apply bg-warning-500/10 border-warning-500/20 text-warning-500; }
.flash[data-variant="info"]    { @apply bg-primary-500/10 border-primary-500/20 text-primary-500; }


/* ============================================================
 * Label
 * Usage: <label class="label">Email</label>
 * ============================================================ */

.label { @apply block text-sm text-text-muted mb-1.5; }


/* ============================================================
 * Icon (SVG sizing)
 * Usage: <svg class="icon" data-size="md">...</svg>
 *
 * Sizes: sm, md (default), lg, xl
 * Icons inherit current text color via stroke="currentColor"
 * ============================================================ */

.icon { @apply shrink-0; }

.icon[data-size="sm"]                     { @apply w-4 h-4; }
.icon[data-size="md"], .icon:not([data-size]) { @apply w-5 h-5; }
.icon[data-size="lg"]                     { @apply w-6 h-6; }
.icon[data-size="xl"]                     { @apply w-8 h-8; }


/* ============================================================
 * Shared utilities
 * ============================================================ */

.error-text { @apply text-sm text-error-500 mt-1; }
```

### How the agent uses these in Templ

Templates stay clean — just HTML with data attributes, no conditional class logic:

```go
// Simple button — defaults to primary/md
templ SubmitButton(label string) {
    <button class="btn" type="submit">{ label }</button>
}

// Button with options — attributes map directly
templ Button(label, variant, size string) {
    <button class="btn" data-variant={variant} data-size={size}>{ label }</button>
}

// Input with error state
templ FormField(label, name, value, errMsg string) {
    <label class="label">{ label }</label>
    <input class="input"
           name={name}
           value={value}
           if errMsg != "" {
               data-error="true"
           } />
    if errMsg != "" {
        <p class="error-text">{ errMsg }</p>
    }
}

// Flash uses Alpine for auto-dismiss
templ Flash(message, variant string) {
    <div class="flash"
         data-variant={variant}
         x-data="{ show: true }"
         x-show="show"
         x-init="setTimeout(() => show = false, 4000)"
         x-transition>
        { message }
    </div>
}
```

### Icons — inline SVGs as Templ components

No icon font, no CDN, no external dependency. Each icon is a Templ function.
Source SVGs from Lucide (https://lucide.dev) or Heroicons (https://heroicons.com).
The agent copies the SVG path data into a Templ function when it needs an icon.

```go
// templates/icons/icons.templ
package icons

// Each icon: inline SVG, inherits text color, sized via data-size
templ Plus(size string) {
    <svg class="icon" data-size={size} xmlns="http://www.w3.org/2000/svg"
         fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15"/>
    </svg>
}

templ ChevronDown(size string) {
    <svg class="icon" data-size={size} xmlns="http://www.w3.org/2000/svg"
         fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" d="m19.5 8.25-7.5 7.5-7.5-7.5"/>
    </svg>
}

// Agent adds more icons as needed — one function per icon
```

Usage in templates:

```go
// Icon inside a button
<button class="btn">
    @icons.Plus("sm")
    New Post
</button>

// Standalone icon
@icons.ChevronDown("md")
```

### Fonts — system font stack

Default: no font files. Tailwind v4 uses `system-ui` out of the box — zero weight,
zero files, works everywhere.

If the user wants a custom font, the agent:
1. Downloads `.woff2` files to `assets/static/fonts/`
2. Adds `@font-face` to `app.css`
3. Files get embedded via `//go:embed` — still a single binary

---

## Build order for the coding agent

Tell Claude Code: "Read this file and build the boilerplate in this exact order."

### Step 1 — Project skeleton
```
go mod init github.com/omakase-dev/go-boilerplate
Install dependencies (go get)
Create directory structure
Create .gitignore (public/, bin/, data/, tmp/, *.db)
Create Makefile
Create .sqlc.yaml
Write internal/config/config.go
Write assets/embed.go (//go:embed for static files)
Write migrations/embed.go (//go:embed for .sql files)
Create .air.toml — must be configured to:
  1. Watch .templ files → run `templ generate`
  2. Watch .sql files in queries/ → run `sqlc generate`
  3. Watch .go files → rebuild and restart server
```

### Step 2 — Database layer
```
Write db/connection.go with SQLite pragmas
Write 3 base migrations (users, sessions, jobs)
Write base SQL queries for users + sessions
Run sqlc generate → verify Go structs generated
Run goose up → verify tables created
```

### Step 3 — Session + middleware layer
```
Write internal/auth/session.go (CreateSession, ValidateSession, DeleteSession — no auth strategy)
Write internal/middleware/session.go (reads session cookie, sets user in context, RequireAuth)
Write internal/middleware/request_id.go
Write internal/middleware/logger.go
Write internal/middleware/recovery.go
```

### Step 4 — Jobs layer
```
Write internal/jobs/queue.go
Write internal/jobs/worker.go
Verify job enqueue + process works with go test
```

### Step 5 — Logger + internal management API
```
Write internal/logger/logger.go (structured JSON to stdout + ring buffer + subscribe/unsubscribe)
Write internal/api/api.go (Mount function with all /internal/* routes)
Write internal/api/health.go (detailed health: db ping, uptime, memory, Go version)
Write internal/api/jobs.go (list, get, retry, cancel, stats)
Write internal/api/logs.go (SSE streaming from ring buffer)
Write internal/middleware/internal_key.go (X-Internal-Key header check)
Wire logger into middleware.Logger (replace default log with structured logger)
Verify: /internal/health returns JSON, /internal/jobs/stats returns counts
```

### Step 6 — Templates
```
Write templates/layouts/base.templ
Write templates/layouts/app.templ (with sidebar)
Write templates/components/flash.templ
Write templates/components/button.templ
Write templates/components/input.templ
Write templates/components/card.templ
Write templates/icons/icons.templ (starter set: Plus, ChevronDown, X, Check, AlertCircle)
Write templates/pages/error_404.templ
Write templates/pages/error_500.templ
Run templ generate → verify Go files produced
```

### Step 7 — Welcome page + route mounting
```
Write templates/pages/welcome.templ (dev-only welcome screen, like Rails "Yay! You're on Rails!")
  — shows: server running confirmation, stack versions, link to /healthz
  — displays: "This page is replaced when the agent generates your app"
Write internal/handlers/welcome.go (serves / in development only)
Wire up cmd/server/main.go with all middleware and handlers
Mount /, /healthz, /internal/*, and /assets/*
Verify: app starts, / renders welcome page, /healthz returns 200
NOTE: Auth handlers + pages (login, register, OAuth callbacks) are NOT in the
boilerplate — the agent generates these based on the user's chosen auth strategy.
The agent replaces the welcome route with the app's actual root route.
```

### Step 8 — Asset pipeline
```
Write assets/css/app.css with design tokens + base components
Download tailwindcss standalone CLI
Run: tailwindcss -i assets/css/app.css -o public/css/app.css
Download htmx.min.js to assets/js/
Download alpine.min.js to assets/js/
Write assets/js/app.js (minimal: flash dismiss, form states)
```

### Step 9 — Dockerfile
```
Write Dockerfile (multi-stage build, single binary output)
NOTE: config/config.go was already created in Step 1.
NOTE: Litestream (SQLite replication) is a deployment concern — the agent
adds it when deploying, not during scaffolding.
```

### Step 10 — Verification
```
go build ./...       must pass
go test ./...        must pass
make dev             app starts on localhost:8080
/                    renders welcome page (dev-only, replaced by agent)
/healthz             returns 200 (simple health check)
/internal/health     returns detailed JSON (with X-Internal-Key header)
/internal/jobs/stats returns job counts (with X-Internal-Key header)
/internal/logs       streams SSE log entries (with X-Internal-Key header)
/internal/*          returns 401 without valid X-Internal-Key
make build           produces single binary
docker build .       produces working image
NOTE: Full auth flow (login/register/OAuth) is verified after the agent
adds the chosen auth strategy, not as part of the boilerplate verification.
```

---

## What success looks like

At the end of Step 10:

```bash
git clone github.com/omakase-dev/go-boilerplate myapp
cd myapp
make dev
# → localhost:8080 running (migrations run automatically on startup)
# → welcome page at / (dev-only, replaced by agent)
# → session layer ready (CreateSession/ValidateSession/DeleteSession)
# → RequireAuth middleware works (redirects to /login)
# → CSRF protection active on all form submissions
# → jobs queue ready (enqueue + process)
# → /healthz returns 200
# → /internal/* management API ready (health, jobs, logs — protected by key)
# → structured JSON logging to stdout + SSE streaming
# → single `make build` produces one binary
# → `docker build` produces deployable image
```

This is the foundation. Every app Omakase generates starts here.
The agent's first task after cloning: wire in the auth strategy the user chose.
Then add features on top — they never change the foundation.
