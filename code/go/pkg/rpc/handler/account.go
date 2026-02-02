// Package handler provides HTTP handlers for the RPC server.
package handler

import (
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/jinba225/v6coin-protocol/pkg/address"
	"github.com/jinba225/v6coin-protocol/pkg/rpc/model"
)

// GetAccount handles GET /api/v1/account/:address
// Returns full account information including balance, nonce, and recent transactions.
func GetAccount(c *gin.Context) {
	var req model.GetAccountRequest
	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(
			model.CodeInvalidParameter,
			"Invalid address parameter",
		))
		return
	}

	// Validate IPv6 address format
	ip := net.ParseIP(req.Address)
	if ip == nil || ip.To4() != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(
			model.CodeInvalidAddress,
			"Invalid IPv6 address format",
		))
		return
	}

	// TODO: Query account from blockchain state service
	// account, err := blockchainService.GetAccount(req.Address)
	// if err != nil {
	// 	c.JSON(http.StatusNotFound, model.ErrorResponse(
	// 		model.CodeAccountNotFound,
	// 		"Account not found",
	// 	))
	// 	return
	// }

	// Mock response for now
	account := &model.AccountResponse{
		Address:      req.Address,
		ID:           address.ExtractIID(ip),
		Balance:      1000000000, // 1 V6 in nano-V6
		Nonce:        0,
		TxCount:      0,
		Transactions: []model.TxInfo{},
	}

	c.JSON(http.StatusOK, model.SuccessResponse(account))
}

// GetBalance handles POST /api/v1/account/balance
// Returns account balance with available and locked amounts.
func GetBalance(c *gin.Context) {
	var req model.GetAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(
			model.CodeInvalidParameter,
			"Invalid request parameters",
		))
		return
	}

	// Validate IPv6 address format
	ip := net.ParseIP(req.Address)
	if ip == nil || ip.To4() != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(
			model.CodeInvalidAddress,
			"Invalid IPv6 address format",
		))
		return
	}

	// TODO: Query balance from blockchain state service
	// balance, err := blockchainService.GetBalance(req.Address)
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, model.ErrorResponse(
	// 		model.CodeInternalServer,
	// 		"Failed to query balance",
	// 	))
	// 	return
	// }

	// Mock response: 1.5 V6 = 1500000000 nano-V6
	balance := &model.BalanceResponse{
		Address: req.Address,
		Balance: 1500000000,
	}

	c.JSON(http.StatusOK, model.SuccessResponse(balance))
}

// GetNonce handles POST /api/v1/account/nonce
// Returns account nonce (transaction counter) for building new transactions.
// Used to prevent replay attacks.
func GetNonce(c *gin.Context) {
	var req model.GetAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(
			model.CodeInvalidParameter,
			"Invalid request parameters",
		))
		return
	}

	// Validate IPv6 address format
	ip := net.ParseIP(req.Address)
	if ip == nil || ip.To4() != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(
			model.CodeInvalidAddress,
			"Invalid IPv6 address format",
		))
		return
	}

	// TODO: Query nonce from blockchain state service
	// nonce, err := blockchainService.GetNonce(req.Address)
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, model.ErrorResponse(
	// 		model.CodeInternalServer,
	// 		"Failed to query nonce",
	// 	))
	// 	return
	// }

	// Mock response
	nonce := &model.NonceResponse{
		Address: req.Address,
		Nonce:   0,
	}

	c.JSON(http.StatusOK, model.SuccessResponse(nonce))
}

// CreateAccount handles POST /api/v1/account/create
// Creates a new account with generated key pair and CGA address.
// WARNING: This is for testing only. Never return passphrase or private key in production.
func CreateAccount(c *gin.Context) {
	var req model.CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(
			model.CodeInvalidParameter,
			"Invalid request parameters",
		))
		return
	}

	// TODO: Generate key pair and CGA address using address service
	// privateKey, err := crypto.GeneratePrivateKey()
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, model.ErrorResponse(
	// 		model.CodeInternalServer,
	// 		"Failed to generate private key",
	// 	))
	// 	return
	// }
	//
	// // Encrypt private key with password
	// encryptedKey, err := crypto.Encrypt(privateKey, req.Password)
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, model.ErrorResponse(
	// 		model.CodeInternalServer,
	// 		"Failed to encrypt private key",
	// 	))
	// 	return
	// }
	//
	// // Generate CGA address
	// networkPrefix := net.IP{0x20, 0x01, 0x0d, 0xb8, 0x00, 0x00, 0x00, 0x00}
	// v6Addr, err := address.GenerateAddress(networkPrefix, privateKey)
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, model.ErrorResponse(
	// 		model.CodeInternalServer,
	// 		"Failed to generate CGA address",
	// 	))
	// 	return
	// }

	// Mock response: Generate a sample CGA address
	networkPrefix := net.IP{0x20, 0x01, 0x0d, 0xb8, 0x00, 0x00, 0x00, 0x00}
	mockIID := uint64(time.Now().UnixNano()) &^ 1 // Ensure unicast address

	fullAddr := make(net.IP, 16)
	copy(fullAddr, networkPrefix)
	// Mock IID
	for i := 0; i < 8; i++ {
		fullAddr[8+i] = byte(mockIID >> (56 - i*8))
	}

	// In production, NEVER return password or passphrase
	response := &model.CreateAccountResponse{
		Address: fullAddr.String(),
	}

	c.JSON(http.StatusOK, model.SuccessResponse(response))
}
