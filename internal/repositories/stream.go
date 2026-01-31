// Package repositories provides data access layer for database operations.
package repositories

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/asgard/pandora/internal/platform/db"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
)

// Stream repository errors.
var (
	ErrStreamNotFound     = errors.New("stream not found")
	ErrStreamAccessDenied = errors.New("access denied to stream")
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

// Postgres returns the underlying Postgres handle.
func (r *StreamRepository) Postgres() *db.PostgresDB {
	return r.pgDB
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

// StreamSession represents a streaming session for a user.
type StreamSession struct {
	ID           string                   `json:"id"`
	StreamID     string                   `json:"streamId"`
	UserID       string                   `json:"userId"`
	ICEServers   []map[string]interface{} `json:"iceServers"`
	SignalingURL string                   `json:"signalingUrl"`
	AuthToken    string                   `json:"authToken"`
	ExpiresAt    time.Time                `json:"expiresAt"`
	CreatedAt    time.Time                `json:"createdAt"`
}

// GetStreams retrieves streams with optional filters.
func (r *StreamRepository) GetStreams(streamType, status string, limit, offset int) ([]*Stream, int, error) {
	if r.pgDB == nil {
		return nil, 0, fmt.Errorf("postgres database not configured")
	}

	whereClause, args := buildStreamFilters(streamType, status)
	countQuery := `SELECT COUNT(*) FROM streams` + whereClause

	var total int
	if err := r.pgDB.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count streams: %w", err)
	}

	query := `
		SELECT id, title, source, source_type, source_id, location, type, status,
		       viewers, latency, resolution, bitrate, started_at, metadata,
		       geo_lat, geo_lon, geo_alt
		FROM streams` + whereClause + `
		ORDER BY started_at DESC NULLS LAST
	`

	if limit > 0 {
		args = append(args, limit)
		query += fmt.Sprintf(" LIMIT $%d", len(args))
	}
	if offset > 0 {
		args = append(args, offset)
		query += fmt.Sprintf(" OFFSET $%d", len(args))
	}

	rows, err := r.pgDB.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query streams: %w", err)
	}
	defer rows.Close()

	var streams []*Stream
	for rows.Next() {
		stream, err := scanStream(rows)
		if err != nil {
			return nil, 0, err
		}
		streams = append(streams, stream)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate streams: %w", err)
	}

	return streams, total, nil
}

// GetStream retrieves a stream by ID.
func (r *StreamRepository) GetStream(id string) (*Stream, error) {
	if r.pgDB == nil {
		return nil, fmt.Errorf("postgres database not configured")
	}

	streamID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid stream ID: %w", err)
	}

	query := `
		SELECT id, title, source, source_type, source_id, location, type, status,
		       viewers, latency, resolution, bitrate, started_at, metadata,
		       geo_lat, geo_lon, geo_alt
		FROM streams
		WHERE id = $1
	`

	row := r.pgDB.QueryRow(query, streamID)
	stream, err := scanStream(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrStreamNotFound
		}
		return nil, err
	}

	return stream, nil
}

// GetStreamStats returns aggregate stream statistics.
func (r *StreamRepository) GetStreamStats() (map[string]interface{}, error) {
	if r.pgDB == nil {
		return nil, fmt.Errorf("postgres database not configured")
	}

	var totalStreams, liveStreams, totalViewers int64

	err := r.pgDB.QueryRow(`
		SELECT COUNT(*) AS total,
		       COALESCE(SUM(CASE WHEN status = 'live' THEN 1 ELSE 0 END), 0) AS live,
		       COALESCE(SUM(viewers), 0) AS viewers
		FROM streams
	`).Scan(&totalStreams, &liveStreams, &totalViewers)

	stats := map[string]interface{}{
		"totalStreams": totalStreams,
		"liveStreams":  liveStreams,
		"totalViewers": totalViewers,
		"byCategory":   map[string]int{},
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query stream stats: %w", err)
	}

	rows, err := r.pgDB.Query(`SELECT type, COUNT(*) FROM streams GROUP BY type`)
	if err != nil {
		return nil, fmt.Errorf("failed to query stream categories: %w", err)
	}
	defer rows.Close()

	byCategory := stats["byCategory"].(map[string]int)
	for rows.Next() {
		var streamType string
		var count int
		if err := rows.Scan(&streamType, &count); err != nil {
			return nil, fmt.Errorf("failed to scan category stats: %w", err)
		}
		byCategory[streamType] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate category stats: %w", err)
	}

	return stats, nil
}

// GetFeaturedStreams returns featured streams.
func (r *StreamRepository) GetFeaturedStreams() ([]*Stream, error) {
	if r.pgDB == nil {
		return nil, fmt.Errorf("postgres database not configured")
	}

	query := `
		SELECT id, title, source, source_type, source_id, location, type, status,
		       viewers, latency, resolution, bitrate, started_at, metadata,
		       geo_lat, geo_lon, geo_alt
		FROM streams
		WHERE status = 'live'
		ORDER BY viewers DESC, started_at DESC NULLS LAST
		LIMIT 5
	`

	rows, err := r.pgDB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query featured streams: %w", err)
	}
	defer rows.Close()

	var streams []*Stream
	for rows.Next() {
		stream, err := scanStream(rows)
		if err != nil {
			return nil, err
		}
		streams = append(streams, stream)
	}

	return streams, rows.Err()
}

// SearchStreams searches streams by query.
func (r *StreamRepository) SearchStreams(query string) ([]*Stream, error) {
	if r.pgDB == nil {
		return nil, fmt.Errorf("postgres database not configured")
	}

	term := strings.TrimSpace(query)
	if term == "" {
		return []*Stream{}, nil
	}

	search := "%" + term + "%"
	rows, err := r.pgDB.Query(`
		SELECT id, title, source, source_type, source_id, location, type, status,
		       viewers, latency, resolution, bitrate, started_at, metadata,
		       geo_lat, geo_lon, geo_alt
		FROM streams
		WHERE title ILIKE $1 OR location ILIKE $1 OR source ILIKE $1
		ORDER BY viewers DESC, started_at DESC NULLS LAST
		LIMIT 100
	`, search)
	if err != nil {
		return nil, fmt.Errorf("failed to search streams: %w", err)
	}
	defer rows.Close()

	var streams []*Stream
	for rows.Next() {
		stream, err := scanStream(rows)
		if err != nil {
			return nil, err
		}
		streams = append(streams, stream)
	}

	return streams, rows.Err()
}

// CreateStreamSession stores a stream session record.
func (r *StreamRepository) CreateStreamSession(session *StreamSession) error {
	if r.pgDB == nil {
		// Allow in-memory session handling when Postgres is unavailable.
		return nil
	}

	sessionID, err := uuid.Parse(session.ID)
	if err != nil {
		return fmt.Errorf("invalid session ID: %w", err)
	}
	streamID, err := uuid.Parse(session.StreamID)
	if err != nil {
		return fmt.Errorf("invalid stream ID: %w", err)
	}
	userID, err := uuid.Parse(session.UserID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	iceServersJSON, err := json.Marshal(session.ICEServers)
	if err != nil {
		return fmt.Errorf("failed to serialize ICE servers: %w", err)
	}

	query := `
		INSERT INTO stream_sessions (
			id, stream_id, user_id, ice_servers, signaling_url, auth_token, expires_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err = r.pgDB.Exec(query,
		sessionID,
		streamID,
		userID,
		iceServersJSON,
		session.SignalingURL,
		session.AuthToken,
		session.ExpiresAt,
		session.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create stream session: %w", err)
	}

	return nil
}

// GetTelemetryCollection returns the MongoDB collection for telemetry.
func (r *StreamRepository) GetTelemetryCollection() *mongo.Collection {
	return r.mongoDB.Collection("satellite_telemetry")
}

type streamScanner interface {
	Scan(dest ...interface{}) error
}

func scanStream(row streamScanner) (*Stream, error) {
	stream := &Stream{}
	var metadataBytes []byte
	var geoLat sql.NullFloat64
	var geoLon sql.NullFloat64
	var geoAlt sql.NullFloat64
	var startedAt sql.NullTime

	err := row.Scan(
		&stream.ID,
		&stream.Title,
		&stream.Source,
		&stream.SourceType,
		&stream.SourceID,
		&stream.Location,
		&stream.Type,
		&stream.Status,
		&stream.Viewers,
		&stream.Latency,
		&stream.Resolution,
		&stream.Bitrate,
		&startedAt,
		&metadataBytes,
		&geoLat,
		&geoLon,
		&geoAlt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan stream: %w", err)
	}

	if startedAt.Valid {
		stream.StartedAt = startedAt.Time
	}

	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &stream.Metadata); err != nil {
			return nil, fmt.Errorf("failed to parse stream metadata: %w", err)
		}
	}

	if geoLat.Valid && geoLon.Valid {
		stream.GeoLocation = &GeoLocation{
			Latitude:  geoLat.Float64,
			Longitude: geoLon.Float64,
		}
		if geoAlt.Valid {
			stream.GeoLocation.Altitude = geoAlt.Float64
		}
	}

	return stream, nil
}

func buildStreamFilters(streamType, status string) (string, []interface{}) {
	clauses := []string{}
	args := []interface{}{}

	if streamType != "" {
		args = append(args, streamType)
		clauses = append(clauses, fmt.Sprintf("type = $%d", len(args)))
	}
	if status != "" {
		args = append(args, status)
		clauses = append(clauses, fmt.Sprintf("status = $%d", len(args)))
	}

	if len(clauses) == 0 {
		return "", args
	}

	return " WHERE " + strings.Join(clauses, " AND "), args
}
