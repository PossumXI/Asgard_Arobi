// Package utils provides utility functions for the application.
package utils

import (
	"fmt"
	"net/http"
)

// APIError represents an API error with status code and message.
type APIError struct {
	Code    string
	Message string
	Status  int
	Err     error
}

func (e *APIError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the underlying error.
func (e *APIError) Unwrap() error {
	return e.Err
}

// NewAPIError creates a new API error.
func NewAPIError(code, message string, status int) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
		Status:  status,
	}
}

// WrapAPIError wraps an error with API error information.
func WrapAPIError(err error, code, message string, status int) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
		Status:  status,
		Err:     err,
	}
}

// Predefined API errors
var (
	ErrNotFound            = NewAPIError("NOT_FOUND", "Resource not found", http.StatusNotFound)
	ErrUnauthorized        = NewAPIError("UNAUTHORIZED", "Unauthorized", http.StatusUnauthorized)
	ErrForbidden           = NewAPIError("FORBIDDEN", "Forbidden", http.StatusForbidden)
	ErrBadRequest          = NewAPIError("BAD_REQUEST", "Bad request", http.StatusBadRequest)
	ErrInternalServer      = NewAPIError("INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError)
	ErrConflict            = NewAPIError("CONFLICT", "Resource conflict", http.StatusConflict)
	ErrUnprocessableEntity = NewAPIError("UNPROCESSABLE", "Unprocessable entity", http.StatusUnprocessableEntity)
)
