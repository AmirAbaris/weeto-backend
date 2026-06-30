package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/AmirAbaris/weeto-backend/internal/handler/httputil"
	authsvc "github.com/AmirAbaris/weeto-backend/internal/service/auth"
	"github.com/jackc/pgx/v5/pgtype"
)

type contextKey string

const UserIDKey contextKey = "userID"

func RequireAuth(jwtSecret string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r.Header.Get("Authorization"))
		if token == "" {
			httputil.WriteError(w, http.StatusUnauthorized, "missing or invalid authorization header")
			return
		}

		userID, err := authsvc.ParseAccessToken(token, jwtSecret)
		if err != nil {
			httputil.WriteError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserIDFromContext(ctx context.Context) (pgtype.UUID, bool) {
	id, ok := ctx.Value(UserIDKey).(pgtype.UUID)
	return id, ok && id.Valid
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(header[len(prefix):])
}
