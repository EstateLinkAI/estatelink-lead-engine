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
	UpdateRole(ctx context.Context, id string, role user.Role) error
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

	newUser, err := user.New(email, passwordHash, user.RoleViewer)
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
	AccessToken  string
	RefreshToken string
	User         *user.User
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

	tokens, err := s.tokens.GenerateTokenPair(existingUser)
	if err != nil {
		return nil, err
	}

	return &LoginOutput{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User:         existingUser,
	}, nil
}

type RefreshInput struct {
	RefreshToken string
}

func (s *Service) Refresh(ctx context.Context, input RefreshInput) (*LoginOutput, error) {
	if strings.TrimSpace(input.RefreshToken) == "" {
		return nil, errors.New("refresh token is required")
	}

	claims, err := s.tokens.VerifyRefreshToken(input.RefreshToken)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	existingUser, err := s.users.FindByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}

	if existingUser == nil {
		return nil, errors.New("user not found")
	}

	tokens, err := s.tokens.GenerateTokenPair(existingUser)
	if err != nil {
		return nil, err
	}

	return &LoginOutput{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User:         existingUser,
	}, nil
}

type UpdateUserRoleInput struct {
	UserID string
	Role   user.Role
}

func (s *Service) UpdateUserRole(ctx context.Context, input UpdateUserRoleInput) error {
	if strings.TrimSpace(input.UserID) == "" {
		return errors.New("user id is required")
	}

	if !input.Role.IsValid() {
		return errors.New("invalid role")
	}

	existingUser, err := s.users.FindByID(ctx, input.UserID)
	if err != nil {
		return err
	}

	if existingUser == nil {
		return errors.New("user not found")
	}

	return s.users.UpdateRole(ctx, input.UserID, input.Role)
}