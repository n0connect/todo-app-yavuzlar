package store

import (
	"errors"
	"todo-app-backend/internal/database"
	"todo-app-backend/internal/models"
	"todo-app-backend/internal/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrNotFound    = errors.New("not found")
	todoRepoLogger = utils.NewLogger("TODO_REPOSITORY")
)

// TodoRepository defines the interface for todo data access
type TodoRepository interface {
	FindByUserUUID(userUUID string) ([]models.Todo, error)
	FindByIDAndUserUUID(id uuid.UUID, userUUID string) (*models.Todo, error)
	Create(todo *models.Todo) error
	Update(todo *models.Todo) error
	DeleteByIDAndUserUUID(id uuid.UUID, userUUID string) (int64, error)
}

// todoRepository implements TodoRepository
type todoRepository struct{}

// NewTodoRepository creates a new TodoRepository instance
func NewTodoRepository() TodoRepository {
	todoRepoLogger.Debug("NewTodoRepository: creating new TodoRepository instance")
	return &todoRepository{}
}

// FindByUserUUID finds all todos for a user, ordered by created_at DESC
func (r *todoRepository) FindByUserUUID(userUUID string) ([]models.Todo, error) {
	todoRepoLogger.Debug("TodoRepository.FindByUserUUID: starting query for userUUID: %s", userUUID)
	var todos []models.Todo
	result := database.DB.Where("user_uuid = ?", userUUID).Order("created_at DESC").Find(&todos)
	if result.Error != nil {
		todoRepoLogger.LogError("TodoRepository.FindByUserUUID", result.Error)
		todoRepoLogger.Debug("TodoRepository.FindByUserUUID: query failed for userUUID: %s", userUUID)
		return nil, result.Error
	}
	todoRepoLogger.Debug("TodoRepository.FindByUserUUID: successfully found %d todos for userUUID: %s", len(todos), userUUID)
	return todos, nil
}

// FindByIDAndUserUUID finds a todo by ID and user UUID (ensures ownership)
func (r *todoRepository) FindByIDAndUserUUID(id uuid.UUID, userUUID string) (*models.Todo, error) {
	todoRepoLogger.Debug("TodoRepository.FindByIDAndUserUUID: starting query for todoID: %s userUUID: %s", id.String(), userUUID)
	var todo models.Todo
	result := database.DB.Where("id = ? AND user_uuid = ?", id, userUUID).First(&todo)
	if result.Error != nil {
		todoRepoLogger.LogError("TodoRepository.FindByIDAndUserUUID", result.Error)
		todoRepoLogger.Debug("TodoRepository.FindByIDAndUserUUID: todo not found for todoID: %s userUUID: %s", id.String(), userUUID)
		// Wrap GORM's ErrRecordNotFound as ErrNotFound for consistent error handling
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, result.Error
	}
	todoRepoLogger.Debug("TodoRepository.FindByIDAndUserUUID: successfully found todo: todoID=%s userUUID=%s", id.String(), userUUID)
	return &todo, nil
}

// Create creates a new todo
func (r *todoRepository) Create(todo *models.Todo) error {
	todoRepoLogger.Debug("TodoRepository.Create: starting create operation for todoID: %s userUUID: %s", todo.ID.String(), todo.UserUUID)
	err := database.DB.Create(todo).Error
	if err != nil {
		todoRepoLogger.LogError("TodoRepository.Create", err)
		todoRepoLogger.Debug("TodoRepository.Create: failed to create todo: todoID=%s userUUID=%s", todo.ID.String(), todo.UserUUID)
		return err
	}
	todoRepoLogger.Debug("TodoRepository.Create: successfully created todo: todoID=%s userUUID=%s", todo.ID.String(), todo.UserUUID)
	return nil
}

// Update updates an existing todo with ownership verification
// Only updates if the todo belongs to the specified user (prevents IDOR)
func (r *todoRepository) Update(todo *models.Todo) error {
	todoRepoLogger.Debug("TodoRepository.Update: starting update operation for todoID: %s userUUID: %s", todo.ID.String(), todo.UserUUID)

	// Use WHERE clause with both ID and UserUUID to ensure ownership
	result := database.DB.Model(&models.Todo{}).
		Where("id = ? AND user_uuid = ?", todo.ID, todo.UserUUID).
		Updates(map[string]interface{}{
			"title":     todo.Title,
			"completed": todo.Completed,
			"tags":      todo.Tags,
			"due_date":  todo.DueDate,
			"priority":  todo.Priority,
		})

	if result.Error != nil {
		todoRepoLogger.LogError("TodoRepository.Update", result.Error)
		todoRepoLogger.Debug("TodoRepository.Update: failed to update todo: todoID=%s userUUID=%s", todo.ID.String(), todo.UserUUID)
		return result.Error
	}

	if result.RowsAffected == 0 {
		todoRepoLogger.Warn("TodoRepository.Update: no rows affected - ownership check failed or todo not found: todoID=%s userUUID=%s", todo.ID.String(), todo.UserUUID)
		return ErrNotFound
	}

	todoRepoLogger.Debug("TodoRepository.Update: successfully updated todo: todoID=%s userUUID=%s", todo.ID.String(), todo.UserUUID)
	return nil
}

// DeleteByIDAndUserUUID deletes a todo by ID and user UUID (ensures ownership)
// Returns the number of rows affected
func (r *todoRepository) DeleteByIDAndUserUUID(id uuid.UUID, userUUID string) (int64, error) {
	todoRepoLogger.Debug("TodoRepository.DeleteByIDAndUserUUID: starting delete operation for todoID: %s userUUID: %s", id.String(), userUUID)
	result := database.DB.Where("id = ? AND user_uuid = ?", id, userUUID).Delete(&models.Todo{})
	if result.Error != nil {
		todoRepoLogger.LogError("TodoRepository.DeleteByIDAndUserUUID", result.Error)
		todoRepoLogger.Debug("TodoRepository.DeleteByIDAndUserUUID: delete failed for todoID: %s userUUID: %s", id.String(), userUUID)
		return 0, result.Error
	}
	rowsAffected := result.RowsAffected
	todoRepoLogger.Debug("TodoRepository.DeleteByIDAndUserUUID: successfully deleted todo: todoID=%s userUUID=%s rowsAffected=%d", id.String(), userUUID, rowsAffected)
	return rowsAffected, nil
}
