package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// ========================================
// ENVIRONMENT VARIABLE HELPERS
// ========================================

func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	value := GetEnv(key, "")
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return defaultValue
	}
	return parsed
}

func getEnvBool(key string, defaultValue bool) bool {
	value := strings.ToLower(GetEnv(key, ""))
	if value == "" {
		return defaultValue
	}
	return value == "true" || value == "1" || value == "yes"
}

// ========================================
// JWT CONFIGURATION
// ========================================

// GetJWTSecret returns JWT secret key from environment
// MUST be at least 32 characters for security
func GetJWTSecret() string {
	return GetEnv("JWT_SECRET", "")
}

// GetJWTExpiration returns JWT token expiration duration
// Defaults to 15 minutes
func GetJWTExpiration() time.Duration {
	minutesStr := GetEnv("JWT_EXPIRATION_MINUTES", "15")
	minutes, err := strconv.Atoi(minutesStr)
	if err != nil || minutes <= 0 {
		return 15 * time.Minute
	}
	return time.Duration(minutes) * time.Minute
}

// ========================================
// API PAYLOAD LIMITS
// ========================================

// GetMaxBase64LoginLen returns max base64 length for login payloads
func GetMaxBase64LoginLen() int {
	return getEnvInt("MAX_BASE64_LOGIN_LEN", 512)
}

// GetMaxBase64TodoLen returns max base64 length for todo payloads
func GetMaxBase64TodoLen() int {
	return getEnvInt("MAX_BASE64_TODO_LEN", 4096)
}

// ========================================
// SECURITY CONFIGURATION
// ========================================

// GetAllowedOrigin returns CORS allowed origin
// Defaults to "*" for development, should be specific in production
func GetAllowedOrigin() string {
	return GetEnv("ALLOWED_ORIGIN", "*")
}

// IsProductionMode returns true if running in production
func IsProductionMode() bool {
	env := strings.ToLower(GetEnv("APP_ENV", "development"))
	return env == "production" || env == "prod"
}

// GetLogLevel returns logging level
// Values: debug, info, warn, error
func GetLogLevel() string {
	return strings.ToLower(GetEnv("LOG_LEVEL", "debug"))
}

// ========================================
// RATE LIMITING (Future)
// ========================================

// GetRateLimitEnabled returns whether rate limiting is enabled
func GetRateLimitEnabled() bool {
	return getEnvBool("RATE_LIMIT_ENABLED", false)
}

// GetRateLimitPerMinute returns max requests per minute per IP
func GetRateLimitPerMinute() int {
	return getEnvInt("RATE_LIMIT_PER_MINUTE", 60)
}

// ========================================
// DATABASE CONFIGURATION
// ========================================

// GetDBHost returns database host
func GetDBHost() string {
	return GetEnv("DB_HOST", "postgres")
}

// GetDBPort returns database port
func GetDBPort() string {
	return GetEnv("DB_PORT", "5432")
}

// GetDBUser returns database user
func GetDBUser() string {
	return GetEnv("DB_USER", "postgres")
}

// GetDBPassword returns database password
func GetDBPassword() string {
	return GetEnv("DB_PASSWORD", "postgres")
}

// GetDBName returns database name
func GetDBName() string {
	return GetEnv("DB_NAME", "todos")
}

// GetDBSSLMode returns database SSL mode
// Values: disable, require, verify-ca, verify-full
func GetDBSSLMode() string {
	if IsProductionMode() {
		return GetEnv("DB_SSL_MODE", "require")
	}
	return GetEnv("DB_SSL_MODE", "disable")
}

// ========================================
// ENCRYPTION CONFIGURATION
// ========================================

// GetEncryptionKey returns the master encryption key
// MUST be 32 bytes (64 hex characters)
func GetEncryptionKey() string {
	return GetEnv("ENCRYPTION_KEY", "")
}
