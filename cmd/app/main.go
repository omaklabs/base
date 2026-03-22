package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// version is set at build time via ldflags
var version = "dev"

func main() {
	cmd := "serve"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	switch cmd {
	case "serve":
		cmdServe()
	case "migrate":
		subcmd := ""
		if len(os.Args) > 2 {
			subcmd = os.Args[2]
		}
		cmdMigrate(subcmd)
	case "generate":
		cmdGenerate(os.Args[2:])
	case "routes":
		cmdRoutes()
	case "seed":
		cmdSeed()
	case "doctor":
		cmdDoctor()
	case "version":
		cmdVersion()
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	log.SetFlags(0)
	fmt.Printf("Omakase App — Go web application framework (version %s)\n\n", version)

	fmt.Println(`Usage: app <command> [arguments]

Commands:
  serve                Start the HTTP server (default)
  migrate [action]     Manage database migrations
    up                 Run all pending migrations (default)
    down               Rollback the last migration
    status             Show migration status
    reset              Drop all tables and re-migrate (dev only)
  generate [type]      Generate code from templates (see below)
  routes               List all registered routes
  seed                 Run database seed data
  doctor               Run diagnostic checks (db, build, tests, codegen)
  version              Show version and build info
  help                 Show this help message

Generators (app generate <type> <name> [options]):
  domain <name> [field:type ...]     Full CRUD domain (handler, templates, queries, migration, tests)
  api <name> [field:type ...]        JSON API handlers (no templates)
  page <domain> <page>               Single templ page in an existing domain
  component <name> [prop:type ...]   Templ component with Props struct
  job <name>                         Background job handler
  middleware <name>                  HTTP middleware
  migration <name>                   Timestamped migration file

  Field types: string, text, int, float, bool, time, ref
  Examples:
    app generate domain post title:string body:text published:bool
    app generate api webhook url:string secret:string active:bool
    app generate page notes dashboard
    app generate component avatar size:string src:string
    app generate job send_welcome_email
    app generate middleware rate_limit

Build commands:
  make build           Full build (generate + css + go build)
  make dev             Build and start with hot reload (air)
  make generate        Run templ generate + sqlc generate
  make css             Compile Tailwind CSS for production
  make test            Run all tests`)

	// Show registered modules
	if len(modules) > 0 {
		fmt.Println("\nRegistered modules:")
		for _, m := range modules {
			fmt.Printf("  %-20s %s\n", m.Name, m.Path)
		}
	}

	// Scan and show domains
	if domains := scanDomains(); len(domains) > 0 {
		fmt.Println("\nDomain packages:")
		for _, d := range domains {
			fmt.Printf("  %s\n", d)
		}
	}

	// Scan and show components
	if comps := scanDir("templates/components", ".templ"); len(comps) > 0 {
		fmt.Printf("\nComponents (%d): %s\n", len(comps), strings.Join(comps, ", "))
	}

	// Scan and show queries
	if queries := scanDir("queries", ".sql"); len(queries) > 0 {
		fmt.Printf("Queries (%d): %s\n", len(queries), strings.Join(queries, ", "))
	}

	// Scan and show migrations
	if migs := scanDir("migrations", ".sql"); len(migs) > 0 {
		fmt.Printf("Migrations: %d total", len(migs))
		if len(migs) > 0 {
			fmt.Printf(" (latest: %s)", migs[len(migs)-1])
		}
		fmt.Println()
	}

	// Show project structure
	fmt.Println("\nProject structure:")
	fmt.Println("  cmd/app/              CLI commands and app entry point")
	fmt.Println("  internal/<domain>/    Domain packages (handler + templates)")
	fmt.Println("  internal/db/          SQLC generated code (do not edit)")
	fmt.Println("  internal/middleware/   HTTP middleware")
	fmt.Println("  internal/server/      Shared infrastructure")
	fmt.Println("  templates/layouts/    Layout wrappers")
	fmt.Println("  templates/components/ UI components (shadcn-style)")
	fmt.Println("  templates/icons/      Icon system")
	fmt.Println("  queries/              SQLC query definitions")
	fmt.Println("  migrations/           Goose migrations")
	fmt.Println("  assets/               Static assets (CSS, JS)")
}

// scanDomains finds domain packages by looking for module.go in internal/*/.
func scanDomains() []string {
	var domains []string
	entries, err := os.ReadDir("internal")
	if err != nil {
		return nil
	}
	skip := map[string]bool{"db": true, "server": true, "middleware": true, "config": true, "view": true, "flash": true, "validate": true, "pagination": true, "testutil": true, "storage": true, "email": true, "jobs": true, "logger": true, "auth": true}
	for _, e := range entries {
		if !e.IsDir() || skip[e.Name()] {
			continue
		}
		if _, err := os.Stat(filepath.Join("internal", e.Name(), "module.go")); err == nil {
			domains = append(domains, e.Name())
		}
	}
	sort.Strings(domains)
	return domains
}

// scanDir lists files with a given extension in a directory, returning base names without extension.
func scanDir(dir, ext string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ext) {
			continue
		}
		// Skip generated files
		if strings.HasSuffix(e.Name(), "_templ.go") || strings.HasSuffix(e.Name(), ".sql.go") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ext)
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

