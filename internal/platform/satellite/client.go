// Package satellite provides integration with open-source satellite tracking APIs.
// Supports N2YO for real-time tracking and CelesTrak for TLE data.
package satellite

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Client provides satellite tracking capabilities via external APIs.
type Client struct {
	httpClient   *http.Client
	n2yoAPIKey   string
	celestrakURL string
	tleAPIURL    string
	cache        *tleCache
	mu           sync.RWMutex
}

// Config holds satellite client configuration.
type Config struct {
	// N2YO API key (get free at n2yo.com/api)
	N2YOAPIKey string
	// CelesTrak base URL (default: https://celestrak.org)
	CelesTrakURL string
	// TLE API URL (default: https://tle.ivanstanojevic.me)
	TLEAPIURL string
	// HTTP timeout
	Timeout time.Duration
	// TLE cache TTL
	CacheTTL time.Duration
}

// DefaultConfig returns sensible defaults for satellite tracking.
func DefaultConfig() Config {
	return Config{
		CelesTrakURL: "https://celestrak.org",
		TLEAPIURL:    "https://tle.ivanstanojevic.me",
		Timeout:      30 * time.Second,
		CacheTTL:     1 * time.Hour,
	}
}

// NewClient creates a new satellite tracking client.
func NewClient(cfg Config) *Client {
	if cfg.CelesTrakURL == "" {
		cfg.CelesTrakURL = "https://celestrak.org"
	}
	if cfg.TLEAPIURL == "" {
		cfg.TLEAPIURL = "https://tle.ivanstanojevic.me"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.CacheTTL == 0 {
		cfg.CacheTTL = 1 * time.Hour
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		n2yoAPIKey:   cfg.N2YOAPIKey,
		celestrakURL: cfg.CelesTrakURL,
		tleAPIURL:    cfg.TLEAPIURL,
		cache:        newTLECache(cfg.CacheTTL),
	}
}

// TLE represents Two-Line Element set for satellite orbit prediction.
type TLE struct {
	SatelliteID int       `json:"satelliteId"`
	Name        string    `json:"name"`
	Line1       string    `json:"line1"`
	Line2       string    `json:"line2"`
	Epoch       time.Time `json:"epoch,omitempty"`
	RetrievedAt time.Time `json:"retrievedAt"`
	Source      string    `json:"source"`
}

// Position represents a satellite's position at a given time.
type Position struct {
	SatelliteID int       `json:"satelliteId"`
	Name        string    `json:"name"`
	Latitude    float64   `json:"satlatitude"`
	Longitude   float64   `json:"satlongitude"`
	Altitude    float64   `json:"sataltitude"` // km
	Azimuth     float64   `json:"azimuth"`     // degrees from observer
	Elevation   float64   `json:"elevation"`   // degrees from observer
	RA          float64   `json:"ra"`          // right ascension
	Dec         float64   `json:"dec"`         // declination
	Timestamp   time.Time `json:"timestamp"`
	Eclipsed    bool      `json:"eclipsed"`
}

// VisualPass represents a predicted visible satellite pass.
type VisualPass struct {
	StartTime      time.Time `json:"startTime"`
	StartAzimuth   float64   `json:"startAz"`
	StartElevation float64   `json:"startEl"`
	MaxTime        time.Time `json:"maxTime"`
	MaxAzimuth     float64   `json:"maxAz"`
	MaxElevation   float64   `json:"maxEl"`
	EndTime        time.Time `json:"endTime"`
	EndAzimuth     float64   `json:"endAz"`
	EndElevation   float64   `json:"endEl"`
	Magnitude      float64   `json:"mag"`
	Duration       int       `json:"duration"` // seconds
}

// SatelliteAbove represents a satellite currently above a location.
type SatelliteAbove struct {
	SatelliteID   int     `json:"satid"`
	Name          string  `json:"satname"`
	IntDesignator string  `json:"intDesignator"`
	LaunchDate    string  `json:"launchDate"`
	Latitude      float64 `json:"satlat"`
	Longitude     float64 `json:"satlng"`
	Altitude      float64 `json:"satalt"` // km
}

// Observer represents a ground observer's location.
type Observer struct {
	Latitude  float64 // degrees
	Longitude float64 // degrees
	Altitude  float64 // meters above sea level
}

// Common NORAD IDs for testing
const (
	NoradISS           = 25544 // International Space Station
	NoradHubble        = 20580 // Hubble Space Telescope
	NoradTIROS1        = 29    // First weather satellite
	NoradLandsat8      = 39084 // Landsat 8
	NoradTerra         = 25994 // Terra (EOS AM-1)
	NoradAqua          = 27424 // Aqua (EOS PM-1)
	NoradNOAA19        = 33591 // NOAA-19
	NoradStarlinkFirst = 44235 // First operational Starlink
)

// GetTLE fetches TLE data for a satellite by NORAD ID.
// Uses TLE API as primary with CelesTrak as fallback, both free and no key required.
func (c *Client) GetTLE(ctx context.Context, noradID int) (*TLE, error) {
	// Check cache first
	if cached := c.cache.get(noradID); cached != nil {
		return cached, nil
	}

	// Try TLE API first
	tle, err := c.getTLEFromAPI(ctx, noradID)
	if err == nil {
		c.cache.set(noradID, tle)
		return tle, nil
	}

	// Fallback to CelesTrak
	tle, err2 := c.getTLEFromCelesTrak(ctx, noradID)
	if err2 == nil {
		c.cache.set(noradID, tle)
		return tle, nil
	}

	return nil, fmt.Errorf("all TLE sources failed: primary: %v, fallback: %v", err, err2)
}

func (c *Client) getTLEFromAPI(ctx context.Context, noradID int) (*TLE, error) {
	url := fmt.Sprintf("%s/api/tle/%d", c.tleAPIURL, noradID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "ASGARD-Satellite-Tracker/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch TLE: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("TLE API error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		SatelliteID int    `json:"satelliteId"`
		Name        string `json:"name"`
		Line1       string `json:"line1"`
		Line2       string `json:"line2"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode TLE: %w", err)
	}

	return &TLE{
		SatelliteID: result.SatelliteID,
		Name:        result.Name,
		Line1:       result.Line1,
		Line2:       result.Line2,
		RetrievedAt: time.Now().UTC(),
		Source:      "tle-api",
	}, nil
}

func (c *Client) getTLEFromCelesTrak(ctx context.Context, noradID int) (*TLE, error) {
	// CelesTrak GP data API (JSON format)
	url := fmt.Sprintf("%s/NORAD/elements/gp.php?CATNR=%d&FORMAT=JSON", c.celestrakURL, noradID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "ASGARD-Satellite-Tracker/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch from CelesTrak: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("CelesTrak error %d: %s", resp.StatusCode, string(body))
	}

	var results []struct {
		ObjectName string `json:"OBJECT_NAME"`
		ObjectID   string `json:"OBJECT_ID"`
		NoradCatID int    `json:"NORAD_CAT_ID"`
		TLELine1   string `json:"TLE_LINE1"`
		TLELine2   string `json:"TLE_LINE2"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("decode CelesTrak: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no TLE found for NORAD %d", noradID)
	}

	r := results[0]
	return &TLE{
		SatelliteID: r.NoradCatID,
		Name:        r.ObjectName,
		Line1:       r.TLELine1,
		Line2:       r.TLELine2,
		RetrievedAt: time.Now().UTC(),
		Source:      "celestrak",
	}, nil
}

// GetPosition fetches current/predicted positions for a satellite.
// Requires N2YO API key. Returns positions for the next `seconds` seconds.
func (c *Client) GetPosition(ctx context.Context, noradID int, observer Observer, seconds int) ([]Position, error) {
	if c.n2yoAPIKey == "" {
		return nil, fmt.Errorf("N2YO API key required for position tracking")
	}
	if seconds < 1 || seconds > 300 {
		seconds = 60
	}

	url := fmt.Sprintf(
		"https://api.n2yo.com/rest/v1/satellite/positions/%d/%.6f/%.6f/%.0f/%d&apiKey=%s",
		noradID,
		observer.Latitude,
		observer.Longitude,
		observer.Altitude,
		seconds,
		c.n2yoAPIKey,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch positions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("N2YO API error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Info struct {
			SatID   int    `json:"satid"`
			SatName string `json:"satname"`
		} `json:"info"`
		Positions []struct {
			Satlatitude  float64 `json:"satlatitude"`
			Satlongitude float64 `json:"satlongitude"`
			Sataltitude  float64 `json:"sataltitude"`
			Azimuth      float64 `json:"azimuth"`
			Elevation    float64 `json:"elevation"`
			RA           float64 `json:"ra"`
			Dec          float64 `json:"dec"`
			Timestamp    int64   `json:"timestamp"`
			Eclipsed     bool    `json:"eclipsed"`
		} `json:"positions"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode positions: %w", err)
	}

	positions := make([]Position, 0, len(result.Positions))
	for _, p := range result.Positions {
		positions = append(positions, Position{
			SatelliteID: result.Info.SatID,
			Name:        result.Info.SatName,
			Latitude:    p.Satlatitude,
			Longitude:   p.Satlongitude,
			Altitude:    p.Sataltitude,
			Azimuth:     p.Azimuth,
			Elevation:   p.Elevation,
			RA:          p.RA,
			Dec:         p.Dec,
			Timestamp:   time.Unix(p.Timestamp, 0).UTC(),
			Eclipsed:    p.Eclipsed,
		})
	}

	return positions, nil
}

// GetVisualPasses fetches upcoming visible passes for a satellite.
// Requires N2YO API key.
func (c *Client) GetVisualPasses(ctx context.Context, noradID int, observer Observer, days int, minVisibility int) ([]VisualPass, error) {
	if c.n2yoAPIKey == "" {
		return nil, fmt.Errorf("N2YO API key required for pass predictions")
	}
	if days < 1 || days > 10 {
		days = 5
	}
	if minVisibility < 1 {
		minVisibility = 60
	}

	url := fmt.Sprintf(
		"https://api.n2yo.com/rest/v1/satellite/visualpasses/%d/%.6f/%.6f/%.0f/%d/%d&apiKey=%s",
		noradID,
		observer.Latitude,
		observer.Longitude,
		observer.Altitude,
		days,
		minVisibility,
		c.n2yoAPIKey,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch passes: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("N2YO API error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Passes []struct {
			StartUTC int64   `json:"startUTC"`
			StartAz  float64 `json:"startAz"`
			StartEl  float64 `json:"startEl"`
			MaxUTC   int64   `json:"maxUTC"`
			MaxAz    float64 `json:"maxAz"`
			MaxEl    float64 `json:"maxEl"`
			EndUTC   int64   `json:"endUTC"`
			EndAz    float64 `json:"endAz"`
			EndEl    float64 `json:"endEl"`
			Mag      float64 `json:"mag"`
			Duration int     `json:"duration"`
		} `json:"passes"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode passes: %w", err)
	}

	passes := make([]VisualPass, 0, len(result.Passes))
	for _, p := range result.Passes {
		passes = append(passes, VisualPass{
			StartTime:      time.Unix(p.StartUTC, 0).UTC(),
			StartAzimuth:   p.StartAz,
			StartElevation: p.StartEl,
			MaxTime:        time.Unix(p.MaxUTC, 0).UTC(),
			MaxAzimuth:     p.MaxAz,
			MaxElevation:   p.MaxEl,
			EndTime:        time.Unix(p.EndUTC, 0).UTC(),
			EndAzimuth:     p.EndAz,
			EndElevation:   p.EndEl,
			Magnitude:      p.Mag,
			Duration:       p.Duration,
		})
	}

	return passes, nil
}

// GetSatellitesAbove fetches satellites currently above a location.
// Requires N2YO API key.
func (c *Client) GetSatellitesAbove(ctx context.Context, observer Observer, searchRadius int, categoryID int) ([]SatelliteAbove, error) {
	if c.n2yoAPIKey == "" {
		return nil, fmt.Errorf("N2YO API key required")
	}
	if searchRadius < 0 || searchRadius > 90 {
		searchRadius = 70
	}

	url := fmt.Sprintf(
		"https://api.n2yo.com/rest/v1/satellite/above/%.6f/%.6f/%.0f/%d/%d&apiKey=%s",
		observer.Latitude,
		observer.Longitude,
		observer.Altitude,
		searchRadius,
		categoryID,
		c.n2yoAPIKey,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch satellites above: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("N2YO API error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Above []SatelliteAbove `json:"above"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return result.Above, nil
}

// tleCache provides thread-safe TLE caching.
type tleCache struct {
	mu      sync.RWMutex
	entries map[int]*tleCacheEntry
	ttl     time.Duration
}

type tleCacheEntry struct {
	tle       *TLE
	expiresAt time.Time
}

func newTLECache(ttl time.Duration) *tleCache {
	return &tleCache{
		entries: make(map[int]*tleCacheEntry),
		ttl:     ttl,
	}
}

func (c *tleCache) get(noradID int) *TLE {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[noradID]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil
	}
	return entry.tle
}

func (c *tleCache) set(noradID int, tle *TLE) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[noradID] = &tleCacheEntry{
		tle:       tle,
		expiresAt: time.Now().Add(c.ttl),
	}
}
