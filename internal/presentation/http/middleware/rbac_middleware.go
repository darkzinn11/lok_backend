package middleware

import (
	"fmt"
	"net/http"

	"lockcenter-backend/internal/domain"
)

func RBACMiddleware(allowedRoles ...domain.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := GetUserClaims(r.Context())
			if !ok {
				http.Error(w, "user claims not found in context", http.StatusForbidden)
				return
			}

			// Director can access everything
			if claims.Role == domain.RoleDirector {
				next.ServeHTTP(w, r)
				return
			}

			for _, role := range allowedRoles {
				if claims.Role == role {
					next.ServeHTTP(w, r)
					return
				}
			}

			http.Error(w, fmt.Sprintf("Role %s does not have permission", claims.Role), http.StatusForbidden)
		})
	}
}
