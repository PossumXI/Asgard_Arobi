// Package handlers provides HTTP handlers for API endpoints.
package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewHealthHandler(t *testing.T) {
	handler := NewHealthHandler()
	if handler == nil {
		t.Fatal("NewHealthHandler() returned nil")
	}
}

func TestHealth_Success(t *testing.T) {
	handler := NewHealthHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rr := httptest.NewRecorder()

	handler.Health(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Health() status = %d, want %d", rr.Code, http.StatusOK)
	}

	// Check content type
	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Health() Content-Type = %s, want application/json", contentType)
	}

	// Parse response
	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check required fields
	if response["status"] != "ok" {
		t.Errorf("response.status = %v, want ok", response["status"])
	}

	if response["service"] != "nysus" {
		t.Errorf("response.service = %v, want nysus", response["service"])
	}

	if response["version"] != "1.0.0" {
		t.Errorf("response.version = %v, want 1.0.0", response["version"])
	}

	// Check timestamp format
	timestamp, ok := response["timestamp"].(string)
	if !ok {
		t.Fatal("response.timestamp is not a string")
	}

	_, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		t.Errorf("response.timestamp is not valid RFC3339: %v", err)
	}
}

func TestHealth_HTTPMethods(t *testing.T) {
	handler := NewHealthHandler()

	// Health endpoint should respond to any method (usually GET)
	methods := []string{
		http.MethodGet,
		http.MethodHead,
		http.MethodPost,   // Typically not used but shouldn't crash
		http.MethodPut,    // Typically not used but shouldn't crash
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/health", nil)
			rr := httptest.NewRecorder()

			handler.Health(rr, req)

			// Should not panic and should return a response
			if rr.Code == 0 {
				t.Error("Health() did not set status code")
			}
		})
	}
}

func TestHealth_ResponseStructure(t *testing.T) {
	handler := NewHealthHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rr := httptest.NewRecorder()

	handler.Health(rr, req)

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify all required fields exist
	requiredFields := []string{"status", "timestamp", "service", "version"}
	for _, field := range requiredFields {
		if _, exists := response[field]; !exists {
			t.Errorf("Health() response missing required field: %s", field)
		}
	}
}

func TestHealth_TimestampIsRecent(t *testing.T) {
	handler := NewHealthHandler()

	before := time.Now().UTC()

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rr := httptest.NewRecorder()

	handler.Health(rr, req)

	after := time.Now().UTC()

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	timestamp, _ := response["timestamp"].(string)
	ts, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		t.Fatalf("Failed to parse timestamp: %v", err)
	}

	// Timestamp should be between before and after
	if ts.Before(before.Add(-1*time.Second)) || ts.After(after.Add(1*time.Second)) {
		t.Errorf("Timestamp %v not in expected range [%v, %v]", ts, before, after)
	}
}

func TestHealth_ConcurrentRequests(t *testing.T) {
	handler := NewHealthHandler()

	// Test that health endpoint is safe for concurrent access
	done := make(chan bool)
	for i := 0; i < 100; i++ {
		go func() {
			req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
			rr := httptest.NewRecorder()
			handler.Health(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("Concurrent Health() status = %d, want %d", rr.Code, http.StatusOK)
			}
			done <- true
		}()
	}

	for i := 0; i < 100; i++ {
		<-done
	}
}

// TestJsonResponse tests the jsonResponse helper function.
func TestJsonResponse(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		data       interface{}
		wantStatus int
	}{
		{
			name:       "ok status",
			status:     http.StatusOK,
			data:       map[string]string{"message": "success"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "created status",
			status:     http.StatusCreated,
			data:       map[string]int{"id": 123},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "not found status",
			status:     http.StatusNotFound,
			data:       map[string]string{"error": "not found"},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "nil data",
			status:     http.StatusNoContent,
			data:       nil,
			wantStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			jsonResponse(rr, tt.status, tt.data)

			if rr.Code != tt.wantStatus {
				t.Errorf("jsonResponse() status = %d, want %d", rr.Code, tt.wantStatus)
			}

			contentType := rr.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("jsonResponse() Content-Type = %s, want application/json", contentType)
			}
		})
	}
}

// TestJsonError tests the jsonError helper function.
func TestJsonError(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		message    string
		code       string
		wantStatus int
	}{
		{
			name:       "bad request",
			status:     http.StatusBadRequest,
			message:    "Invalid input",
			code:       "BAD_REQUEST",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unauthorized",
			status:     http.StatusUnauthorized,
			message:    "Authentication required",
			code:       "UNAUTHORIZED",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "internal server error",
			status:     http.StatusInternalServerError,
			message:    "Something went wrong",
			code:       "INTERNAL_ERROR",
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			jsonError(rr, tt.status, tt.message, tt.code)

			if rr.Code != tt.wantStatus {
				t.Errorf("jsonError() status = %d, want %d", rr.Code, tt.wantStatus)
			}

			var response map[string]interface{}
			if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode error response: %v", err)
			}

			errorObj, ok := response["error"].(map[string]interface{})
			if !ok {
				t.Fatal("response.error is not an object")
			}

			if errorObj["message"] != tt.message {
				t.Errorf("error.message = %v, want %s", errorObj["message"], tt.message)
			}
			if errorObj["code"] != tt.code {
				t.Errorf("error.code = %v, want %s", errorObj["code"], tt.code)
			}
		})
	}
}

// TestValidateEmail tests the email validation helper.
func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  bool
	}{
		{"valid email", "test@example.com", true},
		{"valid with subdomain", "user@mail.example.com", true},
		{"valid with plus", "user+tag@example.com", true},
		{"too short", "a@", false},
		{"no at sign", "testexample.com", false},
		{"no domain", "test@", false},
		{"no dot after at", "test@example", false},
		{"double at", "test@@example.com", false},
		{"at at start", "@example.com", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validateEmail(tt.email)
			if got != tt.want {
				t.Errorf("validateEmail(%q) = %v, want %v", tt.email, got, tt.want)
			}
		})
	}
}

// TestValidatePassword tests the password validation helper.
func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		want     bool
	}{
		{"valid password 8 chars", "password", true},
		{"valid password 16 chars", "strongPassword12", true},
		{"too short 7 chars", "pass123", false},
		{"too short 1 char", "a", false},
		{"empty string", "", false},
		{"exactly 128 chars", string(make([]byte, 128)), true},
		{"too long 129 chars", string(make([]byte, 129)), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For the 128/129 char tests, fill with valid characters
			password := tt.password
			if len(tt.password) > 100 {
				password = ""
				for i := 0; i < len(tt.password); i++ {
					password += "a"
				}
			}

			got := validatePassword(password)
			if got != tt.want {
				t.Errorf("validatePassword(%q...) = %v, want %v", password[:min(10, len(password))], got, tt.want)
			}
		})
	}
}

// TestParsePaginationParams tests pagination parameter parsing.
func TestParsePaginationParams(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantLimit  int
		wantOffset int
	}{
		{
			name:       "defaults",
			query:      "",
			wantLimit:  20,
			wantOffset: 0,
		},
		{
			name:       "custom limit",
			query:      "?limit=50",
			wantLimit:  50,
			wantOffset: 0,
		},
		{
			name:       "custom offset",
			query:      "?offset=10",
			wantLimit:  20,
			wantOffset: 10,
		},
		{
			name:       "both custom",
			query:      "?limit=30&offset=15",
			wantLimit:  30,
			wantOffset: 15,
		},
		{
			name:       "limit exceeds max",
			query:      "?limit=200",
			wantLimit:  20, // Falls back to default
			wantOffset: 0,
		},
		{
			name:       "negative limit",
			query:      "?limit=-5",
			wantLimit:  20, // Falls back to default
			wantOffset: 0,
		},
		{
			name:       "negative offset",
			query:      "?offset=-10",
			wantLimit:  20,
			wantOffset: 0, // Falls back to default
		},
		{
			name:       "invalid limit format",
			query:      "?limit=abc",
			wantLimit:  20,
			wantOffset: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/test"+tt.query, nil)
			limit, offset := parsePaginationParams(req)

			if limit != tt.wantLimit {
				t.Errorf("parsePaginationParams() limit = %d, want %d", limit, tt.wantLimit)
			}
			if offset != tt.wantOffset {
				t.Errorf("parsePaginationParams() offset = %d, want %d", offset, tt.wantOffset)
			}
		})
	}
}

// TestParseInt tests the parseInt helper.
func TestParseInt(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want int
	}{
		{"zero", "0", 0},
		{"positive", "123", 123},
		{"large number", "999999", 999999},
		{"empty string", "", 0},
		{"letters", "abc", 0},
		{"mixed", "12a34", 0},
		{"negative (returns 0)", "-5", 0},
		{"float (returns 0)", "12.34", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseInt(tt.s)
			if got != tt.want {
				t.Errorf("parseInt(%q) = %d, want %d", tt.s, got, tt.want)
			}
		})
	}
}

// min helper for tests
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
