package main

import (
	"log/slog"

	"top1/internal/config"
	"top1/internal/services/mail"
	"top1/internal/tasks"

	"github.com/hibiken/asynq"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		return
	}

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

	mux := asynq.NewServeMux()
	emailHandler := tasks.NewEmailHandler(mailService)
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
