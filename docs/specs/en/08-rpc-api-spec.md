# RPC API Specification

**Version**: v1.0
**Date**: 2026-02-01

---

## 1. Overview

V6Coin provides a REST API for node interaction, blockchain queries, and network monitoring. The API follows RESTful design principles with JSON responses and standard HTTP status codes.

### Design Goals

1. **Simplicity**: Easy to use and understand
2. **Performance**: Fast response times, efficient data retrieval
3. **Security**: Authentication and authorization for sensitive operations
4. **Standard Compliance**: Follows REST and JSON-RPC standards
5. **Developer Friendly**: Comprehensive documentation and examples

---

## 2. API Versioning

All API endpoints are prefixed with `/api/v1/`. Current version: v1.0.

### Versioning Rules
- **Major**: Breaking changes (e.g., `/api/v2/`)
- **Minor**: Backwards compatible additions (e.g., `/api/v1/metrics`)
- **Patch**: Bug fixes

---

## 3. Base URL

```
http://localhost:38901/api/v1
```

**Note**: Port 38901 is the default P2P network port for V6Coin

---

## 4. HTTP Methods

| Method | Usage | Description |
|--------|-------|-------------|
| GET | Retrieve data |
| POST | Create resources |
| PUT | Update resources |
| DELETE | Remove resources |

---

## 5. Common Response Format

All API responses use the following JSON structure:

```json
{
    "success": true,
    "data": {...},
    "error": null,
    "message": "Optional message or error details"
}
```

---

## 6. Status Codes

| Code | Message | Description |
|------|---------|-------------|
| 200 | OK | Request successful |
| 400 | Bad Request | Malformed request syntax or parameters |
| 401 | Unauthorized | Authentication required |
| 403 | Forbidden | Resource not accessible |
| 404 | Not Found | Requested resource does not exist |
| 500 | Internal Error | Server error (should not be returned directly) |
| 503 | Service Unavailable | Service temporarily down |
| 504 | Gateway Timeout | Request took too long |

---

## 7. Data Types

### 7.1 Block Related

**Block Header**:
```go
type BlockHeader struct {
    Version       uint16
    Height        uint64
    PrevBlockHash []byte
    MerkleRoot    []byte
    Timestamp     uint64
    ValidatorID   string
    StateRoot     []byte
    Signature     []byte
}
```

### 7.2 Transaction Related

**Transaction**:
```go
type Transaction struct {
    Version   uint16
    Type      TxType
    From      net.IP
    To        net.IP
    Amount    uint64
    Fee       uint64
    Nonce     uint64
    Timestamp uint64
    Signature []byte
    Data      []byte
}
```

### 7.3 Node Related

**V6Address**:
```go
type V6Address struct {
    NetworkPrefix net.IP
    ID           uint64
    IsTemporary   bool
    AddressType   AddressType
    CreatedAt     time.Time
    ExpiresAt     time.Time
    Index         int
}
```

**NodeContribution**:
```go
type NodeContribution struct {
    NodeID        p2p.PeerID
    OnlineTime    time.Duration
    LastOnline   time.Time
    PacketLoss    float64
    Forwarded     uint64
    Score         float64
    RewardBalance uint64
}
```

---

## 8. Endpoints

### 8.1 Blockchain Queries

#### GET /api/v1/blocks
- **Description**: Retrieve latest blocks

**Request Parameters**:
| Parameter | Type | Required | Description |
|-----------|--------|----------|--------------|
| height | uint64 | No | Starting block height (optional, default: latest) |
| count | uint64 | No | Number of blocks to return (default: 10, max: 100) |

**Response** (Success):
```json
{
  "success": true,
  "data": {
    "blocks": [
      {
        "height": 100,
        "hash": "0x123...",
        "timestamp": 17067296000,
        "transactions": [...]
      }
    ],
    "pagination": {
      "page": 1,
      "pageSize": 10,
      "totalPages": 5
    }
  }
}
```

**Response** (Error):
```json
{
  "success": false,
  "error": "ERR_BLOCK_NOT_FOUND",
  "message": "Block not found at height 100"
}
```

---

#### GET /api/v1/blocks/{hash}

- **Description**: Get block by hash

**Response**:
```json
{
  "success": true,
  "data": {
    "height": 100,
    "hash": "0x123...",
    "timestamp": 17067296000,
    "transactions": [...]
  }
}
```

---

#### GET /api/v1/blocks/latest

- **Description**: Get latest block (shortcut for `/api/v1/blocks?height=latest`)

**Response**: Same as `/api/v1/blocks?height=latest`

---

### 8.2 Transaction Queries

#### GET /api/v1/transactions

- **Description**: Query transactions with filters

**Request Parameters**:
| Parameter | Type | Required | Description |
|-----------|--------|----------|--------------|
| hash | string | No | Transaction hash |
| type | uint8 | No | Transaction type (0=online, 1=offline, etc.) |
| from | string | No | Sender address (16 bytes, IPv6 format) |
| to | string | No | Receiver address |
| limit | uint64 | No | Max results (default: 100) |
| offset | uint64 | No | Pagination offset |

**Response**:
```json
{
  "success": true,
  "data": {
    "transactions": [
      {
        "hash": "0xabcd...",
        "type": 1,
        "from": "2001:db8::1",
        "to": "2001:db8::2",
        "amount": "1000000000000",
        "timestamp": 17067296000,
        "fee": "1000",
        "nonce": 1
      }
    ],
    "pagination": {
      "page": 1,
      "pageSize": 20,
      "totalPages": 3
    }
  }
}
```

#### GET /api/v1/transactions/{hash}

- **Description**: Get transaction by hash

**Response**:
```json
{
  "success": true,
  "data": {
    "hash": "0xabcd...",
    "type": 1,
    "from": "2001:db8::1",
    "to": "2001:db8::2",
    "amount": "1000000000000",
    "timestamp": 17067296000,
    "fee": "1000",
    "nonce": 1
  }
}
```

#### GET /api/v1/transactions/pending

- **Description**: Get pending transactions from pool

**Response**:
```json
{
  "success": true,
  "data": {
    "transactions": [...]
  }
}
```

---

#### GET /api/v1/transactions/mempool

- **Description**: Get memory pool transactions

**Response**:
```json
{
  "success": true,
  "data": {
    "transactions": [...]
  }
}
```

---

### 8.3 Account Queries

#### GET /api/v1/accounts/{address}

- **Description**: Get account information by IPv6 address

**Response**:
```json
{
  "success": true,
  "data": {
    "address": "2001:db8::1",
    "id": uint64,
    "balance": "1000000000000",
    "nonce": 1,
    "transactions": [...]
  }
}
```

#### GET /api/v1/accounts/{address}/balance

- **Description**: Get account balance

**Response**:
```json
{
  "success": true,
  "data": {
    "address": "2001:db8::1",
    "balance": "1000000000000"
  }
}
```

#### GET /api/v1/accounts/{address}/nonce

- **Description**: Get current nonce for address

**Response**:
```json
{
  "success": true,
  "data": {
    "address": "2001:db8::1",
    "nonce": 1
  }
}
```

---

### 8.4 Node Queries

#### GET /api/v1/node/info

- **Description**: Get current node information

**Response**:
```json
{
  "success": true,
  "data": {
    "id": "node1",
    "address": "2001:db8::1",
    "height": 100,
    "peers": 12,
    "syncing": true,
    "chainHead": {
      "height": 100,
      "hash": "0x123...",
      "timestamp": 17067296000
    }
  }
}
```

---

#### GET /api/v1/node/peers

- **Description**: Get connected peer list

**Response**:
```json
{
  "success": true,
  "data": {
    "peers": [
      {
        "id": "peer1",
        "address": "2001:db8::1",
        "height": 100,
        "onlineTime": 86400,
        "lastOnline": "2026-02-01T00:00:00Z",
        "packetLoss": 0.01,
        "forwarded": 104857600,
        "score": 0.654
      }
    ]
  }
}
```

---

#### GET /api/v1/node/stats

- **Description**: Get detailed node statistics

**Response**:
```json
{
  "success": true,
  "data": {
    "uptime": "99.9",
    "uptimePercentage": 0.99,
    "totalForwarded": "104857600",
    "totalReceived": "1056636388",
    "averageLatency": "50ms",
    "packetLoss": 0.01,
    "connections": 12,
    "activeValidators": 10
    "contributionScore": 0.654
  }
}
```

---

### 8.5 Network Queries

#### GET /api/v1/network/status

- **Description**: Get network health status

**Response**:
```json
{
  "success": true,
  "data": {
    "totalPeers": 1250,
    "activePeers": 1180,
    "inactivePeers": 70,
    "averageUptime": "99.5%",
    "networkStatus": "healthy",
    "message": "All systems operational"
  }
}
```

#### GET /api/v1/network/topology

- **Description**: Get network topology and relationships

**Response**:
```json
{
  "success": true,
  "data": {
    "nodes": [
      {
        "id": "node1",
        "peers": [
          {"id": "peer2", "address": "2001:db8::2", "height": 99}
        ]
      },
      {
        "id": "node2",
        "peers": [
          {"id": "peer1", "address": "2001:db8::1", "height": 98}
        ]
      }
    ],
    "edges": [
      {
        "from": "node1",
        "to": "node2",
        "weight": 1
      }
    ]
  }
}
```

---

## 9. Write Operations

### 9.1 POST /api/v1/transactions

- **Description**: Broadcast a new transaction to network

**Request**:
```json
{
  "from": "2001:db8::1",
  "to": "2001:db8::2",
  "amount": 1000000000000",
  "fee": "1000",
  "data": "Optional transaction data"
}
```

**Response**:
```json
{
  "success": true,
  "data": {
    "hash": "0xabcd...",
    "timestamp": 17067296000,
    "status": "pending",
    "confirmations": 3
  }
}
```

### 9.2 POST /api/v1/accounts

- **Description**: Create new account (for testing only)

**Request**:
```json
{
  "password": "securepassword123"
}
```

---

## 10. WebSocket

### 10.1 WS /api/v1/ws

- **Description**: WebSocket endpoint for real-time updates

**Endpoint**: `ws://localhost:38901/api/v1/ws`

**Subscriptions**:
- `blocks/new` - New block notifications
- `transactions/new` - New transaction notifications
- `network/stats` - Network statistics updates

### 10.2 Message Format

**Block Notification**:
```json
{
  "type": "block",
  "data": {
    "height": 101,
    "hash": "0x456...",
    "timestamp": 17067298000
  }
}
```

**Transaction Notification**:
```json
{
  "type": "transaction",
  "data": {
    "hash": "0xabcd...",
    "type": 1,
    "status": "pending"
  }
}
```

---

## 11. Error Handling

### 11.1 Error Response Format

All error responses follow this format:
```json
{
  "success": false,
  "error": "ERR_INVALID_TX_TYPE",
  "message": "Invalid transaction type. Must be one of: 0-7"
}
```

### 11.2 Common Error Codes

| Error Code | HTTP Status | User Action |
|---------|--------------|----------------------|
| 200 | 200 OK | None |
| 400 | 400 Bad Request | Fix request format |
| 401 | 401 Unauthorized | Add authentication |
| 403 | 403 Forbidden | Check permissions |
| 404 | 404 Not Found | Check resource exists |
| 409 | 409 Conflict | Resolve conflict |
| 500 | 500 Internal | Wait and retry |

---

## 12. Authentication

### 12.1 Authentication Methods

**Currently**: Basic Auth (for development only)

**Future**: JWT or OAuth 2.0

**Request Header**:
```
Authorization: Bearer <token>
```

### 12.2 Token Format

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsIn...
  "expires_in": 864,
  "permissions": ["read", "write"]
}
```

---

## 13. Rate Limiting

### 13.1 Default Limits

| Resource Type | Rate Limit |
|--------------|----------|--------||
| Public API | 100 req/min |
| WebSocket | 50 msg/min |
| Complex Queries | 10 req/min |

---

## 14. Code Examples

### 14.1 Basic Block Query

```bash
curl -X GET "http://localhost:38901/api/v1/blocks?height=100"
```

**Response**:
```json
{
  "success": true,
  "data": {
    "blocks": [
      {
        "height": 100,
        "hash": "0x123...",
        "timestamp": 17067296000
      }
    ]
  }
}
```

### 14.2 Get Transaction by Hash

```bash
curl -X GET "http://localhost:38901/api/v1/transactions/0xabcd..."
```

### 14.3 Create Transaction

```bash
curl -X POST "http://localhost:38901/api/v1/transactions" \
  -H "Content-Type: application/json" \
  -d '{
    "from": "2001:db8::1",
    "to": "2001:db8::2",
    "amount": "1000000000000",
    "fee": "1000"
  }'
```

---

## 15. WebSocket Example

### 15.1 Connect and Subscribe

```javascript
const ws = new WebSocket('ws://localhost:38901/api/v1/ws');

ws.onopen = () => {
  ws.send(JSON.stringify({
    "subscribe": ["blocks/new"]
  }));
};

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log(`New ${data.type} notification:`, data);
};
```

---

## 16. Dependencies

### 16.1 Required Modules

- `code/go/pkg/blockchain/chain.go` - Blockchain data structure
- `code/go/pkg/consensus/consensus.go` - Consensus engine
- `code/go/pkg/tx/pool.go` - Transaction pool
- `code/go/pkg/address/address.go` - Address management
- `code/go/pkg/state/` - State DB interface
- `code/go/pkg/p2p/network.go` - P2P networking

---

## 17. Security Considerations

### 17.1 Input Validation

1. **Address Format**:
   - Must be valid IPv6 address (16 bytes, IID bit 0 = 0)
   - Length must be 16 bytes

2. **Amount Validation**:
   - Must be positive integer in nano-V6 units
   - Must not overflow

3. **Signature Verification**:
   - Must be valid Ed25519 signature

### 17.2 Rate Limiting

1. **Global Limit**: 100 requests/second
2. **Per-IP Limit**: 1000 requests/minute
3. **Burst Limit**: 10 requests/second

### 17.3 CORS

1. **Origins**: `*` (allow all), `localhost:38901` (development)
2. **Methods**: `GET, POST, PUT, DELETE`
3. **Headers**: `Content-Type`, `Authorization`

---

## 18. Future Extensions

- **WebSocket Subscriptions**: Event-driven updates
- **GraphQL API**: Querying complex data structures
- **IPFS**: Decentralized file storage
- **Streaming Data**: Large data set retrieval

---

## 19. Version History

| Version | Date | Changes |
|---------|------|---------|
| v1.0 | 2026-02-01 | Initial release |

---

## 20. Related Documentation

- [V6Coin Whitepaper (Chinese)](../whitepaper/V6Coin_Whitepaper_CN.md)
- [V6Coin Whitepaper (English)](../whitepaper/V6Coin_hepaper_EN.md)
- [Transaction Validation Specification](./04-transaction-spec.md)
- [Blockchain Data Structure Specification](./07-blockchain-spec.md)

---

**API Base URL**: `http://localhost:38901/api/v1`