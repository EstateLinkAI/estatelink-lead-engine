package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func LoadEnv() {
	envFile := os.Getenv("ENV_FILE")

	if envFile != "" {
		if err := godotenv.Load(envFile); err != nil {
			log.Printf("Could not load %s, using system environment variables", envFile)
		}
		return
	}

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}
}
