// Package handler provides HTTP handlers for the RPC server.
package handler

import (
	"encoding/hex"
	"errors"
	"net"

	"github.com/gin-gonic/gin"
	"github.com/jinba225/v6coin-protocol/pkg/consensus"
	"github.com/jinba225/v6coin-protocol/pkg/rpc/model"
)

var (
	ErrInvalidParameter = errors.New("invalid parameter")
)

// BlockchainService interface for blockchain operations
type BlockchainService interface {
	GetBlock(hash []byte) *consensus.Block
	GetBlockByHeight(height uint64) []*consensus.Block
	GetChainHead() *consensus.Block
	GetCurrentHeight() uint64
}

// blockchainService is the global blockchain service instance
// This will be initialized when the server starts
var blockchainService BlockchainService

// SetBlockchainService sets the global blockchain service instance
func SetBlockchainService(bc BlockchainService) {
	blockchainService = bc
}

// convertBlockToResponse converts consensus.Block to model.BlockResponse
func convertBlockToResponse(block *consensus.Block) *model.BlockResponse {
	if block == nil {
		return nil
	}

	// Convert transactions
	txs := make([]model.TxInfo, 0, len(block.Transactions))
	for _, tx := range block.Transactions {
		txInfo := model.TxInfo{
			Hash:      hex.EncodeToString(tx.Hash()),
			Type:      uint8(tx.Type),
			From:      ipv6ToString(tx.From),
			To:        ipv6ToString(tx.To),
			Amount:    tx.Amount,
			Fee:       tx.Fee,
			Nonce:     tx.Nonce,
			Timestamp: tx.Timestamp,
			Signature: hex.EncodeToString(tx.Signature),
		}
		if len(tx.Data) > 0 {
			txInfo.Data = hex.EncodeToString(tx.Data)
		}
		txs = append(txs, txInfo)
	}

	return &model.BlockResponse{
		Height:       block.Header.Height,
		Hash:         hex.EncodeToString(block.Header.Hash()),
		PrevHash:     hex.EncodeToString(block.Header.PrevBlockHash),
		MerkleRoot:   hex.EncodeToString(block.Header.MerkleRoot),
		Timestamp:    block.Header.Timestamp,
		ValidatorID:  string(block.Header.ValidatorID),
		StateRoot:    hex.EncodeToString(block.Header.StateRoot),
		Transactions: txs,
		Signature:    hex.EncodeToString(block.Header.Signature),
	}
}

// ipv6ToString converts net.IP to string representation
func ipv6ToString(ip net.IP) string {
	if ip == nil {
		return ""
	}
	return ip.String()
}

// GetBlock retrieves a block by height
// GET /api/v1/block/:height
func GetBlock(c *gin.Context) {
	height, ok := c.Params.Get("height")
	if !ok {
		c.JSON(400, model.ErrorResponse(
			model.CodeMissingParameter,
			"Block height parameter is required",
		))
		return
	}

	// Parse height
	blockHeight, err := uint64FromString(height)
	if err != nil {
		c.JSON(400, model.ErrorResponse(
			model.CodeInvalidParameter,
			"Invalid block height format",
		))
		return
	}

	// Query blockchain service for block
	blocks := blockchainService.GetBlockByHeight(blockHeight)
	if len(blocks) == 0 {
		c.JSON(404, model.ErrorResponse(
			model.CodeBlockNotFound,
			"Block not found at specified height",
		))
		return
	}

	// Return the first block (main chain block)
	block := blocks[0]
	blockResponse := convertBlockToResponse(block)

	c.JSON(200, model.SuccessResponse(blockResponse))
}

// GetBlockByHash retrieves a block by hash
// GET /api/v1/block/hash/:hash
func GetBlockByHash(c *gin.Context) {
	hash, ok := c.Params.Get("hash")
	if !ok {
		c.JSON(400, model.ErrorResponse(
			model.CodeMissingParameter,
			"Block hash parameter is required",
		))
		return
	}

	// Decode hash
	blockHash, err := hex.DecodeString(hash)
	if err != nil {
		c.JSON(400, model.ErrorResponse(
			model.CodeInvalidParameter,
			"Invalid block hash format",
		))
		return
	}

	// Query blockchain service for block
	block := blockchainService.GetBlock(blockHash)
	if block == nil {
		c.JSON(404, model.ErrorResponse(
			model.CodeBlockNotFound,
			"Block not found",
		))
		return
	}

	blockResponse := convertBlockToResponse(block)
	c.JSON(200, model.SuccessResponse(blockResponse))
}

// GetLatestBlock retrieves the latest (chain head) block
// GET /api/v1/block/latest
func GetLatestBlock(c *gin.Context) {
	block := blockchainService.GetChainHead()
	if block == nil {
		c.JSON(500, model.ErrorResponse(
			model.CodeServiceUnavailable,
			"Blockchain service unavailable",
		))
		return
	}

	blockResponse := convertBlockToResponse(block)
	c.JSON(200, model.SuccessResponse(blockResponse))
}

// GetBlocks retrieves multiple blocks with pagination
// POST /api/v1/blocks
func GetBlocks(c *gin.Context) {
	var req model.GetBlockRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, model.ErrorResponse(
			model.CodeInvalidRequest,
			"Invalid request format",
		))
		return
	}

	// Set default values
	var startHeight uint64
	if req.Height != nil {
		startHeight = *req.Height
	} else {
		// If no height specified, start from genesis (0)
		startHeight = 0
	}

	count := uint64(20) // Default count
	if req.Count != nil {
		count = *req.Count
		// Limit count to max 100
		if count > 100 {
			count = 100
		}
	}

	// Get current height to calculate pagination
	currentHeight := blockchainService.GetCurrentHeight()
	if startHeight > currentHeight {
		c.JSON(400, model.ErrorResponse(
			model.CodeInvalidBlockHeight,
			"Start height exceeds current blockchain height",
		))
		return
	}

	// Calculate end height
	endHeight := startHeight + count - 1
	if endHeight > currentHeight {
		endHeight = currentHeight
	}

	// Fetch blocks
	blocks := make([]model.BlockResponse, 0, endHeight-startHeight+1)
	for height := startHeight; height <= endHeight; height++ {
		heightBlocks := blockchainService.GetBlockByHeight(height)
		if len(heightBlocks) > 0 {
			// Return the first block (main chain block)
			block := heightBlocks[0]
			blockResponse := convertBlockToResponse(block)
			blocks = append(blocks, *blockResponse)
		}
	}

	// Calculate pagination
	totalBlocks := currentHeight - startHeight + 1
	pageSize := len(blocks)
	totalPages := int(totalBlocks) / pageSize
	if int(totalBlocks)%pageSize != 0 {
		totalPages++
	}

	response := model.BlocksResponse{
		Blocks: blocks,
		Pagination: model.Pagination{
			Page:       1,
			PageSize:   pageSize,
			TotalPages: totalPages,
		},
	}

	c.JSON(200, model.SuccessResponse(response))
}

// uint64FromString converts a string to uint64
func uint64FromString(s string) (uint64, error) {
	var result uint64
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, ErrInvalidParameter
		}
		result = result*10 + uint64(s[i]-'0')
	}
	return result, nil
}
