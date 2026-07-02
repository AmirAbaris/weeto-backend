package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authsvc "github.com/AmirAbaris/weeto-backend/internal/service/auth"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestRequireAuth(t *testing.T) {
	secret := "test-secret"
	var userID pgtype.UUID
	if err := userID.Scan("550e8400-e29b-41d4-a716-446655440000"); err != nil {
		t.Fatal(err)
	}

	token, err := authsvc.IssueAccessToken(userID, secret, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	var gotID pgtype.UUID
	handler := RequireAuth(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := UserIDFromContext(r.Context())
		if !ok {
			t.Error("expected user id in context")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		gotID = id
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("missing token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("valid token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if gotID != userID {
			t.Fatalf("user id = %v, want %v", gotID, userID)
		}
	})
}
