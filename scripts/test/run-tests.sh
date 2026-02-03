#!/bin/bash
set -e

# 创建构建目录
mkdir -p ../../build/go

echo "Running tests for V6Coin Protocol..."

# Go tests
echo "Running Go tests..."
cd code/go
go test -v -race -coverprofile=../../build/go/coverage.out ./...
go tool cover -html=../../build/go/coverage.out -o ../../build/go/coverage.html
echo "✓ Go tests complete"

# C tests
echo "Running C tests..."
cd ../c
ctest --test-dir build --verbose
echo "✓ C tests complete"

echo "All tests passed!"
