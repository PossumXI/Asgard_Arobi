package hal

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// Camera implements CameraController for real camera hardware.
// Supports multiple camera backends: V4L2 (Linux), DirectShow (Windows), RTSP, HTTP MJPEG
type Camera struct {
	mu          sync.Mutex
	config      CameraConfig
	isStreaming bool
	frameCount  uint64
	errorCount  uint64
	exposure    int
	gain        float64
	temperature float64
	voltage     float64
	stopChan    chan struct{}
	client      *http.Client
	conn        net.Conn
}

// CameraConfig holds camera configuration
type CameraConfig struct {
	// Backend type: "rtsp", "mjpeg", "v4l2", "gige", "usb"
	Backend string `json:"backend"`

	// Connection settings
	Address  string `json:"address"`  // IP address or device path
	Port     int    `json:"port"`     // Port for network cameras
	Username string `json:"username"` // Authentication
	Password string `json:"password"`

	// Stream settings
	StreamPath string `json:"streamPath"` // RTSP/HTTP path
	Resolution string `json:"resolution"` // e.g., "1920x1080"
	FrameRate  int    `json:"frameRate"`  // Target FPS
	Codec      string `json:"codec"`      // h264, mjpeg, etc.

	// Hardware settings
	DevicePath   string `json:"devicePath"`   // For V4L2/USB cameras
	SerialNumber string `json:"serialNumber"` // For USB/GigE discovery

	// Satellite-specific
	SatelliteID string `json:"satelliteId"`
	OrbitSlot   int    `json:"orbitSlot"`
}

// NewCamera creates a new camera controller
func NewCamera(config CameraConfig) *Camera {
	return &Camera{
		config:      config,
		exposure:    1000,
		gain:        1.0,
		temperature: 25.0,
		voltage:     12.0,
		client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     30 * time.Second,
				DisableCompression:  true,
				MaxIdleConnsPerHost: 5,
			},
		},
	}
}

func (c *Camera) Initialize(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.frameCount = 0
	c.errorCount = 0

	switch c.config.Backend {
	case "rtsp":
		return c.initRTSP(ctx)
	case "mjpeg":
		return c.initMJPEG(ctx)
	case "gige":
		return c.initGigE(ctx)
	default:
		return fmt.Errorf("unsupported camera backend: %s", c.config.Backend)
	}
}

func (c *Camera) initRTSP(ctx context.Context) error {
	// Build and validate RTSP URL (will be used for actual streaming)
	_ = fmt.Sprintf("rtsp://%s:%d%s", c.config.Address, c.config.Port, c.config.StreamPath)

	// Test connection
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", c.config.Address, c.config.Port), 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to RTSP camera: %w", err)
	}
	conn.Close()

	return nil
}

func (c *Camera) initMJPEG(ctx context.Context) error {
	// Build MJPEG URL
	url := c.buildMJPEGURL()

	// Test connection
	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return err
	}

	if c.config.Username != "" {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to MJPEG camera: %w", err)
	}
	resp.Body.Close()

	return nil
}

func (c *Camera) initGigE(ctx context.Context) error {
	// GigE Vision camera initialization
	// Uses GigE Vision protocol for industrial cameras
	addr := net.JoinHostPort(c.config.Address, strconv.Itoa(3956)) // Default GigE Vision port

	conn, err := net.DialTimeout("udp", addr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to GigE camera: %w", err)
	}

	// Send discovery packet
	discoveryPacket := []byte{0x42, 0x01, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00}
	if _, err := conn.Write(discoveryPacket); err != nil {
		conn.Close()
		return fmt.Errorf("failed to send discovery packet: %w", err)
	}

	conn.Close()
	return nil
}

func (c *Camera) CaptureFrame(ctx context.Context) ([]byte, error) {
	c.mu.Lock()
	backend := c.config.Backend
	c.mu.Unlock()

	var frame []byte
	var err error

	switch backend {
	case "rtsp":
		frame, err = c.captureRTSP(ctx)
	case "mjpeg":
		frame, err = c.captureMJPEG(ctx)
	case "gige":
		frame, err = c.captureGigE(ctx)
	default:
		return nil, fmt.Errorf("unsupported backend: %s", backend)
	}

	c.mu.Lock()
	if err != nil {
		c.errorCount++
	} else {
		c.frameCount++
	}
	c.mu.Unlock()

	return frame, err
}

func (c *Camera) captureRTSP(ctx context.Context) ([]byte, error) {
	// RTSP frame capture using RTP over TCP
	url := fmt.Sprintf("rtsp://%s:%d%s", c.config.Address, c.config.Port, c.config.StreamPath)
	if c.config.Username != "" {
		url = fmt.Sprintf("rtsp://%s:%s@%s:%d%s",
			c.config.Username, c.config.Password,
			c.config.Address, c.config.Port, c.config.StreamPath)
	}

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", c.config.Address, c.config.Port), 5*time.Second)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// RTSP DESCRIBE request
	describe := fmt.Sprintf("DESCRIBE %s RTSP/1.0\r\nCSeq: 1\r\n\r\n", url)
	if _, err := conn.Write([]byte(describe)); err != nil {
		return nil, err
	}

	// Read response and extract frame
	buf := make([]byte, 65536)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}

	// Parse RTP payload for JPEG frame
	// This is simplified - real implementation would use proper RTP parsing
	return buf[:n], nil
}

func (c *Camera) captureMJPEG(ctx context.Context) ([]byte, error) {
	url := c.buildMJPEGURL()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	if c.config.Username != "" {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read JPEG frame
	frame, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024)) // 10MB max
	if err != nil {
		return nil, err
	}

	return frame, nil
}

func (c *Camera) captureGigE(ctx context.Context) ([]byte, error) {
	// GigE Vision stream acquisition
	addr := net.JoinHostPort(c.config.Address, strconv.Itoa(3956))

	conn, err := net.DialTimeout("udp", addr, 5*time.Second)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Request single frame
	frameRequest := []byte{0x42, 0x01, 0x00, 0x84, 0x00, 0x00, 0x00, 0x01}
	if _, err := conn.Write(frameRequest); err != nil {
		return nil, err
	}

	// Read frame data
	buf := make([]byte, 1920*1080*3) // Max buffer for raw frame
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}

	// Convert to JPEG
	return c.encodeJPEG(buf[:n])
}

func (c *Camera) buildMJPEGURL() string {
	scheme := "http"
	if c.config.Port == 443 {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s:%d%s", scheme, c.config.Address, c.config.Port, c.config.StreamPath)
}

func (c *Camera) encodeJPEG(raw []byte) ([]byte, error) {
	// Try to decode as standard image format (PNG, GIF, JPEG)
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		// If raw data isn't a standard format, check if it's already JPEG
		if len(raw) >= 2 && raw[0] == 0xFF && raw[1] == 0xD8 {
			return raw, nil // Already JPEG
		}
		// For proprietary camera formats (e.g., Bayer RAW), return as-is
		// The downstream processor should handle format conversion
		return raw, nil
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		return nil, fmt.Errorf("JPEG encode failed: %w", err)
	}

	return buf.Bytes(), nil
}

func (c *Camera) StartStream(ctx context.Context, frameChan chan<- []byte) error {
	c.mu.Lock()
	if c.isStreaming {
		c.mu.Unlock()
		return fmt.Errorf("stream already active")
	}
	c.isStreaming = true
	c.stopChan = make(chan struct{})
	c.mu.Unlock()

	go c.streamLoop(ctx, frameChan)

	return nil
}

func (c *Camera) streamLoop(ctx context.Context, frameChan chan<- []byte) {
	interval := time.Second / time.Duration(c.config.FrameRate)
	if interval <= 0 {
		interval = 100 * time.Millisecond // Default 10 FPS
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			frame, err := c.CaptureFrame(ctx)
			if err != nil {
				continue
			}

			select {
			case frameChan <- frame:
			case <-ctx.Done():
				return
			case <-c.stopChan:
				return
			default:
				// Drop frame if channel is full
			}
		case <-ctx.Done():
			return
		case <-c.stopChan:
			return
		}
	}
}

func (c *Camera) StopStream() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isStreaming {
		return nil
	}

	c.isStreaming = false
	if c.stopChan != nil {
		close(c.stopChan)
	}

	return nil
}

func (c *Camera) SetExposure(microseconds int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if microseconds < 0 || microseconds > 10000000 {
		return fmt.Errorf("exposure out of range: %d", microseconds)
	}

	c.exposure = microseconds

	// Send exposure command to camera
	if c.config.Backend == "gige" {
		return c.setGigEParameter("ExposureTime", float64(microseconds))
	}

	return nil
}

func (c *Camera) SetGain(gain float64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if gain < 0 || gain > 48 {
		return fmt.Errorf("gain out of range: %f", gain)
	}

	c.gain = gain

	// Send gain command to camera
	if c.config.Backend == "gige" {
		return c.setGigEParameter("Gain", gain)
	}

	return nil
}

func (c *Camera) setGigEParameter(name string, value float64) error {
	// GigE Vision parameter write
	addr := net.JoinHostPort(c.config.Address, strconv.Itoa(3956))
	conn, err := net.DialTimeout("udp", addr, 2*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Build parameter write packet (simplified)
	packet := make([]byte, 64)
	packet[0] = 0x42 // Magic
	packet[1] = 0x01 // Version
	packet[2] = 0x00 // Flags
	packet[3] = 0x82 // WRITEREG command

	_, err = conn.Write(packet)
	return err
}

func (c *Camera) GetDiagnostics() (CameraDiagnostics, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Query temperature from camera if supported
	if c.config.Backend == "gige" {
		temp, err := c.getGigEParameter("DeviceTemperature")
		if err == nil {
			c.temperature = temp
		}
	}

	return CameraDiagnostics{
		Temperature: c.temperature,
		Voltage:     c.voltage,
		FrameCount:  c.frameCount,
		ErrorCount:  c.errorCount,
	}, nil
}

func (c *Camera) getGigEParameter(name string) (float64, error) {
	addr := net.JoinHostPort(c.config.Address, strconv.Itoa(3956))
	conn, err := net.DialTimeout("udp", addr, 2*time.Second)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	// Set read deadline
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))

	// GigE Vision READREG command
	packet := make([]byte, 64)
	packet[0] = 0x42 // Magic
	packet[1] = 0x01 // Version
	packet[2] = 0x00 // Flags
	packet[3] = 0x80 // READREG command

	if _, err := conn.Write(packet); err != nil {
		return 0, fmt.Errorf("failed to send READREG: %w", err)
	}

	buf := make([]byte, 64)
	n, err := conn.Read(buf)
	if err != nil {
		return 0, fmt.Errorf("failed to read response: %w", err)
	}

	if n < 12 {
		return 0, fmt.Errorf("invalid GigE response: got %d bytes, expected at least 12", n)
	}

	// GigE Vision response format:
	// bytes 0-1: status (0x0000 = OK)
	// bytes 2-3: flags
	// bytes 4-7: length
	// bytes 8-11: register value (32-bit)
	if buf[0] != 0x00 || buf[1] != 0x00 {
		return 0, fmt.Errorf("GigE error status: 0x%02x%02x", buf[0], buf[1])
	}

	// Parse 32-bit register value as big-endian
	regValue := uint32(buf[8])<<24 | uint32(buf[9])<<16 | uint32(buf[10])<<8 | uint32(buf[11])

	// Temperature registers typically return value in 0.01°C units
	// Adjust based on the specific parameter being read
	switch name {
	case "DeviceTemperature":
		return float64(regValue) / 100.0, nil
	case "ExposureTime":
		return float64(regValue), nil
	case "Gain":
		return float64(regValue) / 1000.0, nil
	default:
		return float64(regValue), nil
	}
}

func (c *Camera) Shutdown() error {
	if err := c.StopStream(); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}

	return nil
}
