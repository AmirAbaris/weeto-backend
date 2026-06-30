package auth

import "testing"

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("secret123")
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyPassword("secret123", hash); err != nil {
		t.Fatal(err)
	}
	if err := VerifyPassword("wrong", hash); err == nil {
		t.Fatal("expected mismatch error")
	}
}
