package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/nath070707/estatelink-lead-engine/internal/application/auth"
	"github.com/nath070707/estatelink-lead-engine/internal/domain/user"
)

type contextKey string

const currentUserKey contextKey = "currentUser"

type CurrentUser struct {
	ID    string
	Email string
	Role  user.Role
}

func AuthMiddleware(tokenService *auth.TokenService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				http.Error(w, "missing authorization header", http.StatusUnauthorized)
				return
			}

			tokenString := strings.TrimPrefix(header, "Bearer ")
			if tokenString == header {
				http.Error(w, "invalid authorization header", http.StatusUnauthorized)
				return
			}

			claims, err := tokenService.Verify(tokenString)
			if err != nil {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			currentUser := CurrentUser{
				ID:    claims.UserID,
				Email: claims.Email,
				Role:  claims.Role,
			}

			ctx := context.WithValue(r.Context(), currentUserKey, currentUser)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetCurrentUser(r *http.Request) (CurrentUser, bool) {
	currentUser, ok := r.Context().Value(currentUserKey).(CurrentUser)
	return currentUser, ok
}