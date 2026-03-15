package config

import "strconv"

// Redacted returns the current config as a map with sensitive values masked.
// Used by the /internal/config endpoint.
func (c Config) Redacted() map[string]any {
	// Build a set of sensitive keys from the schema
	sensitive := make(map[string]bool)
	for _, g := range Schema() {
		for _, f := range g.Fields {
			if f.Sensitive {
				sensitive[f.Key] = true
			}
		}
	}

	// Build the flat config map from the struct fields
	raw := map[string]string{
		"ADDR":            c.Addr,
		"ENV":             c.Env,
		"DATABASE_PATH":   c.DatabasePath,
		"CSRF_KEY":        c.CSRFKey,
		"INTERNAL_API_KEY": c.InternalAPIKey,
		"MAIL_FROM":       c.Mail.From,
		"MAIL_HOST":       c.Mail.Host,
		"MAIL_PORT":       strconv.Itoa(c.Mail.Port),
		"MAIL_USERNAME":   c.Mail.Username,
		"MAIL_PASSWORD":   c.Mail.Password,
		"MAIL_ENCRYPTION": c.Mail.Encryption,
	}

	result := make(map[string]any, len(raw))
	for key, val := range raw {
		if sensitive[key] && val != "" {
			result[key] = "****"
		} else {
			result[key] = val
		}
	}

	return result
}
