package tasks

import (
	"encoding/json"

	"github.com/hibiken/asynq"
)

func NewEmailTask(
	to, subject, body string,
	isHTML bool,
) (*asynq.Task, error) {
	payload, err := json.Marshal(EmailPayload{
		To:      to,
		Subject: subject,
		Body:    body,
		IsHTML:  isHTML,
	})
	if err != nil {
		return nil, err
	}

	return asynq.NewTask(TypeSendEmail, payload), nil
}
