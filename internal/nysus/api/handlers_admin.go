package api

import (
	"crypto/rand"
	"database/sql"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/asgard/pandora/internal/controlplane"
	"github.com/asgard/pandora/internal/platform/realtime"
	"github.com/asgard/pandora/internal/services"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type adminUser struct {
	ID               string `json:"id"`
	Email            string `json:"email"`
	FullName         string `json:"fullName"`
	SubscriptionTier string `json:"subscriptionTier"`
	IsGovernment     bool   `json:"isGovernment"`
	CreatedAt        string `json:"createdAt"`
	UpdatedAt        string `json:"updatedAt"`
}

type adminUserUpdate struct {
	FullName         *string `json:"fullName"`
	SubscriptionTier *string `json:"subscriptionTier"`
	IsGovernment     *bool   `json:"isGovernment"`
}

type adminUserCreate struct {
	Email             string `json:"email"`
	FullName          string `json:"fullName"`
	Password          string `json:"password,omitempty"`
	SubscriptionTier  string `json:"subscriptionTier,omitempty"`
	IsGovernment      bool   `json:"isGovernment,omitempty"`
	CreateAccessCode  bool   `json:"createAccessCode,omitempty"`
	ClearanceLevel     string `json:"clearanceLevel,omitempty"`
}

type controlCommandRequest struct {
	TargetDomain string                 `json:"targetDomain"`
	TargetSystem string                 `json:"targetSystem"`
	CommandType  string                 `json:"commandType"`
	Parameters   map[string]interface{} `json:"parameters"`
	Priority     int                    `json:"priority"`
}

func (s *Server) handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminAccess(w, r) {
		return
	}
	if s.pgDB == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Database not available", "DB_UNAVAILABLE")
		return
	}
	switch r.Method {
	case http.MethodGet:
		rows, err := s.pgDB.QueryContext(r.Context(), `
			SELECT id::text, email, COALESCE(full_name, ''), subscription_tier, is_government, created_at, updated_at
			FROM users
			ORDER BY created_at DESC
			LIMIT 200
		`)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "Failed to load users", "DB_ERROR")
			return
		}
		defer rows.Close()

		users := []adminUser{}
		for rows.Next() {
			var item adminUser
			var createdAt time.Time
			var updatedAt time.Time
			if scanErr := rows.Scan(&item.ID, &item.Email, &item.FullName, &item.SubscriptionTier, &item.IsGovernment, &createdAt, &updatedAt); scanErr != nil {
				continue
			}
			item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
			item.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
			users = append(users, item)
		}

		s.writeJSON(w, http.StatusOK, users)
	case http.MethodPost:
		var req adminUserCreate
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.writeError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST")
			return
		}
		req.Email = strings.TrimSpace(req.Email)
		req.FullName = strings.TrimSpace(req.FullName)
		if req.Email == "" || req.FullName == "" {
			s.writeError(w, http.StatusBadRequest, "Email and fullName required", "INVALID_REQUEST")
			return
		}
		tier := strings.ToLower(strings.TrimSpace(req.SubscriptionTier))
		if tier == "" {
			tier = "observer"
		}
		if tier == "free" {
			tier = "observer"
		}
		switch tier {
		case "observer", "supporter", "commander":
		default:
			s.writeError(w, http.StatusBadRequest, "Invalid subscription tier", "INVALID_TIER")
			return
		}

		password := strings.TrimSpace(req.Password)
		if password == "" {
			password = generateTemporaryPassword()
		}
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "Failed to generate password", "INTERNAL_ERROR")
			return
		}

		userID := uuid.New()
		now := time.Now().UTC()
		_, err = s.pgDB.ExecContext(r.Context(), `
			INSERT INTO users (id, email, password_hash, full_name, subscription_tier, is_government, email_verified, email_verified_at, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, true, $7, $8, $9)
		`, userID, req.Email, string(hashedPassword), req.FullName, tier, req.IsGovernment, now, now, now)
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
				s.writeError(w, http.StatusConflict, "Email already exists", "EMAIL_EXISTS")
				return
			}
			s.writeError(w, http.StatusInternalServerError, "Failed to create user", "DB_ERROR")
			return
		}

		response := map[string]interface{}{
			"user": adminUser{
				ID:               userID.String(),
				Email:            req.Email,
				FullName:         req.FullName,
				SubscriptionTier: tier,
				IsGovernment:     req.IsGovernment,
				CreatedAt:        now.Format(time.RFC3339),
				UpdatedAt:        now.Format(time.RFC3339),
			},
			"temporaryPassword": password,
		}

		if req.CreateAccessCode && s.accessCodeService != nil {
			clearance := strings.TrimSpace(req.ClearanceLevel)
			if clearance == "" {
				if req.IsGovernment {
					clearance = "government"
				} else {
					clearance = "interstellar"
				}
			}
			codeResult, err := s.accessCodeService.IssueForUser(r.Context(), services.AccessCodeIssueRequest{
				UserID:        userID.String(),
				CreatedBy:     s.getRequesterID(r),
				ClearanceLevel: clearance,
				Scope:         "all",
			})
			if err == nil {
				response["accessCode"] = codeResult.Code
			}
		}

		s.writeJSON(w, http.StatusCreated, response)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
	}
}

func (s *Server) handleAdminUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}
	if !s.requireAdminAccess(w, r) {
		return
	}
	if s.pgDB == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Database not available", "DB_UNAVAILABLE")
		return
	}

	userID := strings.TrimPrefix(r.URL.Path, "/api/admin/users/")
	userID = strings.Trim(userID, "/")
	if userID == "" {
		s.writeError(w, http.StatusBadRequest, "User ID required", "INVALID_REQUEST")
		return
	}
	if _, err := uuid.Parse(userID); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid user ID", "INVALID_REQUEST")
		return
	}

	var req adminUserUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST")
		return
	}

	fields := []string{}
	args := []interface{}{}
	argNum := 1

	if req.FullName != nil {
		fields = append(fields, fmt.Sprintf("full_name = $%d", argNum))
		args = append(args, strings.TrimSpace(*req.FullName))
		argNum++
	}

	if req.SubscriptionTier != nil {
		tier := strings.ToLower(strings.TrimSpace(*req.SubscriptionTier))
		if tier == "free" {
			tier = "observer"
		}
		switch tier {
		case "observer", "supporter", "commander":
			fields = append(fields, fmt.Sprintf("subscription_tier = $%d", argNum))
			args = append(args, tier)
			argNum++
		default:
			s.writeError(w, http.StatusBadRequest, "Invalid subscription tier", "INVALID_TIER")
			return
		}
	}

	if req.IsGovernment != nil {
		fields = append(fields, fmt.Sprintf("is_government = $%d", argNum))
		args = append(args, *req.IsGovernment)
		argNum++
	}

	if len(fields) == 0 {
		s.writeError(w, http.StatusBadRequest, "No updates provided", "INVALID_REQUEST")
		return
	}

	fields = append(fields, fmt.Sprintf("updated_at = $%d", argNum))
	args = append(args, time.Now().UTC())
	argNum++

	query := fmt.Sprintf("UPDATE users SET %s WHERE id = $%d RETURNING id::text, email, COALESCE(full_name, ''), subscription_tier, is_government, created_at, updated_at",
		strings.Join(fields, ", "),
		argNum,
	)
	args = append(args, userID)

	var item adminUser
	var createdAt time.Time
	var updatedAt time.Time
	err := s.pgDB.QueryRowContext(r.Context(), query, args...).Scan(&item.ID, &item.Email, &item.FullName, &item.SubscriptionTier, &item.IsGovernment, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			s.writeError(w, http.StatusNotFound, "User not found", "NOT_FOUND")
			return
		}
		s.writeError(w, http.StatusInternalServerError, "Failed to update user", "DB_ERROR")
		return
	}
	item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	item.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)

	s.writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleControlPlaneCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}
	if !s.requireAdminAccess(w, r) {
		return
	}

	var req controlCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST")
		return
	}
	req.TargetDomain = strings.TrimSpace(req.TargetDomain)
	req.CommandType = strings.TrimSpace(req.CommandType)
	req.TargetSystem = strings.TrimSpace(req.TargetSystem)

	if req.TargetDomain == "" || req.CommandType == "" {
		s.writeError(w, http.StatusBadRequest, "targetDomain and commandType are required", "INVALID_REQUEST")
		return
	}

	cmd := controlplane.ControlCommand{
		ID:           uuid.New(),
		Timestamp:    time.Now().UTC(),
		Source:       "nysus",
		TargetDomain: controlplane.EventDomain(req.TargetDomain),
		TargetSystem: req.TargetSystem,
		CommandType:  req.CommandType,
		Parameters:   req.Parameters,
		Priority:     req.Priority,
	}

	payload := map[string]interface{}{
		"id":           cmd.ID.String(),
		"timestamp":    cmd.Timestamp,
		"source":       cmd.Source,
		"targetDomain": cmd.TargetDomain,
		"targetSystem": cmd.TargetSystem,
		"commandType":  cmd.CommandType,
		"parameters":   cmd.Parameters,
		"priority":     cmd.Priority,
	}

	if s.natsBridge != nil {
		_ = s.natsBridge.Publish("asgard.controlplane.command", realtime.Event{
			ID:          cmd.ID.String(),
			Type:        realtime.EventTypeControlCommand,
			Source:      "nysus",
			Timestamp:   time.Now().UTC(),
			Payload:     payload,
			AccessLevel: realtime.AccessLevelGovernment,
			Priority:    req.Priority,
		})
	}

	if s.wsManager != nil {
		s.wsManager.Broadcast(realtime.Event{
			ID:          cmd.ID.String(),
			Type:        realtime.EventTypeControlCommand,
			Source:      "nysus",
			Timestamp:   time.Now().UTC(),
			Payload:     payload,
			AccessLevel: realtime.AccessLevelGovernment,
			Priority:    req.Priority,
		})
	}

	s.writeJSON(w, http.StatusOK, payload)
}

func (s *Server) handleControlPlaneStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}
	if !s.requireAdminAccess(w, r) {
		return
	}

	response := map[string]interface{}{
		"timestamp": time.Now().UTC(),
	}

	if s.natsBridge != nil {
		response["nats"] = s.natsBridge.Stats()
	} else {
		response["nats"] = map[string]interface{}{"status": "not_configured"}
	}
	if s.wsManager != nil {
		response["websocket"] = s.wsManager.Stats()
	}

	s.writeJSON(w, http.StatusOK, response)
}

func (s *Server) requireAdminAccess(w http.ResponseWriter, r *http.Request) bool {
	token := extractToken(r)
	if token == "" {
		s.writeError(w, http.StatusUnauthorized, "Unauthorized", "UNAUTHORIZED")
		return false
	}
	_, role, _, isGovernment, err := parseJWTClaims(token)
	if err != nil {
		s.writeError(w, http.StatusUnauthorized, "Unauthorized", "UNAUTHORIZED")
		return false
	}

	switch strings.ToLower(role) {
	case "admin", "government":
		return true
	}
	if isGovernment {
		return true
	}

	s.writeError(w, http.StatusForbidden, "Forbidden", "FORBIDDEN")
	return false
}

func (s *Server) getRequesterID(r *http.Request) string {
	token := extractToken(r)
	if token == "" {
		return ""
	}
	userID, _, _, _, err := parseJWTClaims(token)
	if err != nil {
		return ""
	}
	return userID
}

func generateTemporaryPassword() string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return "Temp-Access-ChangeMe!"
	}
	encoded := strings.TrimRight(base32.StdEncoding.EncodeToString(buf), "=")
	if len(encoded) > 12 {
		encoded = encoded[:12]
	}
	return "Temp-" + encoded + "!"
}
