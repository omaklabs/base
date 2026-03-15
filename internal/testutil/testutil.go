package testutil

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pressly/goose/v3"

	"github.com/omakase-dev/go-boilerplate/internal/db"
	"github.com/omakase-dev/go-boilerplate/migrations"
)

// SetupTestDB creates a temporary SQLite database, runs all migrations, and
// returns the raw connection, a Queries instance, and a cleanup function.
// The caller should defer the cleanup function.
func SetupTestDB(t *testing.T) (*sql.DB, *db.Queries, func()) {
	t.Helper()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	conn, err := db.Connect(dbPath)
	if err != nil {
		t.Fatalf("testutil.SetupTestDB: connect: %v", err)
	}

	// Run migrations using the embedded FS
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		conn.Close()
		t.Fatalf("testutil.SetupTestDB: goose dialect: %v", err)
	}

	if err := goose.Up(conn, "."); err != nil {
		conn.Close()
		t.Fatalf("testutil.SetupTestDB: goose up: %v", err)
	}

	queries := db.New(conn)

	cleanup := func() {
		conn.Close()
	}

	return conn, queries, cleanup
}

// CreateTestUser inserts a test user with the given email and a name derived
// from the email prefix. It returns the created user.
func CreateTestUser(t *testing.T, q *db.Queries, email string) db.User {
	t.Helper()

	// Derive name from email prefix (e.g. "alice@example.com" -> "alice")
	name := email
	if idx := strings.Index(email, "@"); idx > 0 {
		name = email[:idx]
	}

	user, err := q.CreateUser(context.Background(), db.CreateUserParams{
		Email: email,
		Name:  name,
	})
	if err != nil {
		t.Fatalf("testutil.CreateTestUser: %v", err)
	}

	return user
}
