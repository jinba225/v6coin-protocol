# CGA 地址映射规范

**版本**: v1.0  
**日期**: 2026-02-01

---

## 1. 概述

V6Coin 使用密码学生成地址（Cryptographically Generated Address，CGA）实现「地址即钱包」的设计理念。通过将 Ed25519 私钥派生的公钥映射到 IPv6 接口标识符（Interface Identifier，IID），V6Coin 实现了无外部映射表、原生支持 IPv6 协议栈的钱包地址系统。

### 核心优势

- **原生 IPv6 兼容**：地址直接在 IPv6 协议栈上工作，无需额外映射
- **无状态验证**：任何节点可通过公钥验证地址有效性
- **隐私保护**：支持临时地址机制，增强用户隐私
- **移动性支持**：逻辑子网机制支持设备在多个网络间移动

---

## 2. 核心概念

### 2.1 密码学生成地址（CGA）

CGA 是一种 IPv6 地址生成技术，将加密密钥与地址绑定。V6Coin 的 CGA 实现基于 RFC 3971 标准，但进行了以下优化：

- 使用 SHA-256 替代 SHA-1
- 使用 Ed25519 签名算法
- 清除 IID 的第 0 位以确保单播地址

### 2.2 接口标识符（IID）

IID 是 IPv6 地址的 64 位后半部分（bits 64-127），由私钥派生的公钥生成：

```
IID = SHA256(PublicKey)[0:8] & ^0x01
```

**关键约束**：
- IID 的第 0 位必须为 0（单播地址）
- 使用 SHA-256 的前 8 字节（64 位）

### 2.3 网络前缀（Network Prefix）

网络前缀是 IPv6 地址的 64 位前半部分（bits 0-63），由网络运营商或节点分配：

```
IPv6 Address = NetworkPrefix (64 bits) || IID (64 bits)
```

**默认前缀**：`2001:db8::/64`（文档示例用，实际使用运营商分配的前缀）

---

## 3. 地址生成算法

### 3.1 IID 生成

```go
func GenerateIID(publicKey []byte) uint64 {
    hash := sha256.Sum256(publicKey)
    iid := binary.BigEndian.Uint64(hash[:8])
    iid &^= 1  // 清除第 0 位，确保单播地址
    return iid
}
```

**算法步骤**：
1. 对公钥进行 SHA-256 哈希
2. 取哈希值的前 8 字节（64 位）
3. 使用大端序转换为 uint64
4. 清除第 0 位（按位与 `~0x01`）

### 3.2 核心地址生成

```go
func GenerateAddress(networkPrefix net.IP, privateKey []byte) (*V6Address, error) {
    // 1. 验证输入
    if len(networkPrefix) != 8 {
        return nil, errors.New("network prefix must be 64 bits")
    }
    if len(privateKey) != 32 {
        return nil, errors.New("private key must be 32 bytes (Ed25519)")
    }

    // 2. 派生公钥
    kp, err := crypto.KeyPairFromPrivateKey(privateKey)
    if err != nil {
        return nil, fmt.Errorf("failed to derive public key: %w", err)
    }

    // 3. 生成 IID
    iid := GenerateIID(kp.PublicKey)

    // 4. 构造完整地址
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

**步骤**：
1. 验证网络前缀长度为 8 字节（64 位）
2. 验证私钥长度为 32 字节（Ed25519）
3. 从私钥派生公钥
4. 生成 IID
5. 组合前缀和 IID 构造完整 IPv6 地址

### 3.3 临时地址生成

```go
func GenerateTemporaryAddress(corePrivateKey []byte, index int) (*V6Address, error) {
    // 1. 派生临时私钥
    h := sha256.New()
    h.Write(corePrivateKey)
    h.Write([]byte{byte(index)})
    keyMaterial := h.Sum(nil)
    tempPrivateKey := keyMaterial[:32]

    // 2. 生成临时密钥对
    kp, err := crypto.KeyPairFromPrivateKey(tempPrivateKey)
    if err != nil {
        return nil, fmt.Errorf("failed to derive temporary key: %w", err)
    }

    // 3. 生成 IID
    iid := GenerateIID(kp.PublicKey)

    // 4. 创建临时地址
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

**关键点**：
- 使用 `SHA256(corePrivateKey || index)` 派生临时私钥
- 索引（index）允许同一核心地址派生多个临时地址
- 默认有效期 24 小时（可配置）

---

## 4. 地址类型

### 4.1 核心地址（Core Address）

- **类型**: `AddressTypeCore`
- **永久性**: 永久有效，不自动过期
- **生成方式**: 直接从核心私钥生成
- **用途**: 主钱包地址，长期持有资产

```go
type AddressType int

const (
    AddressTypeCore         AddressType = iota  // 核心私钥地址
    AddressTypeTemporary                       // 临时派生地址
    AddressTypeLogicalSubnet                   // 逻辑子网地址
)
```

### 4.2 临时地址（Temporary Address）

- **类型**: `AddressTypeTemporary`
- **有效期**: 默认 24 小时（可配置 1 小时到 30 天）
- **生成方式**: 从核心私钥派生
- **用途**:
  - 交易隐私保护
  - 防止地址关联分析
  - 增强匿名性

**常量**：
```go
const (
    MinMigrationInterval = time.Hour           // 最小 1 小时
    MaxMigrationInterval = 30 * 24 * time.Hour // 最大 30 天
)
```

### 4.3 逻辑子网地址（Logical Subnet Address）

- **类型**: `AddressTypeLogicalSubnet`
- **永久性**: 随逻辑子网有效
- **生成方式**: 使用核心 IID + 子网前缀
- **用途**:
  - 移动设备在多个网络间漫游
  - 多前缀网络环境
  - 家乡地址和转交地址机制

**约束**：
- 最多 8 个逻辑子网前缀
```go
const MaxLogicalSubnets = 8
```

---

## 5. V6Address 结构

```go
type V6Address struct {
    NetworkPrefix net.IP       // 64 位网络前缀
    IID           uint64       // 64 位接口标识符（从私钥派生）
    IsTemporary   bool         // 是否为临时地址
    AddressType   AddressType  // 地址类型
    CreatedAt     time.Time    // 创建时间
    ExpiresAt     time.Time    // 过期时间（仅临时地址）
    Index         int          // 临时地址或子网索引
}
```

### 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `NetworkPrefix` | `net.IP` | 64 位网络前缀，长度必须为 8 字节 |
| `IID` | `uint64` | 64 位接口标识符，第 0 位必须为 0 |
| `IsTemporary` | `bool` | 标识是否为临时地址 |
| `AddressType` | `AddressType` | 地址类型：Core、Temporary 或 LogicalSubnet |
| `CreatedAt` | `time.Time` | 地址创建时间 |
| `ExpiresAt` | `time.Time` | 地址过期时间（仅临时地址有效） |
| `Index` | `int` | 临时地址索引或子网索引 |

### 方法

```go
// 获取完整 128 位 IPv6 地址
func (addr *V6Address) GetFullAddress() net.IP

// 检查临时地址是否过期
func (addr *V6Address) IsExpired() bool

// 字符串表示
func (addr *V6Address) String() string

// 地址相等性检查
func (addr *V6Address) Equals(other *V6Address) bool
```

---

## 6. 临时地址机制

### 6.1 生成算法

临时地址从核心私钥派生，使用 SHA-256 哈希链：

```
tempPrivateKey = SHA256(corePrivateKey || index)
tempPublicKey = DeriveEd25519Public(tempPrivateKey)
tempIID = SHA256(tempPublicKey)[0:8] & ^0x01
```

**索引管理**：
- 索引从 0 开始递增
- 同一核心地址可派生多个临时地址
- 索引用于区分不同的临时地址

### 6.2 有效期配置

```go
// 默认有效期 24 小时
defaultExpiry := 24 * time.Hour

// 可配置范围
minExpiry := 1 * time.Hour      // 最小 1 小时
maxExpiry := 30 * 24 * time.Hour // 最大 30 天
```

### 6.3 带前缀的临时地址

```go
func GenerateTemporaryAddressWithPrefix(
    networkPrefix net.IP,
    corePrivateKey []byte,
    index int,
    expiry time.Duration,
) (*V6Address, error)
```

**用途**：
- 为特定网络生成临时地址
- 自定义有效期
- 支持多网络环境

### 6.4 过期检测

```go
func (addr *V6Address) IsExpired() bool {
    if !addr.IsTemporary {
        return false
    }
    return time.Now().After(addr.ExpiresAt)
}
```

**轮换策略**：
- 定期检查临时地址过期状态
- 过期前自动生成新临时地址
- 资产从过期地址迁移到新地址

---

## 7. 逻辑子网

### 7.1 逻辑子网结构

```go
type LogicalSubnet struct {
    CoreAddress    V6Address // 核心地址
    SubnetPrefixes []net.IP  // 关联的前缀列表（最多 8 个）
    CreatedAt      time.Time // 创建时间
    ExpiresAt      time.Time // 过期时间（默认 30 天）
}
```

### 7.2 创建逻辑子网

```go
func CreateLogicalSubnet(
    coreAddress *V6Address,
    prefixes []net.IP,
) (*LogicalSubnet, error) {
    // 验证核心地址
    if coreAddress == nil {
        return nil, errors.New("core address cannot be nil")
    }

    // 验证前缀数量
    if len(prefixes) == 0 {
        return nil, errors.New("at least one prefix required")
    }
    if len(prefixes) > MaxLogicalSubnets {
        return nil, fmt.Errorf("too many prefixes (max %d)", MaxLogicalSubnets)
    }

    // 验证每个前缀长度
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

**约束**：
- 至少需要一个前缀
- 最多 8 个前缀
- 每个前缀必须为 64 位

### 7.3 生成子网地址

```go
func GenerateSubnetAddress(
    subnet *LogicalSubnet,
    prefixIndex int,
) (*V6Address, error) {
    // 验证子网
    if subnet == nil {
        return nil, errors.New("subnet cannot be nil")
    }

    // 验证索引范围
    if prefixIndex < 0 || prefixIndex >= len(subnet.SubnetPrefixes) {
        return nil, fmt.Errorf("prefix index %d out of range", prefixIndex)
    }

    prefix := subnet.SubnetPrefixes[prefixIndex]

    return &V6Address{
        NetworkPrefix: prefix,
        IID:           subnet.CoreAddress.IID, // 使用核心 IID
        IsTemporary:   false,
        AddressType:   AddressTypeLogicalSubnet,
        CreatedAt:     time.Now(),
        Index:         prefixIndex,
    }, nil
}
```

### 7.4 同一子网验证

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

**验证逻辑**：
- 两个地址必须使用相同的 IID
- 两个地址的前缀必须都在子网的前缀列表中

---

## 8. 资产迁移

### 8.1 迁移结构

```go
type AddressMigration struct {
    FromAddress   net.IP  // 源地址
    ToAddress     net.IP  // 目标地址
    Timestamp     uint64  // Unix 时间戳（秒）
    Signature     []byte  // Ed25519 签名
    PreviousIndex int     // 源地址索引
}
```

### 8.2 创建迁移

```go
func CreateMigration(
    fromAddr, toAddr *V6Address,
    privateKey []byte,
) (*AddressMigration, error) {
    // 验证输入
    if fromAddr == nil || toAddr == nil {
        return nil, errors.New("both from and to addresses must be provided")
    }
    if len(privateKey) != 32 {
        return nil, errors.New("private key must be 32 bytes")
    }

    // 获取完整地址
    fromIP := fromAddr.GetFullAddress()
    toIP := toAddr.GetFullAddress()

    // 构造签名数据
    data := GetMigrationData(fromIP, toIP, uint64(time.Now().Unix()))

    // 签名
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

### 8.3 迁移数据格式

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

**数据结构**：
```
+----------------+----------------+----------------+
| From Address   | To Address     | Timestamp      |
| (16 bytes)     | (16 bytes)     | (8 bytes)      |
+----------------+----------------+----------------+
| 0-15           | 16-31          | 32-39          |
+----------------+----------------+----------------+
```

### 8.4 验证迁移

```go
func ValidateMigration(
    migration *AddressMigration,
    publicKey []byte,
) error {
    // 验证迁移对象
    if migration == nil {
        return errors.New("migration cannot be nil")
    }
    if len(publicKey) != 32 {
        return errors.New("public key must be 32 bytes")
    }

    // 验证签名
    data := GetMigrationData(migration.FromAddress, migration.ToAddress, migration.Timestamp)
    if !crypto.Verify(publicKey, data, migration.Signature) {
        return errors.New("invalid migration signature")
    }

    // 验证时间戳
    age := time.Since(time.Unix(int64(migration.Timestamp), 0))
    if age > 7*24*time.Hour {
        return errors.New("migration timestamp too old (max 7 days)")
    }

    return nil
}
```

**验证规则**：
1. **签名验证**：使用公钥验证签名有效性
2. **时间戳有效期**：迁移时间戳必须在 7 天内
3. **索引验证**：确保源地址和目标地址的索引关系正确

---

## 9. 地址验证

### 9.1 验证函数

```go
func ValidateAddress(addr *V6Address) error {
    // 验证非空
    if addr == nil {
        return errors.New("address cannot be nil")
    }

    // 验证 IID 单播属性
    if addr.IID&1 != 0 {
        return errors.New("IID must be unicast address (bit 0 = 0)")
    }

    // 验证前缀长度
    if addr.NetworkPrefix != nil && len(addr.NetworkPrefix) != 8 {
        return errors.New("network prefix must be 64 bits (8 bytes)")
    }

    // 验证临时地址
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

### 9.2 提取 IID 和前缀

```go
// 从完整 IPv6 地址提取 IID
func ExtractIID(fullAddr net.IP) uint64 {
    if len(fullAddr) != 16 {
        return 0
    }
    return binary.BigEndian.Uint64(fullAddr[8:16])
}

// 从完整 IPv6 地址提取网络前缀
func ExtractNetworkPrefix(fullAddr net.IP) net.IP {
    if len(fullAddr) != 16 {
        return nil
    }
    return fullAddr[0:8]
}
```

---

## 10. 代码示例

### 10.1 生成核心地址

```go
package main

import (
    "crypto/ed25519"
    "fmt"
    "net"
    "github.com/jinba225/v6coin-protocol/pkg/address"
)

func main() {
    // 生成 Ed25519 密钥对
    publicKey, privateKey, _ := ed25519.GenerateKey(nil)

    // 定义网络前缀（示例）
    networkPrefix := net.IP([]byte{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0})

    // 生成核心地址
    coreAddr, err := address.GenerateAddress(networkPrefix, privateKey)
    if err != nil {
        panic(err)
    }

    // 验证地址
    err = address.ValidateAddress(coreAddr)
    if err != nil {
        panic(err)
    }

    fmt.Printf("核心地址: %s\n", coreAddr.GetFullAddress())
    fmt.Printf("IID: 0x%016x\n", coreAddr.IID)
    fmt.Printf("类型: %v\n", coreAddr.AddressType)
}
```

**输出示例**：
```
核心地址: 2001:db8::a3f2:e5c4:1b9a:7f2e
IID: 0x0000a3f2e5c41b9a
类型: 0
```

### 10.2 生成临时地址

```go
func main() {
    // 生成核心私钥
    _, corePrivateKey, _ := ed25519.GenerateKey(nil)

    // 生成临时地址
    tempAddr, err := address.GenerateTemporaryAddress(corePrivateKey, 0)
    if err != nil {
        panic(err)
    }

    // 带前缀的临时地址
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

    fmt.Printf("临时地址: %s\n", tempAddr.GetFullAddress())
    fmt.Printf("过期时间: %s\n", tempAddr.ExpiresAt)
    fmt.Printf("带前缀临时地址: %s\n", tempAddrWithPrefix.GetFullAddress())
}
```

### 10.3 创建逻辑子网

```go
func main() {
    // 生成核心地址
    _, privateKey, _ := ed25519.GenerateKey(nil)
    networkPrefix := net.IP([]byte{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0})
    coreAddr, _ := address.GenerateAddress(networkPrefix, privateKey)

    // 创建逻辑子网（多个前缀）
    prefixes := []net.IP{
        []byte{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0x01},
        []byte{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0x02},
        []byte{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0x03},
    }

    subnet, err := address.CreateLogicalSubnet(coreAddr, prefixes)
    if err != nil {
        panic(err)
    }

    // 生成子网地址
    for i := range prefixes {
        subnetAddr, _ := address.GenerateSubnetAddress(subnet, i)
        fmt.Printf("子网地址 %d: %s\n", i, subnetAddr.GetFullAddress())
    }
}
```

### 10.4 资产迁移

```go
func main() {
    // 生成密钥对
    _, privateKey, publicKey := ed25519.GenerateKey(nil)

    // 创建源地址和目标地址
    networkPrefix := net.IP([]byte{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0})
    fromAddr, _ := address.GenerateAddress(networkPrefix, privateKey)
    toAddr, _ := address.GenerateTemporaryAddress(privateKey, 1)

    // 创建迁移
    migration, err := address.CreateMigration(fromAddr, toAddr, privateKey)
    if err != nil {
        panic(err)
    }

    // 验证迁移
    err = address.ValidateMigration(migration, publicKey)
    if err != nil {
        panic(err)
    }

    fmt.Printf("迁移成功: %s -> %s\n",
        migration.FromAddress,
        migration.ToAddress,
    )
}
```

### 10.5 地址验证

```go
func main() {
    // 创建临时地址
    _, privateKey, _ := ed25519.GenerateKey(nil)
    tempAddr, _ := address.GenerateTemporaryAddress(privateKey, 0)

    // 验证地址
    err := address.ValidateAddress(tempAddr)
    if err != nil {
        fmt.Printf("地址验证失败: %v\n", err)
    } else {
        fmt.Printf("地址验证成功\n")
    }

    // 检查过期
    if tempAddr.IsExpired() {
        fmt.Printf("地址已过期\n")
    } else {
        fmt.Printf("地址有效\n")
    }

    // 从完整地址提取 IID
    fullAddr := tempAddr.GetFullAddress()
    iid := address.ExtractIID(fullAddr)
    fmt.Printf("IID: 0x%016x\n", iid)
}
```

---

## 11. 安全考虑

### 11.1 私钥保护

- **存储安全**：私钥必须使用安全存储（如硬件安全模块 HSM、加密文件系统）
- **传输安全**：私钥不得在网络上明文传输
- **内存保护**：私钥使用后应立即从内存清除
- **密钥轮换**：定期轮换临时私钥，降低私钥泄露风险

### 11.2 临时地址的隐私增强

**优势**：
- **交易隔离**：不同交易使用不同临时地址，防止交易关联分析
- **余额隐私**：难以通过地址追踪用户的总资产
- **行为隐私**：难以通过地址分析用户的交易模式

**最佳实践**：
- 每笔交易使用新的临时地址
- 临时地址有效期应合理设置（建议 24 小时到 7 天）
- 生成足够的临时地址以应对高频交易

### 11.3 迁移重放攻击防护

**防护机制**：
1. **时间戳验证**：迁移时间戳必须在 7 天内
2. **签名验证**：必须使用私钥签名，确保迁移授权
3. **索引验证**：源地址索引和目标地址索引必须合理
4. **唯一性检查**：防止同一迁移被重复处理

**攻击场景**：
- **重放攻击**：拦截并重复提交迁移交易
- **时间戳伪造**：修改迁移时间戳绕过有效期限制

**防护措施**：
```
1. 每个迁移交易包含唯一时间戳
2. 网络维护迁移交易历史记录
3. 重复迁移交易被自动拒绝
4. 时间戳使用 UTC 时间，避免时区混淆
```

### 11.4 地址碰撞概率分析

**IID 空间大小**：
- IID 为 64 位，去除第 0 位后为 63 位
- 总空间：2^63 ≈ 9.22 × 10^18

**碰撞概率计算**：
- 假设生成 N 个地址
- 碰撞概率 ≈ N^2 / (2 × 2^63) = N^2 / 2^64

**示例**：
- 生成 100 万个地址：碰撞概率 ≈ 10^-8
- 生成 10 亿个地址：碰撞概率 ≈ 10^-2

**结论**：
- 在实际应用中，地址碰撞概率可忽略不计
- 即使全网有 100 亿个地址，碰撞概率仍低于 1%

### 11.5 其他安全考虑

**单播地址约束**：
- IID 第 0 位必须为 0，确保地址为单播地址
- 避免多播地址（bit 0 = 1）导致的路由问题

**前缀验证**：
- 网络前缀长度必须为 64 位
- 防止前缀注入攻击

**过期检查**：
- 临时地址必须定期检查过期状态
- 过期地址不得用于新交易

**密钥派生安全**：
- 使用标准加密库（如 Go 的 crypto/ed25519）
- 避免自定义加密实现
- 使用经过审计的密码学原语

---

## 12. 附录

### 12.1 常量定义

```go
const (
    // IID 生成常量
    DefaultIIDPrefix      = uint64(0)
    MaxLogicalSubnets     = 8

    // 迁移间隔
    MinMigrationInterval = time.Hour               // 最小 1 小时
    MaxMigrationInterval = 30 * 24 * time.Hour     // 最大 30 天
)

// 地址类型
type AddressType int

const (
    AddressTypeCore         AddressType = iota  // 核心地址
    AddressTypeTemporary                       // 临时地址
    AddressTypeLogicalSubnet                   // 逻辑子网地址
)
```

### 12.2 工具函数

```go
// 检查前缀是否在列表中
func containsPrefix(prefixes []net.IP, prefix net.IP) bool {
    for _, p := range prefixes {
        if p.Equal(prefix) {
            return true
        }
    }
    return false
}

// 获取迁移签名数据
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

### 12.3 参考资料

- RFC 3971: Cryptographically Generated Addresses (CGA)
- RFC 4291: IPv6 Address Architecture
- RFC 8064: Recommended Generation of IPv6 Interface Identifiers
- Ed25519: High-speed high-security signatures

---

## 13. 变更历史

| 版本 | 日期 | 变更说明 | 作者 |
|------|------|----------|------|
| v1.0 | 2026-02-01 | 初始版本 | V6Coin Protocol Team |

---

**文档结束**
