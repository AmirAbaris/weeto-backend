package auth

import (
	"context"
	"time"

	"github.com/AmirAbaris/weeto-backend/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

// AuthTokens is the access + refresh token bundle returned on login/register/refresh.
type AuthTokens struct {
	AccessToken       string
	RefreshToken      string
	ExpiresIn         int // access token lifetime in seconds (OAuth expires_in)
	RefreshExpiresIn  int // refresh token lifetime in seconds (cookie Max-Age)
}

func IssueAuthTokens(
	ctx context.Context,
	q *db.Queries,
	userID pgtype.UUID,
	secret string,
	accessTTL, refreshTTL time.Duration,
) (AuthTokens, error) {
	access, err := IssueAccessToken(userID, secret, accessTTL)
	if err != nil {
		return AuthTokens{}, err
	}

	refresh, err := CreateRefreshToken(ctx, q, userID, refreshTTL)
	if err != nil {
		return AuthTokens{}, err
	}

	return AuthTokens{
		AccessToken:      access,
		RefreshToken:     refresh,
		ExpiresIn:        int(accessTTL.Seconds()),
		RefreshExpiresIn: int(refreshTTL.Seconds()),
	}, nil
}
