package dtn

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/asgard/pandora/pkg/bundle"
)

// TransportAdapter defines the interface for DTN transport implementations.
// Different transports can be used for TCP, UDP, satellite links, RF, etc.
type TransportAdapter interface {
	// Send transmits a bundle to the specified neighbor node.
	Send(ctx context.Context, neighborID string, b *bundle.Bundle) error

	// Receive returns a channel that delivers incoming bundles.
	// The channel is closed when the transport is disconnected.
	Receive() <-chan *bundle.Bundle

	// Connect establishes a connection to a neighbor node.
	Connect(ctx context.Context, neighborID string, address string) error

	// Disconnect closes the connection to a neighbor node.
	Disconnect(neighborID string) error

	// IsConnected returns true if connected to the specified neighbor.
	IsConnected(neighborID string) bool

	// Start initializes the transport and begins listening for incoming connections.
	Start(ctx context.Context) error

	// Stop gracefully shuts down the transport.
	Stop() error
}

// TCPTransportConfig holds configuration for TCP transport.
type TCPTransportConfig struct {
	ListenAddress    string        // Address to listen on (e.g., ":4556")
	ConnectTimeout   time.Duration // Timeout for connection attempts
	ReadTimeout      time.Duration // Timeout for read operations
	WriteTimeout     time.Duration // Timeout for write operations
	MaxMessageSize   int           // Maximum bundle size in bytes
	ReconnectBackoff time.Duration // Initial backoff for reconnection attempts
	MaxReconnects    int           // Maximum reconnection attempts (0 = unlimited)
}

// DefaultTCPTransportConfig returns sensible defaults for TCP transport.
func DefaultTCPTransportConfig() TCPTransportConfig {
	return TCPTransportConfig{
		ListenAddress:    ":4556",
		ConnectTimeout:   30 * time.Second,
		ReadTimeout:      60 * time.Second,
		WriteTimeout:     30 * time.Second,
		MaxMessageSize:   10 * 1024 * 1024, // 10MB
		ReconnectBackoff: 1 * time.Second,
		MaxReconnects:    10,
	}
}

// TCPTransport implements TransportAdapter using TCP connections.
type TCPTransport struct {
	config      TCPTransportConfig
	localNodeID string

	// Connection management
	connections map[string]*tcpConnection
	connMu      sync.RWMutex

	// Listener for incoming connections
	listener net.Listener

	// Channel for received bundles
	receiveChan chan *bundle.Bundle

	// Lifecycle management
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// tcpConnection represents a single TCP connection to a neighbor.
type tcpConnection struct {
	neighborID string
	address    string
	conn       net.Conn
	mu         sync.Mutex
	active     bool
}

// NewTCPTransport creates a new TCP transport instance.
func NewTCPTransport(localNodeID string, config TCPTransportConfig) *TCPTransport {
	ctx, cancel := context.WithCancel(context.Background())

	return &TCPTransport{
		config:      config,
		localNodeID: localNodeID,
		connections: make(map[string]*tcpConnection),
		receiveChan: make(chan *bundle.Bundle, 1000),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start initializes the TCP transport and begins listening for connections.
func (t *TCPTransport) Start(ctx context.Context) error {
	var err error
	t.listener, err = net.Listen("tcp", t.config.ListenAddress)
	if err != nil {
		return fmt.Errorf("failed to start TCP listener: %w", err)
	}

	log.Printf("[TCP Transport] Listening on %s", t.config.ListenAddress)

	// Start accepting incoming connections
	t.wg.Add(1)
	go t.acceptLoop()

	return nil
}

// Stop gracefully shuts down the TCP transport.
func (t *TCPTransport) Stop() error {
	log.Printf("[TCP Transport] Shutting down")
	t.cancel()

	// Close listener
	if t.listener != nil {
		t.listener.Close()
	}

	// Close all connections
	t.connMu.Lock()
	for id, conn := range t.connections {
		if conn.conn != nil {
			conn.conn.Close()
		}
		delete(t.connections, id)
	}
	t.connMu.Unlock()

	// Wait for goroutines to finish
	t.wg.Wait()

	// Close receive channel
	close(t.receiveChan)

	return nil
}

// Connect establishes a TCP connection to a neighbor node.
func (t *TCPTransport) Connect(ctx context.Context, neighborID string, address string) error {
	t.connMu.Lock()
	defer t.connMu.Unlock()

	// Check if already connected
	if existing, ok := t.connections[neighborID]; ok && existing.active {
		return nil // Already connected
	}

	// Establish connection with timeout
	dialer := net.Dialer{Timeout: t.config.ConnectTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("failed to connect to %s at %s: %w", neighborID, address, err)
	}

	tcpConn := &tcpConnection{
		neighborID: neighborID,
		address:    address,
		conn:       conn,
		active:     true,
	}

	t.connections[neighborID] = tcpConn

	// Start receiver for this connection
	t.wg.Add(1)
	go t.handleConnection(tcpConn)

	log.Printf("[TCP Transport] Connected to neighbor %s at %s", neighborID, address)

	return nil
}

// Disconnect closes the connection to a neighbor node.
func (t *TCPTransport) Disconnect(neighborID string) error {
	t.connMu.Lock()
	defer t.connMu.Unlock()

	conn, ok := t.connections[neighborID]
	if !ok {
		return fmt.Errorf("not connected to neighbor: %s", neighborID)
	}

	conn.mu.Lock()
	conn.active = false
	if conn.conn != nil {
		conn.conn.Close()
	}
	conn.mu.Unlock()

	delete(t.connections, neighborID)

	log.Printf("[TCP Transport] Disconnected from neighbor %s", neighborID)

	return nil
}

// IsConnected returns true if connected to the specified neighbor.
func (t *TCPTransport) IsConnected(neighborID string) bool {
	t.connMu.RLock()
	defer t.connMu.RUnlock()

	conn, ok := t.connections[neighborID]
	if !ok {
		return false
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	return conn.active
}

// Send transmits a bundle to the specified neighbor.
func (t *TCPTransport) Send(ctx context.Context, neighborID string, b *bundle.Bundle) error {
	t.connMu.RLock()
	conn, ok := t.connections[neighborID]
	t.connMu.RUnlock()

	if !ok || !conn.active {
		return fmt.Errorf("not connected to neighbor: %s", neighborID)
	}

	// Serialize the bundle
	data, err := bundle.Marshal(b)
	if err != nil {
		return fmt.Errorf("failed to serialize bundle: %w", err)
	}

	// Check message size
	if len(data) > t.config.MaxMessageSize {
		return fmt.Errorf("bundle size %d exceeds maximum %d", len(data), t.config.MaxMessageSize)
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	if !conn.active || conn.conn == nil {
		return fmt.Errorf("connection to %s is not active", neighborID)
	}

	// Set write deadline
	conn.conn.SetWriteDeadline(time.Now().Add(t.config.WriteTimeout))

	// Write message length (4 bytes, big-endian)
	lengthBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBuf, uint32(len(data)))
	if _, err := conn.conn.Write(lengthBuf); err != nil {
		conn.active = false
		return fmt.Errorf("failed to write message length: %w", err)
	}

	// Write message data
	if _, err := conn.conn.Write(data); err != nil {
		conn.active = false
		return fmt.Errorf("failed to write message data: %w", err)
	}

	log.Printf("[TCP Transport] Sent bundle %s to %s (%d bytes)",
		b.ID.String()[:8], neighborID, len(data))

	return nil
}

// Receive returns the channel for incoming bundles.
func (t *TCPTransport) Receive() <-chan *bundle.Bundle {
	return t.receiveChan
}

// acceptLoop handles incoming TCP connections.
func (t *TCPTransport) acceptLoop() {
	defer t.wg.Done()

	for {
		select {
		case <-t.ctx.Done():
			return
		default:
		}

		// Set accept timeout to allow checking context
		if tcpListener, ok := t.listener.(*net.TCPListener); ok {
			tcpListener.SetDeadline(time.Now().Add(1 * time.Second))
		}

		conn, err := t.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			// Check if it's a timeout error
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			log.Printf("[TCP Transport] Accept error: %v", err)
			continue
		}

		// Generate a temporary neighbor ID for incoming connections
		// In practice, the neighbor would identify itself via a handshake
		remoteAddr := conn.RemoteAddr().String()
		neighborID := fmt.Sprintf("incoming-%s", remoteAddr)

		tcpConn := &tcpConnection{
			neighborID: neighborID,
			address:    remoteAddr,
			conn:       conn,
			active:     true,
		}

		t.connMu.Lock()
		t.connections[neighborID] = tcpConn
		t.connMu.Unlock()

		log.Printf("[TCP Transport] Accepted connection from %s", remoteAddr)

		t.wg.Add(1)
		go t.handleConnection(tcpConn)
	}
}

// handleConnection reads bundles from a TCP connection.
func (t *TCPTransport) handleConnection(conn *tcpConnection) {
	defer t.wg.Done()
	defer func() {
		conn.mu.Lock()
		conn.active = false
		if conn.conn != nil {
			conn.conn.Close()
		}
		conn.mu.Unlock()
	}()

	reader := bufio.NewReader(conn.conn)

	for {
		select {
		case <-t.ctx.Done():
			return
		default:
		}

		// Set read deadline
		conn.conn.SetReadDeadline(time.Now().Add(t.config.ReadTimeout))

		// Read message length (4 bytes)
		lengthBuf := make([]byte, 4)
		_, err := io.ReadFull(reader, lengthBuf)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				log.Printf("[TCP Transport] Connection closed by %s", conn.neighborID)
				return
			}
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue // Timeout, try again
			}
			log.Printf("[TCP Transport] Read error from %s: %v", conn.neighborID, err)
			return
		}

		messageLen := binary.BigEndian.Uint32(lengthBuf)

		// Validate message size
		if int(messageLen) > t.config.MaxMessageSize {
			log.Printf("[TCP Transport] Message too large from %s: %d bytes", conn.neighborID, messageLen)
			return
		}

		// Read message data
		data := make([]byte, messageLen)
		_, err = io.ReadFull(reader, data)
		if err != nil {
			log.Printf("[TCP Transport] Failed to read message from %s: %v", conn.neighborID, err)
			return
		}

		// Deserialize the bundle
		b, err := bundle.Unmarshal(data)
		if err != nil {
			log.Printf("[TCP Transport] Failed to unmarshal bundle from %s: %v", conn.neighborID, err)
			continue
		}

		log.Printf("[TCP Transport] Received bundle %s from %s (%d bytes)",
			b.ID.String()[:8], conn.neighborID, messageLen)

		// Send to receive channel
		select {
		case t.receiveChan <- b:
		case <-t.ctx.Done():
			return
		default:
			log.Printf("[TCP Transport] Receive buffer full, dropping bundle %s", b.ID.String()[:8])
		}
	}
}

// GetConnectionStats returns statistics about current connections.
func (t *TCPTransport) GetConnectionStats() map[string]bool {
	t.connMu.RLock()
	defer t.connMu.RUnlock()

	stats := make(map[string]bool, len(t.connections))
	for id, conn := range t.connections {
		conn.mu.Lock()
		stats[id] = conn.active
		conn.mu.Unlock()
	}
	return stats
}
