package control

import (
	"context"
	"time"
)

// Joint represents a single robot joint
type Joint struct {
	ID          string
	Position    float64 // radians
	Velocity    float64 // rad/s
	Torque      float64 // Nm
	Temperature float64 // Celsius
	Timestamp   time.Time
}

// Pose represents robot position and orientation
type Pose struct {
	Position    Vector3
	Orientation Quaternion
	Timestamp   time.Time
}

// Vector3 represents a 3D vector
type Vector3 struct {
	X float64
	Y float64
	Z float64
}

// Quaternion represents orientation
type Quaternion struct {
	W float64
	X float64
	Y float64
	Z float64
}

// MotionController manages robot movement
type MotionController interface {
	Initialize(ctx context.Context) error
	GetCurrentPose() (Pose, error)
	MoveTo(ctx context.Context, target Pose) error
	Stop() error
	GetJointStates() ([]Joint, error)
	SetJointPositions(positions map[string]float64) error
	IsMoving() bool
}

// HunoidController extends MotionController with battery telemetry.
type HunoidController interface {
	MotionController
	GetBatteryPercent() float64
}

// PerceptionSystem handles sensors
type PerceptionSystem interface {
	GetCameraImage(cameraID string) ([]byte, error)
	GetLidarScan() ([]Point3D, error)
	GetDepthMap() ([][]float64, error)
	DetectObstacles(radius float64) ([]Obstacle, error)
}

// Point3D represents a 3D point
type Point3D struct {
	X         float64
	Y         float64
	Z         float64
	Intensity float64
}

// Obstacle represents a detected obstacle
type Obstacle struct {
	Position Vector3
	Size     Vector3
	Type     string
}

// ManipulatorController handles gripper/arms
type ManipulatorController interface {
	OpenGripper() error
	CloseGripper() error
	GetGripperState() (float64, error) // 0.0 = closed, 1.0 = open
	ReachTo(ctx context.Context, position Vector3) error
}

// NavigationController handles autonomous movement
type NavigationController interface {
	SetGoal(ctx context.Context, goal Pose) error
	GetCurrentGoal() (Pose, error)
	CancelGoal() error
	IsGoalReached() bool
	GetNavigationStatus() NavigationStatus
}

// NavigationStatus represents navigation state
type NavigationStatus struct {
	Active         bool
	DistanceToGoal float64
	EstimatedTime  time.Duration
	CurrentVelocity Vector3
}
