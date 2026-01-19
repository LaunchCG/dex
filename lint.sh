#!/usr/bin/env bash

# Unified linting script for the project
# Usage: ./lint.sh [target] [--embedded]
# target: frontend, backend, or all (default)
# --embedded: Always exit 0 and redirect all output to stdout (for Claude commands)

set -e

TARGET=${1:-all}
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

# Backend linting
lint() {
    run_command "uv run ruff check ." "Ruff Lint Check"
    run_command "uv run flake8 ." "Flake8 Style Check"
    run_command "uv run black . --check" "Black Format Check"
    run_command "uv run isort . --check-only" "Import Sort Check"
    run_command "uv run mypy ." "MyPy Type Check"

    echo "=== Linting Complete ==="
}

# Main execution
echo "Starting linting for: $TARGET"
echo "Embedded mode: $EMBEDDED"
echo ""

lint

echo ""
echo "=== All Linting Complete ==="

# In embedded mode, always exit successfully
if [ "$EMBEDDED" = true ]; then
    exit 0
fi
