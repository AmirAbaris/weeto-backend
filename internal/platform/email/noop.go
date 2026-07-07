package email

import (
	"context"
	"log/slog"
)

type NoopSender struct{}

func NewNoopSender() *NoopSender {
	return &NoopSender{}
}

func (s *NoopSender) Send(ctx context.Context, msg Message) error {
	slog.Info("email noop send",
		"to", msg.To,
		"subject", msg.Subject,
	)
	return nil
}
