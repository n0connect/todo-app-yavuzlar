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
	testJWTSecret := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" // 64 chars
	testEncryptionKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" // 64 hex chars = 32 bytes
	testPepper := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" // 64 hex chars = 32 bytes

	os.Setenv("JWT_SECRET", testJWTSecret)
	os.Setenv("ENCRYPTION_KEY", testEncryptionKey)
	os.Setenv("ACCOUNT_LOOKUP_PEPPER", testPepper)
	os.Setenv("APP_ENV", "test")
	os.Setenv("LOG_LEVEL", "error") // Reduce log noise in tests
}

// TeardownTestEnv cleans up test environment variables
func TeardownTestEnv(t *testing.T) {
	t.Helper()

	os.Unsetenv("JWT_SECRET")
	os.Unsetenv("ENCRYPTION_KEY")
	os.Unsetenv("ACCOUNT_LOOKUP_PEPPER")
	os.Unsetenv("APP_ENV")
	os.Unsetenv("LOG_LEVEL")
}
