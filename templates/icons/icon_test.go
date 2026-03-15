package icons

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func renderToString(t *testing.T, c templ.Component) string {
	t.Helper()
	var buf bytes.Buffer
	if err := c.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render failed: %v", err)
	}
	return buf.String()
}

func TestRenderIcon(t *testing.T) {
	c := renderIcon("md", `<path d="M5 12h14"/>`)
	out := renderToString(t, c)

	if !strings.Contains(out, `class="icon"`) {
		t.Error("expected class=\"icon\" in output")
	}
	if !strings.Contains(out, `data-size="md"`) {
		t.Error("expected data-size=\"md\" in output")
	}
	if !strings.Contains(out, `<path d="M5 12h14"/>`) {
		t.Error("expected SVG path content in output")
	}
	if !strings.Contains(out, `stroke-width="2"`) {
		t.Error("expected stroke-width=\"2\" in output")
	}
	if !strings.Contains(out, `viewBox="0 0 24 24"`) {
		t.Error("expected viewBox in output")
	}
}

func TestRegisterAndCustom(t *testing.T) {
	Register("my-logo", `<rect width="20" height="20" x="2" y="2"/>`)
	c := Custom("my-logo", "lg")
	out := renderToString(t, c)

	if !strings.Contains(out, `<svg`) {
		t.Error("expected <svg> element in custom icon output")
	}
	if !strings.Contains(out, `data-size="lg"`) {
		t.Error("expected data-size=\"lg\" in custom icon output")
	}
	if !strings.Contains(out, `<rect width="20" height="20" x="2" y="2"/>`) {
		t.Error("expected custom SVG content in output")
	}
}

func TestCustomNotFound(t *testing.T) {
	c := Custom("nonexistent-icon", "sm")
	out := renderToString(t, c)

	if !strings.Contains(out, "?nonexistent-icon") {
		t.Error("expected placeholder text with icon name")
	}
	if !strings.Contains(out, `icon not found`) {
		t.Error("expected 'icon not found' title attribute")
	}
	if strings.Contains(out, `<svg`) {
		t.Error("should not render an SVG for a missing icon")
	}
}

func TestGeneratedIconExists(t *testing.T) {
	tests := []struct {
		name string
		fn   func(string) templ.Component
	}{
		{"Search", Search},
		{"User", User},
		{"Plus", Plus},
		{"Check", Check},
		{"X", X},
		{"ChevronDown", ChevronDown},
		{"ArrowRight", ArrowRight},
		{"Home", Home},
		{"Settings", Settings},
		{"Mail", Mail},
		{"Edit", Edit},
		{"Trash2", Trash2},
		{"Eye", Eye},
		{"AlertCircle", AlertCircle},
		{"Menu", Menu},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.fn("md")
			out := renderToString(t, c)

			if !strings.Contains(out, `<svg`) {
				t.Errorf("%s: expected <svg> element", tt.name)
			}
			if !strings.Contains(out, `class="icon"`) {
				t.Errorf("%s: expected class=\"icon\"", tt.name)
			}
			if !strings.Contains(out, `data-size="md"`) {
				t.Errorf("%s: expected data-size=\"md\"", tt.name)
			}
			// Every icon must have at least one SVG child element
			hasChild := strings.Contains(out, "<path") ||
				strings.Contains(out, "<circle") ||
				strings.Contains(out, "<rect") ||
				strings.Contains(out, "<line") ||
				strings.Contains(out, "<polyline") ||
				strings.Contains(out, "<polygon") ||
				strings.Contains(out, "<ellipse")
			if !hasChild {
				t.Errorf("%s: expected SVG child element (path/circle/rect/etc.)", tt.name)
			}
		})
	}
}

func TestSearchIconContent(t *testing.T) {
	out := renderToString(t, Search("sm"))
	// Search should have a circle and a path
	if !strings.Contains(out, `<circle cx="11" cy="11" r="8"/>`) {
		t.Error("Search icon missing circle element")
	}
	if !strings.Contains(out, `<path d="m21 21-4.3-4.3"/>`) {
		t.Error("Search icon missing path element")
	}
}

func TestMenuIconContent(t *testing.T) {
	out := renderToString(t, Menu("md"))
	// Menu should have three lines
	if strings.Count(out, "<line") != 3 {
		t.Errorf("Menu icon expected 3 line elements, got %d", strings.Count(out, "<line"))
	}
}

func TestDifferentSizes(t *testing.T) {
	sizes := []string{"xs", "sm", "md", "lg", "xl"}
	for _, size := range sizes {
		out := renderToString(t, Plus(size))
		expected := `data-size="` + size + `"`
		if !strings.Contains(out, expected) {
			t.Errorf("expected %s in output for size %s", expected, size)
		}
	}
}
