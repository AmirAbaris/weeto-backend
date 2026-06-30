package auth

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func uuidToString(id pgtype.UUID) string {
	u, _ := uuid.FromBytes(id.Bytes[:])
	return u.String()
}

func stringToUUID(s string) (pgtype.UUID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return pgtype.UUID{}, ErrInvalidToken
	}
	var out pgtype.UUID
	copy(out.Bytes[:], u[:])
	out.Valid = true
	return out, nil
}
