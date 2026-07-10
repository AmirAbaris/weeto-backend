package email

import (
	"context"
	"fmt"

	"github.com/resend/resend-go/v2"
)

type ResendSender struct {
	client *resend.Client
	from   string
}

func NewResendSender(apiKey, from string) (*ResendSender, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("resend api key is required")
	}
	if from == "" {
		return nil, fmt.Errorf("resend from address is required")
	}
	return &ResendSender{
		client: resend.NewClient(apiKey),
		from:   from,
	}, nil
}

func (s *ResendSender) Send(ctx context.Context, msg Message) error {
	_, err := s.client.Emails.SendWithContext(ctx, &resend.SendEmailRequest{
		From:    s.from,
		To:      []string{msg.To},
		Subject: msg.Subject,
		Html:    msg.HTML,
	})
	return err
}
