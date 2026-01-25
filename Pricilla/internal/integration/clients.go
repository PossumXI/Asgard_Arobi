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

	"github.com/gorilla/websocket"
)

// ============================================================================
// REAL HTTP CLIENT IMPLEMENTATIONS
// ============================================================================

// HTTPSilenusClient implements SilenusClient using real HTTP/WebSocket connections
type HTTPSilenusClient struct {
	mu         sync.RWMutex
	baseURL    string
	wsURL      string
	apiKey     string
	httpClient *http.Client
	wsConn     *websocket.Conn
}

// NewHTTPSilenusClient creates a new Silenus HTTP client
func NewHTTPSilenusClient(baseURL, wsURL, apiKey string) *HTTPSilenusClient {
	return &HTTPSilenusClient{
		baseURL: baseURL,
		wsURL:   wsURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

func (c *HTTPSilenusClient) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	url := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	return c.httpClient.Do(req)
}

func (c *HTTPSilenusClient) GetLatestFrame(ctx context.Context, satelliteID string) ([]byte, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/api/satellites/%s/frame", satelliteID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get frame: status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (c *HTTPSilenusClient) RequestTerrainMap(ctx context.Context, region GeoCoord, radiusKm float64) (*TerrainMap, error) {
	payload := map[string]interface{}{
		"latitude":  region.Latitude,
		"longitude": region.Longitude,
		"radiusKm":  radiusKm,
	}

	resp, err := c.doRequest(ctx, "POST", "/api/terrain/request", payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to request terrain: %s", string(body))
	}

	var terrain TerrainMap
	if err := json.NewDecoder(resp.Body).Decode(&terrain); err != nil {
		return nil, err
	}

	return &terrain, nil
}

func (c *HTTPSilenusClient) GetSatellitePosition(ctx context.Context, satelliteID string) (*SatellitePosition, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/api/satellites/%s/position", satelliteID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get position: status %d", resp.StatusCode)
	}

	var pos SatellitePosition
	if err := json.NewDecoder(resp.Body).Decode(&pos); err != nil {
		return nil, err
	}

	return &pos, nil
}

func (c *HTTPSilenusClient) GetAllSatellitePositions(ctx context.Context) ([]SatellitePosition, error) {
	resp, err := c.doRequest(ctx, "GET", "/api/satellites/positions", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get positions: status %d", resp.StatusCode)
	}

	var result struct {
		Satellites []SatellitePosition `json:"satellites"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Satellites, nil
}

func (c *HTTPSilenusClient) SubscribeAlerts(ctx context.Context) (<-chan Alert, error) {
	alertChan := make(chan Alert, 100)

	// Connect to WebSocket
	wsURL := c.wsURL + "/ws/alerts"
	
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}
	
	headers := http.Header{}
	if c.apiKey != "" {
		headers.Set("Authorization", "Bearer "+c.apiKey)
	}

	conn, _, err := dialer.DialContext(ctx, wsURL, headers)
	if err != nil {
		close(alertChan)
		return alertChan, fmt.Errorf("failed to connect to alerts WebSocket: %w", err)
	}

	c.mu.Lock()
	c.wsConn = conn
	c.mu.Unlock()

	go func() {
		defer close(alertChan)
		defer conn.Close()

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			var alert Alert
			if err := conn.ReadJSON(&alert); err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					return
				}
				continue
			}

			select {
			case alertChan <- alert:
			case <-ctx.Done():
				return
			default:
				// Drop if buffer full
			}
		}
	}()

	return alertChan, nil
}

func (c *HTTPSilenusClient) GetActiveAlerts(ctx context.Context) ([]Alert, error) {
	resp, err := c.doRequest(ctx, "GET", "/api/alerts?status=active", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get alerts: status %d", resp.StatusCode)
	}

	var result struct {
		Alerts []Alert `json:"alerts"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Alerts, nil
}

func (c *HTTPSilenusClient) GetSatelliteTelemetry(ctx context.Context, satelliteID string) (*SatelliteTelemetry, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/api/satellites/%s/telemetry", satelliteID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get telemetry: status %d", resp.StatusCode)
	}

	var telem SatelliteTelemetry
	if err := json.NewDecoder(resp.Body).Decode(&telem); err != nil {
		return nil, err
	}

	return &telem, nil
}

// ============================================================================
// HTTPHunoidClient - Real Hunoid HTTP Client
// ============================================================================

// HTTPHunoidClient implements HunoidClient using real HTTP connections
type HTTPHunoidClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewHTTPHunoidClient creates a new Hunoid HTTP client
func NewHTTPHunoidClient(baseURL, apiKey string) *HTTPHunoidClient {
	return &HTTPHunoidClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *HTTPHunoidClient) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	url := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	return c.httpClient.Do(req)
}

func (c *HTTPHunoidClient) SendCommand(ctx context.Context, hunoidID string, command HunoidCommand) error {
	resp, err := c.doRequest(ctx, "POST", fmt.Sprintf("/api/hunoids/%s/commands", hunoidID), command)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("command failed: %s", string(body))
	}

	return nil
}

func (c *HTTPHunoidClient) NavigateTo(ctx context.Context, hunoidID string, destination Vector3D) error {
	payload := map[string]interface{}{
		"type": "navigate",
		"target": map[string]float64{
			"x": destination.X,
			"y": destination.Y,
			"z": destination.Z,
		},
	}

	resp, err := c.doRequest(ctx, "POST", fmt.Sprintf("/api/hunoids/%s/navigate", hunoidID), payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("navigation failed: %s", string(body))
	}

	return nil
}

func (c *HTTPHunoidClient) ExecuteAction(ctx context.Context, hunoidID string, action string, params map[string]interface{}) error {
	payload := map[string]interface{}{
		"action": action,
		"params": params,
	}

	resp, err := c.doRequest(ctx, "POST", fmt.Sprintf("/api/hunoids/%s/actions", hunoidID), payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("action failed: %s", string(body))
	}

	return nil
}

func (c *HTTPHunoidClient) GetHunoidState(ctx context.Context, hunoidID string) (*HunoidState, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/api/hunoids/%s/state", hunoidID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get state: status %d", resp.StatusCode)
	}

	var state HunoidState
	if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
		return nil, err
	}

	return &state, nil
}

func (c *HTTPHunoidClient) GetAllHunoidStates(ctx context.Context) ([]HunoidState, error) {
	resp, err := c.doRequest(ctx, "GET", "/api/hunoids/states", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get states: status %d", resp.StatusCode)
	}

	var result struct {
		Hunoids []HunoidState `json:"hunoids"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Hunoids, nil
}

func (c *HTTPHunoidClient) AssignMission(ctx context.Context, hunoidID string, mission Mission) error {
	resp, err := c.doRequest(ctx, "POST", fmt.Sprintf("/api/hunoids/%s/mission", hunoidID), mission)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("mission assignment failed: %s", string(body))
	}

	return nil
}

func (c *HTTPHunoidClient) AbortMission(ctx context.Context, hunoidID string) error {
	resp, err := c.doRequest(ctx, "POST", fmt.Sprintf("/api/hunoids/%s/mission/abort", hunoidID), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("mission abort failed: %s", string(body))
	}

	return nil
}

// ============================================================================
// HTTPSatNetClient - Real Sat_Net DTN Client
// ============================================================================

// HTTPSatNetClient implements SatNetClient using real HTTP connections
type HTTPSatNetClient struct {
	baseURL    string
	wsURL      string
	apiKey     string
	httpClient *http.Client
}

// NewHTTPSatNetClient creates a new Sat_Net HTTP client
func NewHTTPSatNetClient(baseURL, wsURL, apiKey string) *HTTPSatNetClient {
	return &HTTPSatNetClient{
		baseURL: baseURL,
		wsURL:   wsURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second, // DTN operations can be slow
		},
	}
}

func (c *HTTPSatNetClient) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	url := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	return c.httpClient.Do(req)
}

func (c *HTTPSatNetClient) SendBundle(ctx context.Context, destination string, payload []byte, priority int) error {
	body := map[string]interface{}{
		"destination": destination,
		"payload":     payload,
		"priority":    priority,
	}

	resp, err := c.doRequest(ctx, "POST", "/api/bundles", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("send bundle failed: %s", string(respBody))
	}

	return nil
}

func (c *HTTPSatNetClient) ReceiveBundles(ctx context.Context) (<-chan Bundle, error) {
	bundleChan := make(chan Bundle, 100)

	wsURL := c.wsURL + "/ws/bundles"
	
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}
	
	headers := http.Header{}
	if c.apiKey != "" {
		headers.Set("Authorization", "Bearer "+c.apiKey)
	}

	conn, _, err := dialer.DialContext(ctx, wsURL, headers)
	if err != nil {
		close(bundleChan)
		return bundleChan, fmt.Errorf("failed to connect to bundles WebSocket: %w", err)
	}

	go func() {
		defer close(bundleChan)
		defer conn.Close()

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			var bundle Bundle
			if err := conn.ReadJSON(&bundle); err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					return
				}
				continue
			}

			select {
			case bundleChan <- bundle:
			case <-ctx.Done():
				return
			default:
			}
		}
	}()

	return bundleChan, nil
}

func (c *HTTPSatNetClient) SendCommand(ctx context.Context, payloadID string, command Command) error {
	resp, err := c.doRequest(ctx, "POST", fmt.Sprintf("/api/payloads/%s/command", payloadID), command)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("send command failed: %s", string(body))
	}

	return nil
}

func (c *HTTPSatNetClient) SendTrajectory(ctx context.Context, payloadID string, trajectory []Waypoint) error {
	payload := map[string]interface{}{
		"waypoints": trajectory,
	}

	resp, err := c.doRequest(ctx, "POST", fmt.Sprintf("/api/payloads/%s/trajectory", payloadID), payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("send trajectory failed: %s", string(body))
	}

	return nil
}

func (c *HTTPSatNetClient) GetTelemetry(ctx context.Context, payloadID string) (*Telemetry, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/api/payloads/%s/telemetry", payloadID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get telemetry: status %d", resp.StatusCode)
	}

	var telem Telemetry
	if err := json.NewDecoder(resp.Body).Decode(&telem); err != nil {
		return nil, err
	}

	return &telem, nil
}

func (c *HTTPSatNetClient) SubscribeTelemetry(ctx context.Context, payloadID string) (<-chan Telemetry, error) {
	telemChan := make(chan Telemetry, 100)

	wsURL := c.wsURL + fmt.Sprintf("/ws/telemetry/%s", payloadID)
	
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}
	
	headers := http.Header{}
	if c.apiKey != "" {
		headers.Set("Authorization", "Bearer "+c.apiKey)
	}

	conn, _, err := dialer.DialContext(ctx, wsURL, headers)
	if err != nil {
		close(telemChan)
		return telemChan, fmt.Errorf("failed to connect to telemetry WebSocket: %w", err)
	}

	go func() {
		defer close(telemChan)
		defer conn.Close()

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			var telem Telemetry
			if err := conn.ReadJSON(&telem); err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					return
				}
				continue
			}

			select {
			case telemChan <- telem:
			case <-ctx.Done():
				return
			default:
			}
		}
	}()

	return telemChan, nil
}

func (c *HTTPSatNetClient) GetContactWindows(ctx context.Context, satelliteID string, horizon time.Duration) ([]ContactWindow, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/api/satellites/%s/contacts?horizon=%s", satelliteID, horizon), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get contacts: status %d", resp.StatusCode)
	}

	var result struct {
		Windows []ContactWindow `json:"windows"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Windows, nil
}

// ============================================================================
// HTTPGiruClient - Real Giru Security Client
// ============================================================================

// HTTPGiruClient implements GiruClient using real HTTP connections
type HTTPGiruClient struct {
	baseURL    string
	wsURL      string
	apiKey     string
	httpClient *http.Client
}

// NewHTTPGiruClient creates a new Giru HTTP client
func NewHTTPGiruClient(baseURL, wsURL, apiKey string) *HTTPGiruClient {
	return &HTTPGiruClient{
		baseURL: baseURL,
		wsURL:   wsURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *HTTPGiruClient) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	url := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	return c.httpClient.Do(req)
}

func (c *HTTPGiruClient) GetActiveThreats(ctx context.Context) ([]Threat, error) {
	resp, err := c.doRequest(ctx, "GET", "/api/threats?status=active", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get threats: status %d", resp.StatusCode)
	}

	var result struct {
		Threats []Threat `json:"threats"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Threats, nil
}

func (c *HTTPGiruClient) SubscribeThreats(ctx context.Context) (<-chan Threat, error) {
	threatChan := make(chan Threat, 100)

	wsURL := c.wsURL + "/ws/threats"
	
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}
	
	headers := http.Header{}
	if c.apiKey != "" {
		headers.Set("Authorization", "Bearer "+c.apiKey)
	}

	conn, _, err := dialer.DialContext(ctx, wsURL, headers)
	if err != nil {
		close(threatChan)
		return threatChan, fmt.Errorf("failed to connect to threats WebSocket: %w", err)
	}

	go func() {
		defer close(threatChan)
		defer conn.Close()

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			var threat Threat
			if err := conn.ReadJSON(&threat); err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					return
				}
				continue
			}

			select {
			case threatChan <- threat:
			case <-ctx.Done():
				return
			default:
			}
		}
	}()

	return threatChan, nil
}

func (c *HTTPGiruClient) GetThreatZones(ctx context.Context) ([]ThreatZone, error) {
	resp, err := c.doRequest(ctx, "GET", "/api/threat-zones", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get threat zones: status %d", resp.StatusCode)
	}

	var result struct {
		Zones []ThreatZone `json:"zones"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Zones, nil
}

func (c *HTTPGiruClient) RequestSecurityScan(ctx context.Context, target string) (*SecurityScanResult, error) {
	payload := map[string]string{"target": target}

	resp, err := c.doRequest(ctx, "POST", "/api/scans", payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("scan request failed: %s", string(body))
	}

	var result SecurityScanResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *HTTPGiruClient) ReportAnomaly(ctx context.Context, anomaly Anomaly) error {
	resp, err := c.doRequest(ctx, "POST", "/api/anomalies", anomaly)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("report anomaly failed: %s", string(body))
	}

	return nil
}

// ============================================================================
// HTTPNysusClient - Real Nysus Mission Client
// ============================================================================

// HTTPNysusClient implements NysusClient using real HTTP connections
type HTTPNysusClient struct {
	baseURL    string
	wsURL      string
	apiKey     string
	httpClient *http.Client
}

// NewHTTPNysusClient creates a new Nysus HTTP client
func NewHTTPNysusClient(baseURL, wsURL, apiKey string) *HTTPNysusClient {
	return &HTTPNysusClient{
		baseURL: baseURL,
		wsURL:   wsURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *HTTPNysusClient) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	url := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	return c.httpClient.Do(req)
}

func (c *HTTPNysusClient) CreateMission(ctx context.Context, mission Mission) (string, error) {
	resp, err := c.doRequest(ctx, "POST", "/api/missions", mission)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("create mission failed: %s", string(body))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.ID, nil
}

func (c *HTTPNysusClient) GetMission(ctx context.Context, missionID string) (*Mission, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/api/missions/%s", missionID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("mission not found")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get mission: status %d", resp.StatusCode)
	}

	var mission Mission
	if err := json.NewDecoder(resp.Body).Decode(&mission); err != nil {
		return nil, err
	}

	return &mission, nil
}

func (c *HTTPNysusClient) GetActiveMissions(ctx context.Context) ([]Mission, error) {
	resp, err := c.doRequest(ctx, "GET", "/api/missions?status=active,pending", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get missions: status %d", resp.StatusCode)
	}

	var result struct {
		Missions []Mission `json:"missions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Missions, nil
}

func (c *HTTPNysusClient) UpdateMissionStatus(ctx context.Context, missionID string, status string) error {
	payload := map[string]string{"status": status}

	resp, err := c.doRequest(ctx, "PATCH", fmt.Sprintf("/api/missions/%s", missionID), payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update mission failed: %s", string(body))
	}

	return nil
}

func (c *HTTPNysusClient) PublishEvent(ctx context.Context, event Event) error {
	resp, err := c.doRequest(ctx, "POST", "/api/events", event)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("publish event failed: %s", string(body))
	}

	return nil
}

func (c *HTTPNysusClient) SubscribeEvents(ctx context.Context, eventTypes []string) (<-chan Event, error) {
	eventChan := make(chan Event, 100)

	// Build query string for event types
	query := ""
	for i, t := range eventTypes {
		if i > 0 {
			query += ","
		}
		query += t
	}

	wsURL := c.wsURL + "/ws/events"
	if query != "" {
		wsURL += "?types=" + query
	}
	
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}
	
	headers := http.Header{}
	if c.apiKey != "" {
		headers.Set("Authorization", "Bearer "+c.apiKey)
	}

	conn, _, err := dialer.DialContext(ctx, wsURL, headers)
	if err != nil {
		close(eventChan)
		return eventChan, fmt.Errorf("failed to connect to events WebSocket: %w", err)
	}

	go func() {
		defer close(eventChan)
		defer conn.Close()

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			var event Event
			if err := conn.ReadJSON(&event); err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					return
				}
				continue
			}

			select {
			case eventChan <- event:
			case <-ctx.Done():
				return
			default:
			}
		}
	}()

	return eventChan, nil
}

func (c *HTTPNysusClient) GetDashboardStats(ctx context.Context) (*DashboardStats, error) {
	resp, err := c.doRequest(ctx, "GET", "/api/dashboard/stats", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get dashboard stats: status %d", resp.StatusCode)
	}

	var stats DashboardStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, err
	}

	return &stats, nil
}

// ============================================================================
// Factory Functions for Creating Real Clients
// ============================================================================

// CreateRealClients creates real HTTP clients for all ASGARD services
func CreateRealClients(config IntegrationConfig) (SilenusClient, HunoidClient, SatNetClient, GiruClient, NysusClient) {
	var silenusClient SilenusClient
	var hunoidClient HunoidClient
	var satnetClient SatNetClient
	var giruClient GiruClient
	var nysusClient NysusClient

	// Create real clients based on configuration
	if config.SilenusEndpoint != "" {
		wsURL := "ws" + config.SilenusEndpoint[4:] // Convert http to ws
		silenusClient = NewHTTPSilenusClient(config.SilenusEndpoint, wsURL, "")
	}

	if config.HunoidEndpoint != "" {
		hunoidClient = NewHTTPHunoidClient(config.HunoidEndpoint, "")
	}

	if config.SatNetEndpoint != "" {
		wsURL := "ws" + config.SatNetEndpoint[4:]
		satnetClient = NewHTTPSatNetClient(config.SatNetEndpoint, wsURL, "")
	}

	if config.GiruEndpoint != "" {
		wsURL := "ws" + config.GiruEndpoint[4:]
		giruClient = NewHTTPGiruClient(config.GiruEndpoint, wsURL, "")
	}

	if config.NysusEndpoint != "" {
		wsURL := "ws" + config.NysusEndpoint[4:]
		nysusClient = NewHTTPNysusClient(config.NysusEndpoint, wsURL, "")
	}

	return silenusClient, hunoidClient, satnetClient, giruClient, nysusClient
}
