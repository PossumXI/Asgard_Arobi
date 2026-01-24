package test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/asgard/pandora/internal/robotics/control"
)

func TestHunoidHILMotionLoop(t *testing.T) {
	endpoint := os.Getenv("HUNOID_ENDPOINT")
	hunoidID := os.Getenv("HUNOID_ID")
	if endpoint == "" || hunoidID == "" {
		t.Skip("HUNOID_ENDPOINT and HUNOID_ID must be set for HIL motion loop")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	robot, err := control.NewRemoteHunoid(hunoidID, endpoint)
	if err != nil {
		t.Fatalf("hunoid client init failed: %v", err)
	}
	if err := robot.Initialize(ctx); err != nil {
		t.Fatalf("hunoid init failed: %v", err)
	}

	target := control.Pose{
		Position: control.Vector3{X: 0.2, Y: 0.1, Z: 0.0},
		Orientation: control.Quaternion{
			W: 1, X: 0, Y: 0, Z: 0,
		},
		Timestamp: time.Now(),
	}

	if err := robot.MoveTo(ctx, target); err != nil {
		t.Fatalf("move failed: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for robot.IsMoving() && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}

	if robot.IsMoving() {
		t.Fatal("robot did not reach target within timeout")
	}

	pose, err := robot.GetCurrentPose()
	if err != nil {
		t.Fatalf("get pose failed: %v", err)
	}
	if pose.Position.X == 0 && pose.Position.Y == 0 {
		t.Fatal("pose did not update from initial position")
	}
}
