package navigation

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
)

// NavigationMode defines the navigation strategy
type NavigationMode string

const (
	ModeDirectPath     NavigationMode = "direct"         // Straight line, fastest
	ModeTerrainFollow  NavigationMode = "terrain"        // Follow terrain contours
	ModeStealthPath    NavigationMode = "stealth"        // Minimize detection
	ModeEvasive        NavigationMode = "evasive"        // Active threat evasion
	ModeEnergySaving   NavigationMode = "energy"         // Minimize fuel/power
	ModeBallisticArc   NavigationMode = "ballistic"      // Ballistic trajectory
	ModeOrbitalInsert  NavigationMode = "orbital"        // Orbital insertion
	ModeInterplanetary NavigationMode = "interplanetary" // Deep space navigation
)

// Vector3D represents position/velocity in 3D space
type Vector3D struct {
	X float64 `json:"x"` // meters or m/s (East)
	Y float64 `json:"y"` // meters or m/s (North)
	Z float64 `json:"z"` // meters or m/s (Altitude/Up)
}

// GeoCoord represents a geographic coordinate
type GeoCoord struct {
	Latitude  float64 `json:"latitude"`  // degrees
	Longitude float64 `json:"longitude"` // degrees
	Altitude  float64 `json:"altitude"`  // meters MSL
}

// Waypoint represents a navigation point with timing
type Waypoint struct {
	ID          string        `json:"id"`
	Position    Vector3D      `json:"position"`
	GeoPosition *GeoCoord     `json:"geoPosition,omitempty"`
	Velocity    Vector3D      `json:"velocity"`
	Heading     float64       `json:"heading"` // radians
	Timestamp   time.Time     `json:"timestamp"`
	Tolerance   float64       `json:"tolerance"` // meters
	Hold        bool          `json:"hold"`      // hold at waypoint
	HoldTime    time.Duration `json:"holdTime"`
}

// NavigationState represents current navigation status
type NavigationState struct {
	CurrentPosition  Vector3D       `json:"currentPosition"`
	CurrentVelocity  Vector3D       `json:"currentVelocity"`
	CurrentHeading   float64        `json:"currentHeading"`
	TargetWaypoint   *Waypoint      `json:"targetWaypoint"`
	WaypointIndex    int            `json:"waypointIndex"`
	TotalWaypoints   int            `json:"totalWaypoints"`
	DistanceToTarget float64        `json:"distanceToTarget"`
	ETA              time.Duration  `json:"eta"`
	Mode             NavigationMode `json:"mode"`
	Status           string         `json:"status"`        // navigating, holding, arrived, error
	FuelRemaining    float64        `json:"fuelRemaining"` // percentage
	BatteryLevel     float64        `json:"batteryLevel"`  // percentage
	LastUpdate       time.Time      `json:"lastUpdate"`
}

// TerrainData represents terrain information for navigation
type TerrainData struct {
	Elevation [][]float64 `json:"elevation"` // 2D grid of elevations
	CellSize  float64     `json:"cellSize"`  // meters per cell
	OriginX   float64     `json:"originX"`
	OriginY   float64     `json:"originY"`
	Width     int         `json:"width"`
	Height    int         `json:"height"`
}

// ThreatZone represents an area to avoid
type ThreatZone struct {
	ID          string    `json:"id"`
	Center      Vector3D  `json:"center"`
	Radius      float64   `json:"radius"`      // meters
	ThreatLevel float64   `json:"threatLevel"` // 0.0-1.0
	ThreatType  string    `json:"threatType"`  // radar, sam, air_defense
	Active      bool      `json:"active"`
	ValidUntil  time.Time `json:"validUntil"`
}

// NavigationConfig holds navigation parameters
type NavigationConfig struct {
	Mode               NavigationMode `json:"mode"`
	MaxSpeed           float64        `json:"maxSpeed"`          // m/s
	MaxAcceleration    float64        `json:"maxAcceleration"`   // m/s²
	MaxTurnRate        float64        `json:"maxTurnRate"`       // rad/s
	MinAltitude        float64        `json:"minAltitude"`       // meters AGL
	MaxAltitude        float64        `json:"maxAltitude"`       // meters MSL
	TerrainClearance   float64        `json:"terrainClearance"`  // meters above terrain
	WaypointTolerance  float64        `json:"waypointTolerance"` // meters
	EnableTerrainAvoid bool           `json:"enableTerrainAvoid"`
	EnableThreatAvoid  bool           `json:"enableThreatAvoid"`
	StealthPriority    float64        `json:"stealthPriority"` // 0.0-1.0
}

// Navigator provides advanced navigation capabilities
type Navigator struct {
	mu sync.RWMutex

	id          string
	config      NavigationConfig
	waypoints   []Waypoint
	currentIdx  int
	state       NavigationState
	terrain     *TerrainData
	threatZones []ThreatZone

	// Callbacks
	onWaypointReached func(wp Waypoint)
	onNavigationError func(err error)
	onStateChange     func(state NavigationState)
}

// NewNavigator creates a new Navigator instance
func NewNavigator(id string, config NavigationConfig) *Navigator {
	return &Navigator{
		id:          id,
		config:      config,
		waypoints:   make([]Waypoint, 0),
		threatZones: make([]ThreatZone, 0),
		state: NavigationState{
			Status:     "idle",
			LastUpdate: time.Now(),
		},
	}
}

// SetWaypoints sets the navigation route
func (n *Navigator) SetWaypoints(waypoints []Waypoint) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if len(waypoints) == 0 {
		return ErrNoWaypoints
	}

	n.waypoints = waypoints
	n.currentIdx = 0
	n.state.TotalWaypoints = len(waypoints)
	n.state.WaypointIndex = 0
	n.state.TargetWaypoint = &waypoints[0]
	n.state.Status = "ready"
	n.state.LastUpdate = time.Now()

	return nil
}

// SetTerrain provides terrain data for navigation
func (n *Navigator) SetTerrain(terrain *TerrainData) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.terrain = terrain
}

// AddThreatZone adds a threat zone to avoid
func (n *Navigator) AddThreatZone(zone ThreatZone) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.threatZones = append(n.threatZones, zone)
}

// ClearThreatZones removes all threat zones
func (n *Navigator) ClearThreatZones() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.threatZones = make([]ThreatZone, 0)
}

// Start begins navigation
func (n *Navigator) Start(ctx context.Context) error {
	n.mu.Lock()
	if n.state.Status != "ready" && n.state.Status != "paused" {
		n.mu.Unlock()
		return ErrNotReady
	}
	n.state.Status = "navigating"
	n.state.Mode = n.config.Mode
	n.mu.Unlock()

	go n.navigationLoop(ctx)
	return nil
}

// Pause pauses navigation
func (n *Navigator) Pause() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.state.Status == "navigating" {
		n.state.Status = "paused"
	}
}

// Resume resumes paused navigation
func (n *Navigator) Resume() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.state.Status == "paused" {
		n.state.Status = "navigating"
	}
}

// Stop stops navigation
func (n *Navigator) Stop() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.state.Status = "stopped"
}

// GetState returns current navigation state
func (n *Navigator) GetState() NavigationState {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.state
}

// UpdatePosition updates the current position from sensors
func (n *Navigator) UpdatePosition(pos Vector3D, vel Vector3D, heading float64) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.state.CurrentPosition = pos
	n.state.CurrentVelocity = vel
	n.state.CurrentHeading = heading
	n.state.LastUpdate = time.Now()

	// Calculate distance to target waypoint
	if n.state.TargetWaypoint != nil {
		n.state.DistanceToTarget = Distance(pos, n.state.TargetWaypoint.Position)

		// Calculate ETA
		speed := Magnitude(vel)
		if speed > 0 {
			n.state.ETA = time.Duration(n.state.DistanceToTarget/speed) * time.Second
		}
	}
}

// navigationLoop is the main navigation processing loop
func (n *Navigator) navigationLoop(ctx context.Context) {
	ticker := time.NewTicker(50 * time.Millisecond) // 20Hz update rate
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			n.mu.Lock()
			n.state.Status = "stopped"
			n.mu.Unlock()
			return

		case <-ticker.C:
			n.processNavigation()
		}
	}
}

// processNavigation processes one navigation step
func (n *Navigator) processNavigation() {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.state.Status != "navigating" {
		return
	}

	if n.currentIdx >= len(n.waypoints) {
		n.state.Status = "arrived"
		return
	}

	target := n.waypoints[n.currentIdx]
	distance := Distance(n.state.CurrentPosition, target.Position)

	// Check if waypoint reached
	if distance <= n.config.WaypointTolerance {
		if n.onWaypointReached != nil {
			go n.onWaypointReached(target)
		}

		// Handle hold at waypoint
		if target.Hold {
			n.state.Status = "holding"
			go n.holdAtWaypoint(target)
			return
		}

		// Move to next waypoint
		n.currentIdx++
		n.state.WaypointIndex = n.currentIdx

		if n.currentIdx < len(n.waypoints) {
			n.state.TargetWaypoint = &n.waypoints[n.currentIdx]
		} else {
			n.state.Status = "arrived"
			n.state.TargetWaypoint = nil
		}
	}

	n.state.DistanceToTarget = distance
	n.state.LastUpdate = time.Now()

	if n.onStateChange != nil {
		go n.onStateChange(n.state)
	}
}

// holdAtWaypoint implements hold behavior
func (n *Navigator) holdAtWaypoint(wp Waypoint) {
	time.Sleep(wp.HoldTime)

	n.mu.Lock()
	defer n.mu.Unlock()

	if n.state.Status != "holding" {
		return
	}

	n.currentIdx++
	n.state.WaypointIndex = n.currentIdx

	if n.currentIdx < len(n.waypoints) {
		n.state.TargetWaypoint = &n.waypoints[n.currentIdx]
		n.state.Status = "navigating"
	} else {
		n.state.Status = "arrived"
		n.state.TargetWaypoint = nil
	}
}

// CalculateSteeringCommand calculates the steering needed to reach target
func (n *Navigator) CalculateSteeringCommand() SteeringCommand {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.state.TargetWaypoint == nil {
		return SteeringCommand{}
	}

	// Calculate direction to target
	direction := Subtract(n.state.TargetWaypoint.Position, n.state.CurrentPosition)
	targetHeading := math.Atan2(direction.Y, direction.X)

	// Calculate heading error
	headingError := NormalizeAngle(targetHeading - n.state.CurrentHeading)

	// Calculate turn rate (proportional control)
	turnRate := Clamp(headingError*2.0, -n.config.MaxTurnRate, n.config.MaxTurnRate)

	// Calculate speed based on distance and heading error
	distance := Magnitude(direction)
	speedFactor := 1.0 - math.Abs(headingError)/math.Pi // Slow down for large turns

	// Apply stealth mode speed reduction
	if n.config.Mode == ModeStealthPath {
		speedFactor *= (1.0 - n.config.StealthPriority*0.5)
	}

	targetSpeed := math.Min(n.config.MaxSpeed*speedFactor, distance/2.0)

	// Calculate altitude command
	altitudeCommand := n.state.TargetWaypoint.Position.Z
	if n.terrain != nil && n.config.EnableTerrainAvoid {
		terrainElev := n.getTerrainElevation(n.state.CurrentPosition.X, n.state.CurrentPosition.Y)
		minAlt := terrainElev + n.config.TerrainClearance
		if altitudeCommand < minAlt {
			altitudeCommand = minAlt
		}
	}

	return SteeringCommand{
		TargetHeading:   targetHeading,
		TurnRate:        turnRate,
		TargetSpeed:     targetSpeed,
		TargetAltitude:  altitudeCommand,
		CurrentDistance: distance,
	}
}

// getTerrainElevation returns terrain elevation at given position
func (n *Navigator) getTerrainElevation(x, y float64) float64 {
	if n.terrain == nil {
		return 0
	}

	// Convert to grid coordinates
	gridX := int((x - n.terrain.OriginX) / n.terrain.CellSize)
	gridY := int((y - n.terrain.OriginY) / n.terrain.CellSize)

	// Bounds check
	if gridX < 0 || gridX >= n.terrain.Width || gridY < 0 || gridY >= n.terrain.Height {
		return 0
	}

	return n.terrain.Elevation[gridY][gridX]
}

// CheckThreatExposure calculates threat exposure at current position
func (n *Navigator) CheckThreatExposure() ThreatExposure {
	n.mu.RLock()
	defer n.mu.RUnlock()

	exposure := ThreatExposure{
		TotalExposure: 0,
		ActiveThreats: make([]ThreatInfo, 0),
	}

	now := time.Now()
	for _, zone := range n.threatZones {
		if !zone.Active || now.After(zone.ValidUntil) {
			continue
		}

		distance := Distance(n.state.CurrentPosition, zone.Center)
		if distance < zone.Radius {
			threatLevel := zone.ThreatLevel * (1.0 - distance/zone.Radius)
			exposure.TotalExposure += threatLevel
			exposure.ActiveThreats = append(exposure.ActiveThreats, ThreatInfo{
				ZoneID:      zone.ID,
				ThreatType:  zone.ThreatType,
				Distance:    distance,
				ThreatLevel: threatLevel,
			})
		}
	}

	return exposure
}

// GenerateEvasionPath creates an evasion path around threats
func (n *Navigator) GenerateEvasionPath(target Vector3D) []Waypoint {
	n.mu.RLock()
	defer n.mu.RUnlock()

	evasionPath := make([]Waypoint, 0)
	current := n.state.CurrentPosition

	// Check if direct path intersects any threats
	threats := n.findThreatsOnPath(current, target)

	if len(threats) == 0 {
		// Direct path is clear
		evasionPath = append(evasionPath, Waypoint{
			ID:        uuid.New().String(),
			Position:  target,
			Timestamp: time.Now().Add(time.Duration(Distance(current, target)/n.config.MaxSpeed) * time.Second),
			Tolerance: n.config.WaypointTolerance,
		})
		return evasionPath
	}

	// Generate waypoints around threats
	for _, threat := range threats {
		// Calculate perpendicular offset
		direction := Subtract(target, current)
		perpendicular := Vector3D{
			X: -direction.Y,
			Y: direction.X,
			Z: 0,
		}
		perpendicular = Scale(Normalize(perpendicular), threat.Radius*1.5)

		// Create waypoint offset from threat center
		evasionWP := Waypoint{
			ID:        uuid.New().String(),
			Position:  Add(threat.Center, perpendicular),
			Tolerance: n.config.WaypointTolerance,
		}
		evasionPath = append(evasionPath, evasionWP)
	}

	// Add final target
	evasionPath = append(evasionPath, Waypoint{
		ID:        uuid.New().String(),
		Position:  target,
		Tolerance: n.config.WaypointTolerance,
	})

	return evasionPath
}

// findThreatsOnPath finds threats that intersect a path
func (n *Navigator) findThreatsOnPath(start, end Vector3D) []ThreatZone {
	intersecting := make([]ThreatZone, 0)

	for _, zone := range n.threatZones {
		if !zone.Active {
			continue
		}

		// Check if path comes within threat radius
		dist := PointToLineDistance(zone.Center, start, end)
		if dist < zone.Radius {
			intersecting = append(intersecting, zone)
		}
	}

	return intersecting
}

// SteeringCommand represents navigation commands
type SteeringCommand struct {
	TargetHeading   float64 `json:"targetHeading"`
	TurnRate        float64 `json:"turnRate"`
	TargetSpeed     float64 `json:"targetSpeed"`
	TargetAltitude  float64 `json:"targetAltitude"`
	CurrentDistance float64 `json:"currentDistance"`
}

// ThreatExposure represents current threat exposure
type ThreatExposure struct {
	TotalExposure float64      `json:"totalExposure"`
	ActiveThreats []ThreatInfo `json:"activeThreats"`
}

// ThreatInfo contains info about a specific threat
type ThreatInfo struct {
	ZoneID      string  `json:"zoneId"`
	ThreatType  string  `json:"threatType"`
	Distance    float64 `json:"distance"`
	ThreatLevel float64 `json:"threatLevel"`
}

// OnWaypointReached sets callback for waypoint reached events
func (n *Navigator) OnWaypointReached(callback func(wp Waypoint)) {
	n.onWaypointReached = callback
}

// OnNavigationError sets callback for navigation errors
func (n *Navigator) OnNavigationError(callback func(err error)) {
	n.onNavigationError = callback
}

// OnStateChange sets callback for state changes
func (n *Navigator) OnStateChange(callback func(state NavigationState)) {
	n.onStateChange = callback
}

// Errors
var (
	ErrNoWaypoints = &NavigationError{Message: "no waypoints provided"}
	ErrNotReady    = &NavigationError{Message: "navigator not ready"}
)

// NavigationError represents a navigation error
type NavigationError struct {
	Message string
}

func (e *NavigationError) Error() string {
	return e.Message
}

// Vector math helpers

// Distance calculates Euclidean distance between two points
func Distance(a, b Vector3D) float64 {
	dx := b.X - a.X
	dy := b.Y - a.Y
	dz := b.Z - a.Z
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

// Magnitude calculates vector magnitude
func Magnitude(v Vector3D) float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
}

// Normalize returns a unit vector
func Normalize(v Vector3D) Vector3D {
	mag := Magnitude(v)
	if mag == 0 {
		return Vector3D{}
	}
	return Vector3D{X: v.X / mag, Y: v.Y / mag, Z: v.Z / mag}
}

// Scale scales a vector
func Scale(v Vector3D, s float64) Vector3D {
	return Vector3D{X: v.X * s, Y: v.Y * s, Z: v.Z * s}
}

// Add adds two vectors
func Add(a, b Vector3D) Vector3D {
	return Vector3D{X: a.X + b.X, Y: a.Y + b.Y, Z: a.Z + b.Z}
}

// Subtract subtracts two vectors
func Subtract(a, b Vector3D) Vector3D {
	return Vector3D{X: a.X - b.X, Y: a.Y - b.Y, Z: a.Z - b.Z}
}

// Dot returns the dot product
func Dot(a, b Vector3D) float64 {
	return a.X*b.X + a.Y*b.Y + a.Z*b.Z
}

// Cross returns the cross product
func Cross(a, b Vector3D) Vector3D {
	return Vector3D{
		X: a.Y*b.Z - a.Z*b.Y,
		Y: a.Z*b.X - a.X*b.Z,
		Z: a.X*b.Y - a.Y*b.X,
	}
}

// PointToLineDistance calculates distance from point to line segment
func PointToLineDistance(point, lineStart, lineEnd Vector3D) float64 {
	line := Subtract(lineEnd, lineStart)
	len2 := Dot(line, line)
	if len2 == 0 {
		return Distance(point, lineStart)
	}

	t := math.Max(0, math.Min(1, Dot(Subtract(point, lineStart), line)/len2))
	projection := Add(lineStart, Scale(line, t))
	return Distance(point, projection)
}

// NormalizeAngle normalizes angle to [-π, π]
func NormalizeAngle(angle float64) float64 {
	for angle > math.Pi {
		angle -= 2 * math.Pi
	}
	for angle < -math.Pi {
		angle += 2 * math.Pi
	}
	return angle
}

// Clamp clamps value between min and max
func Clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
