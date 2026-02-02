# Cryptographic Algorithm Specification

## Version Information

- **Version**: v1.0
- **Last Updated**: 2026-02-01
- **Protocol Version**: 0x0100

## Overview

V6Coin employs multiple cryptographic algorithms to ensure system security, privacy, and integrity. This specification defines the standard implementation of all cryptographic algorithms used in the V6Coin protocol, including signature, encryption, hashing, key management, and other core mechanisms.

### Design Goals

1. **Security**: Use industry-recognized cryptographic algorithms to ensure transaction security and data privacy
2. **Efficiency**: Optimize algorithm performance for low-power IoT devices
3. **Anonymity**: Protect user transaction privacy through technologies like ring signatures
4. **Extensibility**: Reserve algorithm upgrade space to support future extensions

---

## Ed25519 Signature Algorithm

Ed25519 is an elliptic curve digital signature algorithm based on the Twisted Edwards curve, used by V6Coin for transaction signing and address generation.

### Algorithm Parameters

| Parameter | Value | Description |
|-----------|--------|-------------|
| Curve | Ed25519 | 256-bit elliptic curve |
| Signature Length | 32 bytes | Fixed-length signature |
| Public Key Length | 32 bytes | Compressed public key format |
| Private Key Length | 32 bytes | Seed key |

### Key Generation

```go
// Generate Ed25519 key pair
func GenerateEd25519KeyPair() (publicKey, privateKey []byte, err error) {
    publicKey, privateKey, err := ed25519.GenerateKey(nil)
    if err != nil {
        return nil, nil, err
    }
    return publicKey, privateKey, nil
}
```

### Signature Generation

```go
// Sign message with private key
func SignEd25519(privateKey, message []byte) []byte {
    return ed25519.Sign(privateKey, message)
}
```

**Signature Message Format**:
```
signature = Sign(
    privateKey,
    SHA256(
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
)
```

### Signature Verification

```go
// Verify signature with public key
func VerifyEd25519(publicKey, message, signature []byte) bool {
    return ed25519.Verify(publicKey, message, signature)
}
```

### IID Generation

V6Coin uses Ed25519 public key to generate IPv6 Interface Identifier (IID):

```go
// Generate IID from public key
func GenerateIID(publicKey []byte) []byte {
    hash := sha256.Sum256(publicKey)
    iid := make([]byte, 8)
    copy(iid, hash[0:8])
    // Set bit 0 to 0 (unicast address)
    iid[0] &= 0x7F
    return iid
}
```

**IID Calculation Formula**:
```
IID = SHA256(publicKey)[0:8]
IID[0] &= 0x7F  // Ensure unicast address
```

### Address Generation

Complete V6Coin address (IPv6 address) generation flow:

```go
// Generate complete V6Coin address
func GenerateV6CoinAddress(networkPrefix, publicKey []byte) net.IP {
    iid := GenerateIID(publicKey)
    address := make(net.IP, 16)
    copy(address[0:8], networkPrefix[0:8])  // 64-bit network prefix
    copy(address[8:16], iid)                 // 64-bit IID
    return address
}
```

### Performance Characteristics

| Operation | Time Complexity | Measured Performance (Go) |
|-----------|----------------|--------------------------|
| Key Generation | O(n) | ~0.1ms |
| Signing | O(n) | ~0.05ms |
| Verification | O(n) | ~0.1ms |
| Public Key Compression | O(1) | <0.01ms |

---

## AES-256-GCM Encryption

AES-256-GCM is the symmetric encryption algorithm used by V6Coin for data encryption, supporting Authenticated Encryption (AEAD).

### Algorithm Parameters

| Parameter | Value | Description |
|-----------|--------|-------------|
| Key Length | 32 bytes | 256-bit key |
| Block Size | 16 bytes | 128-bit block |
| Nonce Length | 12 bytes | Recommended length |
| Tag Length | 16 bytes | Authentication tag |

### Key Derivation

V6Coin uses HKDF-SHA256 to derive AES keys from user passwords:

```go
// Derive AES key from password
func DeriveAESKey(password, salt []byte) []byte {
    // Use HKDF-SHA256
    hkdf := hkdf.New(sha256.New, password, salt, nil)
    key := make([]byte, 32)  // AES-256 key
    hkdf.Read(key)
    return key
}
```

### Encryption

```go
// AES-256-GCM encryption
func EncryptAES256GCM(key, plaintext []byte) (nonce, ciphertext, tag []byte, err error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, nil, nil, err
    }

    aesgcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, nil, nil, err
    }

    nonce = make([]byte, aesgcm.NonceSize())
    if _, err := rand.Read(nonce); err != nil {
        return nil, nil, nil, err
    }

    // GCM mode: ciphertext = nonce + sealed data
    sealed := aesgcm.Seal(nil, nonce, plaintext, nil)
    ciphertext = sealed[:len(sealed)-16]
    tag = sealed[len(sealed)-16:]

    return nonce, ciphertext, tag, nil
}
```

### Decryption

```go
// AES-256-GCM decryption
func DecryptAES256GCM(key, nonce, ciphertext, tag []byte) ([]byte, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }

    aesgcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    sealed := append(ciphertext, tag...)
    plaintext, err := aesgcm.Open(nil, nonce, sealed, nil)
    if err != nil {
        return nil, err
    }

    return plaintext, nil
}
```

### Application Scenarios

| Scenario | Usage | Notes |
|----------|--------|-------|
| Private Key Storage | Local Flash storage | Key derived from CPU serial + user password |
| Transaction Data | Offline transaction encryption | Local cache encryption for transaction data |
| Communication Encryption | End-to-end encryption | Optional feature |

### Security Requirements

1. **Nonce Uniqueness**: Each Nonce can only be used once to prevent key leakage
2. **Key Rotation**: Regularly change keys (recommended monthly)
3. **Secure Storage**: Keys must be encrypted, never stored in plaintext

---

## SHA-256 Hashing

SHA-256 is the standard hashing algorithm used by V6Coin for transaction hashing, block hashing, and Merkle tree calculations.

### Algorithm Parameters

| Parameter | Value | Description |
|-----------|--------|-------------|
| Hash Length | 32 bytes | 256-bit output |
| Block Size | 64 bytes | Input block size |
| Rounds | 64 | Compression rounds |

### Hash Calculation

```go
// Calculate SHA-256 hash
func HashSHA256(data []byte) []byte {
    hash := sha256.Sum256(data)
    return hash[:]
}
```

### Transaction Hash

```go
// Calculate transaction hash
func HashTransaction(tx *Transaction) []byte {
    data := bytes.NewBuffer(nil)
    binary.Write(data, binary.BigEndian, tx.Version)
    data.WriteByte(byte(tx.Type))
    data.Write(tx.From)
    data.Write(tx.To)
    binary.Write(data, binary.BigEndian, tx.Amount)
    binary.Write(data, binary.BigEndian, tx.Fee)
    binary.Write(data, binary.BigEndian, tx.Nonce)
    binary.Write(data, binary.BigEndian, tx.Timestamp)
    data.Write(tx.Data)
    return HashSHA256(data.Bytes())
}
```

### Merkle Tree

V6Coin uses Merkle trees to verify transaction integrity within blocks:

```go
// Merkle node
type MerkleNode struct {
    Left  *MerkleNode
    Right *MerkleNode
    Hash  []byte
}

// Calculate Merkle root
func CalculateMerkleRoot(txs []*Transaction) []byte {
    if len(txs) == 0 {
        return HashSHA256([]byte{})
    }

    var nodes []*MerkleNode
    for _, tx := range txs {
        nodes = append(nodes, &MerkleNode{
            Hash: tx.Hash(),
        })
    }

    for len(nodes) > 1 {
        var newLevel []*MerkleNode
        for i := 0; i < len(nodes); i += 2 {
            left := nodes[i]
            var right *MerkleNode
            if i+1 < len(nodes) {
                right = nodes[i+1]
            } else {
                right = left  // Odd node, duplicate last one
            }

            combined := append(left.Hash, right.Hash...)
            hash := HashSHA256(combined)
            newLevel = append(newLevel, &MerkleNode{
                Left:  left,
                Right: right,
                Hash:  hash,
            })
        }
        nodes = newLevel
    }

    return nodes[0].Hash
}
```

### Performance Characteristics

| Operation | Time Complexity | Measured Performance (Go) |
|-----------|----------------|--------------------------|
| Single Hash | O(n) | ~0.01ms (1KB) |
| Merkle Tree Build | O(n log n) | ~1ms (1000 txs) |
| Merkle Proof Verification | O(log n) | ~0.1ms |

---

## BIP-39 Mnemonic

BIP-39 is the mnemonic standard used by V6Coin for private key backup and recovery, making it easy for users to remember and backup private keys.

### Mnemonic Generation

```go
// Generate 12-word mnemonic
func GenerateMnemonic() (string, error) {
    // Generate 128-bit entropy
    entropy := make([]byte, 16)
    if _, err := rand.Read(entropy); err != nil {
        return "", err
    }

    // Calculate checksum
    hash := sha256.Sum256(entropy)
    checksum := hash[0] >> 4  // Take first 4 bits

    // Concatenate entropy + checksum
    combined := append(entropy, checksum)

    // Convert to mnemonic
    mnemonic := bip39.EncodeMnemonic(combined)
    return mnemonic, nil
}
```

### Mnemonic to Private Key

```go
// Derive Ed25519 private key from mnemonic
func MnemonicToPrivateKey(mnemonic, passphrase string) ([]byte, error) {
    // Generate seed
    seed := bip39.NewSeed(mnemonic, passphrase)

    // Derive Ed25519 private key from seed using HKDF
    salt := []byte("V6Coin ed25519 seed")
    hkdf := hkdf.New(sha256.New, seed, salt, nil)
    privateKey := make([]byte, 32)
    hkdf.Read(privateKey)

    return privateKey, nil
}
```

### Key Derivation Path

V6Coin supports hierarchical key derivation using HD wallet structure:

```
m / 44' / 931' / 0' / 0 / 0
 │   │      │     │   │  └─ Index
 │   │      │     │   └─ Account type (external=0, internal=1)
 │   │      │     └─ Account index
 │   │      └─ V6Coin coin type (931)
 │   └─ BIP-44 purpose
 └─ Master key
```

### Security Recommendations

1. **Offline Storage**: Mnemonics must be stored offline to avoid network leakage
2. **Physical Backup**: Use paper or metal backup, avoid digital storage
3. **Distributed Storage**: Multiple backups should be stored separately
4. **Regular Verification**: Periodically test mnemonic recovery process

---

## Ring Signatures

Ring signature is the core technology used by V6Coin for transaction anonymity, hiding the real signer among a group of anonymous members.

### Algorithm Overview

V6Coin adopts an improved ring signature algorithm based on Ed25519 curve:

```go
// Ring signature structure
type RingSignature struct {
    RingSize   int      // Ring size
    KeyImages  [][]byte // Key images
    Signatures []byte   // Ring signature
}
```

### Ring Signature Generation

```go
// Generate ring signature
func GenerateRingSignature(
    message []byte,
    privateKey, publicKey []byte,
    decoyPublicKeys [][]byte,
) (*RingSignature, error) {
    ringSize := len(decoyPublicKeys) + 1

    // Randomly select signer position
    secretIndex := rand.Intn(ringSize)

    // Generate key image
    keyImage := GenerateKeyImage(privateKey, publicKey)

    // Initialize challenge values
    challenges := make([][]byte, ringSize)
    responses := make([][]byte, ringSize)

    // Generate random values
    for i := 0; i < ringSize; i++ {
        if i != secretIndex {
            responses[i] = make([]byte, 32)
            if _, err := rand.Read(responses[i]); err != nil {
                return nil, err
            }
        }
    }

    // Calculate challenge values
    for i := 0; i < ringSize; i++ {
        idx := (secretIndex + i) % ringSize

        var pubKey []byte
        if idx == 0 {
            pubKey = publicKey
        } else {
            pubKey = decoyPublicKeys[idx-1]
        }

        // Calculate hash challenge
        hashInput := bytes.Join([][]byte{
            message,
            pubKey,
            keyImage,
            responses[idx],
        }, nil)

        challenges[idx] = HashSHA256(hashInput)
    }

    // Calculate signer's response
    privateKeyScalar := new(edwards25519.Scalar).SetBytesWithClamping(privateKey)
    challengeScalar := new(edwards25519.Scalar).SetBytesWithClamping(challenges[secretIndex])

    response := new(edwards25519.Scalar).Sub(privateKeyScalar, challengeScalar)
    responses[secretIndex] = response.Bytes()

    return &RingSignature{
        RingSize:   ringSize,
        KeyImages:  [][]byte{keyImage},
        Signatures: bytes.Join(responses, nil),
    }, nil
}
```

### Ring Signature Verification

```go
// Verify ring signature
func VerifyRingSignature(
    message []byte,
    publicKeys [][]byte,
    sig *RingSignature,
) bool {
    ringSize := len(publicKeys)

    // Parse signature
    responses := bytes.SplitN(sig.Signatures, []byte{}, ringSize)

    // Calculate each challenge value
    for i := 0; i < ringSize; i++ {
        pubKey := publicKeys[i]
        keyImage := sig.KeyImages[0]
        response := responses[i]

        hashInput := bytes.Join([][]byte{
            message,
            pubKey,
            keyImage,
            response,
        }, nil)

        computedChallenge := HashSHA256(hashInput)

        // Verify signature equation
        if !verifyRingEquation(pubKey, response, computedChallenge) {
            return false
        }
    }

    return true
}
```

### Ring Size Selection

| Scenario | Recommended Ring Size | Reason |
|----------|---------------------|--------|
| Normal Transaction | 5-7 | Balance anonymity and performance |
| Large Transaction | 10-15 | Enhanced anonymity |
| IoT Micro-payment | 3-5 | Optimize performance |

### Performance Characteristics

| Ring Size | Signing Time | Verification Time | Data Size |
|-----------|--------------|------------------|-----------|
| 5 | ~2ms | ~1ms | 160 bytes |
| 10 | ~4ms | ~2ms | 320 bytes |
| 15 | ~6ms | ~3ms | 480 bytes |

---

## Key Management Best Practices

### Key Storage

1. **Hardware Security Module** (Priority):
   - Use TPM/TEE/Secure Element
   - Private keys never leave secure area

2. **Software Encrypted Storage** (Alternative):
   - AES-256-GCM encryption
   - Key derivation: Key = HKDF(SHA256, Password + CPU Serial, Salt)

3. **Memory Security**:
   - Use secure memory (e.g., Go's secure memory)
   - Promptly erase temporary keys

### Key Rotation

1. **Regular Rotation**:
   - Master key: Once per year
   - Temporary address private key: Every 24 hours
   - Encryption key: Monthly

2. **Secure Migration**:
   - Use multi-signature scheme
   - Record migration proof on-chain

3. **Emergency Revocation**:
   - Immediately regenerate key via mnemonic
   - Broadcast key leakage notification

### Mnemonic Management

| Operation | Best Practice |
|-----------|---------------|
| Generation | Use cryptographic secure random number generator |
| Storage | Offline, paper/metal backup |
| Distribution | Avoid digital transmission |
| Verification | Periodically test recovery process |
| Destruction | Physical shredding or incineration |

---

## Security Considerations

### Key Security

1. **Entropy Source Quality**:
   - Use Cryptographically Secure Pseudo-Random Number Generator (CSPRNG)
   - Avoid pseudo-random generators like `math/rand`

2. **Key Leakage Protection**:
   - Side-channel attack protection
   - Timing attack protection
   - Fault attack protection

3. **Key Lifecycle Management**:
   - Security audit for key generation
   - Key usage logging
   - Key destruction verification

### Algorithm Security

1. **Ed25519 Security**:
   - Proven security: 128-bit security level
   - Side-channel resistant
   - Signature unforgeability

2. **AES-256-GCM Security**:
   - Authenticated encryption, prevents tampering
   - Nonce reuse causes key leakage
   - AEAD mode recommended

3. **SHA-256 Security**:
   - Collision resistance
   - Preimage resistance
   - Second preimage resistance

### Privacy Protection

1. **Ring Signature Anonymity**:
   - Unable to identify real signer
   - Unlinkability
   - Adjustable anonymity set size

2. **Address Anonymity**:
   - Dynamic temporary address switching
   - Core address not exposed externally
   - Logical subnet address pool

3. **Transaction Privacy**:
   - Amount hiding (optional)
   - Timestamp fuzzing
   - Anonymous path forwarding

### Implementation Security

1. **Constant-Time Algorithms**:
   - Avoid timing side-channels
   - All cryptographic operations use constant time

2. **Memory Safety**:
   - Secure erase of sensitive data
   - Avoid memory dump leakage
   - Use secure memory allocators

3. **Error Handling**:
   - Avoid error information leakage
   - Secure degradation strategy
   - Rollback sensitive operations on failure

---

## Implementation Examples

### Complete Transaction Signing Flow

```go
package main

import (
    "crypto/ed25519"
    "crypto/sha256"
    "encoding/binary"
    "fmt"
    "net"
)

type Transaction struct {
    Version   uint16
    Type      uint8
    From      net.IP
    To        net.IP
    Amount    uint64
    Fee       uint64
    Nonce     uint64
    Timestamp uint64
    Data      []byte
    Signature []byte
}

// Calculate transaction hash
func (tx *Transaction) Hash() []byte {
    h := sha256.New()
    binary.Write(h, binary.BigEndian, tx.Version)
    h.WriteByte(tx.Type)
    h.Write(tx.From)
    h.Write(tx.To)
    binary.Write(h, binary.BigEndian, tx.Amount)
    binary.Write(h, binary.BigEndian, tx.Fee)
    binary.Write(h, binary.BigEndian, tx.Nonce)
    binary.Write(h, binary.BigEndian, tx.Timestamp)
    h.Write(tx.Data)
    return h.Sum(nil)
}

// Sign transaction
func (tx *Transaction) Sign(privateKey []byte) error {
    message := tx.Hash()
    tx.Signature = ed25519.Sign(privateKey, message)
    return nil
}

// Verify transaction
func (tx *Transaction) Verify(publicKey []byte) bool {
    message := tx.Hash()
    return ed25519.Verify(publicKey, message, tx.Signature)
}

func main() {
    // Generate key pair
    publicKey, privateKey, _ := ed25519.GenerateKey(nil)

    // Create transaction
    tx := &Transaction{
        Version:   0x0100,
        Type:      0x01,
        From:      net.ParseIP("2001:db8::1"),
        To:        net.ParseIP("2001:db8::2"),
        Amount:    1000000,  // 1,000,000 nano-V6
        Fee:       1000,
        Nonce:     1234567890,
        Timestamp: uint64(1725148800),
        Data:      []byte{},
    }

    // Sign transaction
    tx.Sign(privateKey)

    // Verify transaction
    if tx.Verify(publicKey) {
        fmt.Println("Transaction signature verified")
    } else {
        fmt.Println("Transaction signature verification failed")
    }
}
```

### Key Derivation Flow

```go
package main

import (
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "golang.org/x/crypto/hkdf"
    "golang.org/x/crypto/bip39"
)

// Derive key pair from mnemonic
func KeyPairFromMnemonic(mnemonic, passphrase string) (publicKey, privateKey []byte, err error) {
    // Generate seed
    seed := bip39.NewSeed(mnemonic, passphrase)

    // Derive key using HKDF
    salt := []byte("V6Coin ed25519 seed")
    info := []byte("V6Coin key derivation")

    hkdf := hkdf.New(sha256.New, seed, salt, info)
    privateKey = make([]byte, 32)
    _, err = hkdf.Read(privateKey)
    if err != nil {
        return nil, nil, err
    }

    // Derive public key
    publicKey, _, err := ed25519.GenerateKey(privateKey)
    if err != nil {
        return nil, nil, err
    }

    return publicKey, privateKey, nil
}

func main() {
    // Generate mnemonic
    mnemonic, _ := bip39.NewMnemonic(128)
    fmt.Printf("Mnemonic: %s\n", mnemonic)

    // Derive key pair from mnemonic
    publicKey, privateKey, _ := KeyPairFromMnemonic(mnemonic, "")

    fmt.Printf("Private Key: %s\n", hex.EncodeToString(privateKey))
    fmt.Printf("Public Key: %s\n", hex.EncodeToString(publicKey))
}
```

### AES-256-GCM Encryption Example

```go
package main

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/hex"
    "fmt"
    "golang.org/x/crypto/hkdf"
    "golang.org/x/crypto/sha3"
)

// Derive key from password
func DeriveKey(password, salt []byte) []byte {
    hkdf := hkdf.New(sha3.New256, password, salt, nil)
    key := make([]byte, 32)
    hkdf.Read(key)
    return key
}

// Encrypt data
func Encrypt(key, plaintext []byte) (nonce, ciphertext []byte, err error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, nil, err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, nil, err
    }

    nonce = make([]byte, gcm.NonceSize())
    if _, err := rand.Read(nonce); err != nil {
        return nil, nil, err
    }

    ciphertext = gcm.Seal(nil, nonce, plaintext, nil)
    return nonce, ciphertext, nil
}

// Decrypt data
func Decrypt(key, nonce, ciphertext []byte) ([]byte, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return nil, err
    }

    return plaintext, nil
}

func main() {
    // Derive key
    password := []byte("my-secret-password")
    salt := make([]byte, 16)
    rand.Read(salt)
    key := DeriveKey(password, salt)

    // Encrypt data
    plaintext := []byte("Hello, V6Coin!")
    nonce, ciphertext, _ := Encrypt(key, plaintext)

    fmt.Printf("Key: %s\n", hex.EncodeToString(key))
    fmt.Printf("Nonce: %s\n", hex.EncodeToString(nonce))
    fmt.Printf("Ciphertext: %s\n", hex.EncodeToString(ciphertext))

    // Decrypt data
    decrypted, _ := Decrypt(key, nonce, ciphertext)
    fmt.Printf("Plaintext: %s\n", string(decrypted))
}
```

---

## Reference Documentation

- [RFC 8032: Ed25519](https://datatracker.ietf.org/doc/html/rfc8032)
- [RFC 5869: HKDF](https://datatracker.ietf.org/doc/html/rfc5869)
- [RFC 5116: AEAD](https://datatracker.ietf.org/doc/html/rfc5116)
- [RFC 6234: SHA-256](https://datatracker.ietf.org/doc/html/rfc6234)
- [BIP-39: Mnemonic Code](https://github.com/bitcoin/bips/blob/master/bip-0039.mediawiki)
- [RFC 6986: Merkle Tree](https://datatracker.ietf.org/doc/html/rfc6986)

---

## Version History

| Version | Date | Change Description |
|---------|------|-------------------|
| v1.0 | 2026-02-01 | Initial version, defining core cryptographic algorithms |

---

## Appendix

### A. Algorithm Complexity Summary

| Algorithm | Operation | Time Complexity | Space Complexity |
|-----------|-----------|------------------|------------------|
| Ed25519 | Key Generation | O(n) | O(n) |
| Ed25519 | Signing | O(n) | O(n) |
| Ed25519 | Verification | O(n) | O(n) |
| AES-256-GCM | Encryption | O(n) | O(n) |
| AES-256-GCM | Decryption | O(n) | O(n) |
| SHA-256 | Hashing | O(n) | O(1) |
| Merkle Tree | Build | O(n log n) | O(n) |
| Merkle Tree | Verification | O(log n) | O(1) |
| Ring Signature | Generation | O(n) | O(n) |
| Ring Signature | Verification | O(n) | O(n) |

### B. Constant Definitions

```go
const (
    // Ed25519
    Ed25519PrivateKeySize = 32
    Ed25519PublicKeySize  = 32
    Ed25519SignatureSize = 32

    // AES-256-GCM
    AES256KeySize   = 32
    AES256NonceSize = 12
    AES256TagSize   = 16

    // SHA-256
    SHA256Size = 32

    // BIP-39
    BIP39EntropySize128 = 16  // 12 words
    BIP39EntropySize256 = 32  // 24 words
    BIP39ChecksumBits  = 4

    // Ring Signature
    MinRingSize      = 3
    MaxRingSize      = 15
    DefaultRingSize  = 5
)
```

### C. Error Code Definitions

| Error Code | Error Type | Description |
|-----------|-----------|-------------|
| ERR_INVALID_PRIVATE_KEY | Invalid Private Key | Private key format error |
| ERR_INVALID_PUBLIC_KEY | Invalid Public Key | Public key format error |
| ERR_INVALID_SIGNATURE | Invalid Signature | Signature verification failed |
| ERR_INVALID_NONCE | Invalid Nonce | Nonce already used |
| ERR_KEY_DERIVATION_FAILED | Key Derivation Failed | Key derivation process error |
| ERR_ENCRYPTION_FAILED | Encryption Failed | Data encryption error |
| ERR_DECRYPTION_FAILED | Decryption Failed | Data decryption error |
| ERR_INVALID_MNEMONIC | Invalid Mnemonic | Mnemonic checksum failed |
| ERR_RING_SIGNATURE_INVALID | Invalid Ring Signature | Ring signature verification failed |
| ERR_RING_SIZE_INVALID | Invalid Ring Size | Ring size not in valid range |
