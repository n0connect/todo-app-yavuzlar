package database

import (
	"encoding/hex"
	"fmt"
	"log"

	"todo-app-backend/internal/config"
	"todo-app-backend/internal/models"
	"todo-app-backend/internal/utils"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var dbLogger = utils.NewLogger("DATABASE")

var (
	DB            *gorm.DB
	EncryptionKey []byte
)

func Init() {
	// Configure GORM logger based on environment
	var gormLogLevel logger.LogLevel
	if config.IsProductionMode() {
		gormLogLevel = logger.Error
	} else {
		gormLogLevel = logger.Info
	}

	dbLogger.Info("Initializing database connection...")
	dbLogger.Info("Initializing does not guarantee your config ise secure generated!")

	// Use config functions for consistency
	dbHost := config.GetDBHost()
	dbUser := config.GetDBUser()
	dbPassword := config.GetDBPassword()
	dbName := config.GetDBName()
	dbPort := config.GetDBPort()
	dbSSLMode := config.GetDBSSLMode()

	dbLogger.Debug("Database configuration: host=%s port=%s dbname=%s user=%s sslmode=%s",
		dbHost, dbPort, dbName, dbUser, dbSSLMode)

	// Be careful special information passed to dns
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
		dbHost, dbUser, dbPassword, dbName, dbPort, dbSSLMode)

	dbLogger.Debug("Attempting to connect to PostgreSQL database...")
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{ // GORM is go special database lib.
		Logger: logger.Default.LogMode(gormLogLevel),
	})
	if err != nil {
		dbLogger.LogError("Database connection", err)
		log.Fatal("Failed to connect to database:", err)
	}
	dbLogger.Info("Successfully connected to PostgreSQL database")

	dbLogger.Debug("Running database migrations...")
	
	// Step 1: Add AccountLookup and AccountHash as nullable first (if they don't exist)
	// This handles existing records gracefully
	if !DB.Migrator().HasColumn(&models.User{}, "account_lookup") {
		dbLogger.Debug("Adding account_lookup column as nullable...")
		if err := DB.Migrator().AddColumn(&models.User{}, "AccountLookup"); err != nil {
			dbLogger.LogError("Failed to add account_lookup column", err)
		}
	}
	if !DB.Migrator().HasColumn(&models.User{}, "account_hash") {
		dbLogger.Debug("Adding account_hash column as nullable...")
		if err := DB.Migrator().AddColumn(&models.User{}, "AccountHash"); err != nil {
			dbLogger.LogError("Failed to add account_hash column", err)
		}
	}
	
	// Step 2: Delete existing users that don't have AccountLookup/AccountHash
	// (These are old users from the UUID-based system and are incompatible)
	dbLogger.Debug("Cleaning up incompatible user records...")
	result := DB.Where("account_lookup IS NULL OR account_hash IS NULL").Delete(&models.User{})
	if result.Error == nil && result.RowsAffected > 0 {
		dbLogger.Info("Deleted %d incompatible user records (missing AccountLookup/AccountHash)", result.RowsAffected)
	}
	
	// Step 3: Now run AutoMigrate to set NOT NULL constraints and indexes
	err = DB.AutoMigrate(&models.User{}, &models.Todo{}) // &models.Requests{} not imported!
	if err != nil {
		dbLogger.LogError("Database migration", err)
		log.Fatal("Failed to migrate database:", err)
	}
	dbLogger.Info("Database migrations completed successfully!")

	// Initialize encryption key (NOT ACCEPTABLY)
	initEncryptionKey()

	dbLogger.Info("Database initialization completed successfully")
}

// initEncryptionKey loads and validates the master encryption key
// In production mode not use this method. Do not store raw ENCRYPTION_KEY in environment.
// Update master key length 32 to -> 64 hex and update secure key management architecture.
func initEncryptionKey() {
	keyStr := config.GetEncryptionKey()
	if keyStr == "" {
		dbLogger.Error("ENCRYPTION_KEY environment variable is missing")
		log.Fatal("ENCRYPTION_KEY environment variable is required (32 bytes / 64 hex characters)")
	}

	dbLogger.Debug("Loading encryption key from environment...")

	// Try hex format first (64 hex chars = 32 bytes)
	if decoded, err := hex.DecodeString(keyStr); err == nil && len(decoded) == 32 {
		EncryptionKey = decoded
		dbLogger.Info("Encryption key loaded successfully (hex format, 32 bytes)")
		return
	}

	// Try raw format (32 bytes)
	if len(keyStr) == 32 {
		EncryptionKey = []byte(keyStr)
		dbLogger.Info("Encryption key loaded successfully (raw format, 32 bytes)")
		return
	}

	dbLogger.Error("Invalid encryption key format: length=%d (expected 32 bytes or 64 hex chars)", len(keyStr))
	log.Fatal("ENCRYPTION_KEY must be 32 bytes (64 hex characters or 32 raw bytes)")
}

// ValidateStartupConfig validates all required configuration at startup
func ValidateStartupConfig() error {
	dbLogger.Info("Validation not guarantee your [JWT_SECRET] is created by secure source!")
	dbLogger.Info("Recommendation use openssl with isolated system for high entropy clean source.")
	// Validate JWT secret
	jwtSecret := config.GetJWTSecret()
	if jwtSecret == "" {
		return fmt.Errorf("JWT_SECRET is required please check .env file")
	}
	if len(jwtSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters. If you dont have key plese use openssl for create high entropy CSPRNG for create JWT_SECRET")
	}

	// Validate encryption key
	dbLogger.Info("Validation not guarantee your [ENCRYPTION_KEY] is created by secure source!")
	dbLogger.Info("Recommendation use openssl with isolated system for high entropy clean source.")
	encKey := config.GetEncryptionKey()
	if encKey == "" {
		return fmt.Errorf("ENCRYPTION_KEY is required")
	}

	// Validate ACCOUNT_LOOKUP_PEPPER
	dbLogger.Info("Validation not guarantee your [ACCOUNT_LOOKUP_PEPPER] is created by secure source!")
	dbLogger.Info("Recommendation use openssl with isolated system for high entropy clean source.")
	pepper := config.GetAccountLookupPepper()
	if pepper == nil || len(pepper) < 32 {
		return fmt.Errorf("ACCOUNT_LOOKUP_PEPPER is required and must be at least 32 bytes (can be hex or base64 encoded). Use: openssl rand -hex 32")
	}
	dbLogger.Info("ACCOUNT_LOOKUP_PEPPER validated: length=%d bytes", len(pepper))

	// SECURITY: In production, validate stricter settings
	if config.IsProductionMode() {
		// Validate CORS is not wildcard
		if config.GetAllowedOrigin() == "*" {
			return fmt.Errorf("ALLOWED_ORIGIN cannot be '*' in production mode")
		}

		// Validate SSL mode
		if config.GetDBSSLMode() == "disable" {
			return fmt.Errorf("DB_SSL_MODE cannot be 'disable' in production mode")
		}

		// Validate log level (not debug in production)
		if config.GetLogLevel() == "debug" {
			dbLogger.Warn("LOG_LEVEL is set to 'debug' in production mode - consider using 'info' or higher")
		}

		dbLogger.Info("Production mode validation passed")
	}

	return nil
}
