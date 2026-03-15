package config

// FieldType represents the type of a config field.
type FieldType string

const (
	TypeString  FieldType = "string"
	TypeNumber  FieldType = "number"
	TypeSecret  FieldType = "secret"
	TypeSelect  FieldType = "select"
	TypeBoolean FieldType = "boolean"
)

// SchemaField describes a single configuration field.
type SchemaField struct {
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Type        FieldType `json:"type"`
	Default     string    `json:"default"`
	Required    bool      `json:"required"`
	Sensitive   bool      `json:"sensitive"`
	Options     []string  `json:"options,omitempty"` // for select type
}

// SchemaGroup is a logical group of related config fields.
type SchemaGroup struct {
	Key         string        `json:"key"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Fields      []SchemaField `json:"fields"`
}

var schema []SchemaGroup

func init() {
	schema = []SchemaGroup{
		{
			Key:         "server",
			Name:        "Server",
			Description: "Core server settings",
			Fields: []SchemaField{
				{
					Key:         "ADDR",
					Name:        "Listen Address",
					Description: "Server listen address",
					Type:        TypeString,
					Default:     ":8080",
					Required:    false,
					Sensitive:   false,
				},
				{
					Key:         "ENV",
					Name:        "Environment",
					Description: "Running environment",
					Type:        TypeSelect,
					Default:     "development",
					Required:    true,
					Sensitive:   false,
					Options:     []string{"development", "production"},
				},
			},
		},
		{
			Key:         "database",
			Name:        "Database",
			Description: "Database connection settings",
			Fields: []SchemaField{
				{
					Key:         "DATABASE_PATH",
					Name:        "Database Path",
					Description: "Path to SQLite database file",
					Type:        TypeString,
					Default:     "./data/app.db",
					Required:    true,
					Sensitive:   false,
				},
			},
		},
		{
			Key:         "security",
			Name:        "Security",
			Description: "Security and authentication settings",
			Fields: []SchemaField{
				{
					Key:         "CSRF_KEY",
					Name:        "CSRF Key",
					Description: "32-byte key for CSRF protection",
					Type:        TypeSecret,
					Default:     "",
					Required:    true,
					Sensitive:   true,
				},
				{
					Key:         "INTERNAL_API_KEY",
					Name:        "Internal API Key",
					Description: "Shared secret for /internal/* API endpoints",
					Type:        TypeSecret,
					Default:     "",
					Required:    true,
					Sensitive:   true,
				},
			},
		},
		{
			Key:         "email",
			Name:        "Email",
			Description: "Email / SMTP settings",
			Fields: []SchemaField{
				{
					Key:         "MAIL_FROM",
					Name:        "From Address",
					Description: "Default sender email address",
					Type:        TypeString,
					Default:     "noreply@localhost",
					Required:    true,
					Sensitive:   false,
				},
				{
					Key:         "MAIL_HOST",
					Name:        "SMTP Host",
					Description: "SMTP server hostname",
					Type:        TypeString,
					Default:     "localhost",
					Required:    true,
					Sensitive:   false,
				},
				{
					Key:         "MAIL_PORT",
					Name:        "SMTP Port",
					Description: "SMTP server port",
					Type:        TypeNumber,
					Default:     "587",
					Required:    false,
					Sensitive:   false,
				},
				{
					Key:         "MAIL_USERNAME",
					Name:        "SMTP Username",
					Description: "SMTP authentication username",
					Type:        TypeString,
					Default:     "",
					Required:    false,
					Sensitive:   false,
				},
				{
					Key:         "MAIL_PASSWORD",
					Name:        "SMTP Password",
					Description: "SMTP authentication password",
					Type:        TypeSecret,
					Default:     "",
					Required:    false,
					Sensitive:   true,
				},
				{
					Key:         "MAIL_ENCRYPTION",
					Name:        "Encryption",
					Description: "SMTP connection encryption",
					Type:        TypeSelect,
					Default:     "tls",
					Required:    false,
					Sensitive:   false,
					Options:     []string{"tls", "starttls", "none"},
				},
			},
		},
	}
}

// Schema returns the full config schema.
// The agent adds groups here when building features that need configuration.
func Schema() []SchemaGroup {
	return schema
}

// RegisterGroup adds a config group to the schema.
// Called during init by packages that need configuration.
func RegisterGroup(group SchemaGroup) {
	schema = append(schema, group)
}
