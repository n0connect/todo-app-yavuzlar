package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"todo-app-backend/internal/auth"
	"todo-app-backend/internal/models"
	"todo-app-backend/internal/store"
	"todo-app-backend/internal/todo"
	"todo-app-backend/internal/utils"
)

var (
	todoLogger  = utils.NewLogger("TODOS")
	todoService = todo.NewTodoService()
)

func GetTodosHandler(w http.ResponseWriter, r *http.Request) {
	userUUIDStr, ok := auth.GetUserUUID(r.Context())
	if !ok {
		todoLogger.Error("GetTodosHandler: user UUID not found in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	todoLogger.LogRequest(r.Method, r.URL.Path, userUUIDStr)
	todoLogger.Debug("Starting GetTodosHandler for user: %s", userUUIDStr)

	// Call service layer
	todos, err := todoService.GetTodos(userUUIDStr)
	if err != nil {
		todoLogger.LogError("TodoService.GetTodos", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Transform DTOs to response format
	response := make([]map[string]interface{}, len(todos))
	for i, t := range todos {
		item := map[string]interface{}{
			"id":         t.ID,
			"title":      t.Title,
			"completed":  t.Completed,
			"tags":       t.Tags,
			"priority":   t.Priority,
			"created_at": t.CreatedAt.Format(time.RFC3339),
		}
		if t.DueDate != nil {
			item["due_date"] = t.DueDate.Format(time.RFC3339)
		}
		response[i] = item
	}

	utils.EncodeJSONResponse(w, response, http.StatusOK)
	todoLogger.LogResponse(http.StatusOK, "GetTodosHandler completed successfully")
}

func CreateTodoHandler(w http.ResponseWriter, r *http.Request) {
	userUUIDStr, ok := auth.GetUserUUID(r.Context())
	if !ok {
		todoLogger.Error("CreateTodoHandler: user UUID not found in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	todoLogger.LogRequest(r.Method, r.URL.Path, userUUIDStr)
	todoLogger.Debug("Starting CreateTodoHandler for user: %s", userUUIDStr)

	var req models.TodoRequest
	if err := utils.DecodeJSONRequest(r, &req); err != nil {
		todoLogger.LogError("DecodeJSONRequest", err)
		todoLogger.Warn("Invalid JSON request from user: %s", userUUIDStr)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Call service layer
	todoDTO, err := todoService.CreateTodo(userUUIDStr, req)
	if err != nil {
		todoLogger.LogError("TodoService.CreateTodo", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Transform DTO to response format
	response := map[string]interface{}{
		"id":         todoDTO.ID,
		"title":      todoDTO.Title,
		"completed":  todoDTO.Completed,
		"tags":       todoDTO.Tags,
		"priority":   todoDTO.Priority,
		"created_at": todoDTO.CreatedAt.Format(time.RFC3339),
	}
	if todoDTO.DueDate != nil {
		response["due_date"] = todoDTO.DueDate.Format(time.RFC3339)
	}

	utils.EncodeJSONResponse(w, response, http.StatusCreated)
	todoLogger.LogResponse(http.StatusCreated, "CreateTodoHandler completed successfully")
}

func UpdateTodoHandler(w http.ResponseWriter, r *http.Request) {
	userUUIDStr, ok := auth.GetUserUUID(r.Context())
	if !ok {
		todoLogger.Error("UpdateTodoHandler: user UUID not found in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	todoLogger.LogRequest(r.Method, r.URL.Path, userUUIDStr)

	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/todos/")
	_, err := uuid.Parse(idStr)
	if err != nil {
		todoLogger.LogError("UUID Parse", err)
		todoLogger.Warn("Invalid UUID format in request: %s from user: %s", idStr, userUUIDStr)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	var updateData models.TodoRequest
	if err := utils.DecodeJSONRequest(r, &updateData); err != nil {
		todoLogger.LogError("DecodeJSONRequest", err)
		todoLogger.Warn("Invalid JSON request from user: %s for todo: %s", userUUIDStr, idStr)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Call service layer
	todoDTO, err := todoService.UpdateTodo(userUUIDStr, idStr, updateData)
	if err != nil {
		if err == store.ErrNotFound {
			// SECURITY: Generic error message to prevent enumeration
			// Don't distinguish between "todo doesn't exist" and "todo belongs to another user"
			todoLogger.Warn("UpdateTodoHandler: todo not found id=%s for user: %s", idStr, userUUIDStr)
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}
		todoLogger.LogError("TodoService.UpdateTodo", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Transform DTO to response format
	response := map[string]interface{}{
		"id":         todoDTO.ID,
		"title":      todoDTO.Title,
		"completed":  todoDTO.Completed,
		"tags":       todoDTO.Tags,
		"priority":   todoDTO.Priority,
		"created_at": todoDTO.CreatedAt.Format(time.RFC3339),
	}
	if todoDTO.DueDate != nil {
		response["due_date"] = todoDTO.DueDate.Format(time.RFC3339)
	}

	utils.EncodeJSONResponse(w, response, http.StatusOK)
	todoLogger.LogResponse(http.StatusOK, "UpdateTodoHandler completed successfully")
}

func DeleteTodoHandler(w http.ResponseWriter, r *http.Request) {
	userUUIDStr, ok := auth.GetUserUUID(r.Context())
	if !ok {
		todoLogger.Error("DeleteTodoHandler: user UUID not found in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	todoLogger.LogRequest(r.Method, r.URL.Path, userUUIDStr)

	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/todos/")
	_, err := uuid.Parse(idStr)
	if err != nil {
		todoLogger.LogError("UUID Parse", err)
		todoLogger.Warn("Invalid UUID format in request: %s from user: %s", idStr, userUUIDStr)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Call service layer
	err = todoService.DeleteTodo(userUUIDStr, idStr)
	if err != nil {
		if err == store.ErrNotFound {
			// SECURITY: Generic error message to prevent enumeration
			// Don't distinguish between "todo doesn't exist" and "todo belongs to another user"
			todoLogger.Warn("DeleteTodoHandler: todo not found id=%s for user: %s", idStr, userUUIDStr)
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}
		todoLogger.LogError("TodoService.DeleteTodo", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	todoLogger.LogResponse(http.StatusNoContent, "DeleteTodoHandler completed successfully")
}
