package failsafe

import (
	"context"
	"time"

	"github.com/PossumXI/Asgard/Valkyrie/internal/actuators"
)

// MAVLinkAdapter adapts MAVLinkController to FlightController interface
type MAVLinkAdapter struct {
	controller *actuators.MAVLinkController
}

// NewMAVLinkAdapter creates an adapter for MAVLinkController
func NewMAVLinkAdapter(controller *actuators.MAVLinkController) *MAVLinkAdapter {
	return &MAVLinkAdapter{controller: controller}
}

// SendAttitudeCommand sends attitude command
func (a *MAVLinkAdapter) SendAttitudeCommand(cmd AttitudeCommand) error {
	actuatorCmd := actuators.AttitudeCommand{
		Roll:      cmd.Roll,
		Pitch:     cmd.Pitch,
		Yaw:       cmd.Yaw,
		Throttle:  cmd.Throttle,
		Timestamp: time.Now(),
	}
	
	return a.controller.SendAttitudeCommand(actuatorCmd)
}

// SendPositionCommand sends position command
func (a *MAVLinkAdapter) SendPositionCommand(cmd PositionCommand) error {
	actuatorCmd := actuators.PositionCommand{
		X:         cmd.X,
		Y:         cmd.Y,
		Z:         cmd.Z,
		Timestamp: time.Now(),
	}
	
	return a.controller.SendPositionCommand(actuatorCmd)
}

// SendVelocityCommand sends velocity command
func (a *MAVLinkAdapter) SendVelocityCommand(cmd VelocityCommand) error {
	actuatorCmd := actuators.VelocityCommand{
		Vx:        cmd.Vx,
		Vy:        cmd.Vy,
		Vz:        cmd.Vz,
		Timestamp: time.Now(),
	}
	
	return a.controller.SendVelocityCommand(actuatorCmd)
}

// SetFlightMode sets flight mode
func (a *MAVLinkAdapter) SetFlightMode(mode string) error {
	return a.controller.SetFlightMode(mode)
}

// Arm arms the flight controller
func (a *MAVLinkAdapter) Arm(ctx context.Context) error {
	return a.controller.Arm(ctx)
}

// Disarm disarms the flight controller
func (a *MAVLinkAdapter) Disarm(ctx context.Context) error {
	return a.controller.Disarm(ctx)
}
