package database

import (
	"fmt"
	"log"
	"strings"

	"todo-app-backend/internal/config"
	"todo-app-backend/internal/encryption"
	"todo-app-backend/internal/models"
	"todo-app-backend/internal/utils"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var dbLogger = utils.NewLogger("DATABASE")

var (
	DB *gorm.DB
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

	if err := encryption.InitMasterKeys(); err != nil {
		dbLogger.LogError("Master key initialization", err)
		log.Fatal("Failed to initialize master keys:", err)
	}

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
	if !DB.Migrator().HasColumn(&models.User{}, "master_key_id") {
		dbLogger.Debug("Adding master_key_id column as nullable...")
		if err := DB.Migrator().AddColumn(&models.User{}, "MasterKeyID"); err != nil {
			dbLogger.LogError("Failed to add master_key_id column", err)
		}
	}

	// Step 2: Delete existing users that don't have AccountLookup/AccountHash
	// (These are old users from the UUID-based system and are incompatible)
	dbLogger.Debug("Cleaning up incompatible user records...")
	result := DB.Where("account_lookup IS NULL OR account_hash IS NULL").Delete(&models.User{})
	if result.Error == nil && result.RowsAffected > 0 {
		dbLogger.Info("Deleted %d incompatible user records (missing AccountLookup/AccountHash)", result.RowsAffected)
	}

	// Step 2b: Backfill new crypto columns for existing users
	activeID := encryption.ActiveMasterKeyID()
	if err := DB.Model(&models.User{}).
		Where("master_key_id IS NULL OR master_key_id = ''").
		Update("master_key_id", activeID).Error; err != nil {
		dbLogger.LogError("Backfill master_key_id", err)
	}
	// Step 3: Now run AutoMigrate to set NOT NULL constraints and indexes
	// Note: models.Requests is not a database model (only for HTTP request/response DTOs)
	err = DB.AutoMigrate(&models.User{}, &models.Todo{})
	if err != nil {
		dbLogger.LogError("Database migration", err)
		log.Fatal("Failed to migrate database:", err)
	}
	dbLogger.Info("Database migrations completed successfully!")

	dbLogger.Info("Database initialization completed successfully")
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
	minJWTLen := config.GetJWTSecretMinLength()
	if minJWTLen <= 0 {
		return fmt.Errorf("JWT_SECRET_MIN_LEN is required")
	}
	if len(jwtSecret) < minJWTLen {
		return fmt.Errorf("JWT_SECRET must be at least %d characters. If you dont have key plese use openssl for create high entropy CSPRNG for create JWT_SECRET", minJWTLen)
	}

	if config.GetJWTIssuer() == "" {
		return fmt.Errorf("JWT_ISSUER is required")
	}
	if config.GetJWTAudience() == "" {
		return fmt.Errorf("JWT_AUDIENCE is required")
	}
	if config.GetJWTExpiration() <= 0 {
		return fmt.Errorf("JWT_EXPIRATION_MINUTES is required and must be > 0")
	}

	// Validate encryption key
	dbLogger.Info("Validation not guarantee your [MASTER_KEY_ACTIVE] is created by secure source!")
	dbLogger.Info("Recommendation use openssl with isolated system for high entropy clean source.")
	if err := encryption.InitMasterKeys(); err != nil {
		return err
	}

	// Validate ACCOUNT_LOOKUP_PEPPER
	dbLogger.Info("Validation not guarantee your [ACCOUNT_LOOKUP_PEPPER] is created by secure source!")
	dbLogger.Info("Recommendation use openssl with isolated system for high entropy clean source.")
	pepper := config.GetAccountLookupPepper()
	if pepper == nil || len(pepper) < 32 {
		return fmt.Errorf("ACCOUNT_LOOKUP_PEPPER is required and must be at least 32 bytes (can be hex or base64 encoded). Use: openssl rand -hex 32")
	}
	dbLogger.Info("ACCOUNT_LOOKUP_PEPPER validated: length=%d bytes", len(pepper))

	// Validate base64 payload limits
	if config.GetMaxBase64LoginLen() <= 0 {
		return fmt.Errorf("MAX_BASE64_LOGIN_LEN is required and must be > 0")
	}
	if config.GetMaxBase64TodoLen() <= 0 {
		return fmt.Errorf("MAX_BASE64_TODO_LEN is required and must be > 0")
	}

	// Validate request limits
	if config.GetMaxRequestBodyBytes() <= 0 {
		return fmt.Errorf("MAX_REQUEST_BODY_BYTES is required and must be > 0")
	}
	if config.GetMaxJSONDepth() <= 0 {
		return fmt.Errorf("MAX_JSON_DEPTH is required and must be > 0")
	}
	if config.GetMaxTitleLength() <= 0 || config.GetMinTitleLength() <= 0 {
		return fmt.Errorf("MAX_TITLE_LENGTH and MIN_TITLE_LENGTH are required and must be > 0")
	}
	if config.GetMinTitleLength() > config.GetMaxTitleLength() {
		return fmt.Errorf("MIN_TITLE_LENGTH cannot exceed MAX_TITLE_LENGTH")
	}
	if config.GetMaxTagLength() <= 0 {
		return fmt.Errorf("MAX_TAG_LENGTH is required and must be > 0")
	}
	if config.GetMaxTagsPerTodo() <= 0 {
		return fmt.Errorf("MAX_TAGS_PER_TODO is required and must be > 0")
	}
	if config.GetInternalIDLength() <= 0 {
		return fmt.Errorf("INTERNAL_ID_LENGTH is required and must be > 0")
	}

	// Validate Argon2id parameters
	if config.GetArgon2MemoryKiB() <= 0 || config.GetArgon2Time() <= 0 || config.GetArgon2Parallelism() <= 0 ||
		config.GetArgon2SaltLength() <= 0 || config.GetArgon2HashLength() <= 0 {
		return fmt.Errorf("ARGON2_* parameters are required and must be > 0")
	}

	// Validate pending registration config
	if config.GetPendingTokenTTL() <= 0 {
		return fmt.Errorf("PENDING_TOKEN_TTL_SEC is required and must be > 0")
	}
	if config.GetPendingCleanupInterval() <= 0 {
		return fmt.Errorf("PENDING_CLEANUP_INTERVAL_SEC is required and must be > 0")
	}
	if config.GetPendingIDBytes() <= 0 {
		return fmt.Errorf("PENDING_ID_BYTES is required and must be > 0")
	}

	// Validate rate limiting config
	if config.GetRateLimitMaxTokens() <= 0 || config.GetRateLimitRefillIntervalSeconds() <= 0 ||
		config.GetRateLimitCleanupIntervalSeconds() <= 0 || config.GetRateLimitMaxBuckets() <= 0 {
		return fmt.Errorf("RATE_LIMIT_* configuration is required and must be > 0")
	}

	// Validate server configuration
	if config.GetBackendPort() == "" {
		return fmt.Errorf("BACKEND_PORT is required")
	}
	if config.GetServerReadHeaderTimeout() <= 0 || config.GetServerReadTimeout() <= 0 ||
		config.GetServerWriteTimeout() <= 0 || config.GetServerIdleTimeout() <= 0 {
		return fmt.Errorf("SERVER_*_TIMEOUT_SEC values are required and must be > 0")
	}
	if config.GetServerMaxHeaderBytes() <= 0 {
		return fmt.Errorf("SERVER_MAX_HEADER_BYTES is required and must be > 0")
	}

	// Validate database config
	if config.GetDBHost() == "" || config.GetDBPort() == "" || config.GetDBUser() == "" ||
		config.GetDBPassword() == "" || config.GetDBName() == "" || config.GetDBSSLMode() == "" {
		return fmt.Errorf("DB_* configuration is required")
	}

	// Validate CORS config
	if strings.TrimSpace(config.GetAllowedOrigins()) == "" {
		return fmt.Errorf("ALLOWED_ORIGINS is required")
	}

	// SECURITY: In production, validate stricter settings
	if config.IsProductionMode() {
		// Validate CORS is not wildcard
		if config.GetAllowedOrigins() == "*" {
			return fmt.Errorf("ALLOWED_ORIGINS cannot be '*' in production mode")
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
