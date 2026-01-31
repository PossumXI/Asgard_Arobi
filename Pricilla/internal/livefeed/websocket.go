package livefeed

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
)

// WebSocketHub manages WebSocket connections for live feeds
type WebSocketHub struct {
	mu sync.RWMutex

	feedManager *LiveFeedManager
	clients     map[string]*WebSocketClient
	register    chan *WebSocketClient
	unregister  chan *WebSocketClient
	broadcast   chan []byte
}

// WebSocketClient represents a connected WebSocket client
type WebSocketClient struct {
	ID          string
	UserID      string
	Clearance   ClearanceLevel
	FeedID      string
	Conn        interface{} // WebSocket connection (would be *websocket.Conn in production)
	Send        chan []byte
	Hub         *WebSocketHub
	ConnectedAt time.Time
	LastPing    time.Time
}

// WebSocketMessage represents a WebSocket message
type WebSocketMessage struct {
	Type      string          `json:"type"`
	FeedID    string          `json:"feedId,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub(feedManager *LiveFeedManager) *WebSocketHub {
	return &WebSocketHub{
		feedManager: feedManager,
		clients:     make(map[string]*WebSocketClient),
		register:    make(chan *WebSocketClient),
		unregister:  make(chan *WebSocketClient),
		broadcast:   make(chan []byte),
	}
}

// Run starts the WebSocket hub
func (h *WebSocketHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.ID] = client
			h.mu.Unlock()
			log.Printf("WebSocket client connected: %s (clearance: %s)", client.ID, ClearanceLevelName(client.Clearance))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				close(client.Send)

				// Unsubscribe from feed
				if client.FeedID != "" {
					h.feedManager.Unsubscribe(client.FeedID, client.ID)
				}
			}
			h.mu.Unlock()
			log.Printf("WebSocket client disconnected: %s", client.ID)

		case message := <-h.broadcast:
			h.mu.RLock()
			for _, client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client.ID)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// HandleSubscribe handles feed subscription requests
func (h *WebSocketHub) HandleSubscribe(client *WebSocketClient, feedID string) error {
	// Check if feed exists and client has clearance
	feed, err := h.feedManager.GetFeed(feedID)
	if err != nil {
		return err
	}

	if client.Clearance < feed.Clearance {
		return &AccessDeniedError{
			Required: feed.Clearance,
			Have:     client.Clearance,
		}
	}

	// Create subscriber
	subscriber := &Subscriber{
		ID:           client.ID,
		UserID:       client.UserID,
		Clearance:    client.Clearance,
		FeedID:       feedID,
		ConnectedAt:  time.Now(),
		LastActivity: time.Now(),
	}

	if err := h.feedManager.Subscribe(feedID, subscriber); err != nil {
		return err
	}

	client.FeedID = feedID

	// Start forwarding data from feed to client
	go h.forwardFeedData(client, feedID)

	return nil
}

// forwardFeedData forwards feed data to a client
func (h *WebSocketHub) forwardFeedData(client *WebSocketClient, feedID string) {
	ch, err := h.feedManager.GetSubscriberChannel(feedID, client.ID)
	if err != nil {
		return
	}

	for data := range ch {
		select {
		case client.Send <- data:
		default:
			// Client buffer full, disconnect
			h.unregister <- client
			return
		}
	}
}

// HandleUnsubscribe handles feed unsubscription
func (h *WebSocketHub) HandleUnsubscribe(client *WebSocketClient) {
	if client.FeedID != "" {
		h.feedManager.Unsubscribe(client.FeedID, client.ID)
		client.FeedID = ""
	}
}

// BroadcastToFeed sends a message to all subscribers of a feed
func (h *WebSocketHub) BroadcastToFeed(feedID string, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, client := range h.clients {
		if client.FeedID == feedID {
			select {
			case client.Send <- message:
			default:
				// Skip slow clients
			}
		}
	}
}

// GetConnectedClients returns number of connected clients
func (h *WebSocketHub) GetConnectedClients() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// GetClientsByFeed returns clients subscribed to a feed
func (h *WebSocketHub) GetClientsByFeed(feedID string) []*WebSocketClient {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients := make([]*WebSocketClient, 0)
	for _, client := range h.clients {
		if client.FeedID == feedID {
			clients = append(clients, client)
		}
	}
	return clients
}

// AccessDeniedError represents an access denied error
type AccessDeniedError struct {
	Required ClearanceLevel
	Have     ClearanceLevel
}

func (e *AccessDeniedError) Error() string {
	return "access denied: insufficient clearance"
}

// LiveFeedHTTPHandler handles HTTP requests for live feeds
type LiveFeedHTTPHandler struct {
	feedManager *LiveFeedManager
	wsHub       *WebSocketHub
}

// NewLiveFeedHTTPHandler creates a new HTTP handler
func NewLiveFeedHTTPHandler(feedManager *LiveFeedManager, wsHub *WebSocketHub) *LiveFeedHTTPHandler {
	return &LiveFeedHTTPHandler{
		feedManager: feedManager,
		wsHub:       wsHub,
	}
}

// ServeHTTP handles HTTP requests
func (h *LiveFeedHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Clearance")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	path := r.URL.Path

	switch {
	case path == "/api/v1/feeds" && r.Method == "GET":
		h.handleListFeeds(w, r)
	case path == "/api/v1/feeds" && r.Method == "POST":
		h.handleCreateFeed(w, r)
	case len(path) > 14 && path[:14] == "/api/v1/feeds/":
		feedID := path[14:]
		if r.Method == "GET" {
			h.handleGetFeed(w, r, feedID)
		} else if r.Method == "DELETE" {
			h.handleEndFeed(w, r, feedID)
		}
	case path == "/api/v1/feeds/stats":
		h.handleGetStats(w, r)
	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

// handleListFeeds lists feeds for given clearance level
func (h *LiveFeedHTTPHandler) handleListFeeds(w http.ResponseWriter, r *http.Request) {
	clearanceStr := r.Header.Get("X-Clearance")
	clearance := ParseClearanceLevel(clearanceStr)

	feeds := h.feedManager.GetFeedsForClearance(clearance)
	json.NewEncoder(w).Encode(feeds)
}

// handleCreateFeed creates a new feed
func (h *LiveFeedHTTPHandler) handleCreateFeed(w http.ResponseWriter, r *http.Request) {
	var feed LiveFeed
	if err := json.NewDecoder(r.Body).Decode(&feed); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.feedManager.CreateFeed(&feed); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(feed)
}

// handleGetFeed gets a specific feed
func (h *LiveFeedHTTPHandler) handleGetFeed(w http.ResponseWriter, r *http.Request, feedID string) {
	feed, err := h.feedManager.GetFeed(feedID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Check clearance
	clearanceStr := r.Header.Get("X-Clearance")
	clearance := ParseClearanceLevel(clearanceStr)

	if clearance < feed.Clearance {
		http.Error(w, "Insufficient clearance", http.StatusForbidden)
		return
	}

	json.NewEncoder(w).Encode(feed)
}

// handleEndFeed ends a feed
func (h *LiveFeedHTTPHandler) handleEndFeed(w http.ResponseWriter, r *http.Request, feedID string) {
	if err := h.feedManager.EndFeed(feedID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleGetStats returns feed statistics
func (h *LiveFeedHTTPHandler) handleGetStats(w http.ResponseWriter, r *http.Request) {
	stats := struct {
		TotalFeeds       int `json:"totalFeeds"`
		ActiveFeeds      int `json:"activeFeeds"`
		ConnectedClients int `json:"connectedClients"`
	}{
		TotalFeeds:       len(h.feedManager.feeds),
		ActiveFeeds:      0,
		ConnectedClients: h.wsHub.GetConnectedClients(),
	}

	for _, feed := range h.feedManager.feeds {
		if feed.Status == "active" {
			stats.ActiveFeeds++
		}
	}

	json.NewEncoder(w).Encode(stats)
}
