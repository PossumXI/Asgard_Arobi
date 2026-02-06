// Package vault implements HTTP API handlers for the security vault.
//
// Provides RESTful endpoints for vault management with proper authentication
// and authorization checks.
//
// Copyright 2026 Arobi. All Rights Reserved.
package vault

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

// VaultAPI provides HTTP handlers for the vault
type VaultAPI struct {
	vault        *Vault
	fido2Manager *FIDO2Manager
}

// NewVaultAPI creates a new vault API handler
func NewVaultAPI(vault *Vault) *VaultAPI {
	return &VaultAPI{
		vault:        vault,
		fido2Manager: vault.fido2Manager,
	}
}

// RegisterRoutes registers HTTP routes for the vault API
func (api *VaultAPI) RegisterRoutes(mux *http.ServeMux) {
	// Health and status
	mux.HandleFunc("/vault/health", api.handleHealth)
	mux.HandleFunc("/vault/status", api.handleStatus)
	mux.HandleFunc("/vault/stats", api.handleStats)

	// Vault operations
	mux.HandleFunc("/vault/seal", api.handleSeal)
	mux.HandleFunc("/vault/unseal", api.handleUnseal)

	// Secret management
	mux.HandleFunc("/vault/secrets", api.handleSecrets)
	mux.HandleFunc("/vault/secrets/", api.handleSecretByID)

	// FIDO2 endpoints
	mux.HandleFunc("/vault/fido2/register/begin", api.handleFIDO2RegisterBegin)
	mux.HandleFunc("/vault/fido2/register/complete", api.handleFIDO2RegisterComplete)
	mux.HandleFunc("/vault/fido2/authenticate/begin", api.handleFIDO2AuthBegin)
	mux.HandleFunc("/vault/fido2/authenticate/complete", api.handleFIDO2AuthComplete)
	mux.HandleFunc("/vault/fido2/credentials", api.handleFIDO2Credentials)

	// Audit endpoints
	mux.HandleFunc("/vault/audit", api.handleAudit)
	mux.HandleFunc("/vault/audit/stats", api.handleAuditStats)

	// Agent endpoints
	mux.HandleFunc("/vault/agent/status", api.handleAgentStatus)
	mux.HandleFunc("/vault/agent/anomalies", api.handleAgentAnomalies)
}

// API Response types

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type HealthResponse struct {
	Status    string `json:"status"`
	Sealed    bool   `json:"sealed"`
	Timestamp string `json:"timestamp"`
}

type StoreSecretAPIRequest struct {
	Name          string            `json:"name"`
	Description   string            `json:"description,omitempty"`
	Data          string            `json:"data"`
	Type          string            `json:"type"`
	SecurityLevel string            `json:"security_level"`
	Tags          []string          `json:"tags,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	ExpiresAt     string            `json:"expires_at,omitempty"`
	FIDO2Token    string            `json:"fido2_token,omitempty"`
}

type UnsealRequest struct {
	MasterPassword string `json:"master_password"`
}

// Handlers

func (api *VaultAPI) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.methodNotAllowed(w)
		return
	}

	stats := api.vault.GetStatistics()
	response := HealthResponse{
		Status:    "healthy",
		Sealed:    stats.Sealed,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if stats.Sealed {
		response.Status = "sealed"
	}

	api.jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    response,
	})
}

func (api *VaultAPI) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.methodNotAllowed(w)
		return
	}

	stats := api.vault.GetStatistics()
	api.jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    stats,
	})
}

func (api *VaultAPI) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.methodNotAllowed(w)
		return
	}

	vaultStats := api.vault.GetStatistics()
	fido2Stats := api.fido2Manager.GetStatistics()
	agentStats := api.vault.agentMonitor.GetStatistics()

	api.jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"vault":  vaultStats,
			"fido2":  fido2Stats,
			"agent":  agentStats,
		},
	})
}

func (api *VaultAPI) handleSeal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.methodNotAllowed(w)
		return
	}

	if err := api.vault.Seal(); err != nil {
		api.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	api.jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    map[string]string{"status": "sealed"},
	})
}

func (api *VaultAPI) handleUnseal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.methodNotAllowed(w)
		return
	}

	var req UnsealRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.errorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := api.vault.Unseal(req.MasterPassword); err != nil {
		api.errorResponse(w, http.StatusUnauthorized, err.Error())
		return
	}

	api.jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    map[string]string{"status": "unsealed"},
	})
}

func (api *VaultAPI) handleSecrets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		api.listSecrets(w, r)
	case http.MethodPost:
		api.storeSecret(w, r)
	default:
		api.methodNotAllowed(w)
	}
}

func (api *VaultAPI) listSecrets(w http.ResponseWriter, r *http.Request) {
	// Get filter parameters
	filter := ListSecretsFilter{
		SecurityLevel: SecurityLevel(r.URL.Query().Get("security_level")),
		Type:          SecretType(r.URL.Query().Get("type")),
		Tag:           r.URL.Query().Get("tag"),
	}

	secrets, err := api.vault.ListSecrets(r.Context(), filter)
	if err != nil {
		api.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	api.jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    secrets,
	})
}

func (api *VaultAPI) storeSecret(w http.ResponseWriter, r *http.Request) {
	var req StoreSecretAPIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.errorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get actor from auth header (simplified - production would validate JWT)
	actor := r.Header.Get("X-Vault-Actor")
	if actor == "" {
		actor = "anonymous"
	}

	// Parse expiration if provided
	var expiresAt *time.Time
	if req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			api.errorResponse(w, http.StatusBadRequest, "Invalid expires_at format")
			return
		}
		expiresAt = &t
	}

	// Create store request
	storeReq := StoreSecretRequest{
		Name:          req.Name,
		Description:   req.Description,
		Data:          req.Data,
		Type:          SecretType(req.Type),
		SecurityLevel: SecurityLevel(req.SecurityLevel),
		Actor:         actor,
		Tags:          req.Tags,
		Metadata:      req.Metadata,
		ExpiresAt:     expiresAt,
	}

	// Check if FIDO2 required and get credential
	if api.vault.config.RequireFIDO2[storeReq.SecurityLevel] && req.FIDO2Token != "" {
		creds := api.fido2Manager.GetUserCredentials(actor)
		if len(creds) > 0 {
			storeReq.FIDO2Credential = creds[0]
		}
	}

	entry, err := api.vault.StoreSecret(r.Context(), storeReq)
	if err != nil {
		api.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Redact encrypted data in response
	entry.EncryptedData = "[STORED]"

	api.jsonResponse(w, http.StatusCreated, APIResponse{
		Success: true,
		Data:    entry,
	})
}

func (api *VaultAPI) handleSecretByID(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path /vault/secrets/{id}
	path := strings.TrimPrefix(r.URL.Path, "/vault/secrets/")
	id := strings.TrimSuffix(path, "/")

	if id == "" {
		api.errorResponse(w, http.StatusBadRequest, "Secret ID required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		api.retrieveSecret(w, r, id)
	case http.MethodDelete:
		api.deleteSecret(w, r, id)
	default:
		api.methodNotAllowed(w)
	}
}

func (api *VaultAPI) retrieveSecret(w http.ResponseWriter, r *http.Request, id string) {
	actor := r.Header.Get("X-Vault-Actor")
	if actor == "" {
		actor = "anonymous"
	}

	req := RetrieveSecretRequest{
		EntryID: id,
		Actor:   actor,
	}

	// Check for FIDO2 token
	fido2Token := r.Header.Get("X-FIDO2-Token")
	if fido2Token != "" {
		creds := api.fido2Manager.GetUserCredentials(actor)
		if len(creds) > 0 {
			req.FIDO2Credential = creds[0]
		}
	}

	secret, err := api.vault.RetrieveSecret(r.Context(), req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		} else if strings.Contains(err.Error(), "FIDO2") || strings.Contains(err.Error(), "sealed") {
			statusCode = http.StatusUnauthorized
		}
		api.errorResponse(w, statusCode, err.Error())
		return
	}

	api.jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]string{
			"id":    id,
			"value": secret,
		},
	})
}

func (api *VaultAPI) deleteSecret(w http.ResponseWriter, r *http.Request, id string) {
	actor := r.Header.Get("X-Vault-Actor")
	if actor == "" {
		actor = "anonymous"
	}

	req := DeleteSecretRequest{
		EntryID: id,
		Actor:   actor,
	}

	// Check for FIDO2 token
	fido2Token := r.Header.Get("X-FIDO2-Token")
	if fido2Token != "" {
		creds := api.fido2Manager.GetUserCredentials(actor)
		if len(creds) > 0 {
			req.FIDO2Credential = creds[0]
		}
	}

	if err := api.vault.DeleteSecret(r.Context(), req); err != nil {
		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		} else if strings.Contains(err.Error(), "FIDO2") {
			statusCode = http.StatusUnauthorized
		}
		api.errorResponse(w, statusCode, err.Error())
		return
	}

	api.jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    map[string]string{"deleted": id},
	})
}

// FIDO2 handlers

func (api *VaultAPI) handleFIDO2RegisterBegin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.methodNotAllowed(w)
		return
	}

	var req struct {
		UserID      string `json:"user_id"`
		UserName    string `json:"user_name"`
		DisplayName string `json:"display_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.errorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	challenge, err := api.fido2Manager.BeginRegistration(r.Context(), req.UserID, req.UserName, req.DisplayName)
	if err != nil {
		api.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	api.jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    challenge,
	})
}

func (api *VaultAPI) handleFIDO2RegisterComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.methodNotAllowed(w)
		return
	}

	var req FIDO2RegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.errorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	credential, err := api.fido2Manager.CompleteRegistration(r.Context(), &req)
	if err != nil {
		api.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Don't return private key material
	credential.PublicKey = nil

	api.jsonResponse(w, http.StatusCreated, APIResponse{
		Success: true,
		Data:    credential,
	})
}

func (api *VaultAPI) handleFIDO2AuthBegin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.methodNotAllowed(w)
		return
	}

	var req struct {
		UserID string `json:"user_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.errorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	challenge, allowedCreds, err := api.fido2Manager.BeginAuthentication(r.Context(), req.UserID)
	if err != nil {
		api.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	api.jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"challenge":           challenge,
			"allowed_credentials": allowedCreds,
		},
	})
}

func (api *VaultAPI) handleFIDO2AuthComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.methodNotAllowed(w)
		return
	}

	var cred FIDO2Credential
	if err := json.NewDecoder(r.Body).Decode(&cred); err != nil {
		api.errorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := api.fido2Manager.VerifyAssertion(r.Context(), &cred); err != nil {
		api.errorResponse(w, http.StatusUnauthorized, err.Error())
		return
	}

	api.jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    map[string]string{"status": "authenticated"},
	})
}

func (api *VaultAPI) handleFIDO2Credentials(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.methodNotAllowed(w)
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		api.errorResponse(w, http.StatusBadRequest, "user_id required")
		return
	}

	creds := api.fido2Manager.GetUserCredentials(userID)

	// Redact sensitive data
	for _, cred := range creds {
		cred.PublicKey = nil
		cred.CredentialID = nil
	}

	api.jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    creds,
	})
}

// Audit handlers

func (api *VaultAPI) handleAudit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.methodNotAllowed(w)
		return
	}

	count := 100 // Default
	events := api.vault.auditLog.GetRecentEvents(count)

	api.jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    events,
	})
}

func (api *VaultAPI) handleAuditStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.methodNotAllowed(w)
		return
	}

	stats := api.vault.auditLog.GetStatistics()

	api.jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    stats,
	})
}

// Agent handlers

func (api *VaultAPI) handleAgentStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.methodNotAllowed(w)
		return
	}

	stats := api.vault.agentMonitor.GetStatistics()

	api.jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    stats,
	})
}

func (api *VaultAPI) handleAgentAnomalies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.methodNotAllowed(w)
		return
	}

	anomalies := api.vault.agentMonitor.GetAnomalies()

	api.jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    anomalies,
	})
}

// Helper methods

func (api *VaultAPI) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("[VaultAPI] Failed to encode response: %v", err)
	}
}

func (api *VaultAPI) errorResponse(w http.ResponseWriter, status int, message string) {
	api.jsonResponse(w, status, APIResponse{
		Success: false,
		Error:   message,
	})
}

func (api *VaultAPI) methodNotAllowed(w http.ResponseWriter) {
	api.errorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
}

// StartVaultServer starts the vault HTTP server
func StartVaultServer(ctx context.Context, addr string, vault *Vault) error {
	api := NewVaultAPI(vault)
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

	log.Printf("[VaultAPI] Starting server on %s", addr)
	return server.ListenAndServe()
}
