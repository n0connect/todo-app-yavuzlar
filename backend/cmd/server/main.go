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
	// Log environment mode
	if config.IsProductionMode() {
		mainLogger.Info("Application is Running in PRODUCTION mode")
	} else {
		mainLogger.Info("Application is Running in DEVELOPMENT mode")
	}

	mainLogger.Info("Starting Todo App Backend Server...")

	// Validate database configuration at startup
	mainLogger.Debug("Validating startup configuration...")
	if err := database.ValidateStartupConfig(); err != nil {
		mainLogger.Error("Configuration validation failed: %v", err)
		log.Fatalf("Configuration error: %v", err)
	}
	mainLogger.Info("Configuration validation passed!")

	// Initialize database
	mainLogger.Debug("Initializing database connection...")
	database.Init()

	// Setup routes
	mainLogger.Debug("Setting up HTTP routes...")

	// -- Server Mux Settings
	mainLogger.Debug("Setting mux for HTTP server...")
	mux := http.NewServeMux()
	setupRoutes(mux)

	mainLogger.Info("HTTP routes configured successfully")

	backendPort := config.GetBackendPort()
	if backendPort == "" {
		mainLogger.Error("BACKEND_PORT is required")
		log.Fatal("Configuration error: BACKEND_PORT is required")
	}
	mainLogger.Info("Server starting on port %s", backendPort)

	// -- Server custom srv settings
	mainLogger.Info("Setting Custom server settings for HTTP server...")
	addr := ":" + backendPort
	readHeaderTimeout := config.GetServerReadHeaderTimeout()
	readTimeout := config.GetServerReadTimeout()
	writeTimeout := config.GetServerWriteTimeout()
	idleTimeout := config.GetServerIdleTimeout()
	maxHeaderBytes := config.GetServerMaxHeaderBytes()

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
	}

	mainLogger.Info("Server is ready to accept connections")
	log.Fatal(srv.ListenAndServe())
}

// Fix middleware handlers
// Add http.NewServeMux() for not accept extra endpoints.
func setupRoutes(mux *http.ServeMux) {
	mainLogger.Debug("Registering route: POST /api/v2/register")
	mux.HandleFunc("/api/v2/register", middleware.SecurityHeadersMiddleware(middleware.CORSMiddleware(middleware.ContentTypeMiddleware(middleware.RateLimitMiddleware(handlers.RegisterHandler)))))

	mainLogger.Debug("Registering route: POST /api/v2/login")
	mux.HandleFunc("/api/v2/login", middleware.SecurityHeadersMiddleware(middleware.CORSMiddleware(middleware.ContentTypeMiddleware(middleware.RateLimitMiddleware(handlers.LoginHandler)))))

	mainLogger.Debug("Registering route: GET/POST /api/v2/todos")
	mux.HandleFunc("/api/v2/todos", middleware.SecurityHeadersMiddleware(middleware.CORSMiddleware(middleware.ContentTypeMiddleware(middleware.JWTMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.GetTodosHandler(w, r)
		case http.MethodPost:
			handlers.CreateTodoHandler(w, r)
		default:
			mainLogger.Warn("Method not allowed for /api/v2/todos: %s", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})))))

	mainLogger.Debug("Registering route: PUT/DELETE /api/v2/todos/")
	mux.HandleFunc("/api/v2/todos/", middleware.SecurityHeadersMiddleware(middleware.CORSMiddleware(middleware.ContentTypeMiddleware(middleware.JWTMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			handlers.UpdateTodoHandler(w, r)
		case http.MethodDelete:
			handlers.DeleteTodoHandler(w, r)
		default:
			mainLogger.Warn("Method not allowed for /api/v2/todos/: %s", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})))))

	// Request Path Whitelist: Handle unknown paths with 404
	// Only registered paths are allowed, all others return 404
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		mainLogger.Warn("Path not found: %s %s", r.Method, r.URL.Path)
		http.Error(w, "Not Found", http.StatusNotFound)
	})

	mainLogger.Info("All routes registered successfully")
}
