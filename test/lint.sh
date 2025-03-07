#!/bin/bash

# Directory containing all services
SERVICES_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/services"

# Default to running on all services
run_all=true
specific_services=()
has_errors=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --help|-h)
            echo "Usage: $0 [service_name1] [service_name2] ..."
            echo "If no service names are provided, lint will be run on all services."
            exit 0
            ;;
        *)
            run_all=false
            specific_services+=("$1")
            shift
            ;;
    esac
done

# Function to check if golangci-lint is installed, install if not
check_golangci_lint() {
    if ! command -v golangci-lint &> /dev/null; then
        echo "golangci-lint not found, installing..."
        curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.64.6
        
        # Verify installation
        if ! command -v golangci-lint &> /dev/null; then
            echo "Failed to install golangci-lint. Please install it manually."
            return 1
        fi
    fi
    return 0
}

# Function to run lint on a service
run_lint() {
    local service_dir="$1"
    local service_name=$(basename "$service_dir")
    
    if [[ -f "$service_dir/go.mod" ]]; then
        echo "Running lint for Go service: $service_name"
        (cd "$service_dir" && golangci-lint run ./...)
        local lint_result=$?
        
        if [ $lint_result -ne 0 ]; then
            echo "Lint failed for service: $service_name"
            has_errors=true
        fi
    else
        echo "Skipping lint for service: $service_name"
    fi
}

# Check golangci-lint early
if ! check_golangci_lint; then
    echo "Failed to ensure golangci-lint is available."
    exit 1
fi

# Run lint on services
if $run_all; then
    for service_dir in "$SERVICES_DIR"/*; do
        if [[ -d "$service_dir" ]]; then
            run_lint "$service_dir"
        fi
    done
else
    for service_name in "${specific_services[@]}"; do
        service_dir="$SERVICES_DIR/$service_name"
        if [[ -d "$service_dir" ]]; then
            run_lint "$service_dir"
        fi
    done
fi

# Return error code if any service had lint errors
if $has_errors; then
    echo "Lint errors were found in one or more services."
    exit 1
else
    echo "All lint checks passed successfully."
    exit 0
fi
