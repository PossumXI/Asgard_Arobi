// Package services provides business logic services for the API.
package services

import (
	"context"
	"testing"
	"time"
)

func TestDefaultStreamSessionConfig(t *testing.T) {
	config := DefaultStreamSessionConfig()

	// Verify default ICE servers
	if len(config.ICEServers) == 0 {
		t.Error("DefaultStreamSessionConfig() should have at least one ICE server")
	}

	// Verify STUN servers are configured
	found := false
	for _, server := range config.ICEServers {
		for _, url := range server.URLs {
			if url == "stun:stun.l.google.com:19302" {
				found = true
				break
			}
		}
	}
	if !found {
		t.Error("DefaultStreamSessionConfig() should include Google STUN server")
	}

	// Verify session TTL
	if config.SessionTTL != 24*time.Hour {
		t.Errorf("SessionTTL = %v, want %v", config.SessionTTL, 24*time.Hour)
	}
}

func TestNewStreamService(t *testing.T) {
	service := NewStreamService(nil)
	if service == nil {
		t.Fatal("NewStreamService() returned nil")
	}

	// Verify default config is applied
	if service.sessionConfig.SessionTTL != 24*time.Hour {
		t.Errorf("SessionTTL = %v, want %v", service.sessionConfig.SessionTTL, 24*time.Hour)
	}

	// Verify sessions map is initialized
	if service.sessions == nil {
		t.Error("sessions map should be initialized")
	}
}

func TestNewStreamServiceWithConfig(t *testing.T) {
	customConfig := StreamSessionConfig{
		ICEServers: []ICEServerConfig{
			{URLs: []string{"stun:custom.stun.server:3478"}},
		},
		SignalingURL: "ws://custom.signaling.server/ws",
		SessionTTL:   1 * time.Hour,
	}

	service := NewStreamServiceWithConfig(nil, customConfig)
	if service == nil {
		t.Fatal("NewStreamServiceWithConfig() returned nil")
	}

	if service.sessionConfig.SessionTTL != 1*time.Hour {
		t.Errorf("SessionTTL = %v, want %v", service.sessionConfig.SessionTTL, 1*time.Hour)
	}
	if service.sessionConfig.SignalingURL != "ws://custom.signaling.server/ws" {
		t.Errorf("SignalingURL = %v, want %v", service.sessionConfig.SignalingURL, "ws://custom.signaling.server/ws")
	}
}

func TestSetSessionConfig(t *testing.T) {
	service := NewStreamService(nil)

	newConfig := StreamSessionConfig{
		SessionTTL: 2 * time.Hour,
	}

	service.SetSessionConfig(newConfig)

	if service.sessionConfig.SessionTTL != 2*time.Hour {
		t.Errorf("SessionTTL = %v, want %v", service.sessionConfig.SessionTTL, 2*time.Hour)
	}
}

func TestValidateSession_NotFound(t *testing.T) {
	service := NewStreamService(nil)

	streamID, userID, valid := service.ValidateSession("non-existent-session", "token")
	if valid {
		t.Error("ValidateSession() should return false for non-existent session")
	}
	if streamID != "" {
		t.Errorf("streamID = %v, want empty string", streamID)
	}
	if userID != "" {
		t.Errorf("userID = %v, want empty string", userID)
	}
}

func TestValidateSession_Expired(t *testing.T) {
	service := NewStreamService(nil)

	// Manually add an expired session
	sessionID := "test-session-id"
	service.sessions[sessionID] = sessionRecord{
		streamID:  "stream-123",
		userID:    "user-456",
		authToken: "valid-token",
		expiresAt: time.Now().Add(-1 * time.Hour), // Expired
	}

	streamID, userID, valid := service.ValidateSession(sessionID, "valid-token")
	if valid {
		t.Error("ValidateSession() should return false for expired session")
	}
	if streamID != "" {
		t.Errorf("streamID = %v, want empty string", streamID)
	}
	if userID != "" {
		t.Errorf("userID = %v, want empty string", userID)
	}

	// Verify session was cleaned up
	service.mu.RLock()
	_, exists := service.sessions[sessionID]
	service.mu.RUnlock()
	if exists {
		t.Error("expired session should be removed from sessions map")
	}
}

func TestValidateSession_InvalidToken(t *testing.T) {
	service := NewStreamService(nil)

	// Manually add a valid session
	sessionID := "test-session-id"
	service.sessions[sessionID] = sessionRecord{
		streamID:  "stream-123",
		userID:    "user-456",
		authToken: "correct-token",
		expiresAt: time.Now().Add(1 * time.Hour),
	}

	streamID, userID, valid := service.ValidateSession(sessionID, "wrong-token")
	if valid {
		t.Error("ValidateSession() should return false for invalid token")
	}
	if streamID != "" {
		t.Errorf("streamID = %v, want empty string", streamID)
	}
	if userID != "" {
		t.Errorf("userID = %v, want empty string", userID)
	}
}

func TestValidateSession_Valid(t *testing.T) {
	service := NewStreamService(nil)

	// Manually add a valid session
	sessionID := "test-session-id"
	expectedStreamID := "stream-123"
	expectedUserID := "user-456"
	expectedToken := "correct-token"

	service.sessions[sessionID] = sessionRecord{
		streamID:  expectedStreamID,
		userID:    expectedUserID,
		authToken: expectedToken,
		expiresAt: time.Now().Add(1 * time.Hour),
	}

	streamID, userID, valid := service.ValidateSession(sessionID, expectedToken)
	if !valid {
		t.Error("ValidateSession() should return true for valid session")
	}
	if streamID != expectedStreamID {
		t.Errorf("streamID = %v, want %v", streamID, expectedStreamID)
	}
	if userID != expectedUserID {
		t.Errorf("userID = %v, want %v", userID, expectedUserID)
	}
}

func TestTierOrder(t *testing.T) {
	// Verify tier ordering
	tests := []struct {
		tier          string
		expectedOrder int
	}{
		{"free", 0},
		{"observer", 1},
		{"supporter", 2},
		{"commander", 3},
	}

	for _, tt := range tests {
		t.Run(tt.tier, func(t *testing.T) {
			order, ok := tierOrder[tt.tier]
			if !ok {
				t.Errorf("tier %q not found in tierOrder", tt.tier)
				return
			}
			if order != tt.expectedOrder {
				t.Errorf("tierOrder[%q] = %d, want %d", tt.tier, order, tt.expectedOrder)
			}
		})
	}
}

func TestTierAtLeast(t *testing.T) {
	tests := []struct {
		name         string
		userTier     string
		requiredTier string
		want         bool
	}{
		// Free tier tests
		{"free can access free", "free", "free", true},
		{"free cannot access observer", "free", "observer", false},
		{"free cannot access supporter", "free", "supporter", false},
		{"free cannot access commander", "free", "commander", false},

		// Observer tier tests
		{"observer can access free", "observer", "free", true},
		{"observer can access observer", "observer", "observer", true},
		{"observer cannot access supporter", "observer", "supporter", false},
		{"observer cannot access commander", "observer", "commander", false},

		// Supporter tier tests
		{"supporter can access free", "supporter", "free", true},
		{"supporter can access observer", "supporter", "observer", true},
		{"supporter can access supporter", "supporter", "supporter", true},
		{"supporter cannot access commander", "supporter", "commander", false},

		// Commander tier tests
		{"commander can access free", "commander", "free", true},
		{"commander can access observer", "commander", "observer", true},
		{"commander can access supporter", "commander", "supporter", true},
		{"commander can access commander", "commander", "commander", true},

		// Unknown tier tests
		{"unknown user tier defaults to 0", "unknown", "free", true},
		{"unknown user tier cannot access observer", "unknown", "observer", false},
		{"unknown required tier defaults to 0", "observer", "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tierAtLeast(tt.userTier, tt.requiredTier)
			if got != tt.want {
				t.Errorf("tierAtLeast(%q, %q) = %v, want %v",
					tt.userTier, tt.requiredTier, got, tt.want)
			}
		})
	}
}

func TestGetAllowedStreamTypes(t *testing.T) {
	tests := []struct {
		name         string
		userTier     string
		wantTypes    []string
		notWantTypes []string
	}{
		{
			name:         "free tier gets no streams",
			userTier:     "free",
			wantTypes:    []string{},
			notWantTypes: []string{"civilian", "military", "interstellar"},
		},
		{
			name:         "observer tier gets civilian",
			userTier:     "observer",
			wantTypes:    []string{"civilian"},
			notWantTypes: []string{"military", "interstellar"},
		},
		{
			name:         "supporter tier gets civilian and military",
			userTier:     "supporter",
			wantTypes:    []string{"civilian", "military"},
			notWantTypes: []string{"interstellar"},
		},
		{
			name:         "commander tier gets all types",
			userTier:     "commander",
			wantTypes:    []string{"civilian", "military", "interstellar"},
			notWantTypes: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed := GetAllowedStreamTypes(tt.userTier)

			// Check expected types are present
			for _, wantType := range tt.wantTypes {
				found := false
				for _, gotType := range allowed {
					if gotType == wantType {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("GetAllowedStreamTypes(%q) missing %q", tt.userTier, wantType)
				}
			}

			// Check unwanted types are absent
			for _, notWantType := range tt.notWantTypes {
				for _, gotType := range allowed {
					if gotType == notWantType {
						t.Errorf("GetAllowedStreamTypes(%q) should not include %q", tt.userTier, notWantType)
					}
				}
			}
		})
	}
}

func TestCanAccessStreamType(t *testing.T) {
	tests := []struct {
		name       string
		userTier   string
		streamType string
		want       bool
	}{
		// Free tier
		{"free cannot access civilian", "free", "civilian", false},
		{"free cannot access military", "free", "military", false},
		{"free cannot access interstellar", "free", "interstellar", false},

		// Observer tier
		{"observer can access civilian", "observer", "civilian", true},
		{"observer cannot access military", "observer", "military", false},
		{"observer cannot access interstellar", "observer", "interstellar", false},

		// Supporter tier
		{"supporter can access civilian", "supporter", "civilian", true},
		{"supporter can access military", "supporter", "military", true},
		{"supporter cannot access interstellar", "supporter", "interstellar", false},

		// Commander tier
		{"commander can access civilian", "commander", "civilian", true},
		{"commander can access military", "commander", "military", true},
		{"commander can access interstellar", "commander", "interstellar", true},

		// Unknown stream type (public by default)
		{"free can access unknown type", "free", "unknown_type", true},
		{"observer can access unknown type", "observer", "unknown_type", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CanAccessStreamType(tt.userTier, tt.streamType)
			if got != tt.want {
				t.Errorf("CanAccessStreamType(%q, %q) = %v, want %v",
					tt.userTier, tt.streamType, got, tt.want)
			}
		})
	}
}

func TestGenerateSessionToken(t *testing.T) {
	token1, err := generateSessionToken()
	if err != nil {
		t.Fatalf("generateSessionToken() error = %v", err)
	}

	if token1 == "" {
		t.Error("generateSessionToken() returned empty token")
	}

	// Verify token length (32 bytes = 64 hex chars)
	if len(token1) != 64 {
		t.Errorf("generateSessionToken() length = %d, want 64", len(token1))
	}

	// Verify tokens are unique
	token2, err := generateSessionToken()
	if err != nil {
		t.Fatalf("generateSessionToken() error = %v", err)
	}

	if token1 == token2 {
		t.Error("generateSessionToken() should produce unique tokens")
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		envKey       string
		envValue     string
		defaultValue string
		want         string
	}{
		{
			name:         "returns env value when set",
			envKey:       "TEST_ENV_VAR",
			envValue:     "custom_value",
			defaultValue: "default",
			want:         "custom_value",
		},
		{
			name:         "returns default when env not set",
			envKey:       "TEST_ENV_VAR_UNSET",
			envValue:     "",
			defaultValue: "default_value",
			want:         "default_value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.envKey, tt.envValue)
			}

			got := getEnvOrDefault(tt.envKey, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getEnvOrDefault(%q, %q) = %v, want %v",
					tt.envKey, tt.defaultValue, got, tt.want)
			}
		})
	}
}

func TestICEServerConfig(t *testing.T) {
	config := ICEServerConfig{
		URLs:       []string{"stun:stun.example.com:3478", "turn:turn.example.com:3478"},
		Username:   "user",
		Credential: "pass",
	}

	if len(config.URLs) != 2 {
		t.Errorf("URLs length = %d, want 2", len(config.URLs))
	}
	if config.Username != "user" {
		t.Errorf("Username = %v, want user", config.Username)
	}
	if config.Credential != "pass" {
		t.Errorf("Credential = %v, want pass", config.Credential)
	}
}

func TestErrChatUnavailable(t *testing.T) {
	service := NewStreamService(nil)

	// Without a chat repo, chat operations should return ErrChatUnavailable
	_, err := service.ListChatMessages(context.Background(), "stream-id", 10)
	if err != ErrChatUnavailable {
		t.Errorf("ListChatMessages() error = %v, want %v", err, ErrChatUnavailable)
	}

	_, err = service.AddChatMessage(context.Background(), "stream-id", "user-id", "username", "message")
	if err != ErrChatUnavailable {
		t.Errorf("AddChatMessage() error = %v, want %v", err, ErrChatUnavailable)
	}
}

func TestStreamServiceConcurrency(t *testing.T) {
	service := NewStreamService(nil)

	// Test concurrent session validation
	sessionID := "concurrent-test-session"
	service.sessions[sessionID] = sessionRecord{
		streamID:  "stream-123",
		userID:    "user-456",
		authToken: "token",
		expiresAt: time.Now().Add(1 * time.Hour),
	}

	done := make(chan bool)
	for i := 0; i < 100; i++ {
		go func() {
			service.ValidateSession(sessionID, "token")
			done <- true
		}()
	}

	for i := 0; i < 100; i++ {
		<-done
	}
}
