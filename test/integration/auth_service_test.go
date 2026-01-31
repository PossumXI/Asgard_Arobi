package integration_test

import (
	"os"
	"testing"

	"github.com/asgard/pandora/internal/services"
)

func setupAuthTest(t *testing.T) {
	t.Helper()
	// Set required environment for auth service
	os.Setenv("ASGARD_ENV", "development")
}

func TestAuthServiceCreation(t *testing.T) {
	setupAuthTest(t)

	// Create auth service with nil repos (acceptable for basic testing)
	authService := services.NewAuthService(nil, nil, nil, nil)

	if authService == nil {
		t.Fatal("auth service should not be nil")
	}
}

func TestAuthServiceJWTValidation(t *testing.T) {
	setupAuthTest(t)

	// Create auth service
	authService := services.NewAuthService(nil, nil, nil, nil)

	// Invalid token should fail validation
	_, err := authService.ValidateToken("invalid.token.here")
	if err == nil {
		t.Error("expected error for invalid token")
	}

	// Empty token should fail
	_, err = authService.ValidateToken("")
	if err == nil {
		t.Error("expected error for empty token")
	}
}

func TestAuthServiceMalformedTokens(t *testing.T) {
	setupAuthTest(t)

	authService := services.NewAuthService(nil, nil, nil, nil)

	malformedTokens := []string{
		"not.a.jwt",
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",          // Missing parts
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..invalid", // Invalid signature
		"random.string.here",
		"",
	}

	for _, token := range malformedTokens {
		_, err := authService.ValidateToken(token)
		if err == nil {
			t.Errorf("expected error for token: %q", token)
		}
	}
}
