// Package services provides business logic services for the API.
package services

import (
	"github.com/asgard/pandora/internal/repositories"
)

// StreamService handles stream-related business logic.
type StreamService struct {
	streamRepo *repositories.StreamRepository
}

// NewStreamService creates a new stream service.
func NewStreamService(streamRepo *repositories.StreamRepository) *StreamService {
	return &StreamService{
		streamRepo: streamRepo,
	}
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
func (s *StreamService) CreateStreamSession(streamID, userID string) (map[string]interface{}, error) {
	// In production, this would create a WebRTC session with proper signaling
	// For now, return mock session data
	return map[string]interface{}{
		"streamId":     streamID,
		"sessionId":    "session_" + streamID,
		"iceServers": []map[string]interface{}{
			{
				"urls": []string{"stun:stun.l.google.com:19302"},
			},
		},
		"signallingUrl": "ws://localhost:8080/ws/signaling",
		"authToken":     "mock_token_" + userID,
		"expiresAt":     "2026-12-31T23:59:59Z",
	}, nil
}
