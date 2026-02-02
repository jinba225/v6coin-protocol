// Package model provides data models for RPC API.
package model

// BlockResponse represents block information.
type BlockResponse struct {
	Height       uint64   `json:"height"`       // Block height
	Hash         string   `json:"hash"`         // Block hash (hex encoded)
	PrevHash     string   `json:"prevHash"`     // Previous block hash (hex encoded)
	MerkleRoot   string   `json:"merkleRoot"`   // Merkle root of transactions
	Timestamp    uint64   `json:"timestamp"`    // Block timestamp (Unix timestamp)
	ValidatorID  string   `json:"validatorId"`  // Validator node ID
	StateRoot    string   `json:"stateRoot"`    // State root (hex encoded)
	Transactions []TxInfo `json:"transactions"` // Transactions in this block
	Signature    string   `json:"signature"`    // Block signature (hex encoded)
}

// BlocksResponse represents response for multiple blocks.
type BlocksResponse struct {
	Blocks     []BlockResponse `json:"blocks"`     // List of blocks
	Pagination Pagination      `json:"pagination"` // Pagination metadata
}

// TxInfo represents transaction information.
type TxInfo struct {
	Hash      string `json:"hash"`           // Transaction hash (hex encoded)
	Type      uint8  `json:"type"`           // Transaction type
	From      string `json:"from"`           // Sender address (IPv6 format)
	To        string `json:"to"`             // Receiver address (IPv6 format)
	Amount    uint64 `json:"amount"`         // Amount in nano-V6
	Fee       uint64 `json:"fee"`            // Transaction fee in nano-V6
	Nonce     uint64 `json:"nonce"`          // Nonce value
	Timestamp uint64 `json:"timestamp"`      // Transaction timestamp (Unix timestamp)
	Signature string `json:"signature"`      // Signature (hex encoded)
	Data      string `json:"data,omitempty"` // Optional transaction data (hex encoded)
}

// TransactionResponse represents response for single transaction.
type TransactionResponse struct {
	TxInfo `json:",inline"` // Inline transaction information
	Status string           `json:"status"` // Transaction status (pending, confirmed, etc.)
}

// TransactionsResponse represents response for multiple transactions.
type TransactionsResponse struct {
	Transactions []TxInfo   `json:"transactions"` // List of transactions
	Pagination   Pagination `json:"pagination"`   // Pagination metadata
}

// AccountResponse represents account information.
type AccountResponse struct {
	Address      string   `json:"address"`      // IPv6 address
	ID           uint64   `json:"id"`           // Account ID (IID)
	Balance      uint64   `json:"balance"`      // Balance in nano-V6
	Nonce        uint64   `json:"nonce"`        // Current nonce value
	TxCount      uint64   `json:"txCount"`      // Number of transactions
	Transactions []TxInfo `json:"transactions"` // List of recent transactions
}

// BalanceResponse represents account balance.
type BalanceResponse struct {
	Address string `json:"address"` // IPv6 address
	Balance uint64 `json:"balance"` // Balance in nano-V6
}

// NonceResponse represents account nonce.
type NonceResponse struct {
	Address string `json:"address"` // IPv6 address
	Nonce   uint64 `json:"nonce"`   // Current nonce value
}

// NodeInfoResponse represents current node information.
type NodeInfoResponse struct {
	ID        string    `json:"id"`        // Node ID
	Address   string    `json:"address"`   // Node address (IPv6 format)
	Height    uint64    `json:"height"`    // Current block height
	Peers     uint64    `json:"peers"`     // Number of connected peers
	Syncing   bool      `json:"syncing"`   // Whether node is syncing
	ChainHead BlockHead `json:"chainHead"` // Chain head information
}

// BlockHead represents chain head information.
type BlockHead struct {
	Height    uint64 `json:"height"`    // Block height
	Hash      string `json:"hash"`      // Block hash (hex encoded)
	Timestamp uint64 `json:"timestamp"` // Block timestamp (Unix timestamp)
}

// PeerInfo represents peer node information.
type PeerInfo struct {
	ID         string  `json:"id"`         // Peer node ID
	Address    string  `json:"address"`    // Peer address (IPv6 format)
	Height     uint64  `json:"height"`     // Peer's current block height
	OnlineTime uint64  `json:"onlineTime"` // Online time in seconds
	LastOnline string  `json:"lastOnline"` // Last online time (ISO 8601 format)
	PacketLoss float64 `json:"packetLoss"` // Packet loss rate (0.0 - 1.0)
	Forwarded  uint64  `json:"forwarded"`  // Total forwarded bytes
	Score      float64 `json:"score"`      // Contribution score (0.0 - 1.0)
}

// PeersResponse represents response for peer list.
type PeersResponse struct {
	Peers []PeerInfo `json:"peers"` // List of connected peers
}

// NodeStatsResponse represents detailed node statistics.
type NodeStatsResponse struct {
	Uptime            string  `json:"uptime"`            // Uptime percentage
	UptimePercentage  float64 `json:"uptimePercentage"`  // Uptime as decimal (0.0 - 1.0)
	TotalForwarded    uint64  `json:"totalForwarded"`    // Total bytes forwarded
	TotalReceived     uint64  `json:"totalReceived"`     // Total bytes received
	AverageLatency    string  `json:"averageLatency"`    // Average latency
	PacketLoss        float64 `json:"packetLoss"`        // Packet loss rate (0.0 - 1.0)
	Connections       uint64  `json:"connections"`       // Number of active connections
	ActiveValidators  uint64  `json:"activeValidators"`  // Number of active validators
	ContributionScore float64 `json:"contributionScore"` // Contribution score (0.0 - 1.0)
}

// NetworkStatusResponse represents network health status.
type NetworkStatusResponse struct {
	TotalPeers    uint64 `json:"totalPeers"`    // Total number of peers
	ActivePeers   uint64 `json:"activePeers"`   // Number of active peers
	InactivePeers uint64 `json:"inactivePeers"` // Number of inactive peers
	AverageUptime string `json:"averageUptime"` // Average uptime percentage
	NetworkStatus string `json:"networkStatus"` // Network status (healthy, degraded, etc.)
	Message       string `json:"message"`       // Status message
}

// NetworkNode represents a node in network topology.
type NetworkNode struct {
	ID    string     `json:"id"`    // Node ID
	Peers []NodePeer `json:"peers"` // List of connected peers
}

// NodePeer represents a peer connection in topology.
type NodePeer struct {
	ID      string `json:"id"`      // Peer node ID
	Address string `json:"address"` // Peer address (IPv6 format)
	Height  uint64 `json:"height"`  // Peer's block height
}

// NetworkEdge represents an edge in network topology.
type NetworkEdge struct {
	From   string  `json:"from"`   // Source node ID
	To     string  `json:"to"`     // Destination node ID
	Weight float64 `json:"weight"` // Edge weight
}

// NetworkTopologyResponse represents network topology.
type NetworkTopologyResponse struct {
	Nodes []NetworkNode `json:"nodes"` // Network nodes
	Edges []NetworkEdge `json:"edges"` // Network edges
}

// BroadcastTxResponse represents response for broadcasting a transaction.
type BroadcastTxResponse struct {
	Hash          string `json:"hash"`          // Transaction hash (hex encoded)
	Timestamp     uint64 `json:"timestamp"`     // Broadcast timestamp (Unix timestamp)
	Status        string `json:"status"`        // Transaction status (pending, etc.)
	Confirmations uint64 `json:"confirmations"` // Number of confirmations
}

// CreateAccountResponse represents response for creating an account.
type CreateAccountResponse struct {
	Address string `json:"address"` // Generated IPv6 address
}
