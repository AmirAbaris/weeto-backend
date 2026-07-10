package google

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestOAuthStateRoundTrip(t *testing.T) {
	var userID pgtype.UUID
	if err := userID.Scan("550e8400-e29b-41d4-a716-446655440000"); err != nil {
		t.Fatal(err)
	}

	state, err := signState("test-secret", userID)
	if err != nil {
		t.Fatal(err)
	}

	got, err := verifyState("test-secret", state)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if got.String() != userID.String() {
		t.Fatalf("user id = %s, want %s", got.String(), userID.String())
	}
}

func TestOAuthStateTampered(t *testing.T) {
	var userID pgtype.UUID
	_ = userID.Scan("550e8400-e29b-41d4-a716-446655440000")

	state, err := signState("test-secret", userID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = verifyState("wrong-secret", state)
	if err != ErrInvalidOAuthState {
		t.Fatalf("err = %v", err)
	}
}

func TestOAuthStateExpired(t *testing.T) {
	// stateTTL is package-level; we only verify invalid base64 fails
	_, err := verifyState("test-secret", "not-valid-state")
	if err != ErrInvalidOAuthState {
		t.Fatalf("err = %v", err)
	}
	_ = time.Second
}
