# P2P 网络协议规范

**版本**: v1.0
**日期**: 2026-02-02

---

## 1. 概述

V6Coin 使用去中心化 P2P 网络实现节点间通信。本规范定义了节点发现、消息传递、区块同步、交易广播等核心网络协议。

### 设计目标

1. **原生 IPv6 支持**：充分利用 IPv6 地址空间和特性
2. **高效发现**：使用 Kademlia DHT 快速发现节点
3. **抗 DoS**：多层防护机制防止拒绝服务攻击
4. **NAT 穿透**：支持 NAT66 和双栈网络环境
5. **低延迟**：优化的 Gossip 协议实现快速传播

---

## 2. 节点发现

### 2.1 Kademlia DHT

V6Coin 使用 Kademlia DHT 进行节点发现和路由：

#### 距离度量

使用异或距离（XOR Distance）计算节点间的距离：

```
distance(a, b) = a ⊕ b
```

其中 `a` 和 `b` 是 128 位的节点 ID（通常为 IPv6 地址的哈希）。

#### K-桶结构

每个节点维护 256 个 K-桶，对应 128 位的 ID 空间：

```go
type KBucket struct {
    nodes      []*PeerInfo  // 节点列表
    lastSeen   time.Time    // 最后活跃时间
}

type PeerInfo struct {
    ID         PeerID       // 节点 ID（16 字节）
    Address    net.IP       // IPv6 地址
    Port       uint16       // 端口号
    LastSeen   time.Time    // 最后活跃时间
    Capabilities []Capability // 节点能力
}
```

**参数**：
- `K = 16`：每个 K-桶最多 16 个节点
- `α = 3`：并发查找节点数
- `Concurrency = 3`：并发请求数

#### 节点 ID 生成

节点 ID 使用 IPv6 地址的 SHA-256 哈希：

```
NodeID = SHA256(IPv6Address)[0:16]
```

### 2.2 Bootstrap 节点

V6Coin 维护一组 Bootstrap 节点，用于新节点首次加入网络：

**默认 Bootstrap 节点列表**：

| 节点 ID | IPv6 地址 | 端口 |
|---------|-----------|------|
| 0x0000... | 2001:db8::100 | 38901 |
| 0x0001... | 2001:db8::101 | 38901 |
| 0x0002... | 2001:db8::102 | 38901 |

#### Bootstrap 流程

```go
func Bootstrap(bootstrapPeers []PeerInfo) error {
    for _, peer := range bootstrapPeers {
        // 1. 发送 FIND_NODE 请求，查找自己
        nodes, err := SendFindNodeRequest(peer, localNodeID)

        // 2. 添加发现的节点到 K-桶
        for _, node := range nodes {
            AddToKBucket(node)
        }

        // 3. 递归查找更多节点
        RecursivelyFindNodes(node.ID)
    }
    return nil
}
```

### 2.3 节点查找协议

#### FIND_NODE 消息

```go
type FindNodeMessage struct {
    Version    uint16    // 协议版本
    MessageType MessageType // FIND_NODE = 0x01
    TargetID   PeerID    // 目标节点 ID（16 字节）
    SenderID   PeerID    // 发送方节点 ID
}
```

#### NODES 消息

```go
type NodesMessage struct {
    Version    uint16     // 协议版本
    MessageType MessageType // NODES = 0x02
    Nodes      []PeerInfo // 节点列表（最多 K 个）
    SenderID   PeerID     // 发送方节点 ID
}
```

#### 查找算法

```
1. 从 K-桶中选择距离目标最近的 α 个节点
2. 并发发送 FIND_NODE 请求
3. 等待响应，收集返回的节点
4. 将新节点添加到 K-桶
5. 重复步骤 1-4，直到没有新节点或达到最大迭代次数
```

---

## 3. 握手协议

### 3.1 连接建立

当两个节点建立新的 P2P 连接时，必须完成握手协议：

```go
type Handshake struct {
    Version       uint16         // 协议版本（0x0100）
    NetworkID     uint32         // 网络标识（V6Coin Mainnet = 0x1）
    Timestamp     uint64         // 时间戳（Unix 秒）
    NodeID        PeerID         // 节点 ID（16 字节）
    ListenAddr    string         // 监听地址（IPv6:Port）
    Capabilities  []Capability   // 节点能力列表
    BestHeight    uint64         // 当前区块链高度
    BestBlockHash []byte         // 最新区块哈希（32 字节）
    Nonce         uint64         // 随机数，用于防止重放
    Signature     []byte         // Ed25519 签名（32 字节）
}
```

### 3.2 握手流程

```
节点 A                            节点 B
  |                                 |
  |-- CONNECT ---------------------->|  1. 连接请求
  |<-------------------------------|  2. 接受连接
  |                                 |
  |-- HANDSHAKE ------------------->|  3. 发送握手消息
  |<-------------------------------|  4. 验证握手消息
  |                                 |  5. 返回握手响应
  |<-- HANDSHAKE_ACK --------------|  6. 验证握手响应
  |                                 |
  |=== 连接建立成功 =================|
```

### 3.3 能力协商

节点可以声明支持的功能：

```go
type Capability struct {
    ID      uint8   // 能力 ID
    Version uint8   // 能力版本
    Data    []byte  // 能力特定数据（可选）
}
```

**能力定义**：

| 能力 ID | 名称 | 版本 | 说明 |
|---------|------|------|------|
| 0x01 | FULL_NODE | 1 | 全节点，保存完整区块链 |
| 0x02 | LIGHT_NODE | 1 | 轻节点，仅保存区块头 |
| 0x03 | VALIDATOR | 1 | 验证节点，参与 PoC 共识 |
| 0x04 | RELAY | 1 | 中继节点，仅转发消息 |
| 0x05 | DISCOVERY | 1 | 发现节点，参与 DHT |

### 3.4 握手验证

```go
func ValidateHandshake(hs *Handshake, peerPublicKey []byte) error {
    // 1. 版本检查
    if hs.Version != 0x0100 {
        return ErrInvalidVersion
    }

    // 2. 网络标识检查
    if hs.NetworkID != MainnetNetworkID {
        return ErrInvalidNetworkID
    }

    // 3. 时间戳验证（±2 分钟）
    now := time.Now().Unix()
    if math.Abs(float64(now-int64(hs.Timestamp))) > 120 {
        return ErrInvalidTimestamp
    }

    // 4. 签名验证
    data := GetHandshakeData(hs)
    if !crypto.Verify(peerPublicKey, data, hs.Signature) {
        return ErrInvalidSignature
    }

    return nil
}
```

---

## 4. 消息格式

### 4.1 消息头

所有 P2P 消息共享相同的基础消息头：

```
+-----+-----+-----+-----+-----+-----+-----+-----+
|  Version  |  Type     |   Length           |
+-----+-----+-----+-----+-----+-----+-----+-----+
|  Checksum (4 bytes)                         |
+---------------------------------------------+
|  Payload (Length bytes)                     |
+---------------------------------------------+
```

**字段说明**：
- `Version`: 2 字节，协议版本（0x0100）
- `Type`: 2 字节，消息类型
- `Length`: 4 字节，载荷长度
- `Checksum`: 4 字节，载荷的 CRC32 校验和
- `Payload`: 变长，消息载荷

### 4.2 消息类型

| 消息类型 | 值 | 方向 | 说明 |
|---------|---|------|------|
| HANDSHAKE | 0x01 | 双向 | 握手消息 |
| HANDSHAKE_ACK | 0x02 | 双向 | 握手确认 |
| PING | 0x03 | 双向 | 心跳消息 |
| PONG | 0x04 | 双向 | 心跳响应 |
| FIND_NODE | 0x05 | 请求 | 查找节点 |
| NODES | 0x06 | 响应 | 节点列表 |
| GET_BLOCKS | 0x07 | 请求 | 获取区块 |
| BLOCKS | 0x08 | 响应 | 区块数据 |
| NEW_BLOCK | 0x09 | 广播 | 新区块公告 |
| GET_TRANSACTIONS | 0x0A | 请求 | 获取交易 |
| TRANSACTIONS | 0x0B | 响应 | 交易数据 |
| NEW_TRANSACTION | 0x0C | 广播 | 新交易公告 |
| GET_BLOCK_HASHES | 0x0D | 请求 | 获取区块哈希列表 |
| BLOCK_HASHES | 0x0E | 响应 | 区块哈希列表 |
| ADDR | 0x0F | 广播 | 节点地址广播 |

### 4.3 消息序列化

使用 protobuf 进行消息序列化：

```go
syntax = "proto3";

package v6coin.p2p;

message Message {
    uint32 version = 1;
    uint32 type = 2;
    bytes payload = 3;
}
```

---

## 5. 区块同步

### 5.1 新区块广播

当节点生成或接收到新区块时，立即广播给所有对等节点：

#### NEW_BLOCK 消息

```go
type NewBlockMessage struct {
    Version    uint16    // 协议版本
    Height     uint64    // 区块高度
    BlockHash  []byte    // 区块哈希（32 字节）
    BlockData  *Block    // 区块数据（可选，紧凑格式）
    SenderID   PeerID    // 发送方节点 ID
    Timestamp  uint64    // 时间戳
}
```

#### 广播策略

```go
func BroadcastNewBlock(block *Block) {
    // 1. 验证区块
    if err := ValidateBlock(block); err != nil {
        return
    }

    // 2. 创建 NEW_BLOCK 消息
    msg := &NewBlockMessage{
        Version:   0x0100,
        Height:    block.Header.Height,
        BlockHash: block.Hash(),
        BlockData: block,
        Timestamp: uint64(time.Now().Unix()),
    }

    // 3. 广播给所有对等节点
    for _, peer := range peers {
        peer.SendMessage(msg)
    }
}
```

### 5.2 区块请求

当节点落后于区块链时，请求缺失的区块：

#### GET_BLOCKS 消息

```go
type GetBlocksMessage struct {
    Version    uint16   // 协议版本
    StartHeight uint64  // 起始高度
    EndHeight    uint64  // 结束高度（0 表示请求最新）
    MaxCount    uint32  // 最大返回数量
    SenderID    PeerID   // 发送方节点 ID
}
```

#### BLOCKS 消息

```go
type BlocksMessage struct {
    Version    uint16   // 协议版本
    Blocks     []*Block // 区块列表
    SenderID   PeerID   // 发送方节点 ID
}
```

#### 同步算法

```
1. 比较本地高度和对等节点高度
2. 如果落后，发送 GET_BLOCKS 请求
3. 接收 BLOCKS 响应
4. 验证并按顺序添加到本地链
5. 如果仍有差距，重复步骤 2-4
```

### 5.3 快速同步

使用区块头同步机制加速轻节点的同步：

```go
type GetBlockHashesMessage struct {
    Version    uint16  // 协议版本
    StartHash  []byte  // 起始区块哈希
    MaxCount   uint32  // 最大返回数量
}

type BlockHashesMessage struct {
    Version  uint16   // 协议版本
    Hashes   [][]byte // 区块哈希列表
    StopHash []byte   // 停止哈希
}
```

---

## 6. 交易广播

### 6.1 交易池管理

节点维护交易池（Mempool）：

```go
type TxPool struct {
    pending    map[string]*Transaction  // 待确认交易
    queue      *PriorityQueue           // 优先队列
    maxSize    int                      // 最大容量（10,000）
}
```

### 6.2 新交易广播

#### NEW_TRANSACTION 消息

```go
type NewTransactionMessage struct {
    Version    uint16       // 协议版本
    TxHash     []byte       // 交易哈希（32 字节）
    TxData     *Transaction // 交易数据（紧凑格式）
    SenderID   PeerID       // 发送方节点 ID
}
```

#### Gossip 协议

使用 Gossip 协议实现交易广播：

```go
func BroadcastTransaction(tx *Transaction) {
    // 1. 验证交易
    if err := ValidateTransaction(tx); err != nil {
        return
    }

    // 2. 添加到本地交易池
    txPool.AddTransaction(tx)

    // 3. 选择随机子集的对等节点广播
    peers := SelectRandomPeers(8) // 随机选择 8 个节点

    // 4. 广播消息
    msg := &NewTransactionMessage{
        Version: 0x0100,
        TxHash:  tx.Hash(),
        TxData:  tx,
    }

    for _, peer := range peers {
        if !peer.HasSeenTransaction(tx.Hash()) {
            peer.SendMessage(msg)
        }
    }
}
```

### 6.3 交易请求

当节点需要特定交易时：

```go
type GetTransactionsMessage struct {
    Version  uint16   // 协议版本
    TxHashes [][]byte // 交易哈希列表
    SenderID PeerID   // 发送方节点 ID
}

type TransactionsMessage struct {
    Version    uint16       // 协议版本
    Transactions []*Transaction // 交易列表
    SenderID   PeerID       // 发送方节点 ID
}
```

---

## 7. 连接管理

### 7.1 心跳机制

使用 PING/PONG 消息检测连接活性：

```go
type PingMessage struct {
    Version    uint16  // 协议版本
    Nonce      uint64  // 随机数
    SenderID   PeerID  // 发送方节点 ID
    Timestamp  uint64  // 时间戳
}

type PongMessage struct {
    Version    uint16  // 协议版本
    Nonce      uint64  // 对应 PING 的 Nonce
    SenderID   PeerID  // 发送方节点 ID
    Timestamp  uint64  // 时间戳
}
```

#### 心跳参数

| 参数 | 取值 | 说明 |
|------|------|------|
| PingInterval | 30 秒 | 发送 PING 间隔 |
| PongTimeout | 5 秒 | 等待 PONG 超时 |
| MaxMissedPings | 3 | 最大丢失 PING 次数 |

### 7.2 超时处理

```go
func (p *Peer) CheckTimeout() {
    now := time.Now()

    // 1. 检查 PONG 超时
    if p.lastPingReceived != (time.Time{}) {
        elapsed := now.Sub(p.lastPingReceived)
        if elapsed > time.Duration(PingInterval*3) {
            p.Disconnect(ErrPeerTimeout)
        }
    }

    // 2. 发送定期 PING
    if now.Sub(p.lastPingSent) > time.Duration(PingInterval) {
        p.SendPing()
    }
}
```

### 7.3 重连机制

```go
type ReconnectPolicy struct {
    InitialDelay time.Duration // 初始延迟（5 秒）
    MaxDelay     time.Duration // 最大延迟（5 分钟）
    BackoffFactor float64      // 退避因子（2.0）
    MaxAttempts  int           // 最大尝试次数（10）
}

func (p *Peer) Reconnect() {
    delay := p.reconnectPolicy.InitialDelay

    for i := 0; i < p.reconnectPolicy.MaxAttempts; i++ {
        time.Sleep(delay)

        err := p.Connect()
        if err == nil {
            return // 连接成功
        }

        // 指数退避
        delay = time.Duration(float64(delay) * p.reconnectPolicy.BackoffFactor)
        if delay > p.reconnectPolicy.MaxDelay {
            delay = p.reconnectPolicy.MaxDelay
        }
    }

    log.Error("Failed to reconnect after", p.reconnectPolicy.MaxAttempts, "attempts")
}
```

### 7.4 连接池管理

```go
type PeerManager struct {
    peers        map[PeerID]*Peer  // 活跃对等节点
    maxPeers     int               // 最大对等节点数（128）
    incoming     chan *Peer        // 新连接通道
    disconnectCh chan *Peer        // 断开连接通道
}

func (pm *PeerManager) AddPeer(peer *Peer) error {
    // 1. 检查连接数限制
    if len(pm.peers) >= pm.maxPeers {
        return ErrTooManyPeers
    }

    // 2. 检查重复连接
    if _, exists := pm.peers[peer.ID]; exists {
        return ErrPeerAlreadyConnected
    }

    // 3. 添加到连接池
    pm.peers[peer.ID] = peer

    // 4. 开始心跳
    go peer.Heartbeat()

    return nil
}
```

---

## 8. 安全性

### 8.1 DoS 防护

#### 连接限流

```go
type ConnectionLimiter struct {
    maxConnections    int           // 最大连接数（128）
    maxConnectionsPerIP int         // 单 IP 最大连接数（2）
    connectionAttempts  map[string]int  // 连接尝试计数
    banDuration        time.Duration  // 封禁时长（1 小时）
}

func (cl *ConnectionLimiter) AllowConnection(ip string) bool {
    // 1. 检查是否被封禁
    if cl.isBanned(ip) {
        return false
    }

    // 2. 检查连接数限制
    count := cl.connectionAttempts[ip]
    if count >= cl.maxConnectionsPerIP {
        cl.banIP(ip)
        return false
    }

    // 3. 增加计数
    cl.connectionAttempts[ip] = count + 1
    return true
}
```

#### 消息限流

```go
type MessageRateLimiter struct {
    buckets map[PeerID]*TokenBucket  // 每个节点的令牌桶
    rate    float64                  // 速率（每秒）
    burst   int                      // 突发大小
}

type TokenBucket struct {
    tokens float64
    lastUpdate time.Time
}

func (mrl *MessageRateLimiter) AllowMessage(peerID PeerID) bool {
    bucket := mrl.buckets[peerID]
    now := time.Now()

    // 计算新增令牌
    elapsed := now.Sub(bucket.lastUpdate).Seconds()
    bucket.tokens += elapsed * mrl.rate

    // 限制最大令牌数
    if bucket.tokens > float64(mrl.burst) {
        bucket.tokens = float64(mrl.burst)
    }

    // 消费令牌
    if bucket.tokens >= 1.0 {
        bucket.tokens -= 1.0
        bucket.lastUpdate = now
        return true
    }

    return false
}
```

#### 黑名单机制

```go
type Blacklist struct {
    entries map[string]*BlacklistEntry  // IP 或节点 ID
    mutex   sync.RWMutex
}

type BlacklistEntry struct {
    Reason   string      // 封禁原因
    Expires time.Time    // 过期时间
}

func (bl *Blacklist) Ban(id string, reason string, duration time.Duration) {
    bl.mutex.Lock()
    defer bl.mutex.Unlock()

    bl.entries[id] = &BlacklistEntry{
        Reason:   reason,
        Expires:  time.Now().Add(duration),
    }
}

func (bl *Blacklist) IsBanned(id string) bool {
    bl.mutex.RLock()
    defer bl.mutex.RUnlock()

    entry, exists := bl.entries[id]
    if !exists {
        return false
    }

    // 检查是否过期
    if time.Now().After(entry.Expires) {
        delete(bl.entries, id)
        return false
    }

    return true
}
```

### 8.2 恶意节点检测

#### 行为评分

```go
type PeerScore struct {
    successes  int       // 成功交互次数
    failures  int       // 失败交互次数
    lastUpdate time.Time
}

func (ps *PeerScore) CalculateScore() float64 {
    total := ps.successes + ps.failures
    if total == 0 {
        return 0.5  // 默认评分
    }

    return float64(ps.successes) / float64(total)
}

func (ps *PeerScore) IsMalicious() bool {
    // 评分低于 0.3 认为是恶意节点
    return ps.CalculateScore() < 0.3
}
```

#### 降级处理

```go
func (pm *PeerManager) HandleMaliciousPeer(peer *Peer) {
    // 1. 记录恶意行为
    peer.score.failures++

    // 2. 如果评分过低，断开连接
    if peer.score.IsMalicious() {
        pm.DisconnectPeer(peer.ID)
        pm.blacklist.Ban(peer.ID.String(), "Malicious behavior", 24*time.Hour)
    }

    // 3. 降低优先级
    peer.priority = LowPriority
}
```

### 8.3 加密通信

虽然 IPv6 原生不加密，但节点间通信可以使用 TLS：

```go
func CreateTLSConnection() (*tls.Conn, error) {
    // 生成自签名证书
    cert, err := generateSelfSignedCert()
    if err != nil {
        return nil, err
    }

    // 配置 TLS
    config := &tls.Config{
        Certificates:       []tls.Certificate{cert},
        InsecureSkipVerify: true,  // P2P 网络中可能不需要验证
        MinVersion:         tls.VersionTLS12,
        CipherSuites: []uint16{
            tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
            tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
        },
    }

    // 创建 TLS 连接
    conn, err := tls.Dial("tcp", peerAddr, config)
    if err != nil {
        return nil, err
    }

    return conn, nil
}
```

---

## 9. NAT 穿透

虽然 IPv6 不需要 NAT，但为兼容性提供 NAT66 和双栈环境的穿透方案。

### 9.1 STUN 协议

使用 STUN 协议发现公网地址：

```go
type STUNRequest struct {
    MessageClass STUNClass  // 请求
    Method       STUNMethod // 绑定请求
    TransactionID []byte    // 事务 ID（12 字节）
}

type STUNResponse struct {
    MessageClass STUNClass  // 成功响应
    Method       STUNMethod // 绑定响应
    Attributes   []STUNAttribute
}

type STUNAttribute struct {
    Type   uint16
    Length uint16
    Value  []byte
}
```

### 9.2 UPnP/PCP

使用 UPnP 或 PCP 协议进行端口映射：

```go
func MapPort(port uint16) error {
    // UPnP 设备发现
    devices, err := upnp.Discover()
    if err != nil {
        return err
    }

    // 端口映射
    for _, device := range devices {
        err = device.AddPortMapping("TCP", port, port, "V6Coin P2P", 3600)
        if err != nil {
            return err
        }
    }

    return nil
}
```

### 9.3 中继连接

对于无法直接连接的节点，使用中继节点：

```go
type RelayRequest struct {
    Version   uint16
    TargetID  PeerID
    SenderID  PeerID
}

func (p *Peer) RelayTo(target PeerID) error {
    req := &RelayRequest{
        Version:  0x0100,
        TargetID: target,
        SenderID: p.ID,
    }

    return p.SendMessage(req)
}
```

---

## 10. 实现示例

### 10.1 创建 P2P 节点

```go
package p2p

import (
    "crypto/ed25519"
    "net"
    "time"
)

// P2P 节点结构
type Node struct {
    ID           PeerID
    ListenAddr   net.Addr
    PublicKey    ed25519.PublicKey
    PrivateKey   ed25519.PrivateKey
    PeerManager  *PeerManager
    KBuckets     [256]*KBucket
    TxPool       *TxPool
    Blacklist    *Blacklist
    Config       *Config
}

// 配置
type Config struct {
    ListenAddr       string        // 监听地址
    MaxPeers         int           // 最大对等节点数
    PingInterval     time.Duration // 心跳间隔
    BootstrapPeers   []PeerInfo    // Bootstrap 节点
    EnableUPnP       bool          // 是否启用 UPnP
}

// 创建新节点
func NewNode(privateKey ed25519.PrivateKey, config *Config) (*Node, error) {
    publicKey := privateKey.Public()

    // 生成节点 ID
    nodeID := GenerateNodeID(publicKey)

    return &Node{
        ID:          nodeID,
        PublicKey:   publicKey,
        PrivateKey:  privateKey,
        PeerManager: NewPeerManager(config.MaxPeers),
        KBuckets:    NewKBuckets(),
        TxPool:      NewTxPool(),
        Blacklist:   NewBlacklist(),
        Config:      config,
    }, nil
}

// 生成节点 ID
func GenerateNodeID(publicKey ed25519.PublicKey) PeerID {
    hash := sha256.Sum256(publicKey)
    return hash[:16]
}
```

### 10.2 启动 P2P 节点

```go
func (n *Node) Start() error {
    // 1. 创建监听器
    listener, err := net.Listen("tcp", n.Config.ListenAddr)
    if err != nil {
        return err
    }
    n.ListenAddr = listener.Addr()

    // 2. UPnP 端口映射
    if n.Config.EnableUPnP {
        go n.MapPort()
    }

    // 3. 连接 Bootstrap 节点
    go n.Bootstrap()

    // 4. 开始接受连接
    go n.AcceptConnections()

    // 5. 启动定期任务
    go n.StartPeriodicTasks()

    log.Info("P2P node started:", n.ListenAddr.String())
    return nil
}

func (n *Node) AcceptConnections() {
    for {
        conn, err := listener.Accept()
        if err != nil {
            log.Error("Accept error:", err)
            continue
        }

        // 处理新连接
        go n.HandleNewConnection(conn)
    }
}
```

### 10.3 处理新连接

```go
func (n *Node) HandleNewConnection(conn net.Conn) {
    // 1. 读取握手消息
    var hs Handshake
    decoder := gob.NewDecoder(conn)
    if err := decoder.Decode(&hs); err != nil {
        conn.Close()
        return
    }

    // 2. 验证握手
    if err := ValidateHandshake(&hs, hs.NodeID); err != nil {
        log.Error("Invalid handshake:", err)
        conn.Close()
        return
    }

    // 3. 检查黑名单
    if n.Blacklist.IsBanned(hs.NodeID.String()) {
        log.Error("Peer is blacklisted:", hs.NodeID)
        conn.Close()
        return
    }

    // 4. 创建对等节点
    peer := NewPeer(conn, hs.NodeID, hs.ListenAddr)

    // 5. 发送握手响应
    ack := n.CreateHandshakeAck()
    encoder := gob.NewEncoder(conn)
    if err := encoder.Encode(&ack); err != nil {
        conn.Close()
        return
    }

    // 6. 添加到对等节点管理器
    if err := n.PeerManager.AddPeer(peer); err != nil {
        log.Error("Failed to add peer:", err)
        conn.Close()
        return
    }

    // 7. 开始消息处理
    go peer.StartMessageHandler()
    go peer.Heartbeat()

    log.Info("New peer connected:", peer.ID)
}
```

### 10.4 消息处理

```go
func (p *Peer) StartMessageHandler() {
    decoder := gob.NewDecoder(p.Conn)

    for {
        var msg Message
        if err := decoder.Decode(&msg); err != nil {
            log.Error("Decode error:", err)
            break
        }

        // 处理消息
        if err := p.HandleMessage(&msg); err != nil {
            log.Error("Handle message error:", err)
            break
        }
    }

    p.Disconnect(ErrConnectionClosed)
}

func (p *Peer) HandleMessage(msg *Message) error {
    switch msg.Type {
    case PING:
        return p.handlePing(msg)
    case PONG:
        return p.handlePong(msg)
    case FIND_NODE:
        return p.handleFindNode(msg)
    case NODES:
        return p.handleNodes(msg)
    case NEW_BLOCK:
        return p.handleNewBlock(msg)
    case NEW_TRANSACTION:
        return p.handleNewTransaction(msg)
    default:
        return ErrUnknownMessageType
    }
}
```

### 10.5 广播新区块

```go
func (n *Node) BroadcastBlock(block *Block) {
    // 1. 验证区块
    if err := ValidateBlock(block); err != nil {
        return
    }

    // 2. 创建 NEW_BLOCK 消息
    msg := &NewBlockMessage{
        Version:   0x0100,
        Height:    block.Header.Height,
        BlockHash: block.Hash(),
        BlockData: block,
        Timestamp: uint64(time.Now().Unix()),
    }

    // 3. 序列化消息
    data, err := SerializeMessage(msg)
    if err != nil {
        return
    }

    // 4. 广播给所有对等节点
    for _, peer := range n.PeerManager.peers {
        peer.SendMessage(data)
    }

    log.Info("Broadcasted block:", block.Header.Height)
}
```

### 10.6 同步区块链

```go
func (n *Node) SyncBlockchain() error {
    // 1. 获取本地高度
    localHeight := n.GetLocalHeight()

    // 2. 从对等节点获取最大高度
    maxHeight := n.GetMaxPeerHeight()

    // 3. 如果落后，开始同步
    if localHeight < maxHeight {
        return n.DownloadBlocks(localHeight + 1, maxHeight)
    }

    return nil
}

func (n *Node) DownloadBlocks(start, end uint64) error {
    batchSize := uint32(100)  // 每批下载 100 个区块

    for start <= end {
        // 1. 选择最好的对等节点
        peer := n.SelectBestPeer()

        // 2. 请求区块
        req := &GetBlocksMessage{
            Version:     0x0100,
            StartHeight: start,
            EndHeight:   min(start+uint64(batchSize)-1, end),
            MaxCount:    batchSize,
        }

        // 3. 发送请求
        resp, err := peer.RequestBlocks(req)
        if err != nil {
            log.Error("Request blocks error:", err)
            continue
        }

        // 4. 验证并添加区块
        for _, block := range resp.Blocks {
            if err := ValidateBlock(block); err != nil {
                return err
            }

            if err := n.AddBlock(block); err != nil {
                return err
            }
        }

        start += uint64(batchSize)
    }

    return nil
}
```

---

## 11. 参数配置

### 11.1 网络参数

| 参数 | 取值 | 说明 |
|------|------|------|
| DefaultPort | 38901 | 默认 P2P 端口 |
| ProtocolVersion | 0x0100 | 协议版本 |
| NetworkID | 0x1 | 主网标识 |
| MaxPeers | 128 | 最大对等节点数 |
| MinPeers | 16 | 最小对等节点数 |
| K | 16 | K-桶大小 |
| Alpha | 3 | 并发查找数 |

### 11.2 超时参数

| 参数 | 取值 | 说明 |
|------|------|------|
| PingInterval | 30 秒 | 心跳间隔 |
| PongTimeout | 5 秒 | 等待 PONG 超时 |
| ConnectionTimeout | 10 秒 | 连接超时 |
| HandshakeTimeout | 5 秒 | 握手超时 |
| RequestTimeout | 30 秒 | 请求超时 |
| BlockRequestTimeout | 60 秒 | 区块请求超时 |

### 11.3 限流参数

| 参数 | 取值 | 说明 |
|------|------|------|
| MaxMessageSize | 10 MB | 最大消息大小 |
| MaxMessagesPerSecond | 100 | 每秒最大消息数 |
| MaxConnectionsPerIP | 2 | 单 IP 最大连接数 |
| BanDuration | 1 小时 | 封禁时长 |
| MaxBanDuration | 24 小时 | 最大封禁时长 |

### 11.4 Bootstrap 节点

```go
var DefaultBootstrapPeers = []string{
    "2001:db8::100:38901",
    "2001:db8::101:38901",
    "2001:db8::102:38901",
    "2001:db8::103:38901",
    "2001:db8::104:38901",
}
```

---

## 12. 状态机图

### 12.1 连接状态机

```
                        [DISCONNECTED]
                              |
                              | 连接请求
                              v
                        [CONNECTING]
                              |
                              | 连接成功
                              v
                        [HANDSHAKING]
                              |
                              | 握手成功
                              v
                        [CONNECTED]
                              |
                              | 活跃心跳
                              v
                        [ACTIVE]
                              |
                              | 心跳超时/错误
                              v
                        [DISCONNECTING]
                              |
                              v
                        [DISCONNECTED]
```

### 12.2 节点发现状态机

```
                        [INITIAL]
                              |
                              | 启动
                              v
                        [BOOTSTRAPPING]
                              |
                              | 获取种子节点
                              v
                        [DISCOVERING]
                              |
                              | 查找节点
                              v
                        [MAINTAINING]
                              |
                              | 定期刷新 K-桶
                              v
                        [ACTIVE]
```

---

## 13. 版本历史

| 版本 | 日期 | 变更说明 |
|------|------|----------|
| v1.0 | 2026-02-02 | 初始版本，定义 P2P 网络协议 |

---

## 14. 参考文档

- [RFC 6980](https://datatracker.ietf.org/doc/html/rfc6980) - IPv6 Hop-by-Hop Options Header
- [Kademlia: A Peer-to-peer Information System Based on the XOR Metric](https://pdos.csail.mit.edu/~petar/papers/maymounkov-kademlia-lncs.pdf)
- [RFC 5389](https://datatracker.ietf.org/doc/html/rfc5389) - STUN Protocol
- [V6Coin 白皮书](../whitepaper/V6Coin_Whitepaper_CN.md)
- [IPv6 扩展报头规范](./01-ipv6-header-spec.md)
- [CGA 地址映射规范](./02-cga-address-spec.md)
- [PoC 共识规范](./03-poc-consensus-spec.md)
- [交易验证规范](./04-transaction-spec.md)

---

**文档结束**
