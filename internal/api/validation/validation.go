// Package validation provides request validation utilities.
package validation

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	uuidRegex  = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

// ValidationError represents a validation error.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error on field '%s': %s", e.Field, e.Message)
}

// ValidateEmail validates an email address.
func ValidateEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return &ValidationError{Field: "email", Message: "email is required"}
	}
	if len(email) > 255 {
		return &ValidationError{Field: "email", Message: "email must be less than 255 characters"}
	}
	if !emailRegex.MatchString(email) {
		return &ValidationError{Field: "email", Message: "invalid email format"}
	}
	return nil
}

// ValidatePassword validates a password.
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return &ValidationError{Field: "password", Message: "password must be at least 8 characters"}
	}
	if len(password) > 128 {
		return &ValidationError{Field: "password", Message: "password must be less than 128 characters"}
	}
	hasUpper := false
	hasLower := false
	hasNumber := false
	for _, c := range password {
		if c >= 'A' && c <= 'Z' {
			hasUpper = true
		}
		if c >= 'a' && c <= 'z' {
			hasLower = true
		}
		if c >= '0' && c <= '9' {
			hasNumber = true
		}
	}
	if !hasUpper || !hasLower || !hasNumber {
		return &ValidationError{Field: "password", Message: "password must contain uppercase, lowercase, and numeric characters"}
	}
	return nil
}

// ValidateUUID validates a UUID string.
func ValidateUUID(id string) error {
	if id == "" {
		return &ValidationError{Field: "id", Message: "id is required"}
	}
	if !uuidRegex.MatchString(id) {
		return &ValidationError{Field: "id", Message: "invalid UUID format"}
	}
	return nil
}

// ValidateNonEmpty validates that a string is not empty.
func ValidateNonEmpty(value, fieldName string) error {
	if strings.TrimSpace(value) == "" {
		return &ValidationError{Field: fieldName, Message: fmt.Sprintf("%s is required", fieldName)}
	}
	return nil
}

// ValidateLength validates string length.
func ValidateLength(value, fieldName string, min, max int) error {
	length := len(strings.TrimSpace(value))
	if length < min {
		return &ValidationError{Field: fieldName, Message: fmt.Sprintf("%s must be at least %d characters", fieldName, min)}
	}
	if length > max {
		return &ValidationError{Field: fieldName, Message: fmt.Sprintf("%s must be less than %d characters", fieldName, max)}
	}
	return nil
}
