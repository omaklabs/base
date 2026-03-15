package email

import (
	"context"
	"strings"

	"github.com/omaklabs/base/internal/logger"
)

// DevMailer stores emails in the database and logs them to stdout.
// It never actually sends via SMTP, making it safe for development.
type DevMailer struct {
	store  *Store
	logger *logger.Logger
}

// NewDevMailer returns a DevMailer that persists emails and logs them.
func NewDevMailer(store *Store, log *logger.Logger) *DevMailer {
	return &DevMailer{store: store, logger: log}
}

// Send stores the email in the DB, marks it as sent immediately,
// and logs the details.
func (m *DevMailer) Send(ctx context.Context, msg Message) error {
	id, err := m.store.Create(ctx, msg)
	if err != nil {
		return err
	}

	if err := m.store.MarkSent(ctx, id); err != nil {
		return err
	}

	m.logger.Info("email sent (dev)",
		"id", id,
		"to", strings.Join(msg.To, ", "),
		"subject", msg.Subject,
	)

	return nil
}
