package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/asgard/pandora/internal/services"
)

type PricillaHandler struct {
	service *services.PricillaService
}

func NewPricillaHandler(service *services.PricillaService) *PricillaHandler {
	return &PricillaHandler{service: service}
}

func (h *PricillaHandler) HandleMissions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		missions, err := h.service.GetAllMissions(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(missions)

	case http.MethodPost:
		var mission services.GuidanceMission
		if err := json.NewDecoder(r.Body).Decode(&mission); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		id, err := h.service.CreateMission(r.Context(), &mission)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"id": id})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *PricillaHandler) HandlePayloads(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var state services.PayloadState
	if err := json.NewDecoder(r.Body).Decode(&state); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.service.UpdatePayloadState(r.Context(), &state); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *PricillaHandler) HandleMission(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/pricilla/missions/")
	if id == "" {
		http.Error(w, "Mission ID required", http.StatusBadRequest)
		return
	}

	mission, err := h.service.GetMission(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(mission)
}
