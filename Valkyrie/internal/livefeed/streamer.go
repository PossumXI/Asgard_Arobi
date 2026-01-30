// Package livefeed provides real-time telemetry streaming via WebSocket
package livefeed

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// LiveFeedStreamer broadcasts telemetry to WebSocket clients
type LiveFeedStreamer struct {
	mu        sync.RWMutex
	clients   map[*Client]bool
	broadcast chan *TelemetryMessage

	// Upgrader for WebSocket connections
	upgrader websocket.Upgrader

	// Logger
	logger *logrus.Logger

	// Statistics
	messagesSent   uint64
	clientsServed  uint64
	currentClients int
}

// Client represents a connected WebSocket client
type Client struct {
	conn      *websocket.Conn
	clearance int
	send      chan *TelemetryMessage
	id        string
}

// TelemetryMessage contains flight data
type TelemetryMessage struct {
	Timestamp    time.Time  `json:"timestamp"`
	Position     [3]float64 `json:"position"`
	Velocity     [3]float64 `json:"velocity"`
	Attitude     [3]float64 `json:"attitude"`
	AngularRate  [3]float64 `json:"angular_rate"`
	Acceleration [3]float64 `json:"acceleration"`
	Throttle     float64    `json:"throttle"`
	Fuel         float64    `json:"fuel"`
	Battery      float64    `json:"battery"`
	Status       string     `json:"status"`
	FlightMode   string     `json:"flight_mode"`
	Clearance    int        `json:"clearance"`
	MissionID    string     `json:"mission_id,omitempty"`

	// Alerts
	Alerts []Alert `json:"alerts,omitempty"`

	// Additional telemetry
	Airspeed        float64 `json:"airspeed,omitempty"`
	GroundSpeed     float64 `json:"ground_speed,omitempty"`
	AltitudeAGL     float64 `json:"altitude_agl,omitempty"`
	AltitudeMSL     float64 `json:"altitude_msl,omitempty"`
	Heading         float64 `json:"heading,omitempty"`
	WindSpeed       float64 `json:"wind_speed,omitempty"`
	WindDirection   float64 `json:"wind_direction,omitempty"`
	GPSSatellites   int     `json:"gps_satellites,omitempty"`
	GPSFix          string  `json:"gps_fix,omitempty"`
	SignalStrength  float64 `json:"signal_strength,omitempty"`
	FusionConfidence float64 `json:"fusion_confidence,omitempty"`
}

// Alert represents an in-flight alert
type Alert struct {
	Type     string    `json:"type"`
	Severity string    `json:"severity"` // info, warning, critical
	Message  string    `json:"message"`
	Time     time.Time `json:"time"`
}

// ClearanceLevel defines access tiers
const (
	ClearancePublic     = 0
	ClearanceBasic      = 1
	ClearanceOperator   = 2
	ClearanceCommander  = 3
	ClearanceAdmin      = 4
)

// NewLiveFeedStreamer creates a new streamer
func NewLiveFeedStreamer() *LiveFeedStreamer {
	return &LiveFeedStreamer{
		clients:   make(map[*Client]bool),
		broadcast: make(chan *TelemetryMessage, 100),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for now
			},
		},
		logger: logrus.New(),
	}
}

// HandleWebSocket handles incoming WebSocket connections
func (lfs *LiveFeedStreamer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP to WebSocket
	conn, err := lfs.upgrader.Upgrade(w, r, nil)
	if err != nil {
		lfs.logger.WithError(err).Error("Failed to upgrade WebSocket")
		return
	}

	// Determine clearance from headers or query params
	clearance := ClearancePublic
	if token := r.Header.Get("X-Clearance-Token"); token != "" {
		clearance = lfs.validateClearance(token)
	}

	// Create client
	client := &Client{
		conn:      conn,
		clearance: clearance,
		send:      make(chan *TelemetryMessage, 50),
		id:        r.RemoteAddr,
	}

	// Register client
	lfs.RegisterClient(client)

	lfs.logger.WithFields(logrus.Fields{
		"client":    client.id,
		"clearance": clearance,
	}).Info("Client connected")

	// Start read/write pumps
	ctx, cancel := context.WithCancel(context.Background())
	go client.WritePump(ctx, lfs)
	go client.ReadPump(ctx, cancel, lfs)
}

// RegisterClient adds a new WebSocket client
func (lfs *LiveFeedStreamer) RegisterClient(client *Client) {
	lfs.mu.Lock()
	defer lfs.mu.Unlock()

	lfs.clients[client] = true
	lfs.clientsServed++
	lfs.currentClients++
}

// UnregisterClient removes a client
func (lfs *LiveFeedStreamer) UnregisterClient(client *Client) {
	lfs.mu.Lock()
	defer lfs.mu.Unlock()

	if _, ok := lfs.clients[client]; ok {
		delete(lfs.clients, client)
		close(client.send)
		lfs.currentClients--
		lfs.logger.WithField("client", client.id).Info("Client disconnected")
	}
}

// BroadcastTelemetry sends telemetry to all clients
func (lfs *LiveFeedStreamer) BroadcastTelemetry(msg *TelemetryMessage) {
	select {
	case lfs.broadcast <- msg:
	default:
		// Buffer full, drop oldest
		select {
		case <-lfs.broadcast:
		default:
		}
		lfs.broadcast <- msg
	}
}

// Run starts the streaming loop
func (lfs *LiveFeedStreamer) Run(ctx context.Context) error {
	lfs.logger.Info("LiveFeed streamer started")

	for {
		select {
		case <-ctx.Done():
			lfs.logger.Info("LiveFeed streamer stopping")
			lfs.closeAllClients()
			return ctx.Err()

		case msg := <-lfs.broadcast:
			lfs.sendToClients(msg)
		}
	}
}

// sendToClients distributes messages based on clearance
func (lfs *LiveFeedStreamer) sendToClients(msg *TelemetryMessage) {
	lfs.mu.RLock()
	defer lfs.mu.RUnlock()

	for client := range lfs.clients {
		if client.clearance >= msg.Clearance {
			// Filter message based on clearance
			filteredMsg := lfs.filterMessage(msg, client.clearance)
			select {
			case client.send <- filteredMsg:
				lfs.messagesSent++
			default:
				// Client buffer full, skip
			}
		}
	}
}

// filterMessage removes sensitive data based on clearance
func (lfs *LiveFeedStreamer) filterMessage(msg *TelemetryMessage, clearance int) *TelemetryMessage {
	if clearance >= ClearanceAdmin {
		return msg
	}

	// Create a copy
	filtered := *msg

	// Remove sensitive data for lower clearances
	if clearance < ClearanceCommander {
		filtered.Alerts = nil
		filtered.MissionID = ""
	}

	if clearance < ClearanceOperator {
		filtered.FusionConfidence = 0
		filtered.GPSSatellites = 0
	}

	return &filtered
}

// closeAllClients closes all client connections
func (lfs *LiveFeedStreamer) closeAllClients() {
	lfs.mu.Lock()
	defer lfs.mu.Unlock()

	for client := range lfs.clients {
		client.conn.Close()
		close(client.send)
		delete(lfs.clients, client)
	}
}

// validateClearance validates a clearance token
func (lfs *LiveFeedStreamer) validateClearance(token string) int {
	// TODO: Implement actual token validation
	// For now, return basic clearance
	if token == "admin" {
		return ClearanceAdmin
	}
	if token == "commander" {
		return ClearanceCommander
	}
	if token == "operator" {
		return ClearanceOperator
	}
	return ClearanceBasic
}

// GetStats returns streaming statistics
func (lfs *LiveFeedStreamer) GetStats() (clients int, sent uint64, served uint64) {
	lfs.mu.RLock()
	defer lfs.mu.RUnlock()
	return lfs.currentClients, lfs.messagesSent, lfs.clientsServed
}

// WritePump sends messages to WebSocket
func (c *Client) WritePump(ctx context.Context, lfs *LiveFeedStreamer) {
	ticker := time.NewTicker(30 * time.Second) // Ping interval
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case msg, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			data, err := json.Marshal(msg)
			if err != nil {
				continue
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ReadPump reads messages from WebSocket (for commands)
func (c *Client) ReadPump(ctx context.Context, cancel context.CancelFunc, lfs *LiveFeedStreamer) {
	defer func() {
		cancel()
		lfs.UnregisterClient(c)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(4096)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				lfs.logger.WithError(err).Error("WebSocket read error")
			}
			return
		}

		// Handle incoming commands (if any)
		lfs.handleClientMessage(c, message)
	}
}

// handleClientMessage processes commands from clients
func (lfs *LiveFeedStreamer) handleClientMessage(client *Client, message []byte) {
	var cmd struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(message, &cmd); err != nil {
		return
	}

	switch cmd.Type {
	case "subscribe":
		// Handle subscription changes
		lfs.logger.WithField("client", client.id).Info("Client subscription updated")

	case "command":
		// Handle flight commands (if authorized)
		if client.clearance >= ClearanceCommander {
			lfs.logger.WithFields(logrus.Fields{
				"client": client.id,
				"cmd":    string(cmd.Data),
			}).Info("Flight command received")
		}
	}
}
