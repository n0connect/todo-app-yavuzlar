package store

import (
	"errors"
	"fmt"

	"todo-app-backend/internal/database"
	"todo-app-backend/internal/models"
	"todo-app-backend/internal/utils"

	"gorm.io/gorm"
)

var userRepoLogger = utils.NewLogger("USER_REPOSITORY")

// UserRepository defines the interface for user data access
type UserRepository interface {
	FindByUUID(uuid string) (*models.User, error)
	FindByAccountLookup(lookup string) (*models.User, error)
	Create(user *models.User) error
	ExistsByUUID(uuid string) (bool, error)
	ExistsByAccountLookup(lookup string) (bool, error)
	GetEncryptedKey(uuid string) (string, error)
}

// userRepository implements UserRepository
type userRepository struct{}

// NewUserRepository creates a new UserRepository instance
func NewUserRepository() UserRepository {
	userRepoLogger.Debug("NewUserRepository: creating new UserRepository instance")
	return &userRepository{}
}

// FindByUUID finds a user by UUID
func (r *userRepository) FindByUUID(uuid string) (*models.User, error) {
	userRepoLogger.Debug("UserRepository.FindByUUID: starting query for UUID: %s", uuid)
	var user models.User
	result := database.DB.Where("uuid = ?", uuid).First(&user)
	if result.Error != nil {
		userRepoLogger.LogError("UserRepository.FindByUUID", result.Error)
		userRepoLogger.Debug("UserRepository.FindByUUID: user not found for UUID: %s", uuid)
		// Wrap GORM's ErrRecordNotFound as ErrNotFound for consistent error handling
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, result.Error
	}
	userRepoLogger.Debug("UserRepository.FindByUUID: successfully found user: UUID=%s", uuid)
	return &user, nil
}

// Create creates a new user
func (r *userRepository) Create(user *models.User) error {
	userRepoLogger.Debug("UserRepository.Create: starting create operation for UUID: %s", user.UUID)
	err := database.DB.Create(user).Error
	if err != nil {
		userRepoLogger.LogError("UserRepository.Create", err)
		userRepoLogger.Debug("UserRepository.Create: failed to create user: UUID=%s", user.UUID)
		return err
	}
	userRepoLogger.Debug("UserRepository.Create: successfully created user: UUID=%s", user.UUID)
	return nil
}

// ExistsByUUID checks if a user exists with the given UUID
func (r *userRepository) ExistsByUUID(uuid string) (bool, error) {
	userRepoLogger.Debug("UserRepository.ExistsByUUID: checking existence for UUID: %s", uuid)
	var count int64
	result := database.DB.Model(&models.User{}).Where("uuid = ?", uuid).Count(&count)
	if result.Error != nil {
		userRepoLogger.LogError("UserRepository.ExistsByUUID", result.Error)
		userRepoLogger.Debug("UserRepository.ExistsByUUID: query failed for UUID: %s", uuid)
		return false, result.Error
	}
	exists := count > 0
	userRepoLogger.Debug("UserRepository.ExistsByUUID: UUID=%s exists=%v", uuid, exists)
	return exists, nil
}

// FindByAccountLookup finds a user by account lookup (HMAC)
func (r *userRepository) FindByAccountLookup(lookup string) (*models.User, error) {
	userRepoLogger.Debug("UserRepository.FindByAccountLookup: starting query for lookup")
	var user models.User
	result := database.DB.Where("account_lookup = ?", lookup).First(&user)
	if result.Error != nil {
		userRepoLogger.LogError("UserRepository.FindByAccountLookup", result.Error)
		userRepoLogger.Debug("UserRepository.FindByAccountLookup: user not found")
		// Wrap GORM's ErrRecordNotFound as ErrNotFound for consistent error handling
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, result.Error
	}
	userRepoLogger.Debug("UserRepository.FindByAccountLookup: successfully found user")
	return &user, nil
}

// ExistsByAccountLookup checks if a user exists with the given account lookup
func (r *userRepository) ExistsByAccountLookup(lookup string) (bool, error) {
	userRepoLogger.Debug("UserRepository.ExistsByAccountLookup: checking existence for lookup")
	var count int64
	result := database.DB.Model(&models.User{}).Where("account_lookup = ?", lookup).Count(&count)
	if result.Error != nil {
		userRepoLogger.LogError("UserRepository.ExistsByAccountLookup", result.Error)
		userRepoLogger.Debug("UserRepository.ExistsByAccountLookup: query failed")
		return false, result.Error
	}
	exists := count > 0
	userRepoLogger.Debug("UserRepository.ExistsByAccountLookup: exists=%v", exists)
	return exists, nil
}

// GetEncryptedKey retrieves the encrypted key for a user
func (r *userRepository) GetEncryptedKey(uuid string) (string, error) {
	userRepoLogger.Debug("UserRepository.GetEncryptedKey: retrieving encrypted key for UUID: %s", uuid)
	user, err := r.FindByUUID(uuid)
	if err != nil {
		userRepoLogger.LogError("UserRepository.GetEncryptedKey", err)
		userRepoLogger.Debug("UserRepository.GetEncryptedKey: user not found for UUID: %s", uuid)
		return "", err
	}
	if user.EncryptedKey == "" {
		userRepoLogger.Error("UserRepository.GetEncryptedKey: user has no encryption key: UUID=%s", uuid)
		return "", fmt.Errorf("user has no encryption key")
	}
	userRepoLogger.Debug("UserRepository.GetEncryptedKey: successfully retrieved encrypted key for UUID: %s length=%d", uuid, len(user.EncryptedKey))
	return user.EncryptedKey, nil
}
