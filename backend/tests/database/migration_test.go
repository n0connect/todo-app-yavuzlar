package database

import (
	"testing"

	"todo-app-backend/internal/database"
	"todo-app-backend/internal/models"
	"todo-app-backend/tests/testutil"
)

// TestDatabaseMigration tests that database migrations run successfully
// This test should run BEFORE backend starts to ensure database is properly set up
func TestDatabaseMigration(t *testing.T) {
	// Setup test environment (includes encryption keys)
	testutil.SetupTestEnv(t)
	defer testutil.TeardownTestEnv(t)

	// Setup database environment variables
	// These should be set before database.Init() is called
	testutil.SetupTestDatabaseEnv()

	// Initialize database (this runs migrations)
	// This is what backend does on startup
	database.Init()

	// Verify database connection is established
	if database.DB == nil {
		t.Fatal("database.DB should not be nil after Init()")
	}

	// Verify tables exist
	if !database.DB.Migrator().HasTable(&models.User{}) {
		t.Error("users table should exist after migration")
	}

	if !database.DB.Migrator().HasTable(&models.Todo{}) {
		t.Error("todos table should exist after migration")
	}
}

// TestUserTableSchema tests that User table has correct schema
func TestUserTableSchema(t *testing.T) {
	testutil.SetupTestEnv(t)
	defer testutil.TeardownTestEnv(t)

	// Setup database environment variables
	testutil.SetupTestDatabaseEnv()

	database.Init()

	// Check required columns exist
	hasUUID := database.DB.Migrator().HasColumn(&models.User{}, "uuid")
	if !hasUUID {
		t.Error("users table should have 'uuid' column")
	}

	hasAccountLookup := database.DB.Migrator().HasColumn(&models.User{}, "account_lookup")
	if !hasAccountLookup {
		t.Error("users table should have 'account_lookup' column")
	}

	hasAccountHash := database.DB.Migrator().HasColumn(&models.User{}, "account_hash")
	if !hasAccountHash {
		t.Error("users table should have 'account_hash' column")
	}

	hasEncryptedKey := database.DB.Migrator().HasColumn(&models.User{}, "encrypted_key")
	if !hasEncryptedKey {
		t.Error("users table should have 'encrypted_key' column")
	}

	hasCreatedAt := database.DB.Migrator().HasColumn(&models.User{}, "created_at")
	if !hasCreatedAt {
		t.Error("users table should have 'created_at' column")
	}

	hasUpdatedAt := database.DB.Migrator().HasColumn(&models.User{}, "updated_at")
	if !hasUpdatedAt {
		t.Error("users table should have 'updated_at' column")
	}
}

// TestTodoTableSchema tests that Todo table has correct schema
func TestTodoTableSchema(t *testing.T) {
	testutil.SetupTestEnv(t)
	defer testutil.TeardownTestEnv(t)

	// Setup database environment variables
	testutil.SetupTestDatabaseEnv()

	database.Init()

	// Check required columns exist
	hasID := database.DB.Migrator().HasColumn(&models.Todo{}, "id")
	if !hasID {
		t.Error("todos table should have 'id' column")
	}

	hasUserUUID := database.DB.Migrator().HasColumn(&models.Todo{}, "user_uuid")
	if !hasUserUUID {
		t.Error("todos table should have 'user_uuid' column")
	}

	hasTitle := database.DB.Migrator().HasColumn(&models.Todo{}, "title")
	if !hasTitle {
		t.Error("todos table should have 'title' column")
	}

	hasCompleted := database.DB.Migrator().HasColumn(&models.Todo{}, "completed")
	if !hasCompleted {
		t.Error("todos table should have 'completed' column")
	}

	hasTags := database.DB.Migrator().HasColumn(&models.Todo{}, "tags")
	if !hasTags {
		t.Error("todos table should have 'tags' column")
	}

	hasPriority := database.DB.Migrator().HasColumn(&models.Todo{}, "priority")
	if !hasPriority {
		t.Error("todos table should have 'priority' column")
	}

	hasDueDate := database.DB.Migrator().HasColumn(&models.Todo{}, "due_date")
	if !hasDueDate {
		t.Error("todos table should have 'due_date' column")
	}

	hasCreatedAt := database.DB.Migrator().HasColumn(&models.Todo{}, "created_at")
	if !hasCreatedAt {
		t.Error("todos table should have 'created_at' column")
	}

	hasUpdatedAt := database.DB.Migrator().HasColumn(&models.Todo{}, "updated_at")
	if !hasUpdatedAt {
		t.Error("todos table should have 'updated_at' column")
	}
}

// TestUserTableIndexes tests that User table has correct indexes
func TestUserTableIndexes(t *testing.T) {
	testutil.SetupTestEnv(t)
	defer testutil.TeardownTestEnv(t)

	// Setup database environment variables
	testutil.SetupTestDatabaseEnv()

	database.Init()

	// Check unique index on uuid
	hasUUIDIndex := database.DB.Migrator().HasIndex(&models.User{}, "uuid")
	if !hasUUIDIndex {
		t.Error("users table should have unique index on 'uuid'")
	}

	// Check unique index on account_lookup
	hasAccountLookupIndex := database.DB.Migrator().HasIndex(&models.User{}, "account_lookup")
	if !hasAccountLookupIndex {
		t.Error("users table should have unique index on 'account_lookup'")
	}
}

// TestDatabaseConnection tests that database connection is working
func TestDatabaseConnection(t *testing.T) {
	testutil.SetupTestEnv(t)
	defer testutil.TeardownTestEnv(t)

	// Setup database environment variables
	testutil.SetupTestDatabaseEnv()

	database.Init()

	// Test connection by executing a simple query
	var count int64
	if err := database.DB.Model(&models.User{}).Count(&count).Error; err != nil {
		t.Fatalf("Failed to query database: %v", err)
	}

	// Connection is working if query succeeds
	t.Logf("Database connection verified. Current user count: %d", count)
}

// TestDatabaseConstraints tests that database constraints are properly set
func TestDatabaseConstraints(t *testing.T) {
	testutil.SetupTestEnv(t)
	defer testutil.TeardownTestEnv(t)

	// Setup database environment variables
	testutil.SetupTestDatabaseEnv()

	database.Init()

	// Test NOT NULL constraint on account_lookup
	// This should fail if constraint is not set
	user := &models.User{
		UUID: "test-uuid-123456789012",
		// AccountLookup is missing - should fail
		AccountHash:   "test-hash",
		EncryptedKey:  "test-key",
	}

	err := database.DB.Create(user).Error
	if err == nil {
		t.Error("Creating user without account_lookup should fail (NOT NULL constraint)")
		// Clean up if somehow created
		database.DB.Delete(user)
	} else {
		t.Logf("Expected error when creating user without account_lookup: %v", err)
	}

	// Test NOT NULL constraint on account_hash
	user2 := &models.User{
		UUID:         "test-uuid-987654321098",
		AccountLookup: "test-lookup",
		// AccountHash is missing - should fail
		EncryptedKey: "test-key",
	}

	err = database.DB.Create(user2).Error
	if err == nil {
		t.Error("Creating user without account_hash should fail (NOT NULL constraint)")
		// Clean up if somehow created
		database.DB.Delete(user2)
	} else {
		t.Logf("Expected error when creating user without account_hash: %v", err)
	}
}

// TestDatabaseEncryptionKey tests that encryption key is initialized
func TestDatabaseEncryptionKey(t *testing.T) {
	testutil.SetupTestEnv(t)
	defer testutil.TeardownTestEnv(t)

	// Setup database environment variables
	testutil.SetupTestDatabaseEnv()

	database.Init()

	// Verify encryption key is set
	if database.EncryptionKey == nil {
		t.Fatal("database.EncryptionKey should not be nil after Init()")
	}

	// Verify encryption key length (should be 32 bytes = 64 hex chars)
	if len(database.EncryptionKey) != 32 {
		t.Errorf("database.EncryptionKey length = %d, want 32", len(database.EncryptionKey))
	}
}
