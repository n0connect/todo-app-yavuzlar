package testutil

import (
	"fmt"
	"os"
	"testing"

	"todo-app-backend/internal/config"
	"todo-app-backend/internal/database"
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
func InitializeTestDatabase(t *testing.T) {
	t.Helper()

	// Setup test environment
	SetupTestEnv(t)

	// Initialize database (this sets database.DB and database.EncryptionKey)
	database.Init()
}
