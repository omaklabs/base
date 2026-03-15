package config

import (
	"os"
	"strconv"
)

type MailConfig struct {
	From       string // default sender address
	Host       string // SMTP host
	Port       int    // SMTP port (587 default)
	Username   string // SMTP username
	Password   string // SMTP password
	Encryption string // "tls", "starttls", "none"
}

type Config struct {
	Addr           string
	DatabasePath   string
	CSRFKey        string
	InternalAPIKey string
	Env            string
	Mail           MailConfig
}

func Load() Config {
	mailPort, err := strconv.Atoi(envOr("MAIL_PORT", "587"))
	if err != nil {
		mailPort = 587
	}

	return Config{
		Addr:           envOr("ADDR", ":8080"),
		DatabasePath:   envOr("DATABASE_PATH", "./data/app.db"),
		CSRFKey:        envOr("CSRF_KEY", "change-me-in-production-32bytes!"),
		InternalAPIKey: envOr("INTERNAL_API_KEY", "change-me-in-production"),
		Env:            envOr("ENV", "development"),
		Mail: MailConfig{
			From:       envOr("MAIL_FROM", "noreply@localhost"),
			Host:       envOr("MAIL_HOST", "localhost"),
			Port:       mailPort,
			Username:   envOr("MAIL_USERNAME", ""),
			Password:   envOr("MAIL_PASSWORD", ""),
			Encryption: envOr("MAIL_ENCRYPTION", "tls"),
		},
	}
}

func (c Config) IsDev() bool {
	return c.Env == "development"
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
