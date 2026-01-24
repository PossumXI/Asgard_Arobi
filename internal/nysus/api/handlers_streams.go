package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/asgard/pandora/internal/platform/db"
	"github.com/asgard/pandora/internal/platform/realtime"
	"github.com/asgard/pandora/internal/repositories"
	"github.com/google/uuid"
)

// StreamResponse represents a video stream.
type StreamResponse struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Source      string  `json:"source"`
	SourceType  string  `json:"sourceType"`
	SourceID    string  `json:"sourceId"`
	Location    string  `json:"location"`
	GeoLocation *GeoLoc `json:"geoLocation,omitempty"`
	Type        string  `json:"type"`   // civilian, military, interstellar
	Status      string  `json:"status"` // live, delayed, offline
	Viewers     int     `json:"viewers"`
	Latency     int     `json:"latency"` // ms
	Thumbnail   string  `json:"thumbnail,omitempty"`
	Description string  `json:"description,omitempty"`
	PlaybackURL string  `json:"playbackUrl,omitempty"`
	Resolution  string  `json:"resolution"`
	Bitrate     int     `json:"bitrate"`
	StartedAt   string  `json:"startedAt"`
}

// StreamStats represents streaming statistics.
type StreamStats struct {
	TotalStreams int            `json:"totalStreams"`
	LiveStreams  int            `json:"liveStreams"`
	TotalViewers int            `json:"totalViewers"`
	ByCategory   map[string]int `json:"byCategory"`
}

type chatRequest struct {
	Message  string `json:"message"`
	Username string `json:"username"`
}

// queryStreams retrieves streams from the database
func (s *Server) queryStreams(ctx context.Context, streamType string, limit int) ([]StreamResponse, error) {
	if s.pgDB == nil {
		return nil, sql.ErrConnDone
	}

	query := `
		SELECT st.id, st.title, st.source, st.source_type, st.source_id,
			   st.location, st.latitude, st.longitude, st.stream_type,
			   st.status, st.viewers, st.latency_ms, st.description,
			   st.resolution, st.bitrate, st.started_at, st.playback_url
		FROM streams st
		WHERE st.status != 'offline'
	`
	args := []interface{}{}
	argNum := 1

	if streamType != "" {
		query += " AND st.stream_type = $" + strconv.Itoa(argNum)
		args = append(args, streamType)
		argNum++
	}

	query += " ORDER BY st.viewers DESC LIMIT $" + strconv.Itoa(argNum)
	args = append(args, limit)

	rows, err := s.pgDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	streams := []StreamResponse{}
	for rows.Next() {
		var stream StreamResponse
		var lat, lon sql.NullFloat64
		var desc sql.NullString
		var startedAt time.Time
		var playbackURL sql.NullString

		err := rows.Scan(
			&stream.ID, &stream.Title, &stream.Source, &stream.SourceType, &stream.SourceID,
			&stream.Location, &lat, &lon, &stream.Type,
			&stream.Status, &stream.Viewers, &stream.Latency, &desc,
			&stream.Resolution, &stream.Bitrate, &startedAt, &playbackURL,
		)
		if err != nil {
			continue
		}

		if lat.Valid && lon.Valid {
			stream.GeoLocation = &GeoLoc{Latitude: lat.Float64, Longitude: lon.Float64}
		}
		if desc.Valid {
			stream.Description = desc.String
		}
		if playbackURL.Valid {
			stream.PlaybackURL = playbackURL.String
		}
		stream.StartedAt = startedAt.Format(time.RFC3339)

		streams = append(streams, stream)
	}

	return streams, nil
}

// handleStreams handles GET /api/streams
func (s *Server) handleStreams(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	ctx := r.Context()

	if s.pgDB == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Database not available", "DB_UNAVAILABLE")
		return
	}

	streamType := r.URL.Query().Get("type")
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	streams, err := s.queryStreams(ctx, streamType, limit)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Database query failed", "DB_ERROR")
		return
	}

	// Get total count
	var total int
	countQuery := "SELECT COUNT(*) FROM streams WHERE status != 'offline'"
	if streamType != "" {
		countQuery += " AND stream_type = $1"
		s.pgDB.QueryRowContext(ctx, countQuery, streamType).Scan(&total)
	} else {
		s.pgDB.QueryRowContext(ctx, countQuery).Scan(&total)
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"streams": streams,
		"total":   total,
	})
}

// handleStreamRoutes handles /api/streams/{id}, /api/streams/{id}/session, /api/streams/{id}/chat
func (s *Server) handleStreamRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/streams/")
	path = strings.Trim(path, "/")
	if path == "" {
		s.writeError(w, http.StatusNotFound, "Stream not found", "STREAM_NOT_FOUND")
		return
	}

	parts := strings.Split(path, "/")
	streamID := parts[0]
	if streamID == "" {
		s.writeError(w, http.StatusNotFound, "Stream not found", "STREAM_NOT_FOUND")
		return
	}

	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
			return
		}
		s.handleStreamByID(w, r, streamID)
		return
	}

	if len(parts) == 2 {
		switch parts[1] {
		case "session":
			s.handleStreamSession(w, r, streamID)
			return
		case "chat":
			s.handleStreamChat(w, r, streamID)
			return
		default:
			s.writeError(w, http.StatusNotFound, "Not found", "NOT_FOUND")
			return
		}
	}

	s.writeError(w, http.StatusNotFound, "Not found", "NOT_FOUND")
}

func (s *Server) handleStreamByID(w http.ResponseWriter, r *http.Request, streamID string) {
	if s.pgDB == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Database not available", "DB_UNAVAILABLE")
		return
	}

	ctx := r.Context()
	query := `
		SELECT st.id, st.title, st.source, st.source_type, st.source_id,
		       st.location, st.latitude, st.longitude, st.stream_type,
		       st.status, st.viewers, st.latency_ms, st.description,
		       st.resolution, st.bitrate, st.started_at, st.playback_url
		FROM streams st
		WHERE st.id = $1
	`

	var stream StreamResponse
	var lat, lon sql.NullFloat64
	var desc sql.NullString
	var startedAt time.Time
	var playbackURL sql.NullString

	err := s.pgDB.QueryRowContext(ctx, query, streamID).Scan(
		&stream.ID, &stream.Title, &stream.Source, &stream.SourceType, &stream.SourceID,
		&stream.Location, &lat, &lon, &stream.Type,
		&stream.Status, &stream.Viewers, &stream.Latency, &desc,
		&stream.Resolution, &stream.Bitrate, &startedAt, &playbackURL,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			s.writeError(w, http.StatusNotFound, "Stream not found", "STREAM_NOT_FOUND")
			return
		}
		s.writeError(w, http.StatusInternalServerError, "Database query failed", "DB_ERROR")
		return
	}

	if lat.Valid && lon.Valid {
		stream.GeoLocation = &GeoLoc{Latitude: lat.Float64, Longitude: lon.Float64}
	}
	if desc.Valid {
		stream.Description = desc.String
	}
	if playbackURL.Valid {
		stream.PlaybackURL = playbackURL.String
	}
	stream.StartedAt = startedAt.Format(time.RFC3339)

	s.writeJSON(w, http.StatusOK, stream)
}

func (s *Server) handleStreamSession(w http.ResponseWriter, r *http.Request, streamID string) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}
	if s.streamService == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Stream service unavailable", "SERVICE_UNAVAILABLE")
		return
	}

	userID := ""
	isAnonymous := true
	if token := extractToken(r); token != "" {
		if uid, _, _, _, err := parseJWTClaims(token); err == nil {
			userID = uid
			isAnonymous = false
		}
	}
	if userID == "" {
		userID = uuid.New().String()
	}
	if isAnonymous {
		if err := s.ensureAnonymousUser(r.Context(), userID); err != nil {
			s.writeError(w, http.StatusInternalServerError, "Failed to initialize viewer", "VIEWER_INIT_FAILED")
			return
		}
	}

	session, err := s.streamService.CreateStreamSession(streamID, userID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error(), "SESSION_CREATE_FAILED")
		return
	}

	wsScheme := "ws"
	if r.TLS != nil {
		wsScheme = "wss"
	}
	session["signalingUrl"] = fmt.Sprintf("%s://%s/ws/signaling", wsScheme, r.Host)

	s.writeJSON(w, http.StatusOK, session)
}

func (s *Server) ensureAnonymousUser(ctx context.Context, userID string) error {
	if s.pgDB == nil {
		return nil
	}
	repo := repositories.NewUserRepository(s.pgDB)
	if _, err := repo.GetByID(userID); err == nil {
		return nil
	} else if !strings.Contains(err.Error(), "user not found") {
		return err
	}

	parsedID, err := uuid.Parse(userID)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	user := &db.User{
		ID:               parsedID,
		Email:            fmt.Sprintf("anon-%s@asgard.local", userID),
		PasswordHash:     uuid.New().String(),
		FullName:         sql.NullString{String: "Anonymous", Valid: true},
		SubscriptionTier: "observer",
		IsGovernment:     false,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	return repo.Create(user)
}

func (s *Server) handleStreamChat(w http.ResponseWriter, r *http.Request, streamID string) {
	if s.chatStore == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Chat service unavailable", "CHAT_UNAVAILABLE")
		return
	}

	switch r.Method {
	case http.MethodGet:
		limit := 50
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 200 {
				limit = l
			}
		}
		messages, err := s.chatStore.list(r.Context(), streamID, limit)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "Failed to load chat history", "CHAT_HISTORY_ERROR")
			return
		}
		s.writeJSON(w, http.StatusOK, messages)

	case http.MethodPost:
		var req chatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.writeError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST")
			return
		}
		req.Message = strings.TrimSpace(req.Message)
		if req.Message == "" {
			s.writeError(w, http.StatusBadRequest, "Message is required", "MESSAGE_REQUIRED")
			return
		}

		userID := ""
		username := strings.TrimSpace(req.Username)
		if token := extractToken(r); token != "" {
			if uid, role, _, _, err := parseJWTClaims(token); err == nil {
				userID = uid
				if username == "" && role != "" {
					username = role
				}
			}
		}
		if username == "" {
			username = "Viewer"
		}
		if userID == "" {
			userID = uuid.New().String()
		}

		msg, err := s.chatStore.add(r.Context(), streamID, userID, username, req.Message)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "Failed to send message", "CHAT_SEND_ERROR")
			return
		}

		if s.wsManager != nil {
			s.wsManager.Broadcast(realtime.Event{
				ID:          uuid.New().String(),
				Type:        realtime.EventTypeStreamChat,
				Source:      "nysus",
				Timestamp:   time.Now().UTC(),
				Payload:     map[string]interface{}{"streamId": streamID, "id": msg.ID, "userId": msg.UserID, "username": msg.Username, "message": msg.Message, "timestamp": msg.Timestamp},
				AccessLevel: realtime.AccessLevelPublic,
				Priority:    1,
			})
		}

		s.writeJSON(w, http.StatusOK, msg)

	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
	}
}

// handleStreamStats handles GET /api/streams/stats
func (s *Server) handleStreamStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	ctx := r.Context()

	if s.pgDB == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Database not available", "DB_UNAVAILABLE")
		return
	}

	var totalStreams, liveStreams, totalViewers int

	// Get total streams
	s.pgDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM streams").Scan(&totalStreams)

	// Get live streams
	s.pgDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM streams WHERE status = 'live'").Scan(&liveStreams)

	// Get total viewers
	s.pgDB.QueryRowContext(ctx, "SELECT COALESCE(SUM(viewers), 0) FROM streams WHERE status = 'live'").Scan(&totalViewers)

	// Get counts by category
	byCategory := make(map[string]int)
	rows, err := s.pgDB.QueryContext(ctx,
		"SELECT stream_type, COUNT(*) FROM streams GROUP BY stream_type")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var streamType string
			var count int
			if err := rows.Scan(&streamType, &count); err == nil {
				byCategory[streamType] = count
			}
		}
	}

	s.writeJSON(w, http.StatusOK, StreamStats{
		TotalStreams: totalStreams,
		LiveStreams:  liveStreams,
		TotalViewers: totalViewers,
		ByCategory:   byCategory,
	})
}

// handleRecentStreams handles GET /api/streams/recent
func (s *Server) handleRecentStreams(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	ctx := r.Context()
	if s.pgDB == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Database not available", "DB_UNAVAILABLE")
		return
	}

	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
			limit = l
		}
	}

	query := `
		SELECT st.id, st.title, st.source, st.source_type, st.source_id,
			   st.location, st.latitude, st.longitude, st.stream_type,
			   st.status, st.viewers, st.latency_ms, st.description,
			   st.resolution, st.bitrate, st.started_at, st.playback_url
		FROM streams st
		ORDER BY st.started_at DESC NULLS LAST
		LIMIT $1
	`

	rows, err := s.pgDB.QueryContext(ctx, query, limit)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Database query failed", "DB_ERROR")
		return
	}
	defer rows.Close()

	streams := []StreamResponse{}
	for rows.Next() {
		var stream StreamResponse
		var lat, lon sql.NullFloat64
		var desc sql.NullString
		var startedAt time.Time
		var playbackURL sql.NullString

		err := rows.Scan(
			&stream.ID, &stream.Title, &stream.Source, &stream.SourceType, &stream.SourceID,
			&stream.Location, &lat, &lon, &stream.Type,
			&stream.Status, &stream.Viewers, &stream.Latency, &desc,
			&stream.Resolution, &stream.Bitrate, &startedAt, &playbackURL,
		)
		if err != nil {
			continue
		}

		if lat.Valid && lon.Valid {
			stream.GeoLocation = &GeoLoc{Latitude: lat.Float64, Longitude: lon.Float64}
		}
		if desc.Valid {
			stream.Description = desc.String
		}
		if playbackURL.Valid {
			stream.PlaybackURL = playbackURL.String
		}
		stream.StartedAt = startedAt.Format(time.RFC3339)

		streams = append(streams, stream)
	}

	s.writeJSON(w, http.StatusOK, streams)
}

// handleFeaturedStreams handles GET /api/streams/featured
func (s *Server) handleFeaturedStreams(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	ctx := r.Context()

	if s.pgDB == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Database not available", "DB_UNAVAILABLE")
		return
	}

	// Query featured streams (live with most viewers)
	query := `
		SELECT st.id, st.title, st.source, st.source_type, st.source_id,
			   st.location, st.latitude, st.longitude, st.stream_type,
			   st.status, st.viewers, st.latency_ms, st.description,
			   st.resolution, st.bitrate, st.started_at, st.playback_url
		FROM streams st
		WHERE st.status = 'live' AND st.featured = true
		ORDER BY st.viewers DESC
		LIMIT 6
	`

	rows, err := s.pgDB.QueryContext(ctx, query)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Database query failed", "DB_ERROR")
		return
	}
	defer rows.Close()

	streams := []StreamResponse{}
	for rows.Next() {
		var stream StreamResponse
		var lat, lon sql.NullFloat64
		var desc sql.NullString
		var startedAt time.Time
		var playbackURL sql.NullString

		err := rows.Scan(
			&stream.ID, &stream.Title, &stream.Source, &stream.SourceType, &stream.SourceID,
			&stream.Location, &lat, &lon, &stream.Type,
			&stream.Status, &stream.Viewers, &stream.Latency, &desc,
			&stream.Resolution, &stream.Bitrate, &startedAt, &playbackURL,
		)
		if err != nil {
			continue
		}

		if lat.Valid && lon.Valid {
			stream.GeoLocation = &GeoLoc{Latitude: lat.Float64, Longitude: lon.Float64}
		}
		if desc.Valid {
			stream.Description = desc.String
		}
		if playbackURL.Valid {
			stream.PlaybackURL = playbackURL.String
		}
		stream.StartedAt = startedAt.Format(time.RFC3339)

		streams = append(streams, stream)
	}

	s.writeJSON(w, http.StatusOK, streams)
}

// handleStreamSearch handles GET /api/streams/search
func (s *Server) handleStreamSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	ctx := r.Context()

	if s.pgDB == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Database not available", "DB_UNAVAILABLE")
		return
	}

	searchQuery := strings.ToLower(r.URL.Query().Get("q"))
	if searchQuery == "" {
		s.writeJSON(w, http.StatusOK, []StreamResponse{})
		return
	}

	// Search streams by title or location
	query := `
		SELECT st.id, st.title, st.source, st.source_type, st.source_id,
			   st.location, st.latitude, st.longitude, st.stream_type,
			   st.status, st.viewers, st.latency_ms, st.description,
			   st.resolution, st.bitrate, st.started_at, st.playback_url
		FROM streams st
		WHERE (LOWER(st.title) LIKE $1 OR LOWER(st.location) LIKE $1)
		  AND st.status != 'offline'
		ORDER BY st.viewers DESC
		LIMIT 20
	`

	searchPattern := "%" + searchQuery + "%"
	rows, err := s.pgDB.QueryContext(ctx, query, searchPattern)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Database query failed", "DB_ERROR")
		return
	}
	defer rows.Close()

	streams := []StreamResponse{}
	for rows.Next() {
		var stream StreamResponse
		var lat, lon sql.NullFloat64
		var desc sql.NullString
		var startedAt time.Time
		var playbackURL sql.NullString

		err := rows.Scan(
			&stream.ID, &stream.Title, &stream.Source, &stream.SourceType, &stream.SourceID,
			&stream.Location, &lat, &lon, &stream.Type,
			&stream.Status, &stream.Viewers, &stream.Latency, &desc,
			&stream.Resolution, &stream.Bitrate, &startedAt, &playbackURL,
		)
		if err != nil {
			continue
		}

		if lat.Valid && lon.Valid {
			stream.GeoLocation = &GeoLoc{Latitude: lat.Float64, Longitude: lon.Float64}
		}
		if desc.Valid {
			stream.Description = desc.String
		}
		if playbackURL.Valid {
			stream.PlaybackURL = playbackURL.String
		}
		stream.StartedAt = startedAt.Format(time.RFC3339)

		streams = append(streams, stream)
	}

	s.writeJSON(w, http.StatusOK, streams)
}
