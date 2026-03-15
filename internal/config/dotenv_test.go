package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnv(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")
	content := "APP_NAME=myapp\nAPP_PORT=3000\n"
	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		t.Fatalf("writing .env: %v", err)
	}

	// Ensure vars are not set
	os.Unsetenv("APP_NAME")
	os.Unsetenv("APP_PORT")
	t.Cleanup(func() {
		os.Unsetenv("APP_NAME")
		os.Unsetenv("APP_PORT")
	})

	if err := LoadDotEnv(envFile); err != nil {
		t.Fatalf("LoadDotEnv: %v", err)
	}

	if got := os.Getenv("APP_NAME"); got != "myapp" {
		t.Errorf("APP_NAME = %q, want %q", got, "myapp")
	}
	if got := os.Getenv("APP_PORT"); got != "3000" {
		t.Errorf("APP_PORT = %q, want %q", got, "3000")
	}
}

func TestLoadDotEnvSkipsComments(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")
	content := "# This is a comment\nCOMMENT_TEST_KEY=hello\n# Another comment\n"
	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		t.Fatalf("writing .env: %v", err)
	}

	os.Unsetenv("COMMENT_TEST_KEY")
	t.Cleanup(func() { os.Unsetenv("COMMENT_TEST_KEY") })

	if err := LoadDotEnv(envFile); err != nil {
		t.Fatalf("LoadDotEnv: %v", err)
	}

	if got := os.Getenv("COMMENT_TEST_KEY"); got != "hello" {
		t.Errorf("COMMENT_TEST_KEY = %q, want %q", got, "hello")
	}
}

func TestLoadDotEnvQuotedValues(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")
	content := `DOUBLE_QUOTED="hello world"
SINGLE_QUOTED='foo bar'
UNQUOTED=plain
`
	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		t.Fatalf("writing .env: %v", err)
	}

	os.Unsetenv("DOUBLE_QUOTED")
	os.Unsetenv("SINGLE_QUOTED")
	os.Unsetenv("UNQUOTED")
	t.Cleanup(func() {
		os.Unsetenv("DOUBLE_QUOTED")
		os.Unsetenv("SINGLE_QUOTED")
		os.Unsetenv("UNQUOTED")
	})

	if err := LoadDotEnv(envFile); err != nil {
		t.Fatalf("LoadDotEnv: %v", err)
	}

	if got := os.Getenv("DOUBLE_QUOTED"); got != "hello world" {
		t.Errorf("DOUBLE_QUOTED = %q, want %q", got, "hello world")
	}
	if got := os.Getenv("SINGLE_QUOTED"); got != "foo bar" {
		t.Errorf("SINGLE_QUOTED = %q, want %q", got, "foo bar")
	}
	if got := os.Getenv("UNQUOTED"); got != "plain" {
		t.Errorf("UNQUOTED = %q, want %q", got, "plain")
	}
}

func TestLoadDotEnvExportPrefix(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")
	content := "export EXPORT_TEST_KEY=exported_value\n"
	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		t.Fatalf("writing .env: %v", err)
	}

	os.Unsetenv("EXPORT_TEST_KEY")
	t.Cleanup(func() { os.Unsetenv("EXPORT_TEST_KEY") })

	if err := LoadDotEnv(envFile); err != nil {
		t.Fatalf("LoadDotEnv: %v", err)
	}

	if got := os.Getenv("EXPORT_TEST_KEY"); got != "exported_value" {
		t.Errorf("EXPORT_TEST_KEY = %q, want %q", got, "exported_value")
	}
}

func TestLoadDotEnvDoesNotOverwrite(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")
	content := "OVERWRITE_TEST=from_file\n"
	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		t.Fatalf("writing .env: %v", err)
	}

	t.Setenv("OVERWRITE_TEST", "from_env")

	if err := LoadDotEnv(envFile); err != nil {
		t.Fatalf("LoadDotEnv: %v", err)
	}

	if got := os.Getenv("OVERWRITE_TEST"); got != "from_env" {
		t.Errorf("OVERWRITE_TEST = %q, want %q (should not overwrite)", got, "from_env")
	}
}

func TestLoadDotEnvMissingFile(t *testing.T) {
	err := LoadDotEnv("/tmp/nonexistent-dotenv-file-12345")
	if err != nil {
		t.Errorf("expected nil error for missing file, got: %v", err)
	}
}
