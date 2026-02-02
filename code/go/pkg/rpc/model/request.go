// Package model provides data models for RPC API.
package model

// GetBlockRequest is the request for getting blocks.
type GetBlockRequest struct {
	Height *uint64 `form:"height" json:"height,omitempty"` // Starting block height (nil for latest)
	Count  *uint64 `form:"count" json:"count,omitempty"`   // Number of blocks to return (max 100)
}

// GetTransactionRequest is the request for querying transactions.
type GetTransactionRequest struct {
	Hash   *string `form:"hash" json:"hash,omitempty"`     // Transaction hash
	Type   *uint8  `form:"type" json:"type,omitempty"`     // Transaction type
	From   *string `form:"from" json:"from,omitempty"`     // Sender address (IPv6 format)
	To     *string `form:"to" json:"to,omitempty"`         // Receiver address (IPv6 format)
	Limit  *uint64 `form:"limit" json:"limit,omitempty"`   // Maximum number of results
	Offset *uint64 `form:"offset" json:"offset,omitempty"` // Pagination offset
}

// GetAccountRequest is the request for getting account information.
type GetAccountRequest struct {
	Address string `uri:"address" json:"address"` // IPv6 address (required)
}

// GetNodeInfoRequest is the request for getting current node information.
type GetNodeInfoRequest struct{}

// GetNodePeersRequest is the request for getting connected peers.
type GetNodePeersRequest struct{}

// GetNodeStatsRequest is the request for getting detailed node statistics.
type GetNodeStatsRequest struct{}

// GetNetworkStatusRequest is the request for getting network health status.
type GetNetworkStatusRequest struct{}

// GetNetworkTopologyRequest is the request for getting network topology.
type GetNetworkTopologyRequest struct{}

// BroadcastTransactionRequest is the request for broadcasting a new transaction.
type BroadcastTransactionRequest struct {
	From   string `json:"from" binding:"required"`   // Sender address (IPv6 format)
	To     string `json:"to" binding:"required"`     // Receiver address (IPv6 format)
	Amount uint64 `json:"amount" binding:"required"` // Amount in nano-V6
	Fee    uint64 `json:"fee"`                       // Transaction fee
	Data   string `json:"data,omitempty"`            // Optional transaction data
}

// CreateAccountRequest is the request for creating a new account (testing only).
type CreateAccountRequest struct {
	Password string `json:"password" binding:"required"` // Password for encryption
}
