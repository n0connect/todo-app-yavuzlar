package config

import (
	"encoding/base64"
	"encoding/hex"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultJWTSecretMinLen         = 32
	defaultJWTIssuer               = "todo-app-backend"
	defaultJWTAudience             = "todo-app-frontend"
	defaultJWTExpirationMinutes    = 15
	defaultAllowedOrigins          = "http://localhost"
	defaultAppEnv                  = "development"
	defaultLogLevel                = "info"
	defaultDBHost                  = "postgres"
	defaultDBPort                  = "5432"
	defaultDBUser                  = "postgres"
	defaultDBPassword              = "postgres"
	defaultDBName                  = "todo_app"
	defaultDBSSLMode               = "disable"
	defaultBackendPort             = "8080"
	defaultMaxBase64LoginLen       = 4096
	defaultMaxBase64TodoLen        = 65536
	defaultMaxRequestBodyBytes     = 1048576
	defaultMaxJSONDepth            = 2
	defaultMaxTitleLength          = 1000
	defaultMinTitleLength          = 1
	defaultMaxTagLength            = 6
	defaultMaxTagsPerTodo          = 10
	// Argon2id parameters follow OWASP recommendations (2023)
	// Memory: 64MiB (65536 KiB) - recommended for password hashing
	// Time: 3 iterations - balance between security and performance
	// Parallelism: 4 - utilize multi-core CPUs effectively
	// Note: These defaults are tuned for production with 100-200 concurrent users
	// High-traffic scenarios (1000+ concurrent) may need adjustment or queueing
	defaultArgon2MemoryKiB         = 65536
	defaultArgon2Time              = 3
	defaultArgon2Parallelism       = 4  // Increased from 2 for better CPU utilization
	defaultArgon2SaltLength        = 16
	defaultArgon2HashLength        = 32
	defaultPendingTokenTTLSeconds  = 300
	defaultPendingCleanupSeconds   = 60
	defaultPendingIDBytes          = 16
	defaultInternalIDLength        = 24
	defaultRateLimitMaxTokens      = 20
	defaultRateLimitRefillSeconds  = 3
	defaultRateLimitCleanupSeconds = 300
	defaultRateLimitMaxBuckets     = 10000
	defaultServerReadHeaderTimeout = 5
	defaultServerReadTimeout       = 15
	defaultServerWriteTimeout      = 15
	defaultServerIdleTimeout       = 60
	defaultServerMaxHeaderBytes    = 8192
	defaultPoWDifficulty           = 22  // Increased from 18 to 22 for better security
	minPoWDifficulty               = 20  // Minimum difficulty
	maxPoWDifficulty               = 26  // Maximum difficulty (high load)
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

func getEnvInt(key string) int {
	value := strings.TrimSpace(GetEnv(key, ""))
	if value == "" {
		return 0
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0
	}
	return parsed
}

func getEnvIntDefault(key string, defaultValue int) int {
	value := strings.TrimSpace(GetEnv(key, ""))
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0
	}
	return parsed
}

func getEnvBool(key string) bool {
	value := strings.ToLower(strings.TrimSpace(GetEnv(key, "")))
	if value == "" {
		return false
	}
	return value == "true" || value == "1" || value == "yes"
}

// ========================================
// JWT CONFIGURATION
// ========================================

// GetJWTSecret returns JWT secret key from environment
func GetJWTSecret() string {
	return GetEnv("JWT_SECRET", "")
}

// GetJWTSecretMinLength returns minimum JWT secret length in characters
func GetJWTSecretMinLength() int {
	return getEnvIntDefault("JWT_SECRET_MIN_LEN", defaultJWTSecretMinLen)
}

// GetJWTIssuer returns JWT issuer value
func GetJWTIssuer() string {
	return strings.TrimSpace(GetEnv("JWT_ISSUER", defaultJWTIssuer))
}

// GetJWTAudience returns JWT audience value
func GetJWTAudience() string {
	return strings.TrimSpace(GetEnv("JWT_AUDIENCE", defaultJWTAudience))
}

// GetJWTExpiration returns JWT token expiration duration
func GetJWTExpiration() time.Duration {
	minutes := getEnvIntDefault("JWT_EXPIRATION_MINUTES", defaultJWTExpirationMinutes)
	if minutes <= 0 {
		return 0
	}
	return time.Duration(minutes) * time.Minute
}

// ========================================
// API PAYLOAD LIMITS
// ========================================

// GetMaxBase64LoginLen returns max base64 length for login payloads
func GetMaxBase64LoginLen() int {
	return getEnvIntDefault("MAX_BASE64_LOGIN_LEN", defaultMaxBase64LoginLen)
}

// GetMaxBase64TodoLen returns max base64 length for todo payloads
func GetMaxBase64TodoLen() int {
	return getEnvIntDefault("MAX_BASE64_TODO_LEN", defaultMaxBase64TodoLen)
}

// ========================================
// REQUEST AND VALIDATION LIMITS
// ========================================

// GetMaxRequestBodyBytes returns max request body size in bytes
func GetMaxRequestBodyBytes() int {
	return getEnvIntDefault("MAX_REQUEST_BODY_BYTES", defaultMaxRequestBodyBytes)
}

// GetMaxJSONDepth returns max JSON nesting depth
func GetMaxJSONDepth() int {
	return getEnvIntDefault("MAX_JSON_DEPTH", defaultMaxJSONDepth)
}

// GetMaxTitleLength returns max todo title length in runes
func GetMaxTitleLength() int {
	return getEnvIntDefault("MAX_TITLE_LENGTH", defaultMaxTitleLength)
}

// GetMinTitleLength returns min todo title length in runes
func GetMinTitleLength() int {
	return getEnvIntDefault("MIN_TITLE_LENGTH", defaultMinTitleLength)
}

// GetMaxTagLength returns max tag length
func GetMaxTagLength() int {
	return getEnvIntDefault("MAX_TAG_LENGTH", defaultMaxTagLength)
}

// GetMaxTagsPerTodo returns max number of tags per todo
func GetMaxTagsPerTodo() int {
	return getEnvIntDefault("MAX_TAGS_PER_TODO", defaultMaxTagsPerTodo)
}

// ========================================
// SECURITY CONFIGURATION
// ========================================

// GetAllowedOrigins returns CORS allowed origins (comma-separated)
func GetAllowedOrigins() string {
	return GetEnv("ALLOWED_ORIGINS", defaultAllowedOrigins)
}

// IsProductionMode returns true if running in production
func IsProductionMode() bool {
	env := strings.ToLower(strings.TrimSpace(GetEnv("APP_ENV", defaultAppEnv)))
	return env == "production" || env == "prod"
}

// GetLogLevel returns logging level
// Values: debug, info, warn, error
func GetLogLevel() string {
	return strings.ToLower(strings.TrimSpace(GetEnv("LOG_LEVEL", defaultLogLevel)))
}

// GetTrustedProxies returns comma-separated trusted proxy IPs/CIDRs
func GetTrustedProxies() string {
	return strings.TrimSpace(GetEnv("TRUSTED_PROXIES", ""))
}

// GetHSTSMaxAge returns HSTS max-age in seconds
func GetHSTSMaxAge() int {
	return getEnvInt("HSTS_MAX_AGE")
}

// GetHSTSIncludeSubDomains returns whether HSTS includeSubDomains is enabled
func GetHSTSIncludeSubDomains() bool {
	return getEnvBool("HSTS_INCLUDE_SUBDOMAINS")
}

// GetHSTSPreload returns whether HSTS preload is enabled
func GetHSTSPreload() bool {
	return getEnvBool("HSTS_PRELOAD")
}

// GetSecurityHeaderCOOP returns Cross-Origin-Opener-Policy value
func GetSecurityHeaderCOOP() string {
	value := strings.TrimSpace(GetEnv("SEC_HEADER_COOP", ""))
	if strings.EqualFold(value, "off") || strings.EqualFold(value, "disabled") {
		return ""
	}
	return value
}

// GetSecurityHeaderCORP returns Cross-Origin-Resource-Policy value
func GetSecurityHeaderCORP() string {
	value := strings.TrimSpace(GetEnv("SEC_HEADER_CORP", ""))
	if strings.EqualFold(value, "off") || strings.EqualFold(value, "disabled") {
		return ""
	}
	return value
}

// GetSecurityHeaderCOEP returns Cross-Origin-Embedder-Policy value
func GetSecurityHeaderCOEP() string {
	value := strings.TrimSpace(GetEnv("SEC_HEADER_COEP", ""))
	if strings.EqualFold(value, "off") || strings.EqualFold(value, "disabled") {
		return ""
	}
	return value
}

// ========================================
// RATE LIMITING
// ========================================

// GetRateLimitMaxTokens returns max tokens per bucket
func GetRateLimitMaxTokens() int {
	return getEnvIntDefault("RATE_LIMIT_MAX_TOKENS", defaultRateLimitMaxTokens)
}

// GetRateLimitRefillIntervalSeconds returns refill interval in seconds
func GetRateLimitRefillIntervalSeconds() int {
	return getEnvIntDefault("RATE_LIMIT_REFILL_INTERVAL_SEC", defaultRateLimitRefillSeconds)
}

// GetRateLimitCleanupIntervalSeconds returns cleanup interval in seconds
func GetRateLimitCleanupIntervalSeconds() int {
	return getEnvIntDefault("RATE_LIMIT_CLEANUP_INTERVAL_SEC", defaultRateLimitCleanupSeconds)
}

// GetRateLimitMaxBuckets returns max bucket count before cleanup
func GetRateLimitMaxBuckets() int {
	return getEnvIntDefault("RATE_LIMIT_MAX_BUCKETS", defaultRateLimitMaxBuckets)
}

// ========================================
// DATABASE CONFIGURATION
// ========================================

// GetDBHost returns database host
func GetDBHost() string {
	return strings.TrimSpace(GetEnv("DB_HOST", defaultDBHost))
}

// GetDBPort returns database port
func GetDBPort() string {
	return strings.TrimSpace(GetEnv("DB_PORT", defaultDBPort))
}

// GetDBUser returns database user
func GetDBUser() string {
	return strings.TrimSpace(GetEnv("DB_USER", defaultDBUser))
}

// GetDBPassword returns database password
func GetDBPassword() string {
	return strings.TrimSpace(GetEnv("DB_PASSWORD", defaultDBPassword))
}

// GetDBName returns database name
func GetDBName() string {
	return strings.TrimSpace(GetEnv("DB_NAME", defaultDBName))
}

// GetDBSSLMode returns database SSL mode
// Values: disable, require, verify-ca, verify-full
func GetDBSSLMode() string {
	// Note: Logging is handled in database.connection.go ValidateStartupConfig
	return strings.TrimSpace(GetEnv("DB_SSL_MODE", defaultDBSSLMode))
}

// ========================================
// ENCRYPTION CONFIGURATION
// ========================================

// GetArgon2MemoryKiB returns Argon2 memory in KiB
func GetArgon2MemoryKiB() int {
	return getEnvIntDefault("ARGON2_MEMORY_KIB", defaultArgon2MemoryKiB)
}

// GetArgon2Time returns Argon2 time cost
func GetArgon2Time() int {
	return getEnvIntDefault("ARGON2_TIME", defaultArgon2Time)
}

// GetArgon2Parallelism returns Argon2 parallelism
func GetArgon2Parallelism() int {
	return getEnvIntDefault("ARGON2_PARALLELISM", defaultArgon2Parallelism)
}

// GetArgon2SaltLength returns Argon2 salt length in bytes
func GetArgon2SaltLength() int {
	return getEnvIntDefault("ARGON2_SALT_LENGTH", defaultArgon2SaltLength)
}

// GetArgon2HashLength returns Argon2 hash length in bytes
func GetArgon2HashLength() int {
	return getEnvIntDefault("ARGON2_HASH_LENGTH", defaultArgon2HashLength)
}

// GetAccountLookupPepper returns the pepper for account lookup HMAC
// MUST be at least 32 bytes (can be hex or base64 encoded)
func GetAccountLookupPepper() []byte {
	pepperStr := GetEnv("ACCOUNT_LOOKUP_PEPPER", "")
	if pepperStr == "" {
		return nil
	}

	// Try hex decoding first
	if decoded, err := hex.DecodeString(pepperStr); err == nil && len(decoded) >= 32 {
		return decoded
	}

	// Try base64 decoding
	if decoded, err := base64.StdEncoding.DecodeString(pepperStr); err == nil && len(decoded) >= 32 {
		return decoded
	}

	// Try raw bytes (if exactly 32+ bytes)
	if len(pepperStr) >= 32 {
		return []byte(pepperStr)
	}

	return nil
}

// ========================================
// PENDING REGISTRATION CONFIGURATION
// ========================================

// GetPendingTokenTTL returns pending token TTL
func GetPendingTokenTTL() time.Duration {
	seconds := getEnvIntDefault("PENDING_TOKEN_TTL_SEC", defaultPendingTokenTTLSeconds)
	if seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

// GetPendingCleanupInterval returns pending store cleanup interval
func GetPendingCleanupInterval() time.Duration {
	seconds := getEnvIntDefault("PENDING_CLEANUP_INTERVAL_SEC", defaultPendingCleanupSeconds)
	if seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

// GetPendingIDBytes returns pending ID length in bytes
func GetPendingIDBytes() int {
	return getEnvIntDefault("PENDING_ID_BYTES", defaultPendingIDBytes)
}

// ========================================
// IDENTIFIER CONFIGURATION
// ========================================

// GetInternalIDLength returns internal UUID length
func GetInternalIDLength() int {
	return getEnvIntDefault("INTERNAL_ID_LENGTH", defaultInternalIDLength)
}

// ========================================
// SERVER CONFIGURATION
// ========================================

// GetBackendPort returns backend server port
func GetBackendPort() string {
	return strings.TrimSpace(GetEnv("BACKEND_PORT", defaultBackendPort))
}

// GetServerReadHeaderTimeout returns read header timeout
func GetServerReadHeaderTimeout() time.Duration {
	seconds := getEnvIntDefault("SERVER_READ_HEADER_TIMEOUT_SEC", defaultServerReadHeaderTimeout)
	if seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

// GetServerReadTimeout returns read timeout
func GetServerReadTimeout() time.Duration {
	seconds := getEnvIntDefault("SERVER_READ_TIMEOUT_SEC", defaultServerReadTimeout)
	if seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

// GetServerWriteTimeout returns write timeout
func GetServerWriteTimeout() time.Duration {
	seconds := getEnvIntDefault("SERVER_WRITE_TIMEOUT_SEC", defaultServerWriteTimeout)
	if seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

// GetServerIdleTimeout returns idle timeout
func GetServerIdleTimeout() time.Duration {
	seconds := getEnvIntDefault("SERVER_IDLE_TIMEOUT_SEC", defaultServerIdleTimeout)
	if seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

// GetServerMaxHeaderBytes returns max header bytes
func GetServerMaxHeaderBytes() int {
	return getEnvIntDefault("SERVER_MAX_HEADER_BYTES", defaultServerMaxHeaderBytes)
}

// ========================================
// PROOF OF WORK CONFIGURATION
// ========================================

// GetPoWDifficulty returns the PoW difficulty level
func GetPoWDifficulty() int {
	return getEnvIntDefault("POW_DIFFICULTY", defaultPoWDifficulty)
}

// DefaultPoWDifficulty returns the default PoW difficulty level
func DefaultPoWDifficulty() int {
	return defaultPoWDifficulty
}

// MinPoWDifficulty returns the minimum PoW difficulty level
func MinPoWDifficulty() int {
	return minPoWDifficulty
}

// MaxPoWDifficulty returns the maximum PoW difficulty level
func MaxPoWDifficulty() int {
	return maxPoWDifficulty
}

