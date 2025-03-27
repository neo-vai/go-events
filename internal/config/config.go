package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env      string
	Port     string
	LogLevel string

	DatabaseURL     string
	DatabaseTestURL string

	JWTSecret       string
	JWTExpiresHours int

	APIKeyLength int
	BcryptCost   int

	RateLimitGlobal RequestsPerDuration
	RateLimitLogin  RequestsPerDuration

	MaxEventsLimit int
	HTTPTimeout    time.Duration

	TrustedProxies []string

	DBMaxConns        int
	DBMinConns        int
	DBMaxConnLifetime time.Duration
	DBMaxConnIdleTime time.Duration
}

type RequestsPerDuration struct {
	Requests int
	Per      time.Duration
}

func Load() (*Config, error) {
	cfg := &Config{}

	cfg.Env = getEnv("EVENTS_ENV", "development")
	cfg.Port = getEnv("PORT", "8080")
	cfg.LogLevel = getEnv("LOG_LEVEL", "info")

	cfg.DatabaseURL = os.Getenv("DATABASE_URL")
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	cfg.DatabaseTestURL = os.Getenv("DATABASE_URL_TEST")

	cfg.JWTSecret = os.Getenv("JWT_SECRET")
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	cfg.JWTExpiresHours = getEnvInt("JWT_EXPIRES_HOURS", 24)

	cfg.APIKeyLength = getEnvInt("API_KEY_LENGTH", 32)
	cfg.BcryptCost = getEnvInt("BCRYPT_COST", 12)

	globalLimit, err := parseRateLimit(os.Getenv("RATE_LIMIT_GLOBAL"), "100/1m")
	if err != nil {
		return nil, fmt.Errorf("invalid RATE_LIMIT_GLOBAL: %w", err)
	}
	cfg.RateLimitGlobal = globalLimit

	loginLimit, err := parseRateLimit(os.Getenv("RATE_LIMIT_LOGIN"), "5/1m")
	if err != nil {
		return nil, fmt.Errorf("invalid RATE_LIMIT_LOGIN: %w", err)
	}
	cfg.RateLimitLogin = loginLimit

	cfg.MaxEventsLimit = getEnvInt("MAX_EVENTS_LIMIT", 100)
	httpTimeoutSec := getEnvInt("HTTP_TIMEOUT", 10)
	cfg.HTTPTimeout = time.Duration(httpTimeoutSec) * time.Second

	trusted := getEnv("TRUSTED_PROXIES", "127.0.0.1,::1")
	cfg.TrustedProxies = splitAndTrim(trusted, ",")

	cfg.DBMaxConns = getEnvInt("DB_MAX_CONNS", 25)
	cfg.DBMinConns = getEnvInt("DB_MIN_CONNS", 5)
	cfg.DBMaxConnLifetime = time.Duration(getEnvInt("DB_MAX_CONN_LIFETIME_SEC", 3600)) * time.Second
	cfg.DBMaxConnIdleTime = time.Duration(getEnvInt("DB_MAX_CONN_IDLE_TIME_SEC", 1800)) * time.Second

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultValue
}

func parseRateLimit(val, defaultVal string) (RequestsPerDuration, error) {
	if val == "" {
		val = defaultVal
	}
	parts := splitAndTrim(val, "/")
	if len(parts) != 2 {
		return RequestsPerDuration{}, fmt.Errorf("format must be 'requests/duration' (e.g., '100/1m')")
	}
	requests, err := strconv.Atoi(parts[0])
	if err != nil {
		return RequestsPerDuration{}, fmt.Errorf("invalid request count: %w", err)
	}
	if requests <= 0 {
		return RequestsPerDuration{}, fmt.Errorf("requests must be > 0")
	}
	duration, err := time.ParseDuration(parts[1])
	if err != nil {
		return RequestsPerDuration{}, fmt.Errorf("invalid duration: %w", err)
	}
	if duration <= 0 {
		return RequestsPerDuration{}, fmt.Errorf("duration must be > 0")
	}
	return RequestsPerDuration{Requests: requests, Per: duration}, nil
}

func splitAndTrim(s, sep string) []string {
	if s == "" {
		return []string{}
	}
	parts := strings.Split(s, sep)
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
