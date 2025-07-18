#!/bin/bash

# WireGuard SD-WAN Test Runner Script
# This script runs the comprehensive test suite for the WireGuard SD-WAN project

set -e

echo "🧪 WireGuard SD-WAN Test Suite"
echo "=============================="

# Check for Go installation
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.21 or higher."
    echo "Visit: https://golang.org/dl/"
    exit 1
fi

# Check Go version
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo "✅ Go version: $GO_VERSION"

# Set environment variables for testing
export GO_ENV=test
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=wg_sdwan_test
export DB_USER=test_user
export DB_PASSWORD=test_password
export JWT_SECRET=test-jwt-secret-key
export LOG_LEVEL=debug
export CONTROLLER_HOST=localhost
export CONTROLLER_PORT=8080
export WG_INTERFACE=wg-test
export WG_SUBNET=10.200.0.0/16
export HA_ENABLED=false

echo "🔧 Setting up test environment..."

# Create test directories
mkdir -p tmp/test-data
mkdir -p tmp/test-backups
mkdir -p tmp/test-configs
mkdir -p tmp/test-logs

# Initialize Go modules if needed
if [ ! -f "go.mod" ]; then
    echo "Initializing Go module..."
    go mod init github.com/wg-hubspoke/wg-hubspoke
fi

echo "📦 Installing dependencies..."
go mod tidy

echo "🧪 Running tests..."

# Check if specific test type is requested
TEST_TYPE=${1:-all}
VERBOSE=${2:-false}
COVERAGE=${3:-false}

case $TEST_TYPE in
    "unit")
        echo "🔬 Running unit tests..."
        if [ "$COVERAGE" = "true" ]; then
            go test -v -race -coverprofile=coverage.out ./tests/unit/...
        else
            go test -v -race ./tests/unit/...
        fi
        ;;
    "integration")
        echo "🔗 Running integration tests..."
        if [ "$COVERAGE" = "true" ]; then
            go test -v -race -coverprofile=coverage.out ./tests/integration/...
        else
            go test -v -race ./tests/integration/...
        fi
        ;;
    "functional")
        echo "🎯 Running functional tests..."
        if [ "$COVERAGE" = "true" ]; then
            go test -v -race -coverprofile=coverage.out ./tests/functional/...
        else
            go test -v -race ./tests/functional/...
        fi
        ;;
    "all")
        echo "🚀 Running all tests..."
        
        echo "--- Unit Tests ---"
        go test -v -race ./tests/unit/...
        
        echo "--- Integration Tests ---"
        go test -v -race ./tests/integration/...
        
        echo "--- Functional Tests ---"
        go test -v -race ./tests/functional/...
        
        if [ "$COVERAGE" = "true" ]; then
            echo "📊 Generating coverage report..."
            go test -v -race -coverprofile=coverage.out ./...
            go tool cover -html=coverage.out -o coverage.html
            go tool cover -func=coverage.out
            echo "📈 Coverage report generated: coverage.html"
        fi
        ;;
    *)
        echo "❌ Invalid test type: $TEST_TYPE"
        echo "Valid options: unit, integration, functional, all"
        exit 1
        ;;
esac

echo "🧹 Cleaning up test environment..."

# Remove test files
rm -rf tmp/test-data
rm -rf tmp/test-backups
rm -rf tmp/test-configs
rm -rf tmp/test-logs

# Remove coverage files if not requested
if [ "$COVERAGE" != "true" ]; then
    rm -f coverage.out
    rm -f coverage.html
fi

echo "✅ Test execution completed successfully!"
echo ""
echo "Test Summary:"
echo "- Test Type: $TEST_TYPE"
echo "- Verbose: $VERBOSE"
echo "- Coverage: $COVERAGE"
echo ""
echo "To run specific test types:"
echo "  ./run_tests.sh unit        # Run unit tests only"
echo "  ./run_tests.sh integration # Run integration tests only"
echo "  ./run_tests.sh functional  # Run functional tests only"
echo "  ./run_tests.sh all true true # Run all tests with verbose output and coverage"