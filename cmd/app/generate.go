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

// Field describes a single field in a generated domain/API.
type Field struct {
	Name      string // "title", "first_name"
	Type      string // "string", "text", "int", "float", "bool", "time", "ref"
	GoType    string // "string", "int64", "float64", "bool", "time.Time"
	SQLType   string // "TEXT NOT NULL DEFAULT ''", "INTEGER NOT NULL DEFAULT 0", etc.
	InputType string // "text", "number", "checkbox", "datetime-local", "textarea", "select"
	IsRef     bool   // true if this is a foreign key reference
	RefTable  string // "users" if field is "user:ref"
	Pascal    string // "Title", "First Name" (display label)
	Column    string // "title", "first_name" (SQL column = Name)
}

func cmdGenerate(args []string) {
	if len(args) == 0 {
		printGenerateUsage()
		os.Exit(1)
	}

	switch args[0] {
	case "domain":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: app generate domain <name> [field:type ...]")
			fmt.Fprintln(os.Stderr, "  name: lowercase singular noun (e.g., post, comment, invoice)")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "Field types: string, text, int, float, bool, time, ref")
			fmt.Fprintln(os.Stderr, "  If no fields given, defaults to title:string body:text")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "Examples:")
			fmt.Fprintln(os.Stderr, "  app generate domain post")
			fmt.Fprintln(os.Stderr, "  app generate domain post title:string body:text published:bool")
			fmt.Fprintln(os.Stderr, "  app generate domain invoice amount:float status:string user:ref")
			os.Exit(1)
		}
		fields := parseFields(args[2:])
		generateDomain(args[1], fields)

	case "api":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: app generate api <name> [field:type ...]")
			fmt.Fprintln(os.Stderr, "  name: lowercase singular noun (e.g., webhook, event)")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "Examples:")
			fmt.Fprintln(os.Stderr, "  app generate api webhook url:string secret:string active:bool")
			os.Exit(1)
		}
		fields := parseFields(args[2:])
		generateAPI(args[1], fields)

	case "page":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: app generate page <domain> <page>")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "Examples:")
			fmt.Fprintln(os.Stderr, "  app generate page notes dashboard")
			fmt.Fprintln(os.Stderr, "  app generate page projects settings")
			os.Exit(1)
		}
		generatePage(args[1], args[2])

	case "component":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: app generate component <name> [prop:type ...]")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "Examples:")
			fmt.Fprintln(os.Stderr, "  app generate component modal")
			fmt.Fprintln(os.Stderr, "  app generate component avatar size:string src:string")
			os.Exit(1)
		}
		props := parseComponentProps(args[2:])
		generateComponent(args[1], props)

	case "job":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: app generate job <name>")
			fmt.Fprintln(os.Stderr, "  name: snake_case job name (e.g., send_welcome_email)")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "Examples:")
			fmt.Fprintln(os.Stderr, "  app generate job send_welcome_email")
			os.Exit(1)
		}
		generateJob(args[1])

	case "middleware":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: app generate middleware <name>")
			fmt.Fprintln(os.Stderr, "  name: snake_case middleware name (e.g., rate_limit)")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "Examples:")
			fmt.Fprintln(os.Stderr, "  app generate middleware rate_limit")
			os.Exit(1)
		}
		generateMiddleware(args[1])

	case "migration":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: app generate migration <name>")
			fmt.Fprintln(os.Stderr, "  name: snake_case description (e.g., add_tags_to_posts)")
			os.Exit(1)
		}
		generateMigration(args[1])

	default:
		fmt.Fprintf(os.Stderr, "unknown generator: %s\n", args[0])
		printGenerateUsage()
		os.Exit(1)
	}
}

func printGenerateUsage() {
	fmt.Fprintln(os.Stderr, "Usage: app generate <type> <name> [options...]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Generators:")
	fmt.Fprintln(os.Stderr, "  domain <name> [field:type ...]      Full CRUD domain (handler, module, templates, queries, migration)")
	fmt.Fprintln(os.Stderr, "  api <name> [field:type ...]          JSON API handlers (no templates)")
	fmt.Fprintln(os.Stderr, "  page <domain> <page>                 Single templ page in an existing domain")
	fmt.Fprintln(os.Stderr, "  component <name> [prop:type ...]     Templ component with Props struct")
	fmt.Fprintln(os.Stderr, "  job <name>                           Background job handler")
	fmt.Fprintln(os.Stderr, "  middleware <name>                    HTTP middleware")
	fmt.Fprintln(os.Stderr, "  migration <name>                     Timestamped migration file")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Field types: string, text, int, float, bool, time, ref")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Examples:")
	fmt.Fprintln(os.Stderr, "  app generate domain post title:string body:text published:bool")
	fmt.Fprintln(os.Stderr, "  app generate api webhook url:string secret:string active:bool")
	fmt.Fprintln(os.Stderr, "  app generate page notes dashboard")
	fmt.Fprintln(os.Stderr, "  app generate component avatar size:string src:string")
	fmt.Fprintln(os.Stderr, "  app generate job send_welcome_email")
	fmt.Fprintln(os.Stderr, "  app generate middleware rate_limit")
	fmt.Fprintln(os.Stderr, "  app generate migration add_tags_to_posts")
}

// parseFields parses "name:type" arguments into Field structs.
// If no arguments are provided, returns default fields (title:string, body:text).
func parseFields(args []string) []Field {
	if len(args) == 0 {
		return []Field{
			fieldFromType("title", "string"),
			fieldFromType("body", "text"),
		}
	}

	var fields []Field
	for _, arg := range args {
		parts := strings.SplitN(arg, ":", 2)
		if len(parts) != 2 {
			fmt.Fprintf(os.Stderr, "invalid field %q: expected name:type (e.g., title:string)\n", arg)
			os.Exit(1)
		}
		name := strings.ToLower(parts[0])
		typ := strings.ToLower(parts[1])
		f := fieldFromType(name, typ)
		fields = append(fields, f)
	}
	return fields
}

// fieldFromType creates a Field from a name and type string.
func fieldFromType(name, typ string) Field {
	f := Field{
		Name:   name,
		Type:   typ,
		Column: name,
		Pascal: snakeToPascalLabel(name),
	}

	switch typ {
	case "string":
		f.GoType = "string"
		f.SQLType = "TEXT NOT NULL DEFAULT ''"
		f.InputType = "text"
	case "text":
		f.GoType = "string"
		f.SQLType = "TEXT NOT NULL DEFAULT ''"
		f.InputType = "textarea"
	case "int":
		f.GoType = "int64"
		f.SQLType = "INTEGER NOT NULL DEFAULT 0"
		f.InputType = "number"
	case "float":
		f.GoType = "float64"
		f.SQLType = "REAL NOT NULL DEFAULT 0"
		f.InputType = "number"
	case "bool":
		f.GoType = "bool"
		f.SQLType = "BOOLEAN NOT NULL DEFAULT FALSE"
		f.InputType = "checkbox"
	case "time":
		f.GoType = "time.Time"
		f.SQLType = "DATETIME"
		f.InputType = "datetime-local"
	case "ref":
		f.GoType = "int64"
		f.IsRef = true
		f.RefTable = pluralize(name)
		f.Column = name + "_id"
		f.SQLType = fmt.Sprintf("INTEGER NOT NULL REFERENCES %s(id) ON DELETE CASCADE", pluralize(name))
		f.InputType = "select"
		f.Pascal = snakeToPascalLabel(name)
	default:
		fmt.Fprintf(os.Stderr, "unknown field type %q for field %q\n", typ, name)
		fmt.Fprintln(os.Stderr, "Valid types: string, text, int, float, bool, time, ref")
		os.Exit(1)
	}

	return f
}

// ComponentProp describes a prop for a generated component.
type ComponentProp struct {
	Name   string // "size"
	Type   string // "string"
	GoType string // "string"
	Pascal string // "Size"
}

// parseComponentProps parses "name:type" arguments into ComponentProp structs.
func parseComponentProps(args []string) []ComponentProp {
	var props []ComponentProp
	for _, arg := range args {
		parts := strings.SplitN(arg, ":", 2)
		if len(parts) != 2 {
			fmt.Fprintf(os.Stderr, "invalid prop %q: expected name:type (e.g., size:string)\n", arg)
			os.Exit(1)
		}
		name := strings.ToLower(parts[0])
		typ := strings.ToLower(parts[1])
		goType := propGoType(typ)
		props = append(props, ComponentProp{
			Name:   name,
			Type:   typ,
			GoType: goType,
			Pascal: snakeToPascalIdent(name),
		})
	}
	return props
}

func propGoType(typ string) string {
	switch typ {
	case "string", "text":
		return "string"
	case "int":
		return "int"
	case "float":
		return "float64"
	case "bool":
		return "bool"
	default:
		return "string"
	}
}

// snakeToPascalLabel converts snake_case to "Title Case" for display labels.
// e.g., "first_name" -> "First Name", "title" -> "Title"
func snakeToPascalLabel(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = string(unicode.ToUpper(rune(p[0]))) + p[1:]
		}
	}
	return strings.Join(parts, " ")
}

// snakeToPascalIdent converts snake_case to PascalCase for Go identifiers.
// e.g., "first_name" -> "FirstName", "title" -> "Title"
func snakeToPascalIdent(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = string(unicode.ToUpper(rune(p[0]))) + p[1:]
		}
	}
	return strings.Join(parts, "")
}

// domainData holds template data for domain generation.
type domainData struct {
	Package    string  // "post"
	Pascal     string  // "Post"
	PascalPlur string  // "Posts"
	Lower      string  // "post"
	LowerPlur  string  // "posts"
	Table      string  // "posts"
	ModulePath string  // "github.com/omaklabs/base"
	Timestamp  string  // "20260315143000"
	Fields     []Field // parsed fields
}

// hasFieldType returns true if any field has the given type.
func (d domainData) HasFieldType(typ string) bool {
	for _, f := range d.Fields {
		if f.Type == typ {
			return true
		}
	}
	return false
}

// ListFields returns up to 3 fields for display in list views.
// Text fields are excluded because they are too long for card summaries.
func (d domainData) ListFields() []Field {
	var filtered []Field
	for _, f := range d.Fields {
		if f.Type == "text" {
			continue
		}
		filtered = append(filtered, f)
		if len(filtered) >= 3 {
			break
		}
	}
	return filtered
}

// FirstStringField returns the name of the first string/text field, or "" if none.
func (d domainData) FirstStringField() string {
	for _, f := range d.Fields {
		if f.Type == "string" || f.Type == "text" {
			return f.Column
		}
	}
	return ""
}

// HasNonRefFields returns true if there's at least one non-ref field.
func (d domainData) HasNonRefFields() bool {
	for _, f := range d.Fields {
		if !f.IsRef {
			return true
		}
	}
	return false
}

func generateDomain(name string, fields []Field) {
	name = strings.ToLower(strings.TrimSpace(name))

	if !isValidName(name) {
		fmt.Fprintf(os.Stderr, "invalid domain name %q: use lowercase letters only (e.g., post, comment, invoice)\n", name)
		os.Exit(1)
	}

	d := domainData{
		Package:    name,
		Pascal:     toPascal(name),
		PascalPlur: toPascal(pluralize(name)),
		Lower:      name,
		LowerPlur:  pluralize(name),
		Table:      pluralize(name),
		ModulePath: getModulePath(),
		Timestamp:  time.Now().Format("20060102150405"),
		Fields:     fields,
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

	// Auto-build: run codegen and compile so the domain works immediately
	fmt.Println()
	autoBuild(true, true)
}

func generateAPI(name string, fields []Field) {
	name = strings.ToLower(strings.TrimSpace(name))

	if !isValidName(name) {
		fmt.Fprintf(os.Stderr, "invalid API name %q: use lowercase letters only (e.g., webhook, event)\n", name)
		os.Exit(1)
	}

	d := domainData{
		Package:    name,
		Pascal:     toPascal(name),
		PascalPlur: toPascal(pluralize(name)),
		Lower:      name,
		LowerPlur:  pluralize(name),
		Table:      pluralize(name),
		ModulePath: getModulePath(),
		Timestamp:  time.Now().Format("20060102150405"),
		Fields:     fields,
	}

	domainDir := filepath.Join("internal", d.Package)

	// Check if domain already exists
	if _, err := os.Stat(domainDir); err == nil {
		fmt.Fprintf(os.Stderr, "api %q already exists at %s\n", name, domainDir)
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
		{filepath.Join(domainDir, "module.go"), tmplAPIModule},
		{filepath.Join(domainDir, "handler.go"), tmplAPIHandler},
		{filepath.Join(domainDir, "handler_test.go"), tmplAPIHandlerTest},
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
	autoBuild(true, false) // sqlc + build, no templ (API has no templates)
}

func generatePage(domain, page string) {
	domain = strings.ToLower(strings.TrimSpace(domain))
	page = strings.ToLower(strings.TrimSpace(page))

	if !isValidName(domain) {
		fmt.Fprintf(os.Stderr, "invalid domain name %q: use lowercase letters only\n", domain)
		os.Exit(1)
	}
	if !isValidName(page) {
		fmt.Fprintf(os.Stderr, "invalid page name %q: use lowercase letters only\n", page)
		os.Exit(1)
	}

	domainDir := filepath.Join("internal", domain)
	if _, err := os.Stat(domainDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "domain %q does not exist at %s\n", domain, domainDir)
		os.Exit(1)
	}

	filename := filepath.Join(domainDir, domain+"_"+page+".templ")
	if _, err := os.Stat(filename); err == nil {
		fmt.Fprintf(os.Stderr, "page already exists at %s\n", filename)
		os.Exit(1)
	}

	data := struct {
		Package    string
		Domain     string
		DomainPasc string
		Page       string
		PagePasc   string
		ModulePath string
	}{
		Package:    domain,
		Domain:     domain,
		DomainPasc: toPascal(domain),
		Page:       page,
		PagePasc:   toPascal(page),
		ModulePath: getModulePath(),
	}

	if err := writeTemplateAny(filename, tmplPage, data); err != nil {
		fmt.Fprintf(os.Stderr, "writing %s: %v\n", filename, err)
		os.Exit(1)
	}
	fmt.Printf("  created  %s\n", filename)

	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Add a route in the domain's handler or module")
	fmt.Println("  2. templ generate && go build ./...")
}

func generateComponent(name string, props []ComponentProp) {
	name = strings.ToLower(strings.TrimSpace(name))

	if !isValidMigrationName(name) {
		fmt.Fprintf(os.Stderr, "invalid component name %q: use lowercase letters, numbers, and underscores\n", name)
		os.Exit(1)
	}

	filename := filepath.Join("templates", "components", name+".templ")
	if _, err := os.Stat(filename); err == nil {
		fmt.Fprintf(os.Stderr, "component already exists at %s\n", filename)
		os.Exit(1)
	}

	data := struct {
		Name   string
		Pascal string
		Props  []ComponentProp
	}{
		Name:   name,
		Pascal: snakeToPascalIdent(name),
		Props:  props,
	}

	if err := writeTemplateAny(filename, tmplComponent, data); err != nil {
		fmt.Fprintf(os.Stderr, "writing %s: %v\n", filename, err)
		os.Exit(1)
	}
	fmt.Printf("  created  %s\n", filename)

	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  templ generate && go build ./...")
}

func generateJob(name string) {
	name = strings.ToLower(strings.TrimSpace(name))

	if !isValidMigrationName(name) {
		fmt.Fprintf(os.Stderr, "invalid job name %q: use lowercase letters, numbers, and underscores\n", name)
		os.Exit(1)
	}

	// Determine package name: strip trailing verb parts, use the noun
	// e.g., "send_welcome_email" -> package "jobs" in internal/jobs
	pkgDir := filepath.Join("internal", "jobs")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "creating directory %s: %v\n", pkgDir, err)
		os.Exit(1)
	}

	filename := filepath.Join(pkgDir, name+".go")
	if _, err := os.Stat(filename); err == nil {
		fmt.Fprintf(os.Stderr, "job already exists at %s\n", filename)
		os.Exit(1)
	}

	data := struct {
		Name       string
		FuncName   string
		TypeConst  string
		ModulePath string
	}{
		Name:       name,
		FuncName:   "Handle" + snakeToPascalIdent(name),
		TypeConst:  snakeToPascalIdent(name),
		ModulePath: getModulePath(),
	}

	if err := writeTemplateAny(filename, tmplJob, data); err != nil {
		fmt.Fprintf(os.Stderr, "writing %s: %v\n", filename, err)
		os.Exit(1)
	}
	fmt.Printf("  created  %s\n", filename)

	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Register the job in your domain's Module.Jobs slice:")
	fmt.Fprintf(os.Stderr, "       server.Job{Type: %q, Handler: jobs.%s}\n", name, data.FuncName)
	fmt.Println("  2. go build ./...")
}

func generateMiddleware(name string) {
	name = strings.ToLower(strings.TrimSpace(name))

	if !isValidMigrationName(name) {
		fmt.Fprintf(os.Stderr, "invalid middleware name %q: use lowercase letters, numbers, and underscores\n", name)
		os.Exit(1)
	}

	mwDir := filepath.Join("internal", "middleware")
	if err := os.MkdirAll(mwDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "creating directory %s: %v\n", mwDir, err)
		os.Exit(1)
	}

	filename := filepath.Join(mwDir, name+".go")
	if _, err := os.Stat(filename); err == nil {
		fmt.Fprintf(os.Stderr, "middleware already exists at %s\n", filename)
		os.Exit(1)
	}

	data := struct {
		Name     string
		FuncName string
	}{
		Name:     name,
		FuncName: snakeToPascalIdent(name),
	}

	if err := writeTemplateAny(filename, tmplMiddleware, data); err != nil {
		fmt.Fprintf(os.Stderr, "writing %s: %v\n", filename, err)
		os.Exit(1)
	}
	fmt.Printf("  created  %s\n", filename)

	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Use the middleware in your router:")
	fmt.Fprintf(os.Stderr, "       r.Use(middleware.%s)\n", data.FuncName)
	fmt.Println("  2. go build ./...")
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
	return writeTemplateAny(path, tmplStr, data)
}

func writeTemplateAny(path, tmplStr string, data any) error {
	funcMap := template.FuncMap{
		"snakeToPascalIdent": snakeToPascalIdent,
	}
	t, err := template.New("").Funcs(funcMap).Parse(tmplStr)
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

// pluralize returns the English plural of a singular noun.
// Handles common suffixes: s/ss/sh/ch/x/z → +es, consonant+y → +ies.
func pluralize(s string) string {
	if s == "" {
		return ""
	}
	if strings.HasSuffix(s, "ss") || strings.HasSuffix(s, "sh") ||
		strings.HasSuffix(s, "ch") || strings.HasSuffix(s, "x") ||
		strings.HasSuffix(s, "z") {
		return s + "es"
	}
	if strings.HasSuffix(s, "s") {
		return s + "es"
	}
	if strings.HasSuffix(s, "y") && len(s) > 1 {
		prev := s[len(s)-2]
		if prev != 'a' && prev != 'e' && prev != 'i' && prev != 'o' && prev != 'u' {
			return s[:len(s)-1] + "ies"
		}
	}
	return s + "s"
}

// --- Domain Templates ---

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
		{{.PascalPlur}}Form(nil, validate.Errors{}{{range .Fields}}, {{if eq .Type "bool"}}false{{else if eq .Type "int"}}int64(0){{else if eq .Type "float"}}float64(0){{else if eq .Type "ref"}}int64(0){{else}}""{{end}}{{end}}).Render(r.Context(), w)
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
{{range .Fields}}
{{- if eq .Type "bool"}}
		{{.Name}} := r.FormValue("{{.Column}}") == "on"
{{- else if eq .Type "int"}}
		{{.Name}}Str := r.FormValue("{{.Column}}")
		{{.Name}}, _ := strconv.ParseInt({{.Name}}Str, 10, 64)
{{- else if eq .Type "float"}}
		{{.Name}}Str := r.FormValue("{{.Column}}")
		{{.Name}}, _ := strconv.ParseFloat({{.Name}}Str, 64)
{{- else if eq .Type "ref"}}
		{{.Name}}IDStr := r.FormValue("{{.Column}}")
		{{.Name}}ID, _ := strconv.ParseInt({{.Name}}IDStr, 10, 64)
{{- else}}
		{{.Name}} := r.FormValue("{{.Column}}")
{{- end}}
{{- end}}

		v := validate.New()
{{- range .Fields}}
{{- if eq .Type "string"}}
		v.Required("{{.Column}}", {{.Name}})
{{- else if eq .Type "text"}}
		v.Required("{{.Column}}", {{.Name}})
{{- end}}
{{- end}}

		if v.HasErrors() {
			w.WriteHeader(http.StatusUnprocessableEntity)
			{{.PascalPlur}}Form(nil, v.Errors(){{range .Fields}}, {{if eq .Type "bool"}}{{.Name}}{{else if eq .Type "int"}}{{.Name}}{{else if eq .Type "float"}}{{.Name}}{{else if eq .Type "ref"}}{{.Name}}ID{{else}}{{.Name}}{{end}}{{end}}).Render(r.Context(), w)
			return
		}

		{{.Lower}}, err := deps.Queries.Create{{.Pascal}}(r.Context(), db.Create{{.Pascal}}Params{
			UserID: user.ID,
{{- range .Fields}}
{{- if .IsRef}}
			{{snakeToPascalIdent .Column}}: {{.Name}}ID,
{{- else}}
			{{snakeToPascalIdent .Column}}: {{.Name}},
{{- end}}
{{- end}}
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

		{{.PascalPlur}}Form(&{{.Lower}}, validate.Errors{}{{range .Fields}}, {{if .IsRef}}{{$.Lower}}.{{snakeToPascalIdent .Column}}{{else}}{{$.Lower}}.{{snakeToPascalIdent .Column}}{{end}}{{end}}).Render(r.Context(), w)
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
{{range .Fields}}
{{- if eq .Type "bool"}}
		{{.Name}} := r.FormValue("{{.Column}}") == "on"
{{- else if eq .Type "int"}}
		{{.Name}}Str := r.FormValue("{{.Column}}")
		{{.Name}}, _ := strconv.ParseInt({{.Name}}Str, 10, 64)
{{- else if eq .Type "float"}}
		{{.Name}}Str := r.FormValue("{{.Column}}")
		{{.Name}}, _ := strconv.ParseFloat({{.Name}}Str, 64)
{{- else if eq .Type "ref"}}
		{{.Name}}IDStr := r.FormValue("{{.Column}}")
		{{.Name}}ID, _ := strconv.ParseInt({{.Name}}IDStr, 10, 64)
{{- else}}
		{{.Name}} := r.FormValue("{{.Column}}")
{{- end}}
{{- end}}

		v := validate.New()
{{- range .Fields}}
{{- if eq .Type "string"}}
		v.Required("{{.Column}}", {{.Name}})
{{- else if eq .Type "text"}}
		v.Required("{{.Column}}", {{.Name}})
{{- end}}
{{- end}}

		if v.HasErrors() {
			w.WriteHeader(http.StatusUnprocessableEntity)
			{{.PascalPlur}}Form(&{{.Lower}}, v.Errors(){{range .Fields}}, {{if eq .Type "bool"}}{{.Name}}{{else if eq .Type "int"}}{{.Name}}{{else if eq .Type "float"}}{{.Name}}{{else if eq .Type "ref"}}{{.Name}}ID{{else}}{{.Name}}{{end}}{{end}}).Render(r.Context(), w)
			return
		}

		_, err = deps.Queries.Update{{.Pascal}}(r.Context(), db.Update{{.Pascal}}Params{
{{- range .Fields}}
{{- if .IsRef}}
			{{snakeToPascalIdent .Column}}: {{.Name}}ID,
{{- else}}
			{{snakeToPascalIdent .Column}}: {{.Name}},
{{- end}}
{{- end}}
			ID: id,
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
{{- range .ListFields}}
{{- if eq .Type "bool"}}
								<p class="text-sm text-muted-foreground">
									{{.Pascal}}:
									if item.{{snakeToPascalIdent .Column}} {
										<span class="text-success">Yes</span>
									} else {
										<span class="text-muted-foreground">No</span>
									}
								</p>
{{- else if eq .Type "int"}}
								<p class="text-sm">{{.Pascal}}: { fmt.Sprintf("%d", item.{{snakeToPascalIdent .Column}}) }</p>
{{- else if eq .Type "float"}}
								<p class="text-sm">{{.Pascal}}: { fmt.Sprintf("%.2f", item.{{snakeToPascalIdent .Column}}) }</p>
{{- else if eq .Type "time"}}
								<p class="text-sm text-muted-foreground">{{.Pascal}}: { item.{{snakeToPascalIdent .Column}}.Format("Jan 2, 2006 3:04 PM") }</p>
{{- else if .IsRef}}
								<p class="text-sm text-muted-foreground">{{.Pascal}} ID: { fmt.Sprintf("%d", item.{{snakeToPascalIdent .Column}}) }</p>
{{- else}}
								<h2 class="text-lg font-semibold">{ item.{{snakeToPascalIdent .Column}} }</h2>
{{- end}}
{{- end}}
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
{{- range .ListFields}}
{{- if eq .Type "bool"}}
							<p class="text-sm text-muted-foreground">
								{{.Pascal}}:
								if item.{{snakeToPascalIdent .Column}} {
									<span class="text-success">Yes</span>
								} else {
									<span class="text-muted-foreground">No</span>
								}
							</p>
{{- else if eq .Type "int"}}
							<p class="text-sm">{{.Pascal}}: { fmt.Sprintf("%d", item.{{snakeToPascalIdent .Column}}) }</p>
{{- else if eq .Type "float"}}
							<p class="text-sm">{{.Pascal}}: { fmt.Sprintf("%.2f", item.{{snakeToPascalIdent .Column}}) }</p>
{{- else if eq .Type "time"}}
							<p class="text-sm text-muted-foreground">{{.Pascal}}: { item.{{snakeToPascalIdent .Column}}.Format("Jan 2, 2006 3:04 PM") }</p>
{{- else if .IsRef}}
							<p class="text-sm text-muted-foreground">{{.Pascal}} ID: { fmt.Sprintf("%d", item.{{snakeToPascalIdent .Column}}) }</p>
{{- else}}
							<h2 class="text-lg font-semibold">{ item.{{snakeToPascalIdent .Column}} }</h2>
{{- end}}
{{- end}}
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
					<h1 class="text-2xl font-bold">{{.Pascal}}</h1>
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
				<div class="mt-4 space-y-2">
{{- range .Fields}}
{{- if eq .Type "bool"}}
					<p>
						<span class="font-semibold">{{.Pascal}}:</span>
						if {{$.Lower}}.{{snakeToPascalIdent .Column}} {
							<span class="text-success">Yes</span>
						} else {
							<span class="text-muted-foreground">No</span>
						}
					</p>
{{- else if eq .Type "int"}}
					<p><span class="font-semibold">{{.Pascal}}:</span> { fmt.Sprintf("%d", {{$.Lower}}.{{snakeToPascalIdent .Column}}) }</p>
{{- else if eq .Type "float"}}
					<p><span class="font-semibold">{{.Pascal}}:</span> { fmt.Sprintf("%.2f", {{$.Lower}}.{{snakeToPascalIdent .Column}}) }</p>
{{- else if eq .Type "time"}}
					<p><span class="font-semibold">{{.Pascal}}:</span> { {{$.Lower}}.{{snakeToPascalIdent .Column}}.Format("Jan 2, 2006 3:04 PM") }</p>
{{- else if .IsRef}}
					<p><span class="font-semibold">{{.Pascal}} ID:</span> { fmt.Sprintf("%d", {{$.Lower}}.{{snakeToPascalIdent .Column}}) }</p>
{{- else if eq .Type "text"}}
					<div class="whitespace-pre-wrap">{ {{$.Lower}}.{{snakeToPascalIdent .Column}} }</div>
{{- else}}
					<p><span class="font-semibold">{{.Pascal}}:</span> { {{$.Lower}}.{{snakeToPascalIdent .Column}} }</p>
{{- end}}
{{- end}}
				</div>
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

templ {{.PascalPlur}}Form({{.Lower}} *db.{{.Pascal}}, errors validate.Errors{{range .Fields}}, {{.Name}} {{.GoType}}{{end}}) {
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
						@{{.Lower}}FormFields(errors{{range .Fields}}, {{.Name}}{{end}})
						@components.SubmitButton("primary") {
							Create {{.Pascal}}
						}
					</form>
				} else {
					<h1 class="text-2xl font-bold mb-6">Edit {{.Pascal}}</h1>
					<form method="POST" action={ templ.SafeURL(fmt.Sprintf("/{{.LowerPlur}}/%d", {{.Lower}}.ID)) } class="space-y-4">
						<input type="hidden" name="_method" value="PUT"/>
						@components.CSRFField()
						@{{.Lower}}FormFields(errors{{range .Fields}}, {{.Name}}{{end}})
						@components.SubmitButton("primary") {
							Update {{.Pascal}}
						}
					</form>
				}
			}
		</div>
	}
}

templ {{.Lower}}FormFields(errors validate.Errors{{range .Fields}}, {{.Name}} {{.GoType}}{{end}}) {
{{- range .Fields}}
{{- if eq .InputType "textarea"}}
	<div>
		@components.Label("{{.Column}}") {
			{{.Pascal}}
		}
		@components.TextareaWith(components.TextareaProps{Name: "{{.Column}}", Value: {{.Name}}, Rows: 8, HasError: errors.Error("{{.Column}}") != ""})
		@components.ErrorText(errors.Error("{{.Column}}"))
	</div>
{{- else if eq .InputType "checkbox"}}
	<div class="flex items-center gap-2">
		<input type="checkbox" id="{{.Column}}" name="{{.Column}}" checked?={ {{.Name}} } class="rounded border-input"/>
		@components.Label("{{.Column}}") {
			{{.Pascal}}
		}
		@components.ErrorText(errors.Error("{{.Column}}"))
	</div>
{{- else if eq .InputType "select"}}
	@components.FormField("{{.Pascal}}", "{{.Column}}", "number", fmt.Sprintf("%d", {{.Name}}), errors.Error("{{.Column}}"))
{{- else if eq .InputType "number"}}
	{{- if eq .GoType "int64"}}
	@components.FormField("{{.Pascal}}", "{{.Column}}", "number", fmt.Sprintf("%d", {{.Name}}), errors.Error("{{.Column}}"))
	{{- else}}
	@components.FormField("{{.Pascal}}", "{{.Column}}", "number", fmt.Sprintf("%g", {{.Name}}), errors.Error("{{.Column}}"))
	{{- end}}
{{- else if eq .InputType "datetime-local"}}
	@components.FormField("{{.Pascal}}", "{{.Column}}", "datetime-local", {{.Name}}.Format("2006-01-02T15:04"), errors.Error("{{.Column}}"))
{{- else}}
	@components.FormField("{{.Pascal}}", "{{.Column}}", "{{.InputType}}", {{.Name}}, errors.Error("{{.Column}}"))
{{- end}}
{{- end}}
}
`

var tmplQueries = `-- name: List{{.PascalPlur}} :many
SELECT * FROM {{.Table}} WHERE user_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: Count{{.PascalPlur}}ByUser :one
SELECT COUNT(*) as count FROM {{.Table}} WHERE user_id = ?;

-- name: Get{{.Pascal}} :one
SELECT * FROM {{.Table}} WHERE id = ? LIMIT 1;

-- name: Create{{.Pascal}} :one
INSERT INTO {{.Table}} (user_id{{range .Fields}}, {{.Column}}{{end}}) VALUES (?{{range .Fields}}, ?{{end}}) RETURNING *;

-- name: Update{{.Pascal}} :one
UPDATE {{.Table}} SET {{range $i, $f := .Fields}}{{if $i}}, {{end}}{{$f.Column}} = ?{{end}}, updated_at = CURRENT_TIMESTAMP WHERE id = ? RETURNING *;

-- name: Delete{{.Pascal}} :exec
DELETE FROM {{.Table}} WHERE id = ?;
`

var tmplMigration = `-- +goose Up
CREATE TABLE {{.Table}} (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
{{- range .Fields}}
    {{.Column}} {{.SQLType}},
{{- end}}
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_{{.Table}}_user_id ON {{.Table}}(user_id);

-- +goose Down
DROP TABLE IF EXISTS {{.Table}};
`

// --- API Templates ---

var tmplAPIModule = `package {{.Package}}

import "{{.ModulePath}}/internal/server"

// Module describes the {{.Lower}} API domain.
var Module = server.Module{
	Name:  "{{.LowerPlur}}",
	Path:  "/api/{{.LowerPlur}}",
	Mount: Mount,
}
`

var tmplAPIHandler = `package {{.Package}}

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"{{.ModulePath}}/internal/db"
	"{{.ModulePath}}/internal/middleware"
	"{{.ModulePath}}/internal/pagination"
	"{{.ModulePath}}/internal/server"
)

// Mount registers all {{.Lower}} API routes on the given router.
func Mount(r chi.Router, deps *server.Deps) {
	r.Use(middleware.RequireAuth)
	r.Get("/", handleList{{.PascalPlur}}(deps))
	r.Post("/", handleCreate{{.Pascal}}(deps))
	r.Get("/{id}", handleGet{{.Pascal}}(deps))
	r.Put("/{id}", handleUpdate{{.Pascal}}(deps))
	r.Delete("/{id}", handleDelete{{.Pascal}}(deps))
}

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, msg string) {
	respondJSON(w, status, map[string]string{"error": msg})
}

// {{.Lower}}IDFromURL parses the {id} URL parameter as an int64.
func {{.Lower}}IDFromURL(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
}

func handleList{{.PascalPlur}}(deps *server.Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := middleware.CurrentUser(r)

		total, err := deps.Queries.Count{{.PascalPlur}}ByUser(r.Context(), user.ID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "failed to count {{.LowerPlur}}")
			return
		}

		p := pagination.FromRequest(r, total)

		{{.LowerPlur}}, err := deps.Queries.List{{.PascalPlur}}(r.Context(), db.List{{.PascalPlur}}Params{
			UserID: user.ID,
			Limit:  int64(p.Limit),
			Offset: int64(p.Offset),
		})
		if err != nil {
			respondError(w, http.StatusInternalServerError, "failed to list {{.LowerPlur}}")
			return
		}

		respondJSON(w, http.StatusOK, map[string]any{
			"data":  {{.LowerPlur}},
			"total": total,
			"page":  p.Page,
			"limit": p.Limit,
		})
	}
}

func handleGet{{.Pascal}}(deps *server.Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := {{.Lower}}IDFromURL(r)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid id")
			return
		}

		{{.Lower}}, err := deps.Queries.Get{{.Pascal}}(r.Context(), id)
		if err != nil {
			if err == sql.ErrNoRows {
				respondError(w, http.StatusNotFound, "{{.Lower}} not found")
				return
			}
			respondError(w, http.StatusInternalServerError, "failed to get {{.Lower}}")
			return
		}

		respondJSON(w, http.StatusOK, {{.Lower}})
	}
}

type create{{.Pascal}}Request struct {
{{- range .Fields}}
{{- if .IsRef}}
	{{snakeToPascalIdent .Column}} {{.GoType}} ` + "`" + `json:"{{.Column}}"` + "`" + `
{{- else}}
	{{snakeToPascalIdent .Column}} {{.GoType}} ` + "`" + `json:"{{.Column}}"` + "`" + `
{{- end}}
{{- end}}
}

func handleCreate{{.Pascal}}(deps *server.Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := middleware.CurrentUser(r)

		var req create{{.Pascal}}Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		{{.Lower}}, err := deps.Queries.Create{{.Pascal}}(r.Context(), db.Create{{.Pascal}}Params{
			UserID: user.ID,
{{- range .Fields}}
			{{snakeToPascalIdent .Column}}: req.{{snakeToPascalIdent .Column}},
{{- end}}
		})
		if err != nil {
			respondError(w, http.StatusInternalServerError, "failed to create {{.Lower}}")
			return
		}

		respondJSON(w, http.StatusCreated, {{.Lower}})
	}
}

func handleUpdate{{.Pascal}}(deps *server.Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := {{.Lower}}IDFromURL(r)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid id")
			return
		}

		var req create{{.Pascal}}Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		{{.Lower}}, err := deps.Queries.Update{{.Pascal}}(r.Context(), db.Update{{.Pascal}}Params{
{{- range .Fields}}
			{{snakeToPascalIdent .Column}}: req.{{snakeToPascalIdent .Column}},
{{- end}}
			ID: id,
		})
		if err != nil {
			if err == sql.ErrNoRows {
				respondError(w, http.StatusNotFound, "{{.Lower}} not found")
				return
			}
			respondError(w, http.StatusInternalServerError, "failed to update {{.Lower}}")
			return
		}

		respondJSON(w, http.StatusOK, {{.Lower}})
	}
}

func handleDelete{{.Pascal}}(deps *server.Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := {{.Lower}}IDFromURL(r)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid id")
			return
		}

		if err := deps.Queries.Delete{{.Pascal}}(r.Context(), id); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to delete {{.Lower}}")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
`

var tmplAPIHandlerTest = `package {{.Package}}

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
	r.Header.Set("Content-Type", "application/json")
	return middleware.WithUser(r, user)
}

func setup{{.PascalPlur}}Router(deps *server.Deps) *chi.Mux {
	r := chi.NewRouter()
	r.Route("/api/{{.LowerPlur}}", func(r chi.Router) {
		r.Get("/", handleList{{.PascalPlur}}(deps))
		r.Post("/", handleCreate{{.Pascal}}(deps))
		r.Get("/{id}", handleGet{{.Pascal}}(deps))
		r.Put("/{id}", handleUpdate{{.Pascal}}(deps))
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

	req := newAuthenticatedRequest("GET", "/api/{{.LowerPlur}}", nil, &user)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}
`

// --- Page Template ---

var tmplPage = `package {{.Package}}

import "{{.ModulePath}}/templates/layouts"

templ {{.DomainPasc}}{{.PagePasc}}() {
	@layouts.App() {
		<div class="p-8">
			<h1 class="text-2xl font-bold">{{.PagePasc}}</h1>
		</div>
	}
}
`

// --- Component Template ---

var tmplComponent = `package components
{{if .Props}}
// {{.Pascal}}Props configures the {{.Pascal}} component.
type {{.Pascal}}Props struct {
{{- range .Props}}
	{{.Pascal}} {{.GoType}}
{{- end}}
}

templ {{.Pascal}}(props {{.Pascal}}Props) {
	<div>
		// TODO: implement {{.Pascal}} component
	</div>
}
{{else}}
templ {{.Pascal}}() {
	<div>
		// TODO: implement {{.Pascal}} component
	</div>
}
{{end}}`

// --- Job Template ---

var tmplJob = `package jobs

import "context"

// {{.FuncName}} processes "{{.Name}}" jobs.
func {{.FuncName}}(ctx context.Context, payload []byte) error {
	// TODO: implement
	return nil
}
`

// --- Middleware Template ---

var tmplMiddleware = `package middleware

import "net/http"

// {{.FuncName}} is a middleware that TODO: describe what it does.
func {{.FuncName}}(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: implement
		next.ServeHTTP(w, r)
	})
}
`

// autoBuild runs codegen and compiles the project after scaffolding.
func autoBuild(runTempl, runSqlc bool) {
	// stub — full implementation removed during refactoring
	fmt.Println("Skipping auto-build (stub)")
}
