package hal

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

// RemotePowerController implements PowerController via an HTTP telemetry endpoint.
type RemotePowerController struct {
	baseURL   string
	client    *http.Client
	cache     powerStatus
	cacheUntil time.Time
	mu         sync.RWMutex
}

type powerStatus struct {
	BatteryPercent float64 `json:"batteryPercent"`
	BatteryVoltage float64 `json:"batteryVoltage"`
	SolarPanelPower float64 `json:"solarPanelPower"`
	InEclipse      bool    `json:"inEclipse"`
}

// NewRemotePowerController creates a new power controller using the provided endpoint.
func NewRemotePowerController(baseURL string) (*RemotePowerController, error) {
	trimmed := strings.TrimSpace(baseURL)
	if trimmed == "" {
		return nil, fmt.Errorf("power controller endpoint is required")
	}

	return &RemotePowerController{
		baseURL: trimmed,
		client: &http.Client{Timeout: 10 * time.Second},
	}, nil
}

func (p *RemotePowerController) GetBatteryPercent() (float64, error) {
	status, err := p.getStatus()
	if err != nil {
		return 0, err
	}
	return status.BatteryPercent, nil
}

func (p *RemotePowerController) GetBatteryVoltage() (float64, error) {
	status, err := p.getStatus()
	if err != nil {
		return 0, err
	}
	return status.BatteryVoltage, nil
}

func (p *RemotePowerController) GetSolarPanelPower() (float64, error) {
	status, err := p.getStatus()
	if err != nil {
		return 0, err
	}
	return status.SolarPanelPower, nil
}

func (p *RemotePowerController) IsInEclipse() (bool, error) {
	status, err := p.getStatus()
	if err != nil {
		return false, err
	}
	return status.InEclipse, nil
}

func (p *RemotePowerController) SetPowerMode(mode PowerMode) error {
	payload := map[string]string{"mode": string(mode)}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to encode power mode: %w", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, p.baseURL+"/mode", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create power mode request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to set power mode: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("power mode update failed with status %d", resp.StatusCode)
	}

	return nil
}

func (p *RemotePowerController) getStatus() (powerStatus, error) {
	p.mu.RLock()
	if time.Now().Before(p.cacheUntil) {
		status := p.cache
		p.mu.RUnlock()
		return status, nil
	}
	p.mu.RUnlock()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, p.baseURL, nil)
	if err != nil {
		return powerStatus{}, fmt.Errorf("failed to create power status request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return powerStatus{}, fmt.Errorf("failed to fetch power status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return powerStatus{}, fmt.Errorf("power status request failed with status %d", resp.StatusCode)
	}

	var status powerStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return powerStatus{}, fmt.Errorf("failed to decode power status: %w", err)
	}

	p.mu.Lock()
	p.cache = status
	p.cacheUntil = time.Now().Add(5 * time.Second)
	p.mu.Unlock()

	return status, nil
}
