package components

import "strings"

// cx joins non-empty CSS class strings with a space.
// Usage: cx("base classes", variant, modifier, props.Class)
func cx(classes ...string) string {
	var out []string
	for _, c := range classes {
		if c != "" {
			out = append(out, c)
		}
	}
	return strings.Join(out, " ")
}

// firstOr returns the first element of s, or fallback if s is empty.
// Used by simple component APIs: Button("primary", "sm") where size is optional.
func firstOr(s []string, fallback string) string {
	if len(s) > 0 && s[0] != "" {
		return s[0]
	}
	return fallback
}

// ── Button ──

// ButtonVariant controls button visual style.
type ButtonVariant string

const (
	ButtonVariantPrimary     ButtonVariant = "primary"
	ButtonVariantSecondary   ButtonVariant = "secondary"
	ButtonVariantDestructive ButtonVariant = "destructive"
	ButtonVariantGhost       ButtonVariant = "ghost"
	ButtonVariantOutline     ButtonVariant = "outline"
)

// ButtonSize controls button dimensions.
type ButtonSize string

const (
	ButtonSizeSm ButtonSize = "sm"
	ButtonSizeMd ButtonSize = "md"
	ButtonSizeLg ButtonSize = "lg"
)

// ── Badge ──

// BadgeVariant controls badge color scheme.
type BadgeVariant string

const (
	BadgeVariantDefault     BadgeVariant = "default"
	BadgeVariantSuccess     BadgeVariant = "success"
	BadgeVariantDestructive BadgeVariant = "destructive"
	BadgeVariantWarning     BadgeVariant = "warning"
)

// ── Alert ──

// AlertVariant controls alert color scheme.
type AlertVariant string

const (
	AlertVariantDefault     AlertVariant = "default"
	AlertVariantDestructive AlertVariant = "destructive"
	AlertVariantWarning     AlertVariant = "warning"
	AlertVariantSuccess     AlertVariant = "success"
)

// ── Progress ──

// ProgressVariant controls progress bar color.
type ProgressVariant string

const (
	ProgressVariantDefault     ProgressVariant = "default"
	ProgressVariantSuccess     ProgressVariant = "success"
	ProgressVariantDestructive ProgressVariant = "destructive"
)

// ── Separator ──

// SeparatorOrientation controls separator direction.
type SeparatorOrientation string

const (
	SeparatorHorizontal SeparatorOrientation = "horizontal"
	SeparatorVertical   SeparatorOrientation = "vertical"
)

// ── Form Message ──

// FormMessageVariant controls form message color.
type FormMessageVariant string

const (
	FormMessageError FormMessageVariant = "error"
	FormMessageInfo  FormMessageVariant = "info"
)
