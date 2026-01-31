// Package integration provides clients for ASGARD system integration
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// ASGARDClients manages all ASGARD system connections
type ASGARDClients struct {
	mu sync.RWMutex

	Nysus   *NysusClient
	Silenus *SilenusClient
	SatNet  *SatNetClient
	Giru    *GiruClient
	Hunoid  *HunoidClient

	logger     *logrus.Logger
	httpClient *http.Client
}

// ClientConfig holds common client configuration
type ClientConfig struct {
	BaseURL string
	Timeout time.Duration
	APIKey  string
}

// NysusClient connects to Nysus orchestration
type NysusClient struct {
	config ClientConfig
	client *http.Client
}

// SilenusClient connects to Silenus data services
type SilenusClient struct {
	config ClientConfig
	client *http.Client
}

// SatNetClient connects to Sat_Net DTN
type SatNetClient struct {
	config ClientConfig
	client *http.Client
}

// GiruClient connects to Giru security
type GiruClient struct {
	config ClientConfig
	client *http.Client
}

// HunoidClient connects to Hunoid robotics
type HunoidClient struct {
	config ClientConfig
	client *http.Client
}

// NewASGARDClients creates all ASGARD client connections
func NewASGARDClients(
	nysusURL, silenusURL, satnetURL, giruURL, hunoidURL string,
	timeout time.Duration,
) *ASGARDClients {
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	httpClient := &http.Client{
		Timeout: timeout,
	}

	return &ASGARDClients{
		Nysus: &NysusClient{
			config: ClientConfig{BaseURL: nysusURL, Timeout: timeout},
			client: httpClient,
		},
		Silenus: &SilenusClient{
			config: ClientConfig{BaseURL: silenusURL, Timeout: timeout},
			client: httpClient,
		},
		SatNet: &SatNetClient{
			config: ClientConfig{BaseURL: satnetURL, Timeout: timeout},
			client: httpClient,
		},
		Giru: &GiruClient{
			config: ClientConfig{BaseURL: giruURL, Timeout: timeout},
			client: httpClient,
		},
		Hunoid: &HunoidClient{
			config: ClientConfig{BaseURL: hunoidURL, Timeout: timeout},
			client: httpClient,
		},
		logger:     logrus.New(),
		httpClient: httpClient,
	}
}

// HealthCheck verifies all systems are reachable
func (ac *ASGARDClients) HealthCheck(ctx context.Context) map[string]bool {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	status := make(map[string]bool)

	// Check each system in parallel
	var wg sync.WaitGroup
	var mu sync.Mutex

	systems := []struct {
		name string
		ping func(context.Context) bool
	}{
		{"nysus", ac.Nysus.Ping},
		{"silenus", ac.Silenus.Ping},
		{"satnet", ac.SatNet.Ping},
		{"giru", ac.Giru.Ping},
		{"hunoid", ac.Hunoid.Ping},
	}

	for _, sys := range systems {
		wg.Add(1)
		go func(name string, ping func(context.Context) bool) {
			defer wg.Done()
			result := ping(ctx)
			mu.Lock()
			status[name] = result
			mu.Unlock()
		}(sys.name, sys.ping)
	}

	wg.Wait()
	return status
}

// --------- Nysus Client Methods ---------

// Ping checks if Nysus is reachable
func (nc *NysusClient) Ping(ctx context.Context) bool {
	resp, err := nc.get(ctx, "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// RegisterAgent registers Valkyrie with Nysus
func (nc *NysusClient) RegisterAgent(ctx context.Context, agentID string, capabilities []string) error {
	payload := map[string]interface{}{
		"agent_id":     agentID,
		"agent_type":   "valkyrie",
		"capabilities": capabilities,
		"status":       "active",
	}

	resp, err := nc.post(ctx, "/api/v1/agents/register", payload)
	if err != nil {
		return fmt.Errorf("failed to register agent: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("registration failed: %s", string(body))
	}

	return nil
}

// ReportTelemetry sends telemetry to Nysus
func (nc *NysusClient) ReportTelemetry(ctx context.Context, telemetry interface{}) error {
	resp, err := nc.post(ctx, "/api/v1/telemetry", telemetry)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// GetMission retrieves mission details from Nysus
func (nc *NysusClient) GetMission(ctx context.Context, missionID string) (map[string]interface{}, error) {
	resp, err := nc.get(ctx, "/api/v1/missions/"+missionID)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetRealtimeSatellitePosition retrieves real-time satellite position from Nysus.
func (nc *NysusClient) GetRealtimeSatellitePosition(ctx context.Context, noradID int) (float64, float64, error) {
	if noradID == 0 {
		return 0, 0, fmt.Errorf("norad_id is required")
	}
	path := fmt.Sprintf("/api/satellites/realtime?norad_id=%d", noradID)
	resp, err := nc.get(ctx, path)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, 0, fmt.Errorf("nysus satellite lookup failed: %s", string(body))
	}

	var result struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, 0, err
	}
	return result.Latitude, result.Longitude, nil
}

func (nc *NysusClient) get(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", nc.config.BaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	if nc.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+nc.config.APIKey)
	}
	return nc.client.Do(req)
}

func (nc *NysusClient) post(ctx context.Context, path string, payload interface{}) (*http.Response, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", nc.config.BaseURL+path, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if nc.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+nc.config.APIKey)
	}
	return nc.client.Do(req)
}

// --------- Silenus Client Methods ---------

// Ping checks if Silenus is reachable
func (sc *SilenusClient) Ping(ctx context.Context) bool {
	resp, err := sc.get(ctx, "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// GetWeather retrieves weather data for a location
func (sc *SilenusClient) GetWeather(ctx context.Context, lat, lon float64) (*WeatherData, error) {
	path := fmt.Sprintf("/api/v1/weather?lat=%f&lon=%f", lat, lon)
	resp, err := sc.get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var weather WeatherData
	if err := json.NewDecoder(resp.Body).Decode(&weather); err != nil {
		return nil, err
	}
	return &weather, nil
}

// GetTerrain retrieves terrain data for a location
func (sc *SilenusClient) GetTerrain(ctx context.Context, lat, lon float64, radius float64) (*TerrainData, error) {
	path := fmt.Sprintf("/api/v1/terrain?lat=%f&lon=%f&radius=%f", lat, lon, radius)
	resp, err := sc.get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var terrain TerrainData
	if err := json.NewDecoder(resp.Body).Decode(&terrain); err != nil {
		return nil, err
	}
	return &terrain, nil
}

func (sc *SilenusClient) get(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", sc.config.BaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	return sc.client.Do(req)
}

// WeatherData represents weather information
type WeatherData struct {
	Temperature   float64   `json:"temperature"`
	Humidity      float64   `json:"humidity"`
	WindSpeed     float64   `json:"wind_speed"`
	WindDirection float64   `json:"wind_direction"`
	Visibility    float64   `json:"visibility"`
	Ceiling       float64   `json:"ceiling"`
	Precipitation float64   `json:"precipitation"`
	Conditions    string    `json:"conditions"`
	Timestamp     time.Time `json:"timestamp"`
}

// TerrainData represents terrain information
type TerrainData struct {
	Elevation float64    `json:"elevation"`
	Slope     float64    `json:"slope"`
	Type      string     `json:"type"`
	Obstacles []Obstacle `json:"obstacles"`
}

// Obstacle represents a terrain obstacle
type Obstacle struct {
	Type     string     `json:"type"`
	Position [3]float64 `json:"position"`
	Size     [3]float64 `json:"size"`
}

// --------- SatNet Client Methods ---------

// Ping checks if SatNet is reachable
func (snc *SatNetClient) Ping(ctx context.Context) bool {
	resp, err := snc.get(ctx, "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// SendMessage sends a message via DTN
func (snc *SatNetClient) SendMessage(ctx context.Context, dest string, payload []byte) error {
	data := map[string]interface{}{
		"destination": dest,
		"payload":     payload,
		"priority":    "high",
	}

	resp, err := snc.post(ctx, "/api/v1/messages", data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("message send failed: status %d", resp.StatusCode)
	}
	return nil
}

// GetSatelliteStatus retrieves satellite constellation status
func (snc *SatNetClient) GetSatelliteStatus(ctx context.Context) ([]SatelliteInfo, error) {
	resp, err := snc.get(ctx, "/api/v1/satellites")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var satellites []SatelliteInfo
	if err := json.NewDecoder(resp.Body).Decode(&satellites); err != nil {
		return nil, err
	}
	return satellites, nil
}

func (snc *SatNetClient) get(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", snc.config.BaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	return snc.client.Do(req)
}

func (snc *SatNetClient) post(ctx context.Context, path string, payload interface{}) (*http.Response, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", snc.config.BaseURL+path, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return snc.client.Do(req)
}

// SatelliteInfo represents satellite status
type SatelliteInfo struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Status    string     `json:"status"`
	Position  [3]float64 `json:"position"`
	Visible   bool       `json:"visible"`
	Elevation float64    `json:"elevation"`
}

// --------- Giru Client Methods ---------

// Ping checks if Giru is reachable
func (gc *GiruClient) Ping(ctx context.Context) bool {
	resp, err := gc.get(ctx, "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// ReportAnomaly sends an anomaly report to Giru
func (gc *GiruClient) ReportAnomaly(ctx context.Context, anomaly interface{}) error {
	resp, err := gc.post(ctx, "/api/v1/anomalies", anomaly)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// GetThreatIntel retrieves current threat intelligence
func (gc *GiruClient) GetThreatIntel(ctx context.Context) ([]ThreatIntel, error) {
	resp, err := gc.get(ctx, "/api/v1/threats")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var threats []ThreatIntel
	if err := json.NewDecoder(resp.Body).Decode(&threats); err != nil {
		return nil, err
	}
	return threats, nil
}

func (gc *GiruClient) get(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", gc.config.BaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	return gc.client.Do(req)
}

func (gc *GiruClient) post(ctx context.Context, path string, payload interface{}) (*http.Response, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", gc.config.BaseURL+path, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return gc.client.Do(req)
}

// ThreatIntel represents threat intelligence data
type ThreatIntel struct {
	ID       string     `json:"id"`
	Type     string     `json:"type"`
	Severity string     `json:"severity"`
	Location [3]float64 `json:"location,omitempty"`
	Range    float64    `json:"range,omitempty"`
	Active   bool       `json:"active"`
	Updated  time.Time  `json:"updated"`
}

// --------- Hunoid Client Methods ---------

// Ping checks if Hunoid is reachable
func (hc *HunoidClient) Ping(ctx context.Context) bool {
	resp, err := hc.get(ctx, "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// SendRobotCommand sends a command to a robot
func (hc *HunoidClient) SendRobotCommand(ctx context.Context, robotID string, cmd interface{}) error {
	resp, err := hc.post(ctx, "/api/v1/robots/"+robotID+"/command", cmd)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (hc *HunoidClient) get(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", hc.config.BaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	return hc.client.Do(req)
}

func (hc *HunoidClient) post(ctx context.Context, path string, payload interface{}) (*http.Response, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", hc.config.BaseURL+path, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return hc.client.Do(req)
}
