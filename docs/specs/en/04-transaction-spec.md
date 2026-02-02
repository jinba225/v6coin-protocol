# Transaction Validation Specification

**Version**: v1.0
**Date**: 2026-02-01

---

## 1. Overview

V6Coin transaction validation ensures only valid, legitimate transactions are included in the blockchain. The validation process includes multiple checks: signature verification, balance verification, timestamp validation, and transaction structure validation.

### Design Goals

1. **Security**: Prevent double-spending, replay attacks, invalid signatures
2. **Integrity**: Ensure transaction data is complete and format is correct
3. **Fairness**: Prevent spam transactions and denial of service attacks
4. **Performance**: Fast validation with low latency transaction confirmation

---

## 2. Core Concepts

### 2.1 Transaction Type (TxType)

V6Coin supports multiple transaction types:

| Transaction Type | Value | Description |
|---------------|-------|-------------|
| Online Transaction | 0x01 | Real-time embedded in IPv6 extension header |
| Offline Transaction | 0x02 | Delayed settlement, local signature cache |
| Asset Migration | 0x03 | Cross-address asset migration |
| Staking Transaction | 0x04 | Lock tokens to gain voting rights |
| Unstaking Transaction | 0x05 | Unlock tokens |
| Governance Proposal | 0x06 | Submit governance proposal |
| Vote Transaction | 0x07 | Vote on proposals |

### 2.2 Transaction Pool (Transaction Pool)

Transaction pool manages pending transactions:

1. **Max Capacity**: 10,000 transactions
2. **Priority Mechanism**:
   - Based on fee and Gas limits
   - High-priority transactions are packed first
3. **Deduplication**:
   - Only one copy of same hash transactions
   - Deduplication based on `tx.Hash()`

---

## 3. Data Structure Definitions

### 3.1 Transaction

```go
type Transaction struct {
    Version   uint16         // Transaction version
    Type      TxType          // Transaction type
    From      net.IP          // Sender address (16 bytes)
    To        net.IP          // Receiver address (16 bytes)
    Amount    uint64          // Amount (nano-V6)
    Fee       uint64          // Transaction fee (nano-V6)
    Nonce     uint64          // Random number
    Timestamp uint64          // Transaction timestamp (Unix)
    Signature []byte          // Ed25519 signature (32 bytes)
    Data      []byte          // Optional transaction data
}
```

**Field Descriptions**:
- **Version**: Current version is 0x0100
- **From/To**: Full IPv6 address (16 bytes)
- **Amount**: In nano-V6 units (1 V6 = 10^9 nano-V6)
- **Fee**: Minimum fee is 1 nano-V6 per byte
- **Nonce**: Used to prevent replay attacks
- **Signature**: Ed25519 signature, 32 bytes

### 3.2 Transaction Hash

Transaction hash calculation:
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

## 4. Transaction Validation Rules

### 4.1 Transaction Format Validation

1. **Version Check**:
   ```
   tx.Version == 0x0100
   ```

2. **Transaction Type Check**:
   ```
   0x01 ≤ tx.Type ≤ 0x07
   ```

3. **Address Format Validation**:
   ```
   - Must be valid IPv6 address (16 bytes)
   - IID bit 0 must be 0 (unicast address)
   - Length must be 16 bytes
   ```

4. **Amount Validation**:
   ```
   tx.Amount > 0
   ```

5. **Fee Validation**:
   ```
   tx.Fee >= MinFeePerByte * tx.Size
   where MinFeePerByte = 1 nano-V6
   ```

6. **Timestamp Validation**:
   ```
   - Online transaction: |timestamp - current| ≤ 5 minutes
   - Offline transaction: |timestamp - current| ≤ MaxTTL (7 days)
   ```

### 4.2 Signature Validation

1. **Signature Length Check**:
   ```
   len(tx.Signature) == 32  // Ed25519 signature fixed 32 bytes
   ```

2. **Signature Verification**:
   ```
   Verify(publicKey, txData, tx.Signature)
   where:
     publicKey: Sender's public key
     txData: SHA256(Version + Type + From + To + Amount + Fee + Nonce + Timestamp + Data)
     Signature: Ed25519 signature
   ```

3. **Signature Algorithm**:
   - Uses Ed25519 elliptic curve signature
   - Signature verification fails = reject transaction

### 4.3 Balance Validation

1. **Sender Balance Validation**:
   ```
   balance = stateDB.GetBalance(tx.From)
   totalCost = tx.Amount + tx.Fee
   
   if balance < totalCost:
     return ErrInsufficientBalance
   ```

2. **Account Existence Check**:
   ```
   exists = stateDB.HasAccount(tx.From)
   
   if !exists:
     return ErrUnknownSender
   ```

3. **Nonce Validation**:
   ```
   currentNonce = stateDB.GetNonce(tx.From)
   
   if tx.Nonce <= currentNonce:
     return ErrInvalidNonce  // Prevent replay attacks
   ```

### 4.4 Duplicate Transaction Validation

1. **Transaction Pool Dedup**:
   ```
   if txPool.Contains(txHash):
     return ErrDuplicateTx
   ```

2. **On-Chain Transaction Check**:
   ```
   if blockchain.ContainsTransaction(txHash):
     // For offline transactions, may allow reuse of packed transactions
     // For online transactions, should reject
     return ErrTxAlreadyConfirmed
   ```

---

## 5. Transaction Pool Management

### 5.1 Adding Transaction to Pool

1. **Validate Transaction**:
   ```go
   func (tp *TxPool) AddTransaction(tx *Transaction) error {
       if err := tp.validateTx(tx); err != nil {
           return err
       }
   ```
2. **Calculate Priority**:
   ```go
   func (tp *TxPool) calculatePriority(tx *Transaction) float64 {
       // Base priority: 0.6
       // Gas limit adjustment based on transaction size
       base := 0.6
       
       gasUsed := uint64(len(tx.Data)) / 1000
       gasFactor := 1.0 - math.Min(1.0, float64(gasUsed)/float64(MaxGasLimit))
       
       return base * gasFactor
   }
   ```
3. **Enqueue**:
   ```go
   // If pool is full, try to evict lowest priority transaction
   if tp.pending.Len() >= MaxPoolSize {
       tp.evictLowestPriority()
   }
   
   heap.Push(&tp.pending, newItem)
   ```

### 5.2 Transaction Pool Capacity

1. **Max Transactions**: 10,000 transactions
2. **Max TTL**: 7 days (offline transactions)
3. **Gas Limit**: 1,000 gas per byte

### 5.3 Eviction Strategy

When transaction pool is full, evict transactions by priority:

1. Lowest priority
2. Longest wait time
3. Lowest gas price

---

## 6. Error Code Definitions

| Error Code | Error Type | Description |
|------------|-----------|-------------|
| ERR_INVALID_TX_VERSION | Invalid transaction version | Version is not 0x0100 |
| ERR_INVALID_TX_TYPE | Invalid transaction type | Unsupported transaction type |
| ERR_INVALID_ADDRESS | Invalid address format | Invalid IPv6 address |
| ERR_INVALID_AMOUNT | Invalid amount | Amount ≤ 0 |
| ERR_INVALID_FEE | Invalid fee | Insufficient fee |
| ERR_INVALID_NONCE | Invalid nonce | Nonce already used |
| ERR_INSUFFICIENT_BALANCE | Insufficient balance | Sender's balance insufficient |
| ERR_UNKNOWN_SENDER | Unknown sender | Sender account does not exist |
| ERR_INVALID_SIGNATURE | Invalid signature | Ed25519 signature verification failed |
| ERR_DUPLICATE_TX | Duplicate transaction | Transaction already exists in pool or on chain |
| ERR_TX_TOO_OLD | Transaction too old | Offline transaction expired |
| ERR_POOL_FULL | Pool is full | Cannot add more transactions |
| ERR_TX_EXPIRED | Transaction expired | Transaction validity period has expired |
| ERR_INVALID_DATA | Invalid transaction data | Data field format error |

---

## 7. Transaction Lifecycle

### 7.1 Pending Stage

Status after transaction enters pool:

1. **Pending**: Transaction added to pool, waiting to be packed
2. **Invalid**: Transaction validation failed, rejected
3. **Rejected**: Pool is full, transaction evicted

### 7.2 Confirmation Stage

1. **Mempool**: Transaction has been packed into block, waiting for 3 blocks to confirm
2. **Confirmed**: Transaction confirmed, balance deducted
3. **Orphaned**: Block overwritten by subsequent block

### 7.3 Expiration Processing

Offline and unconfirmed online transactions will expire:

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
        item := tp.all[hash]
        heap.Remove(&tp.pending, item.index)
        delete(tp.all, hash)
    }
    
    return len(toRemove)
}
```

---

## 8. Gas and Fee Model

### 8.1 Gas Calculation

1. **Base Gas**:
   - Transaction data: 1,000 gas per byte
   - Smart contract call: 50 gas (not currently used)

2. **Gas Limit**:
   - Block max gas: 1,000,000
   - Per transaction max gas: 50,000

### 8.2 Fee Calculation

```
BaseFee = MinFeePerByte * tx.Size
PriorityFee = BaseFee * PriorityMultiplier

FinalFee = BaseFee + PriorityFee
```

### 8.3 Fee Distribution

| Role | Allocation |
|-------|------------|
| Validator | 70% |
| Network Maintenance | 15% |
| Treasury | 15% |

---

## 9. Code Examples

### 9.1 Create Transaction Pool

```go
import (
    "github.com/jinba225/v6coin-protocol/pkg/tx"
)

// Create transaction pool
pool := tx.NewTxPool(stateDB)

// Add transaction
tx := &consensus.Transaction{
    Type:      consensus.TxTypeOnline,
    From:      senderAddr,
    To:        receiverAddr,
    Amount:    1000 * 1e9, // 1000 V6 = 1000 * 10^9 nano-V6
    Fee:       100 * 1e9,     // 100 V6 base fee
    Nonce:     uint64(time.Now().Unix()),
    Timestamp: uint64(time.Now().Unix()),
}

err := pool.AddTransaction(tx)
if err != nil {
    log.Error("Failed to add transaction:", err)
}
```

### 9.2 Validate Transaction

```go
import (
    "github.com/jinba225/v6coin-protocol/pkg/crypto"
)

// Verify transaction signature
func VerifyTransaction(tx *consensus.Transaction, publicKey []byte) error {
    // Calculate transaction hash
    txData := tx.Hash()
    
    // Verify Ed25519 signature
    if !crypto.Verify(publicKey, txData, tx.Signature) {
        return errors.New("invalid signature")
    }
    
    return nil
}
```

### 9.3 Query Pending Transactions

```go
// Get top 10 pending transactions (sorted by priority)
pendingTxs := pool.GetPendingTransactions(10)

for _, tx := range pendingTxs {
    fmt.Printf("Hash: %s\n", tx.HashString())
    fmt.Printf("From: %s\n", tx.From.String())
    fmt.Printf("To: %s\n", tx.To.String())
    fmt.Printf("Amount: %d V6\n", tx.Amount/1e9)
}
```

### 9.4 Clean Expired Transactions

```go
// Clean expired offline transactions
removed := pool.CleanExpired()

if removed > 0 {
    log.Printf("Removed %d expired transactions", removed)
}
```

---

## 10. Security Considerations

### 10.1 Replay Attack Prevention

1. **Nonce Mechanism**:
   - Each account maintains an incrementing nonce
   - Only accept transactions with nonce > currentNonce
   - Prevents reusing same transaction hash

2. **Time Window**:
   - Limit old transaction acceptance
   - Reject transactions older than 7 days (offline)

### 10.2 Double-Spend Protection

1. **Transaction Pool Dedup**:
   - Global deduplication based on `tx.Hash()`
   - Same hash transactions only keep latest one

2. **On-Chain Transaction Check**:
   - Confirmed transactions should not be repacked
   - Once confirmed, transaction should be removed from pool

### 10.3 DoS Protection

1. **Transaction Pool Size Limit**:
   - Max 10,000 pending transactions
   - Prevents memory exhaustion and DoS attacks
   - Block gas limit prevents infinite gas attacks

2. **Gas Limit**:
   - Per transaction max 50,000 gas
   - Prevents infinite gas consumption

3. **Fee Mechanism**:
   - All transactions must pay fee
   - Higher priority transactions pay higher fees
   - Prevents spam transactions

---

## 11. Version History

| Version | Date | Changes |
|---------|------|--------|
| v1.0 | 2026-02-01 | Initial version |

---

## 12. Reference Documentation

- [V6Coin Whitepaper (Chinese)](../whitepaper/V6Coin_Whitepaper_CN.md)
- [V6Coin Whitepaper (English)](../whitepaper/V6Coin_Whitepaper_EN.md)
- [IPv6 Extension Header Specification](./01-ipv6-header-spec.md)
- [CGA Address Mapping Specification](./02-cga-address-spec.md)
- [PoC Consensus Specification](./03-poc-consensus-spec.md)
- [Cryptographic Algorithm Specification](./05-crypto-spec.md)
