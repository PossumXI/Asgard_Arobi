// Package middleware provides HTTP middleware for the API server.
package middleware

import (
	"net/http"

	"github.com/asgard/pandora/internal/repositories"
	"github.com/asgard/pandora/internal/services"
)

// RequireEmailVerified creates middleware that requires the user's email to be verified.
// This is required for government-grade access and sensitive operations.
func RequireEmailVerified(userRepo *repositories.UserRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value("auth_claims").(services.TokenClaims)
			if !ok || claims.UserID == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			user, err := userRepo.GetByID(claims.UserID)
			if err != nil {
				http.Error(w, "User not found", http.StatusUnauthorized)
				return
			}

			if !user.EmailVerified {
				http.Error(w, "Email verification required", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireGovernmentMFA creates middleware that requires government users to have FIDO2/WebAuthn setup.
// Government-grade access requires phishing-resistant MFA per NIST 800-63B guidelines.
func RequireGovernmentMFA(userRepo *repositories.UserRepository, webauthnRepo *repositories.WebAuthnRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value("auth_claims").(services.TokenClaims)
			if !ok || claims.UserID == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			user, err := userRepo.GetByID(claims.UserID)
			if err != nil {
				http.Error(w, "User not found", http.StatusUnauthorized)
				return
			}

			// Only enforce MFA requirement for government users
			if !user.IsGovernment {
				next.ServeHTTP(w, r)
				return
			}

			// Check if government user has FIDO2 credentials registered
			if webauthnRepo == nil {
				http.Error(w, "MFA service unavailable", http.StatusServiceUnavailable)
				return
			}

			creds, err := webauthnRepo.GetCredentialsByUserID(claims.UserID)
			if err != nil {
				http.Error(w, "Failed to verify MFA status", http.StatusInternalServerError)
				return
			}

			if len(creds) == 0 {
				http.Error(w, "FIDO2 MFA setup required for government access", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireGovernmentAccess combines email verification and MFA requirements for government users.
// This is a convenience middleware that enforces both checks in sequence.
func RequireGovernmentAccess(userRepo *repositories.UserRepository, webauthnRepo *repositories.WebAuthnRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value("auth_claims").(services.TokenClaims)
			if !ok || claims.UserID == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			user, err := userRepo.GetByID(claims.UserID)
			if err != nil {
				http.Error(w, "User not found", http.StatusUnauthorized)
				return
			}

			// Check email verification
			if !user.EmailVerified {
				http.Error(w, "Email verification required for government access", http.StatusForbidden)
				return
			}

			// For government users, also check FIDO2 MFA
			if user.IsGovernment {
				if webauthnRepo == nil {
					http.Error(w, "MFA service unavailable", http.StatusServiceUnavailable)
					return
				}

				creds, err := webauthnRepo.GetCredentialsByUserID(claims.UserID)
				if err != nil {
					http.Error(w, "Failed to verify MFA status", http.StatusInternalServerError)
					return
				}

				if len(creds) == 0 {
					http.Error(w, "FIDO2 MFA setup required for government access", http.StatusForbidden)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
