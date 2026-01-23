// Package repositories provides data access layer for database operations.
package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/asgard/pandora/internal/platform/db"
	"go.mongodb.org/mongo-driver/mongo"
)

// StreamRepository handles stream database operations.
type StreamRepository struct {
	pgDB    *db.PostgresDB
	mongoDB *db.MongoDB
}

// NewStreamRepository creates a new stream repository.
func NewStreamRepository(pgDB *db.PostgresDB, mongoDB *db.MongoDB) *StreamRepository {
	return &StreamRepository{
		pgDB:    pgDB,
		mongoDB: mongoDB,
	}
}

// Stream represents a video stream.
type Stream struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Source      string                 `json:"source"`
	SourceType  string                 `json:"sourceType"`
	SourceID    string                 `json:"sourceId"`
	Location    string                 `json:"location"`
	Type        string                 `json:"type"`
	Status      string                 `json:"status"`
	Viewers     int                    `json:"viewers"`
	Latency     int                    `json:"latency"`
	Resolution  string                 `json:"resolution"`
	Bitrate     int                    `json:"bitrate"`
	StartedAt   time.Time              `json:"startedAt"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	GeoLocation *GeoLocation           `json:"geoLocation,omitempty"`
}

// GeoLocation represents geographic coordinates.
type GeoLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Altitude  float64 `json:"altitude,omitempty"`
}

// GetStreams retrieves streams with optional filters.
func (r *StreamRepository) GetStreams(streamType, status string, limit, offset int) ([]*Stream, int, error) {
	// In production, this would query a streams table in PostgreSQL
	// For now, return mock data
	streams := []*Stream{
		{
			ID:         "stream_001",
			Title:      "Earth Observation - Pacific",
			Source:     "Silenus-SAT-001",
			SourceType: "satellite",
			SourceID:   "sat_001",
			Location:   "Pacific Ocean",
			Type:       "civilian",
			Status:     "live",
			Viewers:    42,
			Latency:    250,
			Resolution: "1080p",
			Bitrate:    2000,
			StartedAt:  time.Now().Add(-2 * time.Hour),
		},
		{
			ID:         "stream_002",
			Title:      "Search & Rescue - Sector 7",
			Source:     "Hunoid-UNIT-042",
			SourceType: "hunoid",
			SourceID:   "hunoid_042",
			Location:   "Sector 7",
			Type:       "military",
			Status:     "live",
			Viewers:    8,
			Latency:    120,
			Resolution: "720p",
			Bitrate:    1500,
			StartedAt:  time.Now().Add(-30 * time.Minute),
		},
	}

	// Filter by type
	if streamType != "" {
		filtered := []*Stream{}
		for _, s := range streams {
			if s.Type == streamType {
				filtered = append(filtered, s)
			}
		}
		streams = filtered
	}

	// Filter by status
	if status != "" {
		filtered := []*Stream{}
		for _, s := range streams {
			if s.Status == status {
				filtered = append(filtered, s)
			}
		}
		streams = filtered
	}

	total := len(streams)

	// Apply pagination
	if offset > 0 && offset < len(streams) {
		streams = streams[offset:]
	}
	if limit > 0 && limit < len(streams) {
		streams = streams[:limit]
	}

	return streams, total, nil
}

// GetStream retrieves a stream by ID.
func (r *StreamRepository) GetStream(id string) (*Stream, error) {
	streams, _, err := r.GetStreams("", "", 100, 0)
	if err != nil {
		return nil, err
	}

	for _, s := range streams {
		if s.ID == id {
			return s, nil
		}
	}

	return nil, fmt.Errorf("stream not found")
}

// GetStreamStats returns aggregate stream statistics.
func (r *StreamRepository) GetStreamStats() (map[string]interface{}, error) {
	streams, _, err := r.GetStreams("", "", 1000, 0)
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"totalStreams": len(streams),
		"liveStreams":  0,
		"totalViewers": 0,
		"byCategory": map[string]int{
			"civilian":     0,
			"military":     0,
			"interstellar": 0,
		},
	}

	for _, s := range streams {
		if s.Status == "live" {
			stats["liveStreams"] = stats["liveStreams"].(int) + 1
		}
		stats["totalViewers"] = stats["totalViewers"].(int) + s.Viewers

		byCat := stats["byCategory"].(map[string]int)
		byCat[s.Type] = byCat[s.Type] + 1
	}

	return stats, nil
}

// GetFeaturedStreams returns featured streams.
func (r *StreamRepository) GetFeaturedStreams() ([]*Stream, error) {
	streams, _, err := r.GetStreams("", "live", 5, 0)
	if err != nil {
		return nil, err
	}
	return streams, nil
}

// SearchStreams searches streams by query.
func (r *StreamRepository) SearchStreams(query string) ([]*Stream, error) {
	streams, _, err := r.GetStreams("", "", 100, 0)
	if err != nil {
		return nil, err
	}

	results := []*Stream{}
	for _, s := range streams {
		if contains(s.Title, query) || contains(s.Location, query) || contains(s.Source, query) {
			results = append(results, s)
		}
	}

	return results, nil
}

// contains checks if a string contains a substring (case-insensitive).
func contains(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// GetTelemetryCollection returns the MongoDB collection for telemetry.
func (r *StreamRepository) GetTelemetryCollection() *mongo.Collection {
	return r.mongoDB.Collection("satellite_telemetry")
}
