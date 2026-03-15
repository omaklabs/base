// Package email provides the Mailer interface for sending email.
// DevMailer logs emails in development. SMTPMailer sends via SMTP in production.
// All emails are persisted to the database via Store before sending.
package email

import "context"

// Message represents an email to be sent.
type Message struct {
	To      []string
	From    string
	ReplyTo string
	Subject string
	HTML    string
	Text    string
}

// Mailer sends emails. All implementations store the email in the DB first.
type Mailer interface {
	Send(ctx context.Context, msg Message) error
}
