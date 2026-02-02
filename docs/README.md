# V6Coin Protocol 技术白皮书

## 核心技术

### 1. IPv6 扩展报头集成
- Hop-by-Hop Options Header (0x7F)
- 交易数据嵌入 IPv6 数据包
- UDP/ICMP 退化封装方案

### 2. CGA 地址映射
- Ed25519 密钥对生成
- IPv6 接口标识符 (IID) 派生
- 临时地址动态切换机制

### 3. PoC 共识机制
- 连接证明 (Proof of Connection)
- 贡献度计算模型
- 验证节点选举机制
- 前缀反垄断机制

### 4. 交易处理
- 在线实时交易
- 离线延迟交易
- 资产迁移交易
- QoS 优先级映射

### 5. 加密与安全
- Ed25519 数字签名
- AES-256-GCM 加密
- SHA-256 哈希
- 环签名匿名性

## 性能指标

| 指标 | 目标值 |
|------|--------|
| 区块间隔 | 10 秒 |
| 交易确认延迟 | ≤30 秒 |
| 单节点 TPS | ≥150 (1Mbps) |
| 全网 TPS | ≥1,500,000 (10000节点) |
| 最小单位 | 1 nano-V6 (10⁻⁹) |

## 技术参数

| 参数 | 取值 |
|------|------|
| IPv6 扩展报头类型 | Hop-by-Hop Options (0x7F) |
| 加密算法 | Ed25519 / AES-256 / SHA-256 |
| 共识机制 | PoC (Proof of Connection) |
| 总量硬顶 | 830 亿枚 |
| 挖矿分配 | 70% PoC + 15% 基础设施 + 15% 普惠空投 |
| 区块奖励减半周期 | 4 年 |
| 投票权重 | PoC 贡献度 60% + 持币量 40% |

## 开发进度

- [x] Phase 0: 项目基础设施搭建
- [ ] Phase 1: 核心协议开发
- [ ] Phase 2: 区块链与账本系统
- [ ] Phase 3: 测试网络部署
- [ ] Phase 4: 开发者生态建设
- [ ] Phase 5: 安全审计与优化
- [ ] Phase 6: 测试网公测
- [ ] Phase 7: 主网上线准备

## 文档索引

### 技术规范

#### 中文版本
- [01-IPv6 扩展报头规范](./specs/cn/01-ipv6-header-spec.md) - Hop-by-Hop Options Header 数据结构
- [02-CGA 地址映射规范](./specs/cn/02-cga-address-spec.md) - Ed25519 密钥对到 IPv6 IID 的派生
- [03-PoC 共识机制规范](./specs/cn/03-poc-consensus-spec.md) - 连接证明与贡献度计算
- [04-交易验证规范](./specs/cn/04-transaction-spec.md) - 在线/离线交易与 QoS 优先级
- [05-加密算法规范](./specs/cn/05-crypto-spec.md) - Ed25519 / AES-256 / SHA-256
- [06-P2P 网络协议规范](./specs/cn/06-p2p-protocol-spec.md) - 节点发现与消息路由
- [07-区块链数据结构规范](./specs/cn/07-blockchain-spec.md) - 区块与状态数据模型
- [08-RPC API 规范](./specs/cn/08-rpc-api-spec.md) - 节点交互接口定义

#### English Version
- [01-IPv6 Extension Header Spec](./specs/en/01-ipv6-header-spec.md) - Hop-by-Hop Options Header Data Structure
- [02-CGA Address Mapping Spec](./specs/en/02-cga-address-spec.md) - Ed25519 Key to IPv6 IID Derivation
- [03-PoC Consensus Spec](./specs/en/03-poc-consensus-spec.md) - Proof of Connection & Contribution Calculation
- [04-Transaction Spec](./specs/en/04-transaction-spec.md) - Online/Offline Transaction & QoS Priority
- [05-Cryptography Spec](./specs/en/05-crypto-spec.md) - Ed25519 / AES-256 / SHA-256
- [06-P2P Network Protocol Spec](./specs/en/06-p2p-protocol-spec.md) - Node Discovery & Message Routing
- [07-Blockchain Data Structure Spec](./specs/en/07-blockchain-spec.md) - Block & State Data Model
- [08-RPC API Spec](./specs/en/08-rpc-api-spec.md) - Node Interaction Interface

### 其他文档
- [完整白皮书](../V6Coin_Whitepaper_CN.md)
- [API 文档](./api/)
- [贡献指南](../../CONTRIBUTING.md)
