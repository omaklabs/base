package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/pressly/goose/v3"

	"github.com/omakase-dev/go-boilerplate/internal/config"
	"github.com/omakase-dev/go-boilerplate/internal/db"
	"github.com/omakase-dev/go-boilerplate/migrations"
)

func cmdDoctor() {
	config.LoadDotEnv(".env")
	cfg := config.Load()

	useColor := isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())

	fmt.Println("Omakase Doctor")
	fmt.Println("==============")
	fmt.Println()

	failures := 0

	pass := func(msg string) {
		if useColor {
			fmt.Printf("  \033[32m✓\033[0m %s\n", msg)
		} else {
			fmt.Printf("  ✓ %s\n", msg)
		}
	}
	fail := func(msg string) {
		failures++
		if useColor {
			fmt.Printf("  \033[31m✗\033[0m %s\n", msg)
		} else {
			fmt.Printf("  ✗ %s\n", msg)
		}
	}

	// 1. Database connection
	dbConn, err := db.Connect(cfg.DatabasePath)
	if err != nil {
		fail(fmt.Sprintf("Database connection failed: %v", err))
	} else {
		defer dbConn.Close()
		pass("Database connected")
	}

	// 2. Migrations up to date
	if dbConn != nil {
		goose.SetBaseFS(migrations.FS)
		if err := goose.SetDialect("sqlite3"); err != nil {
			fail(fmt.Sprintf("Migrations check error: %v", err))
		} else {
			currentVer, err := goose.GetDBVersion(dbConn)
			if err != nil {
				fail(fmt.Sprintf("Migrations check error: %v", err))
			} else {
				// Check for pending migrations by collecting them
				pendingMigrations, err := goose.CollectMigrations(".", 0, goose.MaxVersion)
				if err != nil {
					fail(fmt.Sprintf("Migrations check error: %v", err))
				} else {
					lastMigration, err := pendingMigrations.Last()
					if err != nil {
						// No migrations found at all
						pass("Migrations up to date (no migrations)")
					} else if lastMigration.Version > currentVer {
						pending := 0
						for _, m := range pendingMigrations {
							if m.Version > currentVer {
								pending++
							}
						}
						fail(fmt.Sprintf("Migrations: %d pending migration(s) (current: %d, latest: %d)", pending, currentVer, lastMigration.Version))
					} else {
						pass(fmt.Sprintf("Migrations up to date (version %d)", currentVer))
					}
				}
			}
		}
	} else {
		fail("Migrations: skipped (no database connection)")
	}

	// 3. Required env vars
	if !cfg.IsDev() {
		envOk := true
		if cfg.CSRFKey == "change-me-in-production-32bytes!" {
			fail("CSRF_KEY is still default (change for production)")
			envOk = false
		}
		if cfg.InternalAPIKey == "change-me-in-production" {
			fail("INTERNAL_API_KEY is still default (change for production)")
			envOk = false
		}
		if envOk {
			pass("Environment variables configured")
		}
	} else {
		pass("Environment variables configured (dev defaults OK)")
	}

	// 4. SQLC output
	modelsPath := filepath.Join("internal", "db", "models.go")
	if _, err := os.Stat(modelsPath); os.IsNotExist(err) {
		fail("SQLC output missing: internal/db/models.go not found (run: sqlc generate)")
	} else {
		// Check for at least one .sql.go file
		sqlGoFiles, _ := filepath.Glob(filepath.Join("internal", "db", "*.sql.go"))
		if len(sqlGoFiles) == 0 {
			fail("SQLC output missing: no .sql.go files found in internal/db/ (run: sqlc generate)")
		} else {
			pass("SQLC output present")
		}
	}

	// 5. Templ output
	templFiles := findTemplFiles("templates")
	if len(templFiles) == 0 {
		pass("Templ output up to date (0 templates)")
	} else {
		missingTempl := []string{}
		for _, tf := range templFiles {
			generated := strings.TrimSuffix(tf, ".templ") + "_templ.go"
			if _, err := os.Stat(generated); os.IsNotExist(err) {
				missingTempl = append(missingTempl, tf)
			}
		}
		if len(missingTempl) > 0 {
			for _, mf := range missingTempl {
				fail(fmt.Sprintf("Missing templ output for: %s", mf))
			}
		} else {
			pass(fmt.Sprintf("Templ output up to date (%d templates)", len(templFiles)))
		}
	}

	// 6. Build check
	buildCmd := exec.Command("go", "build", "./...")
	buildOut, err := buildCmd.CombinedOutput()
	if err != nil {
		fail(fmt.Sprintf("Build failed: %s", strings.TrimSpace(string(buildOut))))
	} else {
		pass("Build passes")
	}

	// 7. Test check
	testCmd := exec.Command("go", "test", "-short", "./...")
	testOut, err := testCmd.CombinedOutput()
	if err != nil {
		// Show first line of test output for context
		lines := strings.Split(strings.TrimSpace(string(testOut)), "\n")
		summary := lines[len(lines)-1]
		fail(fmt.Sprintf("Tests failed: %s", summary))
	} else {
		pass("Tests pass")
	}

	// 8. Data directory
	dataDir := "data"
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		fail("Data directory does not exist (expected ./data/)")
	} else {
		tmpFile := filepath.Join(dataDir, ".doctor-check")
		if err := os.WriteFile(tmpFile, []byte("ok"), 0644); err != nil {
			fail(fmt.Sprintf("Data directory not writable: %v", err))
		} else {
			os.Remove(tmpFile)
			pass("Data directory writable")
		}
	}

	// Summary
	fmt.Println()
	if failures > 0 {
		if useColor {
			fmt.Printf("\033[31m%d check(s) failed.\033[0m\n", failures)
		} else {
			fmt.Printf("%d check(s) failed.\n", failures)
		}
		os.Exit(1)
	} else {
		if useColor {
			fmt.Printf("\033[32mAll checks passed.\033[0m\n")
		} else {
			fmt.Println("All checks passed.")
		}
	}
}

// findTemplFiles walks a directory tree and returns all .templ file paths.
func findTemplFiles(root string) []string {
	var files []string
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(path, ".templ") {
			files = append(files, path)
		}
		return nil
	})
	return files
}
