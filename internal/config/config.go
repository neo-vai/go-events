package config

import (
	"fmt"
	"net/url"
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

	TrustedProxies     []string
	CORSAllowedOrigins []string

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

	corsOrigins := getEnv("CORS_ALLOWED_ORIGINS", "")
	if corsOrigins == "" {
		// In production, it's safer to have an explicit list; for dev, allow all.
		if cfg.Env == "development" {
			cfg.CORSAllowedOrigins = []string{"*"}
		} else {
			cfg.CORSAllowedOrigins = []string{}
		}
	} else {
		cfg.CORSAllowedOrigins = splitAndTrim(corsOrigins, ",")
	}

	cfg.DBMaxConns = getEnvInt("DB_MAX_CONNS", 25)
	cfg.DBMinConns = getEnvInt("DB_MIN_CONNS", 5)
	cfg.DBMaxConnLifetime = time.Duration(getEnvInt("DB_MAX_CONN_LIFETIME_SEC", 3600)) * time.Second
	cfg.DBMaxConnIdleTime = time.Duration(getEnvInt("DB_MAX_CONN_IDLE_TIME_SEC", 1800)) * time.Second

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	// JWT secret strength
	if len(c.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters long")
	}

	// API key length
	if c.APIKeyLength < 16 {
		return fmt.Errorf("API_KEY_LENGTH must be at least 16")
	}

	// Bcrypt cost range
	if c.BcryptCost < 4 || c.BcryptCost > 31 {
		return fmt.Errorf("BCRYPT_COST must be between 4 and 31")
	}

	// Port validation
	portNum, err := strconv.Atoi(c.Port)
	if err != nil || portNum < 1 || portNum > 65535 {
		return fmt.Errorf("PORT must be a valid port number (1-65535)")
	}
	if c.Env != "development" && portNum < 1024 {
		return fmt.Errorf("PORT should be >= 1024 for non-root user")
	}

	// Database URL format
	if _, err := url.Parse(c.DatabaseURL); err != nil {
		return fmt.Errorf("DATABASE_URL is not a valid URL: %w", err)
	}

	// Ensure at least one trusted proxy is set if not empty (optional)
	if len(c.TrustedProxies) == 0 && c.Env != "development" {
		// This is not strictly required, but recommended
		// We'll just warn via log (but we don't have logger here; we'll skip)
	}

	return nil
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
