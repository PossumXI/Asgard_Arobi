package vla

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// HTTPVLA implements VLAModel via an HTTP service.
type HTTPVLA struct {
	endpoint string
	client   *http.Client
	info     ModelInfo
}

// NewHTTPVLA creates a new HTTP-backed VLA client.
func NewHTTPVLA(endpoint string) (*HTTPVLA, error) {
	trimmed := strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if trimmed == "" {
		return nil, fmt.Errorf("VLA endpoint is required")
	}

	return &HTTPVLA{
		endpoint: trimmed,
		client:   &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (v *HTTPVLA) Initialize(ctx context.Context, modelPath string) error {
	payload := map[string]string{"modelPath": modelPath}
	if err := v.postJSON(ctx, "/initialize", payload, nil); err != nil {
		return err
	}

	var info ModelInfo
	if err := v.getJSON("/info", &info); err == nil {
		v.info = info
	}

	return nil
}

func (v *HTTPVLA) InferAction(ctx context.Context, visualObs []byte, textCommand string) (*Action, error) {
	payload := map[string]interface{}{
		"textCommand":     textCommand,
		"visualObsBase64": base64.StdEncoding.EncodeToString(visualObs),
	}

	var resp struct {
		Type       ActionType             `json:"type"`
		Parameters map[string]interface{} `json:"parameters"`
		Confidence float64                `json:"confidence"`
	}

	if err := v.postJSON(ctx, "/infer", payload, &resp); err != nil {
		return nil, err
	}

	return &Action{
		Type:       resp.Type,
		Parameters: resp.Parameters,
		Confidence: resp.Confidence,
	}, nil
}

func (v *HTTPVLA) GetModelInfo() ModelInfo {
	if v.info.Name != "" {
		return v.info
	}
	return ModelInfo{
		Name:    "VLA-HTTP",
		Version: "unknown",
	}
}

func (v *HTTPVLA) Shutdown() error {
	// Optional shutdown endpoint; ignore errors if not supported.
	_ = v.postJSON(context.Background(), "/shutdown", nil, nil)
	return nil
}

func (v *HTTPVLA) getJSON(path string, out interface{}) error {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, v.endpoint+path, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := v.client.Do(req)
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

func (v *HTTPVLA) postJSON(ctx context.Context, path string, payload interface{}, out interface{}) error {
	var body []byte
	var err error
	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to encode payload: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.endpoint+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := v.client.Do(req)
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
