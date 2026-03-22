package main

import (
	"fmt"
	"log"
	"os"
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

	// Show project structure summary
	fmt.Println("\nProject structure:")
	fmt.Println("  cmd/app/             Application entry point and CLI commands")
	fmt.Println("  internal/<domain>/   Domain packages (handler, module, templates)")
	fmt.Println("  internal/db/         SQLC generated code (do not edit)")
	fmt.Println("  internal/middleware/  HTTP middleware")
	fmt.Println("  internal/server/     Shared infrastructure (Deps, Module, error pages)")
	fmt.Println("  templates/layouts/   Layout wrappers (base.templ, app.templ)")
	fmt.Println("  templates/components/Shadcn-style UI components (Button, Card, Input, etc.)")
	fmt.Println("  templates/icons/     Icon system (lucide_gen.go)")
	fmt.Println("  queries/             SQLC query definitions (.sql)")
	fmt.Println("  migrations/          Goose migration files (.sql)")
	fmt.Println("  assets/              Static assets (CSS, JS, images)")
}

