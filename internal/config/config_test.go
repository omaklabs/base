package config

import (
	"os"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	// Clear any env vars that might interfere
	for _, key := range []string{"ADDR", "DATABASE_PATH", "CSRF_KEY", "INTERNAL_API_KEY", "ENV",
		"MAIL_FROM", "MAIL_HOST", "MAIL_PORT", "MAIL_USERNAME", "MAIL_PASSWORD", "MAIL_ENCRYPTION"} {
		t.Setenv(key, "")
	}

	cfg := Load()

	if cfg.Addr != ":8080" {
		t.Errorf("expected Addr :8080, got %s", cfg.Addr)
	}
	if cfg.DatabasePath != "./data/app.db" {
		t.Errorf("expected DatabasePath ./data/app.db, got %s", cfg.DatabasePath)
	}
	if cfg.Env != "development" {
		t.Errorf("expected Env development, got %s", cfg.Env)
	}
	if cfg.Mail.From != "noreply@localhost" {
		t.Errorf("expected Mail.From noreply@localhost, got %s", cfg.Mail.From)
	}
	if cfg.Mail.Host != "localhost" {
		t.Errorf("expected Mail.Host localhost, got %s", cfg.Mail.Host)
	}
	if cfg.Mail.Port != 587 {
		t.Errorf("expected Mail.Port 587, got %d", cfg.Mail.Port)
	}
	if cfg.Mail.Username != "" {
		t.Errorf("expected Mail.Username empty, got %s", cfg.Mail.Username)
	}
	if cfg.Mail.Password != "" {
		t.Errorf("expected Mail.Password empty, got %s", cfg.Mail.Password)
	}
	if cfg.Mail.Encryption != "tls" {
		t.Errorf("expected Mail.Encryption tls, got %s", cfg.Mail.Encryption)
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("ADDR", ":9090")
	t.Setenv("DATABASE_PATH", "/tmp/test.db")
	t.Setenv("CSRF_KEY", "test-csrf-key-32-bytes-long!!!!!")
	t.Setenv("INTERNAL_API_KEY", "test-internal-key")
	t.Setenv("ENV", "production")
	t.Setenv("MAIL_FROM", "sender@example.com")
	t.Setenv("MAIL_HOST", "smtp.example.com")
	t.Setenv("MAIL_PORT", "465")
	t.Setenv("MAIL_USERNAME", "user@example.com")
	t.Setenv("MAIL_PASSWORD", "secret")
	t.Setenv("MAIL_ENCRYPTION", "starttls")

	cfg := Load()

	if cfg.Addr != ":9090" {
		t.Errorf("expected Addr :9090, got %s", cfg.Addr)
	}
	if cfg.DatabasePath != "/tmp/test.db" {
		t.Errorf("expected DatabasePath /tmp/test.db, got %s", cfg.DatabasePath)
	}
	if cfg.CSRFKey != "test-csrf-key-32-bytes-long!!!!!" {
		t.Errorf("expected CSRFKey from env, got %s", cfg.CSRFKey)
	}
	if cfg.InternalAPIKey != "test-internal-key" {
		t.Errorf("expected InternalAPIKey from env, got %s", cfg.InternalAPIKey)
	}
	if cfg.Env != "production" {
		t.Errorf("expected Env production, got %s", cfg.Env)
	}
	if cfg.Mail.From != "sender@example.com" {
		t.Errorf("expected Mail.From sender@example.com, got %s", cfg.Mail.From)
	}
	if cfg.Mail.Host != "smtp.example.com" {
		t.Errorf("expected Mail.Host smtp.example.com, got %s", cfg.Mail.Host)
	}
	if cfg.Mail.Port != 465 {
		t.Errorf("expected Mail.Port 465, got %d", cfg.Mail.Port)
	}
	if cfg.Mail.Username != "user@example.com" {
		t.Errorf("expected Mail.Username user@example.com, got %s", cfg.Mail.Username)
	}
	if cfg.Mail.Password != "secret" {
		t.Errorf("expected Mail.Password secret, got %s", cfg.Mail.Password)
	}
	if cfg.Mail.Encryption != "starttls" {
		t.Errorf("expected Mail.Encryption starttls, got %s", cfg.Mail.Encryption)
	}
}

func TestLoadMailPortInvalid(t *testing.T) {
	t.Setenv("MAIL_PORT", "not-a-number")

	cfg := Load()

	if cfg.Mail.Port != 587 {
		t.Errorf("expected Mail.Port fallback to 587, got %d", cfg.Mail.Port)
	}
}

func TestIsDev(t *testing.T) {
	tests := []struct {
		env  string
		want bool
	}{
		{"development", true},
		{"production", false},
		{"staging", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			cfg := Config{Env: tt.env}
			if got := cfg.IsDev(); got != tt.want {
				t.Errorf("IsDev() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEnvOr(t *testing.T) {
	key := "TEST_ENV_OR_KEY_XYZ"
	os.Unsetenv(key)

	if got := envOr(key, "fallback"); got != "fallback" {
		t.Errorf("expected fallback, got %s", got)
	}

	t.Setenv(key, "actual")
	if got := envOr(key, "fallback"); got != "actual" {
		t.Errorf("expected actual, got %s", got)
	}
}
