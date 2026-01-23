// Package handlers provides HTTP handlers for API endpoints.
package handlers

import (
	"net/http"
	"strconv"

	"github.com/asgard/pandora/internal/services"
	"github.com/go-chi/chi/v5"
)

// StreamHandler handles stream endpoints.
type StreamHandler struct {
	streamService *services.StreamService
}

// NewStreamHandler creates a new stream handler.
func NewStreamHandler(streamService *services.StreamService) *StreamHandler {
	return &StreamHandler{streamService: streamService}
}

// GetStreams handles GET /api/streams
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

	streams, total, err := h.streamService.GetStreams(streamType, status, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"streams": streams,
		"total":   total,
	})
}

// GetStream handles GET /api/streams/{id}
func (h *StreamHandler) GetStream(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	stream, err := h.streamService.GetStream(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
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
	userID := getUserIDFromContext(r)
	if userID == "" {
		userID = "anonymous"
	}

	session, err := h.streamService.CreateStreamSession(streamID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, session)
}
