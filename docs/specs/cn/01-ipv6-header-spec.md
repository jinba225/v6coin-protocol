# IPv6 扩展报头规范

## 版本信息

- **版本号**: v1.0
- **最后更新**: 2026-02-01
- **协议版本**: 0x0100

## 概述

V6Coin 协议通过 IPv6 扩展报头实现交易的直接传输，无需额外封装层。这种方式利用 IPv6 原生特性，将交易数据嵌入到网络层报头中，实现真正的"地址即钱包"理念。

### 设计目标

1. **原生集成**: V6Coin 交易直接作为 IPv6 报头的一部分，无需额外协议层
2. **高效传输**: 利用 Hop-by-Hop Options Header 的强制处理特性，确保所有节点都能识别和处理交易
3. **兼容性强**: 提供退化封装方案，适配不支持扩展报头的网络环境
4. **QoS 支持**: 利用 IPv6 流量分类字段实现服务质量分级

## 扩展报头类型

### Hop-by-Hop Options Header (0x7F)

V6Coin 使用 **Hop-by-Hop Options Header** 作为扩展报头类型，原因如下：

1. **强制处理**: 根据 RFC 8200，所有中间节点都必须处理 Hop-by-Hop Options，这确保了 V6Coin 交易能被每个节点识别
2. **逐跳传递**: 适合需要在传输路径上被每个节点检查和处理的场景
3. **灵活扩展**: 支持自定义 Option Type，便于未来扩展

### Option Type 定义

```
V6Coin Option Type: 0x7F
- 0x7F: V6Coin 交易数据选项（保留给实验性使用）
```

## 数据结构定义

### IPv6 基本报头结构（40 字节）

```
+-----+-----+-----+-----+-----+-----+-----+-----+
|  0  |  1  |  2  |  3  |  4  |  5  |  6  |  7  |
+-----+-----+-----+-----+-----+-----+-----+-----+
| Version | Traffic Class |           Flow Label          |
+---------+---------------+------------------------------+
|        Payload Length   | Next Header |    Hop Limit     |
+------------------------+------------------+-----------+
|                    Source Address                          |
+                                                               +
|                                                               +
+---------------------------------------------------------------+
|                   Destination Address                         |
+                                                               +
|                                                               +
+---------------------------------------------------------------+
```

**字段说明**:
- `Version`: 4 bit，IPv6 版本号（固定为 6）
- `Traffic Class`: 8 bit，流量分类（DSCP + ECN）
- `Flow Label`: 20 bit，流标签
- `Payload Length`: 16 bit，载荷长度
- `Next Header`: 8 bit，下一个报头类型（Hop-by-Hop Options = 0）
- `Hop Limit`: 8 bit，跳数限制
- `Source Address`: 128 bit，源 IPv6 地址
- `Destination Address`: 128 bit，目的 IPv6 地址

### Hop-by-Hop Options Header 结构

```
+-----+-----+-----+-----+-----+-----+-----+-----+
| Next Header | Hdr Ext Len |       Option 1       |
+-------------+-------------+---------------------+
| Option Type | Opt Data Len |   Option Data       |
+-------------+-------------+---------------------+
|             ... Additional Options ...      |
+----------------------------------------------+
```

**字段说明**:
- `Next Header`: 8 bit，下一个报头类型
- `Hdr Ext Len`: 8 bit，扩展报头长度（以 8 字节为单位，不包括前 8 字节）
- `Option Type`: 8 bit，选项类型（V6Coin = 0x7F）
- `Opt Data Len`: 8 bit，选项数据长度
- `Option Data`: 变长，选项数据

### V6Coin 交易结构

```
+-----+-----+-----+-----+-----+-----+-----+-----+
|  Version   |TxType|         Timestamp         |
+-----+-----+-----+-----------------------------------+
|                  Sender Address (16 bytes)     |
+                                               +
|                                               +
+-----------------------------------------------+
|                 Receiver Address (16 bytes)   |
+                                               +
|                                               +
+-----------------------------------------------+
|                  Amount (8 bytes)             |
+-----------------------------------------------+
|                                               |
|             Signature (32 bytes)             |
|                                               |
|                                               +
+-----------------------------------------------+
|                  Reserved (optional)         |
+-----------------------------------------------+
```

**最小长度**: 83 字节（不包括 Option Type 和 Opt Data Len 字段）

## V6Coin 交易格式详解

### 1. Version（协议版本）

- **长度**: 2 字节
- **编码**: 大端序（Big-Endian）
- **取值**: 0x0100 (v1.0)
- **说明**: V6Coin 协议版本号，用于版本兼容性检查

### 2. TxType（交易类型）

- **长度**: 1 字节
- **取值**:
  - `0x01`: 在线交易（Online Transaction）- 实时确认
  - `0x02`: 离线交易（Offline Transaction）- 延迟确认
  - `0x03`: 资产迁移（Migration）- 跨前缀迁移

### 3. Timestamp（时间戳）

- **长度**: 8 字节
- **编码**: 大端序（Big-Endian）
- **单位**: Unix 时间戳（秒）
- **说明**: 交易创建时间，用于防止重放攻击和交易排序

### 4. SenderAddr（发送方地址）

- **长度**: 16 字节
- **格式**: IPv6 地址（128 bit）
- **说明**: 发送方的 V6Coin 钱包地址，即其 IPv6 地址
- **编码**: 原始二进制格式

### 5. RecvAddr（接收方地址）

- **长度**: 16 字节
- **格式**: IPv6 地址（128 bit）
- **说明**: 接收方的 V6Coin 钱包地址
- **编码**: 原始二进制格式

### 6. Amount（金额）

- **长度**: 8 字节
- **编码**: 大端序（Big-Endian）
- **单位**: nano-V6 (10⁻⁹ V6)
- **范围**: 0 ~ 18,446,744,073,709,551,615 nano-V6
- **说明**: 交易金额，最小单位为 nano-V6

### 7. Signature（签名）

- **长度**: 32 字节
- **算法**: Ed25519
- **说明**: 使用发送方私钥对交易数据（除签名字段外的所有字段）进行签名
- **验证**: 接收方使用发送方公钥验证签名的有效性

### 8. Reserved（保留字段）

- **长度**: 可选，变长
- **说明**: 保留字段，用于未来扩展
- **默认值**: 0（如果存在）

## 数据包封装结构

### 完整数据包结构

```
+-------------------------------------------------------+
|                  IPv6 Header (40 bytes)              |
|  Version(4) | Traffic Class(8) | Flow Label(20)       |
|  Payload Length | Next Header(0) | Hop Limit          |
|  Source Address (16 bytes)                            |
|  Destination Address (16 bytes)                       |
+-------------------------------------------------------+
|            Hop-by-Hop Options Header                  |
|  Next Header | Hdr Ext Len                            |
|  Option Type (0x7F) | Opt Data Len                    |
|  V6Coin Transaction Data (83+ bytes)                  |
+-------------------------------------------------------+
```

### 数据包长度计算

```
最小总长度 = 40 (IPv6 Header) + 8 (Hop-by-Hop Base) + 2 (Option Fields) + 83 (Tx Data) + Padding
           = 133 bytes + Padding (pad to 8-byte boundary)
           = 136 bytes
```

## UDP/ICMP 退化封装

### 适用场景

当网络环境不支持 IPv6 扩展报头时（如防火墙过滤、旧版路由器），V6Coin 自动切换到退化封装模式。

### UDP 封装格式

**端口**: 38901

```
+-------------------------------------------------------+
|                   UDP Header (8 bytes)                |
|  Source Port (16) | Dest Port (16)                   |
|  Length (16)      | Checksum (16)                    |
+-------------------------------------------------------+
|            V6Coin Transaction Data                    |
|  Version | TxType | Timestamp | ...                  |
+-------------------------------------------------------+
```

### ICMP 封装格式

**类型**: 专用 V6Coin ICMP Type（待分配）

```
+-------------------------------------------------------+
|                   ICMP Header (8 bytes)               |
|  Type (8) | Code (8) | Checksum (16)                  |
|  Identifier (16) | Sequence (16)                     |
+-------------------------------------------------------+
|            V6Coin Transaction Data                    |
+-------------------------------------------------------+
```

### 退化策略

1. **自动检测**: 发送方通过 ICMP 错误消息或超时检测网络是否支持扩展报头
2. **失败重试**: 扩展报头发送失败后，自动切换到 UDP 封装
3. **回退提示**: 接收方通过响应报头类型确认实际使用的封装方式

## QoS 支持

### Traffic Class 字段结构

```
+-----+-----+-----+-----+-----+-----+-----+-----+
|  0  |  1  |  2  |  3  |  4  |  5  |  6  |  7  |
+-----+-----+-----+-----+-----+-----+-----+-----+
|           DSCP (6 bits)        | ECN (2 bits)|
+----------------------------------------------+
```

### DSCP 映射（QoS 级别）

| QoS 级别 | DSCP 值 | 用途           | 优先级 |
|---------|---------|----------------|--------|
| 紧急    | 46 (EF) | 系统关键交易   | 最高   |
| 高优先级| 34 (AF41)| 价值迁移       | 高     |
| 正常    | 0 (BE)  | 普通交易       | 中     |
| 低优先级| 8 (CS1) | 批量验证       | 低     |

### ECN 映射（显式拥塞通知）

| ECN 值 | 含义         | 说明                     |
|--------|-------------|--------------------------|
| 00     | Non-ECT     | 不支持 ECN               |
| 01     | ECT(1)      | 支持 ECN，传输中         |
| 10     | ECT(0)      | 支持 ECN，传输中         |
| 11     | CE          | 遭遇拥塞，需降低发送速率 |

### QoS 设置示例

```go
// 设置高优先级 QoS
dscp := uint8(34)  // AF41
ecn := uint8(0)    // ECT(0)
header.TrafficClass = ((dscp & 0x3F) << 2) | (ecn & 0x03)
```

## 兼容性处理

### 网络环境检测

1. **扩展报头支持测试**: 发送带 Hop-by-Hop Options 的测试包
2. **响应分析**: 检测 ICMP 参数问题错误消息
3. **回退机制**: 不支持时自动切换到 UDP 封装

### 退化策略优先级

1. **优先级 1**: IPv6 扩展报头（原生模式）
2. **优先级 2**: UDP 封装（端口 38901）
3. **优先级 3**: ICMP 封装（最后备选）

### 兼容性矩阵

| 网络类型         | 扩展报头 | UDP 封装 | ICMP 封装 |
|------------------|----------|----------|-----------|
| 纯 IPv6 网络     | ✅       | ✅       | ✅        |
| NAT66 网络       | ⚠️       | ✅       | ✅        |
| 防火墙严格模式   | ❌       | ✅       | ✅        |
| IPv4/IPv6 双栈   | ❌       | ✅       | ✅        |

## 代码示例

### Go 示例：创建 V6Coin 扩展报头

```go
package main

import (
    "encoding/binary"
    "net"
    "v6coin-protocol/pkg/ipv6"
)

// 创建 V6Coin 交易
func createV6CoinTransaction(sender, receiver string, amount uint64) *ipv6.V6CoinTransaction {
    return &ipv6.V6CoinTransaction{
        Version:    0x0100,
        TxType:     0x01, // 在线交易
        Timestamp:  uint64(1725148800), // 2024-09-01 00:00:00
        SenderAddr: net.ParseIP(sender),
        RecvAddr:   net.ParseIP(receiver),
        Amount:     amount,
        Signature:  make([]byte, 32), // 需要使用私钥签名
    }
}

// 封装到 IPv6 扩展报头
func encapsulateInIPv6Header(source, dest string, tx *ipv6.V6CoinTransaction) ([]byte, error) {
    // 创建 IPv6 基本报头
    header := &ipv6.IPv6Header{
        Version:       6,
        TrafficClass:  0, // DSCP=0, ECN=0
        FlowLabel:     0,
        NextHeader:    ipv6.NextHeaderHopByHopOptions, // 0
        HopLimit:      64,
        SourceAddr:    net.ParseIP(source),
        DestAddr:      net.ParseIP(dest),
    }

    // 创建 V6Coin 扩展报头
    extHeader, err := ipv6.NewV6CoinExtensionHeader(tx)
    if err != nil {
        return nil, err
    }

    // 计算载荷长度
    hbhData, _ := ipv6.SerializeHopByHopOptions(&extHeader.HopByHop)
    header.PayloadLength = uint16(len(hbhData))

    // 序列化 IPv6 报头
    headerData, err := ipv6.SerializeIPv6Header(header)
    if err != nil {
        return nil, err
    }

    // 组合完整数据包
    packet := append(headerData, hbhData...)
    return packet, nil
}

// 设置 QoS
func setQoS(header *ipv6.IPv6Header, dscp, ecn uint8) {
    header.TrafficClass = ipv6.BuildTrafficClass(dscp, ecn)
}
```

### 数据包结构图

```
发送示例：2001:db8::1 → 2001:db8::2，转账 1000000 nano-V6

0000: 60 00 00 00 00 58 00 40 2001 0db8 0000 0000 0000 0000
0020: 0000 0001 2001 0db8 0000 0000 0000 0000 0000 0002 00
0040: 10 7F 53 01 00 00 00 66 d2 90 00 00 2001 0db8 0000 0000
0060: 0000 0000 0000 0001 2001 0db8 0000 0000 0000 0000 0000
0080: 0002 00 00 00 00 00 0F 42 40 00 00 00 00 00 00 00 00 00
00A0: 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00

解析：
- IPv6 Header: 40 bytes
  - Next Header: 0x00 (Hop-by-Hop Options)
  - Payload Length: 0x0058 (88 bytes)
  - Source: 2001:db8::1
  - Dest: 2001:db8::2

- Hop-by-Hop Options Header:
  - Next Header: 0x00 (None)
  - Hdr Ext Len: 0x10 (16 * 8 = 128 bytes, including first 8 bytes = 136 bytes total)
  - Option Type: 0x7F (V6Coin)
  - Opt Data Len: 0x53 (83 bytes)

- V6Coin Transaction:
  - Version: 0x0100
  - TxType: 0x01 (Online)
  - Timestamp: 0x00000066D2900000 (1725148800)
  - Sender: 2001:db8::1
  - Receiver: 2001:db8::2
  - Amount: 0x00000000000F4240 (1000000 nano-V6)
  - Signature: 0x00... (示例为空签名)
```

## 验证规则

### 交易验证

1. **版本检查**: Version 必须等于 0x0100
2. **交易类型**: TxType 必须在 0x01 ~ 0x03 范围内
3. **地址格式**: SenderAddr 和 RecvAddr 必须为有效的 IPv6 地址（16 字节）
4. **签名验证**: Signature 必须为 32 字节且通过 Ed25519 验证
5. **时间戳**: Timestamp 必须在合理范围内（当前时间 ± 24 小时）

### 扩展报头验证

1. **Option Type**: 必须等于 0x7F
2. **数据长度**: Opt Data Len + 83 必须等于实际数据长度
3. **填充检查**: 扩展报头总长度必须为 8 字节的倍数

## 安全考虑

### 签名保护

- 所有交易必须使用 Ed25519 算法签名
- 签名覆盖除 Signature 字段外的所有交易数据
- 接收方必须验证发送方公钥的有效性

### 重放攻击防护

- 每个交易包含唯一的时间戳
- 节点维护最近 24 小时的交易哈希缓存
- 重复时间戳 + 相同发送方 = 拒绝交易

### 隐私保护

- IPv6 地址即钱包地址，天然匿名
- 可选的审计观察位设计允许部分交易可追踪
- 无需额外身份信息即可完成交易

## 参考文档

- [RFC 8200: IPv6 Specification](https://datatracker.ietf.org/doc/html/rfc8200)
- [RFC 8200: IPv6 Extension Headers](https://datatracker.ietf.org/doc/html/rfc8200#section-4)
- [RFC 3168: The Addition of Explicit Congestion Notification (ECN) to IP](https://datatracker.ietf.org/doc/html/rfc3168)
- [RFC 8085: UDP Usage Guidelines](https://datatracker.ietf.org/doc/html/rfc8085)
- [Ed25519: High-speed high-security signatures](https://ed25519.cr.yp.to/)

## 版本历史

| 版本 | 日期       | 变更说明                     |
|------|------------|------------------------------|
| v1.0 | 2026-02-01 | 初始版本，定义基本数据结构   |

## 附录

### A. 字节序说明

- 所有多字节字段使用大端序（Big-Endian）网络字节序
- 确保在不同架构的平台上数据交换的一致性

### B. 错误码定义

| 错误码 | 含义                       | 处理方式     |
|--------|----------------------------|--------------|
| 0x01   | 版本不匹配                 | 拒绝交易     |
| 0x02   | 交易类型无效               | 拒绝交易     |
| 0x03   | 地址格式错误               | 拒绝交易     |
| 0x04   | 签名验证失败               | 拒绝交易     |
| 0x05   | 时间戳无效                 | 拒绝交易     |
| 0x06   | 扩展报头格式错误           | 丢弃数据包   |

### C. 常量定义

```go
const (
    V6CoinVersion      = 0x0100
    V6CoinOptionType   = 0x7F
    TxTypeOnline       = 0x01
    TxTypeOffline      = 0x02
    TxTypeMigration    = 0x03
    MaxTxDataLen       = 83  // 最小交易数据长度
    SignatureLen       = 32  // Ed25519 签名长度
    IPv6AddrLen        = 16  // IPv6 地址长度
    UDPPort            = 38901  // V6Coin UDP 封装端口
)
```
