//go:build test
// +build test

package testutil

import "os"

// SetupTestDatabaseEnv sets up database environment variables for tests
// This should be called before database.Init() in migration tests
// Automatically detects Docker environment and uses appropriate host
func SetupTestDatabaseEnv() {
	// Check if running in Docker
	if IsRunningInDocker() {
		// Use Docker service names
		SetupTestDatabaseEnvForDocker()
		return
	}

	// Local testing - use localhost
	if os.Getenv("DB_HOST") == "" {
		os.Setenv("DB_HOST", "localhost")
	}
	if os.Getenv("DB_PORT") == "" {
		os.Setenv("DB_PORT", "5432")
	}
	if os.Getenv("DB_USER") == "" {
		os.Setenv("DB_USER", "postgres")
	}
	if os.Getenv("DB_PASSWORD") == "" {
		os.Setenv("DB_PASSWORD", "postgres")
	}
	if os.Getenv("DB_NAME") == "" {
		os.Setenv("DB_NAME", "todos")
	}
	if os.Getenv("DB_SSL_MODE") == "" {
		os.Setenv("DB_SSL_MODE", "disable")
	}
}
