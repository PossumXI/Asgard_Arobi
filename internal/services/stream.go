// Package services provides business logic services for the API.
package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	apiwebrtc "github.com/asgard/pandora/internal/api/webrtc"
	"github.com/asgard/pandora/internal/repositories"
	"github.com/google/uuid"
)

// ICEServerConfig represents an ICE server configuration for WebRTC.
type ICEServerConfig struct {
	URLs       []string `json:"urls"`
	Username   string   `json:"username,omitempty"`
	Credential string   `json:"credential,omitempty"`
}

// StreamSessionConfig holds configuration for stream sessions.
type StreamSessionConfig struct {
	ICEServers   []ICEServerConfig
	SignalingURL string
	SessionTTL   time.Duration
}

// DefaultStreamSessionConfig returns the default stream session configuration.
func DefaultStreamSessionConfig() StreamSessionConfig {
	config := StreamSessionConfig{
		ICEServers: []ICEServerConfig{
			{
				URLs: []string{
					"stun:stun.l.google.com:19302",
					"stun:stun1.l.google.com:19302",
				},
			},
		},
		SignalingURL: getEnvOrDefault("SIGNALING_URL", "ws://localhost:8080/ws/signaling"),
		SessionTTL:   24 * time.Hour,
	}

	// Add TURN server if configured
	if turnURL := os.Getenv("TURN_SERVER"); turnURL != "" {
		turnUsername := os.Getenv("TURN_USERNAME")
		turnPassword := os.Getenv("TURN_PASSWORD")
		config.ICEServers = append(config.ICEServers, ICEServerConfig{
			URLs:       []string{turnURL},
			Username:   turnUsername,
			Credential: turnPassword,
		})
	}

	return config
}

func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// StreamService handles stream-related business logic.
type StreamService struct {
	streamRepo    *repositories.StreamRepository
	chatRepo      *repositories.StreamChatRepository
	sessionConfig StreamSessionConfig
	sfu           *apiwebrtc.SFU
	sessions      map[string]sessionRecord
	mu            sync.RWMutex
}

// NewStreamService creates a new stream service with default configuration.
func NewStreamService(streamRepo *repositories.StreamRepository) *StreamService {
	var chatRepo *repositories.StreamChatRepository
	if streamRepo != nil && streamRepo.Postgres() != nil {
		chatRepo = repositories.NewStreamChatRepository(streamRepo.Postgres())
	}
	return &StreamService{
		streamRepo:    streamRepo,
		chatRepo:      chatRepo,
		sessionConfig: DefaultStreamSessionConfig(),
		sessions:      make(map[string]sessionRecord),
	}
}

// NewStreamServiceWithConfig creates a new stream service with custom configuration.
func NewStreamServiceWithConfig(streamRepo *repositories.StreamRepository, config StreamSessionConfig) *StreamService {
	var chatRepo *repositories.StreamChatRepository
	if streamRepo != nil && streamRepo.Postgres() != nil {
		chatRepo = repositories.NewStreamChatRepository(streamRepo.Postgres())
	}
	return &StreamService{
		streamRepo:    streamRepo,
		chatRepo:      chatRepo,
		sessionConfig: config,
		sessions:      make(map[string]sessionRecord),
	}
}

// SetSessionConfig updates the session configuration (useful for runtime config updates).
func (s *StreamService) SetSessionConfig(config StreamSessionConfig) {
	s.sessionConfig = config
}

// SetSFU attaches a WebRTC SFU instance for session management.
func (s *StreamService) SetSFU(sfu *apiwebrtc.SFU) {
	s.sfu = sfu
}

type sessionRecord struct {
	streamID  string
	userID    string
	authToken string
	expiresAt time.Time
}

// GetStreams retrieves streams with optional filters.
func (s *StreamService) GetStreams(streamType, status string, limit, offset int) ([]*repositories.Stream, int, error) {
	return s.streamRepo.GetStreams(streamType, status, limit, offset)
}

// GetStream retrieves a stream by ID.
func (s *StreamService) GetStream(id string) (*repositories.Stream, error) {
	return s.streamRepo.GetStream(id)
}

// GetStreamStats returns aggregate stream statistics.
func (s *StreamService) GetStreamStats() (map[string]interface{}, error) {
	return s.streamRepo.GetStreamStats()
}

// GetFeaturedStreams returns featured streams.
func (s *StreamService) GetFeaturedStreams() ([]*repositories.Stream, error) {
	return s.streamRepo.GetFeaturedStreams()
}

// SearchStreams searches streams by query.
func (s *StreamService) SearchStreams(query string) ([]*repositories.Stream, error) {
	return s.streamRepo.SearchStreams(query)
}

// CreateStreamSession creates a WebRTC session for a stream.
// Returns session details including ICE servers configuration for the client.
func (s *StreamService) CreateStreamSession(streamID, userID string) (map[string]interface{}, error) {
	if _, err := s.streamRepo.GetStream(streamID); err != nil {
		return nil, fmt.Errorf("stream not found: %w", err)
	}

	// Generate unique session ID
	sessionID := uuid.New().String()

	// Convert ICE servers to the response format
	iceServers := make([]map[string]interface{}, 0, len(s.sessionConfig.ICEServers))
	for _, server := range s.sessionConfig.ICEServers {
		serverMap := map[string]interface{}{
			"urls": server.URLs,
		}
		if server.Username != "" {
			serverMap["username"] = server.Username
		}
		if server.Credential != "" {
			serverMap["credential"] = server.Credential
		}
		iceServers = append(iceServers, serverMap)
	}

	// Calculate session expiration
	expiresAt := time.Now().Add(s.sessionConfig.SessionTTL).UTC()

	// Generate authentication token for the session
	authToken, err := generateSessionToken()
	if err != nil {
		return nil, err
	}

	session := &repositories.StreamSession{
		ID:           sessionID,
		StreamID:     streamID,
		UserID:       userID,
		ICEServers:   iceServers,
		SignalingURL: s.sessionConfig.SignalingURL,
		AuthToken:    authToken,
		ExpiresAt:    expiresAt,
		CreatedAt:    time.Now().UTC(),
	}

	if err := s.streamRepo.CreateStreamSession(session); err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.sessions[sessionID] = sessionRecord{
		streamID:  streamID,
		userID:    userID,
		authToken: authToken,
		expiresAt: expiresAt,
	}
	s.mu.Unlock()

	if s.sfu != nil {
		if _, exists := s.sfu.GetSession(sessionID); !exists {
			s.sfu.CreateSession(sessionID, streamID)
		}
	}

	return map[string]interface{}{
		"streamId":      streamID,
		"sessionId":     sessionID,
		"iceServers":    iceServers,
		"signalingUrl":  s.sessionConfig.SignalingURL,
		"authToken":     authToken,
		"expiresAt":     expiresAt.Format(time.RFC3339),
	}, nil
}

// ValidateSession verifies a session token and returns session details.
func (s *StreamService) ValidateSession(sessionID, token string) (string, string, bool) {
	s.mu.RLock()
	record, ok := s.sessions[sessionID]
	s.mu.RUnlock()
	if !ok {
		return "", "", false
	}
	if time.Now().UTC().After(record.expiresAt) {
		s.mu.Lock()
		delete(s.sessions, sessionID)
		s.mu.Unlock()
		return "", "", false
	}
	if record.authToken != token {
		return "", "", false
	}
	return record.streamID, record.userID, true
}

// generateSessionToken creates an authentication token for a stream session.
func generateSessionToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("failed to generate session token: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

// TierOrder defines the subscription tier hierarchy for access control.
var tierOrder = map[string]int{
	"free":      0,
	"observer":  1,
	"supporter": 2,
	"commander": 3,
}

// tierAtLeast checks if userTier is at least the required tier level.
func tierAtLeast(userTier, requiredTier string) bool {
	userLevel, userOk := tierOrder[userTier]
	requiredLevel, reqOk := tierOrder[requiredTier]

	if !userOk {
		userLevel = 0
	}
	if !reqOk {
		requiredLevel = 0
	}

	return userLevel >= requiredLevel
}

// GetAllowedStreamTypes returns stream types accessible to a given tier.
func GetAllowedStreamTypes(userTier string) []string {
	allowed := []string{}

	// Free tier gets no streams (must be at least observer)
	if tierAtLeast(userTier, "observer") {
		allowed = append(allowed, "civilian")
	}
	if tierAtLeast(userTier, "supporter") {
		allowed = append(allowed, "military")
	}
	if tierAtLeast(userTier, "commander") {
		allowed = append(allowed, "interstellar")
	}

	return allowed
}

// CanAccessStreamType checks if a user with the given tier can access a stream type.
func CanAccessStreamType(userTier, streamType string) bool {
	switch streamType {
	case "civilian":
		return tierAtLeast(userTier, "observer")
	case "military":
		return tierAtLeast(userTier, "supporter")
	case "interstellar":
		return tierAtLeast(userTier, "commander")
	default:
		// Unknown stream types are public by default
		return true
	}
}

// GetStreamsForUser retrieves streams filtered by user's subscription tier.
// ctx is used for potential future enhancements like cancellation and tracing.
func (s *StreamService) GetStreamsForUser(ctx context.Context, userTier string, streamType, status string, limit, offset int) ([]*repositories.Stream, int, error) {
	// Get all streams matching the type and status filters
	streams, total, err := s.streamRepo.GetStreams(streamType, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	// Filter streams based on user's tier
	allowedStreams := make([]*repositories.Stream, 0, len(streams))
	for _, stream := range streams {
		if CanAccessStreamType(userTier, stream.Type) {
			allowedStreams = append(allowedStreams, stream)
		}
	}

	// Recalculate total for filtered results
	filteredTotal := 0
	if streamType != "" {
		// If a specific type was requested, check if user can access it
		if CanAccessStreamType(userTier, streamType) {
			filteredTotal = total
		}
	} else {
		// For unfiltered requests, we need to count accessible streams
		// In a production system, this would be done in the repository with proper SQL
		filteredTotal = len(allowedStreams)
	}

	return allowedStreams, filteredTotal, nil
}

// GetStreamForUser retrieves a specific stream if the user has access.
func (s *StreamService) GetStreamForUser(ctx context.Context, userTier, streamID string) (*repositories.Stream, error) {
	stream, err := s.streamRepo.GetStream(streamID)
	if err != nil {
		return nil, err
	}

	// Check if user can access this stream type
	if !CanAccessStreamType(userTier, stream.Type) {
		return nil, repositories.ErrStreamAccessDenied
	}

	return stream, nil
}

// ErrChatUnavailable indicates chat storage is not configured.
var ErrChatUnavailable = errors.New("chat storage not configured")

// ListChatMessages returns chat messages for a stream.
func (s *StreamService) ListChatMessages(ctx context.Context, streamID string, limit int) ([]*repositories.StreamChatMessage, error) {
	if s.chatRepo == nil {
		return nil, ErrChatUnavailable
	}
	return s.chatRepo.List(ctx, streamID, limit)
}

// AddChatMessage persists a chat message.
func (s *StreamService) AddChatMessage(ctx context.Context, streamID, userID, username, message string) (*repositories.StreamChatMessage, error) {
	if s.chatRepo == nil {
		return nil, ErrChatUnavailable
	}
	return s.chatRepo.Add(ctx, streamID, userID, username, message)
}
