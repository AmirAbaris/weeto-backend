package auth

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestIssueAndParseAccessToken(t *testing.T) {
	var userID pgtype.UUID
	if err := userID.Scan("550e8400-e29b-41d4-a716-446655440000"); err != nil {
		t.Fatal(err)
	}

	secret := "test-secret"
	token, err := IssueAccessToken(userID, secret, time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := ParseAccessToken(token, secret)
	if err != nil {
		t.Fatal(err)
	}
	if parsed != userID {
		t.Fatalf("got %v, want %v", parsed, userID)
	}
}

func TestParseAccessTokenRejectsInvalid(t *testing.T) {
	_, err := ParseAccessToken("not-a-jwt", "secret")
	if err == nil {
		t.Fatal("expected error")
	}

	var userID pgtype.UUID
	_ = userID.Scan("550e8400-e29b-41d4-a716-446655440000")
	token, err := IssueAccessToken(userID, "secret-a", time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	_, err = ParseAccessToken(token, "secret-b")
	if err == nil {
		t.Fatal("expected wrong-secret error")
	}
}

func TestUUIDRoundTrip(t *testing.T) {
	var id pgtype.UUID
	if err := id.Scan("6ba7b810-9dad-11d1-80b4-00c04fd430c8"); err != nil {
		t.Fatal(err)
	}

	got, err := stringToUUID(uuidToString(id))
	if err != nil {
		t.Fatal(err)
	}
	if got != id {
		t.Fatalf("got %v, want %v", got, id)
	}
}
