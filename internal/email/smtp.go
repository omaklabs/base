package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strings"

	"github.com/omakase-dev/go-boilerplate/internal/logger"
)

// SMTPConfig holds the SMTP connection settings.
type SMTPConfig struct {
	Host       string
	Port       int
	Username   string
	Password   string
	Encryption string // "tls", "starttls", "none"
}

// SMTPMailer stores emails in the database and sends them via SMTP.
type SMTPMailer struct {
	store  *Store
	config SMTPConfig
	logger *logger.Logger
}

// NewSMTPMailer returns an SMTPMailer.
func NewSMTPMailer(store *Store, cfg SMTPConfig, log *logger.Logger) *SMTPMailer {
	return &SMTPMailer{store: store, config: cfg, logger: log}
}

// Send stores the email in the DB, sends it via SMTP, and updates the status.
func (m *SMTPMailer) Send(ctx context.Context, msg Message) error {
	id, err := m.store.Create(ctx, msg)
	if err != nil {
		return err
	}

	if err := m.sendSMTP(msg); err != nil {
		_ = m.store.MarkFailed(ctx, id, err.Error())
		return err
	}

	if err := m.store.MarkSent(ctx, id); err != nil {
		return err
	}

	m.logger.Info("email sent",
		"id", id,
		"to", strings.Join(msg.To, ", "),
		"subject", msg.Subject,
	)

	return nil
}

// sendSMTP delivers the email via SMTP.
func (m *SMTPMailer) sendSMTP(msg Message) error {
	addr := fmt.Sprintf("%s:%d", m.config.Host, m.config.Port)
	body := buildMIMEMessage(msg)

	var auth smtp.Auth
	if m.config.Username != "" {
		auth = smtp.PlainAuth("", m.config.Username, m.config.Password, m.config.Host)
	}

	switch m.config.Encryption {
	case "tls":
		return m.sendTLS(addr, auth, msg.From, msg.To, body)
	case "starttls":
		return smtp.SendMail(addr, auth, msg.From, msg.To, body)
	default: // "none"
		return m.sendPlain(addr, auth, msg.From, msg.To, body)
	}
}

// sendTLS connects over implicit TLS (e.g. port 465).
func (m *SMTPMailer) sendTLS(addr string, auth smtp.Auth, from string, to []string, body []byte) error {
	tlsCfg := &tls.Config{ServerName: m.config.Host}

	conn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, m.config.Host)
	if err != nil {
		return fmt.Errorf("smtp new client: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}
	for _, rcpt := range to {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("smtp rcpt %s: %w", rcpt, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write(body); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}

	return client.Quit()
}

// sendPlain connects without encryption.
func (m *SMTPMailer) sendPlain(addr string, auth smtp.Auth, from string, to []string, body []byte) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()

	host := m.config.Host
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("smtp new client: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}
	for _, rcpt := range to {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("smtp rcpt %s: %w", rcpt, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write(body); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}

	return client.Quit()
}

// buildMIMEMessage constructs an RFC 2045 multipart/alternative message
// with both plain text and HTML parts.
func buildMIMEMessage(msg Message) []byte {
	boundary := "==OMAKASE_BOUNDARY=="

	var b strings.Builder
	b.WriteString("From: " + msg.From + "\r\n")
	b.WriteString("To: " + strings.Join(msg.To, ", ") + "\r\n")
	if msg.ReplyTo != "" {
		b.WriteString("Reply-To: " + msg.ReplyTo + "\r\n")
	}
	b.WriteString("Subject: " + mime.QEncoding.Encode("utf-8", msg.Subject) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: multipart/alternative; boundary=\"" + boundary + "\"\r\n")
	b.WriteString("\r\n")

	// Plain text part
	if msg.Text != "" {
		b.WriteString("--" + boundary + "\r\n")
		b.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
		b.WriteString("Content-Transfer-Encoding: 7bit\r\n")
		b.WriteString("\r\n")
		b.WriteString(msg.Text + "\r\n")
	}

	// HTML part
	if msg.HTML != "" {
		b.WriteString("--" + boundary + "\r\n")
		b.WriteString("Content-Type: text/html; charset=\"utf-8\"\r\n")
		b.WriteString("Content-Transfer-Encoding: 7bit\r\n")
		b.WriteString("\r\n")
		b.WriteString(msg.HTML + "\r\n")
	}

	b.WriteString("--" + boundary + "--\r\n")

	return []byte(b.String())
}
