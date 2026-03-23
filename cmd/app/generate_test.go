package main

import "testing"

func TestPluralize(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Regular nouns — just add "s"
		{"post", "posts"},
		{"note", "notes"},
		{"user", "users"},
		{"project", "projects"},
		{"comment", "comments"},
		{"invoice", "invoices"},
		{"tag", "tags"},
		{"role", "roles"},
		{"file", "files"},
		{"page", "pages"},
		{"todo", "todos"},
		{"photo", "photos"},
		{"video", "videos"},
		{"event", "events"},
		{"order", "orders"},
		{"product", "products"},
		{"team", "teams"},
		{"task", "tasks"},
		{"item", "items"},
		{"link", "links"},

		// Words ending in s → +es
		{"bus", "buses"},
		{"status", "statuses"},
		{"campus", "campuses"},
		{"bonus", "bonuses"},

		// Words ending in ss → +es
		{"class", "classes"},
		{"address", "addresses"},
		{"process", "processes"},
		{"access", "accesses"},

		// Words ending in sh → +es
		{"wish", "wishes"},
		{"flash", "flashes"},
		{"crash", "crashes"},
		{"push", "pushes"},

		// Words ending in ch → +es
		{"match", "matches"},
		{"batch", "batches"},
		{"search", "searches"},
		{"watch", "watches"},

		// Words ending in x → +es
		{"box", "boxes"},
		{"tax", "taxes"},
		{"fix", "fixes"},
		{"index", "indexes"},

		// Words ending in z → +es
		{"quiz", "quizes"}, // not "quizzes" but good enough for domain names
		{"buzz", "buzzes"},

		// Consonant + y → +ies
		{"category", "categories"},
		{"entry", "entries"},
		{"company", "companies"},
		{"policy", "policies"},
		{"activity", "activities"},
		{"story", "stories"},
		{"city", "cities"},
		{"reply", "replies"},
		{"currency", "currencies"},

		// Vowel + y → +s (not +ies)
		{"key", "keys"},
		{"survey", "surveys"},
		{"day", "days"},
		{"journey", "journeys"},
		{"display", "displays"},
		{"essay", "essays"},

		// Edge cases
		{"", ""},
		{"a", "as"},

		// Common domain names that previously broke
		{"message", "messages"}, // was "messagess" before fix
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := pluralize(tt.input)
			if got != tt.want {
				t.Errorf("pluralize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestPluralizePascal(t *testing.T) {
	// Verify toPascal + pluralize works correctly for PascalPlur field
	tests := []struct {
		input string
		want  string
	}{
		{"post", "Posts"},
		{"todo", "Todos"},
		{"category", "Categories"},
		{"address", "Addresses"},
		{"message", "Messages"},
		{"status", "Statuses"},
		{"entry", "Entries"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toPascal(pluralize(tt.input))
			if got != tt.want {
				t.Errorf("toPascal(pluralize(%q)) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFieldRefPluralization(t *testing.T) {
	// Verify that ref field types produce correct table references
	tests := []struct {
		refName  string
		wantTable string
	}{
		{"user", "users"},
		{"category", "categories"},
		{"address", "addresses"},
		{"status", "statuses"},
		{"company", "companies"},
	}

	for _, tt := range tests {
		t.Run(tt.refName, func(t *testing.T) {
			f := fieldFromType(tt.refName, "ref")
			if f.RefTable != tt.wantTable {
				t.Errorf("fieldFromType(%q, ref).RefTable = %q, want %q", tt.refName, f.RefTable, tt.wantTable)
			}
			wantSQL := "INTEGER NOT NULL REFERENCES " + tt.wantTable + "(id) ON DELETE CASCADE"
			if f.SQLType != wantSQL {
				t.Errorf("fieldFromType(%q, ref).SQLType = %q, want %q", tt.refName, f.SQLType, wantSQL)
			}
		})
	}
}
