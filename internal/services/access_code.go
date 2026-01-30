// Package services provides business logic services for the API.
package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/asgard/pandora/internal/platform/db"
	"github.com/asgard/pandora/internal/repositories"
	"github.com/google/uuid"
)

var (
	ErrAccessCodeRequired     = errors.New("access code required")
	ErrAccessCodeInvalid      = errors.New("access code invalid")
	ErrAccessCodeExpired      = errors.New("access code expired")
	ErrAccessCodeRevoked      = errors.New("access code revoked")
	ErrAccessCodeScopeMismatch = errors.New("access code scope mismatch")
	ErrAccessCodeUsageExceeded = errors.New("access code usage exceeded")
)

// AccessCodeService handles issuance and validation of access codes.
type AccessCodeService struct {
	repo        *repositories.AccessCodeRepository
	userRepo    *repositories.UserRepository
	emailService *EmailService
}

// AccessCodeIssueRequest contains issuance parameters.
type AccessCodeIssueRequest struct {
	UserID                string
	CreatedBy             string
	ClearanceLevel         string
	Scope                 string
	ExpiresAt             time.Time
	MaxUses               *int
	RotationIntervalHours int
	Note                  string
}

// AccessCodeIssueResult contains the generated code and record.
type AccessCodeIssueResult struct {
	Code      string
	CodeLast4 string
	Record    *db.AccessCode
}

// AccessCodeRotationResult contains a rotated code outcome.
type AccessCodeRotationResult struct {
	UserID string
	Email  string
	Code   string
}

// NewAccessCodeService creates a new access code service.
func NewAccessCodeService(
	repo *repositories.AccessCodeRepository,
	userRepo *repositories.UserRepository,
	emailService *EmailService,
) *AccessCodeService {
	return &AccessCodeService{
		repo:        repo,
		userRepo:    userRepo,
		emailService: emailService,
	}
}

// RequiresAccessCode returns true if a user has an active access code.
func (s *AccessCodeService) RequiresAccessCode(ctx context.Context, userID string) (bool, error) {
	if s.repo == nil {
		return false, nil
	}
	_, err := s.repo.GetActiveForUser(ctx, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// List returns access codes for admin views.
func (s *AccessCodeService) List(ctx context.Context) ([]map[string]interface{}, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("access code repository unavailable")
	}
	return s.repo.List(ctx, 200)
}

// GetUserByEmail returns a user by email.
func (s *AccessCodeService) GetUserByEmail(ctx context.Context, email string) (*db.User, error) {
	if s.userRepo == nil {
		return nil, fmt.Errorf("user repository unavailable")
	}
	return s.userRepo.GetByEmail(email)
}

// Revoke revokes a specific access code.
func (s *AccessCodeService) Revoke(ctx context.Context, codeID string) error {
	if s.repo == nil {
		return fmt.Errorf("access code repository unavailable")
	}
	return s.repo.Revoke(ctx, codeID)
}

// IssueForUser issues a new access code for a user.
func (s *AccessCodeService) IssueForUser(ctx context.Context, req AccessCodeIssueRequest) (*AccessCodeIssueResult, error) {
	if req.UserID == "" {
		return nil, fmt.Errorf("user ID required")
	}
	if req.ClearanceLevel == "" {
		req.ClearanceLevel = "civilian"
	}
	if req.Scope == "" {
		req.Scope = "portal"
	}
	if req.RotationIntervalHours <= 0 {
		req.RotationIntervalHours = getAccessCodeRotationHours()
	}
	if req.ExpiresAt.IsZero() {
		req.ExpiresAt = time.Now().UTC().Add(time.Duration(req.RotationIntervalHours) * time.Hour)
	}

	code, hash, last4, err := generateAccessCode()
	if err != nil {
		return nil, err
	}

	codeRecord := &db.AccessCode{
		ID:                    uuid.New(),
		CodeHash:              hash,
		CodeLast4:             last4,
		UserID:                stringToNull(req.UserID),
		CreatedBy:             stringToNull(req.CreatedBy),
		ClearanceLevel:        strings.ToLower(req.ClearanceLevel),
		Scope:                 strings.ToLower(req.Scope),
		IssuedAt:              time.Now().UTC(),
		ExpiresAt:             req.ExpiresAt,
		RotationIntervalHours: req.RotationIntervalHours,
		NextRotationAt:        time.Now().UTC().Add(time.Duration(req.RotationIntervalHours) * time.Hour),
		Note:                  stringToNull(req.Note),
	}
	if req.MaxUses != nil && *req.MaxUses > 0 {
		codeRecord.MaxUses = int32ToNull(*req.MaxUses)
	}

	if err := s.repo.Create(ctx, codeRecord); err != nil {
		return nil, err
	}

	s.sendAccessCodeEmail(ctx, req.UserID, code, codeRecord.ExpiresAt, codeRecord.Scope, codeRecord.ClearanceLevel)

	return &AccessCodeIssueResult{
		Code:      code,
		CodeLast4: last4,
		Record:    codeRecord,
	}, nil
}

// ValidateForUser validates a code against a user and scope.
func (s *AccessCodeService) ValidateForUser(ctx context.Context, code, userID, scope string) (*db.AccessCode, error) {
	if code == "" {
		return nil, ErrAccessCodeRequired
	}
	record, err := s.Validate(ctx, code, scope)
	if err != nil {
		return nil, err
	}
	if !record.UserID.Valid || !strings.EqualFold(record.UserID.String, userID) {
		return nil, ErrAccessCodeInvalid
	}
	if err := s.repo.MarkUsed(ctx, record.ID.String()); err != nil {
		return nil, err
	}
	return record, nil
}

// Validate validates a code and scope.
func (s *AccessCodeService) Validate(ctx context.Context, code, scope string) (*db.AccessCode, error) {
	if code == "" {
		return nil, ErrAccessCodeRequired
	}
	hash := hashAccessCode(code)
	record, err := s.repo.GetByHash(ctx, hash)
	if err != nil {
		return nil, ErrAccessCodeInvalid
	}
	if record.RevokedAt.Valid {
		return nil, ErrAccessCodeRevoked
	}
	if time.Now().UTC().After(record.ExpiresAt) {
		return nil, ErrAccessCodeExpired
	}
	if record.MaxUses.Valid && record.UsageCount >= int(record.MaxUses.Int32) {
		return nil, ErrAccessCodeUsageExceeded
	}
	if scope != "" && record.Scope != "all" && !strings.EqualFold(record.Scope, scope) {
		return nil, ErrAccessCodeScopeMismatch
	}
	return record, nil
}

// RotateForUser revokes existing code and issues a new one.
func (s *AccessCodeService) RotateForUser(ctx context.Context, userID, createdBy string) (*AccessCodeIssueResult, error) {
	existing, _ := s.repo.GetActiveForUser(ctx, userID)
	clearance := "government"
	scope := "all"
	rotationHours := getAccessCodeRotationHours()
	if existing != nil {
		clearance = existing.ClearanceLevel
		scope = existing.Scope
		if existing.RotationIntervalHours > 0 {
			rotationHours = existing.RotationIntervalHours
		}
	}
	if err := s.repo.RevokeByUser(ctx, userID); err != nil {
		return nil, err
	}
	return s.IssueForUser(ctx, AccessCodeIssueRequest{
		UserID:                userID,
		CreatedBy:             createdBy,
		ClearanceLevel:         clearance,
		Scope:                 scope,
		RotationIntervalHours: rotationHours,
	})
}

// RotateDue rotates any codes that are due for rotation.
func (s *AccessCodeService) RotateDue(ctx context.Context) ([]AccessCodeRotationResult, error) {
	due, err := s.repo.ListRotationDue(ctx)
	if err != nil {
		return nil, err
	}
	results := []AccessCodeRotationResult{}
	for _, record := range due {
		if !record.UserID.Valid {
			continue
		}
		_ = s.repo.Revoke(ctx, record.ID.String())
		result, err := s.IssueForUser(ctx, AccessCodeIssueRequest{
			UserID:                record.UserID.String,
			CreatedBy:             record.CreatedBy.String,
			ClearanceLevel:         record.ClearanceLevel,
			Scope:                 record.Scope,
			RotationIntervalHours: record.RotationIntervalHours,
		})
		if err != nil {
			continue
		}
		user, userErr := s.userRepo.GetByID(record.UserID.String)
		if userErr == nil {
			results = append(results, AccessCodeRotationResult{
				UserID: record.UserID.String,
				Email:  user.Email,
				Code:   result.Code,
			})
		}
	}
	return results, nil
}

// RotateAllActive rotates access codes for all users with active codes.
func (s *AccessCodeService) RotateAllActive(ctx context.Context, createdBy string) ([]AccessCodeRotationResult, error) {
	userIDs, err := s.repo.ListActiveUserIDs(ctx)
	if err != nil {
		return nil, err
	}
	results := []AccessCodeRotationResult{}
	for _, userID := range userIDs {
		result, err := s.RotateForUser(ctx, userID, createdBy)
		if err != nil {
			continue
		}
		user, userErr := s.userRepo.GetByID(userID)
		if userErr != nil {
			continue
		}
		results = append(results, AccessCodeRotationResult{
			UserID: userID,
			Email:  user.Email,
			Code:   result.Code,
		})
	}
	return results, nil
}

// StartRotationLoop periodically rotates access codes and emails new codes.
func (s *AccessCodeService) StartRotationLoop(ctx context.Context) {
	interval := getAccessCodeRotationCheckInterval()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rotations, err := s.RotateDue(context.Background())
			if err != nil {
				log.Printf("[AccessCode] rotation failed: %v", err)
				continue
			}
			if len(rotations) > 0 {
				log.Printf("[AccessCode] rotated %d access codes", len(rotations))
			}
		}
	}
}

func (s *AccessCodeService) sendAccessCodeEmail(ctx context.Context, userID, code string, expiresAt time.Time, scope, clearance string) {
	if s.emailService == nil || s.userRepo == nil || userID == "" {
		return
	}
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return
	}
	if user.Email == "" {
		return
	}
	if err := s.emailService.SendAccessCodeEmail(user.Email, code, expiresAt, scope, clearance); err != nil {
		log.Printf("[AccessCode] email send failed for %s: %v", user.Email, err)
	}
}

func generateAccessCode() (string, string, string, error) {
	entropy := make([]byte, 16)
	if _, err := rand.Read(entropy); err != nil {
		return "", "", "", err
	}
	encoded := strings.TrimRight(base32.StdEncoding.EncodeToString(entropy), "=")
	encoded = strings.ToUpper(encoded)
	code := fmt.Sprintf("AG-%s-%s", encoded[:8], encoded[8:16])
	hash := hashAccessCode(code)
	last4 := code[len(code)-4:]
	return code, hash, last4, nil
}

func hashAccessCode(code string) string {
	normalized := strings.TrimSpace(strings.ToUpper(code))
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

func getAccessCodeRotationHours() int {
	if val := strings.TrimSpace(os.Getenv("ACCESS_CODE_ROTATION_HOURS")); val != "" {
		if hours, err := strconv.Atoi(val); err == nil && hours > 0 {
			return hours
		}
	}
	return 24
}

func getAccessCodeRotationCheckInterval() time.Duration {
	if val := strings.TrimSpace(os.Getenv("ACCESS_CODE_ROTATION_CHECK_MINUTES")); val != "" {
		if minutes, err := strconv.Atoi(val); err == nil && minutes > 0 {
			return time.Duration(minutes) * time.Minute
		}
	}
	return 15 * time.Minute
}

func stringToNull(value string) sql.NullString {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: trimmed, Valid: true}
}

func int32ToNull(value int) sql.NullInt32 {
	if value <= 0 {
		return sql.NullInt32{}
	}
	return sql.NullInt32{Int32: int32(value), Valid: true}
}
