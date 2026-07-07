package sms

import (
	"context"

	"github.com/AmirAbaris/weeto-backend/internal/config"
	"github.com/AmirAbaris/weeto-backend/internal/platform/sms/smsir"
)

func NewSender(cfg *config.Config) Sender {
	if cfg.SMSAPIKey == "" {
		return &NoopSender{}
	}
	return &smsirAdapter{smsir.NewClient(cfg.SMSBaseURL, cfg.SMSAPIKey)}
}

type smsirAdapter struct {
	client *smsir.Client
}

func (a *smsirAdapter) VerifySend(ctx context.Context, mobile string, templateID int, params []Parameter) (int64, float64, error) {
	smsirParams := make([]smsir.Parameter, len(params))
	for i, p := range params {
		smsirParams[i] = smsir.Parameter{Name: p.Name, Value: p.Value}
	}
	return a.client.VerifySend(ctx, mobile, templateID, smsirParams)
}
