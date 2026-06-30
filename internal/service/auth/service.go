package auth

import (
	"context"
	"errors"

	"github.com/AmirAbaris/weeto-backend/internal/config"
	"github.com/AmirAbaris/weeto-backend/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Service struct {
	q   *db.Queries
	cfg *config.Config
}

func NewService(q *db.Queries, cfg *config.Config) *Service {
	return &Service{q: q, cfg: cfg}
}

func (s *Service) Register(ctx context.Context, email, password string) (AuthTokens, error) {
	email, err := validateCredentials(email, password)
	if err != nil {
		return AuthTokens{}, err
	}

	if _, err := s.q.GetUserByEmail(ctx, email); err == nil {
		return AuthTokens{}, ErrEmailAlreadyExists
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return AuthTokens{}, err
	}

	hash, err := HashPassword(password)
	if err != nil {
		return AuthTokens{}, err
	}

	user, err := s.q.CreateUser(ctx, db.CreateUserParams{
		Email:        email,
		PasswordHash: hash,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return AuthTokens{}, ErrEmailAlreadyExists
		}
		return AuthTokens{}, err
	}

	return IssueAuthTokens(ctx, s.q, user.ID, s.cfg.JWTSecret, s.cfg.JWTAccessTTL, s.cfg.JWTRefreshTTL)
}

func (s *Service) Login(ctx context.Context, email, password string) (AuthTokens, error) {
	email, err := validateCredentials(email, password)
	if err != nil {
		if errors.Is(err, ErrWeakPassword) || errors.Is(err, ErrInvalidEmail) {
			return AuthTokens{}, ErrInvalidCredentials
		}
		return AuthTokens{}, err
	}

	user, err := s.q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AuthTokens{}, ErrInvalidCredentials
		}
		return AuthTokens{}, err
	}

	if err := VerifyPassword(password, user.PasswordHash); err != nil {
		return AuthTokens{}, ErrInvalidCredentials
	}

	if err := s.q.TouchLastLogin(ctx, user.ID); err != nil {
		return AuthTokens{}, err
	}

	return IssueAuthTokens(ctx, s.q, user.ID, s.cfg.JWTSecret, s.cfg.JWTAccessTTL, s.cfg.JWTRefreshTTL)
}

func (s *Service) Refresh(ctx context.Context, rawRefreshToken string) (AuthTokens, error) {
	if rawRefreshToken == "" {
		return AuthTokens{}, ErrInvalidToken
	}

	newRaw, userID, err := RotateRefreshToken(ctx, s.q, rawRefreshToken, s.cfg.JWTRefreshTTL)
	if err != nil {
		return AuthTokens{}, err
	}

	access, err := IssueAccessToken(userID, s.cfg.JWTSecret, s.cfg.JWTAccessTTL)
	if err != nil {
		return AuthTokens{}, err
	}

	return AuthTokens{
		AccessToken:  access,
		RefreshToken: newRaw,
		ExpiresIn:    int(s.cfg.JWTAccessTTL.Seconds()),
	}, nil
}

func (s *Service) Logout(ctx context.Context, rawRefreshToken string) error {
	if rawRefreshToken == "" {
		return ErrInvalidToken
	}
	return RevokeRefreshTokenByRaw(ctx, s.q, rawRefreshToken)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
