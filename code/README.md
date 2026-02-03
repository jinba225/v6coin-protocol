# V6Coin Protocol - 开发者指南

## 快速开始

### 环境要求

- Go 1.21+
- CMake 3.15+
- GCC/Clang
- OpenSSL 1.1.1+

### 获取代码

```bash
git clone https://github.com/jinba225/v6coin-protocol.git
cd v6coin-protocol
```

### 构建

```bash
# 方式 1：使用构建脚本（推荐）
bash scripts/build/go.sh

# 方式 2：手动构建 Go 版本
cd code/go
go build -o ../../build/go/bin/v6coin-node ./cmd/node

# 构建 C 版本
cd ../c
cmake -B build
cmake --build build
```

### 运行测试

```bash
# 方式 1：运行测试脚本（推荐）
bash scripts/test/run-tests.sh

# 方式 2：手动运行 Go 测试
cd code/go
go test -v -race -coverprofile=../../build/go/coverage.out ./...

# C 测试
cd code/c
ctest --test-dir build --verbose
```

## 核心模块

### 1. Crypto (密码学)

**Go**: `code/go/pkg/crypto/crypto.go`

**C**: `code/c/src/crypto.c`

功能：
- Ed25519 密钥生成与签名
- AES-256 加密/解密
- SHA-256 哈希计算
- IID 生成

### 2. IPv6 (IPv6 扩展报头)

**Go**: `code/go/pkg/ipv6/header.go`

**C**: `code/c/src/v6_coin_header.c`

功能：
- 扩展报头封装/解析
- UDP/ICMP 退化封装
- QoS 字段映射

### 3. Address (地址管理)

**Go**: `code/go/pkg/address/address.go`

**C**: `code/c/src/address.c`

功能：
- CGA 地址生成
- 临时地址切换
- 资产迁移验证

### 4. P2P (点对点网络)

**Go**: `code/go/pkg/p2p/network.go`

功能：
- 节点发现
- 消息广播
- 网络拓扑管理

### 5. Consensus (共识机制)

**Go**: `code/go/pkg/consensus/poc.go`

功能：
- PoC 连接证明
- 贡献度计算
- 验证节点选举

### 6. Blockchain (区块链)

**Go**: `code/go/pkg/blockchain/chain.go`

功能：
- 区块生成/验证
- 账本存储
- 状态管理

### 7. Tx (交易)

**Go**: `code/go/pkg/tx/transaction.go`

功能：
- 交易池管理
- 交易验证
- Gas 费计算

### 8. Wallet (钱包)

**Go**: `code/go/pkg/wallet/wallet.go`

功能：
- 密钥管理
- 地址生成
- 交易签名

## API 接口

### RPC 端口

- JSON-RPC: `9090`

### P2P 端口

- 主网: `38901`
- 测试网: `28901`

## 开发工具

### 代码格式化

```bash
# Go
go fmt ./...

# C
clang-format -i code/c/src/*.c
```

### 代码检查

```bash
# Go
golangci-lint run

# C
cppcheck --enable=all code/c/src/
```

### 性能测试

```bash
# Go
go test -bench=. -benchmem ./...

# C
./benchmarks/perf_test
```

## 贡献指南

详见 [CONTRIBUTING.md](../../CONTRIBUTING.md)

## 激励计划

详见 [Bounty Program](../../bounty/Bounty_Program_CN.md)

## 安全

发现安全问题？请查看 [SECURITY.md](../../SECURITY.md)

## 许可证

MIT License - 详见 [LICENSE](../../LICENSE)
