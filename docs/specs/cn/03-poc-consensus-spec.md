# PoC 共识规范

**版本**: v1.0
**日期**: 2026-02-01

---

## 1. 概述

V6Coin 使用连接证明（Proof of Connection，PoC）作为共识机制，通过节点对网络的连接贡献来达成共识。PoC 旨在：

- **能源高效**：避免 PoW 的能源浪费
- **去中心化**：防止大节点垄断
- **公平性**：长期贡献者获得更多权重

### 设计目标

1. **绿色共识**：基于网络连接贡献，而非计算能力
2. **贡献度公平**：长期提供网络服务的"劳动节点"获得更多话语权
3. **前缀反垄断**：防止单个前缀的设备过度集中
4. **动态验证集**：根据贡献度动态调整验证节点

---

## 2. 核心概念

### 2.1 连接证明（Proof of Connection）

PoC 证明节点持续为网络提供有效服务，包括：
- 数据转发
- 交易广播
- 区块验证
- 网络路由

### 2.2 贡献度计算（Contribution Score）

节点的贡献度由三个维度加权计算：

1. **在线时间权重**：60%
   - 节点在在线窗口（90 天）内的累计在线时间
   - 计算公式：`min(onlineTime / windowTime, 1.0) * 0.6`

2. **丢包率权重**：10%
   - 节点的网络稳定性，丢包率越低分数越高
   - 计算公式：`(1.0 - packetLossRate) * 0.1`

3. **转发量权重**：30%
   - 节点转发的数据总量，相对于网络平均值归一化
   - 计算公式：`min(forwardedBytes / 1GB, 1.0) * 0.3`

**总贡献度公式**：
```
ContributionScore = (OnlineTimeWeight * 0.6) +
                   (PacketLossWeight * 0.1) +
                   (ForwardedWeight * 0.3)
```

### 2.3 验证器选举（Validator Election）

1. **验证器数量**：100 个顶级验证器
2. **选举方式**：按贡献度降序排序
3. **轮换机制**：每个区块高度轮换验证器
4. **当前验证器**：根据区块高度取模选择

### 2.4 前缀反垄断（Prefix Anti-Monopoly）

1. **设备数量限制**：每个 /64 前缀最多 50 个设备
2. **奖励衰减**：前缀内设备越多，单节点奖励越低
3. **计算公式**：
   ```
   RewardMultiplier = max(0.01, R0 * e^(-n/10))
   ```
   其中：
   - `R0` = 1.0（基础奖励乘数）
   - `n` = 该前缀的设备数量
   - `e` = 自然对数底（约 2.718）

---

## 3. 数据结构定义

### 3.1 区块（Block）

```go
type Block struct {
    Header        *BlockHeader        // 区块头
    Transactions  []*Transaction      // 交易列表
    ValidatorSigs [][]byte       // 所有验证器签名
}
```

### 3.2 区块头（BlockHeader）

```go
type BlockHeader struct {
    Version       uint16         // 协议版本（0x0100）
    Height        uint64          // 区块高度
    PrevBlockHash []byte         // 前一个区块的哈希
    MerkleRoot    []byte         // 交易 Merkle 树根
    Timestamp     uint64          // 区块时间戳（Unix）
    ValidatorID   p2p.PeerID      // 创建区块的验证器 ID
    StateRoot     []byte         // 状态树根
    Signature     []byte         // 验证器签名
}
```

### 3.3 交易（Transaction）

```go
type Transaction struct {
    Version   uint16    // 交易版本
    Type      TxType     // 交易类型
    From      net.IP     // 发送方地址（16 字节）
    To        net.IP     // 接收方地址（16 字节）
    Amount    uint64     // 金额（nano-V6）
    Fee       uint64     // 手续费（nano-V6）
    Nonce     uint64     // 随机数
    Timestamp uint64    // 交易时间戳
    Signature []byte     // Ed25519 签名（32 字节）
    Data      []byte     // 可选交易数据
}
```

### 3.4 节点贡献（NodeContribution）

```go
type NodeContribution struct {
    NodeID        p2p.PeerID      // 节点 ID
    OnlineTime    time.Duration    // 总在线时间
    LastOnline    time.Time        // 最后在线时间
    PacketLoss    float64          // 丢包率（0.0 - 1.0）
    Forwarded     uint64          // 转发字节数
    Score         float64          // 贡献度
    RewardBalance uint64          // 待领取奖励余额
}
```

### 3.5 验证器集（ValidatorSet）

```go
type ValidatorSet struct {
    Validators      []p2p.PeerID    // 当前验证器列表（按贡献度排序）
    RoundRobinIndex int             // 轮换索引
}
```

### 3.6 前缀监控器（PrefixMonitor）

```go
type PrefixMonitor struct {
    prefixCounts map[string]int    // /64 前缀 -> 设备数量
}
```

---

## 4. 共识流程

### 4.1 区块生成

1. **验证器选择**：
   - 计算当前区块高度：`currentHeight + 1`
   - 选择验证器：`validatorIndex = currentHeight % 100`
   - 从验证器集中获取验证器 ID

2. **交易打包**：
   - 从交易池中选取交易
   - 按手续费优先级排序
   - 达到区块最大限制（或 10 秒超时）

3. **区块验证**：
   - 验证所有交易签名
   - 验证所有交易余额
   - 计算 Merkle 根

4. **区块签名**：
   - 验证器对区块头签名
   - 收集多个验证器签名

5. **区块广播**：
   - 通过 P2P 网络广播新区块
   - 节点验证后添加到本地链

### 4.2 区块验证

1. **版本验证**：
   - 检查 `version == 0x0100`

2. **高度验证**：
   - 检查 `height == prevBlock.height + 1`

3. **前一个区块验证**：
   - 检查 `prevBlockHash` 是否匹配前一个区块的哈希

4. **交易验证**：
   - 验证所有交易签名
   - 验证所有交易余额
   - 检查交易唯一性

5. **Merkle 根验证**：
   - 重新计算交易 Merkle 树
   - 检查 `merkleRoot` 是否匹配

6. **时间戳验证**：
   - 检查 `timestamp` >= 前一个区块时间戳
   - 检查 `timestamp` <= 当前时间 + 2 分钟（时钟漂移容限）

### 4.3 最长链选择

1. **链验证**：
   - 从创世区块到候选区块完整验证
   - 检查所有区块签名有效性

2. **累计工作量**：
   - 每个区块代表 1 个单位工作量
   - 计算两条链的总工作量

3. **选择最长链**：
   - 选择累计工作量最多的链
   - 如果工作量相同，选择时间戳较新的链

4. **链切换**：
   - 当发现更长链时，自动切换
   - 孤块保存在本地用于潜在重组

---

## 5. 奖励机制

### 5.1 区块奖励

1. **基础奖励**：每个区块 100 V6
2. **交易手续费**：区块中所有交易手续费的总和
3. **总区块奖励**：
   ```
   TotalReward = BaseReward + TransactionFees
   ```

### 5.2 挖矿分配

- **70% PoC 贡献**：根据节点贡献度分配
- **15% 基础设施**：用于开发、测试等
- **15% 普惠空投**：奖励早期参与者

### 5.3 减半周期

每 4 年（约 1,401,600 个区块），区块奖励减半一次：
- 第一次减半：区块 1,401,600
- 第二次减半：区块 2,803,200
- 以此类推...

### 5.4 前缀反垄断奖励调整

1. **计算乘数**：根据前缀设备数量计算
   ```
   multiplier = CalculateRewardMultiplier(prefix)
   ```

2. **最终奖励**：
   ```
   FinalReward = BaseReward * multiplier
   ```

3. **奖励示例**：
   - 1 个设备：`100 * 1.0 = 100 V6`
   - 10 个设备：`100 * 0.91 = 91 V6`
   - 30 个设备：`100 * 0.74 = 74 V6`
   - 50 个设备：`100 * 0.51 = 51 V6`

---

## 6. 安全机制

### 6.1 Sybil 攻击防护

1. **身份绑定**：每个 IP 地址只能注册一个节点 ID
2. **贡献度计算**：需要持续在线时间，短期节点无法获得高分
3. **最低在线时间**：必须在线至少 24 小时才能获得贡献度

### 6.2 双花防护

1. **交易池去重**：相同哈希的交易只保留一个
2. **余额验证**：每个交易验证发送方余额
3. **最长链原则**：只接受最长链上的交易

### 6.3 时间戳攻击防护

1. **区块时间限制**：
   - 区块时间戳不能超前于当前时间 + 2 分钟
   - 拒绝未来时间戳的区块

2. **交易时间限制**：
   - 交易时间戳不能超前于当前时间 + 5 分钟

### 6.4 网络分割防护

1. **最长链选择**：自动处理网络分割，最终选择最长的链
2. **验证器签名**：需要足够多的验证器签名确认
3. **重组限制**：限制重组深度，防止长期链分裂

---

## 7. 参数配置

### 7.1 共识参数

| 参数名 | 取值 | 说明 |
|--------|------|------|
| BlockInterval | 10 秒 | 区块产生间隔 |
| ValidatorCount | 100 | 验证器数量 |
| ConfirmationBlocks | 3 | 交易确认所需区块数 |
| OnlineWindow | 90 天 | 贡献度计算时间窗口 |
| MaxPacketLossRate | 30% | 最大允许丢包率 |
| MaxPrefixDevices | 50 | 单个前缀最大设备数 |
| RevenueDecayFactor | 10.0 | 奖励衰减因子 |
| InitialBlockReward | 100 V6 | 初始区块奖励 |
| HalvingYears | 4 年 | 奖励减半周期 |

### 7.2 前缀反垄断参数

| 参数 | 取值 | 说明 |
|------|------|------|
| MaxLogicalSubnets | 8 | 每个核心地址最多 8 个逻辑子网 |
| MinMigrationInterval | 1 小时 | 临时地址最小切换间隔 |
| MaxMigrationInterval | 30 天 | 临时地址最大有效期 |

---

## 8. 代码示例

### 8.1 创建共识引擎

```go
import (
    "github.com/jinba225/v6coin-protocol/pkg/consensus"
)

// 创建新的共识引擎
engine := consensus.NewConsensusEngine()

// 添加交易
tx := &Transaction{
    Type:      TxTypeOnline,
    From:      senderAddr,
    To:        receiverAddr,
    Amount:    1000 * 1e9, // 1000 V6 = 1000 * 10^9 nano-V6
    Timestamp: uint64(time.Now().Unix()),
}

err := engine.AddTransaction(tx)
if err != nil {
    log.Fatal("Failed to add transaction:", err)
}
```

### 8.2 更新节点贡献度

```go
// 更新节点在线时间
nodeID := p2p.PeerID("node1")

// 节点在线了 1 小时
engine.UpdateContribution(nodeID, func(nc *NodeContribution) {
    nc.UpdateOnlineTime(1 * time.Hour)
})
```

### 8.3 计算贡献度

```go
// 计算节点的贡献度
engine.UpdateContribution(nodeID, func(nc *NodeContribution) {
    // 更新丢包率（假设测量到 0.02，即 2%）
    nc.UpdatePacketLoss(0.02)
    
    // 更新转发量（假设转发了 100MB 数据）
    nc.AddForwardedBytes(100 * 1024 * 1024)
    
    // 贡献度自动重新计算
    // onlineTime: 24小时 / 90天 = 0.00111 (加权 60%)
    // packetLoss: (1.0 - 0.02) = 0.98 (加权 10%)
    // forwarded: 100MB / 1GB = 0.10 (加权 30%)
    // Score = 0.00111*0.6 + 0.98*0.1 + 0.10*0.3 = 0.0097
})
```

### 8.4 前缀注册和反垄断

```go
// 注册设备前缀
deviceAddr := net.ParseIP("2001:db8::1")

// 注册前缀（增加设备计数）
engine.prefixMonitor.RegisterPrefix(deviceAddr)

// 计算奖励乘数（假设该前缀已有 20 个设备）
multiplier := engine.prefixMonitor.CalculateRewardMultiplier(deviceAddr)
fmt.Printf("Reward multiplier: %.2f", multiplier) // 输出: 0.82
```

### 8.5 区块验证

```go
// 验证新接收的区块
newBlock := &Block{
    Header: &BlockHeader{
        Version:       0x0100,
        Height:        currentHeight + 1,
        PrevBlockHash: previousBlock.Hash(),
        MerkleRoot:    CalculateMerkleRoot(transactions),
        Timestamp:     uint64(time.Now().Unix()),
        ValidatorID:   currentValidator.ID,
    },
    Transactions: transactions,
}

err := ValidateBlock(newBlock, previousBlock)
if err != nil {
    log.Error("Invalid block:", err)
    return err
}

// 添加到区块链
engine.SetChainHead(newBlock)
```

---

## 9. 验证规则

### 9.1 贡献度验证

1. **在线时间范围**：`0 ≤ onlineTime ≤ OnlineWindow`
2. **丢包率范围**：`0.0 ≤ packetLoss ≤ MaxPacketLossRate`
3. **贡献度范围**：`0.0 ≤ score ≤ 1.0`
4. **贡献度更新**：只能增加，不能减少

### 9.2 验证器集验证

1. **验证器数量**：必须为 `ValidatorCount` 个（100）
2. **验证器排序**：必须按贡献度降序
3. **轮换索引**：必须在 `[0, ValidatorCount-1]` 范围内
4. **验证器去重**：不能有重复的验证器 ID

### 9.3 区块验证

1. **签名数量**：必须有足够的验证器签名（建议 ≥ 3）
2. **Merkle 根验证**：重新计算并比较
3. **区块间隔验证**：区块时间戳间隔必须在 `BlockInterval ± 2 秒` 范围内

### 9.4 前缀监控验证

1. **前缀长度**：必须是 /64 前缀（8 字节）
2. **设备数量限制**：`deviceCount ≤ MaxPrefixDevices`
3. **前缀格式**：必须是有效的 IPv6 前缀

---

## 10. 错误处理

### 10.1 共识错误

| 错误码 | 说明 | 处理方式 |
|---------|------|---------|
| ERR_CONSENSUS_INVALID_BLOCK | 区块验证失败 | 拒绝区块，不转发 |
| ERR_CONSENSUS_MISSING_TRANSACTION | 交易不存在 | 拒绝区块，请求缺失交易 |
| ERR_CONSENSUS_DUPLICATE_TRANSACTION | 重复交易 | 拒绝区块，检查交易池 |
| ERR_CONSENSUS_INVALID_SIGNATURE | 签名无效 | 拒绝区块，验证公钥 |
| ERR_CONSENSUS_STALE_BLOCK | 过期区块 | 拒绝区块，已存在于链中 |
| ERR_CONSENSUS_PARENT_NOT_FOUND | 前一个区块未找到 | 拒绝区块，请求父区块 |

### 10.2 恢复策略

1. **短暂网络分割**：等待 3-6 个区块，自动恢复
2. **深度重组**：记录警告，但允许重组
3. **异常区块时间**：同步网络时间，拒绝异常区块
4. **验证器失联**：从验证器集中移除，重新选举

---

## 11. 附录

### A. 贡献度计算详解

在线时间归一化：
```
NormalizedOnlineTime = min(
    totalOnlineTime / OnlineWindow,
    1.0
)
```

丢包率归一化：
```
NormalizedPacketLoss = 1.0 - packetLossRate
```

转发量归一化（假设网络平均转发量为 `AvgForwarded`）：
```
NormalizedForwarded = min(
    forwardedBytes / AvgForwarded,
    1.0
)
```

最终贡献度：
```
Score = NormalizedOnlineTime * 0.6 +
        NormalizedPacketLoss * 0.1 +
        NormalizedForwarded * 0.3
```

### B. 前缀反垄断数学推导

奖励衰减公式基于指数衰减：

```
f(n) = R0 * e^(-n/10)
```

其中：
- `f(n)` = 有 n 个设备时的奖励乘数
- `R0` = 1.0（无设备数限制时的基础乘数）
- `n` = 前缀内的设备数量
- `e = 自然对数底 ≈ 2.71828

计算示例：
- n=1:  f(1) = 1.0 * e^(-0.1) ≈ 0.905
- n=10: f(10) = 1.0 * e^(-1) ≈ 0.368
- n=30: f(30) = 1.0 * e^(-3) ≈ 0.050
- n=50: f(50) = 1.0 * e^(-5) ≈ 0.007

然后取最大值和最小值的中间值，确保至少 1% 的基础奖励：
```
RewardMultiplier = max(0.01, min(1.0, f(n)))
```

### C. Merkle 树算法

V6Coin 使用标准的 Merkle Patricia Tree（MPT）：

1. **节点类型**：
   - 扩展节点（Extension）：包含子节点哈希和值
   - 分支节点（Branch）：包含 16 个子节点哈希
   - 叶子节点（Leaf）：包含交易哈希

2. **树构建**：
   - 交易按哈希排序
   - 自底向上构建树
   - 计算每个节点的哈希

3. **Merkle 证明**：
   - 提供从叶子到根的路径
   - 包含兄弟节点哈希
   - 验证无需完整树

---

## 12. 版本历史

| 版本 | 日期 | 变更 |
|------|------|------|
| v1.0 | 2026-02-01 | 初始版本 |

---

## 13. 参考文档

- [RFC 3971](https://datatracker.ietf.org/doc/html/rfc3971) - IPv6 Cryptographically Generated Addresses
- [RFC 6980](https://datatracker.ietf.org/doc/html/rfc6980) - IPv6 Hop-by-Hop Options Header
- [V6Coin 白皮书](../whitepaper/V6Coin_Whitepaper_CN.md)
- [V6Coin 白皮书（英文）](../whitepaper/V6Coin_Whitepaper_EN.md)
