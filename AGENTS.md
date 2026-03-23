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
| CSS | Tailwind CSS | No other CSS frameworks. No `@apply`. Use inline Tailwind classes in templ components. |
| Background jobs | SQLite-backed job queue (`internal/jobs`) | No Redis, no external queue. |
| CSRF | `gorilla/csrf` | Token injected into context via middleware. |
| Sessions | Cookie + SQLite (`internal/auth`) | Strategy-agnostic session management. |
| Build tooling | Go + Tailwind CLI | No Node.js, no npm. Dev uses Tailwind browser CDN (cached locally). Prod uses Tailwind CLI. |

---

## 2. Design Philosophy -- Deep Modules

> From John Ousterhout's *A Philosophy of Software Design*: **The best modules are deep — they hide significant complexity behind a simple interface.**

Every package, function, and type must justify its existence by providing a **simple interface that hides meaningful implementation complexity**. This is the central design principle for all code in this project.

### The Rule

```
Module Depth = (Complexity Hidden) / (Interface Exposed)
```

A **deep module** has a small, simple public API that hides substantial logic — parsing, validation, state management, external API calls, error handling, retries, caching. A **shallow module** exposes an interface almost as complex as its implementation — it adds indirection without absorbing complexity. **Do not create shallow modules.**

### Deep vs. Shallow in Go

```go
// ✅ DEEP: Simple call, complex implementation hidden inside.
// Internally: builds HTTP client, paginates API, parses response,
// normalizes data, handles retries, updates multiple fields.
func (c *Competitor) Scrape(ctx context.Context, q *db.Queries) error {
    data, err := fetchAndParse(c.URL)
    if err != nil {
        return fmt.Errorf("scraping %s: %w", c.URL, err)
    }
    return q.UpdateCompetitor(ctx, db.UpdateCompetitorParams{
        ID:           c.ID,
        Name:         data.Name,
        Pricing:      data.Pricing,
        Features:     data.Features,
        LastScrapedAt: sql.NullTime{Time: time.Now(), Valid: true},
    })
}

// ❌ SHALLOW: Wrapper that just delegates — adds a layer without hiding anything.
func ScrapeCompetitor(ctx context.Context, q *db.Queries, c *Competitor) error {
    return c.Scrape(ctx, q) // Why does this function exist?
}

// ❌ SHALLOW: Function that does almost nothing.
func UpdateCompetitorName(ctx context.Context, q *db.Queries, id int64, name string) error {
    return q.UpdateCompetitorName(ctx, db.UpdateCompetitorNameParams{ID: id, Name: name})
    // This is just the SQLC call with a worse interface.
}
```

### Decision Framework

Before creating a new package, type, or exported function, answer:

1. **What complexity does it hide?** If the answer is "not much" or "it just delegates," don't create it.
2. **Is the interface simpler than the implementation?** If the public API has as many concepts as the internals, the module is shallow.
3. **Does it have a reason to change independently?** If it always changes in lockstep with another module, merge them.
4. **Can I name it with a specific noun/verb?** Vague names (`Manager`, `Handler`, `Processor`, `Service`, `Utils`) usually signal shallow design.

### Where Depth Belongs

| Layer | Deep (✅) | Shallow (❌) |
|-------|-----------|-------------|
| **Domain function** | `SyncIntegration()` — hides HTTP client, pagination, parsing, retry, error recovery | `WrapQuery()` — trivial wrapper around a SQLC call |
| **Package** | `internal/jobs` — hides SQLite queue, polling, retry, scheduling behind `Enqueue()` / `Process()` | `internal/utils` — grab-bag of unrelated helpers |
| **Middleware** | `SessionMiddleware` — hides cookie parsing, DB lookup, expiry, context injection | A middleware that just calls `next.ServeHTTP(w, r)` with one header set |
| **Handler** | Thin — delegates to deep domain functions (handlers are *supposed* to be shallow; depth lives in domain logic) | Fat handler with business logic that should be in a domain function |
| **Job handler** | Thin wrapper calling a deep domain function (correct use of a shallow layer — infrastructure glue) | Job handler containing all the business logic (inverts the depth) |

### Information Hiding Checklist

When designing a function or package's public interface, hide:

- **Implementation details** — Callers shouldn't know *how* you fetch, parse, or sync
- **Error handling & retries** — Callers call one function; the module handles failures
- **Data format transformations** — Accept domain types, return domain types
- **External service protocols** — HTTP details, API pagination, auth tokens stay internal
- **Performance optimizations** — Caching, batch processing, connection pooling stay internal
- **Temporal coupling** — "Call A before B" should be enforced internally, not by the caller

```go
// ✅ DEEP: Hides all the above. Caller just writes: integration.Sync(ctx, q)
func (i *Integration) Sync(ctx context.Context, q *db.Queries) error {
    client := i.buildAuthenticatedClient()      // hides auth protocol
    data, err := client.FetchAllPages()          // hides pagination
    if err != nil {
        return fmt.Errorf("fetching data: %w", err)
    }
    results := processInBatches(data)            // hides batching strategy
    if err := cacheResults(ctx, results); err != nil { // hides caching layer
        return fmt.Errorf("caching results: %w", err)
    }
    return q.UpdateIntegrationSyncedAt(ctx, db.UpdateIntegrationSyncedAtParams{
        ID: i.ID, LastSyncedAt: sql.NullTime{Time: time.Now(), Valid: true},
    })
}
```

### Anti-Patterns to Reject

**1. Pass-Through Functions**
```go
// ❌ Function that just calls another function with the same signature.
func CreateUser(ctx context.Context, q *db.Queries, p db.CreateUserParams) (db.User, error) {
    return q.CreateUser(ctx, p)
}
// Just call q.CreateUser directly.
```

**2. Premature Extraction**
```go
// ❌ Package extracted for a single use case that only one domain will ever need.
package nameutil

func Normalize(name string) string {
    return strings.TrimSpace(strings.Title(name))
}
// Just inline this where it's used. Extract when a second caller needs it.
```

**3. Needless Indirection Layers**
```go
// ❌ Repository pattern wrapping SQLC — SQLC IS the repository.
type UserRepository struct{ q *db.Queries }
func (r *UserRepository) Find(ctx context.Context, id int64) (db.User, error) {
    return r.q.GetUser(ctx, id)
}
// This adds a layer that hides nothing. SQLC already provides this interface.
```

**4. Config Objects for Simple Cases**
```go
// ❌ Over-engineered configuration.
type ScrapingConfig struct {
    Timeout time.Duration
    Retries int
}
// Just use function parameters or constants until complexity warrants a struct.
```

**5. Shallow Packages**
```go
// ❌ Package with one exported function that does one trivial thing.
package slugify

func Slugify(s string) string {
    return strings.ToLower(strings.ReplaceAll(s, " ", "-"))
}
// Put this in the package that uses it. A whole package for one liner is shallow.
```

### The Litmus Test

Before creating any new package, type, or exported function:

> **"Does this hide more complexity than it introduces?"**
>
> If yes → create it. If no → inline it, merge it, or delete it.

---

## 3. Architecture Overview

### File Ownership

The codebase separates **boilerplate** (infrastructure, upgradeable) from **user** (domain logic, never overwritten). See `.omakase.yaml` for the full manifest.

- **Boilerplate files** (`cmd/app/*.go` except `app.go`, `internal/server/`, `internal/middleware/`, etc.) — safe to overwrite during upgrades.
- **User files** (`cmd/app/app.go`, `internal/<domain>/`, `queries/`, user migrations) — never touched by upgrades.
- **`cmd/app/app.go`** — the ONE file in `cmd/app/` that users edit. Contains the `modules` list.

### Module Pattern

Each domain exports a `server.Module` var describing its routes, jobs, seeds. The boilerplate iterates the `modules` list in `app.go` for all registration.

```go
// internal/<domain>/module.go
var Module = server.Module{
    Name:  "<domain>",
    Path:  "/<domain>",
    Mount: Mount,
}
```

```go
// cmd/app/app.go — the ONLY file users edit in cmd/app/
var modules = []server.Module{
    <domain>.Module,
}
```

### Co-Located Templates

Page templates live inside their domain package, not in a separate `templates/pages/` directory. Shared layouts and components stay in `templates/`.

```
internal/<domain>/
├── handler.go              ← handlers
├── handler_test.go         ← tests
├── module.go               ← Module var
├── <domain>_list.templ     ← page templates (co-located)
├── <domain>_show.templ
└── <domain>_form.templ

templates/
├── layouts/            ← shared (boilerplate)
├── components/         ← shared (boilerplate)
└── icons/              ← shared (boilerplate)
```

---

## 4. Package Responsibility Map

| Package | Owns | Never |
|---------|------|-------|
| `internal/config` | Env loading (`Load()`), `.env` file parsing (`LoadDotEnv()`), config schema (`schema.go`), redacted config view (`view.go`) | Store runtime state here. |
| `internal/db` | SQLC-generated code (`*.sql.go`), DB connection (`connection.go`), generated models (`models.go`) | Edit generated files (`*.sql.go`, `models.go`) by hand. They are overwritten by `sqlc generate`. |
| `internal/auth` | Session create / validate / delete, expired session cleanup | Add authentication strategy here. This package is strategy-agnostic -- the agent adds login/signup logic in domain handlers. |
| `internal/middleware` | HTTP middleware chain. Each middleware lives in its own file: `request_id.go`, `request_logger.go`, `recovery.go`, `body_limit.go`, `session.go`, `csrf_context.go`, `flash.go`, `internal_key.go` | Combine multiple middlewares into one file. |
| `internal/server` | Shared HTTP infrastructure: `Deps` struct (dependency injection), `Module` type (domain registration), `IsHTMX()`, `RenderError()`, `RenderNotFound()`, `HandleNotFound()`, `HandleWelcome()`. Co-located templates: `welcome.templ`, `error_*.templ`. | Import domain packages from here. Server is a leaf dependency — domains import it, not the other way around. |
| `internal/<domain>` | User domain packages. Each exports a `Module` var, `Mount()` + CRUD handlers, co-located page templates. Follow the conventions in Section 4. | Mix domains. One package per domain. |
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
| `templates/components/` | 44 shadcn-style UI components, each in its own package directory. Every component: `button/`, `card/`, `input/`, `dialog/`, etc. Shared utils in `shared/`. All use oklch design tokens, `--radius`, light+dark mode, typed variant constants, `Attrs templ.Attributes` escape hatch. | Create CSS component classes. Use `@apply`. Always use the component functions. |
| `templates/icons` | Icon system. `lucide_gen.go` (generated, do not edit). `icon.go` has `Register()` for custom icons and `Custom()` to render them. | Edit `lucide_gen.go` by hand. It is generated by `cmd/icongen`. |
| `migrations/` | SQL migration files. Boilerplate base uses `NNN_` prefix; user migrations use timestamp `YYYYMMDDHHMMSS_` prefix. Embedded via `embed.go`. | Create migrations without both `-- +goose Up` AND `-- +goose Down` sections. |
| `queries/` | SQLC query files (`.sql`). One file per domain. | Write raw SQL in Go files. Write queries here, then run `sqlc generate`. |
| `cmd/app` | CLI entry point (`main.go`), server startup (`serve.go`), router construction (`router.go`), module registration (`app.go`), migration commands (`migrate.go`), route listing (`routes.go`), seed data (`seed.go`), diagnostics (`doctor.go`), version (`version.go`). Only `app.go` is user-edited; all other files are boilerplate. | Put business logic here. This package wires dependencies and delegates to `internal/`. |
| `cmd/icongen` | Icon code generator. Reads `lucideIcons` slice and writes `templates/icons/lucide_gen.go`. | Edit the generated output. Edit the source `lucideIcons` slice in this file, then run `go run ./cmd/icongen`. |
| `assets/` | Static assets (CSS, JS, images). Embedded and served at `/assets/*`. `app.css` is the Tailwind source (`@theme` tokens only, no `@apply`). Dev uses browser CDN (`tailwindcss-browser.js`). Prod uses `app.compiled.css` (from `make css`). | Edit compiled CSS directly. Use `@apply` in CSS. |

---

## 5. How to Add a New Domain (Step-by-Step Checklist)

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
       <domain>.Module,  // ← add here
   }
   ```

8. **Write tests** -- Create `internal/<domain>/handler_test.go` with table-driven tests. Use `testutil.SetupTestDB(t)` and `httptest`. Create a `setupRouter(deps)` helper that wires routes without auth middleware.

9. **Verify** -- Run `go build ./...` and `go test ./...`. Both must pass.

---

## 5b. How to Add a Simple Page (No Database)

For pages like landing, about, features, contact — no migration/queries needed.

1. **Create templ template** in the existing `internal/server/` package:
   ```
   internal/server/landing.templ
   ```
   Use `package server` and import layouts: `@layouts.Base("Page Title") { ... }`

2. **Create handler** in `internal/server/handler.go` (or a new file like `landing_handler.go`):
   ```go
   func HandleLanding() http.HandlerFunc {
       return func(w http.ResponseWriter, r *http.Request) {
           Landing().Render(r.Context(), w)
       }
   }
   ```

3. **Register route** in `cmd/app/app.go` or the appropriate router mount:
   ```go
   r.Get("/landing", server.HandleLanding())
   ```

4. **For forms** (e.g., contact page that saves to DB): use `go run ./cmd/app generate domain` instead — it creates everything including queries and migration.

**Common templ patterns:**
```
templ Landing() {
    @layouts.Base("Landing") {
        <div class="max-w-4xl mx-auto p-8">
            <h1 class="text-4xl font-bold">Welcome</h1>
        </div>
    }
}
```

**Things that break the build:**
- Wrong package name in .templ file (must match the directory)
- Importing a package that doesn't exist in go.mod
- Missing closing braces in templ syntax (templ uses `{ }` not `{{ }}`)
- Using `html/template` syntax instead of templ syntax

---

## 6. How to Remove a Domain

1. **Remove from modules** -- Delete the line from `cmd/app/app.go`.
2. **Delete package** -- Delete `internal/<domain>/` (handlers, templates, module, tests — all in one directory).
3. **Delete queries** -- Delete `queries/<domain>.sql`, then run `sqlc generate`.
4. **Handle migration** -- Create a new down migration or use `migrate reset` (dev only).
5. **Verify** -- Run `go build ./...` and `go test ./...`.

---

## 7. How to Add Configuration for a Feature

1. **Add fields** to the `Config` struct in `internal/config/config.go`. Add env var loading in the `Load()` function using the `envOr()` helper.

2. **Add to redacted view** -- Update `Redacted()` in `internal/config/view.go` to include the new fields. Mark sensitive values.

3. **Register schema group** -- Add a new `SchemaGroup` in `internal/config/schema.go` via the `init()` function, or call `config.RegisterGroup()` from an `init()` in the feature package.

4. **Update tests** -- Add test cases in `internal/config/config_test.go`.

---

## 8. How to Add a Background Job

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
       Name:  "<domain>",
       Path:  "/<domain>",
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

## 9. How to Send Email

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

## 10. How to Add Icons

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

## 11. UI Components & Design Tokens

### Component System

Every component lives in its own package under `templates/components/<name>/`. Each package defines its own `Props` struct and typed variant constants. Shared utilities (`Cx`, `RandomID`) are in `templates/components/shared/`.

Every Props struct has:
- Named fields for options (`Variant`, `Size`, etc.) — typed constants, not strings
- `Class string` — appended to computed classes
- `Attrs templ.Attributes` — escape hatch for `hx-get`, `data-*`, `aria-*`, etc.

**Import pattern** — each component is a separate import:
```go
import (
    "github.com/omaklabs/base/templates/components/button"
    "github.com/omaklabs/base/templates/components/card"
    "github.com/omaklabs/base/templates/components/badge"
    "github.com/omaklabs/base/templates/components/input"
    // ... add only what you use
)
```

**Usage examples:**
```go
// Button — zero-arg gives primary/md defaults
@button.Button() { Save }
@button.Button(button.Props{Variant: button.VariantDestructive, Size: button.SizeSm}) { Delete }
@button.Button(button.Props{Loading: true}) { Saving... }
@button.Button(button.Props{Variant: button.VariantLink}) { Learn more }
@button.Button(button.Props{
    Attrs: templ.Attributes{"hx-delete": "/items/1"},
}) { Delete via HTMX }
@button.LinkButton("/items/new") { New Item }
@button.LinkButton("#", button.Props{Variant: button.VariantOutline}) { Outline Link }
@button.SubmitButton() { Create }

// Card
@card.Card() { content }
@card.Card(card.Props{Padding: "sm"}) { compact }

// Badge — typed variants
@badge.Badge() { Default }
@badge.Badge(badge.Props{Variant: badge.VariantSuccess}) { Active }

// Form fields — one-call composites
@formfield.FormField(formfield.Props{Label: "Email", Name: "email", Type: "email", Value: val, ErrMsg: err})
@formfield.TextareaFormField(formfield.TextareaProps{Label: "Bio", Name: "bio", Rows: 4})
@formfield.SelectFormField(formfield.SelectProps{Label: "Role", Name: "role", Options: opts})

// Individual form elements
@input.Input(input.Props{Type: "email", Name: "email", Placeholder: "you@example.com"})
@input.Input(input.Props{Type: "file", FileAccept: "image/*"})
@textarea.Textarea(textarea.Props{Name: "body", Rows: 8})
@selectfield.SelectField(selectfield.Props{Name: "role", Options: opts})
@label.Label(label.Props{For: "name"}) { Full Name }
@errortext.ErrorText(errMsg)

// Checkbox / Radio
@checkbox.Checkbox(checkbox.Props{ID: "terms", Name: "terms"})
@checkboxfield.CheckboxField(checkboxfield.Props{Label: "Published", Name: "published", Checked: true})
@radiogroup.FormField(radiogroup.FormFieldProps{Label: "Priority", Name: "priority", Options: opts, Value: val})

// Layout
@pageheader.PageHeader(pageheader.Props{Title: "Items"}) { @button.LinkButton("/items/new") { New } }
@emptystate.EmptyState(emptystate.Props{Message: "No items yet."}) { @icons.Inbox() }
@navitem.NavItem(navitem.Props{Href: "/items", Active: true}) { Items }
@separator.Separator()
@separator.Separator(separator.Props{Orientation: separator.Vertical})
@breadcrumb.Breadcrumb(breadcrumb.Props{Items: items})

// Feedback
@alert.Alert(alert.Props{Variant: alert.VariantWarning}) { Warning text }
@spinner.Spinner()
@skeleton.Skeleton(skeleton.Props{Width: "w-full", Height: "h-4"})
@progress.Progress(progress.Props{Value: 75, Variant: progress.VariantSuccess})
@slider.Slider(slider.Props{Name: "volume", Value: 50, Max: 100})

// Typography helpers
@typography.H1() { Page Title }
@typography.H2() { Section }
@typography.P() { Paragraph }
@typography.Muted() { Helper text }

// Theme
@themetoggle.ThemeToggle()

// Dialog (Lit web component)
@dialog.Dialog(dialog.Props{}) {
    @dialog.Trigger() { @button.Button() { Open } }
    @dialog.Content() {
        @dialog.Header() { <h3>Title</h3> }
        @dialog.Body() { <p>Content</p> }
        @dialog.Footer() {
            @dialog.Close() { @button.Button(button.Props{Variant: button.VariantGhost}) { Cancel } }
            @button.Button(button.Props{Variant: button.VariantDestructive}) { Confirm }
        }
    }
}

// AlertDialog (pre-built confirmation — replaces hx-confirm)
@alertdialog.AlertDialog(alertdialog.Props{
    Title: "Are you sure?", Description: "Cannot be undone.",
    ConfirmText: "Delete", Variant: "destructive",
    ConfirmAttrs: templ.Attributes{"hx-delete": "/items/1"},
}) { @button.Button(button.Props{Variant: button.VariantDestructive}) { Delete } }

// Dropdown
@dropdown.Dropdown() {
    @dropdown.Trigger() { @button.Button(button.Props{Variant: button.VariantOutline}) { Menu } }
    @dropdown.Content() {
        @dropdown.Item(dropdown.ItemProps{Href: "/profile"}) { Profile }
        @dropdown.Separator()
        @dropdown.Item(dropdown.ItemProps{Variant: dropdown.ItemVariantDestructive}) { Log Out }
    }
}

// Tabs
@tabs.Tabs(tabs.Props{Default: "overview"}) {
    @tabs.List() {
        @tabs.Trigger(tabs.TriggerProps{Value: "overview"}) { Overview }
        @tabs.Trigger(tabs.TriggerProps{Value: "settings"}) { Settings }
    }
    @tabs.Content(tabs.ContentProps{Value: "overview"}) { Overview content }
    @tabs.Content(tabs.ContentProps{Value: "settings"}) { Settings content }
}

// Toast — server: HX-Trigger header; client: onclick
// Server: w.Header().Set("HX-Trigger", `{"toast": {"variant": "success", "message": "Saved!"}}`)
// Client: @button.Button(button.Props{Attrs: templ.Attributes{"onclick": toast.Show("success", "Saved!", 4000)}})

// Table
@table.Table() {
    @table.Header() { @table.Row() { @table.HeaderCell() { Name } @table.HeaderCell() { Email } } }
    @table.Body() { @table.Row() { @table.Cell() { Alice } @table.Cell() { alice@example.com } } }
}

// Accordion (native <details>, zero JS)
@accordion.Accordion() {
    @accordion.Item(accordion.ItemProps{Value: "q1"}) {
        @accordion.Trigger() { Question? }
        @accordion.Content() { Answer. }
    }
}

// Popover, Combobox, Avatar, Tooltip, Sheet, Sidebar, Switch, CopyButton, TagsInput
@popover.Popover() { @popover.Trigger() { ... } @popover.Content() { ... } }
@combobox.Combobox(combobox.Props{Name: "user", Placeholder: "Select...", Options: opts})
@avatar.Avatar(avatar.Props{Initials: "AJ"})
@tooltip.Tooltip(tooltip.Props{Text: "Help"}) { @button.Button() { Hover } }
@sheet.Sheet(sheet.Props{}) { @sheet.Trigger() { ... } @sheet.Content() { ... } }
@sidebar.Layout() { @sidebar.Sidebar() { ... } <main>{ children... }</main> }
@switchc.Switch(switchc.Props{Name: "notify", Checked: true})
@copybutton.CopyButton(copybutton.Props{TargetID: "api-key"})
@tagsinput.TagsInput(tagsinput.Props{Name: "tags", Value: []string{"go", "htmx"}})

// Form sub-components (for custom layouts)
@form.Item() {
    @label.Label(label.Props{For: "email"}) { Email }
    @input.Input(input.Props{Name: "email", HasError: true})
    @form.Description() { We'll never share your email. }
    @form.Message(form.MessageProps{Variant: form.MessageError}) { Invalid email. }
}
```

### Design Tokens (shadcn pattern)

Colors are defined in the `@theme` block in `base.templ`. Every color is paired with a foreground
(the text color that works on that background). This makes the theme swappable without changing components.

| Token | Tailwind class | Used for |
|-------|---------------|----------|
| `background` / `foreground` | `bg-background`, `text-foreground` | Page bg, default text |
| `card` / `card-foreground` | `bg-card`, `text-card-foreground` | Cards, panels |
| `primary` / `primary-foreground` | `bg-primary`, `text-primary-foreground` | Brand color, primary buttons |
| `secondary` / `secondary-foreground` | `bg-secondary`, `text-secondary-foreground` | Secondary buttons |
| `muted` / `muted-foreground` | `bg-muted`, `text-muted-foreground` | Muted backgrounds, secondary text |
| `accent` / `accent-foreground` | `bg-accent`, `text-accent-foreground` | Hover highlights |
| `destructive` / `destructive-foreground` | `bg-destructive`, `text-destructive` | Danger/error |
| `success` / `success-foreground` | `bg-success`, `text-success` | Success states |
| `warning` / `warning-foreground` | `bg-warning`, `text-warning` | Warning states |
| `border` | `border-border` | Default borders |
| `input` | `border-input` | Input field borders |
| `ring` | `ring-ring` | Focus rings |

Standard Tailwind colors (`bg-red-500`, `text-blue-300`) still work for one-off accents.

### Typography Convention

| Element | Classes |
|---------|---------|
| Page title | `text-2xl font-bold` |
| Section heading | `text-lg font-semibold` |
| Body text | `text-sm` |
| Form labels | `text-sm text-muted-foreground` |
| Helper text | `text-xs text-muted-foreground` |
| Timestamps | `text-xs text-muted-foreground/60` |

### Creating New Components

Every component lives in `templates/components/<name>/` as its own Go package.

**Pure templ component** (no JS):
1. Create `templates/components/<name>/<name>.templ` with `package <name>`
2. Define `Props` struct with `Class string` and `Attrs templ.Attributes`
3. Define typed variant constants if needed (`type Variant string`)
4. Use `shared.Cx()` from `templates/components/shared` to build class strings
5. Run `templ generate`

**Interactive component** (Lit web component, co-located JS/CSS/templ):
1. Create `templates/components/<name>/` with three files:
   - `<name>.templ` — Go types + templ API (renders `<omk-<name>>` custom element)
   - `<name>.js` — Lit class (`import { LitElement } from "/assets/js/lit-all.min.js"`)
   - `<name>.css` — Transitions/animations for `[open]` or `[data-*]` attributes
2. Lit uses light DOM: `createRenderRoot() { return this; }` (Tailwind passes through)
3. Prefix data attributes with component name: `data-dialog-trigger`, `data-sheet-close`
4. Add `Script()` templ with `NewOnceHandle()` and register in `templates/layouts/base.templ`
5. Run `templ generate`

### Server-Side Toast Pattern

Instead of rendering toast HTML, set the `HX-Trigger` response header:
```go
w.Header().Set("HX-Trigger", `{"toast": {"variant": "success", "message": "Item saved!"}}`)
```
The `<omk-toast-container>` listens for this event automatically.

### Import Paths

Every component is at `templates/components/<name>`:

| Package | Import | Main export |
|---------|--------|-------------|
| `accordion` | `templates/components/accordion` | `accordion.Accordion()` |
| `alert` | `templates/components/alert` | `alert.Alert()` |
| `alertdialog` | `templates/components/alertdialog` | `alertdialog.AlertDialog()` |
| `avatar` | `templates/components/avatar` | `avatar.Avatar()` |
| `badge` | `templates/components/badge` | `badge.Badge()` |
| `breadcrumb` | `templates/components/breadcrumb` | `breadcrumb.Breadcrumb()` |
| `button` | `templates/components/button` | `button.Button()`, `button.LinkButton()`, `button.SubmitButton()` |
| `card` | `templates/components/card` | `card.Card()` |
| `checkbox` | `templates/components/checkbox` | `checkbox.Checkbox()` |
| `checkboxfield` | `templates/components/checkboxfield` | `checkboxfield.CheckboxField()` |
| `combobox` | `templates/components/combobox` | `combobox.Combobox()` |
| `copybutton` | `templates/components/copybutton` | `copybutton.CopyButton()` |
| `csrf` | `templates/components/csrf` | `csrf.CSRFField()` |
| `dialog` | `templates/components/dialog` | `dialog.Dialog()`, `Trigger()`, `Content()`, `Header()`, `Body()`, `Footer()`, `Close()` |
| `dropdown` | `templates/components/dropdown` | `dropdown.Dropdown()`, `Trigger()`, `Content()`, `Item()`, `Separator()` |
| `emptystate` | `templates/components/emptystate` | `emptystate.EmptyState()` |
| `errortext` | `templates/components/errortext` | `errortext.ErrorText()` |
| `flash` | `templates/components/flash` | `flash.FlashMessage()` |
| `form` | `templates/components/form` | `form.Item()`, `form.ItemFlex()`, `form.Description()`, `form.Message()` |
| `formfield` | `templates/components/formfield` | `formfield.FormField()`, `formfield.TextareaFormField()`, `formfield.SelectFormField()` |
| `input` | `templates/components/input` | `input.Input()` |
| `label` | `templates/components/label` | `label.Label()` |
| `navitem` | `templates/components/navitem` | `navitem.NavItem()` |
| `pageheader` | `templates/components/pageheader` | `pageheader.PageHeader()` |
| `pagination` | `templates/components/pagination` | `pagination.Pagination()` |
| `popover` | `templates/components/popover` | `popover.Popover()`, `Trigger()`, `Content()` |
| `progress` | `templates/components/progress` | `progress.Progress()` |
| `radiogroup` | `templates/components/radiogroup` | `radiogroup.Group()`, `radiogroup.FormField()` |
| `selectfield` | `templates/components/selectfield` | `selectfield.SelectField()` |
| `separator` | `templates/components/separator` | `separator.Separator()` |
| `sheet` | `templates/components/sheet` | `sheet.Sheet()`, `Trigger()`, `Content()`, `Close()` |
| `sidebar` | `templates/components/sidebar` | `sidebar.Layout()`, `sidebar.Sidebar()`, `Menu()`, `MenuItem()` |
| `skeleton` | `templates/components/skeleton` | `skeleton.Skeleton()` |
| `slider` | `templates/components/slider` | `slider.Slider()` |
| `spinner` | `templates/components/spinner` | `spinner.Spinner()` |
| `switchc` | `templates/components/switchc` | `switchc.Switch()` |
| `table` | `templates/components/table` | `table.Table()`, `Header()`, `Body()`, `Row()`, `HeaderCell()`, `Cell()` |
| `tabs` | `templates/components/tabs` | `tabs.Tabs()`, `List()`, `Trigger()`, `Content()` |
| `tagsinput` | `templates/components/tagsinput` | `tagsinput.TagsInput()` |
| `textarea` | `templates/components/textarea` | `textarea.Textarea()` |
| `themetoggle` | `templates/components/themetoggle` | `themetoggle.ThemeToggle()` |
| `toast` | `templates/components/toast` | `toast.Container()`, `toast.Show()` |
| `tooltip` | `templates/components/tooltip` | `tooltip.Tooltip()` |
| `typography` | `templates/components/typography` | `typography.H1()`, `H2()`, `H3()`, `H4()`, `P()`, `Lead()`, `Muted()`, `InlineCode()` |

### Component Selection Guide

| Need | Use | Not |
|------|-----|----|
| Text/email/password field | `formfield.FormField()` | Raw `<input>` |
| Multi-line text | `formfield.TextareaFormField()` | Raw `<textarea>` |
| Boolean in a form | `checkboxfield.CheckboxField()` | Raw `<input type="checkbox">` |
| Enum with 2-5 options | `radiogroup.FormField()` | selectfield for short lists |
| Enum with 6+ options | `formfield.SelectFormField()` | radiogroup for long lists |
| Searchable list (10+ items) | `combobox.Combobox()` | selectfield without search |
| Delete/destructive confirmation | `alertdialog.AlertDialog()` | `hx-confirm` attribute |
| Success/error feedback after action | `toast.Show()` via `HX-Trigger` header | flash (flash is for page-load only) |
| Page title with actions | `pageheader.PageHeader()` | Raw `<h1>` + flex layout |
| Empty list/table state | `emptystate.EmptyState()` | Raw centered `<p>` |
| Side navigation layout | `sidebar.Layout()` + `sidebar.Sidebar()` | Raw `<aside>` + flex |
| FAQ / collapsible sections | `accordion.Accordion()` | Alpine x-show toggles |
| Contextual mini-form | `popover.Popover()` | dropdown (dropdown is for menus) |
| Menu with actions | `dropdown.Dropdown()` | popover (popover is for content) |
| Tabbed content | `tabs.Tabs()` | Alpine x-show with manual tab state |
| Modal dialog | `dialog.Dialog()` | Alpine x-data overlay |
| Slide-out panel | `sheet.Sheet()` | dialog (dialog is centered) |
| Dark/light mode toggle | `themetoggle.ThemeToggle()` | Manual Alpine toggle |
| Loading state on button | `button.Button()` with `Loading: true` | Manual spinner + disabled |
| Loading placeholder | `skeleton.Skeleton()` | Empty div with pulse class |
| Status label | `badge.Badge()` | Raw `<span>` with colors |
| Persistent info/warning block | `alert.Alert()` | flash (flash auto-dismisses) |
| Copy to clipboard | `copybutton.CopyButton()` | Manual JS |
| Multi-tag input | `tagsinput.TagsInput()` | Raw input + manual JS |
| Range/slider | `slider.Slider()` | Raw `<input type="range">` |
| File upload | `input.Input()` with `Type: "file"` | Raw `<input type="file">` |

### Rules

- Always use components for buttons, cards, form fields, badges, flash messages, navigation
- Use `Attrs: templ.Attributes{...}` to pass HTMX attributes — never drop to raw HTML for this
- For page-level layout (spacing, flex, grid), use inline Tailwind classes directly
- Prefer design tokens over raw Tailwind colors (`bg-primary` not `bg-orange-500`)
- Never use `@apply` in CSS files
- Never create CSS component classes (`.btn`, `.card`, etc.)
- Interactive components use Lit web components (not Alpine inline scripts)
- Alpine.js stays for simple ad-hoc toggles in user pages (flash auto-dismiss, etc.)

---

## 12. Conventions

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
- Use templ component functions from `templates/components/` for all UI primitives (buttons, cards, inputs, badges, etc.).
- Use `@icons.IconName("size")` for icons.
- Pass validation errors as `validate.Errors` to form templates for inline error display.
- Page templates use `package <domain>` (not `package pages`) and live in the domain directory.

---

## 13. What You Must Never Do

- Use GORM or any ORM.
- Write raw SQL strings in handler files. All queries go in `queries/*.sql` and are accessed via SQLC-generated methods.
- Use CGO or any library that requires CGO.
- Add Node.js, npm, or any JavaScript build tooling.
- Use `html/template`. Always use templ.
- Create CSS component classes (`.btn`, `.card`, etc.). Use templ components instead.
- Use `@apply` in CSS files. Write inline Tailwind classes in templ files.
- Use goroutines in handlers. Use `internal/jobs` for async work.
- Skip CSRF tokens on form submissions. Every POST/PUT/DELETE form must include `@components.CSRFField()`.
- Edit SQLC-generated files (`internal/db/*.sql.go`, `internal/db/models.go`) by hand. They are overwritten by `sqlc generate`.
- Edit `templates/icons/lucide_gen.go` by hand. It is generated by `cmd/icongen`.
- Use `fmt.Println` or `log.Printf` for application logging. Use the structured logger (`logger.Info`, `logger.Warn`, `logger.Error`).
- Store business logic in `cmd/app`. Put it in `internal/`.
- Use global mutable state. Inject dependencies via constructors and struct fields (exception: `var Module` in domain packages).
- Inline styles or non-Tailwind CSS classes.
- Hardcode colors (`text-white`, `bg-zinc-800`). Use design tokens (`text-primary-foreground`, `bg-card`).
- Use `os.*` directly for file storage. Use the `storage.Storage` interface.
- Edit boilerplate-owned files in `cmd/app/` (except `app.go`). Register domains via the `modules` list in `app.go` only.

---

## 14. CLI Commands

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

## 15. Change Log

When making significant changes to architecture, database schema, or infrastructure packages, document the decision in `changelog/`.

### What to Log

- Database schema changes (new tables, column renames, index changes)
- Infrastructure package restructuring (new packages in `internal/`, changed interfaces)
- Major refactors affecting multiple domains
- New architectural patterns introduced

### ADR Format

Create a new file: `changelog/YYYY-MM-DD-short-title.md`

```markdown
# Title

## Context

What is the background? What problem are we solving?

## Decision

What change was made?

## Consequences

- What are the implications?
- What migrations or follow-up work is needed?
```

### When NOT to Log

- Adding a new domain (the migration and code are self-documenting)
- Bug fixes (the commit message is enough)
- Adding icons, templates, or queries (routine additions)

---

## 16. Extension Points

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

## 17. Build & Pre-Commit Checklist

Use `make build` for a full build (generates code, compiles CSS, compiles Go). Individual targets:

```
make build           Full build (generate + css + go build)
make dev             Build and start the server
make css             Compile Tailwind CSS
make css-watch       Watch and recompile Tailwind CSS
make generate        Run templ generate + sqlc generate
make test            Run all tests
```

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

If you added or changed `assets/css/app.tailwind.css`, first run:
```
make css
```

If you added icons to `cmd/icongen/main.go`, first run:
```
go run ./cmd/icongen
```
