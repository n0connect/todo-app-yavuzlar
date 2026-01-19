//go:build test
// +build test

package testutil

import (
	"encoding/hex"
	"fmt"
	"os"
	"testing"

	"todo-app-backend/internal/config"
	"todo-app-backend/internal/database"
	"todo-app-backend/internal/encryption"
	"todo-app-backend/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// SetupTestDB initializes a test database connection
// This should be called before running migration or integration tests
func SetupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	// Set test database environment variables if not already set
	// Automatically detect Docker environment
	if IsRunningInDocker() {
		// Docker environment - use service names
		if os.Getenv("DB_HOST") == "" {
			os.Setenv("DB_HOST", "postgres")
		}
	} else {
		// Local environment - use localhost
		if os.Getenv("DB_HOST") == "" {
			os.Setenv("DB_HOST", "localhost")
		}
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

	// Get database configuration
	dbHost := config.GetDBHost()
	dbUser := config.GetDBUser()
	dbPassword := config.GetDBPassword()
	dbName := config.GetDBName()
	dbPort := config.GetDBPort()
	dbSSLMode := config.GetDBSSLMode()

	// Build DSN
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
		dbHost, dbUser, dbPassword, dbName, dbPort, dbSSLMode)

	// Connect to database
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // Silence logs in tests
	})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Verify connection
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get database instance: %v", err)
	}

	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("Failed to ping test database: %v", err)
	}

	return db
}

// TeardownTestDB cleans up test database
func TeardownTestDB(t *testing.T, db *gorm.DB) {
	t.Helper()

	if db == nil {
		return
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Logf("Failed to get database instance for teardown: %v", err)
		return
	}

	if err := sqlDB.Close(); err != nil {
		t.Logf("Failed to close test database: %v", err)
	}
}

// CleanTestDatabase drops all tables and re-runs migrations
// Use this for a clean state before each test
func CleanTestDatabase(t *testing.T, db *gorm.DB) {
	t.Helper()

	// Drop all tables
	if err := db.Migrator().DropTable(&models.Todo{}, &models.User{}); err != nil {
		t.Logf("Failed to drop tables (may not exist): %v", err)
	}

	// Run migrations
	if err := db.AutoMigrate(&models.User{}, &models.Todo{}); err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}
}

// InitializeTestDatabase initializes the global database connection for tests
// This should be called once before running tests that require database
// It also resets global state (rate limiter, pending store) for test isolation
func InitializeTestDatabase(t *testing.T) {
	t.Helper()

	// Setup test environment
	SetupTestEnv(t)

	// Initialize database (this sets database.DB)
	database.Init()

	// Reset global state for test isolation
	// This ensures each test starts with a clean state
	resetGlobalStateForTesting(t)
}

// resetGlobalStateForTesting resets all global state that could affect test isolation
// SECURITY: This function calls test-only reset functions that are only available
// when compiled with -tags test. In production builds, these functions don't exist.
func resetGlobalStateForTesting(t *testing.T) {
	t.Helper()

	// Reset rate limiter (prevents rate limit state from affecting tests)
	// SECURITY: ResetRateLimiterForTesting is only available in test builds
	// It will NOT be compiled into production binaries
	resetRateLimiterForTesting()

	// Reset pending store (prevents pending registrations from affecting tests)
	// SECURITY: ResetPendingStoreForTesting is only available in test builds
	// It will NOT be compiled into production binaries
	resetPendingStoreForTesting()
}

// CleanupTestData removes all test data (users and todos) created during tests
// This should be called in defer after each test that creates data
func CleanupTestData(t *testing.T) {
	t.Helper()

	if database.DB == nil {
		return
	}

	// Delete all todos first (foreign key constraint)
	if err := database.DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.Todo{}).Error; err != nil {
		t.Logf("Failed to cleanup todos: %v", err)
	}

	// Delete all users
	if err := database.DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.User{}).Error; err != nil {
		t.Logf("Failed to cleanup users: %v", err)
	}
}

// CleanupTestUser removes a specific test user by UUID
func CleanupTestUser(t *testing.T, userUUID string) {
	t.Helper()

	if database.DB == nil {
		return
	}

	// Delete user's todos first
	if err := database.DB.Where("user_uuid = ?", userUUID).Delete(&models.Todo{}).Error; err != nil {
		t.Logf("Failed to cleanup todos for user %s: %v", userUUID, err)
	}

	// Delete user
	if err := database.DB.Where("uuid = ?", userUUID).Delete(&models.User{}).Error; err != nil {
		t.Logf("Failed to cleanup user %s: %v", userUUID, err)
	}
}

// GenerateTestEncryptedKey generates a valid encrypted key for testing
// This mimics the real registration flow where a user AES key is generated,
// encoded to hex, and then encrypted with the master key
// NOTE: This function is kept for backward compatibility, but new tests should
// use RegisterTestUser to test the actual system behavior
func GenerateTestEncryptedKey(t *testing.T) string {
	t.Helper()

	// Generate user AES key (32 bytes)
	userAESKey, err := encryption.GenerateUserAESKey()
	if err != nil {
		t.Fatalf("Failed to generate user AES key: %v", err)
	}

	// Convert to hex string
	userKeyHex := hex.EncodeToString(userAESKey)

	// Encrypt with master key
	encryptedUserKey, err := encryption.EncryptWithMasterKey(userKeyHex)
	if err != nil {
		t.Fatalf("Failed to encrypt user key: %v", err)
	}

	return encryptedUserKey
}
