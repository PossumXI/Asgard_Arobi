package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/asgard/pandora/internal/nysus/events"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// WebSocketClient represents a connected WebSocket client.
type WebSocketClient struct {
	hub    *WebSocketHub
	conn   *websocket.Conn
	send   chan []byte
	userID string
	subs   map[string]bool
	subsMu sync.RWMutex
}

// WebSocketHub manages WebSocket connections and broadcasts.
type WebSocketHub struct {
	clients    map[*WebSocketClient]bool
	broadcast  chan []byte
	register   chan *WebSocketClient
	unregister chan *WebSocketClient
	eventBus   *events.EventBus
	mu         sync.RWMutex
}

// NewWebSocketHub creates a new WebSocket hub.
func NewWebSocketHub(eventBus *events.EventBus) *WebSocketHub {
	hub := &WebSocketHub{
		clients:    make(map[*WebSocketClient]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
		eventBus:   eventBus,
	}

	// Subscribe to all events for broadcast
	if eventBus != nil {
		eventBus.SubscribeAll(func(ctx context.Context, event events.Event) error {
			hub.BroadcastEvent(event)
			return nil
		})
	}

	return hub
}

// Run starts the hub's main loop.
func (h *WebSocketHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("[WebSocket] Client connected: %s", client.userID)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("[WebSocket] Client disconnected: %s", client.userID)

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastEvent sends an event to all connected clients.
func (h *WebSocketHub) BroadcastEvent(event events.Event) {
	msg := map[string]interface{}{
		"type":      string(event.Type),
		"timestamp": event.Timestamp.Format(time.RFC3339),
		"payload":   event.Payload,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[WebSocket] Failed to marshal event: %v", err)
		return
	}

	select {
	case h.broadcast <- data:
	default:
		log.Println("[WebSocket] Broadcast channel full")
	}
}

// handleWebSocket handles WebSocket upgrade requests.
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WebSocket] Upgrade error: %v", err)
		return
	}

	userID := "anonymous"
	if token := extractToken(r); token != "" {
		if uid, _, _, _, err := parseJWTClaims(token); err == nil && uid != "" {
			userID = uid
		}
	}

	client := &WebSocketClient{
		hub:    s.wsHub,
		conn:   conn,
		send:   make(chan []byte, 256),
		userID: userID,
		subs:   make(map[string]bool),
	}

	client.hub.register <- client

	// Start read/write pumps
	go client.writePump()
	go client.readPump()
}

// readPump handles incoming messages from the client.
func (c *WebSocketClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(65536)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WebSocket] Read error: %v", err)
			}
			break
		}

		// Handle incoming message
		c.handleMessage(message)
	}
}

// writePump sends messages to the client.
func (c *WebSocketClient) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to current write
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
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

// handleMessage processes incoming WebSocket messages.
func (c *WebSocketClient) handleMessage(data []byte) {
	var msg struct {
		Type    string `json:"type"`
		Channel string `json:"channel"`
	}

	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}

	switch msg.Type {
	case "subscribe":
		c.subsMu.Lock()
		c.subs[msg.Channel] = true
		c.subsMu.Unlock()
		log.Printf("[WebSocket] Client %s subscribed to %s", c.userID, msg.Channel)

	case "unsubscribe":
		c.subsMu.Lock()
		delete(c.subs, msg.Channel)
		c.subsMu.Unlock()

	case "ping":
		response, _ := json.Marshal(map[string]string{
			"type":      "pong",
			"timestamp": time.Now().Format(time.RFC3339),
		})
		c.send <- response
	}
}
