// Package handler provides HTTP handlers for the RPC server.
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/jinba225/v6coin-protocol/pkg/rpc/model"
)

// GetTransaction handles request to get transaction by hash
// GET /api/v1/transaction/:hash
func GetTransaction(c *gin.Context) {
	hash := c.Param("hash")
	if hash == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(
			model.CodeInvalidParameter,
			"Transaction hash required",
		))
		return
	}

	// TODO: Query transaction from service
	// tx, err := blockchainService.GetTransaction(hash)
	// if err != nil {
	//     c.JSON(http.StatusNotFound, model.ErrorResponse(
	//         model.CodeTransactionNotFound,
	//         "Transaction not found",
	//     ))
	//     return
	// }

	// Mock response for now
	tx := &model.TransactionResponse{
		TxInfo: model.TxInfo{
			Hash:      hash,
			Type:      1,
			From:      "2001:db8::1",
			To:        "2001:db8::2",
			Amount:    1000000000,
			Fee:       10000,
			Nonce:     1,
			Timestamp: 1234567890,
			Signature: "signature_placeholder",
		},
		Status: "confirmed",
	}

	c.JSON(http.StatusOK, model.SuccessResponse(tx))
}

// GetTransactions handles batch query for transactions
// Supports filtering by block height range, time range, address
// Supports pagination
// POST /api/v1/transactions
func GetTransactions(c *gin.Context) {
	var req model.GetTransactionRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(
			model.CodeInvalidParameter,
			"Invalid request parameters",
		))
		return
	}

	// Set default pagination values
	limit := uint64(20)
	offset := uint64(0)
	if req.Limit != nil {
		if *req.Limit > 100 {
			limit = 100
		} else {
			limit = *req.Limit
		}
	}
	if req.Offset != nil {
		offset = *req.Offset
	}

	// TODO: Query transactions from service with filters
	// transactions, total, err := blockchainService.GetTransactions(
	//     req.Hash, req.Type, req.From, req.To,
	//     limit, offset,
	// )
	// if err != nil {
	//     c.JSON(http.StatusInternalServerError, model.ErrorResponse(
	//         model.CodeInternalServer, // Note: This error code doesn't exist, need to handle
	//         "Failed to query transactions",
	//     ))
	//     return
	// }

	// Mock response for now
	transactions := []model.TxInfo{
		{
			Hash:      "tx1",
			Type:      1,
			From:      "2001:db8::1",
			To:        "2001:db8::2",
			Amount:    1000000000,
			Fee:       10000,
			Nonce:     1,
			Timestamp: 1234567890,
			Signature: "signature1",
		},
		{
			Hash:      "tx2",
			Type:      1,
			From:      "2001:db8::2",
			To:        "2001:db8::3",
			Amount:    2000000000,
			Fee:       15000,
			Nonce:     2,
			Timestamp: 1234567891,
			Signature: "signature2",
		},
	}

	total := int64(len(transactions))
	totalPages := int((total + int64(limit) - 1) / int64(limit))
	if totalPages < 1 {
		totalPages = 1
	}

	response := model.TransactionsResponse{
		Transactions: transactions,
		Pagination: model.Pagination{
			Page:       int(offset/limit) + 1,
			PageSize:   int(limit),
			TotalPages: totalPages,
		},
	}

	c.JSON(http.StatusOK, model.SuccessResponse(response))
}

// BroadcastTransaction handles request to broadcast a transaction to the network
// POST /api/v1/transaction/broadcast
func BroadcastTransaction(c *gin.Context) {
	var req model.BroadcastTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(
			model.CodeInvalidParameter,
			"Invalid transaction data",
		))
		return
	}

	// TODO: Validate and broadcast transaction via P2P service
	// txHash, err := p2pService.BroadcastTransaction(req.From, req.To, req.Amount, req.Fee, req.Data)
	// if err != nil {
	//     c.JSON(http.StatusInternalServerError, model.ErrorResponse(
	//         model.CodeNetworkError, // Using existing network error code
	//         "Failed to broadcast transaction",
	//     ))
	//     return
	// }

	// Mock transaction hash for now
	txHash := "mock_tx_hash_" + req.From[:8]

	response := model.BroadcastTxResponse{
		Hash:          txHash,
		Timestamp:     1234567890,
		Status:        "pending",
		Confirmations: 0,
	}

	c.JSON(http.StatusOK, model.SuccessResponse(response))
}

// GetTransactionsByAddress handles request to query transaction history by address
// Supports pagination
// GET /api/v1/transactions/address/:address
func GetTransactionsByAddress(c *gin.Context) {
	address := c.Param("address")
	if address == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(
			model.CodeMissingParameter,
			"Address required",
		))
		return
	}

	// Get pagination parameters
	var req struct {
		Limit  *uint64 `form:"limit"`
		Offset *uint64 `form:"offset"`
	}
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(
			model.CodeInvalidParameter,
			"Invalid pagination parameters",
		))
		return
	}

	// Set default pagination values
	limit := uint64(20)
	offset := uint64(0)
	if req.Limit != nil {
		if *req.Limit > 100 {
			limit = 100
		} else {
			limit = *req.Limit
		}
	}
	if req.Offset != nil {
		offset = *req.Offset
	}

	// TODO: Query transactions by address from service
	// transactions, total, err := blockchainService.GetTransactionsByAddress(
	//     address, limit, offset,
	// )
	// if err != nil {
	//     c.JSON(http.StatusInternalServerError, model.ErrorResponse(
	//         model.CodeInternalServer, // Note: This error code doesn't exist, need to handle
	//         "Failed to query transactions",
	//     ))
	//     return
	// }

	// Mock response for now
	transactions := []model.TxInfo{
		{
			Hash:      "tx1",
			Type:      1,
			From:      address,
			To:        "2001:db8::2",
			Amount:    1000000000,
			Fee:       10000,
			Nonce:     1,
			Timestamp: 1234567890,
			Signature: "signature1",
		},
		{
			Hash:      "tx3",
			Type:      1,
			From:      "2001:db8::3",
			To:        address,
			Amount:    500000000,
			Fee:       8000,
			Nonce:     3,
			Timestamp: 1234567892,
			Signature: "signature3",
		},
	}

	total := int64(len(transactions))
	totalPages := int((total + int64(limit) - 1) / int64(limit))
	if totalPages < 1 {
		totalPages = 1
	}

	response := model.TransactionsResponse{
		Transactions: transactions,
		Pagination: model.Pagination{
			Page:       int(offset/limit) + 1,
			PageSize:   int(limit),
			TotalPages: totalPages,
		},
	}

	c.JSON(http.StatusOK, model.SuccessResponse(response))
}
