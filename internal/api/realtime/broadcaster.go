// Package realtime provides real-time event broadcasting via WebSocket.
package realtime

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Event represents a real-time event.
type Event struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

// Broadcaster manages WebSocket connections and broadcasts events.
type Broadcaster struct {
	clients    map[*websocket.Conn]bool
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	broadcast  chan Event
	mu         sync.RWMutex
	done       chan struct{}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

// NewBroadcaster creates a new event broadcaster.
func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		clients:    make(map[*websocket.Conn]bool),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
		broadcast:  make(chan Event, 256),
		done:       make(chan struct{}),
	}
}

// Start begins the broadcaster event loop.
func (b *Broadcaster) Start() {
	for {
		select {
		case conn := <-b.register:
			b.mu.Lock()
			b.clients[conn] = true
			b.mu.Unlock()
			log.Printf("Client connected. Total clients: %d", len(b.clients))

		case conn := <-b.unregister:
			b.mu.Lock()
			if _, ok := b.clients[conn]; ok {
				delete(b.clients, conn)
				conn.Close()
			}
			b.mu.Unlock()
			log.Printf("Client disconnected. Total clients: %d", len(b.clients))

		case event := <-b.broadcast:
			b.mu.RLock()
			for conn := range b.clients {
				if err := conn.WriteJSON(event); err != nil {
					log.Printf("Error broadcasting to client: %v", err)
					b.unregister <- conn
				}
			}
			b.mu.RUnlock()

		case <-b.done:
			return
		}
	}
}

// Broadcast sends an event to all connected clients.
func (b *Broadcaster) Broadcast(eventType string, payload interface{}) {
	event := Event{
		Type:      eventType,
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	}

	select {
	case b.broadcast <- event:
	default:
		log.Printf("Broadcast channel full, dropping event: %s", eventType)
	}
}

// Stop stops the broadcaster.
func (b *Broadcaster) Stop() {
	close(b.done)
	b.mu.Lock()
	for conn := range b.clients {
		conn.Close()
		delete(b.clients, conn)
	}
	b.mu.Unlock()
}

// HandleWebSocket handles WebSocket connections for real-time events.
func HandleWebSocket(w http.ResponseWriter, r *http.Request, broadcaster *Broadcaster) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	broadcaster.register <- conn

	// Handle incoming messages (ping/pong)
	go func() {
		defer func() {
			broadcaster.unregister <- conn
		}()

		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		conn.SetPongHandler(func(string) error {
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			return nil
		})

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				}
				break
			}
		}
	}()

	// Send ping messages
	go func() {
		ticker := time.NewTicker(54 * time.Second)
		defer ticker.Stop()
		defer conn.Close()

		for {
			select {
			case <-ticker.C:
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			case <-broadcaster.done:
				return
			}
		}
	}()
}
