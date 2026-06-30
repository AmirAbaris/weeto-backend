package auth

import "github.com/jackc/pgx/v5/pgtype"

func uuidToString(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}
	return id.String()
}

func stringToUUID(s string) (pgtype.UUID, error) {
	var id pgtype.UUID
	if err := id.Scan(s); err != nil {
		return pgtype.UUID{}, ErrInvalidToken
	}
	return id, nil
}
