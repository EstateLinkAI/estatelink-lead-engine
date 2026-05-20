package auth

import (
	"testing"
	"time"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/user"
)

func TestTokenGenerateAndVerify(t *testing.T) {
	tokenService := NewTokenService("test-secret", time.Hour, 24*time.Hour)

	u := &user.User{
		ID:    "user-123",
		Email: "admin@estatelink.dev",
		Role:  user.RoleAdmin,
	}

	token, err := tokenService.GenerateAccessToken(u)
	if err != nil {
		t.Fatalf("expected token generation to succeed, got %v", err)
	}

	claims, err := tokenService.VerifyAccessToken(token)
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

	if claims.TokenType != TokenTypeAccess {
		t.Fatalf("expected token type %s, got %s", TokenTypeAccess, claims.TokenType)
	}
}

func TestTokenVerifyRejectsInvalidToken(t *testing.T) {
	tokenService := NewTokenService("test-secret", time.Hour, 24*time.Hour)

	_, err := tokenService.VerifyAccessToken("invalid-token")

	if err == nil {
		t.Fatal("expected invalid token to fail")
	}
}

func TestAccessVerificationRejectsRefreshToken(t *testing.T) {
	tokenService := NewTokenService("test-secret", time.Hour, 24*time.Hour)

	u := &user.User{
		ID:    "user-123",
		Email: "admin@estatelink.dev",
		Role:  user.RoleAdmin,
	}

	refreshToken, err := tokenService.GenerateRefreshToken(u)
	if err != nil {
		t.Fatalf("expected refresh token generation to succeed, got %v", err)
	}

	_, err = tokenService.VerifyAccessToken(refreshToken)
	if err == nil {
		t.Fatal("expected refresh token to be rejected for access verification")
	}
}
