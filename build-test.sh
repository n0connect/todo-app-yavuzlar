#!/bin/bash

# build-test.sh - Professional build and test automation script
# This script builds the project in Docker, runs all tests, and starts the system only if all tests pass

set -euo pipefail  # Exit on error, undefined vars, and pipe failures

# Color codes for terminal output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_warn() {
    echo -e "${YELLOW}[WARNING]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

# Progress bar function
show_progress() {
    local current=$1
    local total=$2
    local width=50
    local percentage=$((current * 100 / total))
    local filled=$((width * current / total))
    local unfilled=$((width - filled))
    
    printf "\r["
    printf "%*s" $filled | tr ' ' '='
    printf "%*s" $unfilled | tr ' ' '-'
    printf "] %d%% (%d/%d)" $percentage $current $total
}

# Function to check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    local missing_deps=()
    
    if ! command -v docker &> /dev/null; then
        missing_deps+=("docker")
    fi
    
    if ! command -v docker-compose &> /dev/null && ! command -v docker compose &> /dev/null; then
        missing_deps+=("docker-compose")
    fi
    
    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        log_error "Missing required dependencies: ${missing_deps[*]}"
        exit 1
    fi
    
    log_success "All prerequisites are available"
}

# Function to build Docker images
build_docker_images() {
    log_info "Building Docker images..."
    
    if command -v docker-compose &> /dev/null; then
        docker-compose build
    elif command -v docker compose &> /dev/null; then
        docker compose build
    else
        log_error "Neither docker-compose nor docker compose is available"
        exit 1
    fi
    
    if [ $? -eq 0 ]; then
        log_success "Docker images built successfully"
    else
        log_error "Failed to build Docker images"
        exit 1
    fi
}

# Function to run backend Go tests
run_backend_tests() {
    log_info "Running backend Go tests..."
    
    # Navigate to backend directory
    cd backend
    
    # Run tests and capture output
    local test_output
    test_output=$(go test ./... 2>&1)
    local test_exit_code=$?
    
    # Print test output
    echo "$test_output"
    
    if [ $test_exit_code -eq 0 ]; then
        log_success "All backend tests PASSED"
        cd ..
        return 0
    else
        log_error "Backend tests FAILED"
        cd ..
        return 1
    fi
}

# Function to run any additional tests
run_additional_tests() {
    log_info "Running additional tests..."
    
    # Add any additional test suites here
    # For now, we'll just return success
    log_success "Additional tests completed"
}

# Function to start the system
start_system() {
    log_info "Starting the system..."
    
    if command -v docker-compose &> /dev/null; then
        docker-compose up -d
    elif command -v docker compose &> /dev/null; then
        docker compose up -d
    else
        log_error "Neither docker-compose nor docker compose is available"
        exit 1
    fi
    
    if [ $? -eq 0 ]; then
        log_success "System started successfully"
        log_info "Services are starting in the background..."
        sleep 5  # Give services time to initialize
        
        # Check if services are running
        if command -v docker-compose &> /dev/null; then
            docker-compose ps
        elif command -v docker compose &> /dev/null; then
            docker compose ps
        fi
    else
        log_error "Failed to start the system"
        exit 1
    fi
}

# Function to clean up previous containers
cleanup_containers() {
    log_info "Cleaning up previous containers..."
    
    if command -v docker-compose &> /dev/null; then
        docker-compose down --remove-orphans 2>/dev/null || true
    elif command -v docker compose &> /dev/null; then
        docker compose down --remove-orphans 2>/dev/null || true
    fi
    
    log_success "Previous containers cleaned up"
}

# Function to display usage
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo "Options:"
    echo "  -h, --help     Show this help message"
    echo "  --clean        Clean up previous containers before building"
    echo "  --skip-tests   Skip running tests (just build and start)"
    echo ""
    echo "Example:"
    echo "  $0             # Build, test, and start the system"
    echo "  $0 --clean     # Clean, build, test, and start"
    echo "  $0 --skip-tests # Just build and start without testing"
}

# Main execution
main() {
    local clean_flag=false
    local skip_tests_flag=false
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                usage
                exit 0
                ;;
            --clean)
                clean_flag=true
                shift
                ;;
            --skip-tests)
                skip_tests_flag=true
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                usage
                exit 1
                ;;
        esac
    done
    
    log_info "Starting build and test process..."
    
    # Check prerequisites
    check_prerequisites
    
    # Clean up if requested
    if [ "$clean_flag" = true ]; then
        cleanup_containers
    fi
    
    # Build Docker images
    build_docker_images
    
    # Run tests if not skipped
    if [ "$skip_tests_flag" = false ]; then
        log_info "Starting test sequence..."
        
        # Run backend tests
        if ! run_backend_tests; then
            log_error "Backend tests failed. System will NOT be started."
            exit 1
        fi
        
        # Run additional tests
        if ! run_additional_tests; then
            log_error "Additional tests failed. System will NOT be started."
            exit 1
        fi
        
        log_success "All tests PASSED. Proceeding to start the system."
    else
        log_warn "Tests are skipped. Proceeding directly to start the system."
    fi
    
    # Start the system
    start_system
    
    log_success "Build and test process completed successfully!"
    log_info "System is now running. Check service status with 'docker compose ps'"
}

# Run main function with all arguments
main "$@"