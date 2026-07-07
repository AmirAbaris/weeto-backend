package sms

import "context"

type NoopSender struct{}

func (n *NoopSender) VerifySend(ctx context.Context, mobile string, templateID int, params []Parameter) (int64, float64, error) {
	return 0, 0, nil
}
