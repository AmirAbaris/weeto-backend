package google

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

const stateTTL = 10 * time.Minute

func signState(jwtSecret string, userID pgtype.UUID) (string, error) {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}

	exp := time.Now().Add(stateTTL).Unix()
	payload := fmt.Sprintf("%s.%d.%s", userID.String(), exp, hex.EncodeToString(nonce))
	mac := hmac.New(sha256.New, []byte(jwtSecret))
	mac.Write([]byte(payload))
	sig := hex.EncodeToString(mac.Sum(nil))

	return base64.RawURLEncoding.EncodeToString([]byte(payload + "." + sig)), nil
}

func verifyState(jwtSecret, state string) (pgtype.UUID, error) {
	raw, err := base64.RawURLEncoding.DecodeString(state)
	if err != nil {
		return pgtype.UUID{}, ErrInvalidOAuthState
	}

	parts := strings.Split(string(raw), ".")
	if len(parts) != 4 {
		return pgtype.UUID{}, ErrInvalidOAuthState
	}

	payload := strings.Join(parts[:3], ".")
	sig := parts[3]

	mac := hmac.New(sha256.New, []byte(jwtSecret))
	mac.Write([]byte(payload))
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return pgtype.UUID{}, ErrInvalidOAuthState
	}

	exp, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return pgtype.UUID{}, ErrInvalidOAuthState
	}
	if time.Now().Unix() > exp {
		return pgtype.UUID{}, ErrInvalidOAuthState
	}

	var userID pgtype.UUID
	if err := userID.Scan(parts[0]); err != nil {
		return pgtype.UUID{}, ErrInvalidOAuthState
	}

	return userID, nil
}
