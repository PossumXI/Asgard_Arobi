// Package repositories provides data access layer for database operations.
package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/asgard/pandora/internal/platform/db"
	"github.com/google/uuid"
)

// StreamChatMessage represents a chat message persisted to Postgres.
type StreamChatMessage struct {
	ID        string
	StreamID  string
	UserID    string
	Username  string
	Message   string
	Timestamp time.Time
}

// StreamChatRepository handles stream chat persistence.
type StreamChatRepository struct {
	db *db.PostgresDB
}

// ErrChatUnavailable is returned when chat storage is not configured.
var ErrChatUnavailable = fmt.Errorf("chat storage not configured")

// NewStreamChatRepository creates a new chat repository.
func NewStreamChatRepository(pgDB *db.PostgresDB) *StreamChatRepository {
	return &StreamChatRepository{db: pgDB}
}

// List returns recent chat messages for a stream.
func (r *StreamChatRepository) List(ctx context.Context, streamID string, limit int) ([]*StreamChatMessage, error) {
	if r.db == nil {
		return nil, ErrChatUnavailable
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	streamUUID, err := uuid.Parse(streamID)
	if err != nil {
		return nil, fmt.Errorf("invalid stream ID: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id::text, stream_id::text, COALESCE(user_id::text, ''), username, message, created_at
		FROM stream_chat_messages
		WHERE stream_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, streamUUID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query chat messages: %w", err)
	}
	defer rows.Close()

	messages := make([]*StreamChatMessage, 0, limit)
	for rows.Next() {
		var msg StreamChatMessage
		if scanErr := rows.Scan(&msg.ID, &msg.StreamID, &msg.UserID, &msg.Username, &msg.Message, &msg.Timestamp); scanErr != nil {
			return nil, fmt.Errorf("failed to scan chat message: %w", scanErr)
		}
		messages = append(messages, &msg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate chat messages: %w", err)
	}

	// Reverse to chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// Add inserts a chat message for a stream.
func (r *StreamChatRepository) Add(ctx context.Context, streamID, userID, username, message string) (*StreamChatMessage, error) {
	if r.db == nil {
		return nil, ErrChatUnavailable
	}

	streamUUID, err := uuid.Parse(streamID)
	if err != nil {
		return nil, fmt.Errorf("invalid stream ID: %w", err)
	}

	var userUUID sql.NullString
	if userID != "" {
		if parsed, parseErr := uuid.Parse(userID); parseErr == nil {
			userUUID = sql.NullString{String: parsed.String(), Valid: true}
		}
	}

	if username == "" {
		username = "Viewer"
	}

	msg := &StreamChatMessage{
		ID:        uuid.New().String(),
		StreamID:  streamID,
		UserID:    userUUID.String,
		Username:  username,
		Message:   message,
		Timestamp: time.Now().UTC(),
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO stream_chat_messages (id, stream_id, user_id, username, message, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, msg.ID, streamUUID, nullUUID(userUUID), msg.Username, msg.Message, msg.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to insert chat message: %w", err)
	}

	return msg, nil
}

func nullUUID(value sql.NullString) interface{} {
	if value.Valid {
		return value.String
	}
	return nil
}
