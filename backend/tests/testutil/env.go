//go:build test
// +build test

package testutil

import (
	"os"
	"testing"
)

// SetupTestEnv sets up test environment variables
func SetupTestEnv(t *testing.T) {
	t.Helper()

	// Set required environment variables for testing
	testJWTSecret := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"     // 64 chars
	testEncryptionKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" // 64 hex chars = 32 bytes
	testPepper := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"        // 64 hex chars = 32 bytes

	os.Setenv("JWT_SECRET", testJWTSecret)
	os.Setenv("JWT_SECRET_MIN_LEN", "32")
	os.Setenv("JWT_EXPIRATION_MINUTES", "15")
	os.Setenv("JWT_ISSUER", "todo-app-backend")
	os.Setenv("JWT_AUDIENCE", "todo-app-frontend")
	os.Setenv("MASTER_KEY_ACTIVE", testEncryptionKey)
	os.Setenv("MASTER_KEY_ACTIVE_ID", "test-key-1")
	os.Setenv("MASTER_KEY_OLD", "")
	os.Setenv("ACCOUNT_LOOKUP_PEPPER", testPepper)
	os.Setenv("MAX_BASE64_LOGIN_LEN", "4096")
	os.Setenv("MAX_BASE64_TODO_LEN", "65536")
	os.Setenv("MAX_REQUEST_BODY_BYTES", "1048576")
	os.Setenv("MAX_JSON_DEPTH", "2")
	os.Setenv("MAX_TITLE_LENGTH", "1000")
	os.Setenv("MIN_TITLE_LENGTH", "1")
	os.Setenv("MAX_TAG_LENGTH", "6")
	os.Setenv("MAX_TAGS_PER_TODO", "10")
	os.Setenv("ARGON2_MEMORY_KIB", "65536")
	os.Setenv("ARGON2_TIME", "3")
	os.Setenv("ARGON2_PARALLELISM", "2")
	os.Setenv("ARGON2_SALT_LENGTH", "16")
	os.Setenv("ARGON2_HASH_LENGTH", "32")
	os.Setenv("PENDING_TOKEN_TTL_SEC", "300")
	os.Setenv("PENDING_CLEANUP_INTERVAL_SEC", "60")
	os.Setenv("PENDING_ID_BYTES", "16")
	os.Setenv("INTERNAL_ID_LENGTH", "24")
	os.Setenv("RATE_LIMIT_MAX_TOKENS", "20")
	os.Setenv("RATE_LIMIT_REFILL_INTERVAL_SEC", "3")
	os.Setenv("RATE_LIMIT_CLEANUP_INTERVAL_SEC", "300")
	os.Setenv("RATE_LIMIT_MAX_BUCKETS", "10000")
	os.Setenv("SERVER_READ_HEADER_TIMEOUT_SEC", "5")
	os.Setenv("SERVER_READ_TIMEOUT_SEC", "15")
	os.Setenv("SERVER_WRITE_TIMEOUT_SEC", "15")
	os.Setenv("SERVER_IDLE_TIMEOUT_SEC", "60")
	os.Setenv("SERVER_MAX_HEADER_BYTES", "8192")
	os.Setenv("BACKEND_PORT", "8080")
	os.Setenv("ALLOWED_ORIGINS", "http://localhost:3000")
	if os.Getenv("DB_HOST") == "" {
		os.Setenv("DB_HOST", "localhost")
	}
	if os.Getenv("DB_PORT") == "" {
		os.Setenv("DB_PORT", "5432")
	}
	if os.Getenv("DB_USER") == "" {
		os.Setenv("DB_USER", "postgres")
	}
	if os.Getenv("DB_PASSWORD") == "" {
		os.Setenv("DB_PASSWORD", "postgres")
	}
	if os.Getenv("DB_NAME") == "" {
		os.Setenv("DB_NAME", "todos_test")
	}
	if os.Getenv("DB_SSL_MODE") == "" {
		os.Setenv("DB_SSL_MODE", "disable")
	}
	os.Setenv("APP_ENV", "test")
	os.Setenv("LOG_LEVEL", "error") // Reduce log noise in tests
}

// TeardownTestEnv cleans up test environment variables
func TeardownTestEnv(t *testing.T) {
	t.Helper()

	os.Unsetenv("JWT_SECRET")
	os.Unsetenv("JWT_SECRET_MIN_LEN")
	os.Unsetenv("JWT_EXPIRATION_MINUTES")
	os.Unsetenv("JWT_ISSUER")
	os.Unsetenv("JWT_AUDIENCE")
	os.Unsetenv("MASTER_KEY_ACTIVE")
	os.Unsetenv("MASTER_KEY_ACTIVE_ID")
	os.Unsetenv("MASTER_KEY_OLD")
	os.Unsetenv("ACCOUNT_LOOKUP_PEPPER")
	os.Unsetenv("MAX_BASE64_LOGIN_LEN")
	os.Unsetenv("MAX_BASE64_TODO_LEN")
	os.Unsetenv("MAX_REQUEST_BODY_BYTES")
	os.Unsetenv("MAX_JSON_DEPTH")
	os.Unsetenv("MAX_TITLE_LENGTH")
	os.Unsetenv("MIN_TITLE_LENGTH")
	os.Unsetenv("MAX_TAG_LENGTH")
	os.Unsetenv("MAX_TAGS_PER_TODO")
	os.Unsetenv("ARGON2_MEMORY_KIB")
	os.Unsetenv("ARGON2_TIME")
	os.Unsetenv("ARGON2_PARALLELISM")
	os.Unsetenv("ARGON2_SALT_LENGTH")
	os.Unsetenv("ARGON2_HASH_LENGTH")
	os.Unsetenv("PENDING_TOKEN_TTL_SEC")
	os.Unsetenv("PENDING_CLEANUP_INTERVAL_SEC")
	os.Unsetenv("PENDING_ID_BYTES")
	os.Unsetenv("INTERNAL_ID_LENGTH")
	os.Unsetenv("RATE_LIMIT_MAX_TOKENS")
	os.Unsetenv("RATE_LIMIT_REFILL_INTERVAL_SEC")
	os.Unsetenv("RATE_LIMIT_CLEANUP_INTERVAL_SEC")
	os.Unsetenv("RATE_LIMIT_MAX_BUCKETS")
	os.Unsetenv("SERVER_READ_HEADER_TIMEOUT_SEC")
	os.Unsetenv("SERVER_READ_TIMEOUT_SEC")
	os.Unsetenv("SERVER_WRITE_TIMEOUT_SEC")
	os.Unsetenv("SERVER_IDLE_TIMEOUT_SEC")
	os.Unsetenv("SERVER_MAX_HEADER_BYTES")
	os.Unsetenv("BACKEND_PORT")
	os.Unsetenv("ALLOWED_ORIGINS")
	os.Unsetenv("APP_ENV")
	os.Unsetenv("LOG_LEVEL")
}
