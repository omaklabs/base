package logger

import (
	"encoding/json"
	"io"
	"os"
	"sync"
	"time"
)

// Entry represents a single structured log entry.
type Entry struct {
	Level  string         `json:"level"`
	Msg    string         `json:"msg"`
	Fields map[string]any `json:"fields,omitempty"`
	Time   time.Time      `json:"ts"`
}

// Logger is a structured JSON logger with fan-out to SSE subscribers.
type Logger struct {
	mu          sync.RWMutex
	subscribers map[chan Entry]struct{}
	output      io.Writer
}

// New creates a new Logger that writes JSON lines to the given output.
// If output is nil, os.Stdout is used.
func New(output io.Writer) *Logger {
	if output == nil {
		output = os.Stdout
	}
	return &Logger{
		subscribers: make(map[chan Entry]struct{}),
		output:      output,
	}
}

// Info logs a message at info level.
func (l *Logger) Info(msg string, fields ...any) {
	l.log("info", msg, fields...)
}

// Warn logs a message at warn level.
func (l *Logger) Warn(msg string, fields ...any) {
	l.log("warn", msg, fields...)
}

// Error logs a message at error level.
func (l *Logger) Error(msg string, fields ...any) {
	l.log("error", msg, fields...)
}

// log builds an Entry, writes it as a JSON line to output, and fans out
// to all active subscribers.
func (l *Logger) log(level, msg string, fields ...any) {
	entry := Entry{
		Level: level,
		Msg:   msg,
		Time:  time.Now(),
	}

	// Parse variadic key-value pairs into a map.
	// If there is an odd number of arguments, the last key is ignored.
	if len(fields) > 0 {
		m := make(map[string]any)
		for i := 0; i+1 < len(fields); i += 2 {
			key, ok := fields[i].(string)
			if !ok {
				continue
			}
			m[key] = fields[i+1]
		}
		if len(m) > 0 {
			entry.Fields = m
		}
	}

	// Write JSON line to output
	data, err := json.Marshal(entry)
	if err == nil {
		data = append(data, '\n')
		l.mu.RLock()
		_, _ = l.output.Write(data)
		l.mu.RUnlock()
	}

	// Fan out to all subscribers (non-blocking)
	l.mu.RLock()
	for ch := range l.subscribers {
		select {
		case ch <- entry:
		default:
			// Drop entry if subscriber is slow — channel is full
		}
	}
	l.mu.RUnlock()
}

// Subscribe creates a buffered channel (capacity 100) and registers it
// to receive all future log entries. The caller must eventually call
// Unsubscribe to clean up.
func (l *Logger) Subscribe() chan Entry {
	ch := make(chan Entry, 100)
	l.mu.Lock()
	l.subscribers[ch] = struct{}{}
	l.mu.Unlock()
	return ch
}

// Unsubscribe removes the channel from the subscribers map and closes it.
func (l *Logger) Unsubscribe(ch chan Entry) {
	l.mu.Lock()
	delete(l.subscribers, ch)
	l.mu.Unlock()
	close(ch)
}
