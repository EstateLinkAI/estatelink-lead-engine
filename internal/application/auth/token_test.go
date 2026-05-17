package auth

import (
	"testing"
	"time"

	"github.com/nath070707/estatelink-lead-engine/internal/domain/user"
)

func TestTokenGenerateAndVerify(t *testing.T) {
	tokenService := NewTokenService("test-secret", time.Hour)

	u := &user.User{
		ID:    "user-123",
		Email: "admin@estatelink.dev",
		Role:  user.RoleAdmin,
	}

	token, err := tokenService.Generate(u)
	if err != nil {
		t.Fatalf("expected token generation to succeed, got %v", err)
	}

	claims, err := tokenService.Verify(token)
	if err != nil {
		t.Fatalf("expected token verification to succeed, got %v", err)
	}

	if claims.UserID != u.ID {
		t.Fatalf("expected user id %s, got %s", u.ID, claims.UserID)
	}

	if claims.Email != u.Email {
		t.Fatalf("expected email %s, got %s", u.Email, claims.Email)
	}

	if claims.Role != u.Role {
		t.Fatalf("expected role %s, got %s", u.Role, claims.Role)
	}
}

func TestTokenVerifyRejectsInvalidToken(t *testing.T) {
	tokenService := NewTokenService("test-secret", time.Hour)

	_, err := tokenService.Verify("invalid-token")

	if err == nil {
		t.Fatal("expected invalid token to fail")
	}
}