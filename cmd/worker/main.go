package main

import (
	"log/slog"

	"top1/internal/config"
	"top1/internal/db"
	"top1/internal/repo"
	"top1/internal/services/mail"
	"top1/internal/tasks"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		return
	}

	var pool *pgxpool.Pool
	pool, err = db.Connect(cfg.DbUrl)
	if err != nil {
		slog.Error("Failed to connect to db", "error", err)
		return
	}
	defer pool.Close()

	mailCfg := mail.MailConfig{
		Host:     cfg.SMTPHost,
		Port:     cfg.SMTPPort,
		Username: cfg.SMTPUser,
		Password: cfg.SMTPPass,
		TLS:      cfg.SMTPTLS,
		From:     cfg.MailFrom,
	}
	mailService, err := mail.NewMailService(mailCfg)
	if err != nil {
		slog.Error("Failed to setup Mail Service", "error", err)
		return
	}
	authRepo := repo.NewAuthRepository(pool)

	mux := asynq.NewServeMux()
	emailHandler := tasks.NewEmailHandler(authRepo, mailService)
	mux.HandleFunc(tasks.TypeSendEmail, emailHandler.HandleSendEmail)

	redisOpt, err := asynq.ParseRedisURI(cfg.RedisUrl)
	if err != nil {
		slog.Error("Failed to parse Redis URL", "error", err)
		return
	}

	server := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Queues: map[string]int{
				tasks.QueueEmails: 1,
				"default":         1,
			},
		},
	)

	if err := server.Run(mux); err != nil {
		slog.Error("Worker stopped", "error", err)
	}
}
