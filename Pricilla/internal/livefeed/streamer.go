package livefeed

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// StreamType defines the type of live feed
type StreamType string

const (
	StreamTelemetry StreamType = "telemetry" // Position, velocity, status
	StreamVideo     StreamType = "video"     // Live video feed
	StreamThermal   StreamType = "thermal"   // Thermal imaging
	StreamRadar     StreamType = "radar"     // Radar overlay
	StreamMap       StreamType = "map"       // 3D map visualization
	StreamCommand   StreamType = "command"   // Command feed
	StreamAlert     StreamType = "alert"     // Alert notifications
)

// ClearanceLevel defines access tiers
type ClearanceLevel int

const (
	ClearancePublic   ClearanceLevel = 0 // Public access - basic info only
	ClearanceCivilian ClearanceLevel = 1 // Civilian - humanitarian missions
	ClearanceMilitary ClearanceLevel = 2 // Military - tactical missions
	ClearanceGov      ClearanceLevel = 3 // Government - classified missions
	ClearanceSecret   ClearanceLevel = 4 // Secret - top secret missions
	ClearanceUltra    ClearanceLevel = 5 // Ultra - highest classification
)

// Vector3D represents 3D coordinates
type Vector3D struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// GeoCoord represents geographic coordinates
type GeoCoord struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Altitude  float64 `json:"altitude"`
}

// LiveFeed represents a live data stream
type LiveFeed struct {
	ID            string         `json:"id"`
	MissionID     string         `json:"missionId"`
	PayloadID     string         `json:"payloadId"`
	PayloadType   string         `json:"payloadType"`
	StreamType    StreamType     `json:"streamType"`
	Clearance     ClearanceLevel `json:"clearance"`
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	Status        string         `json:"status"` // active, paused, ended
	ViewerCount   int            `json:"viewerCount"`
	StartedAt     time.Time      `json:"startedAt"`
	Quality       string         `json:"quality"` // 4k, 1080p, 720p, 480p, audio_only
	Encrypted     bool           `json:"encrypted"`
	RecordingPath string         `json:"recordingPath,omitempty"`
}

// TelemetryFrame represents a single telemetry update
type TelemetryFrame struct {
	ID           string    `json:"id"`
	FeedID       string    `json:"feedId"`
	PayloadID    string    `json:"payloadId"`
	Timestamp    time.Time `json:"timestamp"`
	Position     Vector3D  `json:"position"`
	GeoPosition  *GeoCoord `json:"geoPosition,omitempty"`
	Velocity     Vector3D  `json:"velocity"`
	Heading      float64   `json:"heading"`
	Altitude     float64   `json:"altitude"`
	Speed        float64   `json:"speed"`
	Fuel         float64   `json:"fuel"`
	Battery      float64   `json:"battery"`
	Status       string    `json:"status"`
	MissionPhase string    `json:"missionPhase"`
	ETA          string    `json:"eta,omitempty"`
	Distance     float64   `json:"distanceRemaining"`
	Warnings     []string  `json:"warnings,omitempty"`
}

// VideoFrame represents a video frame
type VideoFrame struct {
	ID        string    `json:"id"`
	FeedID    string    `json:"feedId"`
	Timestamp time.Time `json:"timestamp"`
	FrameNum  int64     `json:"frameNum"`
	Data      []byte    `json:"data,omitempty"` // JPEG/H264 data
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	Format    string    `json:"format"` // jpeg, h264, h265
	Keyframe  bool      `json:"keyframe"`
	Encrypted bool      `json:"encrypted"`
}

// MapOverlay represents a 3D map visualization update
type MapOverlay struct {
	ID             string     `json:"id"`
	FeedID         string     `json:"feedId"`
	Timestamp      time.Time  `json:"timestamp"`
	PayloadTrack   []Vector3D `json:"payloadTrack"`          // Historical positions
	PlannedRoute   []Vector3D `json:"plannedRoute"`          // Future waypoints
	ThreatZones    []Zone     `json:"threatZones"`           // Active threats
	NoFlyZones     []Zone     `json:"noFlyZones"`            // Restricted areas
	TerrainMesh    []byte     `json:"terrainMesh,omitempty"` // 3D terrain data
	WeatherOverlay []byte     `json:"weatherOverlay,omitempty"`
}

// Zone represents a geographic zone
type Zone struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Center   Vector3D `json:"center"`
	Radius   float64  `json:"radius"`
	ZoneType string   `json:"zoneType"`
	Severity string   `json:"severity"`
	Active   bool     `json:"active"`
}

// AlertFrame represents an alert notification
type AlertFrame struct {
	ID          string         `json:"id"`
	FeedID      string         `json:"feedId"`
	Timestamp   time.Time      `json:"timestamp"`
	AlertType   string         `json:"alertType"` // warning, critical, info
	Title       string         `json:"title"`
	Message     string         `json:"message"`
	Severity    string         `json:"severity"`
	Clearance   ClearanceLevel `json:"clearance"`
	AckRequired bool           `json:"ackRequired"`
	Position    *Vector3D      `json:"position,omitempty"`
}

// Subscriber represents a feed subscriber
type Subscriber struct {
	ID           string         `json:"id"`
	UserID       string         `json:"userId"`
	Clearance    ClearanceLevel `json:"clearance"`
	FeedID       string         `json:"feedId"`
	Channel      chan []byte    `json:"-"`
	ConnectedAt  time.Time      `json:"connectedAt"`
	LastActivity time.Time      `json:"lastActivity"`
	IPAddress    string         `json:"ipAddress"`
	UserAgent    string         `json:"userAgent"`
}

// LiveFeedManager manages all live feeds
type LiveFeedManager struct {
	mu sync.RWMutex

	feeds       map[string]*LiveFeed
	subscribers map[string]map[string]*Subscriber // feedID -> subscriberID -> subscriber

	// Channels for broadcasting
	telemetryChannels map[string]chan *TelemetryFrame
	videoChannels     map[string]chan *VideoFrame
	mapChannels       map[string]chan *MapOverlay
	alertChannels     map[string]chan *AlertFrame

	ctx    context.Context
	cancel context.CancelFunc
}

// NewLiveFeedManager creates a new live feed manager
func NewLiveFeedManager() *LiveFeedManager {
	return &LiveFeedManager{
		feeds:             make(map[string]*LiveFeed),
		subscribers:       make(map[string]map[string]*Subscriber),
		telemetryChannels: make(map[string]chan *TelemetryFrame),
		videoChannels:     make(map[string]chan *VideoFrame),
		mapChannels:       make(map[string]chan *MapOverlay),
		alertChannels:     make(map[string]chan *AlertFrame),
	}
}

// Start begins the live feed manager
func (lfm *LiveFeedManager) Start(ctx context.Context) error {
	lfm.ctx, lfm.cancel = context.WithCancel(ctx)
	go lfm.cleanupLoop()
	return nil
}

// Stop halts the live feed manager
func (lfm *LiveFeedManager) Stop() {
	if lfm.cancel != nil {
		lfm.cancel()
	}
}

// CreateFeed creates a new live feed
func (lfm *LiveFeedManager) CreateFeed(feed *LiveFeed) error {
	lfm.mu.Lock()
	defer lfm.mu.Unlock()

	if feed.ID == "" {
		feed.ID = uuid.New().String()
	}
	feed.Status = "active"
	feed.StartedAt = time.Now()
	feed.ViewerCount = 0

	lfm.feeds[feed.ID] = feed
	lfm.subscribers[feed.ID] = make(map[string]*Subscriber)

	// Create broadcast channels
	lfm.telemetryChannels[feed.ID] = make(chan *TelemetryFrame, 100)
	lfm.videoChannels[feed.ID] = make(chan *VideoFrame, 30)
	lfm.mapChannels[feed.ID] = make(chan *MapOverlay, 10)
	lfm.alertChannels[feed.ID] = make(chan *AlertFrame, 50)

	// Start broadcaster for this feed
	go lfm.broadcastLoop(feed.ID)

	return nil
}

// GetFeed retrieves a feed by ID
func (lfm *LiveFeedManager) GetFeed(feedID string) (*LiveFeed, error) {
	lfm.mu.RLock()
	defer lfm.mu.RUnlock()

	feed, exists := lfm.feeds[feedID]
	if !exists {
		return nil, fmt.Errorf("feed not found: %s", feedID)
	}
	return feed, nil
}

// GetFeedsForClearance returns feeds accessible at given clearance level
func (lfm *LiveFeedManager) GetFeedsForClearance(clearance ClearanceLevel) []*LiveFeed {
	lfm.mu.RLock()
	defer lfm.mu.RUnlock()

	result := make([]*LiveFeed, 0)
	for _, feed := range lfm.feeds {
		if feed.Clearance <= clearance && feed.Status == "active" {
			result = append(result, feed)
		}
	}
	return result
}

// Subscribe adds a subscriber to a feed
func (lfm *LiveFeedManager) Subscribe(feedID string, subscriber *Subscriber) error {
	lfm.mu.Lock()
	defer lfm.mu.Unlock()

	feed, exists := lfm.feeds[feedID]
	if !exists {
		return fmt.Errorf("feed not found: %s", feedID)
	}

	// Check clearance
	if subscriber.Clearance < feed.Clearance {
		return fmt.Errorf("insufficient clearance: requires %d, have %d", feed.Clearance, subscriber.Clearance)
	}

	if subscriber.ID == "" {
		subscriber.ID = uuid.New().String()
	}
	subscriber.FeedID = feedID
	subscriber.ConnectedAt = time.Now()
	subscriber.LastActivity = time.Now()
	subscriber.Channel = make(chan []byte, 100)

	lfm.subscribers[feedID][subscriber.ID] = subscriber
	feed.ViewerCount++

	return nil
}

// Unsubscribe removes a subscriber from a feed
func (lfm *LiveFeedManager) Unsubscribe(feedID string, subscriberID string) {
	lfm.mu.Lock()
	defer lfm.mu.Unlock()

	if subs, exists := lfm.subscribers[feedID]; exists {
		if sub, ok := subs[subscriberID]; ok {
			close(sub.Channel)
			delete(subs, subscriberID)
			if feed, ok := lfm.feeds[feedID]; ok {
				feed.ViewerCount--
			}
		}
	}
}

// PublishTelemetry publishes telemetry to a feed
func (lfm *LiveFeedManager) PublishTelemetry(feedID string, frame *TelemetryFrame) error {
	lfm.mu.RLock()
	ch, exists := lfm.telemetryChannels[feedID]
	lfm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("feed not found: %s", feedID)
	}

	frame.ID = uuid.New().String()
	frame.FeedID = feedID
	frame.Timestamp = time.Now()

	select {
	case ch <- frame:
		return nil
	default:
		return fmt.Errorf("telemetry channel full")
	}
}

// PublishVideo publishes video to a feed
func (lfm *LiveFeedManager) PublishVideo(feedID string, frame *VideoFrame) error {
	lfm.mu.RLock()
	ch, exists := lfm.videoChannels[feedID]
	lfm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("feed not found: %s", feedID)
	}

	frame.ID = uuid.New().String()
	frame.FeedID = feedID
	frame.Timestamp = time.Now()

	select {
	case ch <- frame:
		return nil
	default:
		return fmt.Errorf("video channel full")
	}
}

// PublishMapOverlay publishes map data to a feed
func (lfm *LiveFeedManager) PublishMapOverlay(feedID string, overlay *MapOverlay) error {
	lfm.mu.RLock()
	ch, exists := lfm.mapChannels[feedID]
	lfm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("feed not found: %s", feedID)
	}

	overlay.ID = uuid.New().String()
	overlay.FeedID = feedID
	overlay.Timestamp = time.Now()

	select {
	case ch <- overlay:
		return nil
	default:
		return fmt.Errorf("map channel full")
	}
}

// PublishAlert publishes an alert to a feed
func (lfm *LiveFeedManager) PublishAlert(feedID string, alert *AlertFrame) error {
	lfm.mu.RLock()
	ch, exists := lfm.alertChannels[feedID]
	lfm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("feed not found: %s", feedID)
	}

	alert.ID = uuid.New().String()
	alert.FeedID = feedID
	alert.Timestamp = time.Now()

	select {
	case ch <- alert:
		return nil
	default:
		return fmt.Errorf("alert channel full")
	}
}

// broadcastLoop broadcasts data to all subscribers of a feed
func (lfm *LiveFeedManager) broadcastLoop(feedID string) {
	for {
		select {
		case <-lfm.ctx.Done():
			return

		case frame := <-lfm.telemetryChannels[feedID]:
			lfm.broadcastToSubscribers(feedID, "telemetry", frame)

		case frame := <-lfm.videoChannels[feedID]:
			lfm.broadcastToSubscribers(feedID, "video", frame)

		case overlay := <-lfm.mapChannels[feedID]:
			lfm.broadcastToSubscribers(feedID, "map", overlay)

		case alert := <-lfm.alertChannels[feedID]:
			lfm.broadcastToSubscribers(feedID, "alert", alert)
		}
	}
}

// broadcastToSubscribers sends data to all subscribers
func (lfm *LiveFeedManager) broadcastToSubscribers(feedID string, dataType string, data interface{}) {
	lfm.mu.RLock()
	subs := lfm.subscribers[feedID]
	lfm.mu.RUnlock()

	message := struct {
		Type      string      `json:"type"`
		FeedID    string      `json:"feedId"`
		Timestamp time.Time   `json:"timestamp"`
		Data      interface{} `json:"data"`
	}{
		Type:      dataType,
		FeedID:    feedID,
		Timestamp: time.Now(),
		Data:      data,
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return
	}

	for _, sub := range subs {
		select {
		case sub.Channel <- jsonData:
			sub.LastActivity = time.Now()
		default:
			// Subscriber buffer full, skip
		}
	}
}

// cleanupLoop removes stale subscribers
func (lfm *LiveFeedManager) cleanupLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-lfm.ctx.Done():
			return
		case <-ticker.C:
			lfm.cleanupStaleSubscribers()
		}
	}
}

// cleanupStaleSubscribers removes inactive subscribers
func (lfm *LiveFeedManager) cleanupStaleSubscribers() {
	lfm.mu.Lock()
	defer lfm.mu.Unlock()

	staleThreshold := 5 * time.Minute

	for feedID, subs := range lfm.subscribers {
		for subID, sub := range subs {
			if time.Since(sub.LastActivity) > staleThreshold {
				close(sub.Channel)
				delete(subs, subID)
				if feed, ok := lfm.feeds[feedID]; ok {
					feed.ViewerCount--
				}
			}
		}
	}
}

// EndFeed ends a live feed
func (lfm *LiveFeedManager) EndFeed(feedID string) error {
	lfm.mu.Lock()
	defer lfm.mu.Unlock()

	feed, exists := lfm.feeds[feedID]
	if !exists {
		return fmt.Errorf("feed not found: %s", feedID)
	}

	feed.Status = "ended"

	// Close all subscriber channels
	for _, sub := range lfm.subscribers[feedID] {
		close(sub.Channel)
	}
	delete(lfm.subscribers, feedID)

	// Close broadcast channels
	close(lfm.telemetryChannels[feedID])
	close(lfm.videoChannels[feedID])
	close(lfm.mapChannels[feedID])
	close(lfm.alertChannels[feedID])

	delete(lfm.telemetryChannels, feedID)
	delete(lfm.videoChannels, feedID)
	delete(lfm.mapChannels, feedID)
	delete(lfm.alertChannels, feedID)

	return nil
}

// GetSubscriberChannel returns the channel for a subscriber
func (lfm *LiveFeedManager) GetSubscriberChannel(feedID, subscriberID string) (chan []byte, error) {
	lfm.mu.RLock()
	defer lfm.mu.RUnlock()

	if subs, exists := lfm.subscribers[feedID]; exists {
		if sub, ok := subs[subscriberID]; ok {
			return sub.Channel, nil
		}
	}
	return nil, fmt.Errorf("subscriber not found")
}

// GetFeedStats returns statistics for a feed
func (lfm *LiveFeedManager) GetFeedStats(feedID string) (*FeedStats, error) {
	lfm.mu.RLock()
	defer lfm.mu.RUnlock()

	feed, exists := lfm.feeds[feedID]
	if !exists {
		return nil, fmt.Errorf("feed not found: %s", feedID)
	}

	stats := &FeedStats{
		FeedID:      feedID,
		ViewerCount: feed.ViewerCount,
		Status:      feed.Status,
		Uptime:      time.Since(feed.StartedAt),
		Timestamp:   time.Now(),
	}

	return stats, nil
}

// FeedStats contains feed statistics
type FeedStats struct {
	FeedID      string        `json:"feedId"`
	ViewerCount int           `json:"viewerCount"`
	Status      string        `json:"status"`
	Uptime      time.Duration `json:"uptime"`
	Timestamp   time.Time     `json:"timestamp"`
}

// ClearanceLevelName returns human-readable name for clearance
func ClearanceLevelName(level ClearanceLevel) string {
	switch level {
	case ClearancePublic:
		return "PUBLIC"
	case ClearanceCivilian:
		return "CIVILIAN"
	case ClearanceMilitary:
		return "MILITARY"
	case ClearanceGov:
		return "GOVERNMENT"
	case ClearanceSecret:
		return "SECRET"
	case ClearanceUltra:
		return "ULTRA"
	default:
		return "UNKNOWN"
	}
}

// ParseClearanceLevel parses clearance level from string
func ParseClearanceLevel(s string) ClearanceLevel {
	switch s {
	case "PUBLIC", "public":
		return ClearancePublic
	case "CIVILIAN", "civilian":
		return ClearanceCivilian
	case "MILITARY", "military":
		return ClearanceMilitary
	case "GOVERNMENT", "government", "GOV", "gov":
		return ClearanceGov
	case "SECRET", "secret":
		return ClearanceSecret
	case "ULTRA", "ultra":
		return ClearanceUltra
	default:
		return ClearancePublic
	}
}
