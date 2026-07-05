package google

import (
	"context"
	"errors"
	"fmt"

	"github.com/AmirAbaris/weeto-backend/internal/config"
	"github.com/AmirAbaris/weeto-backend/internal/db"
	"github.com/AmirAbaris/weeto-backend/internal/platform/crypto"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/oauth2"
	googleoauth "golang.org/x/oauth2/google"
)

type tokenManager struct {
	cfg    *config.Config
	q      *db.Queries
	oauth2 *oauth2.Config
}

func newTokenManager(cfg *config.Config, q *db.Queries) *tokenManager {
	return &tokenManager{
		cfg: cfg,
		q:   q,
		oauth2: &oauth2.Config{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			RedirectURL:  cfg.GoogleRedirectURL,
			Scopes:       []string{scopeCalendarEvents, scopeUserEmail},
			Endpoint:     googleoauth.Endpoint,
		},
	}
}

func (m *tokenManager) oauth2TokenSource(ctx context.Context, ownerID pgtype.UUID) (oauth2.TokenSource, error) {
	creds, err := m.q.GetUserGoogleCredentials(ctx, ownerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotConnected
		}
		return nil, err
	}
	if !creds.GoogleConnectedAt.Valid || !creds.GoogleRefreshToken.Valid || creds.GoogleRefreshToken.String == "" {
		return nil, ErrNotConnected
	}
	if len(m.cfg.TokenEncryptionKey) != 32 {
		return nil, fmt.Errorf("TOKEN_ENCRYPTION_KEY not configured")
	}

	refreshToken, err := crypto.Decrypt(creds.GoogleRefreshToken.String, m.cfg.TokenEncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt refresh token: %w", err)
	}

	token := &oauth2.Token{RefreshToken: string(refreshToken)}
	return m.oauth2.TokenSource(ctx, token), nil
}

func (m *tokenManager) accessToken(ctx context.Context, ownerID pgtype.UUID) (string, error) {
	src, err := m.oauth2TokenSource(ctx, ownerID)
	if err != nil {
		return "", err
	}

	tok, err := src.Token()
	if err != nil {
		return "", fmt.Errorf("refresh access token: %w", err)
	}
	return tok.AccessToken, nil
}
