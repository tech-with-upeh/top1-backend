package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DbUrl string
	Port  string

	RedisUrl string

	SMTPHost string
	SMTPPort int
	SMTPUser string
	SMTPPass string
	SMTPTLS  bool
	MailFrom string

	JWTSecret string
	AppUrl    string
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		slog.Info("No .env file found; using environment variables")
	}

	smtpPort, err := strconv.Atoi(getEnv("SMTP_PORT", "1025"))
	if err != nil {
		return nil, fmt.Errorf("invalid SMTP_PORT: %w", err)
	}

	smtpTLS, err := strconv.ParseBool(getEnv("SMTP_TLS", "false"))
	if err != nil {
		return nil, fmt.Errorf("invalid SMTP_TLS: %w", err)
	}

	config := &Config{
		DbUrl:     os.Getenv("DATABASE_URL"),
		Port:      getEnv("PORT", "8080"),
		SMTPHost:  getEnv("SMTP_HOST", "localhost"),
		SMTPPort:  smtpPort,
		SMTPUser:  os.Getenv("SMTP_USERNAME"),
		SMTPPass:  os.Getenv("SMTP_PASSWORD"),
		SMTPTLS:   smtpTLS,
		MailFrom:  getEnv("MAIL_FROM", "Top1 <noreply@top1.local>"),
		RedisUrl:  getEnv("RedisUrl", "redis://localhost:6379/0"),
		JWTSecret: getEnv("JWTSecret", "thisisasimplejwtsecret"),
		AppUrl:    getEnv("AppUrl", "http://localhost:3000"),
	}

	return config, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
