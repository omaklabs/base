package config

import "testing"

func TestSchemaReturnsDefaultGroups(t *testing.T) {
	groups := Schema()

	expectedKeys := []string{"server", "database", "security", "email"}
	if len(groups) < len(expectedKeys) {
		t.Fatalf("expected at least %d groups, got %d", len(expectedKeys), len(groups))
	}

	found := make(map[string]bool)
	for _, g := range groups {
		found[g.Key] = true
	}

	for _, key := range expectedKeys {
		if !found[key] {
			t.Errorf("expected group %q not found in schema", key)
		}
	}
}

func TestSchemaFieldCount(t *testing.T) {
	groups := Schema()

	expected := map[string]int{
		"server":   2,
		"database": 1,
		"security": 2,
		"email":    6,
	}

	groupMap := make(map[string]SchemaGroup)
	for _, g := range groups {
		groupMap[g.Key] = g
	}

	for key, wantCount := range expected {
		g, ok := groupMap[key]
		if !ok {
			t.Errorf("group %q not found", key)
			continue
		}
		if len(g.Fields) != wantCount {
			t.Errorf("group %q: expected %d fields, got %d", key, wantCount, len(g.Fields))
		}
	}
}

func TestRegisterGroup(t *testing.T) {
	// Save original schema length to verify addition
	originalLen := len(Schema())

	RegisterGroup(SchemaGroup{
		Key:         "custom",
		Name:        "Custom",
		Description: "Custom settings",
		Fields: []SchemaField{
			{
				Key:  "CUSTOM_KEY",
				Name: "Custom Key",
				Type: TypeString,
			},
		},
	})

	groups := Schema()
	if len(groups) != originalLen+1 {
		t.Fatalf("expected %d groups after register, got %d", originalLen+1, len(groups))
	}

	last := groups[len(groups)-1]
	if last.Key != "custom" {
		t.Errorf("last group key = %q, want %q", last.Key, "custom")
	}

	// Clean up: remove the custom group we added
	schema = schema[:originalLen]
}

func TestSchemaFieldTypes(t *testing.T) {
	groups := Schema()

	// Build a flat map of key -> field
	fields := make(map[string]SchemaField)
	for _, g := range groups {
		for _, f := range g.Fields {
			fields[f.Key] = f
		}
	}

	tests := []struct {
		key      string
		wantType FieldType
	}{
		{"ADDR", TypeString},
		{"ENV", TypeSelect},
		{"DATABASE_PATH", TypeString},
		{"CSRF_KEY", TypeSecret},
		{"INTERNAL_API_KEY", TypeSecret},
		{"MAIL_FROM", TypeString},
		{"MAIL_PORT", TypeNumber},
		{"MAIL_PASSWORD", TypeSecret},
		{"MAIL_ENCRYPTION", TypeSelect},
	}

	for _, tt := range tests {
		f, ok := fields[tt.key]
		if !ok {
			t.Errorf("field %q not found in schema", tt.key)
			continue
		}
		if f.Type != tt.wantType {
			t.Errorf("field %q: type = %q, want %q", tt.key, f.Type, tt.wantType)
		}
	}
}
