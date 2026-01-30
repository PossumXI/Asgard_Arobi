// Package realtime provides WebSocket management for real-time event delivery.
package realtime

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/asgard/pandora/internal/platform/observability"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512

	// Maximum number of messages in the send buffer.
	sendBufferSize = 256
)

// createOriginChecker returns a function that validates WebSocket origins.
// In production, only origins in ALLOWED_ORIGINS environment variable are permitted.
// In development, localhost origins are also allowed.
func createOriginChecker() func(r *http.Request) bool {
	allowedOrigins := parseAllowedOrigins()
	isDev := os.Getenv("ASGARD_ENV") == "development"

	return func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			// No origin header - likely same-origin or non-browser client
			// Allow for development, deny for production
			if isDev {
				return true
			}
			log.Printf("[WebSocket] SECURITY: Rejected connection with no Origin header from %s", r.RemoteAddr)
			return false
		}

		// Parse the origin URL
		originURL, err := url.Parse(origin)
		if err != nil {
			log.Printf("[WebSocket] SECURITY: Rejected invalid origin %q from %s", origin, r.RemoteAddr)
			return false
		}

		// Check if origin is in allowed list
		for _, allowed := range allowedOrigins {
			if strings.EqualFold(origin, allowed) || strings.EqualFold(originURL.Host, allowed) {
				return true
			}
		}

		// In development, also allow localhost variants
		if isDev && isLocalhostOrigin(originURL) {
			return true
		}

		// Log rejected origins for security monitoring
		log.Printf("[WebSocket] SECURITY: Rejected origin %q from %s (user-agent: %s)",
			origin, r.RemoteAddr, r.UserAgent())
		observability.GetMetrics().WebSocketMessages.WithLabelValues("rejected_origin", "security").Inc()

		return false
	}
}

// parseAllowedOrigins reads and parses the ALLOWED_ORIGINS environment variable.
func parseAllowedOrigins() []string {
	originsEnv := os.Getenv("ALLOWED_ORIGINS")
	if originsEnv == "" {
		// Default allowed origins for production
		return []string{
			"https://aura-genesis.org",
			"https://www.aura-genesis.org",
			"https://app.aura-genesis.org",
			"https://hubs.aura-genesis.org",
			"https://gov.aura-genesis.org",
		}
	}

	origins := strings.Split(originsEnv, ",")
	result := make([]string, 0, len(origins))
	for _, o := range origins {
		trimmed := strings.TrimSpace(o)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// isLocalhostOrigin checks if the origin is a localhost variant.
func isLocalhostOrigin(u *url.URL) bool {
	host := strings.ToLower(u.Hostname())
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     createOriginChecker(),
}

// Client represents a WebSocket client connection.
type Client struct {
	ID          string
	UserID      string
	AccessLevel AccessLevel
	conn        *websocket.Conn
	send        chan []byte
	manager     *WebSocketManager
	mu          sync.Mutex
	closed      bool
	filters     []EventType // Event types the client wants to receive
}

// WebSocketManager manages all WebSocket connections.
type WebSocketManager struct {
	clients    map[string]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan Event
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewWebSocketManager creates a new WebSocket manager.
func NewWebSocketManager() *WebSocketManager {
	ctx, cancel := context.WithCancel(context.Background())
	manager := &WebSocketManager{
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan Event, 1000),
		ctx:        ctx,
		cancel:     cancel,
	}
	go manager.run()
	return manager
}

// run starts the WebSocket manager event loop.
func (m *WebSocketManager) run() {
	for {
		select {
		case client := <-m.register:
			m.mu.Lock()
			m.clients[client.ID] = client
			m.mu.Unlock()
			observability.UpdateWebSocketConnections(len(m.clients))
			log.Printf("[WebSocket] Client registered: %s (access: %s)", client.ID, client.AccessLevel)

		case client := <-m.unregister:
			m.mu.Lock()
			if _, ok := m.clients[client.ID]; ok {
				delete(m.clients, client.ID)
				close(client.send)
			}
			m.mu.Unlock()
			observability.UpdateWebSocketConnections(len(m.clients))
			log.Printf("[WebSocket] Client unregistered: %s", client.ID)

		case event := <-m.broadcast:
			m.broadcastToClients(event)

		case <-m.ctx.Done():
			return
		}
	}
}

// broadcastToClients sends an event to all authorized clients.
func (m *WebSocketManager) broadcastToClients(event Event) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	message, err := json.Marshal(map[string]interface{}{
		"type":      "event",
		"eventType": event.Type,
		"event":     event,
	})
	if err != nil {
		log.Printf("[WebSocket] Failed to marshal event: %v", err)
		return
	}

	for _, client := range m.clients {
		// Check if client has access to this event
		if !client.canReceiveEvent(event) {
			continue
		}

		// Check if client has filtered for this event type
		if len(client.filters) > 0 && !client.wantsEventType(event.Type) {
			continue
		}

		select {
		case client.send <- message:
			observability.GetMetrics().WebSocketMessages.WithLabelValues("out", string(event.Type)).Inc()
		default:
			// Client buffer is full, consider disconnecting
			log.Printf("[WebSocket] Client %s buffer full, dropping message", client.ID)
		}
	}
}

// canReceiveEvent checks if a client can receive an event based on access level.
func (c *Client) canReceiveEvent(event Event) bool {
	return accessLevelAtLeast(c.AccessLevel, event.AccessLevel)
}

// wantsEventType checks if a client has subscribed to an event type.
func (c *Client) wantsEventType(eventType EventType) bool {
	for _, t := range c.filters {
		if t == eventType {
			return true
		}
	}
	return false
}

// accessLevelAtLeast checks if clientLevel is at least the required level.
func accessLevelAtLeast(clientLevel, requiredLevel AccessLevel) bool {
	levels := map[AccessLevel]int{
		AccessLevelPublic:       0,
		AccessLevelCivilian:     1,
		AccessLevelMilitary:     2,
		AccessLevelInterstellar: 3,
		AccessLevelGovernment:   4,
		AccessLevelAdmin:        5,
	}

	clientRank, ok1 := levels[clientLevel]
	requiredRank, ok2 := levels[requiredLevel]

	if !ok1 || !ok2 {
		return false
	}

	return clientRank >= requiredRank
}

// Broadcast sends an event to all connected clients.
func (m *WebSocketManager) Broadcast(event Event) {
	select {
	case m.broadcast <- event:
	default:
		log.Println("[WebSocket] Broadcast channel full, dropping event")
	}
}

// HandleWebSocket upgrades an HTTP connection to WebSocket.
func (m *WebSocketManager) HandleWebSocket(w http.ResponseWriter, r *http.Request, userID string, accessLevel AccessLevel) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WebSocket] Upgrade failed: %v", err)
		return
	}

	client := &Client{
		ID:          generateClientID(),
		UserID:      userID,
		AccessLevel: accessLevel,
		conn:        conn,
		send:        make(chan []byte, sendBufferSize),
		manager:     m,
		filters:     make([]EventType, 0),
	}

	m.register <- client

	// Start read and write goroutines
	go client.writePump()
	go client.readPump()

	// Send initial welcome message
	welcome := map[string]interface{}{
		"type":        "welcome",
		"clientId":    client.ID,
		"accessLevel": client.AccessLevel,
		"timestamp":   time.Now().UTC(),
	}
	if data, err := json.Marshal(welcome); err == nil {
		client.send <- data
	}
}

// readPump reads messages from the WebSocket connection.
func (c *Client) readPump() {
	defer func() {
		c.manager.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WebSocket] Error: %v", err)
			}
			break
		}

		// Handle client messages (subscriptions, filters, etc.)
		c.handleMessage(message)
	}
}

// handleMessage processes incoming client messages.
func (c *Client) handleMessage(message []byte) {
	var msg struct {
		Type    string   `json:"type"`
		Filters []string `json:"filters,omitempty"`
	}

	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("[WebSocket] Failed to parse message: %v", err)
		return
	}
	observability.GetMetrics().WebSocketMessages.WithLabelValues("in", msg.Type).Inc()

	switch msg.Type {
	case "subscribe":
		// Update client event filters
		c.mu.Lock()
		c.filters = make([]EventType, 0, len(msg.Filters))
		for _, f := range msg.Filters {
			c.filters = append(c.filters, EventType(f))
		}
		c.mu.Unlock()
		log.Printf("[WebSocket] Client %s subscribed to: %v", c.ID, msg.Filters)

		// Send confirmation
		confirm := map[string]interface{}{
			"type":    "subscribed",
			"filters": msg.Filters,
		}
		if data, err := json.Marshal(confirm); err == nil {
			c.send <- data
		}

	case "ping":
		// Respond with pong
		pong := map[string]interface{}{
			"type":      "pong",
			"timestamp": time.Now().UTC(),
		}
		if data, err := json.Marshal(pong); err == nil {
			c.send <- data
		}
	}
}

// writePump writes messages to the WebSocket connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Manager closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Batch write queued messages
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Stop stops the WebSocket manager.
func (m *WebSocketManager) Stop() {
	m.cancel()

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, client := range m.clients {
		client.conn.Close()
	}
}

// GetClientCount returns the number of connected clients.
func (m *WebSocketManager) GetClientCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.clients)
}

// GetClientsByAccessLevel returns clients by access level.
func (m *WebSocketManager) GetClientsByAccessLevel(level AccessLevel) []*Client {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var clients []*Client
	for _, client := range m.clients {
		if client.AccessLevel == level {
			clients = append(clients, client)
		}
	}
	return clients
}

// Stats returns WebSocket manager statistics.
func (m *WebSocketManager) Stats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	accessCounts := make(map[string]int)
	for _, client := range m.clients {
		accessCounts[string(client.AccessLevel)]++
	}

	return map[string]interface{}{
		"total_clients":    len(m.clients),
		"clients_by_level": accessCounts,
	}
}

func generateClientID() string {
	return "ws-" + time.Now().Format("20060102150405") + "-" + randomString(6)
}
