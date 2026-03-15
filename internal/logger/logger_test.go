package logger

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func TestInfoWritesJSON(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf)

	log.Info("server started", "port", 8080)

	var entry Entry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if entry.Level != "info" {
		t.Errorf("level = %q, want %q", entry.Level, "info")
	}
	if entry.Msg != "server started" {
		t.Errorf("msg = %q, want %q", entry.Msg, "server started")
	}
	if entry.Time.IsZero() {
		t.Error("timestamp should not be zero")
	}
	if entry.Fields["port"] != float64(8080) {
		t.Errorf("fields[port] = %v, want %v", entry.Fields["port"], 8080)
	}
}

func TestWarnLevel(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf)

	log.Warn("disk space low")

	var entry Entry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if entry.Level != "warn" {
		t.Errorf("level = %q, want %q", entry.Level, "warn")
	}
	if entry.Msg != "disk space low" {
		t.Errorf("msg = %q, want %q", entry.Msg, "disk space low")
	}
}

func TestErrorLevel(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf)

	log.Error("connection failed", "host", "db.example.com")

	var entry Entry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if entry.Level != "error" {
		t.Errorf("level = %q, want %q", entry.Level, "error")
	}
	if entry.Msg != "connection failed" {
		t.Errorf("msg = %q, want %q", entry.Msg, "connection failed")
	}
	if entry.Fields["host"] != "db.example.com" {
		t.Errorf("fields[host] = %v, want %q", entry.Fields["host"], "db.example.com")
	}
}

func TestFieldsParsing(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf)

	log.Info("request", "method", "GET", "path", "/users", "status", 200)

	var entry Entry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if entry.Fields["method"] != "GET" {
		t.Errorf("fields[method] = %v, want %q", entry.Fields["method"], "GET")
	}
	if entry.Fields["path"] != "/users" {
		t.Errorf("fields[path] = %v, want %q", entry.Fields["path"], "/users")
	}
	if entry.Fields["status"] != float64(200) {
		t.Errorf("fields[status] = %v, want %v", entry.Fields["status"], 200)
	}
}

func TestFieldsOddNumber(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf)

	// Odd number of field args — last key ("orphan") should be ignored
	log.Info("test", "key1", "val1", "orphan")

	var entry Entry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if entry.Fields["key1"] != "val1" {
		t.Errorf("fields[key1] = %v, want %q", entry.Fields["key1"], "val1")
	}
	if _, exists := entry.Fields["orphan"]; exists {
		t.Error("orphan key should not be present in fields")
	}
}

func TestSubscribeReceivesEntries(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf)

	ch := log.Subscribe()
	defer log.Unsubscribe(ch)

	log.Info("hello subscriber", "key", "value")

	select {
	case entry := <-ch:
		if entry.Level != "info" {
			t.Errorf("level = %q, want %q", entry.Level, "info")
		}
		if entry.Msg != "hello subscriber" {
			t.Errorf("msg = %q, want %q", entry.Msg, "hello subscriber")
		}
		if entry.Fields["key"] != "value" {
			t.Errorf("fields[key] = %v, want %q", entry.Fields["key"], "value")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for log entry on subscriber channel")
	}
}

func TestUnsubscribeStopsReceiving(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf)

	ch := log.Subscribe()
	log.Unsubscribe(ch)

	// After unsubscribe, channel should be closed
	_, ok := <-ch
	if ok {
		t.Error("expected channel to be closed after unsubscribe")
	}
}

func TestMultipleSubscribers(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf)

	ch1 := log.Subscribe()
	ch2 := log.Subscribe()
	defer log.Unsubscribe(ch1)
	defer log.Unsubscribe(ch2)

	log.Info("broadcast message")

	for i, ch := range []chan Entry{ch1, ch2} {
		select {
		case entry := <-ch:
			if entry.Msg != "broadcast message" {
				t.Errorf("subscriber %d: msg = %q, want %q", i, entry.Msg, "broadcast message")
			}
		case <-time.After(1 * time.Second):
			t.Fatalf("subscriber %d: timed out waiting for entry", i)
		}
	}
}
