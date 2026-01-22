#!/usr/bin/env bash
set -euo pipefail

echo "Todo App Setup"
echo "=============="

ENV_FILE=".env"
INTERACTIVE=1

for arg in "$@"; do
  case "$arg" in
    --non-interactive)
      INTERACTIVE=0
      ;;
  esac
done

# Dependency check function
check_dependency() {
  local cmd="$1"
  local name="$2"
  local required="${3:-true}"
  local install_hint="${4:-}"
  
  if command -v "$cmd" >/dev/null 2>&1; then
    local version=""
    case "$cmd" in
      openssl)
        version=$(openssl version 2>/dev/null | head -n1 | awk '{print $2}' || echo "unknown")
        ;;
      docker)
        version=$(docker --version 2>/dev/null | awk '{print $3}' | sed 's/,//' || echo "unknown")
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

require_openssl_version() {
  local min_major=3
  local min_minor=2
  local min_patch=0
  local version_raw
  version_raw="$(openssl version 2>/dev/null | awk '{print $2}')"
  if [[ -z "$version_raw" ]] || [[ "$version_raw" == LibreSSL* ]]; then
    echo "❌ ERROR: OpenSSL >= 3.2.0 is required (LibreSSL is not supported)." >&2
    exit 1
  fi
  local version="${version_raw%%[!0-9.]*}"
  local major minor patch
  IFS='.' read -r major minor patch <<<"$version"
  major="${major:-0}"
  minor="${minor:-0}"
  patch="${patch:-0}"
  if (( major < min_major || (major == min_major && minor < min_minor) || (major == min_major && minor == min_minor && patch < min_patch) )); then
    echo "❌ ERROR: OpenSSL >= 3.2.0 is required (found $version_raw)." >&2
    echo "💡 macOS: brew install openssl@3 and ensure PATH points to it." >&2
    echo "💡 Linux: install OpenSSL 3.2.x from your distro or source." >&2
    exit 1
  fi
}

# Check required dependencies
echo ""
echo "🔍 Checking dependencies..."
echo "============================"

check_dependency openssl "OpenSSL" true \
  "macOS: Usually pre-installed\n   Linux: sudo apt-get install openssl (Debian/Ubuntu) or sudo yum install openssl (RHEL/CentOS)\n   Or visit: https://www.openssl.org/source/"
require_openssl_version

# Check optional dependencies (informational)
check_dependency docker "Docker" false \
  "Visit: https://docs.docker.com/get-docker/"
check_dependency docker-compose "Docker Compose" false \
  "Visit: https://docs.docker.com/compose/install/"

# Check if docker compose (v2) is available
if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
  echo "✅ Docker Compose (v2) found"
elif command -v docker-compose >/dev/null 2>&1; then
  echo "✅ Docker Compose (v1) found"
fi

echo ""
echo "============================"
echo ""

# Create .env if missing
if [[ ! -f "$ENV_FILE" ]]; then
  touch "$ENV_FILE"
  echo "Created $ENV_FILE"
fi

# Load existing .env values into current shell (do not fail if file has empty lines)
set -a
# shellcheck disable=SC1090
source "$ENV_FILE"
set +a

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "ERROR: '$1' is required but not found in PATH." >&2
    exit 1
  }
}

upsert_env() {
  local key="$1"
  local value="$2"

  # Remove any existing lines for key (avoid duplicates), then append one canonical line.
  # Works on macOS and Linux.
  if grep -qE "^${key}=" "$ENV_FILE"; then
    tmp="$(mktemp)"
    grep -vE "^${key}=" "$ENV_FILE" > "$tmp"
    mv "$tmp" "$ENV_FILE"
  fi
  printf '%s=%s\n' "$key" "$value" >> "$ENV_FILE"
}

apply_default() {
  local key="$1"
  local value="$2"
  if [[ -z "${!key:-}" ]]; then
    echo "Setting default $key=$value"
    export "$key"="$value"
    upsert_env "$key" "$value"
  fi
}

require_cmd openssl

prompt_input() {
  local prompt="$1"
  local silent="${2:-0}"
  local value
  if [ -r /dev/tty ] && [ -w /dev/tty ]; then
    if [ "$silent" -eq 1 ]; then
      read -r -s -p "$prompt" value < /dev/tty
      echo "" > /dev/tty
    else
      read -r -p "$prompt" value < /dev/tty
    fi
  else
    if [ "$silent" -eq 1 ]; then
      read -r -s -p "$prompt" value
      echo ""
    else
      read -r -p "$prompt" value
    fi
  fi
  printf '%s' "$value"
}

# Generate/load JWT_SECRET (honor JWT_SECRET_MIN_LEN if set)
if [[ -z "${JWT_SECRET:-}" ]]; then
  jwt_min_len="${JWT_SECRET_MIN_LEN:-}"
  if [[ "$jwt_min_len" =~ ^[0-9]+$ ]] && [ "$jwt_min_len" -gt 0 ]; then
    echo "Generating JWT_SECRET (min length: $jwt_min_len)..."
    jwt_bytes=$(( (jwt_min_len + 1) / 2 ))
    JWT_SECRET="$(openssl rand -hex "$jwt_bytes")"
  else
    echo "Generating JWT_SECRET..."
    JWT_SECRET="$(openssl rand -hex 32)"
  fi
fi
upsert_env "JWT_SECRET" "$JWT_SECRET"

# Generate/load ACCOUNT_LOOKUP_PEPPER
if [[ -z "${ACCOUNT_LOOKUP_PEPPER:-}" ]]; then
  echo "Generating ACCOUNT_LOOKUP_PEPPER..."
  ACCOUNT_LOOKUP_PEPPER="$(openssl rand -hex 32)"
fi
upsert_env "ACCOUNT_LOOKUP_PEPPER" "$ACCOUNT_LOOKUP_PEPPER"

# Generate/load MASTER_KEY_ACTIVE
if [[ -z "${MASTER_KEY_ACTIVE:-}" ]]; then
  echo "Generating MASTER_KEY_ACTIVE..."
  MASTER_KEY_ACTIVE="$(openssl rand -hex 32)"
fi
upsert_env "MASTER_KEY_ACTIVE" "$MASTER_KEY_ACTIVE"

# Generate/load MASTER_KEY_ACTIVE_ID
if [[ -z "${MASTER_KEY_ACTIVE_ID:-}" ]]; then
  echo "Generating MASTER_KEY_ACTIVE_ID..."
  MASTER_KEY_ACTIVE_ID="$(openssl rand -hex 4)"
fi
upsert_env "MASTER_KEY_ACTIVE_ID" "$MASTER_KEY_ACTIVE_ID"

# Apply default non-secret config values (override in .env as needed)
apply_default "JWT_SECRET_MIN_LEN" "32"
apply_default "JWT_ISSUER" "todo-app-backend"
apply_default "JWT_AUDIENCE" "todo-app-frontend"
apply_default "JWT_EXPIRATION_MINUTES" "15"
apply_default "ALLOWED_ORIGINS" "http://localhost"
apply_default "APP_ENV" "development"
apply_default "LOG_LEVEL" "info"
apply_default "DB_HOST" "localhost"
apply_default "DB_PORT" "5432"
apply_default "DB_USER" "postgres"
apply_default "DB_PASSWORD" "postgres"
apply_default "DB_NAME" "todo_app"
apply_default "DB_SSL_MODE" "disable"
apply_default "BACKEND_PORT" "8080"
apply_default "MAX_BASE64_LOGIN_LEN" "4096"
apply_default "MAX_BASE64_TODO_LEN" "65536"
apply_default "MAX_REQUEST_BODY_BYTES" "1048576"
apply_default "MAX_JSON_DEPTH" "2"
apply_default "MAX_TITLE_LENGTH" "1000"
apply_default "MIN_TITLE_LENGTH" "1"
apply_default "MAX_TAG_LENGTH" "6"
apply_default "MAX_TAGS_PER_TODO" "10"
apply_default "ARGON2_MEMORY_KIB" "65536"
apply_default "ARGON2_TIME" "3"
apply_default "ARGON2_PARALLELISM" "2"
apply_default "ARGON2_SALT_LENGTH" "16"
apply_default "ARGON2_HASH_LENGTH" "32"
apply_default "PENDING_TOKEN_TTL_SEC" "300"
apply_default "PENDING_CLEANUP_INTERVAL_SEC" "60"
apply_default "PENDING_ID_BYTES" "16"
apply_default "INTERNAL_ID_LENGTH" "24"
apply_default "RATE_LIMIT_MAX_TOKENS" "20"
apply_default "RATE_LIMIT_REFILL_INTERVAL_SEC" "3"
apply_default "RATE_LIMIT_CLEANUP_INTERVAL_SEC" "300"
apply_default "RATE_LIMIT_MAX_BUCKETS" "10000"
apply_default "SERVER_READ_HEADER_TIMEOUT_SEC" "5"
apply_default "SERVER_READ_TIMEOUT_SEC" "15"
apply_default "SERVER_WRITE_TIMEOUT_SEC" "15"
apply_default "SERVER_IDLE_TIMEOUT_SEC" "60"
apply_default "SERVER_MAX_HEADER_BYTES" "8192"

required_vars=(
  "JWT_SECRET_MIN_LEN"
  "JWT_ISSUER"
  "JWT_AUDIENCE"
  "JWT_EXPIRATION_MINUTES"
  "ALLOWED_ORIGINS"
  "APP_ENV"
  "LOG_LEVEL"
  "DB_HOST"
  "DB_PORT"
  "DB_USER"
  "DB_PASSWORD"
  "DB_NAME"
  "DB_SSL_MODE"
  "BACKEND_PORT"
  "MAX_BASE64_LOGIN_LEN"
  "MAX_BASE64_TODO_LEN"
  "MAX_REQUEST_BODY_BYTES"
  "MAX_JSON_DEPTH"
  "MAX_TITLE_LENGTH"
  "MIN_TITLE_LENGTH"
  "MAX_TAG_LENGTH"
  "MAX_TAGS_PER_TODO"
  "ARGON2_MEMORY_KIB"
  "ARGON2_TIME"
  "ARGON2_PARALLELISM"
  "ARGON2_SALT_LENGTH"
  "ARGON2_HASH_LENGTH"
  "PENDING_TOKEN_TTL_SEC"
  "PENDING_CLEANUP_INTERVAL_SEC"
  "PENDING_ID_BYTES"
  "INTERNAL_ID_LENGTH"
  "RATE_LIMIT_MAX_TOKENS"
  "RATE_LIMIT_REFILL_INTERVAL_SEC"
  "RATE_LIMIT_CLEANUP_INTERVAL_SEC"
  "RATE_LIMIT_MAX_BUCKETS"
  "SERVER_READ_HEADER_TIMEOUT_SEC"
  "SERVER_READ_TIMEOUT_SEC"
  "SERVER_WRITE_TIMEOUT_SEC"
  "SERVER_IDLE_TIMEOUT_SEC"
  "SERVER_MAX_HEADER_BYTES"
)

missing=()
for var in "${required_vars[@]}"; do
  if [[ -z "${!var:-}" ]]; then
    missing+=("$var")
  else
    upsert_env "$var" "${!var}"
  fi
done

if [ ${#missing[@]} -gt 0 ]; then
  if [ "$INTERACTIVE" -eq 1 ]; then
    echo ""
    echo "Missing required configuration values. Please enter them now:"
    still_missing=()
    for var in "${missing[@]}"; do
      if [[ "$var" == *"PASSWORD"* ]]; then
        value="$(prompt_input "Enter value for $var: " 1)"
      else
        value="$(prompt_input "Enter value for $var: " 0)"
      fi
      if [[ -n "$value" ]]; then
        upsert_env "$var" "$value"
        export "$var"="$value"
      else
        still_missing+=("$var")
      fi
    done
    missing=("${still_missing[@]}")
  fi
fi

if [ ${#missing[@]} -gt 0 ]; then
  echo ""
  echo "❌ Missing required configuration values (set them in .env before continuing):" >&2
  for var in "${missing[@]}"; do
    echo "   - $var" >&2
  done
  exit 1
fi

if [[ "${JWT_SECRET_MIN_LEN:-}" =~ ^[0-9]+$ ]] && [ "$JWT_SECRET_MIN_LEN" -gt 0 ]; then
  if [ ${#JWT_SECRET} -lt "$JWT_SECRET_MIN_LEN" ]; then
    echo "JWT_SECRET is shorter than JWT_SECRET_MIN_LEN; regenerating..."
    jwt_bytes=$(( (JWT_SECRET_MIN_LEN + 1) / 2 ))
    JWT_SECRET="$(openssl rand -hex "$jwt_bytes")"
    upsert_env "JWT_SECRET" "$JWT_SECRET"
  fi
fi

echo ""
echo "Configuration:"
echo "  JWT_EXPIRATION_MINUTES: $JWT_EXPIRATION_MINUTES"
echo "  ALLOWED_ORIGINS: $ALLOWED_ORIGINS"
echo "  APP_ENV: $APP_ENV"
echo "  LOG_LEVEL: $LOG_LEVEL"
echo "  ACCOUNT_LOOKUP_PEPPER: [generated]"
echo "  MASTER_KEY_ACTIVE: [generated]"
echo "  MASTER_KEY_ACTIVE_ID: $MASTER_KEY_ACTIVE_ID"
echo ""
echo "⚠️  IMPORTANT: Keep your .env file secure and do not commit it to version control!"
echo ""
echo "✅ .env file created successfully!"
echo ""
echo "📝 Next steps:"
echo "   1. Run './build.sh' to build and start services (with tests)"
echo "   2. Or run 'docker compose up -d' to start without tests"
