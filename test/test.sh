#!/bin/bash

# Define services directory relative to project root
SERVICES_DIR=$(realpath "$(dirname "$0")/../services")
SPECIFIC_TESTS=()
TARGET_SERVICE=""

# Display help information
show_help() {
    echo "Usage: $0 [options] [service_name]"
    echo "Run tests for all services, a specific service, or specific tests"
    echo
    echo "Options:"
    echo "  -h, --help                  Show this help message"
    echo "  -t, --test TEST_NAME        Run specific test(s), can be used multiple times"
    echo "  -v, --verbose               Enable verbose output"
    echo
    echo "Examples:"
    echo "  $0                          Test all services"
    echo "  $0 profiler                 Test only the profiler service"
    echo "  $0 -t TestUserCreate        Run specific test across all services"
    echo "  $0 profiler -t TestUserCreate  Run specific test in profiler service"
}

# Parse command line arguments
VERBOSE=false
while [[ $# -gt 0 ]]; do
    key="$1"
    case $key in
        -h|--help)
            show_help
            exit 0
            ;;
        -t|--test)
            SPECIFIC_TESTS+=("$2")
            shift 2
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        *)
            if [ -z "$TARGET_SERVICE" ]; then
                TARGET_SERVICE="$1"
                shift
            else
                echo "Unknown option: $1"
                show_help
                exit 1
            fi
            ;;
    esac
done

# Set verbose flag for go test if needed
VERBOSE_FLAG=""
if $VERBOSE; then
    VERBOSE_FLAG="-v"
fi

# Build the test filter if specific tests are provided
TEST_FILTER=""
if [ ${#SPECIFIC_TESTS[@]} -gt 0 ]; then
    TEST_FILTER="-run="
    for test in "${SPECIFIC_TESTS[@]}"; do
        TEST_FILTER="${TEST_FILTER}${test}|"
    done
    # Remove the trailing pipe
    TEST_FILTER=${TEST_FILTER%|}
fi

# Find and run tests in all service directories
echo "Looking for services in ${SERVICES_DIR}"
SERVICES_COUNT=0
FAILED_SERVICES=()

if [ ! -d "$SERVICES_DIR" ]; then
    echo "Error: Services directory not found at $SERVICES_DIR"
    exit 1
fi

for service_dir in "$SERVICES_DIR"/*; do
    if [ -d "$service_dir" ] && [ -f "$service_dir/go.mod" ]; then
        SERVICE_NAME=$(basename "$service_dir")
        
        # Skip if a target service is specified and this is not it
        if [ -n "$TARGET_SERVICE" ] && [ "$SERVICE_NAME" != "$TARGET_SERVICE" ]; then
            continue
        fi
        
        echo "----------------------------------------"
        echo "Running tests for service: $SERVICE_NAME"
        echo "----------------------------------------"
        
        cd "$service_dir" || continue
        
        if go test $VERBOSE_FLAG ./... $TEST_FILTER; then
            echo "✅ $SERVICE_NAME tests passed"
        else
            echo "❌ $SERVICE_NAME tests failed"
            FAILED_SERVICES+=("$SERVICE_NAME")
        fi
        
        SERVICES_COUNT=$((SERVICES_COUNT+1))
    fi
done

# Check if we were looking for a specific service but didn't find it
if [ -n "$TARGET_SERVICE" ] && [ "$SERVICES_COUNT" -eq 0 ]; then
    echo "Error: Service '$TARGET_SERVICE' not found"
    exit 1
fi

echo "----------------------------------------"
echo "Test Summary"
echo "----------------------------------------"
echo "Services tested: $SERVICES_COUNT"

if [ ${#FAILED_SERVICES[@]} -eq 0 ]; then
    echo "All tests passed! 🎉"
    exit 0
else
    echo "Failed services: ${FAILED_SERVICES[*]}"
    echo "❌ Some tests failed"
    exit 1
fi