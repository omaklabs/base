package icons

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/a-h/templ"
)

var (
	customIcons = make(map[string]string)
	mu          sync.RWMutex
)

// Register adds a custom icon. Call this during app initialization.
// The svgInner should be the inner SVG content (path elements, etc.)
func Register(name, svgInner string) {
	mu.Lock()
	defer mu.Unlock()
	customIcons[name] = svgInner
}

// Custom renders a registered custom icon by name.
// Returns an empty component if the icon is not found.
func Custom(name, size string) templ.Component {
	mu.RLock()
	inner, ok := customIcons[name]
	mu.RUnlock()
	if !ok {
		return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			// Dev placeholder — visible red box with icon name
			_, err := fmt.Fprintf(w, `<span class="inline-flex items-center justify-center rounded bg-error-500/20 text-error-500 text-xs px-1" data-size="%s" title="icon not found: %s">?%s</span>`, size, name, name)
			return err
		})
	}
	return renderIcon(size, inner)
}

// renderIcon creates a templ.Component that renders an SVG icon.
// This is used by both generated Lucide icons and custom icons.
func renderIcon(size, inner string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := fmt.Fprintf(w, `<svg class="icon" data-size="%s" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor">%s</svg>`, size, inner)
		return err
	})
}
