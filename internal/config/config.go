package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port        string
	Env         string
	DBURL       string
	JWTSecret   string
	FrontendURL string

	JWTAccessTTL  time.Duration
	JWTRefreshTTL time.Duration

	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string

	TokenEncryptionKey  []byte // 32 bytes for AES-256
	SessionCookieSecure bool

	ResendAPIKey string
	ResendFrom   string
}

func LoadConfig() (*Config, error) {
	accessTTL, err := parseDuration("JWT_ACCESS_TTL", "15m")
	if err != nil {
		return nil, err
	}
	refreshTTL, err := parseDuration("JWT_REFRESH_TTL", "168h")
	if err != nil {
		return nil, err
	}

	encKey, err := loadEncryptionKey(os.Getenv("TOKEN_ENCRYPTION_KEY"))
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		Port:                getEnv("PORT", "8080"),
		Env:                 getEnv("ENV", "development"),
		DBURL:               os.Getenv("DB_URL"),
		JWTSecret:           os.Getenv("JWT_SECRET"),
		FrontendURL:         os.Getenv("FRONTEND_URL"),
		JWTAccessTTL:        accessTTL,
		JWTRefreshTTL:       refreshTTL,
		GoogleClientID:      os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret:  os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:   os.Getenv("GOOGLE_REDIRECT_URL"),
		TokenEncryptionKey:  encKey,
		SessionCookieSecure: parseBool(os.Getenv("SESSION_COOKIE_SECURE")),
		ResendAPIKey:        os.Getenv("RESEND_TOKEN_KEY"),
		ResendFrom:          getEnv("RESEND_FROM", getEnv("SMTP_FROM", "Weeto <noreply@weeto.ir>")),
	}
	if cfg.DBURL == "" {
		return nil, errors.New("DB_URL is required")
	}
	if cfg.JWTSecret == "" {
		return nil, errors.New("JWT_SECRET is required")
	}
	if cfg.FrontendURL == "" {
		return nil, errors.New("FRONTEND_URL is required")
	}
	// Google + encryption key only required when you wire OAuth — skip strict check for now
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return fallback
}

func parseDuration(key, fallback string) (time.Duration, error) {
	v := getEnv(key, fallback)
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return d, nil
}

func loadEncryptionKey(b64 string) ([]byte, error) {
	if b64 == "" {
		return nil, nil // ok until Google connect phase
	}
	key, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("TOKEN_ENCRYPTION_KEY: invalid base64")
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("TOKEN_ENCRYPTION_KEY: must decode to 32 bytes, got %d", len(key))
	}
	return key, nil
}

func parseBool(s string) bool {
	v, _ := strconv.ParseBool(s)
	return v
}
