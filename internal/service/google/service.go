package google

import (
	"context"
	"errors"
	"fmt"

	"github.com/AmirAbaris/weeto-backend/internal/config"
	"github.com/AmirAbaris/weeto-backend/internal/db"
	googleplatform "github.com/AmirAbaris/weeto-backend/internal/platform/google"
	"github.com/AmirAbaris/weeto-backend/internal/platform/crypto"
	"github.com/jackc/pgx/v5/pgtype"
)

type Service struct {
	cfg   *config.Config
	q     *db.Queries
	oauth *googleplatform.OAuth
}

func NewService(cfg *config.Config, q *db.Queries) *Service {
	return &Service{
		cfg:   cfg,
		q:     q,
		oauth: googleplatform.NewOAuth(cfg),
	}
}

func (s *Service) ConnectURL(ctx context.Context, userID pgtype.UUID) (string, error) {
	if err := s.requireConfigured(); err != nil {
		return "", err
	}
	if !userID.Valid {
		return "", errors.New("invalid user")
	}

	state, err := signState(s.cfg.JWTSecret, userID)
	if err != nil {
		return "", err
	}

	return s.oauth.AuthCodeURL(state), nil
}

func (s *Service) HandleCallback(ctx context.Context, code, state string) (pgtype.UUID, error) {
	if err := s.requireConfigured(); err != nil {
		return pgtype.UUID{}, err
	}
	if code == "" {
		return pgtype.UUID{}, fmt.Errorf("missing authorization code")
	}

	userID, err := verifyState(s.cfg.JWTSecret, state)
	if err != nil {
		return pgtype.UUID{}, err
	}

	result, err := s.oauth.Exchange(ctx, code)
	if err != nil {
		return pgtype.UUID{}, err
	}

	encrypted, err := crypto.Encrypt([]byte(result.RefreshToken), s.cfg.TokenEncryptionKey)
	if err != nil {
		return pgtype.UUID{}, err
	}

	if err := s.q.SetUserGoogleConnection(ctx, db.SetUserGoogleConnectionParams{
		ID:                  userID,
		GoogleID:            pgtype.Text{String: result.GoogleID, Valid: true},
		GoogleRefreshToken:  pgtype.Text{String: encrypted, Valid: true},
	}); err != nil {
		return pgtype.UUID{}, err
	}

	return userID, nil
}

func (s *Service) Disconnect(ctx context.Context, userID pgtype.UUID) error {
	if !userID.Valid {
		return errors.New("invalid user")
	}
	return s.q.ClearUserGoogleConnection(ctx, userID)
}

func (s *Service) IsConnected(ctx context.Context, userID pgtype.UUID) (bool, error) {
	if !userID.Valid {
		return false, errors.New("invalid user")
	}
	return s.q.IsGoogleConnected(ctx, userID)
}

func (s *Service) requireConfigured() error {
	if s.cfg.GoogleClientID == "" || s.cfg.GoogleClientSecret == "" || s.cfg.GoogleRedirectURL == "" {
		return ErrGoogleNotConfigured
	}
	if len(s.cfg.TokenEncryptionKey) != 32 {
		return fmt.Errorf("TOKEN_ENCRYPTION_KEY must be set to 32 bytes (base64)")
	}
	return nil
}
