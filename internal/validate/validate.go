// Package validate provides form validation for HTTP handlers.
// Create a Validator with New(), call check methods (Required, MinLength,
// Email, etc.), then check HasErrors(). Pass Errors() to templates for
// inline error display. Always validate in handlers before querying the DB.
package validate

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Errors maps field names to their validation error messages.
type Errors map[string]string

// HasErrors returns true if there are any validation errors.
func (e Errors) HasErrors() bool {
	return len(e) > 0
}

// Error returns the error message for a field, or empty string.
func (e Errors) Error(field string) string {
	return e[field]
}

// Validator collects validation errors for form fields.
type Validator struct {
	errors Errors
}

// New creates a new Validator.
func New() *Validator {
	return &Validator{errors: make(Errors)}
}

// Errors returns the collected validation errors.
func (v *Validator) Errors() Errors {
	return v.errors
}

// HasErrors returns true if any validation errors were collected.
func (v *Validator) HasErrors() bool {
	return v.errors.HasErrors()
}

// AddError adds a custom error for a field.
func (v *Validator) AddError(field, message string) {
	if _, exists := v.errors[field]; !exists {
		v.errors[field] = message
	}
}

// Required checks that a string value is not empty after trimming.
func (v *Validator) Required(field, value string) {
	if strings.TrimSpace(value) == "" {
		v.AddError(field, fmt.Sprintf("%s is required", field))
	}
}

// MinLength checks minimum string length (in runes, not bytes).
func (v *Validator) MinLength(field, value string, min int) {
	if utf8.RuneCountInString(value) < min {
		v.AddError(field, fmt.Sprintf("%s must be at least %d characters", field, min))
	}
}

// MaxLength checks maximum string length (in runes).
func (v *Validator) MaxLength(field, value string, max int) {
	if utf8.RuneCountInString(value) > max {
		v.AddError(field, fmt.Sprintf("%s must be at most %d characters", field, max))
	}
}

// Email checks that the value is a valid email address.
func (v *Validator) Email(field, value string) {
	_, err := mail.ParseAddress(value)
	if err != nil {
		v.AddError(field, "must be a valid email address")
	}
}

// Matches checks that the value matches a regex pattern.
func (v *Validator) Matches(field, value string, pattern *regexp.Regexp, message string) {
	if !pattern.MatchString(value) {
		v.AddError(field, message)
	}
}

// Equals checks that two values match (useful for password confirmation).
func (v *Validator) Equals(field, value, other string, message string) {
	if value != other {
		v.AddError(field, message)
	}
}

// In checks that the value is in a list of allowed values.
func (v *Validator) In(field, value string, allowed ...string) {
	for _, a := range allowed {
		if value == a {
			return
		}
	}
	v.AddError(field, fmt.Sprintf("%s must be one of: %s", field, strings.Join(allowed, ", ")))
}

// Unique accepts a bool indicating whether a value is already taken (e.g., from a DB check).
// If taken is true, adds an error.
func (v *Validator) Unique(field string, taken bool, message string) {
	if taken {
		v.AddError(field, message)
	}
}

// Check adds an error if the condition is false. Generic escape hatch for custom validations.
func (v *Validator) Check(field string, condition bool, message string) {
	if !condition {
		v.AddError(field, message)
	}
}
