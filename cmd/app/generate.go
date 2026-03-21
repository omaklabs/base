package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"
	"unicode"
)

func cmdGenerate(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: app generate <type> <name>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Types:")
		fmt.Fprintln(os.Stderr, "  domain <name>      Generate a full domain package (handler, module, templates, queries, migration)")
		fmt.Fprintln(os.Stderr, "  migration <name>    Generate a timestamped migration file")
		os.Exit(1)
	}

	switch args[0] {
	case "domain":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: app generate domain <name>")
			fmt.Fprintln(os.Stderr, "  name: lowercase singular noun (e.g., post, comment, invoice)")
			os.Exit(1)
		}
		generateDomain(args[1])
	case "migration":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: app generate migration <name>")
			fmt.Fprintln(os.Stderr, "  name: snake_case description (e.g., add_tags_to_posts)")
			os.Exit(1)
		}
		generateMigration(args[1])
	default:
		fmt.Fprintf(os.Stderr, "unknown generator: %s\n", args[0])
		fmt.Fprintln(os.Stderr, "Valid types: domain, migration")
		os.Exit(1)
	}
}

// domainData holds template data for domain generation.
type domainData struct {
	Package    string // "post"
	Pascal     string // "Post"
	PascalPlur string // "Posts"
	Lower      string // "post"
	LowerPlur  string // "posts"
	Table      string // "posts"
	ModulePath string // "github.com/omaklabs/base"
	Timestamp  string // "20260315143000"
}

func generateDomain(name string) {
	name = strings.ToLower(strings.TrimSpace(name))

	if !isValidName(name) {
		fmt.Fprintf(os.Stderr, "invalid domain name %q: use lowercase letters only (e.g., post, comment, invoice)\n", name)
		os.Exit(1)
	}

	d := domainData{
		Package:    name,
		Pascal:     toPascal(name),
		PascalPlur: toPascal(name) + "s",
		Lower:      name,
		LowerPlur:  name + "s",
		Table:      name + "s",
		ModulePath: getModulePath(),
		Timestamp:  time.Now().Format("20060102150405"),
	}

	domainDir := filepath.Join("internal", d.Package)

	// Check if domain already exists
	if _, err := os.Stat(domainDir); err == nil {
		fmt.Fprintf(os.Stderr, "domain %q already exists at %s\n", name, domainDir)
		os.Exit(1)
	}

	// Create domain directory
	if err := os.MkdirAll(domainDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "creating directory %s: %v\n", domainDir, err)
		os.Exit(1)
	}

	files := []struct {
		path string
		tmpl string
	}{
		{filepath.Join(domainDir, "module.go"), tmplModule},
		{filepath.Join(domainDir, "handler.go"), tmplHandler},
		{filepath.Join(domainDir, "handler_test.go"), tmplHandlerTest},
		{filepath.Join(domainDir, d.LowerPlur+"_list.templ"), tmplList},
		{filepath.Join(domainDir, d.LowerPlur+"_show.templ"), tmplShow},
		{filepath.Join(domainDir, d.LowerPlur+"_form.templ"), tmplForm},
		{filepath.Join("queries", d.LowerPlur+".sql"), tmplQueries},
		{filepath.Join("migrations", d.Timestamp+"_create_"+d.LowerPlur+".sql"), tmplMigration},
	}

	for _, f := range files {
		if err := writeTemplate(f.path, f.tmpl, d); err != nil {
			fmt.Fprintf(os.Stderr, "writing %s: %v\n", f.path, err)
			// Clean up on failure
			os.RemoveAll(domainDir)
			os.Exit(1)
		}
		fmt.Printf("  created  %s\n", f.path)
	}

	// Update app.go to add the module
	if err := addModuleToApp(d); err != nil {
		fmt.Fprintf(os.Stderr, "updating cmd/app/app.go: %v\n", err)
		fmt.Fprintln(os.Stderr, "  add the module manually:")
		fmt.Fprintf(os.Stderr, "    import %q\n", d.ModulePath+"/internal/"+d.Package)
		fmt.Fprintf(os.Stderr, "    add %s.Module to the modules slice\n", d.Package)
	} else {
		fmt.Printf("  updated  cmd/app/app.go\n")
	}

	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  sqlc generate && templ generate && go build ./...")
}

func generateMigration(name string) {
	name = strings.ToLower(strings.TrimSpace(name))

	if !isValidMigrationName(name) {
		fmt.Fprintf(os.Stderr, "invalid migration name %q: use lowercase letters, numbers, and underscores\n", name)
		os.Exit(1)
	}

	timestamp := time.Now().Format("20060102150405")
	filename := filepath.Join("migrations", timestamp+"_"+name+".sql")

	content := "-- +goose Up\n\n\n-- +goose Down\n\n"
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "writing %s: %v\n", filename, err)
		os.Exit(1)
	}

	fmt.Printf("  created  %s\n", filename)
}

// addModuleToApp inserts the import and module entry into cmd/app/app.go.
func addModuleToApp(d domainData) error {
	appFile := filepath.Join("cmd", "app", "app.go")
	data, err := os.ReadFile(appFile)
	if err != nil {
		return err
	}
	content := string(data)

	importPath := fmt.Sprintf("%s/internal/%s", d.ModulePath, d.Package)

	// Add import
	if !strings.Contains(content, importPath) {
		// Find the import block and add the new import before the closing paren
		// Look for the server import line as anchor
		serverImport := fmt.Sprintf("%q", d.ModulePath+"/internal/server")
		newImport := fmt.Sprintf("%q\n\t%s", importPath, serverImport)
		content = strings.Replace(content, serverImport, newImport, 1)
	}

	// Add module to the slice
	moduleEntry := fmt.Sprintf("\t%s.Module,", d.Package)
	if !strings.Contains(content, moduleEntry) {
		// Find the closing brace of the modules slice and insert before it
		content = strings.Replace(content, "}", fmt.Sprintf("\t%s.Module,\n}", d.Package), 1)
	}

	return os.WriteFile(appFile, []byte(content), 0644)
}

func writeTemplate(path, tmplStr string, data domainData) error {
	t, err := template.New("").Parse(tmplStr)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return t.Execute(f, data)
}

func getModulePath() string {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return "github.com/omaklabs/base"
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return "github.com/omaklabs/base"
}

func isValidName(name string) bool {
	return regexp.MustCompile(`^[a-z][a-z0-9]*$`).MatchString(name)
}

func isValidMigrationName(name string) bool {
	return regexp.MustCompile(`^[a-z][a-z0-9_]*$`).MatchString(name)
}

func toPascal(s string) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// --- Templates ---

var tmplModule = `package {{.Package}}

import "{{.ModulePath}}/internal/server"

// Module describes the {{.Lower}} domain.
var Module = server.Module{
	Name:  "{{.LowerPlur}}",
	Path:  "/{{.LowerPlur}}",
	Mount: Mount,
}
`

var tmplHandler = `package {{.Package}}

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"{{.ModulePath}}/internal/db"
	"{{.ModulePath}}/internal/flash"
	"{{.ModulePath}}/internal/middleware"
	"{{.ModulePath}}/internal/pagination"
	"{{.ModulePath}}/internal/server"
	"{{.ModulePath}}/internal/validate"
)

// Mount registers all {{.Lower}} routes on the given router.
func Mount(r chi.Router, deps *server.Deps) {
	r.Use(middleware.RequireAuth)
	r.Get("/", handleList{{.PascalPlur}}(deps))
	r.Get("/new", handleNew{{.Pascal}}())
	r.Post("/", handleCreate{{.Pascal}}(deps))
	r.Get("/{id}", handleShow{{.Pascal}}(deps))
	r.Get("/{id}/edit", handleEdit{{.Pascal}}(deps))
	r.Put("/{id}", handleUpdate{{.Pascal}}(deps))
	r.Post("/{id}", handleUpdate{{.Pascal}}(deps))
	r.Delete("/{id}", handleDelete{{.Pascal}}(deps))
}

// {{.Lower}}IDFromURL parses the {id} URL parameter as an int64.
func {{.Lower}}IDFromURL(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
}

// handleList{{.PascalPlur}} returns a paginated list of {{.LowerPlur}} for the current user.
func handleList{{.PascalPlur}}(deps *server.Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := middleware.CurrentUser(r)

		total, err := deps.Queries.Count{{.PascalPlur}}ByUser(r.Context(), user.ID)
		if err != nil {
			server.RenderError(w, r, err, deps.IsDev)
			return
		}

		p := pagination.FromRequest(r, total)

		{{.LowerPlur}}, err := deps.Queries.List{{.PascalPlur}}(r.Context(), db.List{{.PascalPlur}}Params{
			UserID: user.ID,
			Limit:  int64(p.Limit),
			Offset: int64(p.Offset),
		})
		if err != nil {
			server.RenderError(w, r, err, deps.IsDev)
			return
		}

		if server.IsHTMX(r) {
			{{.PascalPlur}}ListPartial({{.LowerPlur}}, p, "/{{.LowerPlur}}").Render(r.Context(), w)
			return
		}
		{{.PascalPlur}}List({{.LowerPlur}}, p, "/{{.LowerPlur}}").Render(r.Context(), w)
	}
}

// handleShow{{.Pascal}} renders a single {{.Lower}} by ID.
func handleShow{{.Pascal}}(deps *server.Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := {{.Lower}}IDFromURL(r)
		if err != nil {
			server.RenderNotFound(w, r)
			return
		}

		{{.Lower}}, err := deps.Queries.Get{{.Pascal}}(r.Context(), id)
		if err != nil {
			if err == sql.ErrNoRows {
				server.RenderNotFound(w, r)
				return
			}
			server.RenderError(w, r, err, deps.IsDev)
			return
		}

		{{.PascalPlur}}Show({{.Lower}}).Render(r.Context(), w)
	}
}

// handleNew{{.Pascal}} renders an empty {{.Lower}} creation form.
func handleNew{{.Pascal}}() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		{{.PascalPlur}}Form(nil, validate.Errors{}, "", "").Render(r.Context(), w)
	}
}

// handleCreate{{.Pascal}} processes the {{.Lower}} creation form.
func handleCreate{{.Pascal}}(deps *server.Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := middleware.CurrentUser(r)

		if err := r.ParseForm(); err != nil {
			server.RenderError(w, r, err, deps.IsDev)
			return
		}

		title := r.FormValue("title")
		body := r.FormValue("body")

		v := validate.New()
		v.Required("title", title)
		v.MinLength("title", title, 3)

		if v.HasErrors() {
			w.WriteHeader(http.StatusUnprocessableEntity)
			{{.PascalPlur}}Form(nil, v.Errors(), title, body).Render(r.Context(), w)
			return
		}

		{{.Lower}}, err := deps.Queries.Create{{.Pascal}}(r.Context(), db.Create{{.Pascal}}Params{
			UserID: user.ID,
			Title:  title,
			Body:   body,
		})
		if err != nil {
			server.RenderError(w, r, err, deps.IsDev)
			return
		}

		flash.Set(w, "{{.Pascal}} created successfully", "success")
		http.Redirect(w, r, fmt.Sprintf("/{{.LowerPlur}}/%d", {{.Lower}}.ID), http.StatusSeeOther)
	}
}

// handleEdit{{.Pascal}} renders the edit form for an existing {{.Lower}}.
func handleEdit{{.Pascal}}(deps *server.Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := {{.Lower}}IDFromURL(r)
		if err != nil {
			server.RenderNotFound(w, r)
			return
		}

		{{.Lower}}, err := deps.Queries.Get{{.Pascal}}(r.Context(), id)
		if err != nil {
			if err == sql.ErrNoRows {
				server.RenderNotFound(w, r)
				return
			}
			server.RenderError(w, r, err, deps.IsDev)
			return
		}

		{{.PascalPlur}}Form(&{{.Lower}}, validate.Errors{}, {{.Lower}}.Title, {{.Lower}}.Body).Render(r.Context(), w)
	}
}

// handleUpdate{{.Pascal}} processes the {{.Lower}} edit form.
func handleUpdate{{.Pascal}}(deps *server.Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := {{.Lower}}IDFromURL(r)
		if err != nil {
			server.RenderNotFound(w, r)
			return
		}

		{{.Lower}}, err := deps.Queries.Get{{.Pascal}}(r.Context(), id)
		if err != nil {
			if err == sql.ErrNoRows {
				server.RenderNotFound(w, r)
				return
			}
			server.RenderError(w, r, err, deps.IsDev)
			return
		}

		if err := r.ParseForm(); err != nil {
			server.RenderError(w, r, err, deps.IsDev)
			return
		}

		title := r.FormValue("title")
		body := r.FormValue("body")

		v := validate.New()
		v.Required("title", title)
		v.MinLength("title", title, 3)

		if v.HasErrors() {
			w.WriteHeader(http.StatusUnprocessableEntity)
			{{.PascalPlur}}Form(&{{.Lower}}, v.Errors(), title, body).Render(r.Context(), w)
			return
		}

		_, err = deps.Queries.Update{{.Pascal}}(r.Context(), db.Update{{.Pascal}}Params{
			Title: title,
			Body:  body,
			ID:    id,
		})
		if err != nil {
			server.RenderError(w, r, err, deps.IsDev)
			return
		}

		flash.Set(w, "{{.Pascal}} updated successfully", "success")
		http.Redirect(w, r, fmt.Sprintf("/{{.LowerPlur}}/%d", id), http.StatusSeeOther)
	}
}

// handleDelete{{.Pascal}} removes a {{.Lower}} and redirects to the list.
func handleDelete{{.Pascal}}(deps *server.Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := {{.Lower}}IDFromURL(r)
		if err != nil {
			server.RenderNotFound(w, r)
			return
		}

		if err := deps.Queries.Delete{{.Pascal}}(r.Context(), id); err != nil {
			server.RenderError(w, r, err, deps.IsDev)
			return
		}

		flash.Set(w, "{{.Pascal}} deleted successfully", "success")

		// HTMX delete requests: respond with HX-Redirect header
		if server.IsHTMX(r) {
			w.Header().Set("HX-Redirect", "/{{.LowerPlur}}")
			w.WriteHeader(http.StatusOK)
			return
		}

		http.Redirect(w, r, "/{{.LowerPlur}}", http.StatusSeeOther)
	}
}
`

var tmplHandlerTest = `package {{.Package}}

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"{{.ModulePath}}/internal/db"
	"{{.ModulePath}}/internal/middleware"
	"{{.ModulePath}}/internal/server"
	"{{.ModulePath}}/internal/testutil"
)

func testUser(id int64) *db.User {
	return &db.User{
		ID:        id,
		Email:     "test@example.com",
		Name:      "test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func newAuthenticatedRequest(method, target string, body *strings.Reader, user *db.User) *http.Request {
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, target, body)
	} else {
		r = httptest.NewRequest(method, target, nil)
	}
	return middleware.WithUser(r, user)
}

func setup{{.PascalPlur}}Router(deps *server.Deps) *chi.Mux {
	r := chi.NewRouter()
	r.Route("/{{.LowerPlur}}", func(r chi.Router) {
		r.Get("/", handleList{{.PascalPlur}}(deps))
		r.Get("/new", handleNew{{.Pascal}}())
		r.Post("/", handleCreate{{.Pascal}}(deps))
		r.Get("/{id}", handleShow{{.Pascal}}(deps))
		r.Get("/{id}/edit", handleEdit{{.Pascal}}(deps))
		r.Put("/{id}", handleUpdate{{.Pascal}}(deps))
		r.Post("/{id}", handleUpdate{{.Pascal}}(deps))
		r.Delete("/{id}", handleDelete{{.Pascal}}(deps))
	})
	return r
}

func TestList{{.PascalPlur}}Empty(t *testing.T) {
	_, queries, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	user := testutil.CreateTestUser(t, queries, "alice@example.com")
	deps := &server.Deps{Queries: queries, IsDev: true}
	router := setup{{.PascalPlur}}Router(deps)

	req := newAuthenticatedRequest("GET", "/{{.LowerPlur}}", nil, &user)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}
`

var tmplList = `package {{.Package}}

import (
	"fmt"
	"{{.ModulePath}}/internal/db"
	"{{.ModulePath}}/internal/pagination"
	"{{.ModulePath}}/templates/components"
	"{{.ModulePath}}/templates/layouts"
)

templ {{.PascalPlur}}List({{.LowerPlur}} []db.{{.Pascal}}, p pagination.Pagination, baseURL string) {
	@layouts.App() {
		<div class="p-8" id="{{.LowerPlur}}-content" data-paginated>
			@components.FlashMessage()
			@components.PageHeader("{{.PascalPlur}}") {
				@components.LinkButton("primary", "/{{.LowerPlur}}/new") {
					New {{.Pascal}}
				}
			}
			if len({{.LowerPlur}}) == 0 {
				@components.Card() {
					<p class="text-muted-foreground">No {{.LowerPlur}} yet. Create your first {{.Lower}}!</p>
				}
			} else {
				<div class="space-y-4">
					for _, item := range {{.LowerPlur}} {
						@components.Card() {
							<a href={ templ.SafeURL(fmt.Sprintf("/{{.LowerPlur}}/%d", item.ID)) } class="block">
								<h2 class="text-lg font-semibold">{ item.Title }</h2>
								<p class="text-sm text-muted-foreground mt-2">{ item.CreatedAt.Format("Jan 2, 2006 3:04 PM") }</p>
							</a>
						}
					}
				</div>
			}
			<div class="mt-6">
				@components.Pagination(p, baseURL)
			</div>
		</div>
	}
}

templ {{.PascalPlur}}ListPartial({{.LowerPlur}} []db.{{.Pascal}}, p pagination.Pagination, baseURL string) {
	<div class="p-8" id="{{.LowerPlur}}-content" data-paginated>
		@components.FlashMessage()
		@components.PageHeader("{{.PascalPlur}}") {
			@components.LinkButton("primary", "/{{.LowerPlur}}/new") {
				New {{.Pascal}}
			}
		}
		if len({{.LowerPlur}}) == 0 {
			@components.Card() {
				<p class="text-muted-foreground">No {{.LowerPlur}} yet. Create your first {{.Lower}}!</p>
			}
		} else {
			<div class="space-y-4">
				for _, item := range {{.LowerPlur}} {
					@components.Card() {
						<a href={ templ.SafeURL(fmt.Sprintf("/{{.LowerPlur}}/%d", item.ID)) } class="block">
							<h2 class="text-lg font-semibold">{ item.Title }</h2>
							<p class="text-sm text-muted-foreground mt-2">{ item.CreatedAt.Format("Jan 2, 2006 3:04 PM") }</p>
						</a>
					}
				}
			</div>
		}
		<div class="mt-6">
			@components.Pagination(p, baseURL)
		</div>
	</div>
}
`

var tmplShow = `package {{.Package}}

import (
	"fmt"
	"{{.ModulePath}}/internal/db"
	"{{.ModulePath}}/templates/components"
	"{{.ModulePath}}/templates/layouts"
)

templ {{.PascalPlur}}Show({{.Lower}} db.{{.Pascal}}) {
	@layouts.App() {
		<div class="p-8">
			@components.FlashMessage()
			<div class="mb-6">
				<a href="/{{.LowerPlur}}" class="text-primary hover:underline text-sm">&larr; Back to {{.PascalPlur}}</a>
			</div>
			@components.Card() {
				<div class="flex items-start justify-between">
					<h1 class="text-2xl font-bold">{ {{.Lower}}.Title }</h1>
					<div class="flex gap-2">
						@components.LinkButton("ghost", fmt.Sprintf("/{{.LowerPlur}}/%d/edit", {{.Lower}}.ID), "sm") {
							Edit
						}
						<button
							class="inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-lg font-medium transition-colors h-9 px-3 text-sm hover:bg-accent hover:text-accent-foreground cursor-pointer"
							hx-delete={ fmt.Sprintf("/{{.LowerPlur}}/%d", {{.Lower}}.ID) }
							hx-confirm="Are you sure you want to delete this {{.Lower}}?"
						>
							Delete
						</button>
					</div>
				</div>
				<div class="mt-4 whitespace-pre-wrap">{ {{.Lower}}.Body }</div>
				<div class="mt-6 text-sm text-muted-foreground">
					<p>Created: { {{.Lower}}.CreatedAt.Format("Jan 2, 2006 3:04 PM") }</p>
					<p>Updated: { {{.Lower}}.UpdatedAt.Format("Jan 2, 2006 3:04 PM") }</p>
				</div>
			}
		</div>
	}
}
`

var tmplForm = `package {{.Package}}

import (
	"fmt"
	"{{.ModulePath}}/internal/db"
	"{{.ModulePath}}/internal/validate"
	"{{.ModulePath}}/templates/components"
	"{{.ModulePath}}/templates/layouts"
)

templ {{.PascalPlur}}Form({{.Lower}} *db.{{.Pascal}}, errors validate.Errors, title string, body string) {
	@layouts.App() {
		<div class="p-8">
			<div class="mb-6">
				<a href="/{{.LowerPlur}}" class="text-primary hover:underline text-sm">&larr; Back to {{.PascalPlur}}</a>
			</div>
			@components.Card() {
				if {{.Lower}} == nil {
					<h1 class="text-2xl font-bold mb-6">New {{.Pascal}}</h1>
					<form method="POST" action="/{{.LowerPlur}}" class="space-y-4">
						@components.CSRFField()
						@{{.Lower}}FormFields(errors, title, body)
						@components.SubmitButton("primary") {
							Create {{.Pascal}}
						}
					</form>
				} else {
					<h1 class="text-2xl font-bold mb-6">Edit {{.Pascal}}</h1>
					<form method="POST" action={ templ.SafeURL(fmt.Sprintf("/{{.LowerPlur}}/%d", {{.Lower}}.ID)) } class="space-y-4">
						<input type="hidden" name="_method" value="PUT"/>
						@components.CSRFField()
						@{{.Lower}}FormFields(errors, title, body)
						@components.SubmitButton("primary") {
							Update {{.Pascal}}
						}
					</form>
				}
			}
		</div>
	}
}

templ {{.Lower}}FormFields(errors validate.Errors, title string, body string) {
	@components.FormField("Title", "title", "text", title, errors.Error("title"))
	<div>
		@components.Label("body") {
			Body
		}
		@components.TextareaWith(components.TextareaProps{Name: "body", Value: body, Rows: 8, HasError: errors.Error("body") != ""})
		@components.ErrorText(errors.Error("body"))
	</div>
}
`

var tmplQueries = `-- name: List{{.PascalPlur}} :many
SELECT * FROM {{.Table}} WHERE user_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: Count{{.PascalPlur}}ByUser :one
SELECT COUNT(*) as count FROM {{.Table}} WHERE user_id = ?;

-- name: Get{{.Pascal}} :one
SELECT * FROM {{.Table}} WHERE id = ? LIMIT 1;

-- name: Create{{.Pascal}} :one
INSERT INTO {{.Table}} (user_id, title, body) VALUES (?, ?, ?) RETURNING *;

-- name: Update{{.Pascal}} :one
UPDATE {{.Table}} SET title = ?, body = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? RETURNING *;

-- name: Delete{{.Pascal}} :exec
DELETE FROM {{.Table}} WHERE id = ?;
`

var tmplMigration = `-- +goose Up
CREATE TABLE {{.Table}} (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title      TEXT NOT NULL,
    body       TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_{{.Table}}_user_id ON {{.Table}}(user_id);

-- +goose Down
DROP TABLE IF EXISTS {{.Table}};
`
