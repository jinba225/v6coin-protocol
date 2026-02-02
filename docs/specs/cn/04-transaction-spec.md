# 交易验证规范

**版本**: v1.0
**日期**: 2026-02-01

---

## 1. 概述

V6Coin 交易验证机制确保只有有效、合法的交易才能被纳入区块链。验证过程包括多重检查：签名验证、余额验证、时间戳验证、交易结构验证等。

### 设计目标

1. **安全性**：防止双重支付、重放攻击、无效签名
2. **完整性**：确保交易数据完整且格式正确
3. **公平性**：防止垃圾交易和拒绝服务攻击
4. **性能**：快速验证，低延迟交易确认

---

## 2. 核心概念

### 2.1 交易类型（TxType）

V6Coin 支持多种交易类型：

| 交易类型 | 值 | 说明 |
|---------|------|------|
| 在线交易 | 0x01 | 实时嵌入 IPv6 扩展报头 |
| 离线交易 | 0x02 | 延迟结算，本地签名缓存 |
| 资产迁移 | 0x03 | 跨地址资产迁移 |
| 质押交易 | 0x04 | 锁定代币获得投票权 |
| 解除质押 | 0x05 | 解锁代币 |
| 治理提案 | 0x06 | 提交治理提案 |
| 投票交易 | 0x07 | 对提案进行投票 |

### 2.2 交易池（Transaction Pool）

交易池管理待确认的交易：

1. **最大容量**：10,000 笔交易
2. **优先级机制**：
   - 基于手续费和 Gas 限制
   - 高优先级交易优先被打包
3. **去重机制**：
   - 相同哈希的交易只保留一份
   - 基于 `tx.Hash()` 去重

---

## 3. 数据结构定义

### 3.1 交易（Transaction）

```go
type Transaction struct {
    Version   uint16         // 交易版本
    Type      TxType          // 交易类型
    From      net.IP          // 发送方地址（16 字节）
    To        net.IP          // 接收方地址（16 字节）
    Amount    uint64          // 金额（nano-V6）
    Fee       uint64          // 交易手续费（nano-V6）
    Nonce     uint64          // 随机数
    Timestamp uint64          // 交易时间戳（Unix）
    Signature []byte          // Ed25519 签名（32 字节）
    Data      []byte          // 可选交易数据
}
```

**字段说明**：
- **Version**: 当前版本为 0x0100
- **From/To**: 完整 IPv6 地址（16 字节）
- **Amount**: 以 nano-V6 为单位（1 V6 = 10^9 nano-V6）
- **Fee**: 最小手续费为每字节 1 nano-V6
- **Nonce**: 用于防止重放攻击
- **Signature**: Ed25519 签名，32 字节

### 3.2 交易哈希

交易哈希计算：
```
hash = SHA256(
    Version (2 bytes) +
    Type (1 byte) +
    From (16 bytes) +
    To (16 bytes) +
    Amount (8 bytes) +
    Fee (8 bytes) +
    Nonce (8 bytes) +
    Timestamp (8 bytes) +
    Data (variable)
)
```

---

## 4. 交易验证规则

### 4.1 交易格式验证

1. **版本检查**：
   ```
   tx.Version == 0x0100
   ```

2. **交易类型检查**：
   ```
   0x01 ≤ tx.Type ≤ 0x07
   ```

3. **地址格式验证**：
   ```
   - 必须是有效的 IPv6 地址（16 字节）
   - IID 的第 0 位必须为 0（单播地址）
   - 长度必须为 16 字节
   ```

4. **金额验证**：
   ```
   tx.Amount > 0
   ```

5. **手续费验证**：
   ```
   tx.Fee >= MinFeePerByte * tx.Size
   where MinFeePerByte = 1 nano-V6
   ```

6. **时间戳验证**：
   ```
   - 在线交易：|timestamp - current| ≤ 5 分钟
   - 离线交易：|timestamp - current| ≤ MaxTTL（7 天）
   ```

### 4.2 签名验证

1. **签名长度检查**：
   ```
   len(tx.Signature) == 32  // Ed25519 签名固定 32 字节
   ```

2. **签名验证**：
   ```
   Verify(publicKey, txData, tx.Signature)
   where:
     publicKey: 发送方公钥
     txData: SHA256(Version + Type + From + To + Amount + Fee + Nonce + Timestamp + Data)
     Signature: Ed25519 签名
   ```

3. **签名算法**：
   - 使用 Ed25519 椭圆曲线签名
   - 签名验证失败则拒绝交易

### 4.3 余额验证

1. **发送方余额验证**：
   ```
   balance = stateDB.GetBalance(tx.From)
   totalCost = tx.Amount + tx.Fee
   
   if balance < totalCost:
     return ErrInsufficientBalance
   ```

2. **账户存在检查**：
   ```
   exists = stateDB.HasAccount(tx.From)
   
   if !exists:
     return ErrUnknownSender
   ```

3. **Nonce 验证**：
   ```
   currentNonce = stateDB.GetNonce(tx.From)
   
   if tx.Nonce <= currentNonce:
     return ErrInvalidNonce  // 防止重放攻击
   ```

### 4.4 重复交易验证

1. **交易池去重**：
   ```
   if txPool.Contains(txHash):
     return ErrDuplicateTx
   ```

2. **链上交易检查**：
   ```
   if blockchain.ContainsTransaction(txHash):
     // 对于离线交易，可能允许重用已打包交易
     // 对于在线交易，应拒绝
     return ErrTxAlreadyConfirmed
   ```

---

## 5. 交易池管理

### 5.1 添加交易到池

1. **验证交易**：
   ```go
   func (tp *TxPool) AddTransaction(tx *Transaction) error {
       if err := tp.validateTx(tx); err != nil {
           return err
       }
   ```
2. **计算优先级**：
   ```go
   func (tp *TxPool) calculatePriority(tx *Transaction) float64 {
       // 基础优先级：0.6
       // Gas 限制：根据交易大小调整
       base := 0.6
       
       gasUsed := uint64(len(tx.Data)) / 1000
       gasFactor := 1.0 - math.Min(1.0, float64(gasUsed)/float64(MaxGasLimit))
       
       return base * gasFactor
   }
   ```
3. **入队**：
   ```go
   // 如果池已满，尝试驱逐最低优先级交易
   if tp.pending.Len() >= MaxPoolSize {
       tp.evictLowestPriority()
   }
   
   heap.Push(&tp.pending, newItem)
   ```

### 5.2 交易池容量

1. **最大交易数**：10,000 笔
2. **最大 TTL**：7 天（离线交易）
3. **Gas 限制**：每字节 1,000 gas

### 5.3 驱逐策略

当交易池满时，按以下优先级驱逐交易：

1. **最低优先级**
2. **最长时间等待**
3. **最低 Gas 价格**

---

## 6. 错误码定义

| 错误码 | 错误类型 | 说明 |
|---------|---------|------|
| ERR_INVALID_TX_VERSION | 交易版本无效 | 版本不是 0x0100 |
| ERR_INVALID_TX_TYPE | 交易类型无效 | 不支持的交易类型 |
| ERR_INVALID_ADDRESS | 地址格式无效 | 无效的 IPv6 地址 |
| ERR_INVALID_AMOUNT | 金额无效 | 金额 ≤ 0 |
| ERR_INVALID_FEE | 手续费无效 | 手续费不足 |
| ERR_INVALID_NONCE | Nonce 无效 | Nonce 已使用 |
| ERR_INSUFFICIENT_BALANCE | 余额不足 | 发送方余额不足 |
| ERR_UNKNOWN_SENDER | 未知发送方 | 发送方账户不存在 |
| ERR_INVALID_SIGNATURE | 签名无效 | Ed25519 签名验证失败 |
| ERR_DUPLICATE_TX | 重复交易 | 交易已存在于池或链上 |
| ERR_TX_TOO_OLD | 交易过旧 | 离线交易超时 |
| ERR_POOL_FULL | 交易池已满 | 无法添加更多交易 |
| ERR_TX_EXPIRED | 交易已过期 | 交易有效期已过 |
| ERR_INVALID_DATA | 交易数据无效 | Data 字段格式错误 |

---

## 7. 交易生命周期

### 7.1 待确认阶段

交易进入交易池后的状态：

1. **Pending**: 交易已添加到池，等待被打包
2. **Invalid**: 交易验证失败，已拒绝
3. **Rejected**: 交易池已满，被驱逐

### 7.2 确认阶段

1. **Mempool**: 交易已被打包到区块，等待 3 个区块确认
2. **Confirmed**: 交易已确认，余额已扣除
3. **Orphan**: 孤块（已被后续区块覆盖）

### 7.3 过期处理

离线交易和未确认的在线交易会过期：

```go
func (tp *TxPool) CleanExpired() int {
    now := time.Now()
    toRemove := make([]string, 0)
    
    for hash, item := range tp.all {
        if now.Sub(item.AddedAt) > MaxTTL {
            toRemove = append(toRemove, hash)
        }
    }
    
    for _, hash := range toRemove {
        tp.RemoveTransaction(hash)
    }
    
    return len(toRemove)
}
```

---

## 8. Gas 和费用模型

### 8.1 Gas 计算

1. **基础 Gas**：
   - 交易数据：每 1,000 字节 = 1 gas
   - 基础交易：50 gas
   - 智能合约调用：100 gas（暂未使用）

2. **Gas 限制**：
   - 区块最大 Gas：1,000,000
   - 每笔交易最大 Gas：50,000

### 8.2 手续费计算

```
BaseFee = MinFeePerByte * tx.Size
PriorityFee = BaseFee * PriorityMultiplier

FinalFee = BaseFee + PriorityFee
```

### 8.3 费用分配

| 角色 | 费用分配 |
|------|-----------|
| 验证节点 | 70% |
| 网络维护 | 15% |
| 基金金库 | 15% |

---

## 9. 代码示例

### 9.1 创建交易池

```go
import (
    "github.com/jinba225/v6coin-protocol/pkg/tx"
)

// 创建交易池
pool := tx.NewTxPool(stateDB)

// 添加交易
tx := &consensus.Transaction{
    Type:      consensus.TxTypeOnline,
    From:      senderAddr,
    To:        receiverAddr,
    Amount:    1000 * 1e9, // 1000 V6
    Fee:       100 * 1e9,     // 100 V6 手续费
    Nonce:     uint64(time.Now().Unix()),
    Timestamp: uint64(time.Now().Unix()),
}

err := pool.AddTransaction(tx)
if err != nil {
    log.Error("Failed to add transaction:", err)
}
```

### 9.2 验证交易

```go
import (
    "github.com/jinba225/v6coin-protocol/pkg/crypto"
)

// 验证交易签名
func VerifyTransaction(tx *consensus.Transaction, publicKey []byte) error {
    // 计算交易哈希
    txData := tx.Hash()
    
    // 验证 Ed25519 签名
    if !crypto.Verify(publicKey, txData, tx.Signature) {
        return errors.New("invalid signature")
    }
    
    return nil
}
```

### 9.3 查询待确认交易

```go
// 获取前 10 笔待确认交易（按优先级排序）
pendingTxs := pool.GetPendingTransactions(10)

for _, tx := range pendingTxs {
    fmt.Printf("Hash: %s\n", tx.HashString())
    fmt.Printf("From: %s\n", tx.From.String())
    fmt.Printf("To: %s\n", tx.To.String())
    fmt.Printf("Amount: %d V6\n", tx.Amount/1e9)
}
```

### 9.4 清理过期交易

```go
// 清理过期的离线交易
removed := pool.CleanExpired()

if removed > 0 {
    log.Printf("Removed %d expired transactions", removed)
}
```

---

## 10. 安全考虑

### 10.1 防重放攻击

1. **Nonce 机制**：
   - 每个账户维护递增的 Nonce
   - 只接受比当前 Nonce 大的交易
   - 防止重复使用相同交易哈希

2. **时间窗口**：
   - 限制旧交易的接受时间
   - 拒绝超过 7 天的离线交易

### 10.2 双重支付防护

1. **交易池去重**：
   - 基于 `tx.Hash()` 全局去重
   - 相同哈希的交易只保留最新一笔

2. **链上交易检查**：
   - 已确认的交易不应重复打包
   - 在线交易被确认后应从池中移除

### 10.3 DoS 防护

1. **交易池大小限制**：
   - 最大 10,000 笔待确认交易
   - 防止内存耗尽和 DDoS 攻击

2. **Gas 限制**：
   - 单笔交易最大 50,000 gas
   - 防止无限 Gas 攻击

3. **费用机制**：
   - 所有交易必须支付手续费
   - 高优先级交易支付更高费用

---

## 11. 性能优化

### 11.1 交易池优化

1. **堆排序**：
   - 使用优先队列管理交易
   - O(log n) 复杂度插入和提取

2. **高效哈希计算**：
   - 预计算和缓存交易哈希
   - 避免重复计算

3. **批量验证**：
   - 批量验证交易签名
   - 减少 RSA 操作次数

### 11.2 存储优化

1. **状态数据库查询**：
   - 使用高效的查询索引
   - 批量获取账户余额和 Nonce

2. **缓存策略**：
   - 缓存热点账户数据
   - 减少 I/O 操作

---

## 12. 版本历史

| 版本 | 日期 | 更新 |
|------|------|------|
| v1.0 | 2026-02-01 | 初始版本 |

---

## 13. 参考文档

- [V6Coin 白皮书（中文）](../whitepaper/V6Coin_Whitepaper_CN.md)
- [V6Coin Whitepaper（英文）](../whitepaper/V6Coin_Whitepaper_EN.md)
- [IPv6 扩展报头规范](./01-ipv6-header-spec.md)
- [CGA 地址映射规范](./02-cga-address-spec.md)
- [PoC 共识规范](./03-poc-consensus-spec.md)
- [加密算法规范](./05-crypto-spec.md)
- [P2P 网络协议规范](./06-p2p-protocol-spec.md)
- [区块链数据结构规范](./07-blockchain-spec.md)
