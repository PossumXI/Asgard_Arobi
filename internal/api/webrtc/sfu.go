package webrtc

import (
	"errors"
	"log"
	"sync"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
)

// SFU implements a Selective Forwarding Unit for WebRTC streaming.
type SFU struct {
	mu        sync.RWMutex
	sessions  map[string]*Session
	peers     map[string]*Peer
	api       *webrtc.API
	config    webrtc.Configuration
}

// Session represents a streaming session with multiple peers.
type Session struct {
	ID        string
	StreamID  string
	Peers     map[string]*Peer
	mu        sync.RWMutex
	audioTrack *webrtc.TrackLocalStaticSample
	videoTrack *webrtc.TrackLocalStaticSample
}

// Peer represents a WebRTC peer connection.
type Peer struct {
	ID           string
	Connection   *webrtc.PeerConnection
	AudioTrack   *webrtc.TrackRemote
	VideoTrack   *webrtc.TrackRemote
	DataChannel  *webrtc.DataChannel
	OnTrack      func(*webrtc.TrackRemote, *webrtc.RTPReceiver)
	OnDisconnect func()
}

// NewSFU creates a new SFU instance.
func NewSFU(config webrtc.Configuration) *SFU {
	mediaEngine := &webrtc.MediaEngine{}
	
	// Register codecs
	if err := mediaEngine.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:     webrtc.MimeTypeVP8,
			ClockRate:    90000,
			Channels:     0,
			SDPFmtpLine:  "",
			RTCPFeedback: nil,
		},
		PayloadType: 96,
	}, webrtc.RTPCodecTypeVideo); err != nil {
		log.Printf("Failed to register VP8: %v", err)
	}

	if err := mediaEngine.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:     webrtc.MimeTypeOpus,
			ClockRate:    48000,
			Channels:     2,
			SDPFmtpLine:  "",
			RTCPFeedback: nil,
		},
		PayloadType: 111,
	}, webrtc.RTPCodecTypeAudio); err != nil {
		log.Printf("Failed to register Opus: %v", err)
	}

	api := webrtc.NewAPI(webrtc.WithMediaEngine(mediaEngine))

	return &SFU{
		sessions: make(map[string]*Session),
		peers:    make(map[string]*Peer),
		api:      api,
		config:   config,
	}
}

// CreateSession creates a new streaming session.
func (sfu *SFU) CreateSession(sessionID, streamID string) *Session {
	sfu.mu.Lock()
	defer sfu.mu.Unlock()

	session := &Session{
		ID:       sessionID,
		StreamID: streamID,
		Peers:    make(map[string]*Peer),
	}

	sfu.sessions[sessionID] = session
	return session
}

// GetSession retrieves a session by ID.
func (sfu *SFU) GetSession(sessionID string) (*Session, bool) {
	sfu.mu.RLock()
	defer sfu.mu.RUnlock()
	session, ok := sfu.sessions[sessionID]
	return session, ok
}

// CreatePeerConnection creates a new WebRTC peer connection using the SFU's configuration.
func (sfu *SFU) CreatePeerConnection() (*webrtc.PeerConnection, error) {
	return sfu.api.NewPeerConnection(sfu.config)
}

// GetConfig returns the WebRTC configuration (useful for ICE servers).
func (sfu *SFU) GetConfig() webrtc.Configuration {
	return sfu.config
}

// AddPeer adds a peer to a session.
func (sfu *SFU) AddPeer(sessionID, peerID string, pc *webrtc.PeerConnection) (*Peer, error) {
	session, ok := sfu.GetSession(sessionID)
	if !ok {
		return nil, ErrSessionNotFound
	}

	peer := &Peer{
		ID:         peerID,
		Connection: pc,
	}

	session.mu.Lock()
	session.Peers[peerID] = peer
	session.mu.Unlock()

	sfu.mu.Lock()
	sfu.peers[peerID] = peer
	sfu.mu.Unlock()

	// Set up track forwarding
	pc.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
			if track.Kind() == webrtc.RTPCodecTypeAudio {
			peer.AudioTrack = track
		} else if track.Kind() == webrtc.RTPCodecTypeVideo {
			peer.VideoTrack = track
		}

		// Forward track to other peers in session
		sfu.forwardTrack(session, peerID, track, receiver)
	})

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		if state == webrtc.PeerConnectionStateClosed || state == webrtc.PeerConnectionStateFailed {
			sfu.RemovePeer(sessionID, peerID)
		}
	})

	return peer, nil
}

// RemovePeer removes a peer from a session.
func (sfu *SFU) RemovePeer(sessionID, peerID string) {
	session, ok := sfu.GetSession(sessionID)
	if !ok {
		return
	}

	session.mu.Lock()
	delete(session.Peers, peerID)
	session.mu.Unlock()

	sfu.mu.Lock()
	delete(sfu.peers, peerID)
	sfu.mu.Unlock()
}

// forwardTrack forwards a track from one peer to all other peers in the session.
func (sfu *SFU) forwardTrack(session *Session, sourcePeerID string, track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
	session.mu.RLock()
	defer session.mu.RUnlock()

	for peerID, peer := range session.Peers {
		if peerID == sourcePeerID {
			continue // Don't forward to self
		}

		// Create a track for this peer (use RTP track for raw packet forwarding)
		trackLocal, err := webrtc.NewTrackLocalStaticRTP(
			track.Codec().RTPCodecCapability,
			track.ID(),
			track.StreamID(),
		)
		if err != nil {
			log.Printf("Failed to create track local: %v", err)
			continue
		}

		// Add track to peer connection
		if _, err := peer.Connection.AddTrack(trackLocal); err != nil {
			log.Printf("Failed to add track to peer: %v", err)
			continue
		}

		// Forward RTP packets
		go func(p *Peer, t *webrtc.TrackRemote, tl *webrtc.TrackLocalStaticRTP) {
			buf := make([]byte, 1500)
			for {
				n, _, err := t.Read(buf)
				if err != nil {
					return
				}

				// Parse the RTP packet and write it
				packet := &rtp.Packet{}
				if err := packet.Unmarshal(buf[:n]); err != nil {
					log.Printf("Failed to unmarshal RTP packet: %v", err)
					continue
				}

				if err := tl.WriteRTP(packet); err != nil {
					return
				}
			}
		}(peer, track, trackLocal)
	}
}

// CreateOffer creates a WebRTC offer for a peer.
func (sfu *SFU) CreateOffer(peer *Peer) (*webrtc.SessionDescription, error) {
	offer, err := peer.Connection.CreateOffer(nil)
	if err != nil {
		return nil, err
	}

	if err := peer.Connection.SetLocalDescription(offer); err != nil {
		return nil, err
	}

	return &offer, nil
}

// SetRemoteDescription sets the remote description for a peer.
func (sfu *SFU) SetRemoteDescription(peer *Peer, desc webrtc.SessionDescription) error {
	return peer.Connection.SetRemoteDescription(desc)
}

// AddICECandidate adds an ICE candidate to a peer.
func (sfu *SFU) AddICECandidate(peer *Peer, candidate webrtc.ICECandidateInit) error {
	return peer.Connection.AddICECandidate(candidate)
}

// GetOrCreateSession gets an existing session or creates a new one by streamID.
func (sfu *SFU) GetOrCreateSession(streamID string) *Session {
	sfu.mu.RLock()
	// Check if session exists by streamID
	for _, session := range sfu.sessions {
		if session.StreamID == streamID {
			sfu.mu.RUnlock()
			return session
		}
	}
	sfu.mu.RUnlock()

	// Create new session
	sfu.mu.Lock()
	defer sfu.mu.Unlock()
	
	sessionID := streamID // Use streamID as sessionID
	session := &Session{
		ID:       sessionID,
		StreamID: streamID,
		Peers:    make(map[string]*Peer),
	}
	sfu.sessions[sessionID] = session
	return session
}

// AddPeerToSession adds a peer to a session by streamID, creating peer connection.
func (sfu *SFU) AddPeerToSession(streamID, peerID string) (*Peer, error) {
	session := sfu.GetOrCreateSession(streamID)
	
	// Create peer connection
	pc, err := sfu.api.NewPeerConnection(sfu.config)
	if err != nil {
		return nil, err
	}

	return sfu.AddPeer(session.ID, peerID, pc)
}

// GetPeer gets a peer by ID from any session.
func (sfu *SFU) GetPeer(peerID string) (*Peer, bool) {
	sfu.mu.RLock()
	defer sfu.mu.RUnlock()
	peer, ok := sfu.peers[peerID]
	return peer, ok
}

// CreateOfferForPeer creates an offer for a peer.
func (sfu *SFU) CreateOfferForPeer(peerID string) (*webrtc.SessionDescription, error) {
	peer, ok := sfu.GetPeer(peerID)
	if !ok {
		return nil, ErrPeerNotFound
	}
	return sfu.CreateOffer(peer)
}

// SetRemoteDescriptionForPeer sets remote description for a peer.
func (sfu *SFU) SetRemoteDescriptionForPeer(peerID string, desc webrtc.SessionDescription) error {
	peer, ok := sfu.GetPeer(peerID)
	if !ok {
		return ErrPeerNotFound
	}
	return sfu.SetRemoteDescription(peer, desc)
}

// AddICECandidateForPeer adds ICE candidate for a peer.
func (sfu *SFU) AddICECandidateForPeer(peerID string, candidate webrtc.ICECandidateInit) error {
	peer, ok := sfu.GetPeer(peerID)
	if !ok {
		return ErrPeerNotFound
	}
	return sfu.AddICECandidate(peer, candidate)
}

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrPeerNotFound    = errors.New("peer not found")
)
