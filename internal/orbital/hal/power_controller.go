package hal

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net"
	"sync"
	"time"
)

// SatellitePowerController implements PowerController for real satellite power systems.
// Communicates with power management units via SpaceWire, CAN, I2C, or TCP.
type SatellitePowerController struct {
	mu            sync.RWMutex
	config        PowerConfig
	conn          io.ReadWriteCloser
	mode          PowerMode
	lastTelemetry PowerTelemetry
	stopChan      chan struct{}
}

// PowerConfig holds power controller configuration
type PowerConfig struct {
	// Communication protocol: "spacewire", "can", "i2c", "tcp", "serial"
	Protocol string `json:"protocol"`

	// Connection settings
	Address    string `json:"address"`     // IP address or device path
	Port       int    `json:"port"`        // TCP port
	BusID      int    `json:"busId"`       // I2C/CAN bus ID
	DeviceAddr int    `json:"deviceAddr"`  // I2C device address

	// Hardware configuration
	BatteryCapacity  float64 `json:"batteryCapacity"`  // Wh
	SolarPanelPower  float64 `json:"solarPanelPower"`  // W nominal
	NumBatteryCells  int     `json:"numBatteryCells"`
	CellVoltageMin   float64 `json:"cellVoltageMin"`   // V
	CellVoltageMax   float64 `json:"cellVoltageMax"`   // V
	
	// Satellite-specific
	SatelliteID  string `json:"satelliteId"`
	OrbitalPeriod time.Duration `json:"orbitalPeriod"` // Typical LEO: 90-100 min
}

// PowerTelemetry contains real-time power telemetry
type PowerTelemetry struct {
	BatteryVoltage      float64   `json:"batteryVoltage"`      // V
	BatteryCurrent      float64   `json:"batteryCurrent"`      // A
	BatteryTemperature  float64   `json:"batteryTemperature"`  // C
	BatteryPercent      float64   `json:"batteryPercent"`      // %
	SolarPanelVoltage   float64   `json:"solarPanelVoltage"`   // V
	SolarPanelCurrent   float64   `json:"solarPanelCurrent"`   // A
	SolarPanelPower     float64   `json:"solarPanelPower"`     // W
	TotalPowerDraw      float64   `json:"totalPowerDraw"`      // W
	InEclipse           bool      `json:"inEclipse"`
	ChargeState         string    `json:"chargeState"`         // charging, discharging, full
	Timestamp           time.Time `json:"timestamp"`
}

// NewSatellitePowerController creates a new power controller
func NewSatellitePowerController(config PowerConfig) *SatellitePowerController {
	if config.OrbitalPeriod == 0 {
		config.OrbitalPeriod = 92 * time.Minute // Default LEO
	}
	if config.CellVoltageMin == 0 {
		config.CellVoltageMin = 3.0
	}
	if config.CellVoltageMax == 0 {
		config.CellVoltageMax = 4.2
	}
	if config.NumBatteryCells == 0 {
		config.NumBatteryCells = 4
	}

	return &SatellitePowerController{
		config: config,
		mode:   PowerModeNormal,
	}
}

// Initialize establishes connection to the power management unit
func (p *SatellitePowerController) Initialize(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var err error
	switch p.config.Protocol {
	case "tcp":
		err = p.initTCP(ctx)
	case "spacewire":
		err = p.initSpaceWire(ctx)
	case "can":
		err = p.initCAN(ctx)
	default:
		return fmt.Errorf("unsupported protocol: %s", p.config.Protocol)
	}

	if err != nil {
		return err
	}

	// Start telemetry polling
	p.stopChan = make(chan struct{})
	go p.pollTelemetry(ctx)

	return nil
}

func (p *SatellitePowerController) initTCP(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", p.config.Address, p.config.Port)
	
	dialer := net.Dialer{Timeout: 10 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to power controller: %w", err)
	}

	p.conn = conn
	return nil
}

func (p *SatellitePowerController) initSpaceWire(ctx context.Context) error {
	// SpaceWire is typically accessed via special hardware adapters
	// that expose a character device or network interface
	addr := p.config.Address
	if addr == "" {
		addr = "/dev/spacewire0"
	}

	// For SpaceWire over Ethernet gateways
	if p.config.Port > 0 {
		return p.initTCP(ctx)
	}

	return fmt.Errorf("SpaceWire hardware not available")
}

func (p *SatellitePowerController) initCAN(ctx context.Context) error {
	// CAN bus access via SocketCAN on Linux or USB-CAN adapters
	// For satellite ground testing
	addr := fmt.Sprintf("can%d", p.config.BusID)
	
	conn, err := net.Dial("unixgram", "/tmp/can_socket_"+addr)
	if err != nil {
		// Fallback to TCP gateway
		return p.initTCP(ctx)
	}

	p.conn = conn
	return nil
}

func (p *SatellitePowerController) pollTelemetry(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := p.readTelemetry(); err != nil {
				// Log error but continue polling
				continue
			}
		case <-ctx.Done():
			return
		case <-p.stopChan:
			return
		}
	}
}

func (p *SatellitePowerController) readTelemetry() error {
	if p.conn == nil {
		return fmt.Errorf("not connected")
	}

	// Send telemetry request
	cmd := []byte{0xAA, 0x01, 0x00, 0x00} // Read all telemetry command
	if _, err := p.conn.Write(cmd); err != nil {
		return err
	}

	// Read response (48 bytes telemetry frame)
	buf := make([]byte, 48)
	if _, err := io.ReadFull(p.conn, buf); err != nil {
		return err
	}

	// Parse telemetry frame
	p.mu.Lock()
	p.lastTelemetry = PowerTelemetry{
		BatteryVoltage:     float64(binary.BigEndian.Uint16(buf[0:2])) / 100.0,
		BatteryCurrent:     float64(int16(binary.BigEndian.Uint16(buf[2:4]))) / 100.0,
		BatteryTemperature: float64(int16(binary.BigEndian.Uint16(buf[4:6]))) / 10.0,
		SolarPanelVoltage:  float64(binary.BigEndian.Uint16(buf[6:8])) / 100.0,
		SolarPanelCurrent:  float64(binary.BigEndian.Uint16(buf[8:10])) / 100.0,
		TotalPowerDraw:     float64(binary.BigEndian.Uint16(buf[10:12])) / 10.0,
		InEclipse:          buf[12] == 1,
		Timestamp:          time.Now(),
	}
	p.lastTelemetry.SolarPanelPower = p.lastTelemetry.SolarPanelVoltage * p.lastTelemetry.SolarPanelCurrent
	p.lastTelemetry.BatteryPercent = p.calculateBatteryPercent()
	p.lastTelemetry.ChargeState = p.determineChargeState()
	p.mu.Unlock()

	return nil
}

func (p *SatellitePowerController) calculateBatteryPercent() float64 {
	// Calculate SoC from voltage using Li-ion discharge curve
	cellVoltage := p.lastTelemetry.BatteryVoltage / float64(p.config.NumBatteryCells)
	
	// Simplified Li-ion voltage to SoC mapping
	minV := p.config.CellVoltageMin
	maxV := p.config.CellVoltageMax
	
	if cellVoltage <= minV {
		return 0.0
	}
	if cellVoltage >= maxV {
		return 100.0
	}

	// Non-linear mapping (approximation of Li-ion curve)
	normalized := (cellVoltage - minV) / (maxV - minV)
	
	// Apply curve correction
	if normalized < 0.2 {
		return normalized * 50.0
	} else if normalized > 0.9 {
		return 90.0 + (normalized-0.9)*100.0
	}
	return 10.0 + normalized*80.0
}

func (p *SatellitePowerController) determineChargeState() string {
	if p.lastTelemetry.BatteryCurrent > 0.1 {
		return "charging"
	} else if p.lastTelemetry.BatteryCurrent < -0.1 {
		return "discharging"
	}
	return "idle"
}

func (p *SatellitePowerController) GetBatteryPercent() (float64, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.lastTelemetry.Timestamp.IsZero() {
		return 0, fmt.Errorf("no telemetry available")
	}

	// Check for stale telemetry (older than 30 seconds)
	if time.Since(p.lastTelemetry.Timestamp) > 30*time.Second {
		return p.lastTelemetry.BatteryPercent, fmt.Errorf("stale telemetry")
	}

	return p.lastTelemetry.BatteryPercent, nil
}

func (p *SatellitePowerController) GetBatteryVoltage() (float64, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.lastTelemetry.Timestamp.IsZero() {
		return 0, fmt.Errorf("no telemetry available")
	}

	return p.lastTelemetry.BatteryVoltage, nil
}

func (p *SatellitePowerController) GetSolarPanelPower() (float64, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.lastTelemetry.SolarPanelPower, nil
}

func (p *SatellitePowerController) IsInEclipse() (bool, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// If we have telemetry, use hardware-reported eclipse state
	if !p.lastTelemetry.Timestamp.IsZero() {
		return p.lastTelemetry.InEclipse, nil
	}

	// Fallback: Calculate eclipse based on solar panel power
	// If solar power is near zero, likely in eclipse
	if p.lastTelemetry.SolarPanelPower < 1.0 {
		return true, nil
	}

	return false, nil
}

func (p *SatellitePowerController) SetPowerMode(mode PowerMode) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conn == nil {
		return fmt.Errorf("not connected")
	}

	// Map mode to command byte
	var modeByte byte
	switch mode {
	case PowerModeNormal:
		modeByte = 0x00
	case PowerModeLow:
		modeByte = 0x01
	case PowerModeCritical:
		modeByte = 0x02
	default:
		return fmt.Errorf("unknown power mode: %s", mode)
	}

	// Send power mode command
	cmd := []byte{0xAA, 0x02, modeByte, 0x00}
	if _, err := p.conn.Write(cmd); err != nil {
		return err
	}

	// Read acknowledgment
	ack := make([]byte, 4)
	if _, err := io.ReadFull(p.conn, ack); err != nil {
		return err
	}

	if ack[2] != 0x00 {
		return fmt.Errorf("power mode change failed: error code %d", ack[2])
	}

	p.mode = mode
	return nil
}

// GetTelemetry returns the full telemetry structure
func (p *SatellitePowerController) GetTelemetry() PowerTelemetry {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.lastTelemetry
}

// Shutdown closes the connection to the power controller
func (p *SatellitePowerController) Shutdown() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.stopChan != nil {
		close(p.stopChan)
	}

	if p.conn != nil {
		return p.conn.Close()
	}

	return nil
}

// ============================================================================
// Orbital Mechanics Helpers
// ============================================================================

// PredictEclipse predicts eclipse windows for the satellite
func (p *SatellitePowerController) PredictEclipse(startTime time.Time, duration time.Duration) []EclipseWindow {
	var windows []EclipseWindow

	// Simplified eclipse prediction for circular orbit
	// Real implementation would use SGP4/SDP4 propagators
	orbitalPeriod := p.config.OrbitalPeriod
	eclipseFraction := 0.35 // Typical LEO eclipse fraction

	eclipseDuration := time.Duration(float64(orbitalPeriod) * eclipseFraction)

	current := startTime
	for current.Before(startTime.Add(duration)) {
		// Find next eclipse start
		orbitsElapsed := float64(current.Sub(startTime)) / float64(orbitalPeriod)
		nextOrbit := math.Ceil(orbitsElapsed)
		
		eclipseStart := startTime.Add(time.Duration(float64(orbitalPeriod) * (nextOrbit - eclipseFraction/2)))
		eclipseEnd := eclipseStart.Add(eclipseDuration)

		if eclipseStart.After(startTime.Add(duration)) {
			break
		}

		windows = append(windows, EclipseWindow{
			Start:    eclipseStart,
			End:      eclipseEnd,
			Duration: eclipseDuration,
		})

		current = eclipseEnd
	}

	return windows
}

// EclipseWindow represents a predicted eclipse period
type EclipseWindow struct {
	Start    time.Time
	End      time.Time
	Duration time.Duration
}
