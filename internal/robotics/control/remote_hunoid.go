package control

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// RemoteHunoid implements HunoidController via HTTP endpoints.
type RemoteHunoid struct {
	id          string
	baseURL     string
	client      *http.Client
	mu          sync.RWMutex
	lastBattery float64
}

// NewRemoteHunoid creates a new remote Hunoid controller.
func NewRemoteHunoid(id, baseURL string) (*RemoteHunoid, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("hunoid ID is required")
	}
	trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if trimmed == "" {
		return nil, fmt.Errorf("remote hunoid endpoint is required")
	}

	return &RemoteHunoid{
		id:      id,
		baseURL: trimmed,
		client:  &http.Client{Timeout: 15 * time.Second},
	}, nil
}

func (h *RemoteHunoid) Initialize(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create health request: %w", err)
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("hunoid health check failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("hunoid health check returned %d", resp.StatusCode)
	}
	return nil
}

func (h *RemoteHunoid) GetCurrentPose() (Pose, error) {
	var pose Pose
	if err := h.getJSON("/hunoids/"+h.id+"/pose", &pose); err != nil {
		return Pose{}, err
	}
	return pose, nil
}

func (h *RemoteHunoid) MoveTo(ctx context.Context, target Pose) error {
	payload := map[string]interface{}{
		"position":    target.Position,
		"orientation": target.Orientation,
	}
	return h.postJSON(ctx, "/hunoids/"+h.id+"/move", payload, nil)
}

func (h *RemoteHunoid) Stop() error {
	return h.postJSON(context.Background(), "/hunoids/"+h.id+"/stop", nil, nil)
}

func (h *RemoteHunoid) GetJointStates() ([]Joint, error) {
	var joints []Joint
	if err := h.getJSON("/hunoids/"+h.id+"/joints", &joints); err != nil {
		return nil, err
	}
	return joints, nil
}

func (h *RemoteHunoid) SetJointPositions(positions map[string]float64) error {
	return h.postJSON(context.Background(), "/hunoids/"+h.id+"/joints", map[string]interface{}{
		"positions": positions,
	}, nil)
}

func (h *RemoteHunoid) IsMoving() bool {
	var status struct {
		IsMoving bool `json:"isMoving"`
	}
	if err := h.getJSON("/hunoids/"+h.id+"/status", &status); err != nil {
		return false
	}
	return status.IsMoving
}

func (h *RemoteHunoid) GetBatteryPercent() float64 {
	var status struct {
		BatteryPercent float64 `json:"batteryPercent"`
	}
	if err := h.getJSON("/hunoids/"+h.id+"/battery", &status); err != nil {
		h.mu.RLock()
		defer h.mu.RUnlock()
		return h.lastBattery
	}
	h.mu.Lock()
	h.lastBattery = status.BatteryPercent
	h.mu.Unlock()
	return status.BatteryPercent
}

func (h *RemoteHunoid) getJSON(path string, out interface{}) error {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, h.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("request failed with status %d", resp.StatusCode)
	}
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}
	return nil
}

func (h *RemoteHunoid) postJSON(ctx context.Context, path string, payload interface{}, out interface{}) error {
	var body []byte
	var err error
	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to encode payload: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
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
