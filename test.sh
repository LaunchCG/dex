#!/usr/bin/env bash

# Test runner script for the project
# Usage: ./test.sh [--embedded]
# --embedded: Always exit 0 and redirect all output to stdout (for Claude commands)

set -e

EMBEDDED=false

# Check for --embedded flag
for arg in "$@"; do
    if [ "$arg" = "--embedded" ]; then
        EMBEDDED=true
    fi
done

# Function to run command with proper error handling
run_command() {
    local cmd=$1
    local description=$2

    echo "=== $description ==="

    if [ "$EMBEDDED" = true ]; then
        # In embedded mode, capture all output and always succeed
        eval "$cmd" 2>&1 || true
    else
        # In normal mode, let commands fail normally
        eval "$cmd"
    fi
}

# Run tests
test() {
    run_command "uv run pytest" "Running Tests"

    echo "=== Tests Complete ==="
}

# Main execution
echo "Starting tests"
echo "Embedded mode: $EMBEDDED"
echo ""

test

echo ""
echo "=== All Tests Complete ==="

# In embedded mode, always exit successfully
if [ "$EMBEDDED" = true ]; then
    exit 0
fi
