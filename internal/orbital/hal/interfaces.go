package hal

import (
	"context"
	"time"
)

// CameraController defines the contract for imaging sensors
type CameraController interface {
	Initialize(ctx context.Context) error
	CaptureFrame(ctx context.Context) ([]byte, error)
	StartStream(ctx context.Context, frameChan chan<- []byte) error
	StopStream() error
	SetExposure(microseconds int) error
	SetGain(gain float64) error
	GetDiagnostics() (CameraDiagnostics, error)
	Shutdown() error
}

// CameraDiagnostics contains camera health data
type CameraDiagnostics struct {
	Temperature float64
	Voltage     float64
	FrameCount  uint64
	ErrorCount  uint64
}

// IMUController handles inertial measurement unit
type IMUController interface {
	Initialize(ctx context.Context) error
	ReadAcceleration() (x, y, z float64, err error)
	ReadGyroscope() (x, y, z float64, err error)
	ReadMagnetometer() (x, y, z float64, err error)
	Calibrate() error
}

// PowerController manages satellite power systems
type PowerController interface {
	GetBatteryPercent() (float64, error)
	GetBatteryVoltage() (float64, error)
	GetSolarPanelPower() (float64, error)
	IsInEclipse() (bool, error)
	SetPowerMode(mode PowerMode) error
}

// PowerMode defines operational power states
type PowerMode string

const (
	PowerModeNormal   PowerMode = "normal"
	PowerModeLow      PowerMode = "low"
	PowerModeCritical PowerMode = "critical"
)

// GPSController provides position and timing
type GPSController interface {
	GetPosition() (lat, lon, alt float64, err error)
	GetTime() (time.Time, error)
	GetVelocity() (vx, vy, vz float64, err error)
}

// RadioController handles communications
type RadioController interface {
	Initialize(ctx context.Context, frequency float64) error
	Transmit(data []byte) error
	Receive(ctx context.Context) ([]byte, error)
	GetSignalStrength() (float64, error)
	SetTransmitPower(watts float64) error
}
