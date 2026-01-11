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
	dbLogger.Info("Initializing database connection...")

	// Use config functions for consistency
	dbHost := config.GetDBHost()
	dbUser := config.GetDBUser()
	dbPassword := config.GetDBPassword()
	dbName := config.GetDBName()
	dbPort := config.GetDBPort()
	dbSSLMode := config.GetDBSSLMode()

	dbLogger.Debug("Database configuration: host=%s port=%s dbname=%s user=%s sslmode=%s",
		dbHost, dbPort, dbName, dbUser, dbSSLMode)

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
		dbHost, dbUser, dbPassword, dbName, dbPort, dbSSLMode)

	// Configure GORM logger based on environment
	var gormLogLevel logger.LogLevel
	if config.IsProductionMode() {
		gormLogLevel = logger.Error
	} else {
		gormLogLevel = logger.Info
	}

	dbLogger.Debug("Attempting to connect to PostgreSQL database...")
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(gormLogLevel),
	})
	if err != nil {
		dbLogger.LogError("Database connection", err)
		log.Fatal("Failed to connect to database:", err)
	}
	dbLogger.Info("Successfully connected to PostgreSQL database")

	dbLogger.Debug("Running database migrations...")
	err = DB.AutoMigrate(&models.User{}, &models.Todo{})
	if err != nil {
		dbLogger.LogError("Database migration", err)
		log.Fatal("Failed to migrate database:", err)
	}
	dbLogger.Info("Database migrations completed successfully")

	// Initialize encryption key
	initEncryptionKey()

	dbLogger.Info("Database initialization completed successfully")
}

// initEncryptionKey loads and validates the master encryption key
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
	// Validate JWT secret
	jwtSecret := config.GetJWTSecret()
	if jwtSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	if len(jwtSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}

	// Validate encryption key
	encKey := config.GetEncryptionKey()
	if encKey == "" {
		return fmt.Errorf("ENCRYPTION_KEY is required")
	}

	// In production, validate stricter settings
	if config.IsProductionMode() {
		origin := config.GetAllowedOrigin()
		if origin == "*" {
			dbLogger.Warn("SECURITY WARNING: ALLOWED_ORIGIN is set to '*' in production mode")
		}
	}

	return nil
}
