package auth

import (
	"context"
	"testing"
	"time"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/user"
)

type fakeUserRepo struct {
	usersByEmail map[string]*user.User
	usersByID    map[string]*user.User
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		usersByEmail: map[string]*user.User{},
		usersByID:    map[string]*user.User{},
	}
}

func (r *fakeUserRepo) Create(ctx context.Context, u *user.User) error {
	u.ID = "user-123"
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()

	r.usersByEmail[u.Email] = u
	r.usersByID[u.ID] = u

	return nil
}

func (r *fakeUserRepo) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	return r.usersByEmail[email], nil
}

func (r *fakeUserRepo) FindByID(ctx context.Context, id string) (*user.User, error) {
	return r.usersByID[id], nil
}

func TestRegisterCreatesUser(t *testing.T) {
	repo := newFakeUserRepo()
	hasher := NewPasswordHasher()
	tokens := NewTokenService("test-secret", time.Hour)

	service := NewService(repo, hasher, tokens)

	createdUser, err := service.Register(context.Background(), RegisterInput{
		Email:    "ADMIN@EstateLink.Dev",
		Password: "Password123!",
		Role:     user.RoleAdmin,
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if createdUser.Email != "admin@estatelink.dev" {
		t.Fatalf("expected normalized email, got %s", createdUser.Email)
	}

	if createdUser.PasswordHash == "Password123!" {
		t.Fatal("expected password to be hashed")
	}

	if createdUser.Role != user.RoleAdmin {
		t.Fatalf("expected admin role, got %s", createdUser.Role)
	}
}

func TestRegisterRejectsDuplicateEmail(t *testing.T) {
	repo := newFakeUserRepo()
	hasher := NewPasswordHasher()
	tokens := NewTokenService("test-secret", time.Hour)

	service := NewService(repo, hasher, tokens)

	input := RegisterInput{
		Email:    "admin@estatelink.dev",
		Password: "Password123!",
		Role:     user.RoleAdmin,
	}

	if _, err := service.Register(context.Background(), input); err != nil {
		t.Fatalf("expected first register to succeed, got %v", err)
	}

	if _, err := service.Register(context.Background(), input); err == nil {
		t.Fatal("expected duplicate register to fail")
	}
}

func TestLoginReturnsToken(t *testing.T) {
	repo := newFakeUserRepo()
	hasher := NewPasswordHasher()
	tokens := NewTokenService("test-secret", time.Hour)

	service := NewService(repo, hasher, tokens)

	_, err := service.Register(context.Background(), RegisterInput{
		Email:    "admin@estatelink.dev",
		Password: "Password123!",
		Role:     user.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	result, err := service.Login(context.Background(), LoginInput{
		Email:    "admin@estatelink.dev",
		Password: "Password123!",
	})
	if err != nil {
		t.Fatalf("expected login to succeed, got %v", err)
	}

	if result.Token == "" {
		t.Fatal("expected token")
	}

	if result.User.Email != "admin@estatelink.dev" {
		t.Fatalf("expected user email, got %s", result.User.Email)
	}
}

func TestLoginRejectsBadPassword(t *testing.T) {
	repo := newFakeUserRepo()
	hasher := NewPasswordHasher()
	tokens := NewTokenService("test-secret", time.Hour)

	service := NewService(repo, hasher, tokens)

	_, err := service.Register(context.Background(), RegisterInput{
		Email:    "admin@estatelink.dev",
		Password: "Password123!",
		Role:     user.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	_, err = service.Login(context.Background(), LoginInput{
		Email:    "admin@estatelink.dev",
		Password: "wrong-password",
	})

	if err == nil {
		t.Fatal("expected bad password to fail")
	}
}
