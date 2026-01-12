#!/usr/bin/env bash
set -euo pipefail

echo "Todo App Setup"
echo "=============="

ENV_FILE=".env"

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

# Check required dependencies
echo ""
echo "🔍 Checking dependencies..."
echo "============================"

check_dependency openssl "OpenSSL" true \
  "macOS: Usually pre-installed\n   Linux: sudo apt-get install openssl (Debian/Ubuntu) or sudo yum install openssl (RHEL/CentOS)\n   Or visit: https://www.openssl.org/source/"

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

require_cmd openssl

# Generate/load ENCRYPTION_KEY
if [[ -z "${ENCRYPTION_KEY:-}" ]]; then
  echo "Generating ENCRYPTION_KEY..."
  ENCRYPTION_KEY="$(openssl rand -hex 32)"
fi
upsert_env "ENCRYPTION_KEY" "$ENCRYPTION_KEY"

# Generate/load JWT_SECRET
if [[ -z "${JWT_SECRET:-}" ]]; then
  echo "Generating JWT_SECRET..."
  JWT_SECRET="$(openssl rand -hex 32)"
fi
upsert_env "JWT_SECRET" "$JWT_SECRET"

# Generate/load ACCOUNT_LOOKUP_PEPPER
if [[ -z "${ACCOUNT_LOOKUP_PEPPER:-}" ]]; then
  echo "Generating ACCOUNT_LOOKUP_PEPPER..."
  ACCOUNT_LOOKUP_PEPPER="$(openssl rand -hex 32)"
fi
upsert_env "ACCOUNT_LOOKUP_PEPPER" "$ACCOUNT_LOOKUP_PEPPER"

# Defaults
JWT_EXPIRATION_MINUTES="${JWT_EXPIRATION_MINUTES:-15}"
ALLOWED_ORIGIN="${ALLOWED_ORIGIN:-*}"
APP_ENV="${APP_ENV:-development}"
LOG_LEVEL="${LOG_LEVEL:-debug}"

upsert_env "JWT_EXPIRATION_MINUTES" "$JWT_EXPIRATION_MINUTES"
upsert_env "ALLOWED_ORIGIN" "$ALLOWED_ORIGIN"
upsert_env "APP_ENV" "$APP_ENV"
upsert_env "LOG_LEVEL" "$LOG_LEVEL"

echo ""
echo "Configuration:"
echo "  JWT_EXPIRATION_MINUTES: $JWT_EXPIRATION_MINUTES"
echo "  ALLOWED_ORIGIN: $ALLOWED_ORIGIN"
echo "  APP_ENV: $APP_ENV"
echo "  LOG_LEVEL: $LOG_LEVEL"
echo "  ACCOUNT_LOOKUP_PEPPER: [generated]"
echo ""
echo "⚠️  IMPORTANT: Keep your .env file secure and do not commit it to version control!"
echo ""
echo "✅ .env file created successfully!"
echo ""
echo "📝 Next steps:"
echo "   1. Run './build.sh' to build and start services (with tests)"
echo "   2. Or run 'docker compose up -d' to start without tests"
