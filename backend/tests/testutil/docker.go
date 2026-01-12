package testutil

import "os"

// IsRunningInDocker checks if tests are running inside a Docker container
// This is useful for determining the correct database host
func IsRunningInDocker() bool {
	// Check for Docker-specific environment variables or files
	// Method 1: Check for .dockerenv file (most reliable)
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// Method 2: Check for Docker-specific cgroup
	// This is a fallback method
	if cgroup, err := os.ReadFile("/proc/self/cgroup"); err == nil {
		if contains(string(cgroup), "docker") {
			return true
		}
	}

	return false
}

// GetDatabaseHost returns the appropriate database host based on environment
// - If running in Docker: returns "postgres" (Docker service name)
// - If running locally: returns "localhost"
func GetDatabaseHost() string {
	if IsRunningInDocker() {
		return "postgres"
	}
	return "localhost"
}

// SetupTestDatabaseEnvForDocker sets up database environment variables
// for tests running in Docker environment
func SetupTestDatabaseEnvForDocker() {
	// In Docker, use service names
	if os.Getenv("DB_HOST") == "" {
		os.Setenv("DB_HOST", "postgres")
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

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsMiddle(s, substr))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
