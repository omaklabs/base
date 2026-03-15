package config

import "testing"

func TestRedactedMasksSensitiveFields(t *testing.T) {
	cfg := Config{
		Addr:           ":8080",
		Env:            "development",
		DatabasePath:   "./data/app.db",
		CSRFKey:        "my-secret-csrf-key",
		InternalAPIKey: "my-secret-api-key",
		Mail: MailConfig{
			From:       "noreply@localhost",
			Host:       "localhost",
			Port:       587,
			Username:   "user",
			Password:   "secret-password",
			Encryption: "tls",
		},
	}

	redacted := cfg.Redacted()

	sensitiveKeys := []string{"CSRF_KEY", "INTERNAL_API_KEY", "MAIL_PASSWORD"}
	for _, key := range sensitiveKeys {
		val, ok := redacted[key]
		if !ok {
			t.Errorf("missing key %q in redacted output", key)
			continue
		}
		if val != "****" {
			t.Errorf("redacted[%q] = %q, want %q", key, val, "****")
		}
	}
}

func TestRedactedShowsNonSensitiveFields(t *testing.T) {
	cfg := Config{
		Addr:           ":9090",
		Env:            "production",
		DatabasePath:   "/data/prod.db",
		CSRFKey:        "secret",
		InternalAPIKey: "secret",
		Mail: MailConfig{
			From:       "sender@example.com",
			Host:       "smtp.example.com",
			Port:       465,
			Username:   "user@example.com",
			Password:   "secret",
			Encryption: "starttls",
		},
	}

	redacted := cfg.Redacted()

	expected := map[string]string{
		"ADDR":            ":9090",
		"ENV":             "production",
		"DATABASE_PATH":   "/data/prod.db",
		"MAIL_FROM":       "sender@example.com",
		"MAIL_HOST":       "smtp.example.com",
		"MAIL_PORT":       "465",
		"MAIL_USERNAME":   "user@example.com",
		"MAIL_ENCRYPTION": "starttls",
	}

	for key, want := range expected {
		got, ok := redacted[key]
		if !ok {
			t.Errorf("missing key %q in redacted output", key)
			continue
		}
		if got != want {
			t.Errorf("redacted[%q] = %q, want %q", key, got, want)
		}
	}
}

func TestRedactedEmptySensitiveFieldsShowEmpty(t *testing.T) {
	cfg := Config{
		Addr:           ":8080",
		Env:            "development",
		DatabasePath:   "./data/app.db",
		CSRFKey:        "",
		InternalAPIKey: "",
		Mail: MailConfig{
			From:       "noreply@localhost",
			Host:       "localhost",
			Port:       587,
			Username:   "",
			Password:   "",
			Encryption: "tls",
		},
	}

	redacted := cfg.Redacted()

	sensitiveKeys := []string{"CSRF_KEY", "INTERNAL_API_KEY", "MAIL_PASSWORD"}
	for _, key := range sensitiveKeys {
		val, ok := redacted[key]
		if !ok {
			t.Errorf("missing key %q in redacted output", key)
			continue
		}
		if val != "" {
			t.Errorf("redacted[%q] = %q, want empty string for empty sensitive field", key, val)
		}
	}
}
