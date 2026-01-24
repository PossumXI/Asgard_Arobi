package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/asgard/pandora/internal/services"
)

func (s *Server) handlePercilaMissions(w http.ResponseWriter, r *http.Request) {
	service := services.NewPercilaService()
	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		missions, err := service.GetAllMissions(ctx)
		if err != nil {
			s.writeError(w, http.StatusServiceUnavailable, err.Error(), "PERCILA_UNAVAILABLE")
			return
		}
		s.writeJSON(w, http.StatusOK, missions)

	case http.MethodPost:
		var mission services.GuidanceMission
		if err := json.NewDecoder(r.Body).Decode(&mission); err != nil {
			s.writeError(w, http.StatusBadRequest, "Invalid mission data", "INVALID_DATA")
			return
		}
		missionID, err := service.CreateMission(ctx, &mission)
		if err != nil {
			s.writeError(w, http.StatusBadRequest, err.Error(), "MISSION_FAILED")
			return
		}
		created, err := service.GetMission(ctx, missionID)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, err.Error(), "MISSION_FAILED")
			return
		}
		s.writeJSON(w, http.StatusCreated, created)

	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
	}
}

func (s *Server) handlePercilaMission(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/percila/missions/")
	if id == "" {
		s.writeError(w, http.StatusBadRequest, "Mission ID required", "MISSING_ID")
		return
	}

	service := services.NewPercilaService()
	mission, err := service.GetMission(r.Context(), id)
	if err != nil {
		s.writeError(w, http.StatusNotFound, err.Error(), "NOT_FOUND")
		return
	}

	s.writeJSON(w, http.StatusOK, mission)
}

func (s *Server) handlePercilaPayloads(w http.ResponseWriter, r *http.Request) {
	service := services.NewPercilaService()
	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		payloads, err := service.GetPayloadStates(ctx)
		if err != nil {
			s.writeError(w, http.StatusServiceUnavailable, err.Error(), "PERCILA_UNAVAILABLE")
			return
		}
		s.writeJSON(w, http.StatusOK, payloads)

	case http.MethodPost:
		var state services.PayloadState
		if err := json.NewDecoder(r.Body).Decode(&state); err != nil {
			s.writeError(w, http.StatusBadRequest, "Invalid payload state", "INVALID_DATA")
			return
		}
		if err := service.UpdatePayloadState(ctx, &state); err != nil {
			s.writeError(w, http.StatusBadRequest, err.Error(), "PAYLOAD_FAILED")
			return
		}
		s.writeJSON(w, http.StatusOK, state)

	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
	}
}
