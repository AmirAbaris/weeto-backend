package middleware

import (
	"crypto/subtle"
	"net/http"

	"github.com/AmirAbaris/weeto-backend/internal/handler/httputil"
)

func RequireAdminKey(adminKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if adminKey == "" {
				httputil.WriteError(w, http.StatusServiceUnavailable, "admin API not configured")
				return
			}

			token := bearerToken(r.Header.Get("Authorization"))
			if token == "" {
				httputil.WriteError(w, http.StatusUnauthorized, "missing or invalid authorization header")
				return
			}

			if subtle.ConstantTimeCompare([]byte(token), []byte(adminKey)) != 1 {
				httputil.WriteError(w, http.StatusUnauthorized, "invalid admin credentials")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func WithAdminKey(adminKey string, handler http.HandlerFunc) http.Handler {
	return RequireAdminKey(adminKey)(handler)
}
