package auth

import (
	"time"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/user"
	"github.com/golang-jwt/jwt/v5"
)

type TokenService struct {
	secret []byte
	ttl    time.Duration
}

func NewTokenService(secret string, ttl time.Duration) *TokenService {
	return &TokenService{
		secret: []byte(secret),
		ttl:    ttl,
	}
}

type Claims struct {
	UserID string    `json:"user_id"`
	Email  string    `json:"email"`
	Role   user.Role `json:"role"`
	jwt.RegisteredClaims
}

func (s *TokenService) Generate(u *user.User) (string, error) {
	now := time.Now()

	claims := Claims{
		UserID: u.ID,
		Email:  u.Email,
		Role:   u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(s.secret)
}
