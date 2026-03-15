package email

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/omakase-dev/go-boilerplate/internal/logger"
	"github.com/omakase-dev/go-boilerplate/internal/testutil"
)

func TestDevMailerStoresInDB(t *testing.T) {
	_, queries, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	store := NewStore(queries)
	var buf bytes.Buffer
	log := logger.New(&buf)
	mailer := NewDevMailer(store, log)

	msg := Message{
		To:      []string{"alice@example.com"},
		From:    "sender@example.com",
		Subject: "Welcome!",
		HTML:    "<h1>Hello Alice</h1>",
		Text:    "Hello Alice",
	}

	err := mailer.Send(context.Background(), msg)
	if err != nil {
		t.Fatalf("Send() error: %v", err)
	}

	// Verify the email is stored with status=sent
	emails, err := store.List(context.Background(), "", 10, 0)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(emails) != 1 {
		t.Fatalf("expected 1 email, got %d", len(emails))
	}
	if emails[0].Status != "sent" {
		t.Errorf("status = %q, want %q", emails[0].Status, "sent")
	}
	if emails[0].ToAddr != "alice@example.com" {
		t.Errorf("to_addr = %q, want %q", emails[0].ToAddr, "alice@example.com")
	}
	if emails[0].Subject != "Welcome!" {
		t.Errorf("subject = %q, want %q", emails[0].Subject, "Welcome!")
	}
}

func TestDevMailerLogs(t *testing.T) {
	_, queries, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	store := NewStore(queries)
	var buf bytes.Buffer
	log := logger.New(&buf)
	mailer := NewDevMailer(store, log)

	msg := Message{
		To:      []string{"bob@example.com"},
		From:    "sender@example.com",
		Subject: "Test",
		HTML:    "<p>Hello</p>",
	}

	err := mailer.Send(context.Background(), msg)
	if err != nil {
		t.Fatalf("Send() error: %v", err)
	}

	var entry logger.Entry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to unmarshal log entry: %v", err)
	}

	if entry.Level != "info" {
		t.Errorf("level = %q, want %q", entry.Level, "info")
	}
	if entry.Msg != "email sent (dev)" {
		t.Errorf("msg = %q, want %q", entry.Msg, "email sent (dev)")
	}
	if entry.Fields["to"] != "bob@example.com" {
		t.Errorf("fields[to] = %v, want %q", entry.Fields["to"], "bob@example.com")
	}
	if entry.Fields["subject"] != "Test" {
		t.Errorf("fields[subject] = %v, want %q", entry.Fields["subject"], "Test")
	}
}

func TestDevMailerImplementsMailer(t *testing.T) {
	_, queries, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	store := NewStore(queries)
	var buf bytes.Buffer
	log := logger.New(&buf)

	var _ Mailer = NewDevMailer(store, log)
}

func TestSMTPMailerImplementsMailer(t *testing.T) {
	_, queries, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	store := NewStore(queries)
	var buf bytes.Buffer
	log := logger.New(&buf)

	var _ Mailer = NewSMTPMailer(store, SMTPConfig{}, log)
}

func TestStoreCreate(t *testing.T) {
	_, queries, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	store := NewStore(queries)

	msg := Message{
		To:      []string{"alice@example.com"},
		From:    "sender@example.com",
		ReplyTo: "reply@example.com",
		Subject: "Test",
		HTML:    "<p>Hello</p>",
		Text:    "Hello",
	}

	id, err := store.Create(context.Background(), msg)
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero ID")
	}

	email, err := store.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if email.Status != "pending" {
		t.Errorf("status = %q, want %q", email.Status, "pending")
	}
	if email.ToAddr != "alice@example.com" {
		t.Errorf("to_addr = %q, want %q", email.ToAddr, "alice@example.com")
	}
	if email.FromAddr != "sender@example.com" {
		t.Errorf("from_addr = %q, want %q", email.FromAddr, "sender@example.com")
	}
	if email.ReplyTo != "reply@example.com" {
		t.Errorf("reply_to = %q, want %q", email.ReplyTo, "reply@example.com")
	}
	if email.HtmlBody != "<p>Hello</p>" {
		t.Errorf("html_body = %q, want %q", email.HtmlBody, "<p>Hello</p>")
	}
	if email.TextBody != "Hello" {
		t.Errorf("text_body = %q, want %q", email.TextBody, "Hello")
	}
}

func TestStoreMarkSent(t *testing.T) {
	_, queries, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	store := NewStore(queries)
	ctx := context.Background()

	id, err := store.Create(ctx, Message{
		To:      []string{"alice@example.com"},
		Subject: "Test",
	})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := store.MarkSent(ctx, id); err != nil {
		t.Fatalf("MarkSent() error: %v", err)
	}

	email, err := store.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if email.Status != "sent" {
		t.Errorf("status = %q, want %q", email.Status, "sent")
	}
	if !email.SentAt.Valid {
		t.Error("expected SentAt to be set")
	}
}

func TestStoreMarkFailed(t *testing.T) {
	_, queries, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	store := NewStore(queries)
	ctx := context.Background()

	id, err := store.Create(ctx, Message{
		To:      []string{"alice@example.com"},
		Subject: "Test",
	})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := store.MarkFailed(ctx, id, "connection refused"); err != nil {
		t.Fatalf("MarkFailed() error: %v", err)
	}

	email, err := store.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if email.Status != "failed" {
		t.Errorf("status = %q, want %q", email.Status, "failed")
	}
	if email.Error != "connection refused" {
		t.Errorf("error = %q, want %q", email.Error, "connection refused")
	}
}

func TestStoreList(t *testing.T) {
	_, queries, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	store := NewStore(queries)
	ctx := context.Background()

	// Create 3 emails
	id1, _ := store.Create(ctx, Message{To: []string{"a@example.com"}, Subject: "A"})
	_, _ = store.Create(ctx, Message{To: []string{"b@example.com"}, Subject: "B"})
	_, _ = store.Create(ctx, Message{To: []string{"c@example.com"}, Subject: "C"})

	// Mark one as sent
	_ = store.MarkSent(ctx, id1)

	// List all
	all, err := store.List(ctx, "", 10, 0)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("expected 3 emails, got %d", len(all))
	}

	// List by status=pending
	pending, err := store.List(ctx, "pending", 10, 0)
	if err != nil {
		t.Fatalf("List(pending) error: %v", err)
	}
	if len(pending) != 2 {
		t.Errorf("expected 2 pending emails, got %d", len(pending))
	}

	// List by status=sent
	sent, err := store.List(ctx, "sent", 10, 0)
	if err != nil {
		t.Fatalf("List(sent) error: %v", err)
	}
	if len(sent) != 1 {
		t.Errorf("expected 1 sent email, got %d", len(sent))
	}

	// Test pagination
	page, err := store.List(ctx, "", 2, 0)
	if err != nil {
		t.Fatalf("List(limit=2) error: %v", err)
	}
	if len(page) != 2 {
		t.Errorf("expected 2 emails with limit=2, got %d", len(page))
	}
}

func TestStoreStats(t *testing.T) {
	_, queries, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	store := NewStore(queries)
	ctx := context.Background()

	id1, _ := store.Create(ctx, Message{To: []string{"a@example.com"}, Subject: "A"})
	id2, _ := store.Create(ctx, Message{To: []string{"b@example.com"}, Subject: "B"})
	_, _ = store.Create(ctx, Message{To: []string{"c@example.com"}, Subject: "C"})

	_ = store.MarkSent(ctx, id1)
	_ = store.MarkFailed(ctx, id2, "error")

	stats, err := store.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats() error: %v", err)
	}

	if stats["pending"] != 1 {
		t.Errorf("pending = %d, want 1", stats["pending"])
	}
	if stats["sent"] != 1 {
		t.Errorf("sent = %d, want 1", stats["sent"])
	}
	if stats["failed"] != 1 {
		t.Errorf("failed = %d, want 1", stats["failed"])
	}
}

func TestMessageMultipleRecipients(t *testing.T) {
	_, queries, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	store := NewStore(queries)
	ctx := context.Background()

	msg := Message{
		To:      []string{"alice@example.com", "bob@example.com", "carol@example.com"},
		Subject: "Group email",
	}

	id, err := store.Create(ctx, msg)
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	email, err := store.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}

	expected := strings.Join(msg.To, ", ")
	if email.ToAddr != expected {
		t.Errorf("to_addr = %q, want %q", email.ToAddr, expected)
	}
}
