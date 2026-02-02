# Blockchain Data Structure Specification

**Version**: v1.0
**Date**: 2026-02-01

---

## 1. 概述

V6Coin 区块链数据结构设计采用 Merkle Patricia Tree（MPT）结合传统的链式结构，以实现高效的状态查询、快速的状态证明和轻节点支持。

### 设计目标

1. **高效查询**：通过 MPT 快速验证交易和状态
2. **快速同步**：轻节点可以快速同步区块头
3. **状态证明**：验证者可以快速证明状态根
4. **可扩展性**：支持未来的状态层扩展

---

## 2. 核心概念

### 2.1 区块（Block）

区块是 V6Coin 区块链的基本数据单元，包含区块头和交易列表：

```go
type Block struct {
    Header        *BlockHeader        // 区块头
    Transactions  []*Transaction      // 交易列表
    ValidatorSigs [][]byte       // 所有验证器签名
}
```

**核心特征**：
- 包含交易列表和所有验证器签名
- 通过 Merkle 树验证所有交易
- 支持分叉验证

### 2.2 区块头（Block Header）

```go
type BlockHeader struct {
    Version       uint16         // 协议版本（0x0100）
    Height        uint64          // 区块高度
    PrevBlockHash []byte       // 前一个区块的哈希
    MerkleRoot    []byte         // 交易 Merkle 树根
    Timestamp     uint64          // 区块时间戳（Unix）
    ValidatorID   p2p.PeerID      // 创建区块的验证器 ID
    StateRoot     []byte          // 状态树根
    Signature     []byte         // 验证器签名
}
```

**字段说明**：
- **Version**: 固定为 0x0100，便于将来升级
- **Height**: 区块高度，从创世区块开始递增
- **PrevBlockHash**: 前一个区块的哈希，形成链式结构
- **MerkleRoot**: 交易列表的 Merkle 树根
- **Timestamp**: 区块创建的 Unix 时间戳
- **ValidatorID**: 生成此区块的验证器 ID
- **StateRoot**: 当前全局状态的根哈希
- **Signature**: 验证器对区块头的签名

### 2.3 交易（Transaction）

V6Coin 交易的基本定义：

```go
type Transaction struct {
    Version   uint16    // 交易版本
    Type      TxType    // 交易类型
    From      net.IP     // 发送方地址（16 字节）
    To        net.IP     // 接收方地址（16 字节）
    Amount    uint64     // 金额（nano-V6）
    Fee       uint64     // 手续费（nano-V6）
    Nonce     uint64     // 随机数
    Timestamp uint64     // 交易时间戳
    Signature []byte     // Ed25519 签名（32 字节）
    Data      []byte     // 可选交易数据
}
```

**交易类型**：
- `0x01`：在线交易（实时嵌入 IPv6 报头）
- `0x02`：离线交易（本地签名，延迟结算）
- `0x03`：资产迁移
- `0x04`：质押
- `0x05`：解除质押
- `0x06`：治理提案
- `0x07`：投票

---

## 3. 数据结构

### 3.1 Merkle Patricia Tree（MPT）

MPT 是一种特殊的数据结构，提供：

1. **高效路径查询**：O(log n) 时间复杂度
2. **状态证明**：提供验证者快速证明
3. **空间效率**：共享相同前缀的子节点
4. **可扩展性**：支持未来的状态层扩展

#### MPT 节点类型

1. **Extension（扩展节点）**：
   - **类型**：`Extension`
   - **数据**：子节点哈希和值
   - **长度**：2 字节哈希 + 2 字节数据

2. **Branch（分支节点）**：
   - **类型**：`Branch`
   - **数据**：16 字节分支哈希
   - **长度**：16 字节哈希

3. **Leaf（叶子节点）**：
   - **类型**：`Leaf`
   - **数据**：交易哈希或值
   - **长度**：32 字节哈希或数据

#### MPT 结构

```
                           [Root Hash]
                              /         \
                           [Branch Hash]     [Leaf Hash]  [Extension]
                              /           \       /         \         /           \
                              \         [Leaf Hash] [Value]
                              \       /         \       /         /           \
                              \        [Leaf Hash][Value]
```

### 3.2 状态根（State Root）

状态根是整个区块链的全局状态快照：

1. **计算方式**：递归所有账户余额
2. **存储方式**：MPT 的叶子节点存储余额
3. **验证方式**：重新计算验证状态

**状态包含**：
- 所有账户的 V6 余额
- 智能合约状态（如果存在）
- 网络配置

---

## 4. 区块链操作

### 4.1 区块验证

验证器在打包新区块时执行以下检查：

1. **版本验证**：
   - `block.Header.Version == 0x0100`

2. **区块头验证**：
   - 前一个区块存在
   - 前一个区块的高度正确递增
   - 前一个区块的哈希匹配
   - `block.Header.Timestamp > prevBlock.Timestamp`

3. **交易验证**：
   - 所有交易签名有效
   - 所有交易余额充足
   - 所有交易 nonce 唯一
   - 没有重复交易

4. **Merkle 根验证**：
   - 重新计算交易 Merkle 根
   - 与区块头中的 MerkleRoot 匹配

5. **签名验证**：
   - 验证器签名数量足够（建议 ≥ 3）
   - 所有签名对应区块高度
   - 验证器 ID 在选举列表中

6. **时间戳验证**：
   - 区块时间戳不能超前于当前时间 + 2 分钟
   - 每个区块的时间戳递增

### 4.2 区块添加

```go
func (bc *BlockChain) AddBlock(block *consensus.Block) error {
    bc.chainMutex.Lock()
    defer bc.chainMutex.Unlock()
    
    // Validate block
    if err := bc.validator.ValidateBlock(block, bc.GetChainHead()); err != nil {
        return err
    }
    
    // Check for duplicate
    if _, exists := bc.blocks[string(block.Hash())]; exists {
        return ErrDuplicateBlock
    }
    
    // Add to blockchain
    blockHash := string(block.Hash())
    bc.blocks[blockHash] = block
    bc.heightMap[0] = block.Header.Height
    
    // Update genesis block if needed
    if bc.genesisBlock == nil {
        bc.genesisBlock = block
    }
    
    // Update chain head
    bc.chainHead = block
    bc.currentHeight = block.Header.Height
    
    return nil
}
```

### 4.3 区块查询

```go
// GetBlockByHeight 通过高度查询区块
func (bc *BlockChain) GetBlockByHeight(height uint64) (*consensus.Block, error) {
    bc.chainMutex.RLock()
    defer bc.chainMutex.RUnlock()
    
    blockHash, ok := bc.heightMap[height]
    if !ok {
        return nil, ErrBlockNotFound
    }
    
    return bc.blocks[blockHash], nil
}

// GetBlockByHash 通过哈希查询区块
func (bc *BlockChain) GetBlockByHash(hash string) (*consensus.Block, error) {
    bc.chainMutex.RRLock()
    defer bc.chainMutex.RUnlock()
    
    block, ok := bc.blocks[hash]
    if !ok {
        return nil, ErrBlockNotFound
    }
    
    return block, nil
}
```

### 4.4 链验证

```go
// ValidateChain 验证整条链的完整性
func (bc *BlockChain) ValidateChain() error {
    bc.chainMutex.RLock()
    defer bc.chainMutex.RUnlock()
    
    // 检查链的连续性
    prevHeight := uint64(0)
    for height := uint64(1); height <= bc.currentHeight; height++ {
        blockHash, ok := bc.heightMap[height]
        if !ok {
            return ErrChainBroken
        }
        
        // 验证高度连续性
        if height != prevHeight+1 {
            return ErrChainBroken
        }
        
        // 验证哈希链
        block, err := bc.GetBlockByHeight(height)
        if err != nil {
            return err
        }
        
        if !equalBytes(block.Header.PrevBlockHash, prevBlock) {
            return ErrChainBroken
        }
        
        prevHeight = height
    }
    
    return nil
}
```

---

## 5. 分叉处理

### 5.1 分叉检测

当接收到多个新区块时，可能产生链分叉：

1. **分叉类型**：
   - **软分叉（Soft Fork）：较短的链最终被放弃
   - **硬分叉（Hard Fork）：两条链都继续，通过最长链选择机制解决

2. **最长链选择**：
   - 选择累计工作量最多的链
   - 如果工作量相同，选择时间戳较新的链
   - 所有验证器最终在新链上达成一致

3. **分叉解决**：
   - 验证器在新链上重新签名
   - 更新选举轮换索引
   - 通知旧链验证器切换到新链

### 5.2 分叉恢复

如果检测到更长的链：

1. **区块收集**：等待新链追赶上
2. **验证新区块**：逐个验证新区块
3. **切换验证器**：一旦新链超过旧链工作量 10%，验证器自动切换

---

## 6. 错误码定义

| 错误码 | 说明 | 处理方式 |
|---------|------|---------|
| ERR_BLOCK_NOT_FOUND | 区块未找到 | 拒绝请求 |
| ERR_INVALID_CHAIN | 无效的链 | 拒绝请求，重试同步 |
| ERR_BLOCK_NOT_FOUND | 前一个区块未找到 | 拒绝请求，重试同步 |
| ERR_CHAIN_BROKEN | 链不连续 | 拒绝交易，请求重新同步 |
| ERR_DUPLICATE_BLOCK | 重复区块 | 拒绝区块 |
| ERR_INVALID_BLOCK | 无效区块 | 拒绝区块 |
| ERR_INVALID_PARENT | 前一个区块无效 | 拒绝区块 |
| ERR_STATE_ROOT_MISMATCH | 状态根不匹配 | 拒绝区块 |
| ERR_INVALID_SIGNATURE | 签名无效 | 拒绝区块 |
| ERR_INVALID_HEIGHT | 区块高度无效 | 拒绝区块 |
| ERR_INVALID_TIMESTAMP | 时间戳无效 | 拒绝区块 |
| ERR_INVALID_VERSION | 版本无效 | 拒绝区块 |
| ERR_FORK_DETECTED | 检测到分叉 | 暂时回 |

---

## 7. 性能优化

### 7.1 缓存策略

1. **区块缓存**：最近 1000 个区块缓存在内存
2. **状态缓存**：热点账户的余额状态
3. **交易索引**：快速查询交易

### 7.2 并发处理

1. **读写锁**：
   - 区块添加：`chainMutex.Lock()`
   - 区块查询：`chainMutex.RLock()`
   - 高并发读取

2. **MPT 查询**：
   -  O(log n) 时间复杂度的路径遍历
   - 缓存热点路径

### 7.3 存储优化

1. **区块序列化**：高效序列化区块到磁盘
2. **状态快照**：定期保存状态快照到磁盘
3. **快照恢复**：支持从快照快速恢复

---

## 8. 数据持久化

### 8.1 区块存储

**格式**：每区块一个文件，包含序列化数据和签名

**文件命名**：
```
blocks/{height}/{block_hash}.dat
```

**内容**：
- 区块头（二进制序列化）
- 交易列表（数组序列化）
- 验证器签名（JSON）

### 8.2 状态存储

**状态数据库**：LevelDB / BadgerDB

1. **账户状态表**：
   - Key: IPv6 地址
   - Value: 余额（nano-V6）
   - Version: 状态版本

2. **存储位置**：`state/` 目录

### 8.3 归档策略

1. **归档间隔**：每 100,000 个区块
2. **保留策略**：保留最近 1,000 个区块
3. **清理策略**：删除超过 100,000 个旧区块

---

## 9. 安全机制

### 9.1 区块签名攻击防护

1. **签名要求**：
   - 至少 3 个验证器签名
   - 验证器必须在选举列表中
   - 签名与区块高度匹配

2. **签名验证**：
   - 验证公钥来自选举集
   - 检查签名算法（Ed25519）

3. **重放攻击防护**：
   - 基于区块高度的 nonce 机制
   - 区块包含交易已使用的 Nonce

### 9.2 51% 攻击防护

1. **验证器权重**：
   - PoC 贡献度（60%）+ 持币量（40%）
   - 劳动节点（长期在线）有更大权重

2. **前缀反垄断**：
   - 设备数量越多，单个验证器权重越低
   - 最小权重 10%

### 9.3 恶意区块检测

1. **时间戳异常**：
   - 区块时间戳检查
   - 拒绝异常快或慢的区块
2. **工作量异常**：突然算力激增的区块

---

## 10. 代码示例

### 10.1 创建区块链

```go
import (
    "github.com/jinba225/v6coin-protocol/pkg/blockchain"
    "github.com/jinba225/v6coin-protocol/pkg/consensus"
)

// 创建新的区块链
config := &BlockChainConfig{
    GenesisBlock: genesisBlock,
    StateDB:      stateDB,
    Validator:   rewardDistributor,
}

chain := NewBlockChain(config)

// 添加创世区块
genesisBlock := &consensus.Block{
    Header: &BlockHeader{
        Version:       0x0100,
        Height:        0,
        PrevBlockHash: make([]byte, 32),
        MerkleRoot:    CalculateMerkleRoot([]),
        Timestamp:     uint64(time.Now().Unix()),
        ValidatorID:   genesisValidatorID,
        StateRoot:     stateRoot,
        Signature:     genesisSignature,
    },
    Transactions: []*consensus.Transaction{},
    ValidatorSigs: [][]byte{genesisSignature},
}

err := chain.AddBlock(genesisBlock)
if err != nil {
    log.Fatal("Failed to add genesis block:", err)
}
```

### 10.2 查询区块

```go
// 通过高度查询区块
block, err := chain.GetBlockByHeight(100)
if err != nil {
    log.Error("Failed to get block:", err)
    return
}

fmt.Printf("Block Height: %d\n", block.Header.Height)
fmt.Printf("Prev Hash: %s\n", block.Header.PrevBlockHash)
fmt.Printf("Timestamp: %d\n", block.Header.Timestamp)
```

### 10.3 查询交易

```go
// 查询区块中的交易
for _, tx := range block.Transactions {
    fmt.Printf("Hash: %s\n", tx.HashString())
    fmt.Printf("From: %s\n", tx.From.String())
    fmt.Printf("To: %s\n", tx.To.String())
    fmt.Printf("Amount: %d V6\n", tx.Amount/1e9)
}
```

---

## 11. 附录

### A. Merkle Tree 算法详解

**叶子节点哈希计算**：
```go
// 计算叶子节点哈希
func CalculateLeafHash(hash []byte, data []byte) []byte {
    combined := append(hash, data...)
    return sha256.Sum256(combined)
}
```

**根节点哈希计算**：
```go
// 计算根节点哈希
func CalculateRootHash(childHashes [][]byte, leafHashes [][]byte) []byte []byte {
    if len(childHashes) != len(leafHashes) {
        return nil
    }
    
    // 迭代计算哈希，直到剩两个节点
    for i := 0; i < len(leafHashes); i++ {
        newHashes := make([][]byte, len(leafHashes))
        
        for j := 0; j < len(childHashes); j++ {
            combined := append(childHashes[i], leafHashes[j]...)
            hash := sha256.Sum256(combined)
            newHashes[i] = hash[:]
        }
        
        leafHashes = newHashes
    }
    
    // 最终根哈希（两个根节点哈希的较小值）
    rootHash := sha256.Sum256(append(childHashes[0], childHashes[1]))
    return rootHash
}
```

### B. 数据大小计算

| 数据类型 | 大小（近似） |
|---------|-------------|
| 区块头 | 40 字节 |
| 交易（不含数据） | 83 字节 |
| 交易数据 | 可变长度 |
| 区块签名 | 96 字节（3 个验证器 × 32 字节） |

### C. 历史增长

假设数据：
- 区块间隔：10 秒
- 每区块交易：平均 100 笔
- 存储效率：每字节 100 KB/s

**1 年数据量**：
- 区块数：31,536,000
- 交易数：3,153,600,000
- 总大小：约 3.2 TB

---

## 12. 版本历史

| 版本 | 日期 | 更新 |
|---------|------|--------|
| v1.0 | 2026-02-01 | 初始版本 |

---

## 13. 参考文档

- [V6Coin Whitepaper (Chinese)](../whitepaper/V6Coin_Whitepaper_CN.md)
- [V6Coin Whitepaper (English)](../whitepaper/V6Coin_Whitepaper_EN.md)
- [IPv6 Extension Header Specification](./01-ipv6-header-spec.md)
- [CGA Address Mapping Specification](./02-cga-address-spec.md)
- [PoC Consensus Specification](./03-poc-consensus-spec.md)
- [Transaction Validation Specification](./04-transaction-spec.md)
- [Cryptographic Algorithm Specification](./05-crypto-spec.md)
- [P2P Network Protocol Specification](./06-p2p-protocol-spec.md)
- [Blockchain Data Structure Specification](./07-blockchain-spec.md)
