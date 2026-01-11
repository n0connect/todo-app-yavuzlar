package main

import (
	"log"
	"net/http"

	"todo-app-backend/internal/config"
	"todo-app-backend/internal/database"
	"todo-app-backend/internal/handlers"
	"todo-app-backend/internal/middleware"
	"todo-app-backend/internal/utils"
)

var mainLogger = utils.NewLogger("MAIN")

func main() {
	mainLogger.Info("Starting Todo App Backend Server...")

	// Validate configuration at startup
	mainLogger.Debug("Validating startup configuration...")
	if err := database.ValidateStartupConfig(); err != nil {
		mainLogger.Error("Configuration validation failed: %v", err)
		log.Fatalf("Configuration error: %v", err)
	}
	mainLogger.Info("Configuration validation passed")

	// Log environment mode
	if config.IsProductionMode() {
		mainLogger.Info("Running in PRODUCTION mode")
	} else {
		mainLogger.Info("Running in DEVELOPMENT mode")
	}

	// Initialize database
	mainLogger.Debug("Initializing database connection...")
	database.Init()

	// Setup routes
	mainLogger.Debug("Setting up HTTP routes...")
	setupRoutes()
	mainLogger.Info("HTTP routes configured successfully")

	backendPort := config.GetEnv("BACKEND_PORT", "8080")
	mainLogger.Info("Server starting on port %s", backendPort)
	mainLogger.Info("Server is ready to accept connections")

	log.Fatal(http.ListenAndServe(":"+backendPort, nil))
}

func setupRoutes() {
	mainLogger.Debug("Registering route: POST /api/v1/register")
	http.HandleFunc("/api/v1/register", middleware.SecurityHeadersMiddleware(middleware.CORSMiddleware(handlers.RegisterHandler)))

	mainLogger.Debug("Registering route: POST /api/v1/login")
	http.HandleFunc("/api/v1/login", middleware.SecurityHeadersMiddleware(middleware.CORSMiddleware(handlers.LoginHandler)))

	mainLogger.Debug("Registering route: GET/POST /api/v1/todos")
	http.HandleFunc("/api/v1/todos", middleware.SecurityHeadersMiddleware(middleware.CORSMiddleware(middleware.JWTMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.GetTodosHandler(w, r)
		case http.MethodPost:
			handlers.CreateTodoHandler(w, r)
		default:
			mainLogger.Warn("Method not allowed for /api/v1/todos: %s", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))))

	mainLogger.Debug("Registering route: PUT/DELETE /api/v1/todos/")
	http.HandleFunc("/api/v1/todos/", middleware.SecurityHeadersMiddleware(middleware.CORSMiddleware(middleware.JWTMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			handlers.UpdateTodoHandler(w, r)
		} else if r.Method == http.MethodDelete {
			handlers.DeleteTodoHandler(w, r)
		} else {
			mainLogger.Warn("Method not allowed for /api/v1/todos/: %s", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))))

	mainLogger.Info("All routes registered successfully")
}
