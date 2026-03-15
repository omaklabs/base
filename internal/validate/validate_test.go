package validate

import (
	"regexp"
	"testing"
)

func TestRequired(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty string fails", "", true},
		{"whitespace only fails", "   ", true},
		{"tabs and newlines fail", "\t\n  ", true},
		{"non-empty passes", "hello", false},
		{"value with spaces passes", " hello ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()
			v.Required("name", tt.value)
			if v.HasErrors() != tt.wantErr {
				t.Errorf("Required(%q): HasErrors() = %v, want %v", tt.value, v.HasErrors(), tt.wantErr)
			}
			if tt.wantErr {
				if msg := v.Errors().Error("name"); msg != "name is required" {
					t.Errorf("Expected 'name is required', got %q", msg)
				}
			}
		})
	}
}

func TestMinLength(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		min     int
		wantErr bool
	}{
		{"too short fails", "ab", 3, true},
		{"exact length passes", "abc", 3, false},
		{"longer passes", "abcd", 3, false},
		{"empty fails", "", 1, true},
		{"unicode counted correctly", "\u00e9\u00e0\u00fc", 3, false},        // 3 runes, each multi-byte
		{"unicode too short", "\u00e9\u00e0", 3, true},                       // 2 runes
		{"emoji counted as runes", "\U0001f600\U0001f601\U0001f602", 3, false}, // 3 emoji = 3 runes
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()
			v.MinLength("field", tt.value, tt.min)
			if v.HasErrors() != tt.wantErr {
				t.Errorf("MinLength(%q, %d): HasErrors() = %v, want %v", tt.value, tt.min, v.HasErrors(), tt.wantErr)
			}
		})
	}
}

func TestMaxLength(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		max     int
		wantErr bool
	}{
		{"too long fails", "abcde", 3, true},
		{"exact length passes", "abc", 3, false},
		{"shorter passes", "ab", 3, false},
		{"empty passes", "", 3, false},
		{"unicode counted correctly", "\u00e9\u00e0\u00fc\u00f1", 3, true}, // 4 runes > 3
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()
			v.MaxLength("field", tt.value, tt.max)
			if v.HasErrors() != tt.wantErr {
				t.Errorf("MaxLength(%q, %d): HasErrors() = %v, want %v", tt.value, tt.max, v.HasErrors(), tt.wantErr)
			}
		})
	}
}

func TestEmail(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid email passes", "user@example.com", false},
		{"valid email with plus", "user+tag@example.com", false},
		{"valid email with subdomain", "user@sub.example.com", false},
		{"missing @ fails", "userexample.com", true},
		{"missing domain fails", "user@", true},
		{"missing local part fails", "@example.com", true},
		{"empty string fails", "", true},
		{"just text fails", "notanemail", true},
		{"double @ fails", "user@@example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()
			v.Email("email", tt.value)
			if v.HasErrors() != tt.wantErr {
				t.Errorf("Email(%q): HasErrors() = %v, want %v", tt.value, v.HasErrors(), tt.wantErr)
			}
			if tt.wantErr {
				if msg := v.Errors().Error("email"); msg != "must be a valid email address" {
					t.Errorf("Expected 'must be a valid email address', got %q", msg)
				}
			}
		})
	}
}

func TestMatches(t *testing.T) {
	alphanumeric := regexp.MustCompile(`^[a-zA-Z0-9]+$`)

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"matching passes", "abc123", false},
		{"non-matching fails", "abc-123!", true},
		{"empty fails", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()
			v.Matches("username", tt.value, alphanumeric, "must be alphanumeric")
			if v.HasErrors() != tt.wantErr {
				t.Errorf("Matches(%q): HasErrors() = %v, want %v", tt.value, v.HasErrors(), tt.wantErr)
			}
			if tt.wantErr {
				if msg := v.Errors().Error("username"); msg != "must be alphanumeric" {
					t.Errorf("Expected 'must be alphanumeric', got %q", msg)
				}
			}
		})
	}
}

func TestEquals(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		other   string
		wantErr bool
	}{
		{"matching values pass", "secret123", "secret123", false},
		{"different values fail", "secret123", "secret456", true},
		{"empty values match", "", "", false},
		{"one empty fails", "secret", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()
			v.Equals("password_confirm", tt.value, tt.other, "passwords do not match")
			if v.HasErrors() != tt.wantErr {
				t.Errorf("Equals(%q, %q): HasErrors() = %v, want %v", tt.value, tt.other, v.HasErrors(), tt.wantErr)
			}
			if tt.wantErr {
				if msg := v.Errors().Error("password_confirm"); msg != "passwords do not match" {
					t.Errorf("Expected 'passwords do not match', got %q", msg)
				}
			}
		})
	}
}

func TestIn(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		allowed []string
		wantErr bool
	}{
		{"value in list passes", "admin", []string{"admin", "user", "moderator"}, false},
		{"value not in list fails", "superadmin", []string{"admin", "user", "moderator"}, true},
		{"empty value not in list fails", "", []string{"admin", "user"}, true},
		{"single allowed value passes", "yes", []string{"yes"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()
			v.In("role", tt.value, tt.allowed...)
			if v.HasErrors() != tt.wantErr {
				t.Errorf("In(%q, %v): HasErrors() = %v, want %v", tt.value, tt.allowed, v.HasErrors(), tt.wantErr)
			}
		})
	}
}

func TestUnique(t *testing.T) {
	t.Run("taken adds error", func(t *testing.T) {
		v := New()
		v.Unique("email", true, "email is already taken")
		if !v.HasErrors() {
			t.Error("Expected error when taken=true")
		}
		if msg := v.Errors().Error("email"); msg != "email is already taken" {
			t.Errorf("Expected 'email is already taken', got %q", msg)
		}
	})

	t.Run("not taken no error", func(t *testing.T) {
		v := New()
		v.Unique("email", false, "email is already taken")
		if v.HasErrors() {
			t.Error("Expected no error when taken=false")
		}
	})
}

func TestCheck(t *testing.T) {
	t.Run("false condition adds error", func(t *testing.T) {
		v := New()
		v.Check("age", false, "must be at least 18")
		if !v.HasErrors() {
			t.Error("Expected error when condition is false")
		}
		if msg := v.Errors().Error("age"); msg != "must be at least 18" {
			t.Errorf("Expected 'must be at least 18', got %q", msg)
		}
	})

	t.Run("true condition no error", func(t *testing.T) {
		v := New()
		v.Check("age", true, "must be at least 18")
		if v.HasErrors() {
			t.Error("Expected no error when condition is true")
		}
	})
}

func TestFirstErrorOnly(t *testing.T) {
	v := New()
	v.Required("title", "")               // first error: "title is required"
	v.MinLength("title", "", 3)           // second error: should be ignored
	v.MaxLength("title", "", 200)         // passes, but irrelevant

	if msg := v.Errors().Error("title"); msg != "title is required" {
		t.Errorf("Expected first error 'title is required', got %q", msg)
	}

	// Verify only one error exists for the field
	errorCount := 0
	for range v.Errors() {
		errorCount++
	}
	if errorCount != 1 {
		t.Errorf("Expected 1 error entry, got %d", errorCount)
	}
}

func TestHasErrors(t *testing.T) {
	t.Run("empty validator returns false", func(t *testing.T) {
		v := New()
		if v.HasErrors() {
			t.Error("New validator should not have errors")
		}
	})

	t.Run("after error returns true", func(t *testing.T) {
		v := New()
		v.Required("name", "")
		if !v.HasErrors() {
			t.Error("Validator should have errors after failed validation")
		}
	})

	t.Run("after passing validation returns false", func(t *testing.T) {
		v := New()
		v.Required("name", "John")
		if v.HasErrors() {
			t.Error("Validator should not have errors after passing validation")
		}
	})
}

func TestAddError(t *testing.T) {
	v := New()
	v.AddError("custom_field", "something went wrong")

	if !v.HasErrors() {
		t.Error("Expected errors after AddError")
	}

	msg := v.Errors().Error("custom_field")
	if msg != "something went wrong" {
		t.Errorf("Expected 'something went wrong', got %q", msg)
	}

	// Non-existent field returns empty string
	if got := v.Errors().Error("nonexistent"); got != "" {
		t.Errorf("Expected empty string for nonexistent field, got %q", got)
	}
}
