package middleware

import (
	"context"
	"net/http"
	"strings"

	"lockcenter-backend/internal/domain"
)

type contextKey string

const UserClaimsKey contextKey = "user_claims"

type TokenValidator interface {
	ValidateAccessToken(token string) (*domain.UserClaims, error)
}

func AuthMiddleware(validator TokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header is required", http.StatusUnauthorized)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
				return
			}

			claims, err := validator.ValidateAccessToken(parts[1])
			if err != nil {
				http.Error(w, "Invalid or expired access token", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserClaims(ctx context.Context) (*domain.UserClaims, bool) {
	claims, ok := ctx.Value(UserClaimsKey).(*domain.UserClaims)
	return claims, ok
}
