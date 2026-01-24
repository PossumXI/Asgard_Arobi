// Package handlers provides HTTP handlers for API endpoints.
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/asgard/pandora/internal/repositories"
	"github.com/asgard/pandora/internal/services"
	"github.com/go-chi/chi/v5"
)

// StreamHandler handles stream endpoints.
type StreamHandler struct {
	streamService *services.StreamService
}

// NewStreamHandler creates a new stream handler.
func NewStreamHandler(streamService *services.StreamService) *StreamHandler {
	return &StreamHandler{
		streamService: streamService,
	}
}

// GetStreams handles GET /api/streams
// If authenticated, filters streams based on user's subscription tier.
// Unauthenticated users only see public/civilian streams.
func (h *StreamHandler) GetStreams(w http.ResponseWriter, r *http.Request) {
	streamType := r.URL.Query().Get("type")
	status := r.URL.Query().Get("status")

	limit := 20
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			offset = o
		}
	}

	// Extract user tier from context (set by auth middleware)
	userTier := "free" // Default to free tier for unauthenticated users
	if claims, ok := getAuthClaimsFromContext(r); ok {
		userTier = claims.SubscriptionTier
	}

	// Use tier-based filtering
	streams, total, err := h.streamService.GetStreamsForUser(r.Context(), userTier, streamType, status, limit, offset)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error(), "STREAM_FETCH_ERROR")
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"streams": streams,
		"total":   total,
	})
}

// GetStream handles GET /api/streams/{id}
// Checks if user has access to the stream based on their subscription tier.
func (h *StreamHandler) GetStream(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Extract user tier from context (set by auth middleware)
	userTier := "free" // Default to free tier for unauthenticated users
	if claims, ok := getAuthClaimsFromContext(r); ok {
		userTier = claims.SubscriptionTier
	}

	stream, err := h.streamService.GetStreamForUser(r.Context(), userTier, id)
	if err != nil {
		if errors.Is(err, repositories.ErrStreamAccessDenied) {
			jsonError(w, http.StatusForbidden, "Access denied to this stream", "STREAM_ACCESS_DENIED")
			return
		}
		if errors.Is(err, repositories.ErrStreamNotFound) {
			jsonError(w, http.StatusNotFound, "Stream not found", "STREAM_NOT_FOUND")
			return
		}
		jsonError(w, http.StatusNotFound, err.Error(), "STREAM_NOT_FOUND")
		return
	}

	jsonResponse(w, http.StatusOK, stream)
}

// GetStreamStats handles GET /api/streams/stats
func (h *StreamHandler) GetStreamStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.streamService.GetStreamStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, stats)
}

// GetRecentStreams handles GET /api/streams/recent
func (h *StreamHandler) GetRecentStreams(w http.ResponseWriter, r *http.Request) {
	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
			limit = l
		}
	}

	userTier := "free"
	if claims, ok := getAuthClaimsFromContext(r); ok {
		userTier = claims.SubscriptionTier
	}

	streams, _, err := h.streamService.GetStreamsForUser(r.Context(), userTier, "", "", limit, 0)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error(), "STREAM_FETCH_ERROR")
		return
	}

	jsonResponse(w, http.StatusOK, streams)
}

// GetFeaturedStreams handles GET /api/streams/featured
func (h *StreamHandler) GetFeaturedStreams(w http.ResponseWriter, r *http.Request) {
	streams, err := h.streamService.GetFeaturedStreams()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, streams)
}

// SearchStreams handles GET /api/streams/search
func (h *StreamHandler) SearchStreams(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	streams, err := h.streamService.SearchStreams(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, streams)
}

// CreateStreamSession handles POST /api/streams/{id}/session
func (h *StreamHandler) CreateStreamSession(w http.ResponseWriter, r *http.Request) {
	streamID := chi.URLParam(r, "id")
	claims, _ := getAuthClaimsFromContext(r)
	userID := claims.UserID
	userTier := claims.SubscriptionTier
	isGovernment := claims.IsGovernment
	if userID == "" {
		userID = "anonymous"
		userTier = "free"
	}

	stream, err := h.streamService.GetStream(streamID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if !isGovernment && !services.CanAccessStreamType(userTier, stream.Type) {
		http.Error(w, "Forbidden: insufficient access tier", http.StatusForbidden)
		return
	}

	session, err := h.streamService.CreateStreamSession(streamID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, session)
}

// GetStreamChat handles GET /api/streams/{id}/chat
func (h *StreamHandler) GetStreamChat(w http.ResponseWriter, r *http.Request) {
	streamID := chi.URLParam(r, "id")
	if streamID == "" {
		jsonError(w, http.StatusBadRequest, "Stream ID required", "INVALID_REQUEST")
		return
	}

	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 200 {
			limit = l
		}
	}

	messages, err := h.streamService.ListChatMessages(r.Context(), streamID, limit)
	if err != nil {
		if errors.Is(err, services.ErrChatUnavailable) || errors.Is(err, repositories.ErrChatUnavailable) {
			jsonError(w, http.StatusServiceUnavailable, "Chat service unavailable", "CHAT_UNAVAILABLE")
			return
		}
		jsonError(w, http.StatusInternalServerError, "Failed to load chat history", "CHAT_HISTORY_ERROR")
		return
	}

	response := make([]ChatMessage, 0, len(messages))
	for _, msg := range messages {
		response = append(response, ChatMessage{
			ID:        msg.ID,
			StreamID:  msg.StreamID,
			UserID:    msg.UserID,
			Username:  msg.Username,
			Message:   msg.Message,
			Timestamp: msg.Timestamp.UTC().Format(time.RFC3339),
		})
	}

	jsonResponse(w, http.StatusOK, response)
}

// SendStreamChat handles POST /api/streams/{id}/chat
func (h *StreamHandler) SendStreamChat(w http.ResponseWriter, r *http.Request) {
	streamID := chi.URLParam(r, "id")
	if streamID == "" {
		jsonError(w, http.StatusBadRequest, "Stream ID required", "INVALID_REQUEST")
		return
	}

	claims, ok := getAuthClaimsFromContext(r)
	if !ok || claims.UserID == "" {
		jsonError(w, http.StatusUnauthorized, "Authentication required", "UNAUTHORIZED")
		return
	}

	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST")
		return
	}
	req.Message = strings.TrimSpace(req.Message)
	if req.Message == "" {
		jsonError(w, http.StatusBadRequest, "Message is required", "MESSAGE_REQUIRED")
		return
	}

	userID := claims.UserID
	username := strings.TrimSpace(req.Username)
	if username == "" && claims.Role != "" {
		username = claims.Role
	}
	if username == "" {
		username = "User"
	}

	message, err := h.streamService.AddChatMessage(r.Context(), streamID, userID, username, req.Message)
	if err != nil {
		if errors.Is(err, services.ErrChatUnavailable) || errors.Is(err, repositories.ErrChatUnavailable) {
			jsonError(w, http.StatusServiceUnavailable, "Chat service unavailable", "CHAT_UNAVAILABLE")
			return
		}
		jsonError(w, http.StatusInternalServerError, "Failed to send message", "CHAT_SEND_ERROR")
		return
	}

	jsonResponse(w, http.StatusOK, ChatMessage{
		ID:        message.ID,
		StreamID:  message.StreamID,
		UserID:    message.UserID,
		Username:  message.Username,
		Message:   message.Message,
		Timestamp: message.Timestamp.UTC().Format(time.RFC3339),
	})
}

type chatRequest struct {
	Message  string `json:"message"`
	Username string `json:"username"`
}

type ChatMessage struct {
	ID        string `json:"id"`
	StreamID  string `json:"streamId"`
	UserID    string `json:"userId"`
	Username  string `json:"username"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}
