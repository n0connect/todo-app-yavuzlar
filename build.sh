#!/bin/bash
set -euo pipefail

# Docker Rebuild Script with Tests
# This script will:
# 1. Stop current compose services
# 2. Rebuild images (using cache when possible)
# 3. Run ALL tests (migration tests first, then all other tests)
# 4. Start services ONLY if all tests pass
#
# Use './build.sh --clean' for full cleanup if needed

if [ "$1" == "--clean" ]; then
    echo "🧹 Full cleanup mode - removing unused Docker objects..."
    docker system prune -f
    echo "🗑️  Clearing build cache..."
    docker builder prune -f
fi

echo "🛑 Stopping current services..."
docker compose down

echo "🔨 Building Docker images..."
docker compose build

echo ""
echo "🧪 Running ALL tests before starting services..."
echo "================================================"

# Check if we're in Docker or local
if [ -f "/.dockerenv" ] || grep -q docker /proc/self/cgroup 2>/dev/null; then
    echo "📍 Running tests in Docker environment..."
    TEST_ENV="docker"
else
    echo "📍 Running tests in local environment..."
    TEST_ENV="local"
fi

# Function to run tests in Docker container
run_tests_in_docker() {
    echo ""
    echo "📦 Starting test container..."
    docker compose run --rm backend sh -c "
        echo '🧪 Running migration tests...'
        go test -tags test ./tests/database -v || exit 1
        
        echo ''
        echo '🧪 Running authentication tests...'
        go test -tags test ./tests/auth -v || exit 1
        
        echo ''
        echo '🧪 Running encryption tests...'
        go test -tags test ./tests/encryption -v || exit 1
        
        echo ''
        echo '🧪 Running middleware tests...'
        go test -tags test ./tests/middleware -v || exit 1
        
        echo ''
        echo '🧪 Running todo service tests...'
        go test -tags test ./tests/todo -v || exit 1
        
        echo ''
        echo '🧪 Running integration tests...'
        go test -tags test ./tests/integration -v || exit 1
        
        echo ''
        echo '🧪 Running handler tests...'
        go test -tags test ./tests/handlers -v || exit 1
        
        echo ''
        echo '🧪 Running utils tests...'
        go test -tags test ./tests/utils -v || exit 1
        
        echo ''
        echo '✅ All tests passed!'
    "
}

# Function to run tests locally
run_tests_local() {
    echo ""
    echo "🧪 Running migration tests (MUST PASS FIRST)..."
    cd backend
    go test -tags test ./tests/database -v || exit 1
    
    echo ""
    echo "🧪 Running authentication tests..."
    go test -tags test ./tests/auth -v || exit 1
    
    echo ""
    echo "🧪 Running encryption tests..."
    go test -tags test ./tests/encryption -v || exit 1
    
    echo ""
    echo "🧪 Running middleware tests..."
    go test -tags test ./tests/middleware -v || exit 1
    
    echo ""
    echo "🧪 Running todo service tests..."
    go test -tags test ./tests/todo -v || exit 1
    
    echo ""
    echo "🧪 Running integration tests..."
    go test -tags test ./tests/integration -v || exit 1
    
    echo ""
    echo "🧪 Running handler tests..."
    go test -tags test ./tests/handlers -v || exit 1
    
    echo ""
    echo "🧪 Running utils tests..."
    go test -tags test ./tests/utils -v || exit 1
    
    echo ""
    echo "✅ All tests passed!"
    cd ..
}

# Run tests based on environment
TEST_EXIT_CODE=0
if [ "$TEST_ENV" == "docker" ]; then
    run_tests_in_docker || TEST_EXIT_CODE=$?
else
    # Check if Docker Compose is running (for database)
    if docker compose ps postgres 2>/dev/null | grep -q "Up"; then
        echo "✅ Database is running, running tests locally..."
        run_tests_local || TEST_EXIT_CODE=$?
    else
        echo "⚠️  Database not running. Starting database for tests..."
        docker compose up -d postgres
        
        # Wait for database to be ready
        echo "⏳ Waiting for database to be ready..."
        timeout=30
        while [ $timeout -gt 0 ]; do
            if docker compose exec -T postgres pg_isready -U postgres >/dev/null 2>&1; then
                echo "✅ Database is ready!"
                break
            fi
            sleep 1
            timeout=$((timeout - 1))
        done
        
        if [ $timeout -eq 0 ]; then
            echo "❌ Database failed to start in time"
            exit 1
        fi
        
        run_tests_local || TEST_EXIT_CODE=$?
    fi
fi

if [ $TEST_EXIT_CODE -ne 0 ]; then
    echo ""
    echo "❌ Tests failed! Services will NOT be started."
    echo "🔍 Fix the failing tests before starting services."
    exit 1
fi

echo ""
echo "✅ All tests passed! Starting services..."
docker compose up -d

echo ""
echo "✅ Build complete! Services are running in background."
echo "📋 Check status with: docker compose ps"
echo "📊 View logs with: docker compose logs -f"
echo ""
echo "💡 Tip: Use './build.sh --clean' for full cleanup if you have issues"