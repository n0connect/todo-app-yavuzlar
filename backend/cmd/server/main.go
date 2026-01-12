package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"todo-app-backend/internal/auth"
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

	backendPort := config.GetEnv("BACKEND_PORT", "8080")
	mainLogger.Info("Server starting on port %s", backendPort)

	// -- Server custom srv settings
	mainLogger.Info("Setting Custom server settings for HTTP server...")
	addr := ":" + backendPort
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB
	}

	mainLogger.Info("Server is ready to accept connections")

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	serverErr := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	mainLogger.Info("Server started successfully. Waiting for shutdown signal...")

	// Wait for interrupt signal or server error
	select {
	case err := <-serverErr:
		mainLogger.Error("Server error: %v", err)
		cleanup()
		log.Fatal(err)
	case sig := <-sigChan:
		mainLogger.Info("Received signal: %v. Starting graceful shutdown...", sig)
		cleanup()
		gracefulShutdown(srv)
	}
}

// cleanup performs cleanup operations before shutdown
func cleanup() {
	mainLogger.Debug("Performing cleanup operations...")
	auth.StopCleanup() // Stop pending store cleanup goroutine
	mainLogger.Debug("Cleanup completed")
}

// gracefulShutdown shuts down the server gracefully
func gracefulShutdown(srv *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		mainLogger.Error("Server forced to shutdown: %v", err)
		log.Fatal(err)
	}

	mainLogger.Info("Server gracefully stopped")
}

// Fix middleware handlers
// Add http.NewServeMux() for not accept extra endpoints.
func setupRoutes(mux *http.ServeMux) {
	mainLogger.Debug("Registering route: POST /api/v1/register")
	mux.HandleFunc("/api/v1/register", middleware.SecurityHeadersMiddleware(middleware.CORSMiddleware(middleware.RateLimitMiddleware(handlers.RegisterHandler))))

	mainLogger.Debug("Registering route: POST /api/v1/login")
	mux.HandleFunc("/api/v1/login", middleware.SecurityHeadersMiddleware(middleware.CORSMiddleware(middleware.RateLimitMiddleware(handlers.LoginHandler))))

	mainLogger.Debug("Registering route: GET/POST /api/v1/todos")
	mux.HandleFunc("/api/v1/todos", middleware.SecurityHeadersMiddleware(middleware.CORSMiddleware(middleware.JWTMiddleware(func(w http.ResponseWriter, r *http.Request) {
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
	mux.HandleFunc("/api/v1/todos/", middleware.SecurityHeadersMiddleware(middleware.CORSMiddleware(middleware.JWTMiddleware(func(w http.ResponseWriter, r *http.Request) {
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
