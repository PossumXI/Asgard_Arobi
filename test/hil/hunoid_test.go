package hil

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/asgard/pandora/internal/robotics/control"
	"github.com/asgard/pandora/internal/robotics/ethics"
	"github.com/asgard/pandora/internal/robotics/vla"
)

// TestHunoidHILSuite runs all Hunoid HIL tests
func TestHunoidHILSuite(t *testing.T) {
	config := DefaultConfig()
	config.SilenusEnabled = false // Only test Hunoid
	config.VerboseLogging = testing.Verbose()

	suite := NewHILTestSuite(config)
	if err := suite.SetupHardware(); err != nil {
		t.Skipf("HIL hardware unavailable: %v", err)
	}
	defer func() {
		if err := suite.Close(); err != nil {
			t.Errorf("Teardown failed: %v", err)
		}
		suite.PrintSummary()
	}()

	// Motion controller tests
	suite.RunTest(t, "Motion/Initialize", testMotionInitialize)
	suite.RunTest(t, "Motion/GetPose", testMotionGetPose)
	suite.RunTest(t, "Motion/JointStates", testMotionJointStates)
	suite.RunTest(t, "Motion/SetJointPositions", testMotionSetJointPositions)
	suite.RunTestWithContext(t, "Motion/MoveTo", 10*time.Second, testMotionMoveTo)

	// Manipulator tests
	suite.RunTest(t, "Manipulator/GripperOpen", testManipulatorGripperOpen)
	suite.RunTest(t, "Manipulator/GripperClose", testManipulatorGripperClose)
	suite.RunTest(t, "Manipulator/GripperCycle", testManipulatorGripperCycle)
	suite.RunTestWithContext(t, "Manipulator/ReachTo", 5*time.Second, testManipulatorReachTo)
	suite.RunTest(t, "Manipulator/ReachLimits", testManipulatorReachLimits)

	// Ethics kernel tests
	suite.RunTest(t, "Ethics/SafeAction", testEthicsSafeAction)
	suite.RunTest(t, "Ethics/HarmfulAction", testEthicsHarmfulAction)
	suite.RunTest(t, "Ethics/LowConfidenceAction", testEthicsLowConfidenceAction)
	suite.RunTest(t, "Ethics/TransparencyViolation", testEthicsTransparencyViolation)

	// VLA and mission execution tests
	suite.RunTest(t, "VLA/ModelInfo", testVLAModelInfo)
	suite.RunTest(t, "VLA/InferAction", testVLAInferAction)
	suite.RunTestWithContext(t, "Mission/PickAndPlace", 15*time.Second, testMissionPickAndPlace)
	suite.RunTestWithContext(t, "Mission/NavigateAndInspect", 60*time.Second, testMissionNavigateAndInspect)
}

// =============================================================================
// Motion Controller Tests
// =============================================================================

func testMotionInitialize(t *testing.T, suite *HILTestSuite) {
	motion := suite.Hunoid().Motion()
	if motion == nil {
		t.Fatal("Motion controller is nil")
	}

	// Motion should already be initialized by suite setup
	// Verify by getting current pose
	pose, err := motion.GetCurrentPose()
	if err != nil {
		t.Fatalf("Failed to get current pose: %v", err)
	}

	// Verify pose has valid timestamp
	if pose.Timestamp.IsZero() {
		t.Error("Pose timestamp is zero")
	}
}

func testMotionGetPose(t *testing.T, suite *HILTestSuite) {
	motion := suite.Hunoid().Motion()

	pose, err := motion.GetCurrentPose()
	if err != nil {
		t.Fatalf("Failed to get pose: %v", err)
	}

	// Log position
	t.Logf("Position: x=%.3f, y=%.3f, z=%.3f",
		pose.Position.X, pose.Position.Y, pose.Position.Z)

	// Verify quaternion is normalized (w²+x²+y²+z² ≈ 1)
	qMag := pose.Orientation.W*pose.Orientation.W +
		pose.Orientation.X*pose.Orientation.X +
		pose.Orientation.Y*pose.Orientation.Y +
		pose.Orientation.Z*pose.Orientation.Z

	if math.Abs(qMag-1.0) > 0.01 {
		t.Errorf("Quaternion not normalized: magnitude = %f", qMag)
	}

	suite.RecordMetric("pose_x", pose.Position.X)
	suite.RecordMetric("pose_y", pose.Position.Y)
	suite.RecordMetric("pose_z", pose.Position.Z)
}

func testMotionJointStates(t *testing.T, suite *HILTestSuite) {
	motion := suite.Hunoid().Motion()

	joints, err := motion.GetJointStates()
	if err != nil {
		t.Fatalf("Failed to get joint states: %v", err)
	}

	if len(joints) == 0 {
		t.Fatal("No joints returned")
	}

	// Verify expected joints exist
	expectedJoints := []string{
		"head_pan", "head_tilt",
		"left_shoulder", "left_elbow", "left_wrist",
		"right_shoulder", "right_elbow", "right_wrist",
		"left_hip", "left_knee", "left_ankle",
		"right_hip", "right_knee", "right_ankle",
	}

	jointMap := make(map[string]control.Joint)
	for _, j := range joints {
		jointMap[j.ID] = j
	}

	for _, expected := range expectedJoints {
		if _, exists := jointMap[expected]; !exists {
			t.Errorf("Missing expected joint: %s", expected)
		}
	}

	t.Logf("Found %d joints", len(joints))
	suite.RecordMetric("joint_count", float64(len(joints)))
}

func testMotionSetJointPositions(t *testing.T, suite *HILTestSuite) {
	motion := suite.Hunoid().Motion()

	// Set some joint positions
	positions := map[string]float64{
		"head_pan":  0.5,  // Turn head right
		"head_tilt": -0.2, // Tilt head down
	}

	if err := motion.SetJointPositions(positions); err != nil {
		t.Fatalf("Failed to set joint positions: %v", err)
	}

	// Verify positions were set
	joints, err := motion.GetJointStates()
	if err != nil {
		t.Fatalf("Failed to get joint states: %v", err)
	}

	for _, joint := range joints {
		if expected, ok := positions[joint.ID]; ok {
			if math.Abs(joint.Position-expected) > 0.01 {
				t.Errorf("Joint %s position mismatch: expected %.3f, got %.3f",
					joint.ID, expected, joint.Position)
			}
		}
	}
}

func testMotionMoveTo(ctx context.Context, t *testing.T, suite *HILTestSuite) {
	suite.SkipIfSlow(t)

	motion := suite.Hunoid().Motion()

	// Get current pose
	startPose, err := motion.GetCurrentPose()
	if err != nil {
		t.Fatalf("Failed to get start pose: %v", err)
	}

	// Create target pose 0.5m forward
	targetPose := control.Pose{
		Position: control.Vector3{
			X: startPose.Position.X + 0.5,
			Y: startPose.Position.Y,
			Z: startPose.Position.Z,
		},
		Orientation: startPose.Orientation,
		Timestamp:   time.Now(),
	}

	// Start movement
	if err := motion.MoveTo(ctx, targetPose); err != nil {
		t.Fatalf("Failed to start movement: %v", err)
	}

	// Verify robot is moving
	time.Sleep(100 * time.Millisecond)
	if !motion.IsMoving() {
		t.Log("Robot not reported as moving (may have reached target quickly)")
	}

	// Wait for movement to complete with timeout
	moveStart := time.Now()
	for motion.IsMoving() {
		select {
		case <-ctx.Done():
			t.Fatal("Context cancelled during movement")
		case <-time.After(100 * time.Millisecond):
			// Continue waiting
		}

		if time.Since(moveStart) > 8*time.Second {
			if err := motion.Stop(); err != nil {
				t.Errorf("Failed to stop motion: %v", err)
			}
			t.Fatal("Movement took too long")
		}
	}

	moveDuration := time.Since(moveStart)

	// Verify final position
	finalPose, err := motion.GetCurrentPose()
	if err != nil {
		t.Fatalf("Failed to get final pose: %v", err)
	}

	// Calculate distance to target
	dx := finalPose.Position.X - targetPose.Position.X
	dy := finalPose.Position.Y - targetPose.Position.Y
	dz := finalPose.Position.Z - targetPose.Position.Z
	distance := math.Sqrt(dx*dx + dy*dy + dz*dz)

	if distance > 0.05 { // 5cm tolerance
		t.Errorf("Final position %.3fm from target", distance)
	}

	t.Logf("Moved to target in %v (final distance: %.3fm)", moveDuration, distance)

	suite.RecordMetric("move_duration_ms", float64(moveDuration.Milliseconds()))
	suite.RecordMetric("final_distance", distance)
}

// =============================================================================
// Manipulator Tests
// =============================================================================

func testManipulatorGripperOpen(t *testing.T, suite *HILTestSuite) {
	manip := suite.Hunoid().Manipulator()
	if manip == nil {
		t.Fatal("Manipulator is nil")
	}

	// Open gripper
	if err := manip.OpenGripper(); err != nil {
		t.Fatalf("Failed to open gripper: %v", err)
	}

	// Verify state
	state, err := manip.GetGripperState()
	if err != nil {
		t.Fatalf("Failed to get gripper state: %v", err)
	}

	if state < 0.9 { // Should be ~1.0 (fully open)
		t.Errorf("Gripper not fully open: state = %.2f", state)
	}

	suite.RecordMetric("gripper_state", state)
}

func testManipulatorGripperClose(t *testing.T, suite *HILTestSuite) {
	manip := suite.Hunoid().Manipulator()

	// Close gripper
	if err := manip.CloseGripper(); err != nil {
		t.Fatalf("Failed to close gripper: %v", err)
	}

	// Verify state
	state, err := manip.GetGripperState()
	if err != nil {
		t.Fatalf("Failed to get gripper state: %v", err)
	}

	if state > 0.1 { // Should be ~0.0 (fully closed)
		t.Errorf("Gripper not fully closed: state = %.2f", state)
	}

	suite.RecordMetric("gripper_state", state)
}

func testManipulatorGripperCycle(t *testing.T, suite *HILTestSuite) {
	manip := suite.Hunoid().Manipulator()

	// Cycle gripper multiple times
	for i := 0; i < 3; i++ {
		if err := manip.OpenGripper(); err != nil {
			t.Fatalf("Cycle %d: Failed to open gripper: %v", i, err)
		}

		openState, _ := manip.GetGripperState()
		if openState < 0.9 {
			t.Errorf("Cycle %d: Gripper not fully open: %.2f", i, openState)
		}

		if err := manip.CloseGripper(); err != nil {
			t.Fatalf("Cycle %d: Failed to close gripper: %v", i, err)
		}

		closedState, _ := manip.GetGripperState()
		if closedState > 0.1 {
			t.Errorf("Cycle %d: Gripper not fully closed: %.2f", i, closedState)
		}
	}

	t.Log("Gripper cycled successfully 3 times")
}

func testManipulatorReachTo(ctx context.Context, t *testing.T, suite *HILTestSuite) {
	manip := suite.Hunoid().Manipulator()

	// Test reaching to various positions
	positions := []control.Vector3{
		{X: 0.3, Y: 0.0, Z: 0.5},
		{X: 0.4, Y: 0.2, Z: 0.3},
		{X: 0.2, Y: -0.2, Z: 0.6},
	}

	for i, pos := range positions {
		if err := manip.ReachTo(ctx, pos); err != nil {
			t.Errorf("Position %d: Failed to reach (%.2f, %.2f, %.2f): %v",
				i, pos.X, pos.Y, pos.Z, err)
		}
	}

	t.Log("Reached all target positions successfully")
}

func testManipulatorReachLimits(t *testing.T, suite *HILTestSuite) {
	manip := suite.Hunoid().Manipulator()
	ctx := suite.Context()

	// Test position outside reach envelope
	outOfReach := control.Vector3{X: 2.0, Y: 0.0, Z: 0.0}
	err := manip.ReachTo(ctx, outOfReach)

	if err == nil {
		t.Error("Expected error when reaching outside envelope")
	} else {
		t.Logf("Correctly rejected out-of-reach position: %v", err)
	}
}

// =============================================================================
// Ethics Kernel Tests
// =============================================================================

func testEthicsSafeAction(t *testing.T, suite *HILTestSuite) {
	ethicsKernel := suite.Hunoid().Ethics()
	if ethicsKernel == nil {
		t.Fatal("Ethics kernel is nil")
	}

	ctx := suite.Context()

	// Create a safe action
	action := &vla.Action{
		Type:       vla.ActionInspect,
		Confidence: 0.95,
		Parameters: map[string]interface{}{
			"target": "shelf",
		},
	}

	decision, err := ethicsKernel.Evaluate(ctx, action)
	if err != nil {
		t.Fatalf("Ethics evaluation failed: %v", err)
	}

	if decision.Decision != ethics.DecisionApproved {
		t.Errorf("Safe action should be approved, got: %s (%s)",
			decision.Decision, decision.Reasoning)
	}

	t.Logf("Safe action decision: %s (score: %.2f)", decision.Decision, decision.Score)

	suite.RecordMetric("ethics_score", decision.Score)
}

func testEthicsHarmfulAction(t *testing.T, suite *HILTestSuite) {
	ethicsKernel := suite.Hunoid().Ethics()
	ctx := suite.Context()

	// Create a potentially harmful action
	action := &vla.Action{
		Type:       vla.ActionPickUp,
		Confidence: 0.9,
		Parameters: map[string]interface{}{
			"target": "object",
			"force":  "aggressive", // This should trigger NoHarmRule
		},
	}

	decision, err := ethicsKernel.Evaluate(ctx, action)
	if err != nil {
		t.Fatalf("Ethics evaluation failed: %v", err)
	}

	// Should be rejected or escalated
	if decision.Decision == ethics.DecisionApproved {
		t.Error("Harmful action should not be approved")
	}

	t.Logf("Harmful action decision: %s (%s)", decision.Decision, decision.Reasoning)
}

func testEthicsLowConfidenceAction(t *testing.T, suite *HILTestSuite) {
	ethicsKernel := suite.Hunoid().Ethics()
	ctx := suite.Context()

	// Create a low-confidence critical action
	action := &vla.Action{
		Type:       vla.ActionPickUp,
		Confidence: 0.3, // Below 0.6 threshold
		Parameters: map[string]interface{}{
			"target": "fragile_item",
			"force":  "gentle",
		},
	}

	decision, err := ethicsKernel.Evaluate(ctx, action)
	if err != nil {
		t.Fatalf("Ethics evaluation failed: %v", err)
	}

	// Should be escalated due to ProportionalityRule
	if decision.Decision == ethics.DecisionApproved {
		t.Error("Low confidence critical action should not be approved")
	}

	// Verify rule was checked
	checked := false
	for _, rule := range decision.RulesChecked {
		if rule == "proportionality" {
			checked = true
			break
		}
	}

	if !checked {
		t.Error("Proportionality rule was not checked")
	}

	t.Logf("Low confidence decision: %s (rules: %v)", decision.Decision, decision.RulesChecked)
}

func testEthicsTransparencyViolation(t *testing.T, suite *HILTestSuite) {
	ethicsKernel := suite.Hunoid().Ethics()
	ctx := suite.Context()

	// Create an action without parameters (except ActionWait)
	action := &vla.Action{
		Type:       vla.ActionNavigate, // Requires parameters
		Confidence: 0.9,
		Parameters: map[string]interface{}{}, // Empty parameters
	}

	decision, err := ethicsKernel.Evaluate(ctx, action)
	if err != nil {
		t.Fatalf("Ethics evaluation failed: %v", err)
	}

	// Should flag transparency issue
	if decision.Score >= 1.0 {
		t.Error("Action without parameters should have reduced score")
	}

	t.Logf("Transparency violation decision: %s (score: %.2f)", decision.Decision, decision.Score)
}

// =============================================================================
// VLA Model Tests
// =============================================================================

func testVLAModelInfo(t *testing.T, suite *HILTestSuite) {
	vlaModel := suite.Hunoid().VLA()
	if vlaModel == nil {
		t.Fatal("VLA model is nil")
	}

	info := vlaModel.GetModelInfo()

	if info.Name == "" {
		t.Error("Model name is empty")
	}

	if len(info.SupportedActions) == 0 {
		t.Error("No supported actions")
	}

	t.Logf("VLA Model: %s v%s (supports %d actions)",
		info.Name, info.Version, len(info.SupportedActions))
}

func testVLAInferAction(t *testing.T, suite *HILTestSuite) {
	vlaModel := suite.Hunoid().VLA()
	ctx := suite.Context()

	// Initialize VLA
	if err := vlaModel.Initialize(ctx, "mock_model.pt"); err != nil {
		t.Fatalf("Failed to initialize VLA: %v", err)
	}

	// Test various commands
	testCases := []struct {
		command      string
		expectedType vla.ActionType
	}{
		{"pick up the box", vla.ActionPickUp},
		{"put down the item", vla.ActionPutDown},
		{"navigate to the door", vla.ActionNavigate},
		{"open the drawer", vla.ActionOpen},
		{"close the cabinet", vla.ActionClose},
		{"inspect the area", vla.ActionInspect},
		{"do nothing", vla.ActionWait},
	}

	for _, tc := range testCases {
		action, err := vlaModel.InferAction(ctx, []byte{}, tc.command)
		if err != nil {
			t.Errorf("Command '%s': inference failed: %v", tc.command, err)
			continue
		}

		if action.Type != tc.expectedType {
			t.Errorf("Command '%s': expected %s, got %s",
				tc.command, tc.expectedType, action.Type)
		}

		if action.Confidence <= 0 || action.Confidence > 1 {
			t.Errorf("Command '%s': invalid confidence: %.2f",
				tc.command, action.Confidence)
		}
	}
}

// =============================================================================
// Mission Execution Pipeline Tests
// =============================================================================

func testMissionPickAndPlace(ctx context.Context, t *testing.T, suite *HILTestSuite) {
	suite.SkipIfSlow(t)

	motion := suite.Hunoid().Motion()
	manip := suite.Hunoid().Manipulator()
	ethicsKernel := suite.Hunoid().Ethics()
	vlaModel := suite.Hunoid().VLA()

	// Initialize VLA
	if err := vlaModel.Initialize(ctx, "mock_model.pt"); err != nil {
		t.Fatalf("Failed to initialize VLA: %v", err)
	}

	// Simulate pick and place mission
	steps := []string{
		"move to the table",
		"pick up the box",
		"move to the shelf",
		"put down the box",
	}

	for _, step := range steps {
		// 1. Get action from VLA
		action, err := vlaModel.InferAction(ctx, []byte{}, step)
		if err != nil {
			t.Fatalf("Step '%s': VLA inference failed: %v", step, err)
		}

		// 2. Evaluate with ethics kernel
		decision, err := ethicsKernel.Evaluate(ctx, action)
		if err != nil {
			t.Fatalf("Step '%s': ethics evaluation failed: %v", step, err)
		}

		if decision.Decision == ethics.DecisionRejected {
			t.Fatalf("Step '%s': action rejected: %s", step, decision.Reasoning)
		}

		// 3. Execute action (simplified)
		switch action.Type {
		case vla.ActionNavigate:
			targetPose := control.Pose{
				Position:    control.Vector3{X: 1.0, Y: 0.0, Z: 0.0},
				Orientation: control.Quaternion{W: 1, X: 0, Y: 0, Z: 0},
			}
			if err := motion.MoveTo(ctx, targetPose); err != nil {
				t.Errorf("Step '%s': move failed: %v", step, err)
			}
			// Wait for movement
			for motion.IsMoving() {
				select {
				case <-ctx.Done():
					return
				case <-time.After(100 * time.Millisecond):
				}
			}

		case vla.ActionPickUp:
			if err := manip.OpenGripper(); err != nil {
				t.Errorf("Step '%s': open gripper failed: %v", step, err)
			}
			if err := manip.ReachTo(ctx, control.Vector3{X: 0.4, Y: 0, Z: 0.3}); err != nil {
				t.Errorf("Step '%s': reach failed: %v", step, err)
			}
			if err := manip.CloseGripper(); err != nil {
				t.Errorf("Step '%s': close gripper failed: %v", step, err)
			}

		case vla.ActionPutDown:
			if err := manip.ReachTo(ctx, control.Vector3{X: 0.3, Y: 0.2, Z: 0.5}); err != nil {
				t.Errorf("Step '%s': reach failed: %v", step, err)
			}
			if err := manip.OpenGripper(); err != nil {
				t.Errorf("Step '%s': open gripper failed: %v", step, err)
			}
		}

		t.Logf("Completed step: %s", step)
	}

	t.Log("Pick and place mission completed successfully")
}

func testMissionNavigateAndInspect(ctx context.Context, t *testing.T, suite *HILTestSuite) {
	suite.SkipIfSlow(t)

	motion := suite.Hunoid().Motion()
	ethicsKernel := suite.Hunoid().Ethics()
	vlaModel := suite.Hunoid().VLA()

	// Initialize VLA
	if err := vlaModel.Initialize(ctx, "mock_model.pt"); err != nil {
		t.Fatalf("Failed to initialize VLA: %v", err)
	}

	// Define waypoints to visit and inspect
	waypoints := []control.Pose{
		{Position: control.Vector3{X: 1.0, Y: 0.0, Z: 0.0}, Orientation: control.Quaternion{W: 1}},
		{Position: control.Vector3{X: 1.0, Y: 1.0, Z: 0.0}, Orientation: control.Quaternion{W: 1}},
		{Position: control.Vector3{X: 0.0, Y: 1.0, Z: 0.0}, Orientation: control.Quaternion{W: 1}},
		{Position: control.Vector3{X: 0.0, Y: 0.0, Z: 0.0}, Orientation: control.Quaternion{W: 1}},
	}

	for i, wp := range waypoints {
		// Navigate to waypoint
		t.Logf("Navigating to waypoint %d", i+1)

		// Check ethics for navigation
		navAction := &vla.Action{
			Type:       vla.ActionNavigate,
			Confidence: 0.9,
			Parameters: map[string]interface{}{
				"x": wp.Position.X,
				"y": wp.Position.Y,
			},
		}

		decision, _ := ethicsKernel.Evaluate(ctx, navAction)
		if decision.Decision == ethics.DecisionRejected {
			t.Errorf("Navigation to waypoint %d rejected", i+1)
			continue
		}

		if err := motion.MoveTo(ctx, wp); err != nil {
			t.Errorf("Failed to move to waypoint %d: %v", i+1, err)
			continue
		}

		// Wait for arrival (mock robot moves at ~0.1m/s, so 1m takes ~10s)
		timeout := time.After(12 * time.Second)
	waitLoop:
		for motion.IsMoving() {
			select {
			case <-ctx.Done():
				t.Log("Context cancelled during navigation")
				motion.Stop()
				break waitLoop
			case <-timeout:
				motion.Stop()
				t.Logf("Waypoint %d: navigation timeout (acceptable for mock)", i+1)
				break waitLoop
			case <-time.After(100 * time.Millisecond):
			}
		}

		// Simulate inspection at waypoint
		inspectAction, _ := vlaModel.InferAction(ctx, []byte{}, "inspect the area")
		inspectDecision, _ := ethicsKernel.Evaluate(ctx, inspectAction)

		if inspectDecision.Decision != ethics.DecisionRejected {
			t.Logf("Inspection at waypoint %d: approved (score: %.2f)",
				i+1, inspectDecision.Score)
		}
	}

	t.Log("Navigate and inspect mission completed")
}

// =============================================================================
// Integration Test with Both Silenus and Hunoid
// =============================================================================

func TestFullSystemIntegration(t *testing.T) {
	config := DefaultConfig()
	config.VerboseLogging = testing.Verbose()

	suite := NewHILTestSuite(config)
	if err := suite.SetupHardware(); err != nil {
		t.Skipf("HIL hardware unavailable: %v", err)
	}
	defer suite.Close()

	// Test that both systems can be accessed
	if suite.Silenus() == nil {
		t.Error("Silenus adapter not initialized")
	}

	if suite.Hunoid() == nil {
		t.Error("Hunoid adapter not initialized")
	}

	// Test camera capture and VLA inference together
	ctx := context.Background()

	camera := suite.Silenus().Camera()
	vlaModel := suite.Hunoid().VLA()

	// Capture image
	frame, err := camera.CaptureFrame(ctx)
	if err != nil {
		t.Fatalf("Failed to capture frame: %v", err)
	}

	// Initialize and infer with VLA
	if err := vlaModel.Initialize(ctx, "mock.pt"); err != nil {
		t.Fatalf("Failed to init VLA: %v", err)
	}

	action, err := vlaModel.InferAction(ctx, frame, "pick up the object in front")
	if err != nil {
		t.Fatalf("VLA inference failed: %v", err)
	}

	t.Logf("Full system test: captured %d bytes, inferred action: %s (conf: %.2f)",
		len(frame), action.Type, action.Confidence)
}
