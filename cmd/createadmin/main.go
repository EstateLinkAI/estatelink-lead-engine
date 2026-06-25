package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/application/auth"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/config"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/user"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/infrastructure/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	config.LoadEnv()

	databaseURL := mustGetEnv("DATABASE_URL")
	adminEmail := mustGetEnv("ADMIN_EMAIL")
	adminPassword := mustGetEnv("ADMIN_PASSWORD")

	ctx := context.Background()

	db, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	userRepo := postgres.NewUserRepository(db)
	hasher := auth.NewPasswordHasher()

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
		return
	}

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

func mustGetEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		log.Fatalf("%s environment variable is required", name)
	}

	return value

}
