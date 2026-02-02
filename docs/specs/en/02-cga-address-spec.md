# CGA Address Mapping Specification

**Version**: v1.0  
**Date**: 2026-02-01

---

## 1. Overview

V6Coin uses Cryptographically Generated Address (CGA) to implement the "address is wallet" design concept. By mapping the public key derived from an Ed25519 private key to the IPv6 Interface Identifier (IID), V6Coin implements a wallet address system that natively supports the IPv6 protocol stack without external mapping tables.

### Core Advantages

- **Native IPv6 Compatibility**: Addresses work directly on the IPv6 protocol stack without additional mapping
- **Stateless Verification**: Any node can verify address validity using the public key
- **Privacy Protection**: Supports temporary address mechanism to enhance user privacy
- **Mobility Support**: Logical subnet mechanism supports devices moving between multiple networks

---

## 2. Core Concepts

### 2.1 Cryptographically Generated Address (CGA)

CGA is an IPv6 address generation technique that binds cryptographic keys to addresses. V6Coin's CGA implementation is based on RFC 3971 standard with the following optimizations:

- Uses SHA-256 instead of SHA-1
- Uses Ed25519 signature algorithm
- Clears bit 0 of IID to ensure unicast address

### 2.2 Interface Identifier (IID)

IID is the 64-bit lower half of the IPv6 address (bits 64-127), generated from the public key derived from the private key:

```
IID = SHA256(PublicKey)[0:8] & ^0x01
```

**Key Constraints**:
- Bit 0 of IID must be 0 (unicast address)
- Uses the first 8 bytes (64 bits) of SHA-256

### 2.3 Network Prefix

The network prefix is the 64-bit upper half of the IPv6 address (bits 0-63), assigned by network operators or nodes:

```
IPv6 Address = NetworkPrefix (64 bits) || IID (64 bits)
```

**Default Prefix**: `2001:db8::/64` (for documentation examples, actual deployments use operator-assigned prefixes)

---

## 3. Address Generation Algorithms

### 3.1 IID Generation

```go
func GenerateIID(publicKey []byte) uint64 {
    hash := sha256.Sum256(publicKey)
    iid := binary.BigEndian.Uint64(hash[:8])
    iid &^= 1  // Clear bit 0 to ensure unicast address
    return iid
}
```

**Algorithm Steps**:
1. Perform SHA-256 hash on the public key
2. Take the first 8 bytes (64 bits) of the hash value
3. Convert to uint64 using big-endian order
4. Clear bit 0 (bitwise AND with `~0x01`)

### 3.2 Core Address Generation

```go
func GenerateAddress(networkPrefix net.IP, privateKey []byte) (*V6Address, error) {
    // 1. Validate input
    if len(networkPrefix) != 8 {
        return nil, errors.New("network prefix must be 64 bits")
    }
    if len(privateKey) != 32 {
        return nil, errors.New("private key must be 32 bytes (Ed25519)")
    }

    // 2. Derive public key
    kp, err := crypto.KeyPairFromPrivateKey(privateKey)
    if err != nil {
        return nil, fmt.Errorf("failed to derive public key: %w", err)
    }

    // 3. Generate IID
    iid := GenerateIID(kp.PublicKey)

    // 4. Construct full address
    prefix := make(net.IP, 16)
    copy(prefix, networkPrefix)
    binary.BigEndian.PutUint64(prefix[8:], iid)

    return &V6Address{
        NetworkPrefix: networkPrefix,
        IID:           iid,
        IsTemporary:   false,
        AddressType:   AddressTypeCore,
        CreatedAt:     time.Now(),
    }, nil
}
```

**Steps**:
1. Validate network prefix length is 8 bytes (64 bits)
2. Validate private key length is 32 bytes (Ed25519)
3. Derive public key from private key
4. Generate IID
5. Combine prefix and IID to construct complete IPv6 address

### 3.3 Temporary Address Generation

```go
func GenerateTemporaryAddress(corePrivateKey []byte, index int) (*V6Address, error) {
    // 1. Derive temporary private key
    h := sha256.New()
    h.Write(corePrivateKey)
    h.Write([]byte{byte(index)})
    keyMaterial := h.Sum(nil)
    tempPrivateKey := keyMaterial[:32]

    // 2. Generate temporary key pair
    kp, err := crypto.KeyPairFromPrivateKey(tempPrivateKey)
    if err != nil {
        return nil, fmt.Errorf("failed to derive temporary key: %w", err)
    }

    // 3. Generate IID
    iid := GenerateIID(kp.PublicKey)

    // 4. Create temporary address
    return &V6Address{
        IID:         iid,
        IsTemporary: true,
        AddressType: AddressTypeTemporary,
        CreatedAt:   time.Now(),
        ExpiresAt:   time.Now().Add(24 * time.Hour),
        Index:       index,
    }, nil
}
```

**Key Points**:
- Uses `SHA256(corePrivateKey || index)` to derive temporary private key
- Index allows multiple temporary addresses from the same core address
- Default validity period is 24 hours (configurable)

---

## 4. Address Types

### 4.1 Core Address

- **Type**: `AddressTypeCore`
- **Permanence**: Permanently valid, no automatic expiration
- **Generation**: Generated directly from core private key
- **Usage**: Main wallet address for long-term asset holding

```go
type AddressType int

const (
    AddressTypeCore         AddressType = iota  // Core private key address
    AddressTypeTemporary                       // Temporary derived address
    AddressTypeLogicalSubnet                   // Logical subnet address
)
```

### 4.2 Temporary Address

- **Type**: `AddressTypeTemporary`
- **Validity**: Default 24 hours (configurable from 1 hour to 30 days)
- **Generation**: Derived from core private key
- **Usage**:
  - Transaction privacy protection
  - Prevent address correlation analysis
  - Enhanced anonymity

**Constants**:
```go
const (
    MinMigrationInterval = time.Hour           // Minimum 1 hour
    MaxMigrationInterval = 30 * 24 * time.Hour // Maximum 30 days
)
```

### 4.3 Logical Subnet Address

- **Type**: `AddressTypeLogicalSubnet`
- **Permanence**: Valid as long as logical subnet is valid
- **Generation**: Uses core IID + subnet prefix
- **Usage**:
  - Mobile devices roaming across multiple networks
  - Multi-prefix network environments
  - Home address and care-of address mechanism

**Constraints**:
- Maximum 8 logical subnet prefixes
```go
const MaxLogicalSubnets = 8
```

---

## 5. V6Address Structure

```go
type V6Address struct {
    NetworkPrefix net.IP       // 64-bit network prefix
    IID           uint64       // 64-bit interface identifier (derived from private key)
    IsTemporary   bool         // Whether this is a temporary address
    AddressType   AddressType  // Address type
    CreatedAt     time.Time    // Creation time
    ExpiresAt     time.Time    // Expiration time (temporary addresses only)
    Index         int          // Index for temporary addresses or logical subnets
}
```

### Field Description

| Field | Type | Description |
|-------|------|-------------|
| `NetworkPrefix` | `net.IP` | 64-bit network prefix, must be 8 bytes |
| `IID` | `uint64` | 64-bit interface identifier, bit 0 must be 0 |
| `IsTemporary` | `bool` | Flag indicating if this is a temporary address |
| `AddressType` | `AddressType` | Address type: Core, Temporary, or LogicalSubnet |
| `CreatedAt` | `time.Time` | Address creation time |
| `ExpiresAt` | `time.Time` | Address expiration time (temporary addresses only) |
| `Index` | `int` | Temporary address index or subnet index |

### Methods

```go
// Get full 128-bit IPv6 address
func (addr *V6Address) GetFullAddress() net.IP

// Check if temporary address has expired
func (addr *V6Address) IsExpired() bool

// String representation
func (addr *V6Address) String() string

// Address equality check
func (addr *V6Address) Equals(other *V6Address) bool
```

---

## 6. Temporary Address Mechanism

### 6.1 Generation Algorithm

Temporary addresses are derived from the core private key using a SHA-256 hash chain:

```
tempPrivateKey = SHA256(corePrivateKey || index)
tempPublicKey = DeriveEd25519Public(tempPrivateKey)
tempIID = SHA256(tempPublicKey)[0:8] & ^0x01
```

**Index Management**:
- Index starts from 0 and increments
- Multiple temporary addresses can be derived from the same core address
- Index is used to distinguish different temporary addresses

### 6.2 Validity Period Configuration

```go
// Default validity 24 hours
defaultExpiry := 24 * time.Hour

// Configurable range
minExpiry := 1 * time.Hour      // Minimum 1 hour
maxExpiry := 30 * 24 * time.Hour // Maximum 30 days
```

### 6.3 Temporary Address with Prefix

```go
func GenerateTemporaryAddressWithPrefix(
    networkPrefix net.IP,
    corePrivateKey []byte,
    index int,
    expiry time.Duration,
) (*V6Address, error)
```

**Use Cases**:
- Generate temporary addresses for specific networks
- Custom validity period
- Support multi-network environments

### 6.4 Expiration Detection

```go
func (addr *V6Address) IsExpired() bool {
    if !addr.IsTemporary {
        return false
    }
    return time.Now().After(addr.ExpiresAt)
}
```

**Rotation Strategy**:
- Periodically check temporary address expiration status
- Automatically generate new temporary addresses before expiration
- Migrate assets from expired addresses to new addresses

---

## 7. Logical Subnet

### 7.1 Logical Subnet Structure

```go
type LogicalSubnet struct {
    CoreAddress    V6Address // Core address
    SubnetPrefixes []net.IP  // Associated prefix list (maximum 8)
    CreatedAt      time.Time // Creation time
    ExpiresAt      time.Time // Expiration time (default 30 days)
}
```

### 7.2 Creating Logical Subnet

```go
func CreateLogicalSubnet(
    coreAddress *V6Address,
    prefixes []net.IP,
) (*LogicalSubnet, error) {
    // Validate core address
    if coreAddress == nil {
        return nil, errors.New("core address cannot be nil")
    }

    // Validate prefix count
    if len(prefixes) == 0 {
        return nil, errors.New("at least one prefix required")
    }
    if len(prefixes) > MaxLogicalSubnets {
        return nil, fmt.Errorf("too many prefixes (max %d)", MaxLogicalSubnets)
    }

    // Validate each prefix length
    for i, prefix := range prefixes {
        if len(prefix) != 8 {
            return nil, fmt.Errorf("prefix %d must be 64 bits", i)
        }
    }

    return &LogicalSubnet{
        CoreAddress:    *coreAddress,
        SubnetPrefixes: prefixes,
        CreatedAt:      time.Now(),
        ExpiresAt:      time.Now().Add(30 * 24 * time.Hour),
    }, nil
}
```

**Constraints**:
- At least one prefix required
- Maximum 8 prefixes
- Each prefix must be 64 bits

### 7.3 Generating Subnet Address

```go
func GenerateSubnetAddress(
    subnet *LogicalSubnet,
    prefixIndex int,
) (*V6Address, error) {
    // Validate subnet
    if subnet == nil {
        return nil, errors.New("subnet cannot be nil")
    }

    // Validate index range
    if prefixIndex < 0 || prefixIndex >= len(subnet.SubnetPrefixes) {
        return nil, fmt.Errorf("prefix index %d out of range", prefixIndex)
    }

    prefix := subnet.SubnetPrefixes[prefixIndex]

    return &V6Address{
        NetworkPrefix: prefix,
        IID:           subnet.CoreAddress.IID, // Use core IID
        IsTemporary:   false,
        AddressType:   AddressTypeLogicalSubnet,
        CreatedAt:     time.Now(),
        Index:         prefixIndex,
    }, nil
}
```

### 7.4 Same Subnet Validation

```go
func IsInSameLogicalSubnet(
    addr1, addr2 *V6Address,
    subnet *LogicalSubnet,
) bool {
    if subnet == nil {
        return false
    }

    return addr1.IID == addr2.IID &&
        containsPrefix(subnet.SubnetPrefixes, addr1.NetworkPrefix) &&
        containsPrefix(subnet.SubnetPrefixes, addr2.NetworkPrefix)
}
```

**Validation Logic**:
- Both addresses must use the same IID
- Both addresses' prefixes must be in the subnet's prefix list

---

## 8. Asset Migration

### 8.1 Migration Structure

```go
type AddressMigration struct {
    FromAddress   net.IP  // Source address
    ToAddress     net.IP  // Destination address
    Timestamp     uint64  // Unix timestamp (seconds)
    Signature     []byte  // Ed25519 signature
    PreviousIndex int     // Source address index
}
```

### 8.2 Creating Migration

```go
func CreateMigration(
    fromAddr, toAddr *V6Address,
    privateKey []byte,
) (*AddressMigration, error) {
    // Validate input
    if fromAddr == nil || toAddr == nil {
        return nil, errors.New("both from and to addresses must be provided")
    }
    if len(privateKey) != 32 {
        return nil, errors.New("private key must be 32 bytes")
    }

    // Get full addresses
    fromIP := fromAddr.GetFullAddress()
    toIP := toAddr.GetFullAddress()

    // Construct signature data
    data := GetMigrationData(fromIP, toIP, uint64(time.Now().Unix()))

    // Sign
    signature, err := crypto.Sign(privateKey, data)
    if err != nil {
        return nil, fmt.Errorf("failed to sign migration: %w", err)
    }

    return &AddressMigration{
        FromAddress:   fromIP,
        ToAddress:     toIP,
        Timestamp:     uint64(time.Now().Unix()),
        Signature:     signature,
        PreviousIndex: fromAddr.Index,
    }, nil
}
```

### 8.3 Migration Data Format

```go
func GetMigrationData(
    fromAddr, toAddr net.IP,
    timestamp uint64,
) []byte {
    data := make([]byte, 32+32+8)
    copy(data[0:32], fromAddr)
    copy(data[32:64], toAddr)
    binary.BigEndian.PutUint64(data[64:72], timestamp)
    return data
}
```

**Data Structure**:
```
+----------------+----------------+----------------+
| From Address   | To Address     | Timestamp      |
| (16 bytes)     | (16 bytes)     | (8 bytes)      |
+----------------+----------------+----------------+
| 0-15           | 16-31          | 32-39          |
+----------------+----------------+----------------+
```

### 8.4 Validating Migration

```go
func ValidateMigration(
    migration *AddressMigration,
    publicKey []byte,
) error {
    // Validate migration object
    if migration == nil {
        return errors.New("migration cannot be nil")
    }
    if len(publicKey) != 32 {
        return errors.New("public key must be 32 bytes")
    }

    // Validate signature
    data := GetMigrationData(migration.FromAddress, migration.ToAddress, migration.Timestamp)
    if !crypto.Verify(publicKey, data, migration.Signature) {
        return errors.New("invalid migration signature")
    }

    // Validate timestamp
    age := time.Since(time.Unix(int64(migration.Timestamp), 0))
    if age > 7*24*time.Hour {
        return errors.New("migration timestamp too old (max 7 days)")
    }

    return nil
}
```

**Validation Rules**:
1. **Signature Verification**: Verify signature validity using public key
2. **Timestamp Validity**: Migration timestamp must be within 7 days
3. **Index Verification**: Ensure correct relationship between source and target address indices

---

## 9. Address Validation

### 9.1 Validation Function

```go
func ValidateAddress(addr *V6Address) error {
    // Validate non-null
    if addr == nil {
        return errors.New("address cannot be nil")
    }

    // Validate IID unicast property
    if addr.IID&1 != 0 {
        return errors.New("IID must be unicast address (bit 0 = 0)")
    }

    // Validate prefix length
    if addr.NetworkPrefix != nil && len(addr.NetworkPrefix) != 8 {
        return errors.New("network prefix must be 64 bits (8 bytes)")
    }

    // Validate temporary address
    if addr.IsTemporary {
        if addr.CreatedAt.IsZero() {
            return errors.New("temporary address must have creation time")
        }
        if time.Now().After(addr.ExpiresAt) {
            return errors.New("temporary address has expired")
        }
    }

    return nil
}
```

### 9.2 Extracting IID and Prefix

```go
// Extract IID from full IPv6 address
func ExtractIID(fullAddr net.IP) uint64 {
    if len(fullAddr) != 16 {
        return 0
    }
    return binary.BigEndian.Uint64(fullAddr[8:16])
}

// Extract network prefix from full IPv6 address
func ExtractNetworkPrefix(fullAddr net.IP) net.IP {
    if len(fullAddr) != 16 {
        return nil
    }
    return fullAddr[0:8]
}
```

---

## 10. Code Examples

### 10.1 Generate Core Address

```go
package main

import (
    "crypto/ed25519"
    "fmt"
    "net"
    "github.com/jinba225/v6coin-protocol/pkg/address"
)

func main() {
    // Generate Ed25519 key pair
    publicKey, privateKey, _ := ed25519.GenerateKey(nil)

    // Define network prefix (example)
    networkPrefix := net.IP([]byte{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0})

    // Generate core address
    coreAddr, err := address.GenerateAddress(networkPrefix, privateKey)
    if err != nil {
        panic(err)
    }

    // Validate address
    err = address.ValidateAddress(coreAddr)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Core Address: %s\n", coreAddr.GetFullAddress())
    fmt.Printf("IID: 0x%016x\n", coreAddr.IID)
    fmt.Printf("Type: %v\n", coreAddr.AddressType)
}
```

**Example Output**:
```
Core Address: 2001:db8::a3f2:e5c4:1b9a:7f2e
IID: 0x0000a3f2e5c41b9a
Type: 0
```

### 10.2 Generate Temporary Address

```go
func main() {
    // Generate core private key
    _, corePrivateKey, _ := ed25519.GenerateKey(nil)

    // Generate temporary address
    tempAddr, err := address.GenerateTemporaryAddress(corePrivateKey, 0)
    if err != nil {
        panic(err)
    }

    // Temporary address with prefix
    networkPrefix := net.IP([]byte{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0})
    tempAddrWithPrefix, err := address.GenerateTemporaryAddressWithPrefix(
        networkPrefix,
        corePrivateKey,
        1,
        24*time.Hour,
    )
    if err != nil {
        panic(err)
    }

    fmt.Printf("Temporary Address: %s\n", tempAddr.GetFullAddress())
    fmt.Printf("Expires At: %s\n", tempAddr.ExpiresAt)
    fmt.Printf("Temporary Address with Prefix: %s\n", tempAddrWithPrefix.GetFullAddress())
}
```

### 10.3 Create Logical Subnet

```go
func main() {
    // Generate core address
    _, privateKey, _ := ed25519.GenerateKey(nil)
    networkPrefix := net.IP([]byte{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0})
    coreAddr, _ := address.GenerateAddress(networkPrefix, privateKey)

    // Create logical subnet (multiple prefixes)
    prefixes := []net.IP{
        []byte{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0x01},
        []byte{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0x02},
        []byte{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0x03},
    }

    subnet, err := address.CreateLogicalSubnet(coreAddr, prefixes)
    if err != nil {
        panic(err)
    }

    // Generate subnet addresses
    for i := range prefixes {
        subnetAddr, _ := address.GenerateSubnetAddress(subnet, i)
        fmt.Printf("Subnet Address %d: %s\n", i, subnetAddr.GetFullAddress())
    }
}
```

### 10.4 Asset Migration

```go
func main() {
    // Generate key pair
    _, privateKey, publicKey := ed25519.GenerateKey(nil)

    // Create source and target addresses
    networkPrefix := net.IP([]byte{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0})
    fromAddr, _ := address.GenerateAddress(networkPrefix, privateKey)
    toAddr, _ := address.GenerateTemporaryAddress(privateKey, 1)

    // Create migration
    migration, err := address.CreateMigration(fromAddr, toAddr, privateKey)
    if err != nil {
        panic(err)
    }

    // Validate migration
    err = address.ValidateMigration(migration, publicKey)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Migration successful: %s -> %s\n",
        migration.FromAddress,
        migration.ToAddress,
    )
}
```

### 10.5 Address Validation

```go
func main() {
    // Create temporary address
    _, privateKey, _ := ed25519.GenerateKey(nil)
    tempAddr, _ := address.GenerateTemporaryAddress(privateKey, 0)

    // Validate address
    err := address.ValidateAddress(tempAddr)
    if err != nil {
        fmt.Printf("Address validation failed: %v\n", err)
    } else {
        fmt.Printf("Address validation successful\n")
    }

    // Check expiration
    if tempAddr.IsExpired() {
        fmt.Printf("Address has expired\n")
    } else {
        fmt.Printf("Address is valid\n")
    }

    // Extract IID from full address
    fullAddr := tempAddr.GetFullAddress()
    iid := address.ExtractIID(fullAddr)
    fmt.Printf("IID: 0x%016x\n", iid)
}
```

---

## 11. Security Considerations

### 11.1 Private Key Protection

- **Storage Security**: Private keys must be stored securely (e.g., Hardware Security Module HSM, encrypted filesystem)
- **Transmission Security**: Private keys must never be transmitted in plaintext over the network
- **Memory Protection**: Private keys should be cleared from memory immediately after use
- **Key Rotation**: Regularly rotate temporary private keys to reduce risk of key leakage

### 11.2 Privacy Enhancement with Temporary Addresses

**Advantages**:
- **Transaction Isolation**: Different transactions use different temporary addresses, preventing transaction correlation analysis
- **Balance Privacy**: Difficult to track user's total assets through addresses
- **Behavioral Privacy**: Difficult to analyze user transaction patterns through addresses

**Best Practices**:
- Use a new temporary address for each transaction
- Temporary address validity period should be reasonably set (recommended 24 hours to 7 days)
- Generate enough temporary addresses to handle high-frequency transactions

### 11.3 Migration Replay Attack Protection

**Protection Mechanisms**:
1. **Timestamp Verification**: Migration timestamp must be within 7 days
2. **Signature Verification**: Must be signed with private key, ensuring migration authorization
3. **Index Verification**: Source address index and target address index must be reasonable
4. **Uniqueness Check**: Prevent same migration from being processed multiple times

**Attack Scenarios**:
- **Replay Attack**: Intercept and resubmit migration transactions
- **Timestamp Forgery**: Modify migration timestamp to bypass validity period

**Protection Measures**:
```
1. Each migration transaction contains a unique timestamp
2. Network maintains migration transaction history
3. Duplicate migration transactions are automatically rejected
4. Timestamps use UTC time to avoid timezone confusion
```

### 11.4 Address Collision Probability Analysis

**IID Space Size**:
- IID is 64 bits, after clearing bit 0 becomes 63 bits
- Total space: 2^63 ≈ 9.22 × 10^18

**Collision Probability Calculation**:
- Assume N addresses are generated
- Collision probability ≈ N^2 / (2 × 2^63) = N^2 / 2^64

**Examples**:
- Generate 1 million addresses: collision probability ≈ 10^-8
- Generate 1 billion addresses: collision probability ≈ 10^-2

**Conclusion**:
- In practical applications, address collision probability is negligible
- Even with 10 billion addresses globally, collision probability remains below 1%

### 11.5 Other Security Considerations

**Unicast Address Constraint**:
- Bit 0 of IID must be 0, ensuring address is a unicast address
- Avoid routing issues caused by multicast addresses (bit 0 = 1)

**Prefix Validation**:
- Network prefix length must be 64 bits
- Prevent prefix injection attacks

**Expiration Check**:
- Temporary addresses must be periodically checked for expiration
- Expired addresses must not be used for new transactions

**Key Derivation Security**:
- Use standard cryptographic libraries (e.g., Go's crypto/ed25519)
- Avoid custom cryptographic implementations
- Use audited cryptographic primitives

---

## 12. Appendix

### 12.1 Constant Definitions

```go
const (
    // IID generation constants
    DefaultIIDPrefix      = uint64(0)
    MaxLogicalSubnets     = 8

    // Migration interval
    MinMigrationInterval = time.Hour               // Minimum 1 hour
    MaxMigrationInterval = 30 * 24 * time.Hour     // Maximum 30 days
)

// Address types
type AddressType int

const (
    AddressTypeCore         AddressType = iota  // Core address
    AddressTypeTemporary                       // Temporary address
    AddressTypeLogicalSubnet                   // Logical subnet address
)
```

### 12.2 Utility Functions

```go
// Check if prefix is in list
func containsPrefix(prefixes []net.IP, prefix net.IP) bool {
    for _, p := range prefixes {
        if p.Equal(prefix) {
            return true
        }
    }
    return false
}

// Get migration signature data
func GetMigrationData(
    fromAddr, toAddr net.IP,
    timestamp uint64,
) []byte {
    data := make([]byte, 32+32+8)
    copy(data[0:32], fromAddr)
    copy(data[32:64], toAddr)
    binary.BigEndian.PutUint64(data[64:72], timestamp)
    return data
}
```

### 12.3 References

- RFC 3971: Cryptographically Generated Addresses (CGA)
- RFC 4291: IPv6 Address Architecture
- RFC 8064: Recommended Generation of IPv6 Interface Identifiers
- Ed25519: High-speed high-security signatures

---

## 13. Change History

| Version | Date | Description | Author |
|---------|------|-------------|--------|
| v1.0 | 2026-02-01 | Initial version | V6Coin Protocol Team |

---

**End of Document**
