package tasks

import (
	"context"
	"encoding/json"
	"fmt"

	"top1/internal/repo"
	mailpkg "top1/internal/services/mail"

	"github.com/hibiken/asynq"
)

type EmailHandler struct {
	authRepo    *repo.AuthRepository
	mailService *mailpkg.MailService
}

func NewEmailHandler(
	authRepo *repo.AuthRepository,
	mailService *mailpkg.MailService,
) *EmailHandler {
	return &EmailHandler{
		authRepo:    authRepo,
		mailService: mailService,
	}
}

func (h *EmailHandler) HandleSendEmail(
	ctx context.Context,
	task *asynq.Task,
) error {
	var payload EmailPayload

	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("decode email payload: %w", err)
	}

	if payload.VerificationID == "" {
		return fmt.Errorf("missing verification ID")
	}

	idempotencyKey := payload.VerificationID

	claimToken, claimed, err := h.authRepo.ClaimEmailDelivery(
		ctx,
		idempotencyKey,
		"email_verification",
		payload.To,
	)
	if err != nil {
		return fmt.Errorf("claim email delivery: %w", err)
	}

	if !claimed {
		status, err := h.authRepo.GetEmailDeliveryStatus(ctx, idempotencyKey)
		if err != nil {
			return fmt.Errorf("get email delivery status: %w", err)
		}

		switch status {
		case "sent":
			// Already delivered. Acknowledge the duplicate task.
			return nil

		case "processing":
			// Another worker currently owns this delivery.
			// Return an error so Asynq retries later.
			return fmt.Errorf("email delivery %q is already processing", idempotencyKey)

		default:
			return fmt.Errorf(
				"email delivery %q could not be claimed; current status: %s",
				idempotencyKey,
				status,
			)
		}
	}

	if err := h.mailService.Send(
		ctx,
		payload.To,
		payload.Subject,
		payload.Body,
	); err != nil {
		_, releaseErr := h.authRepo.ReleaseEmailDeliveryClaim(
			ctx,
			idempotencyKey,
			claimToken,
			err.Error(),
		)

		if releaseErr != nil {
			return fmt.Errorf(
				"send email: %w; release claim: %v",
				err,
				releaseErr,
			)
		}

		return fmt.Errorf("send email: %w", err)
	}

	marked, err := h.authRepo.MarkEmailDeliverySent(
		ctx,
		idempotencyKey,
		claimToken,
	)
	if err != nil {
		return fmt.Errorf("mark email delivery as sent: %w", err)
	}

	if !marked {
		return fmt.Errorf("email delivery claim was lost before marking sent")
	}

	return nil
}
