// Package notifications provides HTTP API for notification services.
//
// Copyright 2026 Arobi. All Rights Reserved.
package notifications

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/asgard/pandora/internal/notifications/email"
)

// NotificationAPI provides HTTP handlers for notifications
type NotificationAPI struct {
	emailClient         *email.ResendClient
	accessKeyManager    *AccessKeyManager
	verificationManager *VerificationCodeManager
}

// NewNotificationAPI creates a new notification API
func NewNotificationAPI() *NotificationAPI {
	// Initialize Resend client
	apiKey := os.Getenv("RESEND_API_KEY")
	cfg := email.DefaultResendConfig()
	cfg.APIKey = apiKey

	var emailClient *email.ResendClient
	if apiKey != "" {
		emailClient = email.NewResendClient(cfg)
		log.Println("[NotificationAPI] Email client initialized with Resend")
	} else {
		log.Println("[NotificationAPI] Warning: RESEND_API_KEY not set, email notifications disabled")
	}

	// Initialize managers
	accessKeyCfg := DefaultAccessKeyConfig()
	accessKeyManager := NewAccessKeyManager(emailClient, accessKeyCfg)
	verificationManager := NewVerificationCodeManager(emailClient)

	return &NotificationAPI{
		emailClient:         emailClient,
		accessKeyManager:    accessKeyManager,
		verificationManager: verificationManager,
	}
}

// RegisterRoutes registers HTTP routes
func (api *NotificationAPI) RegisterRoutes(mux *http.ServeMux) {
	// Access key endpoints
	mux.HandleFunc("/api/access-keys", api.handleAccessKeys)
	mux.HandleFunc("/api/access-keys/", api.handleAccessKeyByID)
	mux.HandleFunc("/api/access-keys/validate", api.handleValidateKey)
	mux.HandleFunc("/api/access-keys/founder", api.handleGenerateFounderKey)

	// Verification endpoints
	mux.HandleFunc("/api/verify/send", api.handleSendVerification)
	mux.HandleFunc("/api/verify/check", api.handleCheckVerification)

	// Notification endpoints
	mux.HandleFunc("/api/notify/alert", api.handleSendAlert)

	// Status endpoint
	mux.HandleFunc("/api/notifications/status", api.handleStatus)
}

// API Response types
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type GenerateKeyAPIRequest struct {
	KeyType     string   `json:"key_type"`
	IssuedTo    string   `json:"issued_to"`
	Description string   `json:"description,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	SendEmail   bool     `json:"send_email"`
	EmailTo     string   `json:"email_to,omitempty"`
	ExpiresIn   string   `json:"expires_in,omitempty"` // e.g., "24h", "7d"
}

type ValidateKeyRequest struct {
	Key string `json:"key"`
}

type SendVerificationRequest struct {
	Email string `json:"email"`
}

type CheckVerificationRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type SendAlertRequest struct {
	To          string `json:"to"`
	AlertType   string `json:"alert_type"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
}

// Handlers

func (api *NotificationAPI) handleAccessKeys(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		keys := api.accessKeyManager.ListKeys()
		api.jsonResponse(w, http.StatusOK, APIResponse{Success: true, Data: keys})

	case http.MethodPost:
		var req GenerateKeyAPIRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			api.errorResponse(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Parse expiration
		var expiration time.Duration
		if req.ExpiresIn != "" {
			var err error
			expiration, err = time.ParseDuration(req.ExpiresIn)
			if err != nil {
				api.errorResponse(w, http.StatusBadRequest, "Invalid expires_in format")
				return
			}
		}

		genReq := GenerateKeyRequest{
			KeyType:     KeyType(req.KeyType),
			IssuedTo:    req.IssuedTo,
			IssuedBy:    r.Header.Get("X-User-ID"),
			Description: req.Description,
			Permissions: req.Permissions,
			SendEmail:   req.SendEmail,
			EmailTo:     req.EmailTo,
			Expiration:  expiration,
		}

		key, rawKey, err := api.accessKeyManager.GenerateAccessKey(r.Context(), genReq)
		if err != nil {
			api.errorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		api.jsonResponse(w, http.StatusCreated, APIResponse{
			Success: true,
			Data: map[string]interface{}{
				"key_id":     key.ID,
				"key":        rawKey, // Only returned once!
				"key_type":   key.KeyType,
				"expires_at": key.ExpiresAt,
				"issued_to":  key.IssuedTo,
			},
		})

	default:
		api.methodNotAllowed(w)
	}
}

func (api *NotificationAPI) handleAccessKeyByID(w http.ResponseWriter, r *http.Request) {
	// Extract key ID from path
	keyID := r.URL.Path[len("/api/access-keys/"):]
	if keyID == "" {
		api.errorResponse(w, http.StatusBadRequest, "Key ID required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		key := api.accessKeyManager.GetKey(keyID)
		if key == nil {
			api.errorResponse(w, http.StatusNotFound, "Key not found")
			return
		}
		api.jsonResponse(w, http.StatusOK, APIResponse{Success: true, Data: key})

	case http.MethodDelete:
		reason := r.URL.Query().Get("reason")
		revokedBy := r.Header.Get("X-User-ID")
		if revokedBy == "" {
			revokedBy = "system"
		}

		if err := api.accessKeyManager.RevokeKey(r.Context(), keyID, revokedBy, reason); err != nil {
			api.errorResponse(w, http.StatusBadRequest, err.Error())
			return
		}
		api.jsonResponse(w, http.StatusOK, APIResponse{Success: true, Data: map[string]string{"revoked": keyID}})

	default:
		api.methodNotAllowed(w)
	}
}

func (api *NotificationAPI) handleValidateKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.methodNotAllowed(w)
		return
	}

	var req ValidateKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.errorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	key, err := api.accessKeyManager.ValidateKey(r.Context(), req.Key)
	if err != nil {
		api.errorResponse(w, http.StatusUnauthorized, err.Error())
		return
	}

	api.jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"valid":       true,
			"key_type":    key.KeyType,
			"permissions": key.Permissions,
			"issued_to":   key.IssuedTo,
		},
	})
}

func (api *NotificationAPI) handleGenerateFounderKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.methodNotAllowed(w)
		return
	}

	// This endpoint requires special authorization
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		api.errorResponse(w, http.StatusUnauthorized, "Authorization required")
		return
	}

	issuedBy := r.Header.Get("X-User-ID")
	if issuedBy == "" {
		issuedBy = "system"
	}

	key, rawKey, err := api.accessKeyManager.GenerateFounderKey(r.Context(), issuedBy)
	if err != nil {
		api.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	api.jsonResponse(w, http.StatusCreated, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"key_id":      key.ID,
			"key":         rawKey, // Sent via email AND returned
			"key_type":    key.KeyType,
			"expires_at":  key.ExpiresAt,
			"emailed_to":  "Gaetano@aura-genesis.org",
			"permissions": key.Permissions,
		},
	})
}

func (api *NotificationAPI) handleSendVerification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.methodNotAllowed(w)
		return
	}

	var req SendVerificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.errorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Email == "" {
		api.errorResponse(w, http.StatusBadRequest, "Email required")
		return
	}

	if err := api.verificationManager.GenerateVerificationCode(r.Context(), req.Email); err != nil {
		api.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	api.jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    map[string]string{"message": "Verification code sent"},
	})
}

func (api *NotificationAPI) handleCheckVerification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.methodNotAllowed(w)
		return
	}

	var req CheckVerificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.errorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := api.verificationManager.VerifyCode(req.Email, req.Code); err != nil {
		api.errorResponse(w, http.StatusUnauthorized, err.Error())
		return
	}

	api.jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    map[string]bool{"verified": true},
	})
}

func (api *NotificationAPI) handleSendAlert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.methodNotAllowed(w)
		return
	}

	var req SendAlertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.errorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if api.emailClient == nil {
		api.errorResponse(w, http.StatusServiceUnavailable, "Email service not configured")
		return
	}

	if err := api.emailClient.SendSecurityAlertEmail(r.Context(), req.To, req.AlertType, req.Description, req.Severity); err != nil {
		api.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	api.jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    map[string]string{"message": "Alert sent"},
	})
}

func (api *NotificationAPI) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.methodNotAllowed(w)
		return
	}

	emailEnabled := api.emailClient != nil && os.Getenv("RESEND_API_KEY") != ""

	api.jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"email_enabled":  emailEnabled,
			"email_provider": "Resend",
			"founder_email":  "Gaetano@aura-genesis.org",
			"active_keys":    len(api.accessKeyManager.ListKeys()),
		},
	})
}

// Helper methods

func (api *NotificationAPI) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (api *NotificationAPI) errorResponse(w http.ResponseWriter, status int, message string) {
	api.jsonResponse(w, status, APIResponse{Success: false, Error: message})
}

func (api *NotificationAPI) methodNotAllowed(w http.ResponseWriter) {
	api.errorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
}

// StartNotificationServer starts the notification HTTP server
func StartNotificationServer(ctx context.Context, addr string) error {
	api := NewNotificationAPI()
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()

	log.Printf("[NotificationAPI] Starting server on %s", addr)
	return server.ListenAndServe()
}
