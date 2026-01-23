// Package signaling provides WebRTC signaling server for Hubs streaming.
package signaling

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/asgard/pandora/internal/services"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

// Server handles WebRTC signaling for stream sessions.
type Server struct {
	streamService *services.StreamService
	sessions      map[string]*Session
	mu            sync.RWMutex
}

// Session represents a WebRTC signaling session.
type Session struct {
	StreamID  string
	SessionID string
	Conn      *websocket.Conn
}

// NewServer creates a new signaling server.
func NewServer(streamService *services.StreamService) *Server {
	return &Server{
		streamService: streamService,
		sessions:       make(map[string]*Session),
	}
}

// HandleWebSocket handles WebSocket connections for WebRTC signaling.
func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// Read messages from client
	for {
		var msg map[string]interface{}
		if err := conn.ReadJSON(&msg); err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}

		msgType, ok := msg["type"].(string)
		if !ok {
			continue
		}

		switch msgType {
		case "join":
			sessionID, _ := msg["sessionId"].(string)
			streamID, _ := msg["streamId"].(string)
			s.handleJoin(conn, streamID, sessionID)

		case "offer":
			s.handleOffer(conn, msg)

		case "answer":
			s.handleAnswer(conn, msg)

		case "ice-candidate":
			s.handleICECandidate(conn, msg)

		default:
			log.Printf("Unknown message type: %s", msgType)
		}
	}
}

func (s *Server) handleJoin(conn *websocket.Conn, streamID, sessionID string) {
	s.mu.Lock()
	s.sessions[sessionID] = &Session{
		StreamID:  streamID,
		SessionID: sessionID,
		Conn:      conn,
	}
	s.mu.Unlock()

	log.Printf("Client joined stream %s with session %s", streamID, sessionID)

	// Send offer to client (in production, this would come from media server)
	offer := map[string]interface{}{
		"type": "offer",
		"sdp": map[string]interface{}{
			"type": "offer",
			"sdp":  "mock_sdp_offer",
		},
	}

	if err := conn.WriteJSON(offer); err != nil {
		log.Printf("Error sending offer: %v", err)
	}
}

func (s *Server) handleOffer(conn *websocket.Conn, msg map[string]interface{}) {
	// In production, forward offer to media server
	log.Printf("Received offer")
}

func (s *Server) handleAnswer(conn *websocket.Conn, msg map[string]interface{}) {
	// In production, forward answer to media server
	log.Printf("Received answer")
}

func (s *Server) handleICECandidate(conn *websocket.Conn, msg map[string]interface{}) {
	// In production, forward ICE candidate to peer
	log.Printf("Received ICE candidate")
}

// BroadcastToSession sends a message to a specific session.
func (s *Server) BroadcastToSession(sessionID string, message interface{}) error {
	s.mu.RLock()
	session, ok := s.sessions[sessionID]
	s.mu.RUnlock()

	if !ok {
		return nil
	}

	return session.Conn.WriteJSON(message)
}
