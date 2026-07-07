package smsir

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVerifySendSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/send/verify" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.Header.Get("x-api-key") != "test-key" {
			t.Fatalf("api key = %q", r.Header.Get("x-api-key"))
		}

		var body verifyRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body.Mobile != "9121234567" || body.TemplateID != 123456 {
			t.Fatalf("body = %+v", body)
		}
		if len(body.Parameters) != 1 || body.Parameters[0].Name != "Code" || body.Parameters[0].Value != "12345" {
			t.Fatalf("params = %+v", body.Parameters)
		}

		_ = json.NewEncoder(w).Encode(apiResponse{
			Status:  1,
			Message: "موفق",
			Data:    &verifyResponse{MessageID: 89545112, Cost: 1},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-key")
	messageID, cost, err := client.VerifySend(context.Background(), "9121234567", 123456, []Parameter{
		{Name: "Code", Value: "12345"},
	})
	if err != nil {
		t.Fatalf("VerifySend: %v", err)
	}
	if messageID != 89545112 || cost != 1 {
		t.Fatalf("messageID=%d cost=%f", messageID, cost)
	}
}

func TestVerifySendRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiResponse{Status: 0, Message: "invalid"})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-key")
	_, _, err := client.VerifySend(context.Background(), "9121234567", 123456, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}
