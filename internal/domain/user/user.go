package user

import (
	"errors"
	"time"
)

type Role string

const (
	RoleAdmin   Role = "admin"
	RoleAnalyst Role = "analyst"
	RoleViewer  Role = "viewer"
)

type User struct {
	ID           string
	Email        string
	PasswordHash string
	Role         Role
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func New(email string, passwordHash string, role Role) (*User, error) {
	if email == "" {
		return nil, errors.New("email is required")
	}

	if passwordHash == "" {
		return nil, errors.New("password hash is required")
	}

	if !role.IsValid() {
		return nil, errors.New("invalid role")
	}

	return &User{
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
	}, nil
}

func (r Role) IsValid() bool {
	switch r {
	case RoleAdmin, RoleAnalyst, RoleViewer:
		return true
	default:
		return false
	}
}