package conventions

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

var projectRoot = findProjectRoot()

func findProjectRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "."
}

// scanFiles walks dir and returns all file paths matching the glob pattern.
func scanFiles(t *testing.T, dir string, pattern string) []string {
	t.Helper()
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err != nil {
			return err
		}
		if matched {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scanning files in %s: %v", dir, err)
	}
	return files
}

// fileContains reads a file and returns true if any of the patterns are found.
func fileContains(t *testing.T, path string, patterns ...string) bool {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading file %s: %v", path, err)
	}
	content := string(data)
	for _, p := range patterns {
		if strings.Contains(content, p) {
			return true
		}
	}
	return false
}

// domainDirs returns all directories under internal/ that contain a handler.go file.
// These are domain packages following the domain-per-package convention.
func domainDirs(t *testing.T) []string {
	t.Helper()
	internalDir := filepath.Join(projectRoot, "internal")
	entries, err := os.ReadDir(internalDir)
	if err != nil {
		t.Fatalf("reading internal/: %v", err)
	}
	var dirs []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		handlerFile := filepath.Join(internalDir, e.Name(), "handler.go")
		if _, err := os.Stat(handlerFile); err == nil {
			dirs = append(dirs, filepath.Join(internalDir, e.Name()))
		}
	}
	return dirs
}

// TestAllMigrationsHaveDown verifies each .sql migration has both +goose Up and +goose Down.
func TestAllMigrationsHaveDown(t *testing.T) {
	migrationsDir := filepath.Join(projectRoot, "migrations")
	files := scanFiles(t, migrationsDir, "*.sql")

	if len(files) == 0 {
		t.Skip("no migration files found")
	}

	for _, f := range files {
		name := filepath.Base(f)
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(f)
			if err != nil {
				t.Fatalf("reading %s: %v", f, err)
			}
			content := string(data)

			if !strings.Contains(content, "-- +goose Up") {
				t.Errorf("%s is missing -- +goose Up", name)
			}
			if !strings.Contains(content, "-- +goose Down") {
				t.Errorf("%s is missing -- +goose Down", name)
			}
		})
	}
}

// TestNoRawSQLInHandlers verifies domain handlers use SQLC queries, not raw SQL.
func TestNoRawSQLInHandlers(t *testing.T) {
	dirs := domainDirs(t)
	if len(dirs) == 0 {
		t.Skip("no domain directories found")
	}

	forbidden := []string{"db.Exec(", "db.Query(", "db.QueryRow(", "sql.Open("}

	for _, dir := range dirs {
		files := scanFiles(t, dir, "*.go")
		for _, f := range files {
			name := filepath.Base(f)
			if strings.HasSuffix(name, "_test.go") {
				continue
			}
			t.Run(filepath.Base(dir)+"/"+name, func(t *testing.T) {
				data, err := os.ReadFile(f)
				if err != nil {
					t.Fatalf("reading %s: %v", f, err)
				}
				content := string(data)
				for _, pattern := range forbidden {
					if strings.Contains(content, pattern) {
						t.Errorf("%s contains raw SQL call %q — use SQLC queries instead", name, pattern)
					}
				}
			})
		}
	}
}

// TestAllHandlerFilesHaveTests verifies each handler file has a corresponding test file.
func TestAllHandlerFilesHaveTests(t *testing.T) {
	// Check domain packages (dirs with handler.go)
	dirs := domainDirs(t)

	// Also check the server package
	serverDir := filepath.Join(projectRoot, "internal", "server")
	if info, err := os.Stat(serverDir); err == nil && info.IsDir() {
		dirs = append(dirs, serverDir)
	}

	if len(dirs) == 0 {
		t.Skip("no handler directories found")
	}

	for _, dir := range dirs {
		files := scanFiles(t, dir, "*.go")
		for _, f := range files {
			name := filepath.Base(f)
			// Skip test files themselves
			if strings.HasSuffix(name, "_test.go") {
				continue
			}
			// Skip pure data-definition files
			if name == "deps.go" || name == "module.go" {
				continue
			}
			// Skip generated templ output files
			if strings.HasSuffix(name, "_templ.go") {
				continue
			}
			t.Run(filepath.Base(dir)+"/"+name, func(t *testing.T) {
				testFile := strings.TrimSuffix(f, ".go") + "_test.go"
				if _, err := os.Stat(testFile); os.IsNotExist(err) {
					t.Errorf("handler %s/%s has no corresponding test file %s", filepath.Base(dir), name, filepath.Base(testFile))
				}
			})
		}
	}
}

// TestTemplFilesHaveGeneratedOutput verifies each .templ file has a _templ.go file.
func TestTemplFilesHaveGeneratedOutput(t *testing.T) {
	// Scan both templates/ (shared components/layouts) and internal/ (co-located domain templates)
	var files []string
	for _, dir := range []string{
		filepath.Join(projectRoot, "templates"),
		filepath.Join(projectRoot, "internal"),
	} {
		files = append(files, scanFiles(t, dir, "*.templ")...)
	}

	if len(files) == 0 {
		t.Skip("no .templ files found")
	}

	for _, f := range files {
		rel, _ := filepath.Rel(projectRoot, f)
		t.Run(rel, func(t *testing.T) {
			generated := strings.TrimSuffix(f, ".templ") + "_templ.go"
			if _, err := os.Stat(generated); os.IsNotExist(err) {
				t.Errorf("missing generated output for %s (expected %s)", rel, filepath.Base(generated))
			}
		})
	}
}

// TestNoFmtPrintInHandlers verifies handlers and middleware use structured logging.
func TestNoFmtPrintInHandlers(t *testing.T) {
	// Collect domain dirs + server + middleware
	dirs := domainDirs(t)
	dirs = append(dirs, filepath.Join(projectRoot, "internal", "server"))
	dirs = append(dirs, filepath.Join(projectRoot, "internal", "middleware"))

	forbidden := []string{"fmt.Print(", "fmt.Println("}

	for _, dir := range dirs {
		files := scanFiles(t, dir, "*.go")
		for _, f := range files {
			name := filepath.Base(f)
			if strings.HasSuffix(name, "_test.go") {
				continue
			}
			t.Run(filepath.Base(dir)+"/"+name, func(t *testing.T) {
				data, err := os.ReadFile(f)
				if err != nil {
					t.Fatalf("reading %s: %v", f, err)
				}
				content := string(data)
				for _, pattern := range forbidden {
					if strings.Contains(content, pattern) {
						t.Errorf("%s contains %q — use structured logger instead", name, pattern)
					}
				}
			})
		}
	}
}

// TestMigrationsAreOrdered verifies migration version numbers are strictly increasing.
// Supports both sequential (NNN_) and timestamp (YYYYMMDDHHMMSS_) prefixes.
func TestMigrationsAreOrdered(t *testing.T) {
	migrationsDir := filepath.Join(projectRoot, "migrations")
	files := scanFiles(t, migrationsDir, "*.sql")

	if len(files) == 0 {
		t.Skip("no migration files found")
	}

	// Extract version numbers from filenames
	re := regexp.MustCompile(`^(\d+)_`)
	var versions []int64
	for _, f := range files {
		name := filepath.Base(f)
		matches := re.FindStringSubmatch(name)
		if matches == nil {
			t.Errorf("migration %s does not follow NNN_name.sql or YYYYMMDDHHMMSS_name.sql pattern", name)
			continue
		}
		n, err := strconv.ParseInt(matches[1], 10, 64)
		if err != nil {
			t.Errorf("migration %s has invalid number prefix: %v", name, err)
			continue
		}
		versions = append(versions, n)
	}

	sort.Slice(versions, func(i, j int) bool { return versions[i] < versions[j] })

	for i := 1; i < len(versions); i++ {
		if versions[i] <= versions[i-1] {
			t.Errorf("migration version %d is not strictly greater than %d", versions[i], versions[i-1])
		}
	}
}

// TestNoGlobalVarsInInternal scans internal/**/*.go for package-level var declarations
// and flags any that aren't explicitly allowed.
func TestNoGlobalVarsInInternal(t *testing.T) {
	internalDir := filepath.Join(projectRoot, "internal")
	files := scanFiles(t, internalDir, "*.go")

	// Allowed patterns for package-level vars:
	// - var _ SomeInterface = ... (interface compliance)
	// - var ErrSomething = ... (sentinel errors)
	// Allowed patterns for package-level vars:
	// - var _ SomeInterface = ... (interface compliance)
	// - var ErrSomething = ... (sentinel errors)
	// - var Module = ... (domain module registration)
	allowedVarRe := regexp.MustCompile(`^var\s+_\s+|^var\s+Err[A-Z]\w*\s|^var\s+Module\s`)

	for _, f := range files {
		rel, _ := filepath.Rel(projectRoot, f)
		name := filepath.Base(f)

		// Skip test files
		if strings.HasSuffix(name, "_test.go") {
			continue
		}

		// Skip generated SQLC files
		if strings.HasSuffix(name, ".sql.go") {
			continue
		}

		// Skip explicitly allowed files
		if rel == filepath.Join("internal", "config", "schema.go") {
			continue
		}
		// Skip templates/icons directory
		if strings.Contains(rel, filepath.Join("templates", "icons")) {
			continue
		}

		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("reading %s: %v", f, err)
		}

		// Track brace depth to identify package-level declarations.
		// Depth 0 = package level; depth > 0 = inside a function/type/etc.
		lines := strings.Split(string(data), "\n")
		braceDepth := 0
		inString := false
		inRawString := false
		inLineComment := false
		inBlockComment := false

		for lineNum, line := range lines {
			inLineComment = false

			// Count braces, accounting for strings and comments
			for i := 0; i < len(line); i++ {
				ch := line[i]

				// Handle block comments
				if inBlockComment {
					if ch == '*' && i+1 < len(line) && line[i+1] == '/' {
						inBlockComment = false
						i++ // skip '/'
					}
					continue
				}

				// Handle line comments
				if inLineComment {
					break
				}

				// Handle raw strings
				if inRawString {
					if ch == '`' {
						inRawString = false
					}
					continue
				}

				// Handle strings
				if inString {
					if ch == '\\' {
						i++ // skip escaped char
					} else if ch == '"' {
						inString = false
					}
					continue
				}

				// Detect comment/string starts
				if ch == '/' && i+1 < len(line) {
					if line[i+1] == '/' {
						inLineComment = true
						break
					}
					if line[i+1] == '*' {
						inBlockComment = true
						i++
						continue
					}
				}
				if ch == '"' {
					inString = true
					continue
				}
				if ch == '`' {
					inRawString = true
					continue
				}
				if ch == '\'' {
					// Skip character literals like '{'
					if i+2 < len(line) && line[i+2] == '\'' {
						i += 2
					}
					continue
				}

				if ch == '{' {
					braceDepth++
				} else if ch == '}' {
					braceDepth--
				}
			}

			// Only check var declarations at package level (braceDepth == 0)
			if braceDepth > 0 {
				continue
			}

			trimmed := strings.TrimSpace(line)

			// Check for var ( blocks at package level
			if strings.HasPrefix(trimmed, "var (") {
				// We'll check the contents of the block in following iterations.
				// The opening brace was already counted above, so braceDepth is now > 0.
				// Lines inside will be at braceDepth > 0 until the closing ).
				// Actually, var ( ... ) uses parens not braces, so we need to handle this differently.
				// Scan ahead for the block contents.
				j := lineNum + 1
				for j < len(lines) {
					blockLine := strings.TrimSpace(lines[j])
					if blockLine == ")" {
						break
					}
					if blockLine != "" && !strings.HasPrefix(blockLine, "//") {
						fields := strings.Fields(blockLine)
						if len(fields) > 0 {
							varName := fields[0]
							if varName != "_" && !strings.HasPrefix(varName, "Err") && varName != "Module" {
								t.Errorf("%s:%d: global var %q — avoid package-level mutable state", rel, j+1, varName)
							}
						}
					}
					j++
				}
				continue
			}

			// Check single-line var declarations at package level
			if strings.HasPrefix(trimmed, "var ") {
				if allowedVarRe.MatchString(trimmed) {
					continue
				}
				t.Errorf("%s:%d: global var declaration — avoid package-level mutable state: %s", rel, lineNum+1, trimmed)
			}
		}
	}
}
