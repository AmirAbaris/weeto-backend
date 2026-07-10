package email

import (
	"fmt"

	"github.com/AmirAbaris/weeto-backend/internal/config"
)

func NewSender(cfg *config.Config) (Sender, error) {
	if cfg.ResendAPIKey != "" {
		return NewResendSender(cfg.ResendAPIKey, cfg.ResendFrom)
	}
	if cfg.Env == "development" {
		return NewNoopSender(), nil
	}
	return nil, fmt.Errorf("RESEND_TOKEN_KEY is required in non-development environments")
}
