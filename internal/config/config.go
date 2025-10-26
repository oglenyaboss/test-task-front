package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppPort           string
	AppEnv            string
	JWTSecret         string
	AccessTokenTTL    time.Duration
	RefreshTokenTTL   time.Duration
	DatabaseURL       string
	CORSAllowedOrigin []string
	ArtificialDelay   time.Duration
}

func Load() (*Config, error) {
	cfg := &Config{}

	cfg.AppPort = getEnv("APP_PORT", "8080")
	cfg.AppEnv = getEnv("APP_ENV", "dev")
	cfg.JWTSecret = getEnv("JWT_SECRET", "")
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	accessTTLStr := getEnv("ACCESS_TOKEN_TTL", "15m")
	refreshTTLStr := getEnv("REFRESH_TOKEN_TTL", "168h")

	var err error
	if cfg.AccessTokenTTL, err = time.ParseDuration(accessTTLStr); err != nil {
		return nil, fmt.Errorf("invalid ACCESS_TOKEN_TTL: %w", err)
	}
	if cfg.RefreshTokenTTL, err = time.ParseDuration(refreshTTLStr); err != nil {
		return nil, fmt.Errorf("invalid REFRESH_TOKEN_TTL: %w", err)
	}

	cfg.DatabaseURL = getEnv("DATABASE_URL", "")
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	origins := strings.Split(getEnv("CORS_ALLOWED_ORIGINS", "*"), ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}
	cfg.CORSAllowedOrigin = origins

	delayMS := getEnv("ARTIFICIAL_DELAY_MS", "400")
	if delayMS == "" {
		delayMS = "0"
	}
	delayVal, err := strconv.Atoi(delayMS)
	if err != nil || delayVal < 0 {
		return nil, fmt.Errorf("invalid ARTIFICIAL_DELAY_MS")
	}
	cfg.ArtificialDelay = time.Duration(delayVal) * time.Millisecond

	return cfg, nil
}

func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		panic(err)
	}
	return cfg
}

func getEnv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}
