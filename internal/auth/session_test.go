package auth

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/omakase-dev/go-boilerplate/internal/db"
	"github.com/omakase-dev/go-boilerplate/migrations"
	"github.com/pressly/goose/v3"
)

// testEnv holds the database objects needed by each test.
type testEnv struct {
	DB      *sql.DB
	Queries *db.Queries
	UserID  int64
}

// setupTestDB creates a fresh SQLite database with migrations applied
// and a test user inserted. The database lives in t.TempDir() so it
// is automatically cleaned up.
func setupTestDB(t *testing.T) testEnv {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")

	conn, err := db.Connect(dbPath)
	if err != nil {
		t.Fatalf("db.Connect: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	// Run goose migrations
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("goose.SetDialect: %v", err)
	}
	if err := goose.Up(conn, "."); err != nil {
		t.Fatalf("goose.Up: %v", err)
	}

	q := db.New(conn)

	// Insert a test user
	user, err := q.CreateUser(context.Background(), db.CreateUserParams{
		Email: "test@example.com",
		Name:  "Test User",
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	return testEnv{
		DB:      conn,
		Queries: q,
		UserID:  user.ID,
	}
}

func TestCreateSession(t *testing.T) {
	env := setupTestDB(t)
	ctx := context.Background()

	token, err := CreateSession(ctx, env.Queries, env.UserID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	if token == "" {
		t.Fatal("expected non-empty token")
	}

	// Token should be 64 hex characters (32 bytes)
	if len(token) != 64 {
		t.Errorf("token length = %d, want 64", len(token))
	}
}

func TestCreateSessionUniqueness(t *testing.T) {
	env := setupTestDB(t)
	ctx := context.Background()

	token1, err := CreateSession(ctx, env.Queries, env.UserID)
	if err != nil {
		t.Fatalf("CreateSession (1): %v", err)
	}

	token2, err := CreateSession(ctx, env.Queries, env.UserID)
	if err != nil {
		t.Fatalf("CreateSession (2): %v", err)
	}

	if token1 == token2 {
		t.Error("two sessions produced identical tokens")
	}
}

func TestValidateSession(t *testing.T) {
	env := setupTestDB(t)
	ctx := context.Background()

	token, err := CreateSession(ctx, env.Queries, env.UserID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	userID, err := ValidateSession(ctx, env.Queries, token)
	if err != nil {
		t.Fatalf("ValidateSession: %v", err)
	}

	if userID != env.UserID {
		t.Errorf("ValidateSession returned userID %d, want %d", userID, env.UserID)
	}
}

func TestValidateSessionInvalidToken(t *testing.T) {
	env := setupTestDB(t)
	ctx := context.Background()

	_, err := ValidateSession(ctx, env.Queries, "nonexistent-token")
	if err == nil {
		t.Fatal("expected error for invalid token, got nil")
	}

	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got: %v", err)
	}
}

func TestDeleteSession(t *testing.T) {
	env := setupTestDB(t)
	ctx := context.Background()

	token, err := CreateSession(ctx, env.Queries, env.UserID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	// Verify it exists
	if _, err := ValidateSession(ctx, env.Queries, token); err != nil {
		t.Fatalf("ValidateSession before delete: %v", err)
	}

	// Delete it
	if err := DeleteSession(ctx, env.Queries, token); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}

	// Verify it no longer validates
	_, err = ValidateSession(ctx, env.Queries, token)
	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound after delete, got: %v", err)
	}
}

func TestDeleteUserSessions(t *testing.T) {
	env := setupTestDB(t)
	ctx := context.Background()

	// Create multiple sessions for the same user
	token1, err := CreateSession(ctx, env.Queries, env.UserID)
	if err != nil {
		t.Fatalf("CreateSession (1): %v", err)
	}
	token2, err := CreateSession(ctx, env.Queries, env.UserID)
	if err != nil {
		t.Fatalf("CreateSession (2): %v", err)
	}

	// Delete all sessions for the user
	if err := DeleteUserSessions(ctx, env.Queries, env.UserID); err != nil {
		t.Fatalf("DeleteUserSessions: %v", err)
	}

	// Both tokens should now be invalid
	if _, err := ValidateSession(ctx, env.Queries, token1); err != ErrSessionNotFound {
		t.Errorf("token1: expected ErrSessionNotFound, got: %v", err)
	}
	if _, err := ValidateSession(ctx, env.Queries, token2); err != ErrSessionNotFound {
		t.Errorf("token2: expected ErrSessionNotFound, got: %v", err)
	}
}

func TestValidateExpiredSession(t *testing.T) {
	env := setupTestDB(t)
	ctx := context.Background()

	// Create a session, then manually backdate its expiry to the past
	token, err := CreateSession(ctx, env.Queries, env.UserID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	// Confirm it works before expiry
	if _, err := ValidateSession(ctx, env.Queries, token); err != nil {
		t.Fatalf("ValidateSession before expiry: %v", err)
	}

	// Backdating the session expiry to 1 hour ago.
	// Must use UTC so the stored string compares correctly with SQLite's CURRENT_TIMESTAMP.
	past := time.Now().UTC().Add(-1 * time.Hour)
	_, err = env.DB.ExecContext(ctx,
		"UPDATE sessions SET expires_at = ? WHERE token = ?",
		past, token,
	)
	if err != nil {
		t.Fatalf("backdating session: %v", err)
	}

	// Now validation should fail
	_, err = ValidateSession(ctx, env.Queries, token)
	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound for expired session, got: %v", err)
	}
}

func TestCleanExpiredSessions(t *testing.T) {
	env := setupTestDB(t)
	ctx := context.Background()

	// Create a session and backdate its expiry
	token, err := CreateSession(ctx, env.Queries, env.UserID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	past := time.Now().UTC().Add(-1 * time.Hour)
	_, err = env.DB.ExecContext(ctx,
		"UPDATE sessions SET expires_at = ? WHERE token = ?",
		past, token,
	)
	if err != nil {
		t.Fatalf("backdating session: %v", err)
	}

	// Clean expired sessions
	if err := CleanExpiredSessions(ctx, env.Queries); err != nil {
		t.Fatalf("CleanExpiredSessions: %v", err)
	}

	// Verify the row is actually gone (not just filtered by query)
	var count int
	err = env.DB.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM sessions WHERE token = ?", token,
	).Scan(&count)
	if err != nil {
		t.Fatalf("counting sessions: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 sessions after cleanup, got %d", count)
	}
}
