package shared

import (
	"crypto/rand"
	"fmt"
	"strings"
)

// Cx joins non-empty CSS class strings with a space.
func Cx(classes ...string) string {
	var out []string
	for _, c := range classes {
		if c != "" {
			out = append(out, c)
		}
	}
	return strings.Join(out, " ")
}

// FirstOr returns the first element of s, or fallback if s is empty.
func FirstOr(s []string, fallback string) string {
	if len(s) > 0 && s[0] != "" {
		return s[0]
	}
	return fallback
}

// RandomID generates a short random ID for accessibility linking.
func RandomID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("c-%x", b)
}
