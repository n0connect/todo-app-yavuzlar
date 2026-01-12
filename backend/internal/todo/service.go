package todo

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"todo-app-backend/internal/encryption"
	"todo-app-backend/internal/models"
	"todo-app-backend/internal/store"
	"todo-app-backend/internal/utils"
)

// TodoDTO represents a todo in the service layer
type TodoDTO struct {
	ID        string
	Title     string // Plain text - frontend uses textContent for XSS safety
	Completed bool
	Tags      []string
	DueDate   *time.Time
	Priority  string
	CreatedAt time.Time
}

// TodoService handles business logic for todos
type TodoService struct {
	todoRepo store.TodoRepository
	userRepo store.UserRepository
	logger   *utils.Logger
}

// NewTodoService creates a new TodoService instance
func NewTodoService() *TodoService {
	return &TodoService{
		todoRepo: store.NewTodoRepository(),
		userRepo: store.NewUserRepository(),
		logger:   utils.NewLogger("TODO_SERVICE"),
	}
}

// GetTodos retrieves all todos for a user
func (s *TodoService) GetTodos(userUUIDStr string) ([]TodoDTO, error) {
	s.logger.Debug("GetTodos: starting for user: %s", userUUIDStr)

	// Get user encryption key via repository
	encryptedKey, err := s.userRepo.GetEncryptedKey(userUUIDStr)
	if err != nil {
		s.logger.LogError("UserRepository.GetEncryptedKey", err)
		return nil, err
	}
	userKey, err := encryption.DecryptUserKey(encryptedKey)
	if err != nil {
		s.logger.LogError("DecryptUserKey", err)
		return nil, err
	}

	// Get todos from repository
	todos, err := s.todoRepo.FindByUserUUID(userUUIDStr)
	if err != nil {
		s.logger.LogError("TodoRepository.FindByUserUUID", err)
		return nil, err
	}

	// Decrypt and transform todos
	var result []TodoDTO
	decryptErrors := 0
	for _, todo := range todos {
		// Decrypt title with binary AAD
		aad := encryption.BuildAAD(userUUIDStr, todo.ID, encryption.FieldTitle, encryption.PurposeStoredRecord)
		decryptedTitle, err := encryption.DecryptWithAAD(todo.Title, userKey, aad)
		if err != nil {
			s.logger.LogError("DecryptWithAAD (title)", err)
			s.logger.Error("SECURITY: Failed to decrypt todo id=%s for user=%s - AAD mismatch or invalid ciphertext", todo.ID.String(), userUUIDStr)
			decryptErrors++
			continue
		}

		// Decrypt tags
		var decryptedTags []string
		if len(todo.Tags) > 0 {
			tagAAD := encryption.BuildAAD(userUUIDStr, todo.ID, encryption.FieldTags, encryption.PurposeStoredRecord)
			for _, encryptedTag := range todo.Tags {
				decryptedTag, err := encryption.DecryptWithAAD(encryptedTag, userKey, tagAAD)
				if err != nil {
					s.logger.LogError("DecryptWithAAD (tag)", err)
					continue
				}
				decryptedTags = append(decryptedTags, decryptedTag)
			}
		}

		// Note: No HTML encoding here - frontend uses textContent which auto-escapes
		result = append(result, TodoDTO{
			ID:        todo.ID.String(),
			Title:     decryptedTitle,
			Completed: todo.Completed,
			Tags:      decryptedTags,
			DueDate:   todo.DueDate,
			Priority:  string(todo.Priority),
			CreatedAt: todo.CreatedAt,
		})
	}

	if decryptErrors > 0 {
		s.logger.Warn("GetTodos: completed with %d decryption errors out of %d todos", decryptErrors, len(todos))
	}

	s.logger.Info("GetTodos: successfully retrieved %d todos for user: %s", len(result), userUUIDStr)
	return result, nil
}

// CreateTodo creates a new todo
func (s *TodoService) CreateTodo(userUUIDStr string, req models.TodoRequest) (*TodoDTO, error) {
	s.logger.Debug("CreateTodo: starting for user: %s", userUUIDStr)

	// Validate title
	validatedTitle, err := utils.ValidateTitle(req.Title)
	if err != nil {
		s.logger.LogError("ValidateTitle", err)
		return nil, err
	}

	// Get user encryption key via repository
	encryptedKey, err := s.userRepo.GetEncryptedKey(userUUIDStr)
	if err != nil {
		s.logger.LogError("UserRepository.GetEncryptedKey", err)
		return nil, err
	}
	userKey, err := encryption.DecryptUserKey(encryptedKey)
	if err != nil {
		s.logger.LogError("DecryptUserKey", err)
		return nil, err
	}

	// Generate todo ID BEFORE encryption (required for AAD)
	todoID := uuid.New()

	// Encrypt title with binary AAD
	aad := encryption.BuildAAD(userUUIDStr, todoID, encryption.FieldTitle, encryption.PurposeStoredRecord)
	encryptedTitle, err := encryption.EncryptWithAAD(validatedTitle, userKey, aad)
	if err != nil {
		s.logger.LogError("EncryptWithAAD", err)
		return nil, err
	}

	// Parse due date if provided
	var dueDate *time.Time
	if req.DueDate != "" {
		parsed, err := time.Parse(time.RFC3339, req.DueDate)
		if err != nil {
			// Try date-only format
			parsed, err = time.Parse("2006-01-02", req.DueDate)
			if err != nil {
				s.logger.LogError("ParseDueDate", err)
				return nil, err
			}
		}
		dueDate = &parsed
	}

	// Validate priority
	priority := models.PriorityMedium
	if req.Priority != "" {
		switch req.Priority {
		case "low":
			priority = models.PriorityLow
		case "high":
			priority = models.PriorityHigh
		default:
			priority = models.PriorityMedium
		}
	}

	// Validate tags (max 10 tags, each max 30 chars) and encrypt
	// SECURITY: Validates UTF-8, null bytes, and truncates by runes (not bytes)
	var encryptedTags pq.StringArray
	var plainTags []string
	tagAAD := encryption.BuildAAD(userUUIDStr, todoID, encryption.FieldTags, encryption.PurposeStoredRecord)
	for i, tag := range req.Tags {
		if i >= 10 {
			break
		}

		// Validate UTF-8 encoding
		if !utf8.ValidString(tag) {
			s.logger.Warn("CreateTodo: invalid UTF-8 tag, skipping")
			continue
		}

		// Check for null bytes (security)
		if strings.Contains(tag, "\x00") {
			s.logger.Warn("CreateTodo: null byte in tag, skipping")
			continue
		}

		// Trim whitespace
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue // Skip empty tags
		}

		// Truncate by runes, not bytes (prevents Unicode corruption)
		runes := []rune(tag)
		if len(runes) > 30 {
			tag = string(runes[:30])
		}

		plainTags = append(plainTags, tag)
		encryptedTag, err := encryption.EncryptWithAAD(tag, userKey, tagAAD)
		if err != nil {
			s.logger.LogError("EncryptWithAAD (tag)", err)
			continue
		}
		encryptedTags = append(encryptedTags, encryptedTag)
	}

	// Create todo in database
	todo := models.Todo{
		ID:        todoID,
		UserUUID:  userUUIDStr,
		Title:     encryptedTitle,
		Completed: req.Completed,
		Tags:      encryptedTags,
		DueDate:   dueDate,
		Priority:  priority,
		CreatedAt: time.Now(),
	}

	if err := s.todoRepo.Create(&todo); err != nil {
		s.logger.LogError("TodoRepository.Create", err)
		return nil, err
	}

	s.logger.Info("CreateTodo: successfully created todo id=%s for user: %s", todoID.String(), userUUIDStr)

	// Return DTO - no HTML encoding, frontend uses textContent
	return &TodoDTO{
		ID:        todo.ID.String(),
		Title:     validatedTitle,
		Completed: todo.Completed,
		Tags:      plainTags,
		DueDate:   todo.DueDate,
		Priority:  string(todo.Priority),
		CreatedAt: todo.CreatedAt,
	}, nil
}

// UpdateTodo updates an existing todo
func (s *TodoService) UpdateTodo(userUUIDStr string, todoIDStr string, req models.TodoRequest) (*TodoDTO, error) {
	s.logger.Debug("UpdateTodo: starting for user: %s, todoID: %s", userUUIDStr, todoIDStr)

	todoID, err := uuid.Parse(todoIDStr)
	if err != nil {
		s.logger.LogError("UUID Parse (todo)", err)
		return nil, err
	}

	// Get todo from repository
	todo, err := s.todoRepo.FindByIDAndUserUUID(todoID, userUUIDStr)
	if err != nil {
		s.logger.LogError("TodoRepository.FindByIDAndUserUUID", err)
		return nil, err
	}

	// Get user encryption key via repository
	encryptedKey, err := s.userRepo.GetEncryptedKey(userUUIDStr)
	if err != nil {
		s.logger.LogError("UserRepository.GetEncryptedKey", err)
		return nil, err
	}
	userKey, err := encryption.DecryptUserKey(encryptedKey)
	if err != nil {
		s.logger.LogError("DecryptUserKey", err)
		return nil, err
	}

	var responseTitle string
	if req.Title != "" {
		// Validate and encrypt new title
		validatedTitle, err := utils.ValidateTitle(req.Title)
		if err != nil {
			s.logger.LogError("ValidateTitle", err)
			return nil, err
		}

		aad := encryption.BuildAAD(userUUIDStr, todoID, encryption.FieldTitle, encryption.PurposeStoredRecord)
		encryptedTitle, err := encryption.EncryptWithAAD(validatedTitle, userKey, aad)
		if err != nil {
			s.logger.LogError("EncryptWithAAD", err)
			return nil, err
		}

		todo.Title = encryptedTitle
		responseTitle = validatedTitle
	} else {
		// Decrypt existing title for response
		aad := encryption.BuildAAD(userUUIDStr, todoID, encryption.FieldTitle, encryption.PurposeStoredRecord)
		decryptedTitle, err := encryption.DecryptWithAAD(todo.Title, userKey, aad)
		if err != nil {
			s.logger.LogError("DecryptWithAAD", err)
			s.logger.Error("SECURITY: Failed to decrypt existing title for todo: %s user: %s", todoIDStr, userUUIDStr)
			return nil, err
		}
		responseTitle = decryptedTitle
	}

	// Update completed status
	todo.Completed = req.Completed

	// Track plain tags for response
	var responseTags []string

	// Update tags if provided - encrypt each tag
	// SECURITY: Validates UTF-8, null bytes, and truncates by runes (not bytes)
	if req.Tags != nil {
		var encryptedTags pq.StringArray
		tagAAD := encryption.BuildAAD(userUUIDStr, todoID, encryption.FieldTags, encryption.PurposeStoredRecord)
		for i, tag := range req.Tags {
			if i >= 10 {
				break
			}

			// Validate UTF-8 encoding
			if !utf8.ValidString(tag) {
				s.logger.Warn("UpdateTodo: invalid UTF-8 tag, skipping")
				continue
			}

			// Check for null bytes (security)
			if strings.Contains(tag, "\x00") {
				s.logger.Warn("UpdateTodo: null byte in tag, skipping")
				continue
			}

			// Trim whitespace
			tag = strings.TrimSpace(tag)
			if tag == "" {
				continue // Skip empty tags
			}

			// Truncate by runes, not bytes (prevents Unicode corruption)
			runes := []rune(tag)
			if len(runes) > 30 {
				tag = string(runes[:30])
			}

			responseTags = append(responseTags, tag)
			encryptedTag, err := encryption.EncryptWithAAD(tag, userKey, tagAAD)
			if err != nil {
				s.logger.LogError("EncryptWithAAD (tag)", err)
				continue
			}
			encryptedTags = append(encryptedTags, encryptedTag)
		}
		todo.Tags = encryptedTags
	} else {
		// Decrypt existing tags for response
		if len(todo.Tags) > 0 {
			tagAAD := encryption.BuildAAD(userUUIDStr, todoID, encryption.FieldTags, encryption.PurposeStoredRecord)
			for _, encryptedTag := range todo.Tags {
				decryptedTag, err := encryption.DecryptWithAAD(encryptedTag, userKey, tagAAD)
				if err != nil {
					s.logger.LogError("DecryptWithAAD (tag)", err)
					continue
				}
				responseTags = append(responseTags, decryptedTag)
			}
		}
	}

	// Update due date if provided
	if req.DueDate != "" {
		parsed, err := time.Parse(time.RFC3339, req.DueDate)
		if err != nil {
			parsed, err = time.Parse("2006-01-02", req.DueDate)
			if err != nil {
				s.logger.LogError("ParseDueDate", err)
				return nil, err
			}
		}
		todo.DueDate = &parsed
	}

	// Update priority if provided
	if req.Priority != "" {
		switch req.Priority {
		case "low":
			todo.Priority = models.PriorityLow
		case "high":
			todo.Priority = models.PriorityHigh
		default:
			todo.Priority = models.PriorityMedium
		}
	}

	// Save to database
	if err := s.todoRepo.Update(todo); err != nil {
		s.logger.LogError("TodoRepository.Update", err)
		return nil, err
	}

	s.logger.Info("UpdateTodo: successfully updated todo id=%s for user: %s", todoIDStr, userUUIDStr)

	// Return DTO - no HTML encoding, frontend uses textContent
	return &TodoDTO{
		ID:        todo.ID.String(),
		Title:     responseTitle,
		Completed: todo.Completed,
		Tags:      responseTags,
		DueDate:   todo.DueDate,
		Priority:  string(todo.Priority),
		CreatedAt: todo.CreatedAt,
	}, nil
}

// DeleteTodo deletes a todo
func (s *TodoService) DeleteTodo(userUUIDStr string, todoIDStr string) error {
	s.logger.Debug("DeleteTodo: starting for user: %s, todoID: %s", userUUIDStr, todoIDStr)

	todoID, err := uuid.Parse(todoIDStr)
	if err != nil {
		s.logger.LogError("UUID Parse", err)
		return err
	}

	rowsAffected, err := s.todoRepo.DeleteByIDAndUserUUID(todoID, userUUIDStr)
	if err != nil {
		s.logger.LogError("TodoRepository.DeleteByIDAndUserUUID", err)
		return err
	}

	if rowsAffected == 0 {
		s.logger.Warn("DeleteTodo: todo not found id=%s for user: %s", todoIDStr, userUUIDStr)
		return store.ErrNotFound
	}

	s.logger.Info("DeleteTodo: successfully deleted todo id=%s for user: %s", todoIDStr, userUUIDStr)
	return nil
}
