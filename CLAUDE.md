# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

V6Coin Protocol 是一个基于 IPv6 的物联网原生价值激励层区块链协议。核心特性包括：
- 原生 IPv6 地址即钱包（通过 Ed25519 密钥生成 64 位 IID）
- PoC（连接证明）共识机制，替代 PoW 的绿色共识
- IPv6 扩展报头嵌入交易（Hop-by-Hop Options Header 0x7F）
- UDP/ICMP 退化封装适配现实网络环境

## 常用命令

### 构建
```bash
# 方式 1：使用构建脚本（推荐）
bash scripts/build/go.sh

# 方式 2：手动构建
cd code/go
go build -o ../../build/go/bin/v6coin-node ./cmd/node

# C 版本（用于嵌入式系统集成）
cd code/c
cmake -B build
cmake --build build
```

### 测试
```bash
# 运行测试脚本（输出到 build/go/）
bash scripts/test/run-tests.sh

# 手动运行测试
cd code/go
go test -v -race -coverprofile=../../build/go/coverage.out ./...

# 运行单个包测试
go test -v ./pkg/crypto

# 运行集成测试
go test -v ./tests/integration

# 查看覆盖率报告
open build/go/coverage.html  # macOS
```

### 运行节点
```bash
# 启动节点
./build/go/bin/v6coin-node start -port 38901 -datadir /tmp/v6coin -validator

# 查看节点状态
./build/go/bin/v6coin-node status

# 从助记词恢复钱包
./build/go/bin/v6coin-node start -mnemonic "word1 word2 ..." -password yourpassword
```

### 代码质量
```bash
# 格式化 Go 代码
go fmt ./...

# 运行 linter
golangci-lint run
```

## 代码架构

### 核心模块设计

```
code/go/
├── cmd/node/          # 主节点入口，负责组件协调
├── pkg/
│   ├── crypto/        # Ed25519 签名、AES-256 加密、SHA-256 哈希、助记词
│   ├── ipv6/          # IPv6 扩展报头封装/解析、UDP 退化封装
│   ├── address/       # CGA 地址生成、临时地址切换、资产迁移验证
│   ├── consensus/     # PoC 共识、区块/交易结构、验证器选举、前缀监控
│   ├── blockchain/    # 区块链管理、分叉处理、状态执行
│   ├── state/         # 状态数据库（MPT + 内存）、交易执行器
│   ├── tx/            # 交易池、交易验证、Gas 费计算
│   ├── p2p/           # P2P 网络、节点发现、消息路由、连接管理
│   ├── wallet/        # 钱包、密钥管理、交易构建和签名
│   └── rpc/           # JSON-RPC 服务（Gin 框架）
└── tests/integration/ # 端到端集成测试
```

### 关键设计模式

**1. IPv6 地址即钱包**
- 通过 `crypto.GenerateIID()` 从 Ed25519 私钥生成 64 位 IID
- 结合网络前缀形成完整 128 位 IPv6 地址
- 见 `pkg/address/address.go:GenerateAddress()`

**2. PoC 共识机制**
- 基于节点贡献度评分：在线时长（60%）、转发量（30%）、丢包率（10%）
- 前缀反垄断：同一 /64 前缀设备越多，单个奖励越少（指数衰减 e^(-n/10)）
- 验证器集合取 Top 100 贡献节点，轮流出块
- 见 `pkg/consensus/consensus.go` 和 `pkg/consensus/poc.go`

**3. 区块链与状态分离**
- `blockchain.BlockChain` 负责区块存储和链式验证
- `state.StateDB` 负责账户状态管理（Merkle Patricia Trie）
- `state.TransactionExecutor` 执行交易并生成收据
- 见 `pkg/blockchain/chain.go:80` AddBlock 方法

**4. P2P 网络层**
- 消息格式：版本(2) + 类型(1) + 源地址(16) + 目标地址(16) + 时间戳(8) + ID(16) + TTL(1) + 跳数(1) + 载荷
- 支持 IPv6 扩展报头（首选）和 UDP 端口 38901（退化模式）
- 见 `pkg/p2p/message.go` 和 `pkg/p2p/network.go`

**5. 交易类型**
- `TxTypeOnline (0x01)`: 在线交易，嵌入 IPv6 扩展报头
- `TxTypeOffline (0x02)`: 离线交易，7 天有效期
- `TxTypeMigration (0x03)`: 资产迁移
- `TxTypeStake/Unstake (0x04/0x05)`: 质押/解押
- `TxTypeGovernance/Vote (0x06/0x07)`: 治理提案/投票

### 节点启动流程

主节点 `cmd/node/main.go` 启动时依次初始化：

1. **StateDB**: 内存状态数据库
2. **BlockChain**: 创世区块 → 区块链实例
3. **ConsensusEngine**: PoC 共识引擎
4. **TxPool**: 交易池
5. **Wallet**: 钱包（可选，提供助记词时）
6. **P2PNetwork**: 连接管理器
7. **RPCServer**: JSON-RPC 服务（端口 9090）

后台 Worker 协程：
- `blockWorker`: 处理新区块（验证、执行、状态更新）
- `transactionWorker`: 处理新交易（加入交易池）
- `syncWorker`: 定期同步区块链（10 秒间隔）
- `consensusWorker`: 验证节点定期提议新区块（10 秒间隔）
- `peerDiscovery`: 节点发现和连接（30 秒间隔）

### RPC 接口

- 默认端口: `9090`
- 框架: Gin (github.com/gin-gonic/gin)
- 中间件: Logger, CORS, Recovery, Request ID
- 路由定义: `pkg/rpc/routes.go`
- 处理器: `pkg/rpc/handler/` 目录（account, block, network, node, transaction）

## 技术规范参考

- IPv6 扩展报头规范: `docs/specs/cn/ipv6_header_spec.md`
- CGA 地址映射: `docs/specs/cn/cga_address_spec.md`
- PoC 共识机制: `docs/specs/cn/poc_consensus_spec.md`
- 交易验证流程: `docs/specs/cn/tx_validation_spec.md`
- 完整白皮书: `docs/whitepaper/V6Coin_Whitepaper_CN.md`

## 常量与配置

**区块**: 10 秒间隔，初始奖励 100 V6，每 4 年减半
**交易**: 最小单位 1 nano-V6 (10⁻⁹ V6)，离线交易有效期 7 天
**网络**: 默认端口 38901，最大连接数 50，最大跳数 64
**共识**: 100 个验证节点，在线窗口 90 天，最大丢包率 30%
**治理**: 投票权重 = PoC 贡献度(60%) + 持币量(40%)，单节点上限 5%

## 测试策略

- 单元测试: 每个 `pkg` 目录下都有 `*_test.go` 文件
- 集成测试: `tests/integration/` 目录，测试完整交易流程
- 测试数据: 使用 `github.com/stretchr/testify` 断言库
- 覆盖率: 使用 `go test -cover` 生成覆盖率报告
