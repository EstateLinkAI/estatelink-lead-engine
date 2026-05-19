package auth

import (
	"context"
	"errors"
	"strings"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/user"
)

type UserRepository interface {
	Create(ctx context.Context, u *user.User) error
	FindByEmail(ctx context.Context, email string) (*user.User, error)
	FindByID(ctx context.Context, id string) (*user.User, error)
}

type Service struct {
	users  UserRepository
	hasher *PasswordHasher
	tokens *TokenService
}

func NewService(users UserRepository, hasher *PasswordHasher, tokens *TokenService) *Service {
	return &Service{
		users:  users,
		hasher: hasher,
		tokens: tokens,
	}
}

type RegisterInput struct {
	Email    string
	Password string
	Role     user.Role
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (*user.User, error) {
	email := strings.TrimSpace(strings.ToLower(input.Email))

	if email == "" {
		return nil, errors.New("email is required")
	}

	if input.Password == "" {
		return nil, errors.New("password is required")
	}

	if !input.Role.IsValid() {
		return nil, errors.New("invalid role")
	}

	existingUser, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	if existingUser != nil {
		return nil, errors.New("user already exists")
	}

	passwordHash, err := s.hasher.Hash(input.Password)
	if err != nil {
		return nil, err
	}

	newUser, err := user.New(email, passwordHash, input.Role)
	if err != nil {
		return nil, err
	}

	if err := s.users.Create(ctx, newUser); err != nil {
		return nil, err
	}

	return newUser, nil
}

type LoginInput struct {
	Email    string
	Password string
}

type LoginOutput struct {
	Token string
	User  *user.User
}

func (s *Service) Login(ctx context.Context, input LoginInput) (*LoginOutput, error) {
	email := strings.TrimSpace(strings.ToLower(input.Email))

	if email == "" {
		return nil, errors.New("email is required")
	}

	if input.Password == "" {
		return nil, errors.New("password is required")
	}

	existingUser, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	if existingUser == nil {
		return nil, errors.New("invalid email or password")
	}

	if err := s.hasher.Compare(input.Password, existingUser.PasswordHash); err != nil {
		return nil, errors.New("invalid email or password")
	}

	token, err := s.tokens.Generate(existingUser)
	if err != nil {
		return nil, err
	}

	return &LoginOutput{
		Token: token,
		User:  existingUser,
	}, nil
}
