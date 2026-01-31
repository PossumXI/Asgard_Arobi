package hal

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"math"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// SpaceGPSController implements GPSController for space-grade GPS receivers.
// Supports various receivers: NovAtel OEM7, Trimble, ublox, custom satellite receivers.
type SpaceGPSController struct {
	mu       sync.RWMutex
	config   GPSConfig
	conn     io.ReadWriteCloser
	position GPSPosition
	velocity GPSVelocity
	stopChan chan struct{}
}

// GPSConfig holds GPS receiver configuration
type GPSConfig struct {
	// Protocol: "nmea", "rtcm", "binary", "novatel", "ublox"
	Protocol string `json:"protocol"`

	// Connection settings
	Address    string `json:"address"`    // IP address or serial device
	Port       int    `json:"port"`       // TCP port or baud rate
	SerialPort string `json:"serialPort"` // e.g., "/dev/ttyUSB0", "COM3"

	// Receiver settings
	UpdateRate    int    `json:"updateRate"`    // Hz
	Constellation string `json:"constellation"` // GPS, GLONASS, Galileo, BeiDou, combined

	// Space-specific
	SpaceMode   bool    `json:"spaceMode"`   // Enable space vehicle mode (>12km altitude)
	MaxAltitude float64 `json:"maxAltitude"` // m, for export control compliance
	MaxVelocity float64 `json:"maxVelocity"` // m/s
	SatelliteID string  `json:"satelliteId"`
}

// GPSPosition represents a GPS fix
type GPSPosition struct {
	Latitude      float64   `json:"latitude"`  // degrees
	Longitude     float64   `json:"longitude"` // degrees
	Altitude      float64   `json:"altitude"`  // meters
	HDOP          float64   `json:"hdop"`      // Horizontal dilution of precision
	VDOP          float64   `json:"vdop"`      // Vertical dilution of precision
	NumSatellites int       `json:"numSatellites"`
	FixType       string    `json:"fixType"` // none, 2D, 3D, RTK
	Timestamp     time.Time `json:"timestamp"`
}

// GPSVelocity represents velocity from GPS
type GPSVelocity struct {
	VX        float64   `json:"vx"`      // m/s ECEF X
	VY        float64   `json:"vy"`      // m/s ECEF Y
	VZ        float64   `json:"vz"`      // m/s ECEF Z
	Speed     float64   `json:"speed"`   // m/s ground speed
	Heading   float64   `json:"heading"` // degrees
	Timestamp time.Time `json:"timestamp"`
}

// NewSpaceGPSController creates a new GPS controller
func NewSpaceGPSController(config GPSConfig) *SpaceGPSController {
	if config.UpdateRate == 0 {
		config.UpdateRate = 1
	}
	if config.Protocol == "" {
		config.Protocol = "nmea"
	}

	return &SpaceGPSController{
		config: config,
	}
}

// Initialize establishes connection to the GPS receiver
func (g *SpaceGPSController) Initialize(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	var err error
	if g.config.SerialPort != "" {
		err = g.initSerial()
	} else {
		err = g.initTCP(ctx)
	}

	if err != nil {
		return err
	}

	// Configure receiver for space mode if enabled
	if g.config.SpaceMode {
		if err := g.enableSpaceMode(); err != nil {
			return err
		}
	}

	// Start position updates
	g.stopChan = make(chan struct{})
	go g.readLoop(ctx)

	return nil
}

func (g *SpaceGPSController) initSerial() error {
	// Serial port configuration
	// Real implementation would use serial library
	return fmt.Errorf("serial port support requires platform-specific implementation")
}

func (g *SpaceGPSController) initTCP(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", g.config.Address, g.config.Port)

	dialer := net.Dialer{Timeout: 10 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to GPS receiver: %w", err)
	}

	g.conn = conn
	return nil
}

func (g *SpaceGPSController) enableSpaceMode() error {
	// Different receivers have different commands for space mode
	var cmd string
	switch g.config.Protocol {
	case "novatel":
		// NovAtel OEM7 space mode command
		cmd = "LOG COM1 BESTXYZB ONTIME 1\r\n"
		cmd += "DYNAMICS AIR\r\n"
	case "ublox":
		// ublox space mode (platform model: airborne 4g)
		cmd = "$PUBX,41,1,0007,0003,115200,0*18\r\n"
	default:
		// Generic NMEA configuration
		return nil
	}

	if g.conn != nil && cmd != "" {
		_, err := g.conn.Write([]byte(cmd))
		return err
	}

	return nil
}

func (g *SpaceGPSController) readLoop(ctx context.Context) {
	if g.conn == nil {
		return
	}

	reader := bufio.NewReader(g.conn)

	for {
		select {
		case <-ctx.Done():
			return
		case <-g.stopChan:
			return
		default:
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				continue
			}
			return
		}

		g.parseNMEA(strings.TrimSpace(line))
	}
}

func (g *SpaceGPSController) parseNMEA(sentence string) {
	if len(sentence) < 6 || sentence[0] != '$' {
		return
	}

	// Verify checksum
	if !g.verifyNMEAChecksum(sentence) {
		return
	}

	parts := strings.Split(sentence[1:], ",")
	if len(parts) < 2 {
		return
	}

	msgType := parts[0]

	switch {
	case strings.HasSuffix(msgType, "GGA"):
		g.parseGGA(parts)
	case strings.HasSuffix(msgType, "RMC"):
		g.parseRMC(parts)
	case strings.HasSuffix(msgType, "VTG"):
		g.parseVTG(parts)
	case strings.HasSuffix(msgType, "GSA"):
		g.parseGSA(parts)
	}
}

func (g *SpaceGPSController) verifyNMEAChecksum(sentence string) bool {
	asterisk := strings.LastIndex(sentence, "*")
	if asterisk == -1 || asterisk+3 > len(sentence) {
		return false
	}

	data := sentence[1:asterisk]
	provided := sentence[asterisk+1:]

	var checksum byte
	for i := 0; i < len(data); i++ {
		checksum ^= data[i]
	}

	expected := fmt.Sprintf("%02X", checksum)
	return strings.ToUpper(provided[:2]) == expected
}

func (g *SpaceGPSController) parseGGA(parts []string) {
	// $xxGGA,time,lat,N/S,lon,E/W,quality,numSV,HDOP,alt,M,geoid,M,age,refID*cs
	if len(parts) < 15 {
		return
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	// Parse latitude
	if lat, err := g.parseLatLon(parts[2], parts[3]); err == nil {
		g.position.Latitude = lat
	}

	// Parse longitude
	if lon, err := g.parseLatLon(parts[4], parts[5]); err == nil {
		g.position.Longitude = lon
	}

	// Fix quality
	quality, _ := strconv.Atoi(parts[6])
	switch quality {
	case 0:
		g.position.FixType = "none"
	case 1:
		g.position.FixType = "GPS"
	case 2:
		g.position.FixType = "DGPS"
	case 4:
		g.position.FixType = "RTK"
	case 5:
		g.position.FixType = "Float RTK"
	default:
		g.position.FixType = "3D"
	}

	// Number of satellites
	g.position.NumSatellites, _ = strconv.Atoi(parts[7])

	// HDOP
	g.position.HDOP, _ = strconv.ParseFloat(parts[8], 64)

	// Altitude (MSL)
	if alt, err := strconv.ParseFloat(parts[9], 64); err == nil {
		g.position.Altitude = alt
	}

	g.position.Timestamp = time.Now()
}

func (g *SpaceGPSController) parseRMC(parts []string) {
	// $xxRMC,time,status,lat,N/S,lon,E/W,speed,course,date,mag,magdir,mode*cs
	if len(parts) < 12 {
		return
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	// Speed (knots to m/s)
	if speed, err := strconv.ParseFloat(parts[7], 64); err == nil {
		g.velocity.Speed = speed * 0.514444 // knots to m/s
	}

	// Course/heading
	if heading, err := strconv.ParseFloat(parts[8], 64); err == nil {
		g.velocity.Heading = heading
	}

	g.velocity.Timestamp = time.Now()
}

func (g *SpaceGPSController) parseVTG(parts []string) {
	// $xxVTG,course,T,course,M,speed,N,speed,K,mode*cs
	if len(parts) < 9 {
		return
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	// Course over ground (true)
	if heading, err := strconv.ParseFloat(parts[1], 64); err == nil {
		g.velocity.Heading = heading
	}

	// Speed in km/h to m/s
	if speed, err := strconv.ParseFloat(parts[7], 64); err == nil {
		g.velocity.Speed = speed / 3.6
	}
}

func (g *SpaceGPSController) parseGSA(parts []string) {
	// $xxGSA,mode,fix,prn,...,pdop,hdop,vdop*cs
	if len(parts) < 18 {
		return
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	// HDOP
	if hdop, err := strconv.ParseFloat(parts[16], 64); err == nil {
		g.position.HDOP = hdop
	}

	// VDOP
	if vdop, err := strconv.ParseFloat(parts[17], 64); err == nil {
		// Remove checksum if present
		vdopStr := strings.Split(parts[17], "*")[0]
		if v, err := strconv.ParseFloat(vdopStr, 64); err == nil {
			g.position.VDOP = v
		} else {
			g.position.VDOP = vdop
		}
	}
}

func (g *SpaceGPSController) parseLatLon(value, direction string) (float64, error) {
	if value == "" {
		return 0, fmt.Errorf("empty value")
	}

	// NMEA format: DDDMM.MMMMM or DDMM.MMMMM
	var degrees, minutes float64

	if len(value) > 4 && strings.Contains(value, ".") {
		dotPos := strings.Index(value, ".")
		degLen := dotPos - 2 // Minutes always 2 digits before decimal

		var err error
		degrees, err = strconv.ParseFloat(value[:degLen], 64)
		if err != nil {
			return 0, err
		}

		minutes, err = strconv.ParseFloat(value[degLen:], 64)
		if err != nil {
			return 0, err
		}
	} else {
		return 0, fmt.Errorf("invalid format")
	}

	result := degrees + minutes/60.0

	if direction == "S" || direction == "W" {
		result = -result
	}

	return result, nil
}

// GetPosition returns the current GPS position
func (g *SpaceGPSController) GetPosition() (lat, lon, alt float64, err error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.position.Timestamp.IsZero() {
		return 0, 0, 0, fmt.Errorf("no GPS fix available")
	}

	// Check for stale data (older than 10 seconds)
	if time.Since(g.position.Timestamp) > 10*time.Second {
		return g.position.Latitude, g.position.Longitude, g.position.Altitude,
			fmt.Errorf("stale GPS data")
	}

	return g.position.Latitude, g.position.Longitude, g.position.Altitude, nil
}

// GetTime returns the current GPS time
func (g *SpaceGPSController) GetTime() (time.Time, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.position.Timestamp.IsZero() {
		return time.Time{}, fmt.Errorf("no GPS time available")
	}

	return g.position.Timestamp, nil
}

// GetVelocity returns the current velocity in ECEF coordinates
func (g *SpaceGPSController) GetVelocity() (vx, vy, vz float64, err error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.velocity.Timestamp.IsZero() {
		return 0, 0, 0, fmt.Errorf("no velocity data available")
	}

	// Convert speed and heading to ECEF velocity
	// For satellites, this is typically provided directly by the receiver
	// This conversion is simplified for ground-based or low-altitude use

	// Convert heading to radians
	headingRad := g.velocity.Heading * math.Pi / 180.0
	latRad := g.position.Latitude * math.Pi / 180.0
	lonRad := g.position.Longitude * math.Pi / 180.0

	// Calculate ECEF velocity components
	speed := g.velocity.Speed

	// East and North velocity
	vE := speed * math.Sin(headingRad)
	vN := speed * math.Cos(headingRad)

	// Convert to ECEF
	vx = -vE*math.Sin(lonRad) - vN*math.Sin(latRad)*math.Cos(lonRad)
	vy = vE*math.Cos(lonRad) - vN*math.Sin(latRad)*math.Sin(lonRad)
	vz = vN * math.Cos(latRad)

	return vx, vy, vz, nil
}

// GetFullPosition returns the complete position structure
func (g *SpaceGPSController) GetFullPosition() GPSPosition {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.position
}

// GetFullVelocity returns the complete velocity structure
func (g *SpaceGPSController) GetFullVelocity() GPSVelocity {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.velocity
}

// Shutdown closes the GPS receiver connection
func (g *SpaceGPSController) Shutdown() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.stopChan != nil {
		close(g.stopChan)
	}

	if g.conn != nil {
		return g.conn.Close()
	}

	return nil
}

// ============================================================================
// Orbital Position Calculator (when GPS unavailable)
// ============================================================================

// TLEPositionCalculator calculates position from TLE when GPS is unavailable
type TLEPositionCalculator struct {
	mu  sync.RWMutex
	tle TLE
	gps *SpaceGPSController
}

// TLE represents two-line element set for orbit propagation
type TLE struct {
	Name         string
	Line1        string
	Line2        string
	Epoch        time.Time
	Inclination  float64
	RAAN         float64
	Eccentricity float64
	ArgPerigee   float64
	MeanAnomaly  float64
	MeanMotion   float64
}

// NewTLEPositionCalculator creates a hybrid position provider
func NewTLEPositionCalculator(gps *SpaceGPSController, tle TLE) *TLEPositionCalculator {
	return &TLEPositionCalculator{
		gps: gps,
		tle: tle,
	}
}

// GetPosition returns position from GPS if available, otherwise propagates TLE
func (o *TLEPositionCalculator) GetPosition() (lat, lon, alt float64, err error) {
	// Try GPS first
	if o.gps != nil {
		lat, lon, alt, err = o.gps.GetPosition()
		if err == nil {
			return lat, lon, alt, nil
		}
	}

	// Fallback to TLE propagation
	return o.propagateTLE(time.Now())
}

func (o *TLEPositionCalculator) propagateTLE(t time.Time) (lat, lon, alt float64, err error) {
	o.mu.RLock()
	tle := o.tle
	o.mu.RUnlock()

	if tle.Line1 == "" {
		return 0, 0, 0, fmt.Errorf("no TLE data available")
	}

	// Simplified SGP4 propagation
	// Real implementation would use full SGP4/SDP4 algorithm

	// Time since epoch
	dt := t.Sub(tle.Epoch).Minutes()

	// Mean motion in rad/min
	n := tle.MeanMotion * 2 * math.Pi / 1440.0

	// Mean anomaly at time t
	M := tle.MeanAnomaly + n*dt
	M = math.Mod(M, 2*math.Pi)

	// Simplified Kepler solver for eccentric anomaly
	E := M
	for i := 0; i < 10; i++ {
		E = M + tle.Eccentricity*math.Sin(E)
	}

	// True anomaly
	v := 2 * math.Atan2(
		math.Sqrt(1+tle.Eccentricity)*math.Sin(E/2),
		math.Sqrt(1-tle.Eccentricity)*math.Cos(E/2),
	)

	// Argument of latitude
	u := v + tle.ArgPerigee

	// Semi-major axis (from mean motion)
	mu := 398600.4418 // km^3/s^2
	a := math.Pow(mu/math.Pow(n*60, 2), 1.0/3.0)

	// Orbital radius
	r := a * (1 - tle.Eccentricity*math.Cos(E))

	// Position in orbital plane
	xOrbital := r * math.Cos(u)
	yOrbital := r * math.Sin(u)

	// Rotate to Earth-centered coordinates
	raanRad := tle.RAAN * math.Pi / 180.0
	inclRad := tle.Inclination * math.Pi / 180.0

	// Earth rotation since epoch
	gmst := o.calculateGMST(t)

	// ECEF position
	x := xOrbital*(math.Cos(raanRad)*math.Cos(gmst)+math.Sin(raanRad)*math.Sin(gmst)*math.Cos(inclRad)) -
		yOrbital*(math.Sin(raanRad)*math.Cos(gmst)-math.Cos(raanRad)*math.Sin(gmst)*math.Cos(inclRad))
	y := xOrbital*(-math.Cos(raanRad)*math.Sin(gmst)+math.Sin(raanRad)*math.Cos(gmst)*math.Cos(inclRad)) -
		yOrbital*(math.Sin(raanRad)*math.Sin(gmst)+math.Cos(raanRad)*math.Cos(gmst)*math.Cos(inclRad))
	z := xOrbital*math.Sin(raanRad)*math.Sin(inclRad) + yOrbital*math.Cos(raanRad)*math.Sin(inclRad)

	// Convert ECEF to geodetic
	lat, lon, alt = o.ecefToGeodetic(x, y, z)

	return lat, lon, alt * 1000, nil // alt in meters
}

func (o *TLEPositionCalculator) calculateGMST(t time.Time) float64 {
	// Calculate Greenwich Mean Sidereal Time
	jd := float64(t.Unix())/86400.0 + 2440587.5
	T := (jd - 2451545.0) / 36525.0
	gmst := 280.46061837 + 360.98564736629*(jd-2451545.0) + 0.000387933*T*T
	return math.Mod(gmst, 360.0) * math.Pi / 180.0
}

func (o *TLEPositionCalculator) ecefToGeodetic(x, y, z float64) (lat, lon, alt float64) {
	// WGS84 parameters
	a := 6378.137 // km
	f := 1.0 / 298.257223563
	_ = a * (1 - f) // b (semi-minor axis) - kept for reference
	e2 := 2*f - f*f

	// Longitude
	lon = math.Atan2(y, x) * 180 / math.Pi

	// Iterative latitude calculation
	p := math.Sqrt(x*x + y*y)
	lat = math.Atan2(z, p*(1-e2))

	for i := 0; i < 10; i++ {
		sinLat := math.Sin(lat)
		N := a / math.Sqrt(1-e2*sinLat*sinLat)
		lat = math.Atan2(z+e2*N*sinLat, p)
	}

	// Altitude
	sinLat := math.Sin(lat)
	N := a / math.Sqrt(1-e2*sinLat*sinLat)
	alt = p/math.Cos(lat) - N

	lat = lat * 180 / math.Pi

	return lat, lon, alt
}

// UpdateTLE updates the TLE data
func (o *TLEPositionCalculator) UpdateTLE(tle TLE) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.tle = tle
}
