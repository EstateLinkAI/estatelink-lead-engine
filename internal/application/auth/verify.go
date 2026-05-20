package auth

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidTokenType = errors.New("invalid token type")

func (s *TokenService) VerifyAccessToken(tokenString string) (*Claims, error) {
	return s.verify(tokenString, TokenTypeAccess)
}

func (s *TokenService) VerifyRefreshToken(tokenString string) (*Claims, error) {
	return s.verify(tokenString, TokenTypeRefresh)
}

func (s *TokenService) verify(tokenString string, expectedType TokenType) (*Claims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwt.Token) (interface{}, error) {
			return s.secret, nil
		},
	)

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}

	if claims.TokenType != expectedType {
		return nil, ErrInvalidTokenType
	}

	return claims, nil
}
