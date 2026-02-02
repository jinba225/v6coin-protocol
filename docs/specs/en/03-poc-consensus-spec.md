# PoC Consensus Specification

**Version**: v1.0
**Date**: 2026-02-01

---

## 1. Overview

V6Coin uses Proof of Connection (PoC) as its consensus mechanism, achieving consensus through nodes' contributions to network connectivity. PoC aims to:

- **Energy Efficient**: Avoid PoW's energy waste
- **Decentralized**: Prevent large nodes from monopolizing
- **Fair**: Long-term contributors get more voting weight

### Design Goals

1. **Green Consensus**: Based on network connection contribution, not computational power
2. **Contribution Fairness**: "Labor nodes" providing long-term network services get more voice
3. **Prefix Anti-Monopoly**: Prevent device concentration on single prefixes
4. **Dynamic Validator Set**: Adjust validators based on contribution scores

---

## 2. Core Concepts

### 2.1 Proof of Connection (PoC)

PoC proves a node continuously provides effective services to the network, including:
- Data forwarding
- Transaction broadcasting
- Block validation
- Network routing

### 2.2 Contribution Score Calculation

Node contribution score is weighted by three dimensions:

1. **Online Time Weight**: 60%
   - Cumulative online time within online window (90 days)
   - Formula: `min(onlineTime / windowTime, 1.0) * 0.6`

2. **Packet Loss Weight**: 10%
   - Node stability, lower packet loss = higher score
   - Formula: `(1.0 - packetLossRate) * 0.1`

3. **Forwarded Bytes Weight**: 30%
   - Total data forwarded, normalized against network average
   - Formula: `min(forwardedBytes / 1GB, 1.0) * 0.3`

**Total Contribution Formula**:
```
ContributionScore = (OnlineTimeWeight * 0.6) +
                   (PacketLossWeight * 0.1) +
                   (ForwardedWeight * 0.3)
```

### 2.3 Validator Election

1. **Validator Count**: 100 top validators
2. **Election Method**: Sort by contribution score descending
3. **Rotation Mechanism**: Rotate validators for each block height
4. **Current Validator**: Selected based on block height modulo

### 2.4 Prefix Anti-Monopoly

1. **Device Limit**: Maximum 50 devices per /64 prefix
2. **Reward Decay**: More devices in a prefix = lower single-node reward
3. **Formula**:
   ```
   RewardMultiplier = max(0.01, R0 * e^(-n/10))
   ```
   Where:
   - `R0` = 1.0 (base reward multiplier)
   - `n` = number of devices in that prefix
   - `e` = natural logarithm base (~2.718)

---

## 3. Data Structure Definitions

### 3.1 Block

```go
type Block struct {
    Header        *BlockHeader        // Block header
    Transactions  []*Transaction      // Transaction list
    ValidatorSigs [][]byte       // Signatures from all validator nodes
}
```

### 3.2 Block Header

```go
type BlockHeader struct {
    Version       uint16         // Protocol version (0x0100)
    Height        uint64          // Block height
    PrevBlockHash []byte         // Hash of previous block
    MerkleRoot    []byte         // Merkle root of transactions
    Timestamp     uint64          // Block timestamp (Unix)
    ValidatorID   p2p.PeerID      // ID of validator node who created this block
    StateRoot     []byte          // Root of state trie
    Signature     []byte          // Validator's signature
}
```

### 3.3 Transaction

```go
type Transaction struct {
    Version   uint16    // Transaction version
    Type      TxType     // Transaction type
    From      net.IP     // Sender address (16 bytes)
    To        net.IP     // Receiver address (16 bytes)
    Amount    uint64     // Amount in nano-V6
    Fee       uint64     // Transaction fee
    Nonce     uint64     // Random number
    Timestamp uint64    // Transaction timestamp
    Signature []byte     // Ed25519 signature (32 bytes)
    Data      []byte     // Optional transaction data
}
```

### 3.4 Node Contribution

```go
type NodeContribution struct {
    NodeID        p2p.PeerID      // Node ID
    OnlineTime    time.Duration    // Total online time
    LastOnline    time.Time        // Last online time
    PacketLoss    float64          // Packet loss rate (0.0 - 1.0)
    Forwarded     uint64          // Total forwarded bytes
    Score         float64          // Contribution score
    RewardBalance uint64          // Pending reward balance
}
```

### 3.5 Validator Set

```go
type ValidatorSet struct {
    Validators      []p2p.PeerID    // Current validators (sorted by contribution)
    RoundRobinIndex int             // Rotation index
}
```

### 3.6 Prefix Monitor

```go
type PrefixMonitor struct {
    prefixCounts map[string]int    // /64 prefix -> device count
}
```

---

## 4. Consensus Process

### 4.1 Block Generation

1. **Validator Selection**:
   - Calculate current block height: `currentHeight + 1`
   - Select validator: `validatorIndex = currentHeight % 100`
   - Get validator ID from validator set

2. **Transaction Packaging**:
   - Select transactions from pool
   - Sort by fee priority
   - Reach block size limit or 10 second timeout

3. **Block Validation**:
   - Verify all transaction signatures
   - Verify all transaction balances
   - Calculate Merkle tree

4. **Block Signing**:
   - Validator signs block header
   - Collect multiple validator signatures
   - Broadcast to network

5. **Block Broadcast**:
   - Broadcast new block via P2P network
   - Nodes validate and add to local chain

### 4.2 Block Validation

1. **Version Check**:
   - Check `version == 0x0100`

2. **Height Check**:
   - Check `height == prevBlock.height + 1`

3. **Previous Block Hash**:
   - Check `prevBlockHash` matches previous block's hash

4. **Transaction Verification**:
   - Verify all transaction signatures
   - Verify all transaction balances
   - Check transaction uniqueness

5. **Merkle Root Check**:
   - Recalculate transaction Merkle tree
   - Verify `merkleRoot` matches computed value

6. **Timestamp Check**:
   - Check `timestamp >= previousBlock.timestamp`
   - Check `timestamp <= current time + 2 minutes` (clock drift tolerance)

### 4.3 Longest Chain Selection

1. **Chain Validation**:
   - Full validation from genesis to candidate block
   - Verify all block signatures

2. **Cumulative Work**:
   - Each block represents 1 unit of work
   - Calculate total work for both chains

3. **Select Longest Chain**:
   - Select chain with highest cumulative work
   - If work equal, select chain with newer timestamp

4. **Chain Switching**:
   - When longer chain found, automatically switch
   - Old blocks kept locally for potential reorg

---

## 5. Reward Mechanism

### 5.1 Block Rewards

1. **Base Reward**: 100 V6 per block
2. **Transaction Fees**: Sum of all transaction fees in block
3. **Total Block Reward**:
   ```
   TotalReward = BaseReward + TransactionFees
   ```

### 5.2 Mining Distribution

- **70% PoC Contribution**: Distributed based on node contribution scores
- **15% Infrastructure**: Used for development, testing, etc.
- **15% Universal Airdrop**: Reward early participants

### 5.3 Halving Cycle

Every 4 years (approximately 1,401,600 blocks), block rewards halve:
- First halving: Block 1,401,600
- Second halving: Block 2,803,200
- And so on...

### 5.4 Prefix Anti-Monopoly Reward Adjustment

1. **Calculate Multiplier**: Based on prefix device count
   ```
   multiplier = CalculateRewardMultiplier(prefix)
   ```

2. **Final Reward**:
   ```
   FinalReward = BaseReward * multiplier
   ```

3. **Reward Examples**:
   - 1 device: `100 * 1.0 = 100 V6`
   - 10 devices: `100 * 0.91 = 91 V6`
   - 30 devices: `100 * 0.74 = 74 V6`
   - 50 devices: `100 * 0.51 = 51 V6`

---

## 6. Security Mechanisms

### 6.1 Sybil Attack Protection

1. **Identity Binding**: One IP address can only register one node ID
2. **Contribution Scoring**: Requires sustained online time, short-term nodes cannot gain high scores
3. **Minimum Online Time**: Must be online at least 24 hours to gain contribution score

### 6.2 Double Spend Protection

1. **Transaction Pool Dedup**: Same hash transactions only keep one
2. **Balance Verification**: Verify sender's balance for each transaction
3. **Longest Chain Rule**: Only accept transactions on longest chain

### 6.3 Timestamp Attack Protection

1. **Block Timestamp Limit**:
   - Block timestamp cannot exceed current time + 2 minutes
   - Reject future-timestamped blocks

2. **Transaction Timestamp Limit**:
   - Transaction timestamp cannot exceed current time + 5 minutes
   - Reject future-timestamped transactions

### 6.4 Network Split Protection

1. **Longest Chain Selection**: Automatically handle network splits
2. **Validator Signatures**: Require sufficient validator signatures for confirmation
3. **Reorg Limit**: Limit reorg depth to prevent long-term chain instability

---

## 7. Parameter Configuration

### 7.1 Consensus Parameters

| Parameter | Value | Description |
|-----------|-------|-------------|
| BlockInterval | 10 seconds | Block production interval |
| ValidatorCount | 100 | Number of validators |
| ConfirmationBlocks | 3 | Blocks needed for transaction confirmation |
| OnlineWindow | 90 days | Contribution score calculation time window |
| MaxPacketLossRate | 30% | Maximum allowed packet loss rate |
| MaxPrefixDevices | 50 | Maximum devices per prefix |
| RevenueDecayFactor | 10.0 | Reward decay factor |
| InitialBlockReward | 100 V6 | Initial block reward |
| HalvingYears | 4 years | Reward halving cycle |

### 7.2 Prefix Anti-Monopoly Parameters

| Parameter | Value | Description |
|-----------|-------|-------------|
| MaxLogicalSubnets | 8 | Maximum logical subnets per core address |
| MinMigrationInterval | 1 hour | Minimum temporary address switch interval |
| MaxMigrationInterval | 30 days | Maximum temporary address validity period |

---

## 8. Code Examples

### 8.1 Create Consensus Engine

```go
import (
    "github.com/jinba225/v6coin-protocol/pkg/consensus"
)

// Create new consensus engine
engine := consensus.NewConsensusEngine()

// Add transaction
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

### 8.2 Update Node Contribution

```go
// Update node's online time
nodeID := p2p.PeerID("node1")

// Node has been online for 1 hour
engine.UpdateContribution(nodeID, func(nc *NodeContribution) {
    nc.UpdateOnlineTime(1 * time.Hour)
})
```

### 8.3 Calculate Contribution Score

```go
// Calculate node's contribution score
engine.UpdateContribution(nodeID, func(nc *NodeContribution) {
    // Update packet loss (assume measured 2%, i.e., 0.02)
    nc.UpdatePacketLoss(0.02)
    
    // Update forwarded bytes (assume forwarded 100MB data)
    nc.AddForwardedBytes(100 * 1024 * 1024)
    
    // Contribution score automatically recalculated:
    // onlineTime: 24 hours / 90 days = 0.00011 (60% weight)
    // packetLoss: (1.0 - 0.02) = 0.98 (10% weight)
    // forwarded: 100MB / 1GB = 0.10 (30% weight)
    // Score = 0.00011*0.6 + 0.98*0.1 + 0.10*0.3 = 0.000066 + 0.098 + 0.03 = 0.128
})
```

### 8.4 Prefix Registration and Anti-Monopoly

```go
// Register device prefix
deviceAddr := net.ParseIP("2001:db8::1")

// Register prefix (increment device count)
engine.prefixMonitor.RegisterPrefix(deviceAddr)

// Calculate reward multiplier (assume 20 devices in this prefix)
multiplier := engine.prefixMonitor.CalculateRewardMultiplier(deviceAddr)
fmt.Printf("Reward multiplier: %.2f", multiplier) // Output: 0.82
```

### 8.5 Block Validation

```go
// Validate newly received block
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

// Add to blockchain
engine.SetChainHead(newBlock)
```

---

## 9. Validation Rules

### 9.1 Contribution Score Validation

1. **Online Time Range**: `0 ≤ onlineTime ≤ OnlineWindow`
2. **Packet Loss Range**: `0.0 ≤ packetLoss ≤ MaxPacketLossRate`
3. **Contribution Score Range**: `0.0 ≤ score ≤ 1.0`
4. **Score Update**: Can only increase, cannot decrease

### 9.2 Validator Set Validation

1. **Validator Count**: Must be exactly `ValidatorCount` validators (100)
2. **Validator Sorting**: Must be sorted by contribution score descending
3. **Round Robin Index**: Must be within `[0, ValidatorCount-1]` range
4. **Validator Dedup**: No duplicate validator IDs

### 9.3 Block Validation

1. **Signature Count**: Must have sufficient validator signatures (recommended ≥ 3)
2. **Merkle Root Validation**: Recalculate and compare
3. **Block Interval Check**: Block timestamp interval must be within `BlockInterval ± 2 seconds`

### 9.4 Prefix Monitor Validation

1. **Prefix Length**: Must be /64 prefix (8 bytes)
2. **Device Count Limit**: `deviceCount ≤ MaxPrefixDevices`
3. **Prefix Format**: Must be valid IPv6 prefix

---

## 10. Error Handling

### 10.1 Consensus Errors

| Error Code | Description | Handling |
|------------|-------------|----------|
| ERR_CONSENSUS_INVALID_BLOCK | Invalid block | Reject block, don't forward |
| ERR_CONSENSUS_MISSING_TRANSACTION | Transaction not found | Reject block, request missing transaction |
| ERR_CONSENSUS_DUPLICATE_TRANSACTION | Duplicate transaction | Reject block, check transaction pool |
| ERR_CONSENSUS_INVALID_SIGNATURE | Invalid signature | Reject block, verify public key |
| ERR_CONSENSUS_STALE_BLOCK | Old block | Reject block, already exists in chain |
| ERR_CONSENSUS_PARENT_NOT_FOUND | Previous block not found | Reject block, request parent block |

### 10.2 Recovery Strategies

1. **Transient Network Split**: Wait for 3-6 blocks, automatically recover
2. **Deep Reorg**: Log warning, but allow reorg
3. **Abnormal Block Timestamp**: Sync network time, reject abnormal blocks
4. **Validator Offline**: Remove from validator set, re-elect

---

## 11. Appendix

### A. Contribution Score Calculation Details

Online time normalization:
```
NormalizedOnlineTime = min(
    totalOnlineTime / OnlineWindow,
    1.0
)
```

Packet loss normalization:
```
NormalizedPacketLoss = 1.0 - packetLossRate
```

Forwarded bytes normalization (assuming network average forwarded volume is `AvgForwarded`):
```
NormalizedForwarded = min(
    forwardedBytes / AvgForwarded,
    1.0
)
```

Final contribution score:
```
Score = NormalizedOnlineTime * 0.6 +
        NormalizedPacketLoss * 0.1 +
        NormalizedForwarded * 0.3
```

### B. Prefix Anti-Monopoly Mathematical Derivation

Reward decay formula uses exponential decay:

```
f(n) = R0 * e^(-n/10)
```

Where:
- `f(n)` = Reward multiplier with n devices
- `R0` = 1.0 (base multiplier when no device limit)
- `n` = Number of devices in that prefix
- `e` = Natural logarithm base (~2.71828)

Calculation examples:
- n=1: f(1) = 1.0 * e^(-0.1) ≈ 0.905
- n=10: f(10) = 1.0 * e^(-1) ≈ 0.368
- n=30: f(30) = 1.0 * e^(-3) ≈ 0.050
- n=50: f(50) = 1.0 * e^(-5) ≈ 0.007

Then take max of base (0.01) and calculated value, ensuring at least 1% base reward:
```
RewardMultiplier = max(0.01, min(1.0, f(n)))
```

### C. Merkle Tree Algorithm

V6Coin uses standard Merkle Patricia Tree (MPT):

1. **Node Types**:
   - Extension nodes: Contain child node hashes and value
   - Branch nodes: Contain 16 child node hashes
   - Leaf nodes: Contain transaction hashes

2. **Tree Construction**:
   - Transactions sorted by hash
   - Tree built bottom-up
   - Calculate each node's hash

3. **Merkle Proof**:
   - Provide path from leaf to root
   - Include sibling node hashes
   - Verify without full tree

---

## 12. Version History

| Version | Date | Changes |
|---------|------|---------|
| v1.0 | 2026-02-01 | Initial version |

---

## 13. Reference Documentation

- [RFC 3971](https://datatracker.ietf.org/doc/html/rfc3971) - IPv6 Cryptographically Generated Addresses
- [RFC 6980](https://datatracker.ietf.org/doc/html/rfc6980) - IPv6 Hop-by-Hop Options Header
- [V6Coin Whitepaper](../whitepaper/V6Coin_Whitepaper_CN.md)
- [V6Coin Whitepaper (English)](../whitepaper/V6Coin_Whitepaper_EN.md)
