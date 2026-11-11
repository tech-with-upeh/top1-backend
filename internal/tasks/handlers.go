package tasks

import (
	"context"
	"encoding/json"

	mailpkg "top1/internal/services/mail"

	"github.com/hibiken/asynq"
)

type EmailHandler struct {
	mailService *mailpkg.MailService
}

func NewEmailHandler(mailService *mailpkg.MailService) *EmailHandler {
	return &EmailHandler{
		mailService: mailService,
	}
}

func (h *EmailHandler) HandleSendEmail(
	ctx context.Context,
	task *asynq.Task,
) error {
	var payload EmailPayload

	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return err
	}

	return h.mailService.Send(
		ctx,
		payload.To,
		payload.Subject,
		payload.Body,
	)
}
