// Package middleware provides HTTP middleware for the API server.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/asgard/pandora/internal/services"
)

// RequireAuth creates middleware that requires authentication.
func RequireAuth(authService *services.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

			ctx := contextWithAuthClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuth creates middleware that optionally authenticates.
func OptionalAuth(authService *services.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r)
			if token != "" {
				if claims, err := authService.ValidateToken(token); err == nil {
					ctx := contextWithAuthClaims(r.Context(), claims)
					r = r.WithContext(ctx)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	return r.URL.Query().Get("token")
}

func contextWithAuthClaims(ctx context.Context, claims services.TokenClaims) context.Context {
	return context.WithValue(ctx, "auth_claims", claims)
}
