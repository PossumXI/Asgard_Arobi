package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/asgard/pandora/internal/platform/db"
	"github.com/asgard/pandora/internal/services"
)

type accessCodeValidateRequest struct {
	Code  string `json:"code"`
	Scope string `json:"scope,omitempty"`
}

type accessCodeValidateResponse struct {
	Valid          bool   `json:"valid"`
	UserID         string `json:"userId,omitempty"`
	ClearanceLevel string `json:"clearanceLevel,omitempty"`
	Scope          string `json:"scope,omitempty"`
}

func (s *Server) handleAccessCodeValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}
	if s.accessCodeService == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Access code service unavailable", "SERVICE_UNAVAILABLE")
		return
	}

	var req accessCodeValidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST")
		return
	}
	req.Scope = strings.TrimSpace(req.Scope)
	req.Code = strings.TrimSpace(req.Code)

	record, err := s.accessCodeService.Validate(r.Context(), req.Code, req.Scope)
	if err != nil {
		s.writeJSON(w, http.StatusOK, accessCodeValidateResponse{Valid: false})
		return
	}

	userID := ""
	if record.UserID.Valid {
		userID = record.UserID.String
	}
	s.writeJSON(w, http.StatusOK, accessCodeValidateResponse{
		Valid:          true,
		UserID:         userID,
		ClearanceLevel: record.ClearanceLevel,
		Scope:          record.Scope,
	})
}

type adminAccessCodeRequest struct {
	UserID                string `json:"userId,omitempty"`
	Email                 string `json:"email,omitempty"`
	ClearanceLevel         string `json:"clearanceLevel,omitempty"`
	Scope                 string `json:"scope,omitempty"`
	ExpiresInHours        int    `json:"expiresInHours,omitempty"`
	MaxUses               *int   `json:"maxUses,omitempty"`
	RotationIntervalHours int    `json:"rotationIntervalHours,omitempty"`
	Note                  string `json:"note,omitempty"`
}

func (s *Server) handleAdminAccessCodes(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminAccess(w, r) {
		return
	}
	if s.accessCodeService == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Access code service unavailable", "SERVICE_UNAVAILABLE")
		return
	}

	switch r.Method {
	case http.MethodGet:
		records, err := s.accessCodeService.List(r.Context())
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "Failed to load access codes", "DB_ERROR")
			return
		}
		s.writeJSON(w, http.StatusOK, records)
		return
	case http.MethodPost:
		var req adminAccessCodeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.writeError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST")
			return
		}
		userID := strings.TrimSpace(req.UserID)
		if userID == "" && strings.TrimSpace(req.Email) != "" {
			user, err := s.accessCodeService.GetUserByEmail(r.Context(), strings.TrimSpace(req.Email))
			if err != nil {
				s.writeError(w, http.StatusNotFound, "User not found", "USER_NOT_FOUND")
				return
			}
			userID = user.ID.String()
		}
		if userID == "" {
			s.writeError(w, http.StatusBadRequest, "userId or email required", "INVALID_REQUEST")
			return
		}

		expiresAt := time.Time{}
		if req.ExpiresInHours > 0 {
			expiresAt = time.Now().UTC().Add(time.Duration(req.ExpiresInHours) * time.Hour)
		}

		result, err := s.accessCodeService.IssueForUser(r.Context(), services.AccessCodeIssueRequest{
			UserID:                userID,
			CreatedBy:             s.getRequesterID(r),
			ClearanceLevel:         req.ClearanceLevel,
			Scope:                 req.Scope,
			ExpiresAt:             expiresAt,
			MaxUses:               req.MaxUses,
			RotationIntervalHours: req.RotationIntervalHours,
			Note:                  req.Note,
		})
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "Failed to issue access code", "ISSUE_FAILED")
			return
		}
		s.writeJSON(w, http.StatusOK, map[string]interface{}{
			"code":      result.Code,
			"codeLast4": result.CodeLast4,
			"record":    formatAccessCodeRecord(result.Record),
		})
		return
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}
}

type adminAccessCodeRotateRequest struct {
	UserID    string `json:"userId,omitempty"`
	Email     string `json:"email,omitempty"`
	RotateAll bool   `json:"rotateAll,omitempty"`
}

func (s *Server) handleAdminAccessCodesRotate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}
	if !s.requireAdminAccess(w, r) {
		return
	}
	if s.accessCodeService == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Access code service unavailable", "SERVICE_UNAVAILABLE")
		return
	}

	var req adminAccessCodeRotateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST")
		return
	}

	if req.RotateAll {
		rotations, err := s.accessCodeService.RotateAllActive(r.Context(), s.getRequesterID(r))
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "Failed to rotate access codes", "ROTATE_FAILED")
			return
		}
		s.writeJSON(w, http.StatusOK, rotations)
		return
	}

	userID := strings.TrimSpace(req.UserID)
	if userID == "" && strings.TrimSpace(req.Email) != "" {
		user, err := s.accessCodeService.GetUserByEmail(r.Context(), strings.TrimSpace(req.Email))
		if err != nil {
			s.writeError(w, http.StatusNotFound, "User not found", "USER_NOT_FOUND")
			return
		}
		userID = user.ID.String()
	}
	if userID == "" {
		s.writeError(w, http.StatusBadRequest, "userId or email required", "INVALID_REQUEST")
		return
	}

	result, err := s.accessCodeService.RotateForUser(r.Context(), userID, s.getRequesterID(r))
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to rotate access code", "ROTATE_FAILED")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"code":      result.Code,
		"codeLast4": result.CodeLast4,
		"record":    formatAccessCodeRecord(result.Record),
	})
}

func (s *Server) handleAdminAccessCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}
	if !s.requireAdminAccess(w, r) {
		return
	}
	if s.accessCodeService == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Access code service unavailable", "SERVICE_UNAVAILABLE")
		return
	}

	codeID := strings.TrimPrefix(r.URL.Path, "/api/admin/access-codes/")
	codeID = strings.Trim(codeID, "/")
	if codeID == "" {
		s.writeError(w, http.StatusBadRequest, "Access code ID required", "INVALID_REQUEST")
		return
	}

	if err := s.accessCodeService.Revoke(r.Context(), codeID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to revoke access code", "REVOKE_FAILED")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

func formatAccessCodeRecord(record *db.AccessCode) map[string]interface{} {
	if record == nil {
		return map[string]interface{}{}
	}
	response := map[string]interface{}{
		"id":                    record.ID.String(),
		"codeLast4":             record.CodeLast4,
		"clearanceLevel":        record.ClearanceLevel,
		"scope":                 record.Scope,
		"issuedAt":              record.IssuedAt.UTC().Format(time.RFC3339),
		"expiresAt":             record.ExpiresAt.UTC().Format(time.RFC3339),
		"usageCount":            record.UsageCount,
		"rotationIntervalHours": record.RotationIntervalHours,
		"nextRotationAt":        record.NextRotationAt.UTC().Format(time.RFC3339),
	}
	if record.UserID.Valid {
		response["userId"] = record.UserID.String
	}
	if record.CreatedBy.Valid {
		response["createdBy"] = record.CreatedBy.String
	}
	if record.MaxUses.Valid {
		response["maxUses"] = record.MaxUses.Int32
	}
	if record.RevokedAt.Valid {
		response["revokedAt"] = record.RevokedAt.Time.UTC().Format(time.RFC3339)
	}
	if record.LastUsedAt.Valid {
		response["lastUsedAt"] = record.LastUsedAt.Time.UTC().Format(time.RFC3339)
	}
	if record.Note.Valid {
		response["note"] = record.Note.String
	}
	return response
}
