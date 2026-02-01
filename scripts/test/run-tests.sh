#!/bin/bash
set -e

echo "Running tests for V6Coin Protocol..."

# Go tests
echo "Running Go tests..."
cd code/go
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
echo "✓ Go tests complete"

# C tests
echo "Running C tests..."
cd ../c
ctest --test-dir build --verbose
echo "✓ C tests complete"

echo "All tests passed!"
