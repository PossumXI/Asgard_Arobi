package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/asgard/pandora/internal/robotics/ethics"
	"github.com/asgard/pandora/internal/robotics/vla"
	"github.com/google/uuid"
)

// PercilaService manages the PERCILA AI guidance system integration.
type PercilaService struct {
	baseURL      string
	client       *http.Client
	ethicsKernel *ethics.EthicalKernel
	auditService *AuditService
}

type Vector3D struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type Waypoint struct {
	ID        string    `json:"id"`
	Position  Vector3D  `json:"position"`
	Velocity  Vector3D  `json:"velocity"`
	Timestamp time.Time `json:"timestamp"`
	Stealth   bool      `json:"stealth"`
}

type Trajectory struct {
	ID           string     `json:"id"`
	PayloadID    string     `json:"payloadId"`
	Waypoints    []Waypoint `json:"waypoints"`
	StealthScore float64    `json:"stealthScore"`
	Confidence   float64    `json:"confidence"`
}

type GuidanceMission struct {
	ID              string      `json:"id"`
	Type            string      `json:"type"`
	PayloadID       string      `json:"payloadId"`
	PayloadType     string      `json:"payloadType"`
	StartPosition   Vector3D    `json:"startPosition"`
	TargetPosition  Vector3D    `json:"targetPosition"`
	Priority        int         `json:"priority"`
	StealthRequired bool        `json:"stealthRequired"`
	Status          string      `json:"status"`
	Trajectory      *Trajectory `json:"trajectory"`
	CreatedAt       time.Time   `json:"createdAt"`
}

type PayloadState struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	Position   Vector3D  `json:"position"`
	Velocity   Vector3D  `json:"velocity"`
	Fuel       float64   `json:"fuel"`
	Battery    float64   `json:"battery"`
	Health     float64   `json:"health"`
	Status     string    `json:"status"`
	LastUpdate time.Time `json:"lastUpdate"`
}

func NewPercilaService() *PercilaService {
	return &PercilaService{
		baseURL:      strings.TrimRight(strings.TrimSpace(os.Getenv("PERCILA_ENDPOINT")), "/"),
		client:       &http.Client{Timeout: 20 * time.Second},
		ethicsKernel: ethics.NewEthicalKernel(),
	}
}

// NewPercilaServiceWithAudit creates a PERCILA service with audit integration.
func NewPercilaServiceWithAudit(auditService *AuditService) *PercilaService {
	service := NewPercilaService()
	service.auditService = auditService
	return service
}

func (s *PercilaService) CreateMission(ctx context.Context, mission *GuidanceMission) (string, error) {
	if err := s.ensureConfigured(); err != nil {
		return "", err
	}

	if mission.ID == "" {
		mission.ID = uuid.New().String()
	}
	mission.CreatedAt = time.Now().UTC()

	if s.ethicsKernel != nil {
		action := &vla.Action{
			Type: vla.ActionNavigate,
			Parameters: map[string]interface{}{
				"mission_type":     mission.Type,
				"payload_id":       mission.PayloadID,
				"stealth_required": mission.StealthRequired,
				"priority":         mission.Priority,
				"target_position":  mission.TargetPosition,
				"start_position":   mission.StartPosition,
				"payload_type":     mission.PayloadType,
			},
			Confidence: 0.9,
		}
		decision, err := s.ethicsKernel.Evaluate(ctx, action)
		if err != nil {
			return "", fmt.Errorf("ethics evaluation failed: %w", err)
		}
		if s.auditService != nil {
			_ = s.auditService.LogEthicalDecision(ctx, mission.PayloadID, string(decision.Decision), decision.Reasoning, map[string]interface{}{
				"mission_id":   mission.ID,
				"mission_type": mission.Type,
				"score":        decision.Score,
			})
		}
		if decision.Decision != ethics.DecisionApproved {
			mission.Status = "pending_review"
			return "", fmt.Errorf("mission blocked by ethics review: %s", decision.Reasoning)
		}
	}

	var resp GuidanceMission
	if err := s.postJSON(ctx, "/api/percila/missions", mission, &resp); err != nil {
		return "", err
	}
	return resp.ID, nil
}

func (s *PercilaService) GetMission(ctx context.Context, id string) (*GuidanceMission, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, err
	}

	var resp GuidanceMission
	if err := s.getJSON(ctx, "/api/percila/missions/"+id, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

func (s *PercilaService) GetAllMissions(ctx context.Context) ([]*GuidanceMission, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, err
	}

	var resp []*GuidanceMission
	if err := s.getJSON(ctx, "/api/percila/missions", &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *PercilaService) UpdatePayloadState(ctx context.Context, state *PayloadState) error {
	if err := s.ensureConfigured(); err != nil {
		return err
	}

	state.LastUpdate = time.Now().UTC()
	return s.postJSON(ctx, "/api/percila/payloads", state, nil)
}

func (s *PercilaService) GetPayloadStates(ctx context.Context) ([]*PayloadState, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, err
	}

	var resp []*PayloadState
	if err := s.getJSON(ctx, "/api/percila/payloads", &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *PercilaService) ensureConfigured() error {
	if s.baseURL == "" {
		return fmt.Errorf("PERCILA_ENDPOINT is not configured")
	}
	return nil
}

func (s *PercilaService) getJSON(ctx context.Context, path string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("request failed with status %d", resp.StatusCode)
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}
	return nil
}

func (s *PercilaService) postJSON(ctx context.Context, path string, payload interface{}, out interface{}) error {
	var body []byte
	var err error
	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to encode payload: %w", err)
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("request failed with status %d", resp.StatusCode)
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}
	return nil
}
