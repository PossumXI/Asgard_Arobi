// Package middleware provides HTTP middleware for the API server.
package middleware

import (
	"net/http"

	"github.com/asgard/pandora/internal/platform/realtime"
	"github.com/asgard/pandora/internal/services"
)

// TierOrder defines the subscription tier hierarchy.
// Higher index = higher privileges.
var TierOrder = map[string]int{
	"free":      0,
	"observer":  1,
	"supporter": 2,
	"commander": 3,
}

// TierAtLeast checks if userTier is at least the required tier level.
// Tier hierarchy: free < observer < supporter < commander
func TierAtLeast(userTier, requiredTier string) bool {
	userLevel, userOk := TierOrder[userTier]
	requiredLevel, reqOk := TierOrder[requiredTier]

	if !userOk {
		userLevel = 0 // Default to free tier if unknown
	}
	if !reqOk {
		requiredLevel = 0 // Default to free tier if unknown
	}

	return userLevel >= requiredLevel
}

// RequireTier creates middleware that requires a minimum subscription tier.
// Users must be authenticated and have at least the specified tier.
func RequireTier(authService *services.AuthService, minTier string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// First, require authentication
			token := extractToken(r)
			if token == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			claims, err := authService.ValidateToken(token)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Check tier
			if !TierAtLeast(claims.SubscriptionTier, minTier) {
				http.Error(w, "Forbidden: Insufficient subscription tier", http.StatusForbidden)
				return
			}

			// Add claims to context and continue
			ctx := contextWithAuthClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAccessLevel creates middleware that requires a minimum access level.
// Maps subscription tiers to access levels for authorization.
func RequireAccessLevel(authService *services.AuthService, level realtime.AccessLevel) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// First, require authentication
			token := extractToken(r)
			if token == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			claims, err := authService.ValidateToken(token)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Get user's access level from their role and government status
			userAccessLevel := realtime.AccessLevelFromUserRole(claims.Role, claims.IsGovernment)

			// Check if user has sufficient access
			if !AccessLevelAtLeast(userAccessLevel, level) {
				http.Error(w, "Forbidden: Insufficient access level", http.StatusForbidden)
				return
			}

			// Add claims to context and continue
			ctx := contextWithAuthClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AccessLevelAtLeast checks if clientLevel is at least the required level.
// Access level hierarchy: public < civilian < military < interstellar < government < admin
func AccessLevelAtLeast(clientLevel, requiredLevel realtime.AccessLevel) bool {
	levels := map[realtime.AccessLevel]int{
		realtime.AccessLevelPublic:       0,
		realtime.AccessLevelCivilian:     1,
		realtime.AccessLevelMilitary:     2,
		realtime.AccessLevelInterstellar: 3,
		realtime.AccessLevelGovernment:   4,
		realtime.AccessLevelAdmin:        5,
	}

	clientRank, ok1 := levels[clientLevel]
	requiredRank, ok2 := levels[requiredLevel]

	if !ok1 || !ok2 {
		return false
	}

	return clientRank >= requiredRank
}

// TierToAccessLevel maps subscription tiers to access levels.
func TierToAccessLevel(tier string) realtime.AccessLevel {
	switch tier {
	case "commander":
		return realtime.AccessLevelInterstellar
	case "supporter":
		return realtime.AccessLevelMilitary
	case "observer":
		return realtime.AccessLevelCivilian
	case "free":
		return realtime.AccessLevelPublic
	default:
		return realtime.AccessLevelPublic
	}
}

// StreamTypeToAccessLevel maps stream types to required access levels.
func StreamTypeToAccessLevel(streamType string) realtime.AccessLevel {
	switch streamType {
	case "interstellar":
		return realtime.AccessLevelInterstellar
	case "military":
		return realtime.AccessLevelMilitary
	case "civilian":
		return realtime.AccessLevelCivilian
	default:
		return realtime.AccessLevelPublic
	}
}

// CanAccessStreamType checks if a user with the given tier can access a stream type.
func CanAccessStreamType(userTier, streamType string) bool {
	userAccessLevel := TierToAccessLevel(userTier)
	requiredAccessLevel := StreamTypeToAccessLevel(streamType)
	return AccessLevelAtLeast(userAccessLevel, requiredAccessLevel)
}

// GetAllowedStreamTypes returns the stream types a user can access based on their tier.
func GetAllowedStreamTypes(userTier string) []string {
	allowed := []string{}

	// Free tier gets no streams (must be at least observer)
	if TierAtLeast(userTier, "observer") {
		allowed = append(allowed, "civilian")
	}
	if TierAtLeast(userTier, "supporter") {
		allowed = append(allowed, "military")
	}
	if TierAtLeast(userTier, "commander") {
		allowed = append(allowed, "interstellar")
	}

	return allowed
}
