# Blockchain Data Structure Specification

**Version**: v1.0
**Date**: 2026-02-01

---

## 1. Overview

The V6Coin blockchain data structure design employs Merkle Patricia Tree (MPT) combined with traditional chain structure to achieve efficient state queries, fast state proofs, and light node support.

### Design Goals

1. **Efficient Query**: Fast verification of transactions and states through MPT
2. **Fast Synchronization**: Light nodes can quickly synchronize block headers
3. **State Proofs**: Validators can quickly prove state roots
4. **Scalability**: Supports future state layer expansion

---

## 2. Core Concepts

### 2.1 Block

A block is the basic data unit of the V6Coin blockchain, containing block header and transaction list:

```go
type Block struct {
    Header        *BlockHeader        // Block header
    Transactions  []*Transaction      // Transaction list
    ValidatorSigs [][]byte       // Signatures from all validator nodes
}
```

**Core Features**:
- Contains transaction list and all validator signatures
- Verifies all transactions through Merkle tree
- Supports fork validation

### 2.2 Block Header

```go
type BlockHeader struct {
    Version       uint16         // Protocol version (0x0100)
    Height        uint64          // Block height
    PrevBlockHash []byte       // Hash of previous block
    MerkleRoot    []byte         // Merkle tree root of transactions
    Timestamp     uint64          // Block timestamp (Unix)
    ValidatorID   p2p.PeerID      // Validator ID who created this block
    StateRoot     []byte          // Root of state trie
    Signature     []byte         // Validator's signature
}
```

**Field Descriptions**:
- **Version**: Fixed at 0x0100 for future upgrades
- **Height**: Block height, incrementing from genesis block
- **PrevBlockHash**: Hash of previous block, forming chain structure
- **MerkleRoot**: Merkle tree root of transaction list
- **Timestamp**: Unix timestamp of block creation
- **ValidatorID**: Validator ID that generated this block
- **StateRoot**: Root hash of current global state
- **Signature**: Validator's signature of block header

### 2.3 Transaction

Basic definition of V6Coin transaction:

```go
type Transaction struct {
    Version   uint16    // Transaction version
    Type      TxType    // Transaction type
    From      net.IP     // Sender address (16 bytes)
    To        net.IP     // Receiver address (16 bytes)
    Amount    uint64     // Amount (nano-V6)
    Fee       uint64     // Transaction fee (nano-V6)
    Nonce     uint64     // Random number
    Timestamp uint64     // Transaction timestamp
    Signature []byte     // Ed25519 signature (32 bytes)
    Data      []byte     // Optional transaction data
}
```

**Transaction Types**:
- `0x01`: Online transaction (real-time embedded in IPv6 header)
- `0x02`: Offline transaction (local signature, delayed settlement)
- `0x03`: Asset migration
- `0x04`: Staking
- `0x05`: Unstaking
- `0x06`: Governance proposal
- `0x07`: Voting

---

## 3. Data Structures

### 3.1 Merkle Patricia Tree (MPT)

MPT is a special data structure that provides:

1. **Efficient Path Query**: O(log n) time complexity
2. **State Proofs**: Provides fast verification for validators
3. **Space Efficiency**: Child nodes sharing the same prefix
4. **Scalability**: Supports future state layer expansion

#### MPT Node Types

1. **Extension Node**:
   - **Type**: `Extension`
   - **Data**: Child node hash and value
   - **Length**: 2 bytes hash + 2 bytes data

2. **Branch Node**:
   - **Type**: `Branch`
   - **Data**: 16 bytes branch hash
   - **Length**: 16 bytes hash

3. **Leaf Node**:
   - **Type**: `Leaf`
   - **Data**: Transaction hash or value
   - **Length**: 32 bytes hash or data

#### MPT Structure

```
                            [Root Hash]
                               /         \
                            [Branch Hash]     [Leaf Hash]  [Extension]
                               /           \       /         \         /           \
                               \         [Leaf Hash] [Value]
                               \       /         \       /         /           \
                               \        [Leaf Hash][Value]
```

### 3.2 State Root

State root is the global state snapshot of the entire blockchain:

1. **Calculation**: Recursively all account balances
2. **Storage**: MPT leaf nodes store balances
3. **Verification**: Recalculate to verify state

**State Includes**:
- All account V6 balances
- Smart contract states (if exist)
- Network configuration

---

## 4. Blockchain Operations

### 4.1 Block Validation

Validators perform the following checks when packaging new blocks:

1. **Version Validation**:
   - `block.Header.Version == 0x0100`

2. **Block Header Validation**:
   - Previous block exists
   - Previous block height increments correctly
   - Previous block hash matches
   - `block.Header.Timestamp > prevBlock.Timestamp`

3. **Transaction Validation**:
   - All transaction signatures valid
   - All transaction balances sufficient
   - All transaction nonces unique
   - No duplicate transactions

4. **Merkle Root Validation**:
   - Recalculate transaction Merkle root
   - Match with MerkleRoot in block header

5. **Signature Validation**:
   - Validator signature count sufficient (recommended ≥ 3)
   - All signatures correspond to block height
   - Validator ID in election list

6. **Timestamp Validation**:
   - Block timestamp cannot exceed current time + 2 minutes
   - Block timestamp increments for each block

### 4.2 Block Addition

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

### 4.3 Block Query

```go
// GetBlockByHeight queries block by height
func (bc *BlockChain) GetBlockByHeight(height uint64) (*consensus.Block, error) {
    bc.chainMutex.RLock()
    defer bc.chainMutex.RUnlock()

    blockHash, ok := bc.heightMap[height]
    if !ok {
        return nil, ErrBlockNotFound
    }

    return bc.blocks[blockHash], nil
}

// GetBlockByHash queries block by hash
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

### 4.4 Chain Validation

```go
// ValidateChain validates the integrity of the entire chain
func (bc *BlockChain) ValidateChain() error {
    bc.chainMutex.RLock()
    defer bc.chainMutex.RUnlock()

    // Check chain continuity
    prevHeight := uint64(0)
    for height := uint64(1); height <= bc.currentHeight; height++ {
        blockHash, ok := bc.heightMap[height]
        if !ok {
            return ErrChainBroken
        }

        // Validate height continuity
        if height != prevHeight+1 {
            return ErrChainBroken
        }

        // Validate hash chain
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

## 5. Fork Handling

### 5.1 Fork Detection

When multiple new blocks are received, chain forking may occur:

1. **Fork Types**:
   - **Soft Fork**: Shorter chain is eventually abandoned
   - **Hard Fork**: Both chains continue, resolved through longest chain selection mechanism

2. **Longest Chain Selection**:
   - Select chain with most cumulative work
   - If work equal, select chain with newer timestamp
   - All validators eventually converge on new chain

3. **Fork Resolution**:
   - Validators re-sign on new chain
   - Update election rotation index
   - Notify old chain validators to switch to new chain

### 5.2 Fork Recovery

If a longer chain is detected:

1. **Block Collection**: Wait for new chain to catch up
2. **Validate New Blocks**: Validate new blocks one by one
3. **Switch Validators**: Once new chain exceeds old chain work by 10%, validators automatically switch

---

## 6. Error Code Definitions

| Error Code | Description | Handling |
|---------|------|---------|
| ERR_BLOCK_NOT_FOUND | Block not found | Reject request |
| ERR_INVALID_CHAIN | Invalid chain | Reject request, retry synchronization |
| ERR_BLOCK_NOT_FOUND | Previous block not found | Reject request, retry synchronization |
| ERR_CHAIN_BROKEN | Chain discontinuous | Reject transaction, request resynchronization |
| ERR_DUPLICATE_BLOCK | Duplicate block | Reject block |
| ERR_INVALID_BLOCK | Invalid block | Reject block |
| ERR_INVALID_PARENT | Previous block invalid | Reject block |
| ERR_STATE_ROOT_MISMATCH | State root mismatch | Reject block |
| ERR_INVALID_SIGNATURE | Invalid signature | Reject block |
| ERR_INVALID_HEIGHT | Invalid block height | Reject block |
| ERR_INVALID_TIMESTAMP | Invalid timestamp | Reject block |
| ERR_INVALID_VERSION | Invalid version | Reject block |
| ERR_FORK_DETECTED | Fork detected | Temporarily roll back |

---

## 7. Performance Optimization

### 7.1 Caching Strategy

1. **Block Cache**: Recent 1000 blocks cached in memory
2. **State Cache**: Hot account balance states
3. **Transaction Index**: Fast transaction query

### 7.2 Concurrent Processing

1. **Read-Write Lock**:
   - Block addition: `chainMutex.Lock()`
   - Block query: `chainMutex.RLock()`
   - High concurrency reads

2. **MPT Query**:
   - Path traversal with O(log n) time complexity
   - Cache hot paths

### 7.3 Storage Optimization

1. **Block Serialization**: Efficiently serialize blocks to disk
2. **State Snapshot**: Periodically save state snapshots to disk
3. **Snapshot Recovery**: Support fast recovery from snapshot

---

## 8. Data Persistence

### 8.1 Block Storage

**Format**: One file per block, containing serialized data and signatures

**File Naming**:
```
blocks/{height}/{block_hash}.dat
```

**Contents**:
- Block header (binary serialization)
- Transaction list (array serialization)
- Validator signatures (JSON)

### 8.2 State Storage

**State Database**: LevelDB / BadgerDB

1. **Account State Table**:
   - Key: IPv6 address
   - Value: Balance (nano-V6)
   - Version: State version

2. **Storage Location**: `state/` directory

### 8.3 Archiving Strategy

1. **Archive Interval**: Every 100,000 blocks
2. **Retention Policy**: Keep recent 1,000 blocks
3. **Cleanup Strategy**: Delete old blocks exceeding 100,000

---

## 9. Security Mechanisms

### 9.1 Block Signature Attack Protection

1. **Signature Requirements**:
   - At least 3 validator signatures
   - Validators must be in election list
   - Signatures match block height

2. **Signature Verification**:
   - Verify public key from election set
   - Check signature algorithm (Ed25519)

3. **Replay Attack Protection**:
   - Nonce mechanism based on block height
   - Block includes nonces used by transactions

### 9.2 51% Attack Protection

1. **Validator Weight**:
   - PoC contribution (60%) + token holding (40%)
   - Labor nodes (long-term online) have greater weight

2. **Prefix Anti-Monopoly**:
   - More devices = lower single validator weight
   - Minimum weight 10%

### 9.3 Malicious Block Detection

1. **Timestamp Anomaly**:
   - Block timestamp check
   - Reject abnormally fast or slow blocks
2. **Work Anomaly**: Blocks with sudden hash power surge

---

## 10. Code Examples

### 10.1 Create Blockchain

```go
import (
    "github.com/jinba225/v6coin-protocol/pkg/blockchain"
    "github.com/jinba225/v6coin-protocol/pkg/consensus"
)

// Create new blockchain
config := &BlockChainConfig{
    GenesisBlock: genesisBlock,
    StateDB:      stateDB,
    Validator:   rewardDistributor,
}

chain := NewBlockChain(config)

// Add genesis block
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

### 10.2 Query Block

```go
// Query block by height
block, err := chain.GetBlockByHeight(100)
if err != nil {
    log.Error("Failed to get block:", err)
    return
}

fmt.Printf("Block Height: %d\n", block.Header.Height)
fmt.Printf("Prev Hash: %s\n", block.Header.PrevBlockHash)
fmt.Printf("Timestamp: %d\n", block.Header.Timestamp)
```

### 10.3 Query Transaction

```go
// Query transactions in block
for _, tx := range block.Transactions {
    fmt.Printf("Hash: %s\n", tx.HashString())
    fmt.Printf("From: %s\n", tx.From.String())
    fmt.Printf("To: %s\n", tx.To.String())
    fmt.Printf("Amount: %d V6\n", tx.Amount/1e9)
}
```

---

## 11. Appendix

### A. Merkle Tree Algorithm Details

**Leaf Node Hash Calculation**:
```go
// Calculate leaf node hash
func CalculateLeafHash(hash []byte, data []byte) []byte {
    combined := append(hash, data...)
    return sha256.Sum256(combined)
}
```

**Root Node Hash Calculation**:
```go
// Calculate root node hash
func CalculateRootHash(childHashes [][]byte, leafHashes [][]byte) []byte []byte {
    if len(childHashes) != len(leafHashes) {
        return nil
    }

    // Iteratively calculate hashes until only two nodes remain
    for i := 0; i < len(leafHashes); i++ {
        newHashes := make([][]byte, len(leafHashes))

        for j := 0; j < len(childHashes); j++ {
            combined := append(childHashes[i], leafHashes[j]...)
            hash := sha256.Sum256(combined)
            newHashes[i] = hash[:]
        }

        leafHashes = newHashes
    }

    // Final root hash (minimum of two root node hashes)
    rootHash := sha256.Sum256(append(childHashes[0], childHashes[1]))
    return rootHash
}
```

### B. Data Size Calculation

| Data Type | Size (Approx) |
|---------|-------------|
| Block Header | 40 bytes |
| Transaction (without data) | 83 bytes |
| Transaction Data | Variable length |
| Block Signature | 96 bytes (3 validators × 32 bytes) |

### C. Historical Growth

Assumptions:
- Block interval: 10 seconds
- Transactions per block: Average 100
- Storage efficiency: 100 KB/s per byte

**1 Year Data Volume**:
- Block count: 31,536,000
- Transaction count: 3,153,600,000
- Total size: Approximately 3.2 TB

---

## 12. Version History

| Version | Date | Update |
|---------|------|--------|
| v1.0 | 2026-02-01 | Initial version |

---

## 13. Reference Documentation

- [V6Coin Whitepaper (Chinese)](../whitepaper/V6Coin_Whitepaper_CN.md)
- [V6Coin Whitepaper (English)](../whitepaper/V6Coin_Whitepaper_EN.md)
- [IPv6 Extension Header Specification](./01-ipv6-header-spec.md)
- [CGA Address Mapping Specification](./02-cga-address-spec.md)
- [PoC Consensus Specification](./03-poc-consensus-spec.md)
- [Transaction Validation Specification](./04-transaction-spec.md)
- [Cryptographic Algorithm Specification](./05-crypto-spec.md)
- [P2P Network Protocol Specification](./06-p2p-protocol-spec.md)
- [Blockchain Data Structure Specification](./07-blockchain-spec.md)
