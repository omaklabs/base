# Plan: Upgradeable & Modular Boilerplate Architecture

## Problems

### 1. Upgradeability

Adding a domain feature requires editing boilerplate-owned files (`router.go`, `serve.go`, `seed.go`). An upgrade from a newer boilerplate version always produces merge conflicts. There is no boundary between "boilerplate code" and "user code."

### 2. Modularity (AI agent friendliness)

A domain is scattered across 4+ directories. To add "posts," an AI agent must touch:

```
internal/posts/handler.go              ← handlers
templates/pages/posts_list.templ       ← template 1
templates/pages/posts_show.templ       ← template 2
templates/pages/posts_form.templ       ← template 3
queries/posts.sql                      ← queries
migrations/NNN_create_posts.sql        ← migration
cmd/app/app.go                         ← registration line
```

To remove "posts," the AI must find and delete files from all those locations. Miss one template and you get orphans. Miss the registration line and the build breaks.

### Goals

1. Upgrade boilerplate by replacing files → `go build` → done. Zero merge conflicts.
2. Add a domain = create one directory + one query file + one migration + one registration line.
3. Remove a domain = delete one directory + remove query file + remove registration line.
4. Each domain is self-describing — an AI reads one directory to understand everything.

---

## Part 1: Upgrade Boundary

### Design Principle

**One registration file.** The user edits exactly one file in `cmd/app/` — `app.go`. Everything else in `cmd/app/` is boilerplate-owned. Boilerplate calls user code through well-defined hook functions.

**Import direction:**
```
cmd/app/app.go  →  internal/notes/, internal/posts/, ...  (user domains)
cmd/app/*.go    →  internal/server/, internal/middleware/   (boilerplate infra)
server          ←  notes, posts  (domains import server, never the reverse)
```

### File Ownership: cmd/app/

| File | Owner | Editable by user? | What changes |
|------|-------|-------------------|--------------|
| `main.go` | BOILERPLATE | No | (none) |
| `serve.go` | BOILERPLATE | No | Calls `registerJobs()` and `registerSchedules()` |
| `router.go` | BOILERPLATE | No | Calls `registerRoutes()` |
| `seed.go` | BOILERPLATE | No | Calls `registerSeeds()` to get seed list |
| `routes.go` | BOILERPLATE | No | (none) |
| `migrate.go` | BOILERPLATE | No | (none) |
| `doctor.go` | BOILERPLATE | No | (none) |
| `version.go` | BOILERPLATE | No | (none) |
| **`app.go`** | **USER** | **Yes** | All domain registration in one place |

### File Ownership: internal/

| Package | Owner | Notes |
|---------|-------|-------|
| `server/` | BOILERPLATE | Deps, Module type, helpers, welcome, HandleNotFound |
| `middleware/` | BOILERPLATE | |
| `db/` | BOILERPLATE + GENERATED | Connection is boilerplate; `*.sql.go` generated from user queries |
| `jobs/` | BOILERPLATE | |
| `email/` | BOILERPLATE | |
| `storage/` | BOILERPLATE | |
| `auth/` | BOILERPLATE | |
| `validate/` | BOILERPLATE | |
| `pagination/` | BOILERPLATE | |
| `flash/` | BOILERPLATE | |
| `view/` | BOILERPLATE | |
| `logger/` | BOILERPLATE | |
| `api/` | BOILERPLATE | |
| `config/` | BOILERPLATE | |
| `testutil/` | BOILERPLATE | |
| `conventions/` | BOILERPLATE | |
| `notes/` | USER | Domain package (reference example) |
| `<any other>/` | USER | Domain packages added by user |

### File Ownership: Other

| Path | Owner |
|------|-------|
| `templates/layouts/` | BOILERPLATE |
| `templates/components/` | BOILERPLATE |
| `templates/icons/` | BOILERPLATE + GENERATED |
| `templates/pages/` | REMOVED (templates co-located in domain packages) |
| `queries/` | USER |
| `migrations/` | BOILERPLATE base (001-005) + USER (timestamp-prefixed) |
| `assets/` | BOILERPLATE base + USER additions |
| `CONVENTIONS.md` | BOILERPLATE |
| `.omakase.yaml` | BOILERPLATE (manifest) |
| `go.mod` / `go.sum` | MIXED |

---

## Part 2: Module Pattern (Self-Describing Domains)

### The Module struct

Each domain exports a `Module` variable that fully describes what it provides. Defined in the `server` package (boilerplate-owned):

```go
// internal/server/module.go — BOILERPLATE
package server

import (
    "context"

    "github.com/go-chi/chi/v5"
    "github.com/omaklabs/base/internal/db"
    "github.com/omaklabs/base/internal/jobs"
)

// Module describes a domain package's contributions to the app.
// Each domain exports a Module var. The boilerplate iterates these
// for route mounting, job registration, seed execution, etc.
type Module struct {
    // Name is a human-readable identifier (e.g., "notes", "billing").
    Name string

    // Path is the URL prefix (e.g., "/notes", "/billing").
    Path string

    // Mount registers the domain's routes on the given router.
    // The domain applies its own middleware (e.g., RequireAuth) inside.
    Mount func(r chi.Router, deps *Deps)

    // Jobs lists background job handlers this domain provides.
    Jobs []Job

    // Schedules lists recurring tasks this domain needs.
    Schedules []Schedule

    // Seeds lists seed functions for development data.
    Seeds []Seed
}

// Job pairs a job type name with its handler function.
type Job struct {
    Type    string
    Handler jobs.JobHandler
}

// Schedule wraps a recurring task definition.
type Schedule struct {
    jobs.Schedule
}

// Seed pairs a seed name with its function.
type Seed struct {
    Name string
    Fn   func(ctx context.Context, q *db.Queries) error
}
```

### Domain package uses the Module struct

```go
// internal/notes/module.go — USER
package notes

import "github.com/omaklabs/base/internal/server"

// Module describes the notes domain.
var Module = server.Module{
    Name:  "notes",
    Path:  "/notes",
    Mount: Mount,
    // Jobs: []server.Job{
    //     {Type: "note_cleanup", Handler: handleNoteCleanup},
    // },
    // Seeds: []server.Seed{
    //     {Name: "notes", Fn: seedNotes},
    // },
}
```

### app.go becomes a module list

```go
// cmd/app/app.go — USER FILE
package main

import (
    "github.com/omaklabs/base/internal/notes"
    "github.com/omaklabs/base/internal/server"
)

// modules lists all domain modules in the app.
// Add a domain: import the package, append its Module here.
// Remove a domain: delete the line and remove the import.
var modules = []server.Module{
    notes.Module,
    // posts.Module,
    // billing.Module,
}
```

### Boilerplate iterates modules

```go
// cmd/app/router.go — BOILERPLATE
func buildRouter(..., deps *server.Deps, ...) *chi.Mux {
    // ... middleware setup ...

    router.Get("/", server.HandleWelcome())

    // Mount all domain routes from modules
    for _, m := range modules {
        if m.Mount != nil {
            router.Route(m.Path, func(r chi.Router) { m.Mount(r, deps) })
        }
    }

    router.NotFound(server.HandleNotFound())
    // ...
}
```

```go
// cmd/app/serve.go — BOILERPLATE
func cmdServe() {
    // ... deps setup ...

    // Register all jobs from modules
    for _, m := range modules {
        for _, j := range m.Jobs {
            deps.Queue.Register(j.Type, j.Handler)
        }
    }

    // Start all schedules from modules
    for _, m := range modules {
        for _, s := range m.Schedules {
            // ... start scheduler ...
        }
    }

    // ...
}
```

```go
// cmd/app/seed.go — BOILERPLATE
func cmdSeed() {
    // ... db setup ...

    // Collect seeds from all modules
    var seeds []Seed
    for _, m := range modules {
        for _, s := range m.Seeds {
            seeds = append(seeds, Seed{Name: s.Name, Fn: s.Fn})
        }
    }

    for _, s := range seeds { ... }
}
```

No separate `registerRoutes()`, `registerJobs()`, `registerSchedules()`, `registerSeeds()` functions needed — the module struct carries everything. The boilerplate iterates one `modules` slice for all purposes.

---

## Part 3: Co-Located Templates

### Problem

Domain templates live in `templates/pages/`, separated from the handlers that use them. The AI must look in two places to understand a domain, and removing a domain risks leaving orphan template files.

### Solution: Move page templates into domain packages

Templ files can live in any Go package. `templ generate` scans the entire project tree. Moving templates into the domain package means:

- Handler calls `NotesList(...)` directly (same package, no import).
- Delete `internal/notes/` = delete all code AND templates in one shot.
- The AI reads one directory to understand the entire domain.

### Before (scattered)

```
internal/notes/
├── handler.go
├── handler_test.go
└── module.go

templates/pages/               ← separate directory
├── notes_list.templ
├── notes_show.templ
├── notes_form.templ
├── welcome.templ              ← boilerplate
├── error_404.templ            ← boilerplate
├── error_500.templ            ← boilerplate
└── error_dev.templ            ← boilerplate
```

### After (co-located)

```
internal/notes/                ← everything in one place
├── handler.go
├── handler_test.go
├── module.go
├── notes_list.templ           ← moved here
├── notes_show.templ           ← moved here
└── notes_form.templ           ← moved here

internal/server/               ← boilerplate templates move here
├── deps.go
├── module.go
├── helpers.go
├── helpers_test.go
├── welcome.go
├── welcome.templ              ← moved from templates/pages/
├── welcome_test.go
├── error_404.templ            ← moved from templates/pages/
├── error_500.templ            ← moved from templates/pages/
└── error_dev.templ            ← moved from templates/pages/

templates/                     ← only shared infrastructure remains
├── layouts/
│   └── base.templ
├── components/
│   ├── button.templ
│   ├── card.templ
│   ├── csrf.templ
│   ├── flash.templ
│   ├── input.templ
│   ├── pagination.templ
│   └── submit_button.templ
└── icons/
    ├── icon.go
    └── lucide_gen.go

templates/pages/               ← REMOVED (empty, deleted)
```

### How co-located templates work

The templ file declares the domain's package:

```
// internal/notes/notes_list.templ
package notes                  // ← same package as handler.go

import (
    "fmt"
    "github.com/omaklabs/base/internal/db"
    "github.com/omaklabs/base/internal/pagination"
    "github.com/omaklabs/base/templates/components"
    "github.com/omaklabs/base/templates/layouts"
)

templ NotesList(notes []db.Note, p pagination.Pagination, baseURL string) {
    @layouts.App() {
        // ... (same template content as before)
    }
}
```

The handler calls it without an import prefix:

```go
// internal/notes/handler.go
func handleListNotes(deps *server.Deps) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // ...
        NotesList(notes, p, "/notes").Render(r.Context(), w)  // ← same package!
    }
}
```

Import graph stays clean and acyclic:
```
internal/notes → internal/server          (deps, helpers)
internal/notes → internal/db              (query types)
internal/notes → templates/layouts        (base HTML shell)
internal/notes → templates/components     (shared UI components)
```

### What changes for server/ templates

Error and welcome templates move from `templates/pages/` (package `pages`) to `internal/server/` (package `server`). The handler functions already live in `server`, so the calls simplify:

```go
// internal/server/helpers.go — BEFORE
pages.Error404().Render(r.Context(), w)

// internal/server/helpers.go — AFTER
Error404().Render(r.Context(), w)    // same package
```

```go
// internal/server/welcome.go — BEFORE
pages.Welcome().Render(r.Context(), w)

// internal/server/welcome.go — AFTER
Welcome().Render(r.Context(), w)     // same package
```

---

## Part 4: Auto-Discovery via Codegen (Optional)

### Problem

Even with the module pattern, the AI must still edit `app.go` to add/remove a module line. Can we eliminate that?

### Solution: A code generator scans for Module vars

A small generator (like the existing `cmd/icongen/`) scans `internal/*/` for exported `Module` variables and generates the module list:

```go
// cmd/modgen/main.go — BOILERPLATE
// Scans internal/*/ for packages that export a server.Module var.
// Generates cmd/app/modules_gen.go with imports and module list.
```

Usage:
```bash
go run ./cmd/modgen
```

Generated output:
```go
// cmd/app/modules_gen.go — GENERATED, DO NOT EDIT
package main

import (
    "github.com/omaklabs/base/internal/notes"
    "github.com/omaklabs/base/internal/posts"
    "github.com/omaklabs/base/internal/server"
)

var modules = []server.Module{
    notes.Module,
    posts.Module,
}
```

### AI workflow with codegen

| Action | Steps |
|--------|-------|
| Add domain | 1. Create `internal/posts/` (handler + templates + module) |
| | 2. Create `queries/posts.sql` |
| | 3. Create `migrations/TIMESTAMP_create_posts.sql` |
| | 4. Run `sqlc generate && templ generate && go run ./cmd/modgen` |
| | Done. No file editing beyond the new package. |
| Remove domain | 1. Delete `internal/posts/` |
| | 2. Delete `queries/posts.sql` |
| | 3. Add down migration |
| | 4. Run `sqlc generate && go run ./cmd/modgen` |
| | Done. No file editing. |

### When to implement

This is optional and can be deferred. The manual `app.go` approach (Part 2) is simpler and works fine. Codegen is a quality-of-life improvement for when there are many domains or frequent add/remove cycles.

If implemented, `app.go` is replaced by `modules_gen.go` (generated) and the user never edits any file in `cmd/app/`.

---

## Part 5: Complete Domain Package Structure

With all three parts combined, a domain package is fully self-contained:

```
internal/notes/
├── module.go              ← Module var (name, path, mount, jobs, seeds)
├── handler.go             ← Mount() + all handler functions
├── handler_test.go        ← tests (same package, no import gymnastics)
├── notes_list.templ       ← list page template
├── notes_list_templ.go    ← generated by templ
├── notes_show.templ       ← show page template
├── notes_show_templ.go    ← generated by templ
├── notes_form.templ       ← form template (create + edit)
└── notes_form_templ.go    ← generated by templ
```

External dependencies (queries, migrations) stay centralized because they must:
- **Queries** (`queries/notes.sql`) — SQLC needs one output package; generated types may reference each other via foreign keys.
- **Migrations** (`migrations/`) — must run in global order across all domains.

### The AI's mental model

```
"Everything about notes is in internal/notes/.
 Queries are in queries/notes.sql.
 The migration is in migrations/.
 Registration is one line in app.go (or auto-generated)."
```

One directory to read. One directory to delete. One line to toggle.

---

## Part 6: Migration Strategy

### Problem

Boilerplate ships migrations `001` through `005`. User adds `006`. Next boilerplate version also adds `006`. Conflict.

### Solution: Timestamp prefixes for new migrations

- **Keep** existing `001` through `005` as-is (already applied on user databases).
- **New boilerplate** migrations use reserved timestamps: `20250101NNNNNN_` prefix.
- **New user** migrations use current timestamps: `YYYYMMDDHHMMSS_` prefix.
- Goose parses the prefix as int64. Order: `5 < 20250101000001 < 20260315143000`. Correct.

```
migrations/
├── 001_create_users.sql                  # existing boilerplate
├── 002_create_sessions.sql               # existing boilerplate
├── 003_create_jobs.sql                   # existing boilerplate
├── 004_create_emails.sql                 # existing boilerplate
├── 005_create_notes.sql                  # existing boilerplate
├── 20250101000001_add_job_priority.sql   # future boilerplate upgrade
├── 20260315143000_create_posts.sql       # user-added
├── 20260320091500_add_tags.sql           # user-added
└── embed.go
```

No existing databases break. No renaming. `embed.go` (`//go:embed *.sql`) picks up everything automatically.

### Convention test update

`TestMigrationsAreNumberedSequentially` → `TestMigrationsAreOrdered`:

```go
func TestMigrationsAreOrdered(t *testing.T) {
    // Extract version numbers from filenames as int64
    // Verify each number > previous (strictly increasing, no gaps required)
}
```

---

## Part 7: Manifest (.omakase.yaml)

Declares which files are boilerplate-owned. An upgrade tool reads this to know what's safe to replace.

```yaml
# .omakase.yaml — boilerplate file ownership manifest
# Files listed under 'boilerplate' are safe to overwrite during upgrades.
# Files not listed are assumed user-owned.
version: "1.0.0"

boilerplate:
  # cmd/app — all except app.go (or modules_gen.go if using codegen)
  - cmd/app/main.go
  - cmd/app/serve.go
  - cmd/app/router.go
  - cmd/app/routes.go
  - cmd/app/migrate.go
  - cmd/app/seed.go
  - cmd/app/doctor.go
  - cmd/app/version.go

  # codegen tool (optional, Part 4)
  - cmd/modgen/**

  # internal infrastructure
  - internal/server/**
  - internal/middleware/**
  - internal/db/connection.go
  - internal/db/db.go
  - internal/jobs/**
  - internal/email/**
  - internal/storage/**
  - internal/auth/**
  - internal/validate/**
  - internal/pagination/**
  - internal/flash/**
  - internal/view/**
  - internal/logger/**
  - internal/api/**
  - internal/config/**
  - internal/testutil/**
  - internal/conventions/**

  # shared templates
  - templates/layouts/**
  - templates/components/**
  - templates/icons/**

  # base migrations
  - migrations/001_*.sql
  - migrations/002_*.sql
  - migrations/003_*.sql
  - migrations/004_*.sql
  - migrations/005_*.sql
  - migrations/embed.go

  # docs
  - CONVENTIONS.md
  - .omakase.yaml

user:
  - cmd/app/app.go
  - internal/notes/**
  - queries/**
```

---

## Implementation Order

### Phase A: Upgrade boundary (Parts 1 + 2)

**Step 1:** Create `internal/server/module.go` with `Module`, `Job`, `Schedule`, `Seed` types.

**Step 2:** Create `internal/notes/module.go` with `var Module = server.Module{...}`.

**Step 3:** Create `cmd/app/app.go` with `var modules` list containing `notes.Module`.

**Step 4:** Update `cmd/app/router.go` — loop over `modules` for route mounting, remove `notes` import.

**Step 5:** Update `cmd/app/serve.go` — loop over `modules` for job and schedule registration.

**Step 6:** Update `cmd/app/seed.go` — collect seeds from `modules`, remove `var seeds` global, export `Seed`/`SeedFunc` types.

**Step 7:** Verify: `go build ./... && go test ./... && go run ./cmd/app routes`

### Phase B: Co-located templates (Part 3)

**Step 8:** Move `templates/pages/notes_*.templ` to `internal/notes/`, change `package pages` to `package notes`.

**Step 9:** Move `templates/pages/welcome.templ`, `error_*.templ` to `internal/server/`, change `package pages` to `package server`.

**Step 10:** Update handlers to call templates without `pages.` prefix.

**Step 11:** Run `templ generate` to regenerate all `_templ.go` files in new locations.

**Step 12:** Delete `templates/pages/` directory (now empty).

**Step 13:** Update convention tests — template generation check should scan `internal/` too, not just `templates/`.

**Step 14:** Verify: `go build ./... && go test ./... && go run ./cmd/app routes && go run ./cmd/app doctor`

### Phase C: Housekeeping

**Step 15:** Update migration convention test (`TestMigrationsAreOrdered`).

**Step 16:** Create `.omakase.yaml` manifest.

**Step 17:** Update `CONVENTIONS.md` — package map, checklists, extension points, naming conventions.

### Phase D: Codegen (Optional, Part 4)

**Step 18:** Create `cmd/modgen/main.go` that scans for Module vars and generates `modules_gen.go`.

**Step 19:** Replace `cmd/app/app.go` with generated `cmd/app/modules_gen.go`.

**Step 20:** Add `//go:generate go run ../modgen` to `cmd/app/main.go` or a generate file.

---

## Verification Checklist

- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] `go run ./cmd/app routes` shows identical route tree
- [ ] `go run ./cmd/app doctor` all checks pass
- [ ] `go run ./cmd/app seed` runs cleanly
- [ ] Domain packages are self-contained (handlers + templates + module in one dir)
- [ ] No `templates/pages/` directory exists
- [ ] `var modules` in `app.go` is the only place domains are registered
- [ ] No domain imports in `cmd/app/` files except `app.go`
- [ ] No `var seeds` package-level global remains
- [ ] Timestamp-prefixed migrations sort correctly after sequential ones
- [ ] `.omakase.yaml` accurately reflects file ownership
- [ ] Adding a domain = create directory + query file + migration + one line in app.go
- [ ] Removing a domain = delete directory + remove query file + remove line from app.go

---

## CONVENTIONS.md Updates Summary

| Section | Change |
|---------|--------|
| Package Map | Add `server/` (Module type), update `notes/` (co-located templates), remove `templates/pages/` |
| How to Add a Domain | Step 4: create `internal/<domain>/` with handler + module + templates. Step 7: add to `modules` in `app.go` |
| How to Add a Background Job | Register via `Jobs` field in domain Module, not `serve.go` |
| Naming | Handler files: `internal/<domain>/handler.go`. Templates: `internal/<domain>/<domain>_<view>.templ` |
| Extension Points | All registration through `app.go` modules list |
| What You Must Never Do | Add: "Edit boilerplate-owned files in `cmd/app/` except `app.go`" |

---

## Risk Assessment

| Risk | Mitigation |
|------|------------|
| User forgets to add module to `app.go` | `go build` fails (unused import) or `./app routes` shows missing routes. Codegen (Phase D) eliminates this entirely. |
| Co-located templates import wrong layout | `templ generate` catches type mismatches at compile time |
| Template naming collision across domains | Impossible — each domain is its own Go package with its own namespace |
| Timestamp migration ordering surprise | Goose uses int64 comparison; `5 < 20250101000001` is always true |
| `.omakase.yaml` gets out of sync | Convention test can validate manifest matches actual file tree (future) |
| `var Module` in domain packages fails global var convention test | Add `Module` to the allowed-var pattern in `TestNoGlobalVarsInInternal` (same as `Err` prefix) |
