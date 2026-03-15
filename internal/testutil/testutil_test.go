package testutil

import (
	"context"
	"testing"
)

func TestSetupTestDB(t *testing.T) {
	conn, queries, cleanup := SetupTestDB(t)
	defer cleanup()

	// Verify the connection is alive
	if err := conn.Ping(); err != nil {
		t.Fatalf("Ping() error: %v", err)
	}

	// Verify queries instance is usable (not nil)
	if queries == nil {
		t.Fatal("queries should not be nil")
	}

	// Verify migrations ran by checking that expected tables exist
	tables := []string{"users", "sessions", "jobs"}
	for _, table := range tables {
		var name string
		err := conn.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?",
			table,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %q should exist after migrations: %v", table, err)
		}
	}
}

func TestCreateTestUser(t *testing.T) {
	_, queries, cleanup := SetupTestDB(t)
	defer cleanup()

	email := "alice@example.com"
	user := CreateTestUser(t, queries, email)

	if user.Email != email {
		t.Errorf("user.Email = %q, want %q", user.Email, email)
	}
	if user.Name != "alice" {
		t.Errorf("user.Name = %q, want %q", user.Name, "alice")
	}
	if user.ID == 0 {
		t.Error("user.ID should not be zero")
	}

	// Verify the user is retrievable
	found, err := queries.GetUserByEmail(context.Background(), email)
	if err != nil {
		t.Fatalf("GetUserByEmail() error: %v", err)
	}
	if found.ID != user.ID {
		t.Errorf("found.ID = %d, want %d", found.ID, user.ID)
	}
	if found.Email != email {
		t.Errorf("found.Email = %q, want %q", found.Email, email)
	}
}
