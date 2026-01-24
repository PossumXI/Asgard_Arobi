package control

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// RemoteManipulator implements ManipulatorController via HTTP endpoints.
type RemoteManipulator struct {
	hunoidID string
	baseURL  string
	client   *http.Client
}

// NewRemoteManipulator creates a new remote manipulator controller.
func NewRemoteManipulator(hunoidID, baseURL string) (*RemoteManipulator, error) {
	if strings.TrimSpace(hunoidID) == "" {
		return nil, fmt.Errorf("hunoid ID is required")
	}
	trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if trimmed == "" {
		return nil, fmt.Errorf("remote manipulator endpoint is required")
	}

	return &RemoteManipulator{
		hunoidID: hunoidID,
		baseURL:  trimmed,
		client:   &http.Client{Timeout: 15 * time.Second},
	}, nil
}

func (m *RemoteManipulator) OpenGripper() error {
	return m.postJSON(context.Background(), "/hunoids/"+m.hunoidID+"/manipulator/open", nil, nil)
}

func (m *RemoteManipulator) CloseGripper() error {
	return m.postJSON(context.Background(), "/hunoids/"+m.hunoidID+"/manipulator/close", nil, nil)
}

func (m *RemoteManipulator) GetGripperState() (float64, error) {
	var resp struct {
		State float64 `json:"state"`
	}
	if err := m.getJSON("/hunoids/"+m.hunoidID+"/manipulator/state", &resp); err != nil {
		return 0, err
	}
	return resp.State, nil
}

func (m *RemoteManipulator) ReachTo(ctx context.Context, position Vector3) error {
	payload := map[string]interface{}{"position": position}
	return m.postJSON(ctx, "/hunoids/"+m.hunoidID+"/manipulator/reach", payload, nil)
}

func (m *RemoteManipulator) getJSON(path string, out interface{}) error {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, m.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := m.client.Do(req)
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

func (m *RemoteManipulator) postJSON(ctx context.Context, path string, payload interface{}, out interface{}) error {
	var body []byte
	var err error
	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to encode payload: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
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
