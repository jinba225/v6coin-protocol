# P2P Network Protocol Specification

**Version**: v1.0
**Date**: 2026-02-02

---

## 1. Overview

V6Coin uses a decentralized P2P network for inter-node communication. This specification defines core network protocols including node discovery, message passing, block synchronization, and transaction broadcasting.

### Design Goals

1. **Native IPv6 Support**: Fully leverage IPv6 address space and features
2. **Efficient Discovery**: Use Kademlia DHT for fast node discovery
3. **DoS Resistance**: Multi-layer protection against denial-of-service attacks
4. **NAT Traversal**: Support for NAT66 and dual-stack network environments
5. **Low Latency**: Optimized Gossip protocol for rapid propagation

---

## 2. Node Discovery

### 2.1 Kademlia DHT

V6Coin uses Kademlia DHT for node discovery and routing:

#### Distance Metric

Uses XOR Distance to calculate distance between nodes:

```
distance(a, b) = a ⊕ b
```

Where `a` and `b` are 128-bit node IDs (typically the hash of IPv6 addresses).

#### K-Bucket Structure

Each node maintains 256 k-buckets corresponding to the 128-bit ID space:

```go
type KBucket struct {
    nodes      []*PeerInfo  // Node list
    lastSeen   time.Time    // Last active time
}

type PeerInfo struct {
    ID         PeerID       // Node ID (16 bytes)
    Address    net.IP       // IPv6 address
    Port       uint16       // Port number
    LastSeen   time.Time    // Last active time
    Capabilities []Capability // Node capabilities
}
```

**Parameters**:
- `K = 16`: Maximum 16 nodes per k-bucket
- `α = 3`: Concurrent lookup nodes
- `Concurrency = 3`: Concurrent request count

#### Node ID Generation

Node ID uses SHA-256 hash of IPv6 address:

```
NodeID = SHA256(IPv6Address)[0:16]
```

### 2.2 Bootstrap Nodes

V6Coin maintains a set of bootstrap nodes for new nodes to join the network initially:

**Default Bootstrap Node List**:

| Node ID | IPv6 Address | Port |
|---------|--------------|------|
| 0x0000... | 2001:db8::100 | 38901 |
| 0x0001... | 2001:db8::101 | 38901 |
| 0x0002... | 2001:db8::102 | 38901 |

#### Bootstrap Process

```go
func Bootstrap(bootstrapPeers []PeerInfo) error {
    for _, peer := range bootstrapPeers {
        // 1. Send FIND_NODE request, looking for self
        nodes, err := SendFindNodeRequest(peer, localNodeID)

        // 2. Add discovered nodes to k-buckets
        for _, node := range nodes {
            AddToKBucket(node)
        }

        // 3. Recursively find more nodes
        RecursivelyFindNodes(node.ID)
    }
    return nil
}
```

### 2.3 Node Lookup Protocol

#### FIND_NODE Message

```go
type FindNodeMessage struct {
    Version    uint16    // Protocol version
    MessageType MessageType // FIND_NODE = 0x01
    TargetID   PeerID    // Target node ID (16 bytes)
    SenderID   PeerID    // Sender node ID
}
```

#### NODES Message

```go
type NodesMessage struct {
    Version    uint16     // Protocol version
    MessageType MessageType // NODES = 0x02
    Nodes      []PeerInfo // Node list (max K nodes)
    SenderID   PeerID     // Sender node ID
}
```

#### Lookup Algorithm

```
1. Select α nodes closest to target from k-buckets
2. Send concurrent FIND_NODE requests
3. Wait for responses, collect returned nodes
4. Add new nodes to k-buckets
5. Repeat steps 1-4 until no new nodes or max iterations reached
```

---

## 3. Handshake Protocol

### 3.1 Connection Establishment

When two nodes establish a new P2P connection, they must complete the handshake protocol:

```go
type Handshake struct {
    Version       uint16         // Protocol version (0x0100)
    NetworkID     uint32         // Network ID (V6Coin Mainnet = 0x1)
    Timestamp     uint64         // Timestamp (Unix seconds)
    NodeID        PeerID         // Node ID (16 bytes)
    ListenAddr    string         // Listen address (IPv6:Port)
    Capabilities  []Capability   // Node capability list
    BestHeight    uint64         // Current blockchain height
    BestBlockHash []byte         // Latest block hash (32 bytes)
    Nonce         uint64         // Random number for replay protection
    Signature     []byte         // Ed25519 signature (32 bytes)
}
```

### 3.2 Handshake Flow

```
Node A                            Node B
  |                                 |
  |-- CONNECT ---------------------->|  1. Connection request
  |<-------------------------------|  2. Accept connection
  |                                 |
  |-- HANDSHAKE ------------------->|  3. Send handshake message
  |<-------------------------------|  4. Verify handshake message
  |                                 |  5. Return handshake response
  |<-- HANDSHAKE_ACK --------------|  6. Verify handshake response
  |                                 |
  |=== Connection Established ======|
```

### 3.3 Capability Negotiation

Nodes can declare their supported capabilities:

```go
type Capability struct {
    ID      uint8   // Capability ID
    Version uint8   // Capability version
    Data    []byte  // Capability-specific data (optional)
}
```

**Capability Definitions**:

| Capability ID | Name | Version | Description |
|--------------|------|---------|-------------|
| 0x01 | FULL_NODE | 1 | Full node, stores complete blockchain |
| 0x02 | LIGHT_NODE | 1 | Light node, stores only block headers |
| 0x03 | VALIDATOR | 1 | Validator node, participates in PoC consensus |
| 0x04 | RELAY | 1 | Relay node, only forwards messages |
| 0x05 | DISCOVERY | 1 | Discovery node, participates in DHT |

### 3.4 Handshake Validation

```go
func ValidateHandshake(hs *Handshake, peerPublicKey []byte) error {
    // 1. Version check
    if hs.Version != 0x0100 {
        return ErrInvalidVersion
    }

    // 2. Network ID check
    if hs.NetworkID != MainnetNetworkID {
        return ErrInvalidNetworkID
    }

    // 3. Timestamp validation (±2 minutes)
    now := time.Now().Unix()
    if math.Abs(float64(now-int64(hs.Timestamp))) > 120 {
        return ErrInvalidTimestamp
    }

    // 4. Signature validation
    data := GetHandshakeData(hs)
    if !crypto.Verify(peerPublicKey, data, hs.Signature) {
        return ErrInvalidSignature
    }

    return nil
}
```

---

## 4. Message Format

### 4.1 Message Header

All P2P messages share a common base message header:

```
+-----+-----+-----+-----+-----+-----+-----+-----+
|  Version  |  Type     |   Length           |
+-----+-----+-----+-----+-----+-----+-----+-----+
|  Checksum (4 bytes)                         |
+---------------------------------------------+
|  Payload (Length bytes)                     |
+---------------------------------------------+
```

**Field Descriptions**:
- `Version`: 2 bytes, protocol version (0x0100)
- `Type`: 2 bytes, message type
- `Length`: 4 bytes, payload length
- `Checksum`: 4 bytes, CRC32 checksum of payload
- `Payload`: Variable length, message payload

### 4.2 Message Types

| Message Type | Value | Direction | Description |
|--------------|-------|-----------|-------------|
| HANDSHAKE | 0x01 | Bidirectional | Handshake message |
| HANDSHAKE_ACK | 0x02 | Bidirectional | Handshake acknowledgment |
| PING | 0x03 | Bidirectional | Heartbeat message |
| PONG | 0x04 | Bidirectional | Heartbeat response |
| FIND_NODE | 0x05 | Request | Find node |
| NODES | 0x06 | Response | Node list |
| GET_BLOCKS | 0x07 | Request | Get blocks |
| BLOCKS | 0x08 | Response | Block data |
| NEW_BLOCK | 0x09 | Broadcast | New block announcement |
| GET_TRANSACTIONS | 0x0A | Request | Get transactions |
| TRANSACTIONS | 0x0B | Response | Transaction data |
| NEW_TRANSACTION | 0x0C | Broadcast | New transaction announcement |
| GET_BLOCK_HASHES | 0x0D | Request | Get block hash list |
| BLOCK_HASHES | 0x0E | Response | Block hash list |
| ADDR | 0x0F | Broadcast | Node address broadcast |

### 4.3 Message Serialization

Use protobuf for message serialization:

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

## 5. Block Synchronization

### 5.1 New Block Broadcast

When a node generates or receives a new block, immediately broadcast to all peers:

#### NEW_BLOCK Message

```go
type NewBlockMessage struct {
    Version    uint16    // Protocol version
    Height     uint64    // Block height
    BlockHash  []byte    // Block hash (32 bytes)
    BlockData  *Block    // Block data (optional, compact format)
    SenderID   PeerID    // Sender node ID
    Timestamp  uint64    // Timestamp
}
```

#### Broadcast Strategy

```go
func BroadcastNewBlock(block *Block) {
    // 1. Validate block
    if err := ValidateBlock(block); err != nil {
        return
    }

    // 2. Create NEW_BLOCK message
    msg := &NewBlockMessage{
        Version:   0x0100,
        Height:    block.Header.Height,
        BlockHash: block.Hash(),
        BlockData: block,
        Timestamp: uint64(time.Now().Unix()),
    }

    // 3. Broadcast to all peers
    for _, peer := range peers {
        peer.SendMessage(msg)
    }
}
```

### 5.2 Block Request

When a node falls behind the blockchain, request missing blocks:

#### GET_BLOCKS Message

```go
type GetBlocksMessage struct {
    Version     uint16  // Protocol version
    StartHeight uint64  // Starting height
    EndHeight   uint64  // Ending height (0 means request latest)
    MaxCount    uint32  // Maximum return count
    SenderID    PeerID  // Sender node ID
}
```

#### BLOCKS Message

```go
type BlocksMessage struct {
    Version  uint16   // Protocol version
    Blocks   []*Block // Block list
    SenderID PeerID   // Sender node ID
}
```

#### Synchronization Algorithm

```
1. Compare local height with peer heights
2. If behind, send GET_BLOCKS request
3. Receive BLOCKS response
4. Validate and add to local chain in order
5. If still behind, repeat steps 2-4
```

### 5.3 Fast Sync

Use block header sync mechanism for light nodes:

```go
type GetBlockHashesMessage struct {
    Version  uint16  // Protocol version
    StartHash []byte // Starting block hash
    MaxCount uint32  // Maximum return count
}

type BlockHashesMessage struct {
    Version  uint16   // Protocol version
    Hashes   [][]byte // Block hash list
    StopHash []byte   // Stop hash
}
```

---

## 6. Transaction Broadcasting

### 6.1 Transaction Pool Management

Nodes maintain a transaction pool (mempool):

```go
type TxPool struct {
    pending    map[string]*Transaction  // Pending transactions
    queue      *PriorityQueue           // Priority queue
    maxSize    int                      // Max capacity (10,000)
}
```

### 6.2 New Transaction Broadcast

#### NEW_TRANSACTION Message

```go
type NewTransactionMessage struct {
    Version  uint16       // Protocol version
    TxHash   []byte       // Transaction hash (32 bytes)
    TxData   *Transaction // Transaction data (compact format)
    SenderID PeerID       // Sender node ID
}
```

#### Gossip Protocol

Use Gossip protocol for transaction broadcasting:

```go
func BroadcastTransaction(tx *Transaction) {
    // 1. Validate transaction
    if err := ValidateTransaction(tx); err != nil {
        return
    }

    // 2. Add to local transaction pool
    txPool.AddTransaction(tx)

    // 3. Select random subset of peers to broadcast
    peers := SelectRandomPeers(8) // Randomly select 8 nodes

    // 4. Broadcast message
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

### 6.3 Transaction Request

When a node needs specific transactions:

```go
type GetTransactionsMessage struct {
    Version  uint16   // Protocol version
    TxHashes [][]byte // Transaction hash list
    SenderID PeerID   // Sender node ID
}

type TransactionsMessage struct {
    Version      uint16       // Protocol version
    Transactions []*Transaction // Transaction list
    SenderID     PeerID       // Sender node ID
}
```

---

## 7. Connection Management

### 7.1 Heartbeat Mechanism

Use PING/PONG messages to detect connection liveness:

```go
type PingMessage struct {
    Version   uint16 // Protocol version
    Nonce     uint64 // Random number
    SenderID  PeerID // Sender node ID
    Timestamp uint64 // Timestamp
}

type PongMessage struct {
    Version   uint16 // Protocol version
    Nonce     uint64 // Corresponding PING's Nonce
    SenderID  PeerID // Sender node ID
    Timestamp uint64 // Timestamp
}
```

#### Heartbeat Parameters

| Parameter | Value | Description |
|-----------|-------|-------------|
| PingInterval | 30 seconds | PING send interval |
| PongTimeout | 5 seconds | PONG wait timeout |
| MaxMissedPings | 3 | Maximum missed PING count |

### 7.2 Timeout Handling

```go
func (p *Peer) CheckTimeout() {
    now := time.Now()

    // 1. Check PONG timeout
    if p.lastPingReceived != (time.Time{}) {
        elapsed := now.Sub(p.lastPingReceived)
        if elapsed > time.Duration(PingInterval*3) {
            p.Disconnect(ErrPeerTimeout)
        }
    }

    // 2. Send periodic PING
    if now.Sub(p.lastPingSent) > time.Duration(PingInterval) {
        p.SendPing()
    }
}
```

### 7.3 Reconnection Mechanism

```go
type ReconnectPolicy struct {
    InitialDelay time.Duration // Initial delay (5 seconds)
    MaxDelay     time.Duration // Maximum delay (5 minutes)
    BackoffFactor float64      // Backoff factor (2.0)
    MaxAttempts  int           // Maximum attempts (10)
}

func (p *Peer) Reconnect() {
    delay := p.reconnectPolicy.InitialDelay

    for i := 0; i < p.reconnectPolicy.MaxAttempts; i++ {
        time.Sleep(delay)

        err := p.Connect()
        if err == nil {
            return // Connection successful
        }

        // Exponential backoff
        delay = time.Duration(float64(delay) * p.reconnectPolicy.BackoffFactor)
        if delay > p.reconnectPolicy.MaxDelay {
            delay = p.reconnectPolicy.MaxDelay
        }
    }

    log.Error("Failed to reconnect after", p.reconnectPolicy.MaxAttempts, "attempts")
}
```

### 7.4 Connection Pool Management

```go
type PeerManager struct {
    peers        map[PeerID]*Peer  // Active peers
    maxPeers     int               // Maximum peers (128)
    incoming     chan *Peer        // New connection channel
    disconnectCh chan *Peer        // Disconnection channel
}

func (pm *PeerManager) AddPeer(peer *Peer) error {
    // 1. Check connection limit
    if len(pm.peers) >= pm.maxPeers {
        return ErrTooManyPeers
    }

    // 2. Check duplicate connection
    if _, exists := pm.peers[peer.ID]; exists {
        return ErrPeerAlreadyConnected
    }

    // 3. Add to connection pool
    pm.peers[peer.ID] = peer

    // 4. Start heartbeat
    go peer.Heartbeat()

    return nil
}
```

---

## 8. Security

### 8.1 DoS Protection

#### Connection Rate Limiting

```go
type ConnectionLimiter struct {
    maxConnections    int           // Maximum connections (128)
    maxConnectionsPerIP int         // Max connections per IP (2)
    connectionAttempts  map[string]int  // Connection attempt count
    banDuration        time.Duration  // Ban duration (1 hour)
}

func (cl *ConnectionLimiter) AllowConnection(ip string) bool {
    // 1. Check if banned
    if cl.isBanned(ip) {
        return false
    }

    // 2. Check connection limit
    count := cl.connectionAttempts[ip]
    if count >= cl.maxConnectionsPerIP {
        cl.banIP(ip)
        return false
    }

    // 3. Increment count
    cl.connectionAttempts[ip] = count + 1
    return true
}
```

#### Message Rate Limiting

```go
type MessageRateLimiter struct {
    buckets map[PeerID]*TokenBucket  // Token bucket per node
    rate    float64                  // Rate (per second)
    burst   int                      // Burst size
}

type TokenBucket struct {
    tokens float64
    lastUpdate time.Time
}

func (mrl *MessageRateLimiter) AllowMessage(peerID PeerID) bool {
    bucket := mrl.buckets[peerID]
    now := time.Now()

    // Calculate added tokens
    elapsed := now.Sub(bucket.lastUpdate).Seconds()
    bucket.tokens += elapsed * mrl.rate

    // Limit max tokens
    if bucket.tokens > float64(mrl.burst) {
        bucket.tokens = float64(mrl.burst)
    }

    // Consume token
    if bucket.tokens >= 1.0 {
        bucket.tokens -= 1.0
        bucket.lastUpdate = now
        return true
    }

    return false
}
```

#### Blacklist Mechanism

```go
type Blacklist struct {
    entries map[string]*BlacklistEntry  // IP or node ID
    mutex   sync.RWMutex
}

type BlacklistEntry struct {
    Reason   string      // Ban reason
    Expires time.Time    // Expiration time
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

    // Check if expired
    if time.Now().After(entry.Expires) {
        delete(bl.entries, id)
        return false
    }

    return true
}
```

### 8.2 Malicious Node Detection

#### Behavior Scoring

```go
type PeerScore struct {
    successes  int       // Successful interaction count
    failures  int       // Failed interaction count
    lastUpdate time.Time
}

func (ps *PeerScore) CalculateScore() float64 {
    total := ps.successes + ps.failures
    if total == 0 {
        return 0.5  // Default score
    }

    return float64(ps.successes) / float64(total)
}

func (ps *PeerScore) IsMalicious() bool {
    // Score below 0.3 is considered malicious
    return ps.CalculateScore() < 0.3
}
```

#### Downgrade Handling

```go
func (pm *PeerManager) HandleMaliciousPeer(peer *Peer) {
    // 1. Record malicious behavior
    peer.score.failures++

    // 2. If score too low, disconnect
    if peer.score.IsMalicious() {
        pm.DisconnectPeer(peer.ID)
        pm.blacklist.Ban(peer.ID.String(), "Malicious behavior", 24*time.Hour)
    }

    // 3. Lower priority
    peer.priority = LowPriority
}
```

### 8.3 Encrypted Communication

Although IPv6 is natively unencrypted, node communication can use TLS:

```go
func CreateTLSConnection() (*tls.Conn, error) {
    // Generate self-signed certificate
    cert, err := generateSelfSignedCert()
    if err != nil {
        return nil, err
    }

    // Configure TLS
    config := &tls.Config{
        Certificates:       []tls.Certificate{cert},
        InsecureSkipVerify: true,  // P2P networks may not need verification
        MinVersion:         tls.VersionTLS12,
        CipherSuites: []uint16{
            tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
            tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
        },
    }

    // Create TLS connection
    conn, err := tls.Dial("tcp", peerAddr, config)
    if err != nil {
        return nil, err
    }

    return conn, nil
}
```

---

## 9. NAT Traversal

Although IPv6 doesn't require NAT, traversal solutions are provided for NAT66 and dual-stack environments.

### 9.1 STUN Protocol

Use STUN protocol to discover public address:

```go
type STUNRequest struct {
    MessageClass STUNClass  // Request
    Method       STUNMethod // Binding request
    TransactionID []byte    // Transaction ID (12 bytes)
}

type STUNResponse struct {
    MessageClass STUNClass  // Success response
    Method       STUNMethod // Binding response
    Attributes   []STUNAttribute
}

type STUNAttribute struct {
    Type   uint16
    Length uint16
    Value  []byte
}
```

### 9.2 UPnP/PCP

Use UPnP or PCP protocol for port mapping:

```go
func MapPort(port uint16) error {
    // UPnP device discovery
    devices, err := upnp.Discover()
    if err != nil {
        return err
    }

    // Port mapping
    for _, device := range devices {
        err = device.AddPortMapping("TCP", port, port, "V6Coin P2P", 3600)
        if err != nil {
            return err
        }
    }

    return nil
}
```

### 9.3 Relay Connection

For nodes that cannot connect directly, use relay nodes:

```go
type RelayRequest struct {
    Version  uint16
    TargetID PeerID
    SenderID PeerID
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

## 10. Implementation Examples

### 10.1 Creating a P2P Node

```go
package p2p

import (
    "crypto/ed25519"
    "net"
    "time"
)

// P2P node structure
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

// Configuration
type Config struct {
    ListenAddr       string        // Listen address
    MaxPeers         int           // Maximum peers
    PingInterval     time.Duration // Heartbeat interval
    BootstrapPeers   []PeerInfo    // Bootstrap nodes
    EnableUPnP       bool          // Enable UPnP
}

// Create new node
func NewNode(privateKey ed25519.PrivateKey, config *Config) (*Node, error) {
    publicKey := privateKey.Public()

    // Generate node ID
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

// Generate node ID
func GenerateNodeID(publicKey ed25519.PublicKey) PeerID {
    hash := sha256.Sum256(publicKey)
    return hash[:16]
}
```

### 10.2 Starting a P2P Node

```go
func (n *Node) Start() error {
    // 1. Create listener
    listener, err := net.Listen("tcp", n.Config.ListenAddr)
    if err != nil {
        return err
    }
    n.ListenAddr = listener.Addr()

    // 2. UPnP port mapping
    if n.Config.EnableUPnP {
        go n.MapPort()
    }

    // 3. Connect to bootstrap nodes
    go n.Bootstrap()

    // 4. Start accepting connections
    go n.AcceptConnections()

    // 5. Start periodic tasks
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

        // Handle new connection
        go n.HandleNewConnection(conn)
    }
}
```

### 10.3 Handling New Connections

```go
func (n *Node) HandleNewConnection(conn net.Conn) {
    // 1. Read handshake message
    var hs Handshake
    decoder := gob.NewDecoder(conn)
    if err := decoder.Decode(&hs); err != nil {
        conn.Close()
        return
    }

    // 2. Validate handshake
    if err := ValidateHandshake(&hs, hs.NodeID); err != nil {
        log.Error("Invalid handshake:", err)
        conn.Close()
        return
    }

    // 3. Check blacklist
    if n.Blacklist.IsBanned(hs.NodeID.String()) {
        log.Error("Peer is blacklisted:", hs.NodeID)
        conn.Close()
        return
    }

    // 4. Create peer
    peer := NewPeer(conn, hs.NodeID, hs.ListenAddr)

    // 5. Send handshake response
    ack := n.CreateHandshakeAck()
    encoder := gob.NewEncoder(conn)
    if err := encoder.Encode(&ack); err != nil {
        conn.Close()
        return
    }

    // 6. Add to peer manager
    if err := n.PeerManager.AddPeer(peer); err != nil {
        log.Error("Failed to add peer:", err)
        conn.Close()
        return
    }

    // 7. Start message handling
    go peer.StartMessageHandler()
    go peer.Heartbeat()

    log.Info("New peer connected:", peer.ID)
}
```

### 10.4 Message Processing

```go
func (p *Peer) StartMessageHandler() {
    decoder := gob.NewDecoder(p.Conn)

    for {
        var msg Message
        if err := decoder.Decode(&msg); err != nil {
            log.Error("Decode error:", err)
            break
        }

        // Handle message
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

### 10.5 Broadcasting New Blocks

```go
func (n *Node) BroadcastBlock(block *Block) {
    // 1. Validate block
    if err := ValidateBlock(block); err != nil {
        return
    }

    // 2. Create NEW_BLOCK message
    msg := &NewBlockMessage{
        Version:   0x0100,
        Height:    block.Header.Height,
        BlockHash: block.Hash(),
        BlockData: block,
        Timestamp: uint64(time.Now().Unix()),
    }

    // 3. Serialize message
    data, err := SerializeMessage(msg)
    if err != nil {
        return
    }

    // 4. Broadcast to all peers
    for _, peer := range n.PeerManager.peers {
        peer.SendMessage(data)
    }

    log.Info("Broadcasted block:", block.Header.Height)
}
```

### 10.6 Blockchain Synchronization

```go
func (n *Node) SyncBlockchain() error {
    // 1. Get local height
    localHeight := n.GetLocalHeight()

    // 2. Get max height from peers
    maxHeight := n.GetMaxPeerHeight()

    // 3. If behind, start sync
    if localHeight < maxHeight {
        return n.DownloadBlocks(localHeight + 1, maxHeight)
    }

    return nil
}

func (n *Node) DownloadBlocks(start, end uint64) error {
    batchSize := uint32(100)  // Download 100 blocks per batch

    for start <= end {
        // 1. Select best peer
        peer := n.SelectBestPeer()

        // 2. Request blocks
        req := &GetBlocksMessage{
            Version:     0x0100,
            StartHeight: start,
            EndHeight:   min(start+uint64(batchSize)-1, end),
            MaxCount:    batchSize,
        }

        // 3. Send request
        resp, err := peer.RequestBlocks(req)
        if err != nil {
            log.Error("Request blocks error:", err)
            continue
        }

        // 4. Validate and add blocks
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

## 11. Configuration Parameters

### 11.1 Network Parameters

| Parameter | Value | Description |
|-----------|-------|-------------|
| DefaultPort | 38901 | Default P2P port |
| ProtocolVersion | 0x0100 | Protocol version |
| NetworkID | 0x1 | Mainnet network ID |
| MaxPeers | 128 | Maximum peers |
| MinPeers | 16 | Minimum peers |
| K | 16 | K-bucket size |
| Alpha | 3 | Concurrent lookup count |

### 11.2 Timeout Parameters

| Parameter | Value | Description |
|-----------|-------|-------------|
| PingInterval | 30 seconds | Heartbeat interval |
| PongTimeout | 5 seconds | PONG wait timeout |
| ConnectionTimeout | 10 seconds | Connection timeout |
| HandshakeTimeout | 5 seconds | Handshake timeout |
| RequestTimeout | 30 seconds | Request timeout |
| BlockRequestTimeout | 60 seconds | Block request timeout |

### 11.3 Rate Limiting Parameters

| Parameter | Value | Description |
|-----------|-------|-------------|
| MaxMessageSize | 10 MB | Maximum message size |
| MaxMessagesPerSecond | 100 | Maximum messages per second |
| MaxConnectionsPerIP | 2 | Maximum connections per IP |
| BanDuration | 1 hour | Ban duration |
| MaxBanDuration | 24 hours | Maximum ban duration |

### 11.4 Bootstrap Nodes

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

## 12. State Machine Diagrams

### 12.1 Connection State Machine

```
                        [DISCONNECTED]
                              |
                              | Connection request
                              v
                        [CONNECTING]
                              |
                              | Connection successful
                              v
                        [HANDSHAKING]
                              |
                              | Handshake successful
                              v
                        [CONNECTED]
                              |
                              | Active heartbeat
                              v
                        [ACTIVE]
                              |
                              | Heartbeat timeout / Error
                              v
                        [DISCONNECTING]
                              |
                              v
                        [DISCONNECTED]
```

### 12.2 Node Discovery State Machine

```
                        [INITIAL]
                              |
                              | Start
                              v
                        [BOOTSTRAPPING]
                              |
                              | Get seed nodes
                              v
                        [DISCOVERING]
                              |
                              | Find nodes
                              v
                        [MAINTAINING]
                              |
                              | Periodically refresh k-buckets
                              v
                        [ACTIVE]
```

---

## 13. Version History

| Version | Date | Changes |
|---------|------|---------|
| v1.0 | 2026-02-02 | Initial version, defines P2P network protocol |

---

## 14. Reference Documents

- [RFC 6980](https://datatracker.ietf.org/doc/html/rfc6980) - IPv6 Hop-by-Hop Options Header
- [Kademlia: A Peer-to-peer Information System Based on XOR Metric](https://pdos.csail.mit.edu/~petar/papers/maymounkov-kademlia-lncs.pdf)
- [RFC 5389](https://datatracker.ietf.org/doc/html/rfc5389) - STUN Protocol
- [V6Coin Whitepaper](../whitepaper/V6Coin_Whitepaper_EN.md)
- [IPv6 Extension Header Specification](./01-ipv6-header-spec.md)
- [CGA Address Mapping Specification](./02-cga-address-spec.md)
- [PoC Consensus Specification](./03-poc-consensus-spec.md)
- [Transaction Validation Specification](./04-transaction-spec.md)

---

**End of Document**
