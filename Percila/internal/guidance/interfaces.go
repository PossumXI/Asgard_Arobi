package guidance

import (
	"context"
	"time"
)

// PayloadType defines the type of vehicle being guided
type PayloadType string

const (
	PayloadHunoid       PayloadType = "hunoid"
	PayloadUAV          PayloadType = "uav"
	PayloadRocket       PayloadType = "rocket"
	PayloadMissile      PayloadType = "missile"
	PayloadSpacecraft   PayloadType = "spacecraft"
	PayloadDrone        PayloadType = "drone"
	PayloadGroundRobot  PayloadType = "ground_robot"
	PayloadSubmarine    PayloadType = "submarine"
	PayloadInterstellar PayloadType = "interstellar"
)

// Waypoint represents a navigation point
type Waypoint struct {
	Position    Vector3D
	Velocity    Vector3D
	Timestamp   time.Time
	Constraints WaypointConstraints
}

// Vector3D represents position/velocity in 3D space
type Vector3D struct {
	X float64 // meters or m/s
	Y float64
	Z float64 // altitude
}

// WaypointConstraints defines limits at this waypoint
type WaypointConstraints struct {
	MaxSpeed        float64
	MaxAcceleration float64
	StealthRequired bool
	MinAltitude     float64
	MaxAltitude     float64
	NoFlyZone       bool
}

// Trajectory represents a complete flight path
type Trajectory struct {
	ID              string
	PayloadType     PayloadType
	Waypoints       []Waypoint
	TotalDistance   float64
	EstimatedTime   time.Duration
	FuelRequired    float64
	StealthScore    float64 // 0.0-1.0 (higher = more stealth)
	ThreatExposure  float64 // 0.0-1.0 (lower = safer)
	Confidence      float64 // AI confidence in trajectory
	CreatedAt       time.Time
}

// GuidanceComputer plans and updates trajectories
type GuidanceComputer interface {
	PlanTrajectory(ctx context.Context, req TrajectoryRequest) (*Trajectory, error)
	UpdateTrajectory(ctx context.Context, currentState State, traj *Trajectory) (*Trajectory, error)
	ValidateTrajectory(traj *Trajectory) error
	OptimizeForStealth(traj *Trajectory) (*Trajectory, error)
	OptimizeForSpeed(traj *Trajectory) (*Trajectory, error)
	OptimizeForFuel(traj *Trajectory) (*Trajectory, error)
}

// TrajectoryRequest contains mission parameters
type TrajectoryRequest struct {
	PayloadType     PayloadType
	PayloadID       string
	StartPosition   Vector3D
	TargetPosition  Vector3D
	MaxTime         time.Duration
	Priority        Priority
	Constraints     MissionConstraints
	StealthMode     StealthMode
}

// Priority defines mission urgency
type Priority string

const (
	PriorityLow      Priority = "low"
	PriorityNormal   Priority = "normal"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical"
)

// StealthMode defines stealth requirements
type StealthMode string

const (
	StealthModeNone    StealthMode = "none"
	StealthModeLow     StealthMode = "low"
	StealthModeMedium  StealthMode = "medium"
	StealthModeHigh    StealthMode = "high"
	StealthModeMaximum StealthMode = "maximum"
)

// MissionConstraints defines mission limits
type MissionConstraints struct {
	StealthRequired   bool
	MaxDetectionRisk  float64
	NoFlyZones        []Zone
	MustAvoidThreats  []ThreatLocation
	WeatherConstraints bool
	TimeWindow        *TimeWindow
}

// Zone represents a geographic area
type Zone struct {
	Center Vector3D
	Radius float64
	Type   string // "no_fly", "restricted", "danger"
}

// ThreatLocation represents a known threat
type ThreatLocation struct {
	Position     Vector3D
	ThreatType   string // "radar", "sam", "interceptor", "jamming"
	EffectRadius float64
	Confidence   float64
	LastUpdated  time.Time
}

// TimeWindow represents a time constraint
type TimeWindow struct {
	Start time.Time
	End   time.Time
}

// State represents current payload state
type State struct {
	Position     Vector3D
	Velocity     Vector3D
	Acceleration Vector3D
	Heading      float64 // radians
	Fuel         float64 // percentage
	Battery      float64 // percentage (for electric vehicles)
	Timestamp    time.Time
	PayloadID    string
}

// Magnitude calculates vector magnitude
func Magnitude(v Vector3D) float64 {
	return v.X*v.X + v.Y*v.Y + v.Z*v.Z
}

// Distance calculates distance between two points
func Distance(a, b Vector3D) float64 {
	dx := b.X - a.X
	dy := b.Y - a.Y
	dz := b.Z - a.Z
	return dx*dx + dy*dy + dz*dz
}
