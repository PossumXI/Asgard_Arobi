// Package signaling provides WebRTC signaling server for Hubs streaming.
package signaling

import (
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/asgard/pandora/internal/api/webrtc"
	"github.com/asgard/pandora/internal/services"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	pionwebrtc "github.com/pion/webrtc/v4"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

// Server handles WebRTC signaling for stream sessions.
type Server struct {
	streamService *services.StreamService
	sfu           *webrtc.SFU
	sessions      map[string]*SignalingSession
	mu            sync.RWMutex
}

// SignalingSession represents a WebRTC signaling session with peer connection.
type SignalingSession struct {
	StreamID   string
	SessionID  string
	PeerID     string
	Role       string
	Conn       *websocket.Conn
	SFUSession *webrtc.Session
	SFUPeer    *webrtc.Peer
	mu         sync.Mutex
}

// NewServer creates a new signaling server with the provided SFU.
// If sfu is nil, it will be created with default configuration.
func NewServer(streamService *services.StreamService, sfu *webrtc.SFU) *Server {
	if sfu == nil {
		// Create default SFU configuration with STUN/TURN servers
		config := pionwebrtc.Configuration{
			ICEServers: []pionwebrtc.ICEServer{
				{
					URLs: []string{
						"stun:stun.l.google.com:19302",
						"stun:stun1.l.google.com:19302",
					},
				},
			},
		}

		// Add TURN server if configured
		if turnURL := os.Getenv("TURN_SERVER"); turnURL != "" {
			turnUsername := os.Getenv("TURN_USERNAME")
			turnPassword := os.Getenv("TURN_PASSWORD")
			config.ICEServers = append(config.ICEServers, pionwebrtc.ICEServer{
				URLs:       []string{turnURL},
				Username:   turnUsername,
				Credential: turnPassword,
			})
		}

		sfu = webrtc.NewSFU(config)
	}

	return &Server{
		streamService: streamService,
		sfu:           sfu,
		sessions:      make(map[string]*SignalingSession),
	}
}

// GetSFU returns the SFU instance for external use (e.g., getting ICE servers).
func (s *Server) GetSFU() *webrtc.SFU {
	return s.sfu
}

// HandleWebSocket handles WebSocket connections for WebRTC signaling.
func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()
	sessionToken := r.URL.Query().Get("token")

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
			role, _ := msg["role"].(string)
			s.handleJoin(conn, streamID, sessionID, sessionToken, role)

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

	// Clean up session on disconnect
	s.cleanupConnection(conn)
}

func (s *Server) handleJoin(conn *websocket.Conn, streamID, sessionID, token, role string) {
	if role == "" {
		role = "subscriber"
	}
	if s.streamService != nil {
		storedStreamID, _, ok := s.streamService.ValidateSession(sessionID, token)
		if !ok {
			log.Printf("Invalid or expired session: %s", sessionID)
			s.sendError(conn, "Invalid or expired session")
			return
		}
		if storedStreamID != streamID {
			log.Printf("Session stream mismatch: %s != %s", storedStreamID, streamID)
			s.sendError(conn, "Session does not match stream")
			return
		}
	}

	// Generate a unique peer ID for this connection
	peerID := uuid.New().String()

	// Get or create SFU session for this stream
	sfuSession := s.sfu.CreateSession(sessionID, streamID)
	if existingSession, ok := s.sfu.GetSession(sessionID); ok {
		sfuSession = existingSession
	}

	// Create peer connection using the SFU's API
	pc, err := s.sfu.CreatePeerConnection()
	if err != nil {
		log.Printf("Failed to create peer connection: %v", err)
		s.sendError(conn, "Failed to create peer connection")
		return
	}

	// Add peer to the SFU session
	sfuPeer, err := s.sfu.AddPeer(sessionID, peerID, pc)
	if err != nil {
		log.Printf("Failed to add peer to session: %v", err)
		pc.Close()
		s.sendError(conn, "Failed to add peer to session")
		return
	}

	// Store the signaling session
	signalingSession := &SignalingSession{
		StreamID:   streamID,
		SessionID:  sessionID,
		PeerID:     peerID,
		Role:       role,
		Conn:       conn,
		SFUSession: sfuSession,
		SFUPeer:    sfuPeer,
	}

	s.mu.Lock()
	s.sessions[sessionID] = signalingSession
	s.mu.Unlock()

	log.Printf("Client joined stream %s with session %s (peer %s)", streamID, sessionID, peerID)

	// Set up ICE candidate handler to forward candidates to the client
	pc.OnICECandidate(func(candidate *pionwebrtc.ICECandidate) {
		if candidate == nil {
			return
		}

		candidateInit := candidate.ToJSON()
		msg := map[string]interface{}{
			"type":      "ice-candidate",
			"sessionId": sessionID,
			"candidate": map[string]interface{}{
				"candidate":     candidateInit.Candidate,
				"sdpMLineIndex": candidateInit.SDPMLineIndex,
				"sdpMid":        candidateInit.SDPMid,
			},
		}

		signalingSession.mu.Lock()
		if err := conn.WriteJSON(msg); err != nil {
			log.Printf("Error sending ICE candidate: %v", err)
		}
		signalingSession.mu.Unlock()
	})

	// Set up connection state handler for logging
	pc.OnConnectionStateChange(func(state pionwebrtc.PeerConnectionState) {
		log.Printf("Peer %s connection state: %s", peerID, state.String())
		if state == pionwebrtc.PeerConnectionStateFailed ||
			state == pionwebrtc.PeerConnectionStateClosed {
			s.removeSession(sessionID)
		}
	})

	if role == "publisher" {
		readyMsg := map[string]interface{}{
			"type":      "ready",
			"sessionId": sessionID,
		}
		if err := conn.WriteJSON(readyMsg); err != nil {
			log.Printf("Error sending ready: %v", err)
		}
		return
	}

	// Create an offer to send to the client (subscriber mode)
	offer, err := pc.CreateOffer(nil)
	if err != nil {
		log.Printf("Failed to create offer: %v", err)
		s.sendError(conn, "Failed to create offer")
		return
	}

	// Set local description
	if err := pc.SetLocalDescription(offer); err != nil {
		log.Printf("Failed to set local description: %v", err)
		s.sendError(conn, "Failed to set local description")
		return
	}

	// Send offer to client
	offerMsg := map[string]interface{}{
		"type":      "offer",
		"sessionId": sessionID,
		"sdp": map[string]interface{}{
			"type": offer.Type.String(),
			"sdp":  offer.SDP,
		},
	}

	if err := conn.WriteJSON(offerMsg); err != nil {
		log.Printf("Error sending offer: %v", err)
	}
}

func (s *Server) handleOffer(conn *websocket.Conn, msg map[string]interface{}) {
	sessionID, _ := msg["sessionId"].(string)
	sdpData, ok := msg["sdp"].(map[string]interface{})
	if !ok {
		log.Printf("Invalid offer format")
		s.sendError(conn, "Invalid offer format")
		return
	}

	sdpType, _ := sdpData["type"].(string)
	sdpStr, _ := sdpData["sdp"].(string)

	// Get the session
	s.mu.RLock()
	session, ok := s.sessions[sessionID]
	s.mu.RUnlock()
	if !ok {
		log.Printf("Session not found: %s", sessionID)
		s.sendError(conn, "Session not found")
		return
	}

	// Parse SDP type
	var parsedType pionwebrtc.SDPType
	switch sdpType {
	case "offer":
		parsedType = pionwebrtc.SDPTypeOffer
	case "pranswer":
		parsedType = pionwebrtc.SDPTypePranswer
	case "answer":
		parsedType = pionwebrtc.SDPTypeAnswer
	case "rollback":
		parsedType = pionwebrtc.SDPTypeRollback
	default:
		log.Printf("Unknown SDP type: %s", sdpType)
		s.sendError(conn, "Invalid SDP type")
		return
	}

	offer := pionwebrtc.SessionDescription{
		Type: parsedType,
		SDP:  sdpStr,
	}

	// Set remote description
	if err := session.SFUPeer.Connection.SetRemoteDescription(offer); err != nil {
		log.Printf("Failed to set remote description: %v", err)
		s.sendError(conn, "Failed to set remote description")
		return
	}

	// Create answer
	answer, err := session.SFUPeer.Connection.CreateAnswer(nil)
	if err != nil {
		log.Printf("Failed to create answer: %v", err)
		s.sendError(conn, "Failed to create answer")
		return
	}

	// Set local description
	if err := session.SFUPeer.Connection.SetLocalDescription(answer); err != nil {
		log.Printf("Failed to set local description: %v", err)
		s.sendError(conn, "Failed to set local description")
		return
	}

	// Send answer
	answerMsg := map[string]interface{}{
		"type":      "answer",
		"sessionId": sessionID,
		"sdp": map[string]interface{}{
			"type": answer.Type.String(),
			"sdp":  answer.SDP,
		},
	}

	session.mu.Lock()
	if err := conn.WriteJSON(answerMsg); err != nil {
		log.Printf("Error sending answer: %v", err)
	}
	session.mu.Unlock()

	log.Printf("Processed offer and sent answer for session %s", sessionID)
}

func (s *Server) handleAnswer(conn *websocket.Conn, msg map[string]interface{}) {
	sessionID, _ := msg["sessionId"].(string)
	sdpData, ok := msg["sdp"].(map[string]interface{})
	if !ok {
		log.Printf("Invalid answer format")
		s.sendError(conn, "Invalid answer format")
		return
	}

	sdpType, _ := sdpData["type"].(string)
	sdpStr, _ := sdpData["sdp"].(string)

	// Get the session
	s.mu.RLock()
	session, ok := s.sessions[sessionID]
	s.mu.RUnlock()
	if !ok {
		log.Printf("Session not found: %s", sessionID)
		s.sendError(conn, "Session not found")
		return
	}

	// Parse SDP type
	var parsedType pionwebrtc.SDPType
	switch sdpType {
	case "answer":
		parsedType = pionwebrtc.SDPTypeAnswer
	case "pranswer":
		parsedType = pionwebrtc.SDPTypePranswer
	default:
		log.Printf("Invalid answer SDP type: %s", sdpType)
		s.sendError(conn, "Invalid SDP type for answer")
		return
	}

	answer := pionwebrtc.SessionDescription{
		Type: parsedType,
		SDP:  sdpStr,
	}

	// Set remote description
	if err := session.SFUPeer.Connection.SetRemoteDescription(answer); err != nil {
		log.Printf("Failed to set remote description: %v", err)
		s.sendError(conn, "Failed to set remote description")
		return
	}

	log.Printf("Answer processed for session %s", sessionID)
}

func (s *Server) handleICECandidate(conn *websocket.Conn, msg map[string]interface{}) {
	sessionID, _ := msg["sessionId"].(string)
	candidateData, ok := msg["candidate"].(map[string]interface{})
	if !ok {
		log.Printf("Invalid ICE candidate format")
		s.sendError(conn, "Invalid ICE candidate format")
		return
	}

	candidateStr, _ := candidateData["candidate"].(string)
	sdpMid, _ := candidateData["sdpMid"].(string)

	// SDPMLineIndex can be float64 or nil from JSON
	var sdpMLineIndex *uint16
	if idx, ok := candidateData["sdpMLineIndex"].(float64); ok {
		val := uint16(idx)
		sdpMLineIndex = &val
	}

	// Get the session
	s.mu.RLock()
	session, ok := s.sessions[sessionID]
	s.mu.RUnlock()
	if !ok {
		log.Printf("Session not found: %s", sessionID)
		s.sendError(conn, "Session not found")
		return
	}

	candidate := pionwebrtc.ICECandidateInit{
		Candidate:     candidateStr,
		SDPMid:        &sdpMid,
		SDPMLineIndex: sdpMLineIndex,
	}

	// Add ICE candidate to the peer connection
	if err := session.SFUPeer.Connection.AddICECandidate(candidate); err != nil {
		log.Printf("Failed to add ICE candidate: %v", err)
		s.sendError(conn, "Failed to add ICE candidate")
		return
	}

	log.Printf("ICE candidate processed for session %s", sessionID)
}

func (s *Server) sendError(conn *websocket.Conn, message string) {
	errMsg := map[string]interface{}{
		"type":    "error",
		"message": message,
	}
	if err := conn.WriteJSON(errMsg); err != nil {
		log.Printf("Error sending error message: %v", err)
	}
}

func (s *Server) removeSession(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if session, ok := s.sessions[sessionID]; ok {
		if session.SFUPeer != nil && session.SFUPeer.Connection != nil {
			session.SFUPeer.Connection.Close()
		}
		s.sfu.RemovePeer(sessionID, session.PeerID)
		delete(s.sessions, sessionID)
		log.Printf("Removed session %s", sessionID)
	}
}

func (s *Server) cleanupConnection(conn *websocket.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for sessionID, session := range s.sessions {
		if session.Conn == conn {
			if session.SFUPeer != nil && session.SFUPeer.Connection != nil {
				session.SFUPeer.Connection.Close()
			}
			s.sfu.RemovePeer(sessionID, session.PeerID)
			delete(s.sessions, sessionID)
			log.Printf("Cleaned up session %s on disconnect", sessionID)
			return
		}
	}
}

// BroadcastToSession sends a message to a specific session.
func (s *Server) BroadcastToSession(sessionID string, message interface{}) error {
	s.mu.RLock()
	session, ok := s.sessions[sessionID]
	s.mu.RUnlock()

	if !ok {
		return nil
	}

	session.mu.Lock()
	defer session.mu.Unlock()
	return session.Conn.WriteJSON(message)
}
