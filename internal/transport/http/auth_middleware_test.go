package http

import (
	"context"
	nethttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/application/auth"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/user"
)

func TestAuthMiddlewareRejectsMissingToken(t *testing.T) {
	tokenService := auth.NewTokenService("test-secret", time.Hour)

	handler := AuthMiddleware(tokenService)(
		nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
			w.WriteHeader(nethttp.StatusOK)
		}),
	)

	req := httptest.NewRequest(nethttp.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAuthMiddlewareAllowsValidToken(t *testing.T) {
	tokenService := auth.NewTokenService("test-secret", time.Hour)

	token, err := tokenService.Generate(&user.User{
		ID:    "user-123",
		Email: "admin@estatelink.dev",
		Role:  user.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	handler := AuthMiddleware(tokenService)(
		nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
			currentUser, ok := GetCurrentUser(r)
			if !ok {
				t.Fatal("expected current user in context")
			}

			if currentUser.Email != "admin@estatelink.dev" {
				t.Fatalf("expected email admin@estatelink.dev, got %s", currentUser.Email)
			}

			w.WriteHeader(nethttp.StatusOK)
		}),
	)

	req := httptest.NewRequest(nethttp.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRequireRoleAllowsAdmin(t *testing.T) {
	handler := withCurrentUser(CurrentUser{
		ID:    "user-123",
		Email: "admin@estatelink.dev",
		Role:  user.RoleAdmin,
	}, RequireRole(user.RoleAdmin, user.RoleAnalyst)(
		nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
			w.WriteHeader(nethttp.StatusOK)
		}),
	))

	req := httptest.NewRequest(nethttp.MethodPost, "/api/listings", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRequireRoleRejectsViewer(t *testing.T) {
	handler := withCurrentUser(CurrentUser{
		ID:    "user-123",
		Email: "viewer@estatelink.dev",
		Role:  user.RoleViewer,
	}, RequireRole(user.RoleAdmin, user.RoleAnalyst)(
		nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
			w.WriteHeader(nethttp.StatusOK)
		}),
	))

	req := httptest.NewRequest(nethttp.MethodPost, "/api/listings", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func withCurrentUser(currentUser CurrentUser, next nethttp.Handler) nethttp.Handler {
	return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		ctx := r.Context()
		ctx = contextWithCurrentUser(ctx, currentUser)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func contextWithCurrentUser(ctx context.Context, currentUser CurrentUser) context.Context {
	return context.WithValue(ctx, currentUserKey, currentUser)
}
