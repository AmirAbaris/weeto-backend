package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// accessTokenClaims is the JWT payload for short-lived access tokens.
type accessTokenClaims struct {
	jwt.RegisteredClaims
}

func IssueAccessToken(userID pgtype.UUID, secret string, ttl time.Duration) (string, error) {
	if !userID.Valid {
		return "", fmt.Errorf("invalid user id")
	}

	now := time.Now()
	claims := accessTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   uuidToString(userID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ParseAccessToken(tokenStr, secret string) (pgtype.UUID, error) {
	claims := &accessTokenClaims{}

	token, err := jwt.ParseWithClaims(
		tokenStr,
		claims,
		func(t *jwt.Token) (any, error) {
			if t.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(secret), nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil || !token.Valid {
		return pgtype.UUID{}, ErrInvalidToken
	}
	if claims.Subject == "" {
		return pgtype.UUID{}, ErrInvalidToken
	}

	return stringToUUID(claims.Subject)
}
