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
	fmt.Println(`Usage: app <command> [arguments]

Commands:
  serve              Start the HTTP server (default)
  migrate [action]   Manage database migrations
    up               Run all pending migrations (default)
    down             Rollback the last migration
    status           Show migration status
    reset            Drop all tables and re-migrate (dev only)
  generate [type]    Generate code from templates
    domain <name>    Generate a full domain package
    migration <name> Generate a timestamped migration file
  routes             List all registered routes
  seed               Run database seed data
  doctor             Run diagnostic checks on the project
  version            Show version info
  help               Show this help message`)
}
