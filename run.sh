#!/bin/bash
# run.sh - Project runner

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

log() { echo -e "${BLUE}[INFO]${NC} $1"; }
ok()  { echo -e "${GREEN}[OK]${NC} $1"; }
err() { echo -e "${RED}[ERR]${NC} $1"; }

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

DC="docker compose"
command -v docker-compose &>/dev/null && DC="docker-compose"

usage() {
    cat << EOF
Usage: $0 [OPTIONS]

OPTIONS:
    --test-run   Test first, then run if all tests pass
    --test       Run tests only (exit after)
    --clean      Clean everything before other operations
    -h, --help   Show this message

EXAMPLES:
    $0                  # Build and run
    $0 --test           # Run tests only
    $0 --test-run       # Test then run
    $0 --clean          # Clean and build/run
    $0 --clean --test   # Clean then test
EOF
}

clean_all() {
    log "Cleaning everything..."

    $DC -f docker-compose.test.yaml down -v --rmi local 2>/dev/null || true
    $DC -f docker-compose.yaml down -v --rmi local 2>/dev/null || true

    docker images --filter "reference=*todo*" -q | xargs -r docker rmi -f 2>/dev/null || true
    docker builder prune -f 2>/dev/null || true

    rm -rf "$SCRIPT_DIR/test-results" 2>/dev/null || true

    ok "Clean complete"
}

run_tests() {
    log "Running tests..."
    mkdir -p "$SCRIPT_DIR/test-results"

    $DC -f docker-compose.test.yaml build
    $DC -f docker-compose.test.yaml up --abort-on-container-exit --exit-code-from test-runner
    local exit_code=$?

    $DC -f docker-compose.test.yaml down -v

    return $exit_code
}

build_and_run() {
    log "Building..."
    $DC -f docker-compose.yaml build

    log "Starting services..."
    $DC -f docker-compose.yaml up -d

    ok "Services running"
    $DC -f docker-compose.yaml ps
}

# Parse args
DO_CLEAN=false
DO_TEST=false
DO_RUN=true

while [[ $# -gt 0 ]]; do
    case $1 in
        --clean)    DO_CLEAN=true; shift ;;
        --test)     DO_TEST=true; DO_RUN=false; shift ;;
        --test-run) DO_TEST=true; DO_RUN=true; shift ;;
        -h|--help)  usage; exit 0 ;;
        *)          err "Unknown: $1"; usage; exit 1 ;;
    esac
done

# Execute
if $DO_CLEAN; then
    clean_all
fi

if $DO_TEST; then
    if ! run_tests; then
        err "Tests failed"
        exit 1
    fi
    ok "All tests passed"
fi

if $DO_RUN; then
    build_and_run
fi
