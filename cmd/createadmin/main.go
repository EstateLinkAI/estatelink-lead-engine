package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/user"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/infrastructure/postgres"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/application/auth"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	adminEmail := os.Getenv("ADMIN_EMAIL")
	adminPassword := os.Getenv("ADMIN_PASSWORD")

	if databaseURL == "" || adminEmail == "" || adminPassword == "" {
		log.Fatal("DATABASE_URL, ADMIN_EMAIL, and ADMIN_PASSWORD environment variables are required")
	}

	ctx := context.Background()

	db, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	userRepo := postgres.NewUserRepository(db)
	hasher := auth.NewPasswordHasher()

	// check if user exists
	existingUser, err := userRepo.FindByEmail(ctx, adminEmail)
	if err != nil {
		log.Fatalf("failed to query user: %v", err)
	}

	passwordHash, err := hasher.Hash(adminPassword)
	if err != nil {
		log.Fatalf("failed to hash password: %v", err)
	}

	if existingUser != nil {
		fmt.Println("User exists, promoting to admin...")
		if err := userRepo.UpdateRole(ctx, existingUser.ID, user.RoleAdmin); err != nil {
			log.Fatalf("failed to update role: %v", err)
		}
		fmt.Println("User promoted to admin successfully")
	} else {
		newUser := &user.User{
			Email:        adminEmail,
			PasswordHash: passwordHash,
			Role:         user.RoleAdmin,
		}

		if err := userRepo.Create(ctx, newUser); err != nil {
			log.Fatalf("failed to create user: %v", err)
		}

		fmt.Println("Admin user created successfully:", newUser.Email)
	}
}