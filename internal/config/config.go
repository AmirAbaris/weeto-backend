package config

import (
	"errors"
	"os"
)

type Config struct {
	Port        string
	Env         string
	DBURL       string
	JWTSecret   string
	FrontendURL string
}

func LoadConfig() (*Config, error) {
	cfg := &Config{
		Port:        os.Getenv("PORT"),
		Env:         os.Getenv("ENV"),
		DBURL:       os.Getenv("DB_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		FrontendURL: os.Getenv("FRONTEND_URL"),
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
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return fallback
}
