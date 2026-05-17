package http

import (
	"net/http"

	"github.com/nath070707/estatelink-lead-engine/internal/domain/user"
)

func RequireRole(allowedRoles ...user.Role) func(http.Handler) http.Handler {
	allowed := make(map[user.Role]bool)

	for _, role := range allowedRoles {
		allowed[role] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			currentUser, ok := GetCurrentUser(r)
			if !ok {
				http.Error(w, "not authenticated", http.StatusUnauthorized)
				return
			}

			if !allowed[currentUser.Role] {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}