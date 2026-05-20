package auth

import (
	"time"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/user"
	"github.com/golang-jwt/jwt/v5"
)

type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

type TokenService struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewTokenService(secret string, accessTTL, refreshTTL time.Duration) *TokenService {
	return &TokenService{
		secret:     []byte(secret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

type Claims struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	Role      user.Role `json:"role"`
	TokenType TokenType `json:"token_type"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

func (s *TokenService) GenerateAccessToken(u *user.User) (string, error) {
	return s.generate(u, TokenTypeAccess, s.accessTTL)
}

func (s *TokenService) GenerateRefreshToken(u *user.User) (string, error) {
	return s.generate(u, TokenTypeRefresh, s.refreshTTL)
}

func (s *TokenService) GenerateTokenPair(u *user.User) (*TokenPair, error) {
	accessToken, err := s.GenerateAccessToken(u)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.GenerateRefreshToken(u)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *TokenService) generate(u *user.User, tokenType TokenType, ttl time.Duration) (string, error) {
	now := time.Now()

	claims := Claims{
		UserID:    u.ID,
		Email:     u.Email,
		Role:      u.Role,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(s.secret)
}
