// Package response provides standardized API response types.
package response

import (
	"encoding/json"
	"net/http"
)

// Response represents a standard API response.
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *Error      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// Error represents an API error.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"status"`
}

// Success sends a successful JSON response.
func Success(w http.ResponseWriter, status int, data interface{}) {
	response := Response{
		Success: true,
		Data:    data,
	}
	sendJSON(w, status, response)
}

// SendError sends an error JSON response.
func SendError(w http.ResponseWriter, status int, code, message string) {
	response := Response{
		Success: false,
		Error: &Error{
			Code:    code,
			Message: message,
			Status:  status,
		},
	}
	sendJSON(w, status, response)
}

// PaginatedResponse represents a paginated API response.
type PaginatedResponse struct {
	Data     interface{} `json:"data"`
	Total    int         `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
	HasMore  bool        `json:"hasMore"`
}

// Paginated sends a paginated JSON response.
func Paginated(w http.ResponseWriter, data interface{}, total, page, pageSize int) {
	response := PaginatedResponse{
		Data:     data,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		HasMore:  (page*pageSize) < total,
	}
	sendJSON(w, http.StatusOK, response)
}

func sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
