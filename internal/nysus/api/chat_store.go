package api

import (
	"context"
	"database/sql"
	"strings"
	"sync"
	"time"

	"github.com/asgard/pandora/internal/platform/db"
	"github.com/google/uuid"
)

// ChatMessage represents a single chat message for a stream.
type ChatMessage struct {
	ID        string `json:"id"`
	StreamID  string `json:"streamId"`
	UserID    string `json:"userId"`
	Username  string `json:"username"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

type chatStore struct {
	mu       sync.RWMutex
	pgDB     *db.PostgresDB
	messages map[string][]ChatMessage
}

func newChatStore(pgDB *db.PostgresDB) *chatStore {
	return &chatStore{
		pgDB:     pgDB,
		messages: make(map[string][]ChatMessage),
	}
}

func (c *chatStore) list(ctx context.Context, streamID string, limit int) ([]ChatMessage, error) {
	if limit <= 0 {
		limit = 50
	}

	if c.pgDB != nil {
		streamUUID, err := uuid.Parse(streamID)
		if err == nil {
			rows, err := c.pgDB.QueryContext(ctx, `
				SELECT id::text, stream_id::text, COALESCE(user_id::text, ''), username, message, created_at
				FROM stream_chat_messages
				WHERE stream_id = $1
				ORDER BY created_at DESC
				LIMIT $2
			`, streamUUID, limit)
			if err == nil {
				defer rows.Close()
				messages := make([]ChatMessage, 0, limit)
				for rows.Next() {
					var msg ChatMessage
					var createdAt time.Time
					if scanErr := rows.Scan(&msg.ID, &msg.StreamID, &msg.UserID, &msg.Username, &msg.Message, &createdAt); scanErr != nil {
						continue
					}
					msg.Timestamp = createdAt.UTC().Format(time.RFC3339)
					messages = append(messages, msg)
				}

				// Reverse to chronological order
				for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
					messages[i], messages[j] = messages[j], messages[i]
				}

				return messages, rows.Err()
			}

			if !isMissingTableError(err) {
				return nil, err
			}
		}
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	items := c.messages[streamID]
	if len(items) <= limit {
		return append([]ChatMessage(nil), items...), nil
	}
	return append([]ChatMessage(nil), items[len(items)-limit:]...), nil
}

func (c *chatStore) add(ctx context.Context, streamID, userID, username, message string) (ChatMessage, error) {
	now := time.Now().UTC()
	if username == "" {
		username = "Anonymous"
	}
	msg := ChatMessage{
		ID:        uuid.New().String(),
		StreamID:  streamID,
		UserID:    userID,
		Username:  username,
		Message:   message,
		Timestamp: now.Format(time.RFC3339),
	}

	if c.pgDB != nil {
		streamUUID, streamErr := uuid.Parse(streamID)
		if streamErr == nil {
			var userUUID sql.NullString
			if parsedUser, err := uuid.Parse(userID); err == nil {
				userUUID = sql.NullString{String: parsedUser.String(), Valid: true}
			}

			_, err := c.pgDB.ExecContext(ctx, `
				INSERT INTO stream_chat_messages (id, stream_id, user_id, username, message, created_at)
				VALUES ($1, $2, $3, $4, $5, $6)
			`, msg.ID, streamUUID, nullUUID(userUUID), msg.Username, msg.Message, now)
			if err == nil {
				return msg, nil
			}
			if !isMissingTableError(err) {
				return msg, err
			}
		}
	}

	c.mu.Lock()
	c.messages[streamID] = append(c.messages[streamID], msg)
	if len(c.messages[streamID]) > 200 {
		c.messages[streamID] = c.messages[streamID][len(c.messages[streamID])-200:]
	}
	c.mu.Unlock()

	return msg, nil
}

func nullUUID(value sql.NullString) interface{} {
	if value.Valid {
		return value.String
	}
	return nil
}

func isMissingTableError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "stream_chat_messages") && strings.Contains(err.Error(), "does not exist")
}
