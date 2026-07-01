package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLogging_recordsStatusAndRequestID(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /test", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("hi"))
	})

	handler := RequestID(Logging(logger, LoggingOptions{})(mux))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusTeapot)
	}

	requestID := rec.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Fatal("expected X-Request-ID response header")
	}

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("unmarshal log: %v", err)
	}

	if entry["msg"] != "request" {
		t.Fatalf("msg = %v, want request", entry["msg"])
	}
	if int(entry["status"].(float64)) != http.StatusTeapot {
		t.Fatalf("status = %v, want %d", entry["status"], http.StatusTeapot)
	}
	if entry["request_id"] != requestID {
		t.Fatalf("request_id = %v, want %s", entry["request_id"], requestID)
	}
	if entry["path"] != "/test" {
		t.Fatalf("path = %v, want /test", entry["path"])
	}
}

func TestLogging_skipsConfiguredPaths(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := Logging(logger, LoggingOptions{
		SkipPaths: map[string]struct{}{"/health": {}},
	})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if strings.TrimSpace(buf.String()) != "" {
		t.Fatalf("expected no log output, got %q", buf.String())
	}
}

func TestLogging_usesClientIPFromForwardedFor(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	handler := Logging(logger, LoggingOptions{})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/ip", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.1, 10.0.0.1")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("unmarshal log: %v", err)
	}
	if entry["remote_addr"] != "203.0.113.1" {
		t.Fatalf("remote_addr = %v, want 203.0.113.1", entry["remote_addr"])
	}
}
