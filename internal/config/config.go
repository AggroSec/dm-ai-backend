package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	DBURL                 string
	JWTSecret             string
	JWTExpiry             time.Duration
	JWTRefreshExpiry      time.Duration
	InternalSecret        string
	OpenRouterAPIKey      string
	OpenRouterModel       string
	OpenRouterCombatModel string
	OpenRouterMaxTokens   int
	Port                  string
	AppEnv                string
	DataDir               string
	DebugLogging          bool
}

func LoadConfig() (*Config, error) {
	cfg := &Config{}

	cfg.DBURL = requireEnv("DB_URL")
	cfg.JWTSecret = requireEnv("JWT_SECRET")
	cfg.InternalSecret = requireEnv("INTERNAL_SECRET")
	cfg.OpenRouterAPIKey = requireEnv("OPENROUTER_API_KEY")
	cfg.OpenRouterModel = requireEnv("OPENROUTER_MODEL")
	cfg.OpenRouterCombatModel = requireEnv("OPENROUTER_COMBAT_MODEL")
	cfg.DataDir = requireEnv("DATA_DIR")
	cfg.DebugLogging = parseBool("DEBUG_LOGGING")

	cfg.Port = getEnvOrDefault("PORT", "8080")
	cfg.AppEnv = getEnvOrDefault("APP_ENV", "development")

	var err error
	cfg.JWTExpiry, err = parseDuration("JWT_EXPIRY", "15m")
	if err != nil {
		return nil, err
	}
	cfg.JWTRefreshExpiry, err = parseDuration("JWT_REFRESH_EXPIRY", "168h")
	if err != nil {
		return nil, err
	}
	cfg.OpenRouterMaxTokens, err = parseInt("OPENROUTER_MAX_TOKENS", "2048")
	if err != nil {
		return nil, err
	}

	return cfg, nil

}

func parseBool(key string) bool {
	val := os.Getenv(key)
	if val == "" {
		panic(fmt.Sprintf("boolean not set correctly for %s", key))
	}
	var result bool
	if strings.ToLower(val) == "true" {
		result = true
	} else if strings.ToLower(val) == "false" {
		result = false
	} else {
		panic(fmt.Sprintf("boolean not set correctly for %s", key))
	}
	return result
}

func requireEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		panic(fmt.Sprintf("required env var %s is not set", key))
	}
	return val
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func parseDuration(key, defaultVal string) (time.Duration, error) {
	val := getEnvOrDefault(key, defaultVal)
	d, err := time.ParseDuration(val)
	if err != nil {
		return 0, fmt.Errorf("invalid duration for %s: %w", key, err)
	}
	return d, nil
}

func parseInt(key, defaultVal string) (int, error) {
	val := getEnvOrDefault(key, defaultVal)
	n, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("invalid int for %s: %w", key, err)
	}
	return n, nil
}
