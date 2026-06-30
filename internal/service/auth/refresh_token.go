package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"time"

	"github.com/AmirAbaris/weeto-backend/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

func generateRefreshToken() (raw string, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}

	raw = base64.RawURLEncoding.EncodeToString(b)
	hash = hashToken(raw)

	return raw, hash, nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func CreateRefreshToken(
	ctx context.Context,
	q *db.Queries,
	userID pgtype.UUID,
	ttl time.Duration,
) (raw string, err error) {
	raw, tokenHash, err := generateRefreshToken()
	if err != nil {
		return "", err
	}
	expiresAt := pgtype.Timestamptz{
		Time:  time.Now().Add(ttl),
		Valid: true,
	}
	_, err = q.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return "", err
	}
	return raw, nil
}

func RotateRefreshToken(
	ctx context.Context,
	q *db.Queries,
	rawToken string,
	ttl time.Duration,
) (newRaw string, userID pgtype.UUID, err error) {
	row, err := q.GetRefreshTokenByHash(ctx, hashToken(rawToken))
	if err != nil {
		return "", pgtype.UUID{}, ErrInvalidToken
	}

	newRaw, err = CreateRefreshToken(ctx, q, row.UserID, ttl)
	if err != nil {
		return "", pgtype.UUID{}, err
	}

	if err := q.RevokeRefreshToken(ctx, row.ID); err != nil {
		return "", pgtype.UUID{}, err
	}

	return newRaw, row.UserID, nil
}

func RevokeRefreshTokenByRaw(ctx context.Context, q *db.Queries, rawToken string) error {
	row, err := q.GetRefreshTokenByHash(ctx, hashToken(rawToken))
	if err != nil {
		return ErrInvalidToken
	}
	return q.RevokeRefreshToken(ctx, row.ID)
}

func RevokeAllForUser(ctx context.Context, q *db.Queries, userID pgtype.UUID) error {
	return q.RevokeAllUserRefreshTokens(ctx, userID)
}
