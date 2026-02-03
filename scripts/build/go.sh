#!/bin/bash
# 构建 Go 项目并输出到 build/ 目录

set -e

mkdir -p ../../build/go/bin

cd code/go
echo "Building V6Coin Node..."

# 构建主程序
go build -o ../../build/go/bin/v6coin-node ./cmd/node

echo "✓ Build complete: ../../build/go/bin/v6coin-node"
