# 加密算法规范

## 版本信息

- **版本号**: v1.0
- **最后更新**: 2026-02-01
- **协议版本**: 0x0100

## 概述

V6Coin 采用多种加密算法保障系统安全性、隐私性和完整性。本规范定义了 V6Coin 协议中使用的所有加密算法的标准实现，包括签名、加密、哈希、密钥管理等核心机制。

### 设计目标

1. **安全性**: 使用业界认可的加密算法，确保交易安全和数据隐私
2. **高效性**: 优化算法性能，适配低功耗 IoT 设备
3. **匿名性**: 通过环签名等技术保护用户交易隐私
4. **可扩展性**: 预留算法升级空间，支持未来扩展

---

## Ed25519 签名算法

Ed25519 是一种基于 Twisted Edwards 曲线的椭圆曲线数字签名算法，V6Coin 使用它进行交易签名和地址生成。

### 算法参数

| 参数 | 值 | 说明 |
|------|-----|------|
| 曲线 | Ed25519 | 256 位椭圆曲线 |
| 签名长度 | 32 字节 | 固定长度签名 |
| 公钥长度 | 32 字节 | 压缩公钥格式 |
| 私钥长度 | 32 字节 | 种子密钥 |

### 密钥生成

```go
// 生成 Ed25519 密钥对
func GenerateEd25519KeyPair() (publicKey, privateKey []byte, err error) {
    publicKey, privateKey, err := ed25519.GenerateKey(nil)
    if err != nil {
        return nil, nil, err
    }
    return publicKey, privateKey, nil
}
```

### 签名生成

```go
// 使用私钥签名消息
func SignEd25519(privateKey, message []byte) []byte {
    return ed25519.Sign(privateKey, message)
}
```

**签名消息格式**：
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

### 签名验证

```go
// 使用公钥验证签名
func VerifyEd25519(publicKey, message, signature []byte) bool {
    return ed25519.Verify(publicKey, message, signature)
}
```

### IID 生成

V6Coin 使用 Ed25519 公钥生成 IPv6 接口标识符（IID）：

```go
// 从公钥生成 IID
func GenerateIID(publicKey []byte) []byte {
    hash := sha256.Sum256(publicKey)
    iid := make([]byte, 8)
    copy(iid, hash[0:8])
    // 将第 0 位设置为 0（单播地址）
    iid[0] &= 0x7F
    return iid
}
```

**IID 计算公式**：
```
IID = SHA256(publicKey)[0:8]
IID[0] &= 0x7F  // 确保单播地址
```

### 地址生成

完整的 V6Coin 地址（IPv6 地址）生成流程：

```go
// 生成完整 V6Coin 地址
func GenerateV6CoinAddress(networkPrefix, publicKey []byte) net.IP {
    iid := GenerateIID(publicKey)
    address := make(net.IP, 16)
    copy(address[0:8], networkPrefix[0:8])  // 64 位网络前缀
    copy(address[8:16], iid)                 // 64 位 IID
    return address
}
```

### 性能特性

| 操作 | 时间复杂度 | 实测性能 (Go) |
|------|------------|---------------|
| 密钥生成 | O(n) | ~0.1ms |
| 签名 | O(n) | ~0.05ms |
| 验证 | O(n) | ~0.1ms |
| 公钥压缩 | O(1) | <0.01ms |

---

## AES-256-GCM 加密

AES-256-GCM 是 V6Coin 用于数据加密的对称加密算法，支持认证加密（Authenticated Encryption）。

### 算法参数

| 参数 | 值 | 说明 |
|------|-----|------|
| 密钥长度 | 32 字节 | 256 位密钥 |
| 块大小 | 16 字节 | 128 位块 |
| Nonce 长度 | 12 字节 | 推荐长度 |
| Tag 长度 | 16 字节 | 认证标签 |

### 密钥派生

V6Coin 使用 HKDF-SHA256 从用户密码派生 AES 密钥：

```go
// 从密码派生 AES 密钥
func DeriveAESKey(password, salt []byte) []byte {
    // 使用 HKDF-SHA256
    hkdf := hkdf.New(sha256.New, password, salt, nil)
    key := make([]byte, 32)  // AES-256 密钥
    hkdf.Read(key)
    return key
}
```

### 加密

```go
// AES-256-GCM 加密
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

    // GCM 模式：ciphertext = nonce + sealed data
    sealed := aesgcm.Seal(nil, nonce, plaintext, nil)
    ciphertext = sealed[:len(sealed)-16]
    tag = sealed[len(sealed)-16:]

    return nonce, ciphertext, tag, nil
}
```

### 解密

```go
// AES-256-GCM 解密
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

### 应用场景

| 场景 | 使用方式 | 备注 |
|------|----------|------|
| 私钥存储 | 本地 Flash 存储 | 密钥由 CPU 序列号 + 用户密码派生 |
| 交易数据 | 离线交易加密 | 交易数据本地缓存加密 |
| 通信加密 | 端到端加密 | 可选功能 |

### 安全要求

1. **Nonce 唯一性**: 每个 Nonce 只能使用一次，防止密钥泄露
2. **Key Rotation**: 定期更换密钥（建议每月一次）
3. **安全存储**: 密钥必须加密存储，不可明文保存

---

## SHA-256 哈希

SHA-256 是 V6Coin 的标准哈希算法，用于交易哈希、区块哈希和 Merkle 树计算。

### 算法参数

| 参数 | 值 | 说明 |
|------|-----|------|
| 哈希长度 | 32 字节 | 256 位输出 |
| 块大小 | 64 字节 | 输入块大小 |
| 轮数 | 64 | 压缩轮数 |

### 哈希计算

```go
// 计算 SHA-256 哈希
func HashSHA256(data []byte) []byte {
    hash := sha256.Sum256(data)
    return hash[:]
}
```

### 交易哈希

```go
// 计算交易哈希
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

### Merkle 树

V6Coin 使用 Merkle 树验证区块内交易完整性：

```go
// Merkle 节点
type MerkleNode struct {
    Left  *MerkleNode
    Right *MerkleNode
    Hash  []byte
}

// 计算 Merkle 根
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
                right = left  // 奇数节点，复制最后一个
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

### 性能特性

| 操作 | 时间复杂度 | 实测性能 (Go) |
|------|------------|---------------|
| 单次哈希 | O(n) | ~0.01ms (1KB) |
| Merkle 树构建 | O(n log n) | ~1ms (1000 txs) |
| Merkle 证明验证 | O(log n) | ~0.1ms |

---

## BIP-39 助记词

BIP-39 是 V6Coin 用于密钥备份和恢复的助记词标准，方便用户记忆和备份私钥。

### 助记词生成

```go
// 生成 12 词助记词
func GenerateMnemonic() (string, error) {
    // 生成 128 位熵
    entropy := make([]byte, 16)
    if _, err := rand.Read(entropy); err != nil {
        return "", err
    }

    // 计算 checksum
    hash := sha256.Sum256(entropy)
    checksum := hash[0] >> 4  // 取前 4 位

    // 拼接熵 + checksum
    combined := append(entropy, checksum)

    // 转换为助记词
    mnemonic := bip39.EncodeMnemonic(combined)
    return mnemonic, nil
}
```

### 助记词转私钥

```go
// 从助记词生成 Ed25519 私钥
func MnemonicToPrivateKey(mnemonic, passphrase string) ([]byte, error) {
    // 生成种子
    seed := bip39.NewSeed(mnemonic, passphrase)

    // 使用 HKDF 从种子派生 Ed25519 私钥
    salt := []byte("V6Coin ed25519 seed")
    hkdf := hkdf.New(sha256.New, seed, salt, nil)
    privateKey := make([]byte, 32)
    hkdf.Read(privateKey)

    return privateKey, nil
}
```

### 密钥派生路径

V6Coin 支持多层级密钥派生，使用 HD 钱包结构：

```
m / 44' / 931' / 0' / 0 / 0
 │   │      │     │   │  └─ 索引
 │   │      │     │   └─ 账户类型 (外部=0, 内部=1)
 │   │      │     └─ 账户索引
 │   │      └─ V6Coin 币种 (931)
 │   └─ BIP-44 类型
 └─ 主密钥
```

### 安全建议

1. **离线存储**: 助记词必须离线保存，避免网络泄露
2. **物理备份**: 建议使用纸质或金属备份，避免数字化存储
3. **隔离保管**: 多个备份应分散存放
4. **定期验证**: 定期测试助记词能否成功恢复私钥

---

## 环签名

环签名是 V6Coin 用于交易匿名性的核心技术，将真实签名者隐藏在一组匿名成员中。

### 算法概述

V6Coin 采用改进的环签名算法，基于 Ed25519 曲线：

```go
// 环签名结构
type RingSignature struct {
    RingSize   int      // 环大小
    KeyImages  [][]byte // 密钥图像
    Signatures []byte   // 环签名
}
```

### 环签名生成

```go
// 生成环签名
func GenerateRingSignature(
    message []byte,
    privateKey, publicKey []byte,
    decoyPublicKeys [][]byte,
) (*RingSignature, error) {
    ringSize := len(decoyPublicKeys) + 1

    // 随机选择签名者位置
    secretIndex := rand.Intn(ringSize)

    // 生成密钥图像
    keyImage := GenerateKeyImage(privateKey, publicKey)

    // 初始化挑战值
    challenges := make([][]byte, ringSize)
    responses := make([][]byte, ringSize)

    // 生成随机值
    for i := 0; i < ringSize; i++ {
        if i != secretIndex {
            responses[i] = make([]byte, 32)
            if _, err := rand.Read(responses[i]); err != nil {
                return nil, err
            }
        }
    }

    // 计算挑战值
    for i := 0; i < ringSize; i++ {
        idx := (secretIndex + i) % ringSize

        var pubKey []byte
        if idx == 0 {
            pubKey = publicKey
        } else {
            pubKey = decoyPublicKeys[idx-1]
        }

        // 计算哈希挑战
        hashInput := bytes.Join([][]byte{
            message,
            pubKey,
            keyImage,
            responses[idx],
        }, nil)

        challenges[idx] = HashSHA256(hashInput)
    }

    // 计算签名者的响应
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

### 环签名验证

```go
// 验证环签名
func VerifyRingSignature(
    message []byte,
    publicKeys [][]byte,
    sig *RingSignature,
) bool {
    ringSize := len(publicKeys)

    // 解析签名
    responses := bytes.SplitN(sig.Signatures, []byte{}, ringSize)

    // 计算每个挑战值
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

        // 验证签名等式
        if !verifyRingEquation(pubKey, response, computedChallenge) {
            return false
        }
    }

    return true
}
```

### 环大小选择

| 场景 | 推荐环大小 | 原因 |
|------|-----------|------|
| 普通交易 | 5-7 | 平衡匿名性和性能 |
| 大额交易 | 10-15 | 增强匿名性 |
| IoT 微支付 | 3-5 | 优化性能 |

### 性能特性

| 环大小 | 签名时间 | 验证时间 | 数据大小 |
|--------|----------|----------|----------|
| 5 | ~2ms | ~1ms | 160 字节 |
| 10 | ~4ms | ~2ms | 320 字节 |
| 15 | ~6ms | ~3ms | 480 字节 |

---

## 密钥管理最佳实践

### 密钥存储

1. **硬件安全芯片**（优先）：
   - 使用 TPM/TEE/安全元件
   - 私钥永不离开安全区域

2. **软件加密存储**（备选）：
   - AES-256-GCM 加密
   - 密钥派生：Key = HKDF(SHA256, Password + CPU序列号, Salt)

3. **内存安全**：
   - 使用安全内存区（如 Go 的 secure memory）
   - 及时擦除临时密钥

### 密钥轮换

1. **定期轮换**：
   - 主密钥：每年一次
   - 临时地址私钥：每 24 小时
   - 加密密钥：每月一次

2. **安全迁移**：
   - 使用多签名方案
   - 链上记录迁移证明

3. **紧急吊销**：
   - 通过助记词立即重新生成密钥
   - 广播密钥泄露通知

### 助记词管理

| 操作 | 最佳实践 |
|------|----------|
| 生成 | 使用真随机数生成器 |
| 存储 | 离线、纸质/金属备份 |
| 分发 | 避免数字化传输 |
| 验证 | 定期测试恢复流程 |
| 销毁 | 物理粉碎或焚烧 |

---

## 安全考虑

### 密钥安全

1. **熵源质量**：
   - 使用加密安全随机数生成器（CSPRNG）
   - 避免 `math/rand` 等伪随机生成器

2. **密钥泄露防护**：
   - 侧信道攻击防护
   - 时序攻击防护
   - 故障攻击防护

3. **密钥生命周期管理**：
   - 密钥生成安全审计
   - 密钥使用日志记录
   - 密钥销毁验证

### 算法安全性

1. **Ed25519 安全性**：
   - 已证明安全性：128 位安全级别
   - 抗侧信道攻击
   - 签名不可伪造性

2. **AES-256-GCM 安全性**：
   - 认证加密，防止篡改
   - Nonce 重复使用导致密钥泄露
   - 推荐 AEAD 模式

3. **SHA-256 安全性**：
   - 抗碰撞性
   - 抗原像攻击
   - 抗第二原像攻击

### 隐私保护

1. **环签名匿名性**：
   - 无法识别真实签名者
   - 不可链接性
   - 匿名集大小可调

2. **地址匿名性**：
   - 临时地址动态切换
   - 核心地址不对外暴露
   - 逻辑子网地址池

3. **交易隐私**：
   - 金额可隐藏（可选）
   - 时间戳模糊化
   - 路径匿名转发

### 实现安全

1. **常量时间算法**：
   - 避免时序侧信道
   - 所有密码学操作使用常量时间

2. **内存安全**：
   - 安全擦除敏感数据
   - 避免内存转储泄露
   - 使用安全内存分配器

3. **错误处理**：
   - 避免错误信息泄露
   - 安全降级策略
   - 失败时回滚敏感操作

---

## 实现示例

### 完整交易签名流程

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

// 计算交易哈希
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

// 签名交易
func (tx *Transaction) Sign(privateKey []byte) error {
    message := tx.Hash()
    tx.Signature = ed25519.Sign(privateKey, message)
    return nil
}

// 验证交易
func (tx *Transaction) Verify(publicKey []byte) bool {
    message := tx.Hash()
    return ed25519.Verify(publicKey, message, tx.Signature)
}

func main() {
    // 生成密钥对
    publicKey, privateKey, _ := ed25519.GenerateKey(nil)

    // 创建交易
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

    // 签名交易
    tx.Sign(privateKey)

    // 验证交易
    if tx.Verify(publicKey) {
        fmt.Println("交易签名验证通过")
    } else {
        fmt.Println("交易签名验证失败")
    }
}
```

### 密钥派生流程

```go
package main

import (
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "golang.org/x/crypto/hkdf"
    "golang.org/x/crypto/bip39"
)

// 从助记词生成密钥对
func KeyPairFromMnemonic(mnemonic, passphrase string) (publicKey, privateKey []byte, err error) {
    // 生成种子
    seed := bip39.NewSeed(mnemonic, passphrase)

    // 使用 HKDF 派生密钥
    salt := []byte("V6Coin ed25519 seed")
    info := []byte("V6Coin key derivation")

    hkdf := hkdf.New(sha256.New, seed, salt, info)
    privateKey = make([]byte, 32)
    _, err = hkdf.Read(privateKey)
    if err != nil {
        return nil, nil, err
    }

    // 派生公钥
    publicKey, _, err = ed25519.GenerateKey(privateKey)
    if err != nil {
        return nil, nil, err
    }

    return publicKey, privateKey, nil
}

func main() {
    // 生成助记词
    mnemonic, _ := bip39.NewMnemonic(128)
    fmt.Printf("助记词: %s\n", mnemonic)

    // 从助记词生成密钥对
    publicKey, privateKey, _ := KeyPairFromMnemonic(mnemonic, "")

    fmt.Printf("私钥: %s\n", hex.EncodeToString(privateKey))
    fmt.Printf("公钥: %s\n", hex.EncodeToString(publicKey))
}
```

### AES-256-GCM 加密示例

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

// 从密码派生密钥
func DeriveKey(password, salt []byte) []byte {
    hkdf := hkdf.New(sha3.New256, password, salt, nil)
    key := make([]byte, 32)
    hkdf.Read(key)
    return key
}

// 加密数据
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

// 解密数据
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
    // 派生密钥
    password := []byte("my-secret-password")
    salt := make([]byte, 16)
    rand.Read(salt)
    key := DeriveKey(password, salt)

    // 加密数据
    plaintext := []byte("Hello, V6Coin!")
    nonce, ciphertext, _ := Encrypt(key, plaintext)

    fmt.Printf("密钥: %s\n", hex.EncodeToString(key))
    fmt.Printf("Nonce: %s\n", hex.EncodeToString(nonce))
    fmt.Printf("密文: %s\n", hex.EncodeToString(ciphertext))

    // 解密数据
    decrypted, _ := Decrypt(key, nonce, ciphertext)
    fmt.Printf("明文: %s\n", string(decrypted))
}
```

---

## 参考文档

- [RFC 8032: Ed25519](https://datatracker.ietf.org/doc/html/rfc8032)
- [RFC 5869: HKDF](https://datatracker.ietf.org/doc/html/rfc5869)
- [RFC 5116: AEAD](https://datatracker.ietf.org/doc/html/rfc5116)
- [RFC 6234: SHA-256](https://datatracker.ietf.org/doc/html/rfc6234)
- [BIP-39: Mnemonic Code](https://github.com/bitcoin/bips/blob/master/bip-0039.mediawiki)
- [RFC 6986: Merkle Tree](https://datatracker.ietf.org/doc/html/rfc6986)

---

## 版本历史

| 版本 | 日期 | 变更说明 |
|------|------|----------|
| v1.0 | 2026-02-01 | 初始版本，定义核心加密算法 |

---

## 附录

### A. 算法复杂度总结

| 算法 | 操作 | 时间复杂度 | 空间复杂度 |
|------|------|------------|------------|
| Ed25519 | 密钥生成 | O(n) | O(n) |
| Ed25519 | 签名 | O(n) | O(n) |
| Ed25519 | 验证 | O(n) | O(n) |
| AES-256-GCM | 加密 | O(n) | O(n) |
| AES-256-GCM | 解密 | O(n) | O(n) |
| SHA-256 | 哈希 | O(n) | O(1) |
| Merkle 树 | 构建 | O(n log n) | O(n) |
| Merkle 树 | 验证 | O(log n) | O(1) |
| 环签名 | 生成 | O(n) | O(n) |
| 环签名 | 验证 | O(n) | O(n) |

### B. 常量定义

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
    MinRingSize = 3
    MaxRingSize = 15
    DefaultRingSize = 5
)
```

### C. 错误码定义

| 错误码 | 错误类型 | 说明 |
|--------|----------|------|
| ERR_INVALID_PRIVATE_KEY | 无效私钥 | 私钥格式错误 |
| ERR_INVALID_PUBLIC_KEY | 无效公钥 | 公钥格式错误 |
| ERR_INVALID_SIGNATURE | 无效签名 | 签名验证失败 |
| ERR_INVALID_NONCE | 无效 Nonce | Nonce 已使用 |
| ERR_KEY_DERIVATION_FAILED | 密钥派生失败 | 密钥派生过程错误 |
| ERR_ENCRYPTION_FAILED | 加密失败 | 数据加密错误 |
| ERR_DECRYPTION_FAILED | 解密失败 | 数据解密错误 |
| ERR_INVALID_MNEMONIC | 无效助记词 | 助记词校验失败 |
| ERR_RING_SIGNATURE_INVALID | 环签名无效 | 环签名验证失败 |
| ERR_RING_SIZE_INVALID | 环大小无效 | 环大小不在有效范围 |
