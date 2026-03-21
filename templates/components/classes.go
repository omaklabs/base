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
