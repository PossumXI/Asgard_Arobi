// Package utils provides utility functions for the application.
package utils

import (
	"errors"
	"net/http"
	"testing"
)

func TestNewAPIError(t *testing.T) {
	tests := []struct {
		name        string
		code        string
		message     string
		status      int
		wantCode    string
		wantMessage string
		wantStatus  int
	}{
		{
			name:        "not found error",
			code:        "NOT_FOUND",
			message:     "Resource not found",
			status:      http.StatusNotFound,
			wantCode:    "NOT_FOUND",
			wantMessage: "Resource not found",
			wantStatus:  http.StatusNotFound,
		},
		{
			name:        "bad request error",
			code:        "BAD_REQUEST",
			message:     "Invalid input",
			status:      http.StatusBadRequest,
			wantCode:    "BAD_REQUEST",
			wantMessage: "Invalid input",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "internal error",
			code:        "INTERNAL_ERROR",
			message:     "Something went wrong",
			status:      http.StatusInternalServerError,
			wantCode:    "INTERNAL_ERROR",
			wantMessage: "Something went wrong",
			wantStatus:  http.StatusInternalServerError,
		},
		{
			name:        "custom error",
			code:        "CUSTOM_CODE",
			message:     "Custom message",
			status:      http.StatusTeapot,
			wantCode:    "CUSTOM_CODE",
			wantMessage: "Custom message",
			wantStatus:  http.StatusTeapot,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewAPIError(tt.code, tt.message, tt.status)

			if err == nil {
				t.Fatal("NewAPIError() returned nil")
			}
			if err.Code != tt.wantCode {
				t.Errorf("Code = %v, want %v", err.Code, tt.wantCode)
			}
			if err.Message != tt.wantMessage {
				t.Errorf("Message = %v, want %v", err.Message, tt.wantMessage)
			}
			if err.Status != tt.wantStatus {
				t.Errorf("Status = %v, want %v", err.Status, tt.wantStatus)
			}
			if err.Err != nil {
				t.Error("Err should be nil for NewAPIError")
			}
		})
	}
}

func TestWrapAPIError(t *testing.T) {
	originalErr := errors.New("original error")

	tests := []struct {
		name        string
		err         error
		code        string
		message     string
		status      int
		wantWrapped bool
	}{
		{
			name:        "wrap standard error",
			err:         originalErr,
			code:        "WRAPPED",
			message:     "Wrapped error",
			status:      http.StatusBadRequest,
			wantWrapped: true,
		},
		{
			name:        "wrap nil error",
			err:         nil,
			code:        "NO_WRAP",
			message:     "No original error",
			status:      http.StatusOK,
			wantWrapped: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := WrapAPIError(tt.err, tt.code, tt.message, tt.status)

			if err == nil {
				t.Fatal("WrapAPIError() returned nil")
			}
			if err.Code != tt.code {
				t.Errorf("Code = %v, want %v", err.Code, tt.code)
			}
			if err.Message != tt.message {
				t.Errorf("Message = %v, want %v", err.Message, tt.message)
			}
			if err.Status != tt.status {
				t.Errorf("Status = %v, want %v", err.Status, tt.status)
			}
			if tt.wantWrapped && err.Err == nil {
				t.Error("Err should not be nil when wrapping an error")
			}
			if !tt.wantWrapped && err.Err != nil {
				t.Error("Err should be nil when wrapping nil")
			}
		})
	}
}

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name       string
		apiErr     *APIError
		wantSubstr string
	}{
		{
			name: "error without wrapped error",
			apiErr: &APIError{
				Code:    "TEST",
				Message: "Test message",
				Status:  http.StatusBadRequest,
				Err:     nil,
			},
			wantSubstr: "Test message",
		},
		{
			name: "error with wrapped error",
			apiErr: &APIError{
				Code:    "WRAPPED",
				Message: "Wrapper message",
				Status:  http.StatusInternalServerError,
				Err:     errors.New("underlying cause"),
			},
			wantSubstr: "underlying cause",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.apiErr.Error()

			if errStr == "" {
				t.Error("Error() returned empty string")
			}
			if !containsSubstring(errStr, tt.wantSubstr) {
				t.Errorf("Error() = %q, should contain %q", errStr, tt.wantSubstr)
			}
		})
	}
}

func TestAPIError_ErrorFormat(t *testing.T) {
	// Test error message format with wrapped error
	originalErr := errors.New("database connection failed")
	apiErr := WrapAPIError(originalErr, "DB_ERROR", "Database error", http.StatusInternalServerError)

	errStr := apiErr.Error()

	// Should contain both message and original error
	if !containsSubstring(errStr, "Database error") {
		t.Errorf("Error() = %q, should contain message", errStr)
	}
	if !containsSubstring(errStr, "database connection failed") {
		t.Errorf("Error() = %q, should contain original error", errStr)
	}
}

func TestAPIError_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")

	tests := []struct {
		name       string
		apiErr     *APIError
		wantUnwrap error
	}{
		{
			name: "unwrap returns wrapped error",
			apiErr: &APIError{
				Code:    "TEST",
				Message: "Test",
				Status:  http.StatusBadRequest,
				Err:     originalErr,
			},
			wantUnwrap: originalErr,
		},
		{
			name: "unwrap returns nil when no wrapped error",
			apiErr: &APIError{
				Code:    "TEST",
				Message: "Test",
				Status:  http.StatusBadRequest,
				Err:     nil,
			},
			wantUnwrap: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unwrapped := tt.apiErr.Unwrap()

			if unwrapped != tt.wantUnwrap {
				t.Errorf("Unwrap() = %v, want %v", unwrapped, tt.wantUnwrap)
			}
		})
	}
}

func TestAPIError_ErrorsIs(t *testing.T) {
	originalErr := errors.New("original error")
	apiErr := WrapAPIError(originalErr, "WRAPPED", "Wrapped", http.StatusBadRequest)

	// errors.Is should work with wrapped errors
	if !errors.Is(apiErr, originalErr) {
		t.Error("errors.Is should return true for wrapped error")
	}

	otherErr := errors.New("other error")
	if errors.Is(apiErr, otherErr) {
		t.Error("errors.Is should return false for different error")
	}
}

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        *APIError
		wantCode   string
		wantStatus int
	}{
		{
			name:       "ErrNotFound",
			err:        ErrNotFound,
			wantCode:   "NOT_FOUND",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "ErrUnauthorized",
			err:        ErrUnauthorized,
			wantCode:   "UNAUTHORIZED",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "ErrForbidden",
			err:        ErrForbidden,
			wantCode:   "FORBIDDEN",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "ErrBadRequest",
			err:        ErrBadRequest,
			wantCode:   "BAD_REQUEST",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "ErrInternalServer",
			err:        ErrInternalServer,
			wantCode:   "INTERNAL_ERROR",
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "ErrConflict",
			err:        ErrConflict,
			wantCode:   "CONFLICT",
			wantStatus: http.StatusConflict,
		},
		{
			name:       "ErrUnprocessableEntity",
			err:        ErrUnprocessableEntity,
			wantCode:   "UNPROCESSABLE",
			wantStatus: http.StatusUnprocessableEntity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Fatal("predefined error is nil")
			}
			if tt.err.Code != tt.wantCode {
				t.Errorf("Code = %v, want %v", tt.err.Code, tt.wantCode)
			}
			if tt.err.Status != tt.wantStatus {
				t.Errorf("Status = %v, want %v", tt.err.Status, tt.wantStatus)
			}
			if tt.err.Message == "" {
				t.Error("Message should not be empty")
			}
		})
	}
}

func TestAPIError_ImplementsError(t *testing.T) {
	// Verify APIError implements error interface
	var _ error = &APIError{}
	var _ error = NewAPIError("TEST", "test", http.StatusOK)
	var _ error = WrapAPIError(nil, "TEST", "test", http.StatusOK)
}

func TestAPIError_StatusCodes(t *testing.T) {
	statusCodes := []int{
		http.StatusOK,
		http.StatusCreated,
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusUnprocessableEntity,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
	}

	for _, status := range statusCodes {
		t.Run(http.StatusText(status), func(t *testing.T) {
			err := NewAPIError("CODE", "message", status)
			if err.Status != status {
				t.Errorf("Status = %d, want %d", err.Status, status)
			}
		})
	}
}

func TestAPIError_EmptyFields(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		message string
		status  int
	}{
		{
			name:    "empty code",
			code:    "",
			message: "Some message",
			status:  http.StatusBadRequest,
		},
		{
			name:    "empty message",
			code:    "SOME_CODE",
			message: "",
			status:  http.StatusBadRequest,
		},
		{
			name:    "zero status",
			code:    "SOME_CODE",
			message: "Some message",
			status:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewAPIError(tt.code, tt.message, tt.status)

			// Should not panic and should create error
			if err == nil {
				t.Fatal("NewAPIError() returned nil")
			}
			if err.Code != tt.code {
				t.Errorf("Code = %v, want %v", err.Code, tt.code)
			}
			if err.Message != tt.message {
				t.Errorf("Message = %v, want %v", err.Message, tt.message)
			}
			if err.Status != tt.status {
				t.Errorf("Status = %v, want %v", err.Status, tt.status)
			}
		})
	}
}

func TestAPIError_ChainedWrapping(t *testing.T) {
	// Test multiple levels of error wrapping
	level1 := errors.New("level 1 error")
	level2 := WrapAPIError(level1, "LEVEL2", "Level 2", http.StatusBadRequest)
	level3 := WrapAPIError(level2, "LEVEL3", "Level 3", http.StatusInternalServerError)

	// errors.Is should work through the chain
	if !errors.Is(level3, level1) {
		t.Error("errors.Is should find level1 through chain")
	}

	// Unwrap should return immediate wrapped error
	if level3.Unwrap() != level2 {
		t.Error("level3.Unwrap() should return level2")
	}
}

// containsSubstring is a helper to check if a string contains a substring.
func containsSubstring(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
