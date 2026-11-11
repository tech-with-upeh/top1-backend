package mail

import (
	"context"
	"fmt"
	"time"

	mail "github.com/wneessen/go-mail"
)

type MailConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	TLS      bool
}

type MailService struct {
	client *mail.Client
	from   string
}

func NewMailService(cfg MailConfig) (*MailService, error) {
	opts := []mail.Option{
		mail.WithPort(cfg.Port),
		mail.WithTimeout(10 * time.Second),
	}

	if cfg.TLS {
		opts = append(opts, mail.WithTLSPolicy(mail.TLSMandatory))
	} else {
		opts = append(opts, mail.WithTLSPolicy(mail.NoTLS))
	}

	if cfg.Username != "" && cfg.Password != "" {
		opts = append(opts,
			mail.WithSMTPAuth(mail.SMTPAuthAutoDiscover),
			mail.WithUsername(cfg.Username),
			mail.WithPassword(cfg.Password),
		)
	} else if cfg.Username != "" || cfg.Password != "" {
		return nil, fmt.Errorf("SMTP username and password must both be provided")
	}

	client, err := mail.NewClient(cfg.Host, opts...)
	if err != nil {
		return nil, fmt.Errorf("create SMTP client: %w", err)
	}

	return &MailService{
		client: client,
		from:   cfg.From,
	}, nil
}

func (s *MailService) Send(
	ctx context.Context,
	to string,
	subject string,
	html string,
) error {
	msg := mail.NewMsg()

	if err := msg.From(s.from); err != nil {
		return err
	}

	if err := msg.To(to); err != nil {
		return err
	}

	msg.Subject(subject)
	msg.SetBodyString(mail.TypeTextHTML, html)

	if err := s.client.DialAndSendWithContext(ctx, msg); err != nil {
		return fmt.Errorf("send email: %w", err)
	}

	return nil
}
