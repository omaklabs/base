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
