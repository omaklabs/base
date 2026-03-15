package email

import (
	"context"
	"strings"

	"github.com/omakase-dev/go-boilerplate/internal/db"
)

// Store provides email persistence backed by the database.
type Store struct {
	queries *db.Queries
}

// NewStore creates a new email Store.
func NewStore(queries *db.Queries) *Store {
	return &Store{queries: queries}
}

// Create inserts an email record with status=pending and returns the email ID.
func (s *Store) Create(ctx context.Context, msg Message) (int64, error) {
	email, err := s.queries.CreateEmail(ctx, db.CreateEmailParams{
		ToAddr:   strings.Join(msg.To, ", "),
		FromAddr: msg.From,
		ReplyTo:  msg.ReplyTo,
		Subject:  msg.Subject,
		HtmlBody: msg.HTML,
		TextBody: msg.Text,
	})
	if err != nil {
		return 0, err
	}
	return email.ID, nil
}

// MarkSent updates the email status to sent.
func (s *Store) MarkSent(ctx context.Context, id int64) error {
	return s.queries.MarkEmailSent(ctx, id)
}

// MarkFailed updates the email status to failed with an error message.
func (s *Store) MarkFailed(ctx context.Context, id int64, errMsg string) error {
	return s.queries.MarkEmailFailed(ctx, db.MarkEmailFailedParams{
		Error: errMsg,
		ID:    id,
	})
}

// List returns emails filtered by status with pagination.
func (s *Store) List(ctx context.Context, status string, limit, offset int) ([]db.Email, error) {
	return s.queries.ListEmails(ctx, db.ListEmailsParams{
		FilterStatus: status,
		QueryLimit:   int64(limit),
		QueryOffset:  int64(offset),
	})
}

// Get returns a single email by ID.
func (s *Store) Get(ctx context.Context, id int64) (db.Email, error) {
	return s.queries.GetEmail(ctx, id)
}

// Stats returns email counts grouped by status.
func (s *Store) Stats(ctx context.Context) (map[string]int64, error) {
	rows, err := s.queries.CountEmailsByStatus(ctx)
	if err != nil {
		return nil, err
	}

	stats := make(map[string]int64)
	for _, row := range rows {
		stats[row.Status] = row.Count
	}
	return stats, nil
}
