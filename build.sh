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

# Dependency check function
check_dependency() {
  local cmd="$1"
  local name="$2"
  local required="${3:-true}"
  local install_hint="${4:-}"
  
  if command -v "$cmd" >/dev/null 2>&1; then
    local version=""
    case "$cmd" in
      docker)
        if docker --version >/dev/null 2>&1; then
          version=$(docker --version 2>/dev/null | awk '{print $3}' | sed 's/,//' || echo "unknown")
        else
          version="unknown"
        fi
        ;;
      go)
        version=$(go version 2>/dev/null | awk '{print $3}' | sed 's/go//' || echo "unknown")
        ;;
      *)
        version="installed"
        ;;
    esac
    echo "✅ $name found (version: $version)"
    return 0
  else
    if [ "$required" = "true" ]; then
      echo "❌ ERROR: $name is required but not found in PATH." >&2
      if [ -n "$install_hint" ]; then
        echo "" >&2
        echo "💡 Installation hint:" >&2
        echo "   $install_hint" >&2
      fi
      exit 1
    else
      echo "⚠️  WARNING: $name not found (optional)" >&2
      return 1
    fi
  fi
}

# Check Docker service status
check_docker_service() {
  if ! docker info >/dev/null 2>&1; then
    echo "❌ ERROR: Docker daemon is not running." >&2
    echo "" >&2
    echo "💡 Please start Docker:" >&2
    echo "   macOS: Open Docker Desktop application" >&2
    echo "   Linux: sudo systemctl start docker" >&2
    echo "   Or visit: https://docs.docker.com/get-docker/" >&2
    exit 1
  fi
  echo "✅ Docker daemon is running"
}

# Docker Compose command helper (supports both v1 and v2)
DOCKER_COMPOSE_CMD=""
check_docker_compose() {
  if docker compose version >/dev/null 2>&1; then
    local version=$(docker compose version 2>/dev/null | awk '{print $4}' || echo "unknown")
    echo "✅ Docker Compose (v2) found (version: $version)"
    DOCKER_COMPOSE_CMD="docker compose"
    return 0
  elif command -v docker-compose >/dev/null 2>&1; then
    local version=$(docker-compose --version 2>/dev/null | awk '{print $4}' | sed 's/,//' || echo "unknown")
    echo "✅ Docker Compose (v1) found (version: $version)"
    DOCKER_COMPOSE_CMD="docker-compose"
    return 0
  else
    echo "❌ ERROR: Docker Compose is required but not found." >&2
    echo "" >&2
    echo "💡 Installation hint:" >&2
    echo "   Docker Compose v2: Usually included with Docker Desktop" >&2
    echo "   Or visit: https://docs.docker.com/compose/install/" >&2
    exit 1
  fi
}

# Helper: read variable from .env without sourcing it
get_env_var_from_file() {
  local var_name="$1"
  local env_file=".env"
  grep -E "^[[:space:]]*${var_name}[[:space:]]*=" "$env_file" | \
    sed -n "s/^[[:space:]]*${var_name}[[:space:]]*=[[:space:]]*//p" | \
    sed -e 's/^"\(.*\)"$/\1/' -e "s/^'\(.*\)'$/\1/" | \
    sed -e 's/[[:space:]]*$//' | \
    head -n 1
}

# Check .env file and required variables
check_env_file() {
  local env_file=".env"
  local missing_vars=()
  local invalid_vars=()
  
  echo ""
  echo "🔍 Checking .env file and required variables..."
  echo "================================================"
  
  # Check if .env file exists
  if [ ! -f "$env_file" ]; then
    echo "❌ ERROR: .env file not found!" >&2
    echo "" >&2
    echo "💡 Please create a .env file with the following required variables:" >&2
    echo "   - JWT_SECRET (at least 32 characters)" >&2
    echo "   - MASTER_KEY_ACTIVE (32 bytes: 64 hex characters or 32 raw bytes)" >&2
    echo "   - MASTER_KEY_ACTIVE_ID (non-empty key id)" >&2
    echo "   - ACCOUNT_LOOKUP_PEPPER (at least 32 bytes: hex, base64, or raw)" >&2
    echo "   - JWT_SECRET_MIN_LEN, JWT_ISSUER, JWT_AUDIENCE, JWT_EXPIRATION_MINUTES" >&2
    echo "   - ALLOWED_ORIGINS, BACKEND_PORT" >&2
    echo "   - MAX_BASE64_LOGIN_LEN, MAX_BASE64_TODO_LEN" >&2
    echo "   - MAX_REQUEST_BODY_BYTES, MAX_JSON_DEPTH, MAX_TITLE_LENGTH, MIN_TITLE_LENGTH, MAX_TAG_LENGTH, MAX_TAGS_PER_TODO" >&2
    echo "   - INTERNAL_ID_LENGTH" >&2
    echo "   - ARGON2_MEMORY_KIB, ARGON2_TIME, ARGON2_PARALLELISM, ARGON2_SALT_LENGTH, ARGON2_HASH_LENGTH" >&2
    echo "   - PENDING_TOKEN_TTL_SEC, PENDING_CLEANUP_INTERVAL_SEC, PENDING_ID_BYTES" >&2
    echo "   - RATE_LIMIT_MAX_TOKENS, RATE_LIMIT_REFILL_INTERVAL_SEC, RATE_LIMIT_CLEANUP_INTERVAL_SEC, RATE_LIMIT_MAX_BUCKETS" >&2
    echo "   - SERVER_READ_HEADER_TIMEOUT_SEC, SERVER_READ_TIMEOUT_SEC, SERVER_WRITE_TIMEOUT_SEC, SERVER_IDLE_TIMEOUT_SEC, SERVER_MAX_HEADER_BYTES" >&2
    echo "   - DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME, DB_SSL_MODE" >&2
    echo "" >&2
    echo "💡 Example generation commands:" >&2
    echo "   JWT_SECRET: openssl rand -base64 32" >&2
    echo "   MASTER_KEY_ACTIVE: openssl rand -hex 32" >&2
    echo "   MASTER_KEY_ACTIVE_ID: openssl rand -hex 4" >&2
    echo "   ACCOUNT_LOOKUP_PEPPER: openssl rand -hex 32" >&2
    exit 1
  fi
  
  echo "✅ .env file found"
  
  # Helper function to get variable value from .env file
  get_env_var() {
    local var_name="$1"
    # Extract value from .env file, handling comments and whitespace
    grep -E "^[[:space:]]*${var_name}[[:space:]]*=" "$env_file" | \
      sed -n "s/^[[:space:]]*${var_name}[[:space:]]*=[[:space:]]*//p" | \
      sed -e 's/^"\(.*\)"$/\1/' -e "s/^'\(.*\)'$/\1/" | \
      sed -e 's/[[:space:]]*$//' | \
      head -n 1
  }

  check_nonempty() {
    local var_name="$1"
    local value
    value=$(get_env_var "$var_name")
    if [ -z "$value" ]; then
      missing_vars+=("$var_name")
      return 1
    fi
    echo "✅ $var_name is set"
    return 0
  }

  check_positive_int() {
    local var_name="$1"
    local value
    value=$(get_env_var "$var_name")
    if [ -z "$value" ]; then
      missing_vars+=("$var_name")
      return 1
    fi
    if ! [[ "$value" =~ ^[0-9]+$ ]] || [ "$value" -le 0 ]; then
      invalid_vars+=("$var_name (must be a positive integer, current: $value)")
      return 1
    fi
    echo "✅ $var_name is set (value: $value)"
    return 0
  }
  
  # Check JWT_SECRET
  local jwt_secret=$(get_env_var "JWT_SECRET")
  if [ -z "$jwt_secret" ]; then
    missing_vars+=("JWT_SECRET")
  else
    local jwt_min_len=$(get_env_var "JWT_SECRET_MIN_LEN")
    if [[ "$jwt_min_len" =~ ^[0-9]+$ ]] && [ "$jwt_min_len" -gt 0 ]; then
      if [ ${#jwt_secret} -lt "$jwt_min_len" ]; then
        invalid_vars+=("JWT_SECRET (must be at least ${jwt_min_len} characters, current: ${#jwt_secret})")
      else
        echo "✅ JWT_SECRET is set and valid (length: ${#jwt_secret})"
      fi
    elif [ ${#jwt_secret} -lt 32 ]; then
      invalid_vars+=("JWT_SECRET (must be at least 32 characters, current: ${#jwt_secret})")
    else
      echo "✅ JWT_SECRET is set and valid (length: ${#jwt_secret})"
    fi
  fi
  
  # Check MASTER_KEY_ACTIVE
  local master_key_active=$(get_env_var "MASTER_KEY_ACTIVE")
  if [ -z "$master_key_active" ]; then
    missing_vars+=("MASTER_KEY_ACTIVE")
  else
    local key_len=${#master_key_active}
    if [ $key_len -eq 64 ]; then
      if [[ "$master_key_active" =~ ^[0-9a-fA-F]{64}$ ]]; then
        echo "✅ MASTER_KEY_ACTIVE is set and valid (64 hex characters)"
      else
        invalid_vars+=("MASTER_KEY_ACTIVE (64 characters but not valid hex format)")
      fi
    elif [ $key_len -eq 32 ]; then
      echo "✅ MASTER_KEY_ACTIVE is set and valid (32 raw bytes)"
    else
      invalid_vars+=("MASTER_KEY_ACTIVE (must be 32 bytes or 64 hex characters, current: $key_len)")
    fi
  fi

  # Check MASTER_KEY_ACTIVE_ID
  local master_key_active_id=$(get_env_var "MASTER_KEY_ACTIVE_ID")
  if [ -z "$master_key_active_id" ]; then
    missing_vars+=("MASTER_KEY_ACTIVE_ID")
  else
    echo "✅ MASTER_KEY_ACTIVE_ID is set (length: ${#master_key_active_id})"
  fi
  
  # Check ACCOUNT_LOOKUP_PEPPER
  local account_pepper=$(get_env_var "ACCOUNT_LOOKUP_PEPPER")
  if [ -z "$account_pepper" ]; then
    missing_vars+=("ACCOUNT_LOOKUP_PEPPER")
  else
    local pepper_len=${#account_pepper}
    # Try to decode as hex or base64 to get actual byte length
    local byte_len=$pepper_len
    
    # If it's hex (even length and only hex chars), divide by 2
    if [[ "$account_pepper" =~ ^[0-9a-fA-F]+$ ]] && [ $((pepper_len % 2)) -eq 0 ]; then
      byte_len=$((pepper_len / 2))
    fi
    
    # Base64 encoded strings are typically longer, but we'll check raw length
    # The backend accepts hex, base64, or raw bytes, so we check raw length >= 32
    if [ $byte_len -ge 32 ] || [ $pepper_len -ge 32 ]; then
      echo "✅ ACCOUNT_LOOKUP_PEPPER is set and valid (length: $pepper_len)"
    else
      invalid_vars+=("ACCOUNT_LOOKUP_PEPPER (must be at least 32 bytes, current decoded length: ~$byte_len)")
    fi
  fi

  # JWT config
  check_positive_int "JWT_SECRET_MIN_LEN"
  check_nonempty "JWT_ISSUER"
  check_nonempty "JWT_AUDIENCE"
  check_positive_int "JWT_EXPIRATION_MINUTES"

  # CORS and server config
  check_nonempty "ALLOWED_ORIGINS"
  check_nonempty "BACKEND_PORT"
  check_positive_int "SERVER_READ_HEADER_TIMEOUT_SEC"
  check_positive_int "SERVER_READ_TIMEOUT_SEC"
  check_positive_int "SERVER_WRITE_TIMEOUT_SEC"
  check_positive_int "SERVER_IDLE_TIMEOUT_SEC"
  check_positive_int "SERVER_MAX_HEADER_BYTES"

  # Request/validation limits
  check_positive_int "MAX_REQUEST_BODY_BYTES"
  check_positive_int "MAX_JSON_DEPTH"
  check_positive_int "MAX_TITLE_LENGTH"
  check_positive_int "MIN_TITLE_LENGTH"
  check_positive_int "MAX_TAG_LENGTH"
  check_positive_int "MAX_TAGS_PER_TODO"
  check_positive_int "MAX_BASE64_LOGIN_LEN"
  check_positive_int "MAX_BASE64_TODO_LEN"
  check_positive_int "INTERNAL_ID_LENGTH"

  # Argon2id config
  check_positive_int "ARGON2_MEMORY_KIB"
  check_positive_int "ARGON2_TIME"
  check_positive_int "ARGON2_PARALLELISM"
  check_positive_int "ARGON2_SALT_LENGTH"
  check_positive_int "ARGON2_HASH_LENGTH"

  # Pending registration config
  check_positive_int "PENDING_TOKEN_TTL_SEC"
  check_positive_int "PENDING_CLEANUP_INTERVAL_SEC"
  check_positive_int "PENDING_ID_BYTES"

  # Rate limiting config
  check_positive_int "RATE_LIMIT_MAX_TOKENS"
  check_positive_int "RATE_LIMIT_REFILL_INTERVAL_SEC"
  check_positive_int "RATE_LIMIT_CLEANUP_INTERVAL_SEC"
  check_positive_int "RATE_LIMIT_MAX_BUCKETS"

  # Database config
  check_nonempty "DB_HOST"
  check_nonempty "DB_PORT"
  check_nonempty "DB_USER"
  check_nonempty "DB_PASSWORD"
  check_nonempty "DB_NAME"
  check_nonempty "DB_SSL_MODE"
  
  # Report missing variables
  if [ ${#missing_vars[@]} -gt 0 ]; then
    echo "" >&2
    echo "❌ ERROR: Missing required environment variables:" >&2
    for var in "${missing_vars[@]}"; do
      echo "   - $var" >&2
    done
    echo "" >&2
    echo "💡 Please add these variables to your .env file" >&2
    exit 1
  fi
  
  # Report invalid variables
  if [ ${#invalid_vars[@]} -gt 0 ]; then
    echo "" >&2
    echo "❌ ERROR: Invalid environment variable values:" >&2
    for var in "${invalid_vars[@]}"; do
      echo "   - $var" >&2
    done
    echo "" >&2
    echo "💡 Please fix these variables in your .env file" >&2
    exit 1
  fi
  
  echo "✅ All required environment variables are set and valid"
  echo ""
}

# Check dependencies
echo ""
echo "🔍 Checking dependencies..."
echo "============================"

check_dependency docker "Docker" true \
  "Visit: https://docs.docker.com/get-docker/"
check_docker_service
check_docker_compose

# Check .env file before proceeding
check_env_file

# Load DB settings for local test runs (avoid sourcing .env)
ENV_DB_USER="$(get_env_var_from_file "DB_USER")"
ENV_DB_PASSWORD="$(get_env_var_from_file "DB_PASSWORD")"
ENV_DB_NAME="$(get_env_var_from_file "DB_NAME")"
ENV_DB_PORT="$(get_env_var_from_file "DB_PORT")"
ENV_DB_SSL_MODE="$(get_env_var_from_file "DB_SSL_MODE")"
TEST_DB_HOST="localhost"

# Check Go only if running tests locally
if [ -f "/.dockerenv" ] || grep -q docker /proc/self/cgroup 2>/dev/null; then
  echo "ℹ️  Running in Docker environment - Go check skipped"
else
  check_dependency go "Go" false \
    "Visit: https://go.dev/doc/install"
fi

echo ""
echo "============================"
echo ""

# Handle --clean flag
if [ "${1:-}" == "--clean" ]; then
    echo "🧹 Full cleanup mode - removing unused Docker objects..."
    docker system prune -f
    echo "🗑️  Clearing build cache..."
    docker builder prune -f
fi

echo "🛑 Stopping current services..."
if $DOCKER_COMPOSE_CMD ps 2>/dev/null | grep -q "Up"; then
  $DOCKER_COMPOSE_CMD down
else
  echo "ℹ️  No running services to stop"
fi

echo "🔨 Building Docker images..."
$DOCKER_COMPOSE_CMD build

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
    $DOCKER_COMPOSE_CMD run --rm backend sh -c "
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

# Ensure the configured DB exists in the Postgres container (for local tests)
ensure_db_exists() {
    local db_name="$1"
    if [ -z "$db_name" ]; then
        echo "⚠️  Skipping database creation: DB_NAME is empty"
        return 1
    fi
    if ! [[ "$db_name" =~ ^[A-Za-z0-9_]+$ ]]; then
        echo "⚠️  Skipping database creation: invalid DB_NAME '$db_name'"
        return 1
    fi
    if ! $DOCKER_COMPOSE_CMD exec -T postgres psql -U "$ENV_DB_USER" -d postgres -tAc \
        "SELECT 1 FROM pg_database WHERE datname='${db_name}'" | grep -q 1; then
        echo "🛠️  Creating database '$db_name' for tests..."
        $DOCKER_COMPOSE_CMD exec -T postgres psql -U "$ENV_DB_USER" -d postgres \
          -c "CREATE DATABASE ${db_name};"
    fi
}

# Run go test with explicit DB settings (for local tests)
go_test_with_db_env() {
    DB_HOST="$TEST_DB_HOST" \
    DB_PORT="$ENV_DB_PORT" \
    DB_USER="$ENV_DB_USER" \
    DB_PASSWORD="$ENV_DB_PASSWORD" \
    DB_NAME="$ENV_DB_NAME" \
    DB_SSL_MODE="$ENV_DB_SSL_MODE" \
    go test -tags test "$@"
}

# Function to run tests locally
run_tests_local() {
    echo ""
    echo "🧪 Running migration tests (MUST PASS FIRST)..."
    cd backend
    go_test_with_db_env ./tests/database -v || exit 1
    
    echo ""
    echo "🧪 Running authentication tests..."
    go_test_with_db_env ./tests/auth -v || exit 1
    
    echo ""
    echo "🧪 Running encryption tests..."
    go_test_with_db_env ./tests/encryption -v || exit 1
    
    echo ""
    echo "🧪 Running middleware tests..."
    go_test_with_db_env ./tests/middleware -v || exit 1
    
    echo ""
    echo "🧪 Running todo service tests..."
    go_test_with_db_env ./tests/todo -v || exit 1
    
    echo ""
    echo "🧪 Running integration tests..."
    go_test_with_db_env ./tests/integration -v || exit 1
    
    echo ""
    echo "🧪 Running handler tests..."
    go_test_with_db_env ./tests/handlers -v || exit 1
    
    echo ""
    echo "🧪 Running utils tests..."
    go_test_with_db_env ./tests/utils -v || exit 1
    
    echo ""
    echo "✅ All tests passed!"
    cd ..
}

# Run tests based on environment
TEST_EXIT_CODE=0
if [ "$TEST_ENV" == "docker" ]; then
    ensure_db_exists "$ENV_DB_NAME"
    run_tests_in_docker || TEST_EXIT_CODE=$?
else
    # Check if Docker Compose is running (for database)
    if $DOCKER_COMPOSE_CMD ps postgres 2>/dev/null | grep -q "Up"; then
        echo "✅ Database is running, running tests locally..."
        ensure_db_exists "$ENV_DB_NAME"
        run_tests_local || TEST_EXIT_CODE=$?
    else
        echo "⚠️  Database not running. Starting database for tests..."
        $DOCKER_COMPOSE_CMD up -d postgres
        
        # Wait for database to be ready
        echo "⏳ Waiting for database to be ready..."
        timeout=30
        while [ $timeout -gt 0 ]; do
            if $DOCKER_COMPOSE_CMD exec -T postgres pg_isready -U "$ENV_DB_USER" >/dev/null 2>&1; then
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

        ensure_db_exists "$ENV_DB_NAME"
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
$DOCKER_COMPOSE_CMD up -d

echo ""
echo "✅ Build complete! Services are running in background."
echo "📋 Check status with: $DOCKER_COMPOSE_CMD ps"
echo "📊 View logs with: $DOCKER_COMPOSE_CMD logs -f"
echo ""
echo "💡 Tip: Use './build.sh --clean' for full cleanup if you have issues"
