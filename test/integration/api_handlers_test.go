package integration_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/asgard/pandora/internal/api/handlers"
)

func TestHealthHandler(t *testing.T) {
	handler := handlers.NewHealthHandler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.Health(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("expected status 'ok', got '%v'", response["status"])
	}

	if response["service"] != "nysus" {
		t.Errorf("expected service 'nysus', got '%v'", response["service"])
	}

	// Verify timestamp is present
	if response["timestamp"] == nil {
		t.Error("expected timestamp in response")
	}
}

func TestHealthHandlerResponseFormat(t *testing.T) {
	handler := handlers.NewHealthHandler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.Health(w, req)

	// Check Content-Type header
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Logf("Content-Type: %s", contentType)
	}

	// Verify body is valid JSON
	var response interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("response is not valid JSON: %v", err)
	}
}

func TestHealthHandlerMultipleCalls(t *testing.T) {
	handler := handlers.NewHealthHandler()

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()

		handler.Health(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("call %d: expected status 200, got %d", i, w.Code)
		}
	}
}
