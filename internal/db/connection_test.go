package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConnect(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := Connect(dbPath)
	if err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	defer db.Close()

	// Verify file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatal("database file was not created")
	}

	// Verify connection works
	if err := db.Ping(); err != nil {
		t.Fatalf("Ping() error: %v", err)
	}
}

func TestConnectPragmas(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := Connect(dbPath)
	if err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	defer db.Close()

	tests := []struct {
		pragma   string
		expected string
	}{
		{"PRAGMA journal_mode", "wal"},
		{"PRAGMA synchronous", "1"}, // NORMAL = 1
		{"PRAGMA foreign_keys", "1"},
		{"PRAGMA busy_timeout", "5000"},
		{"PRAGMA temp_store", "2"}, // MEMORY = 2
	}

	for _, tt := range tests {
		t.Run(tt.pragma, func(t *testing.T) {
			var val string
			if err := db.QueryRow(tt.pragma).Scan(&val); err != nil {
				t.Fatalf("querying %s: %v", tt.pragma, err)
			}
			if val != tt.expected {
				t.Errorf("%s = %q, want %q", tt.pragma, val, tt.expected)
			}
		})
	}
}

func TestConnectCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b", "c")
	dbPath := filepath.Join(nested, "test.db")

	db, err := Connect(dbPath)
	if err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	defer db.Close()

	if _, err := os.Stat(nested); os.IsNotExist(err) {
		t.Fatal("nested directory was not created")
	}
}

func TestConnectForeignKeys(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := Connect(dbPath)
	if err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	defer db.Close()

	// Create parent table
	_, err = db.Exec(`CREATE TABLE parents (id INTEGER PRIMARY KEY)`)
	if err != nil {
		t.Fatalf("creating parents table: %v", err)
	}

	// Create child table with FK
	_, err = db.Exec(`CREATE TABLE children (
		id INTEGER PRIMARY KEY,
		parent_id INTEGER NOT NULL REFERENCES parents(id)
	)`)
	if err != nil {
		t.Fatalf("creating children table: %v", err)
	}

	// Insert without parent should fail (FK enforced)
	_, err = db.Exec(`INSERT INTO children (parent_id) VALUES (999)`)
	if err == nil {
		t.Fatal("expected foreign key violation, got nil")
	}
}
