// Package auth provides strategy-agnostic session management.
//
// This layer manages sessions AFTER authentication succeeds.
// It has no opinion on HOW the user authenticates — only that
// a valid userID is provided when creating a session.
package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/omaklabs/base/internal/db"
)

// SessionDuration is how long a session token remains valid.
const SessionDuration = 30 * 24 * time.Hour // 30 days

var (
	ErrSessionNotFound = errors.New("session not found or expired")
	ErrTokenGeneration = errors.New("failed to generate session token")
)

// CreateSession generates a new session for the given user.
// It returns the opaque token that the caller should store (e.g. in a cookie).
func CreateSession(ctx context.Context, q *db.Queries, userID int64) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrTokenGeneration, err)
	}

	_, err = q.CreateSession(ctx, db.CreateSessionParams{
		ID:        uuid.New().String(),
		UserID:    userID,
		Token:     token,
		ExpiresAt: time.Now().UTC().Add(SessionDuration),
	})
	if err != nil {
		return "", fmt.Errorf("creating session: %w", err)
	}

	return token, nil
}

// ValidateSession checks that the token maps to a non-expired session
// and returns the associated userID.
func ValidateSession(ctx context.Context, q *db.Queries, token string) (int64, error) {
	session, err := q.GetSessionByToken(ctx, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrSessionNotFound
		}
		return 0, fmt.Errorf("querying session: %w", err)
	}

	return session.UserID, nil
}

// DeleteSession removes a single session by its token.
func DeleteSession(ctx context.Context, q *db.Queries, token string) error {
	return q.DeleteSession(ctx, token)
}

// DeleteUserSessions removes all sessions for a given user.
func DeleteUserSessions(ctx context.Context, q *db.Queries, userID int64) error {
	return q.DeleteSessionsByUserID(ctx, userID)
}

// CleanExpiredSessions removes all sessions that have passed their expiry.
func CleanExpiredSessions(ctx context.Context, q *db.Queries) error {
	return q.DeleteExpiredSessions(ctx)
}

// generateToken produces a cryptographically random 32-byte hex string (64 chars).
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
