package config

import (
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DbUrl string
	Port  string
}

func Load() (*Config, error) {
	if godotenv.Load() != nil {
		slog.Error("Error: .env file not found, using environment variables")
	}

	var config *Config = &Config{
		DbUrl: os.Getenv("DATABASE_URL"),
		Port:  os.Getenv("PORT"),
	}
	return config, nil
}
