# RPC API 规范

**版本**: v1.0
**日期**: 2026-02-01

---

## 1. 概述

V6Coin 提供了 REST API 用于节点交互、区块链查询和网络监控。该 API 遵循 RESTful 设计原则，使用 JSON 响应和标准 HTTP 状态码。

### 设计目标

1. **简洁性**: 易于使用和理解
2. **性能**: 快速响应时间，高效数据检索
3. **安全性**: 敏感操作的认证和授权
4. **标准兼容**: 遵循 REST 和 JSON-RPC 标准
5. **开发者友好**: 完整的文档和示例

---

## 2. API 版本控制

所有 API 端点以 `/api/v1/` 为前缀。当前版本：v1.0。

### 版本规则
- **主版本**: 破坏性变更（例如 `/api/v2/`）
- **次版本**: 向后兼容的新增功能（例如 `/api/v1/metrics`）
- **修订版本**: Bug 修复

---

## 3. 基础 URL

```
http://localhost:38901/api/v1
```

**注意**: 端口 38901 是 V6Coin 的默认 P2P 网络端口

---

## 4. HTTP 方法

| 方法 | 用途 | 说明 |
|--------|-------|-------------|
| GET | 获取数据 | 检索资源 |
| POST | 创建资源 | 创建新资源 |
| PUT | 更新资源 | 更新现有资源 |
| DELETE | 删除资源 | 删除资源 |

---

## 5. 通用响应格式

所有 API 响应使用以下 JSON 结构：

```json
{
    "success": true,
    "data": {...},
    "error": null,
    "message": "可选消息或错误详情"
}
```

---

## 6. 状态码

| 代码 | 消息 | 说明 |
|------|---------|-------------|
| 200 | OK | 请求成功 |
| 400 | Bad Request | 请求语法或参数格式错误 |
| 401 | Unauthorized | 需要认证 |
| 403 | Forbidden | 资源不可访问 |
| 404 | Not Found | 请求的资源不存在 |
| 500 | Internal Error | 服务器错误（不应直接返回） |
| 503 | Service Unavailable | 服务暂时不可用 |
| 504 | Gateway Timeout | 请求超时 |

---

## 7. 数据类型

### 7.1 区块相关

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

### 7.2 交易相关

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

### 7.3 节点相关

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

## 8. API 端点

### 8.1 区块链查询

#### GET /api/v1/blocks
- **描述**: 获取最新区块

**请求参数**:
| 参数 | 类型 | 必填 | 说明 |
|-----------|--------|----------|--------------|
| height | uint64 | 否 | 起始区块高度（可选，默认：最新） |
| count | uint64 | 否 | 返回区块数量（默认：10，最大：100） |

**响应** (成功):
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

**响应** (错误):
```json
{
  "success": false,
  "error": "ERR_BLOCK_NOT_FOUND",
  "message": "Block not found at height 100"
}
```

---

#### GET /api/v1/blocks/{hash}

- **描述**: 根据哈希获取区块

**响应**:
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

- **描述**: 获取最新区块（等同于 `/api/v1/blocks?height=latest` 的快捷方式）

**响应**: 同 `/api/v1/blocks?height=latest`

---

### 8.2 交易查询

#### GET /api/v1/transactions

- **描述**: 使用过滤器查询交易

**请求参数**:
| 参数 | 类型 | 必填 | 说明 |
|-----------|--------|----------|--------------|
| hash | string | 否 | 交易哈希 |
| type | uint8 | 否 | 交易类型（0=在线，1=离线等） |
| from | string | 否 | 发送方地址（16 字节，IPv6 格式） |
| to | string | 否 | 接收方地址 |
| limit | uint64 | 否 | 最大结果数（默认：100） |
| offset | uint64 | 否 | 分页偏移量 |

**响应**:
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

- **描述**: 根据哈希获取交易

**响应**:
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

- **描述**: 从交易池中获取待处理交易

**响应**:
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

- **描述**: 获取内存池交易

**响应**:
```json
{
  "success": true,
  "data": {
    "transactions": [...]
  }
}
```

---

### 8.3 账户查询

#### GET /api/v1/accounts/{address}

- **描述**: 根据 IPv6 地址获取账户信息

**响应**:
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

- **描述**: 获取账户余额

**响应**:
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

- **描述**: 获取地址的当前 nonce 值

**响应**:
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

### 8.4 节点查询

#### GET /api/v1/node/info

- **描述**: 获取当前节点信息

**响应**:
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

- **描述**: 获取已连接的节点列表

**响应**:
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

- **描述**: 获取详细节点统计信息

**响应**:
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

### 8.5 网络查询

#### GET /api/v1/network/status

- **描述**: 获取网络健康状态

**响应**:
```json
{
  "success": true,
  "data": {
    "totalPeers": 1250,
    "activePeers": 1180,
    "inactivePeers": 70,
    "averageUptime": "99.5%",
    "networkStatus": "healthy",
    "message": "所有系统正常运行"
  }
}
```

#### GET /api/v1/network/topology

- **描述**: 获取网络拓扑和关系

**响应**:
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

## 9. 写操作

### 9.1 POST /api/v1/transactions

- **描述**: 向网络广播新交易

**请求**:
```json
{
  "from": "2001:db8::1",
  "to": "2001:db8::2",
  "amount": 1000000000000",
  "fee": "1000",
  "data": "可选的交易数据"
}
```

**响应**:
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

- **描述**: 创建新账户（仅用于测试）

**请求**:
```json
{
  "password": "securepassword123"
}
```

---

## 10. WebSocket

### 10.1 WS /api/v1/ws

- **描述**: 用于实时更新的 WebSocket 端点

**端点**: `ws://localhost:38901/api/v1/ws`

**订阅**:
- `blocks/new` - 新区块通知
- `transactions/new` - 新交易通知
- `network/stats` - 网络统计更新

### 10.2 消息格式

**区块通知**:
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

**交易通知**:
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

## 11. 错误处理

### 11.1 错误响应格式

所有错误响应遵循以下格式：
```json
{
  "success": false,
  "error": "ERR_INVALID_TX_TYPE",
  "message": "无效的交易类型。必须为以下之一：0-7"
}
```

### 11.2 常见错误码

| 错误码 | HTTP 状态 | 用户操作 |
|---------|--------------|----------------------|
| 200 | 200 OK | 无需操作 |
| 400 | 400 Bad Request | 修正请求格式 |
| 401 | 401 Unauthorized | 添加认证 |
| 403 | 403 Forbidden | 检查权限 |
| 404 | 404 Not Found | 检查资源是否存在 |
| 409 | 409 Conflict | 解决冲突 |
| 500 | 500 Internal | 等待并重试 |

---

## 12. 认证

### 12.1 认证方式

**当前**: Basic Auth（仅用于开发）

**未来**: JWT 或 OAuth 2.0

**请求头**:
```
Authorization: Bearer <token>
```

### 12.2 Token 格式

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsIn...",
  "expires_in": 864,
  "permissions": ["read", "write"]
}
```

---

## 13. 速率限制

### 13.1 默认限制

| 资源类型 | 速率限制 |
|--------------|----------|
| 公共 API | 100 次/分钟 |
| WebSocket | 50 条消息/分钟 |
| 复杂查询 | 10 次/分钟 |

---

## 14. 代码示例

### 14.1 基本区块查询

```bash
curl -X GET "http://localhost:38901/api/v1/blocks?height=100"
```

**响应**:
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

### 14.2 根据哈希获取交易

```bash
curl -X GET "http://localhost:38901/api/v1/transactions/0xabcd..."
```

### 14.3 创建交易

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

## 15. WebSocket 示例

### 15.1 连接和订阅

```javascript
const ws = new WebSocket('ws://localhost:38901/api/v1/ws');

ws.onopen = () => {
  ws.send(JSON.stringify({
    "subscribe": ["blocks/new"]
  }));
};

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log(`新 ${data.type} 通知:`, data);
};
```

---

## 16. 依赖

### 16.1 必需模块

- `code/go/pkg/blockchain/chain.go` - 区块链数据结构
- `code/go/pkg/consensus/consensus.go` - 共识引擎
- `code/go/pkg/tx/pool.go` - 交易池
- `code/go/pkg/address/address.go` - 地址管理
- `code/go/pkg/state/` - 状态数据库接口
- `code/go/pkg/p2p/network.go` - P2P 网络

---

## 17. 安全考虑

### 17.1 输入验证

1. **地址格式**:
   - 必须是有效的 IPv6 地址（16 字节，IID 第 0 位 = 0）
   - 长度必须为 16 字节

2. **金额验证**:
   - 必须是 nano-V6 单位的正整数
   - 不得溢出

3. **签名验证**:
   - 必须是有效的 Ed25519 签名

### 17.2 速率限制

1. **全局限制**: 100 请求/秒
2. **单 IP 限制**: 1000 请求/分钟
3. **突发限制**: 10 请求/秒

### 17.3 CORS

1. **允许的源**: `*`（允许所有），`localhost:38901`（开发环境）
2. **方法**: `GET, POST, PUT, DELETE`
3. **请求头**: `Content-Type`, `Authorization`

---

## 18. 未来扩展

- **WebSocket 订阅**: 事件驱动的更新
- **GraphQL API**: 查询复杂数据结构
- **IPFS**: 去中心化文件存储
- **流式数据**: 大数据集检索

---

## 19. 版本历史

| 版本 | 日期 | 变更 |
|---------|------|---------|
| v1.0 | 2026-02-01 | 初始版本 |

---

## 20. 相关文档

- [V6Coin 白皮书（中文）](../whitepaper/V6Coin_Whitepaper_CN.md)
- [V6Coin 白皮书（英文）](../whitepaper/V6Coin_Whitepaper_EN.md)
- [交易验证规范](./04-transaction-spec.md)
- [区块链数据结构规范](./07-blockchain-spec.md)

---

**API 基础 URL**: `http://localhost:38901/api/v1`
