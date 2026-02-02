// Package rpc provides route configuration for the RPC server.
package rpc

import (
	"github.com/gin-gonic/gin"
	"github.com/jinba225/v6coin-protocol/pkg/rpc/handler"
	"github.com/jinba225/v6coin-protocol/pkg/rpc/middleware"
	"github.com/jinba225/v6coin-protocol/pkg/rpc/model"
)

// SetupRoutes configures all API routes for the RPC server.
// It applies middleware and registers all handler functions.
func SetupRoutes(router *gin.Engine) {
	// Apply middleware in order:
	// 1. Recovery - recover from panics
	// 2. RequestID - add unique request ID to each request
	// 3. CORS - handle cross-origin requests
	// 4. Logger - log HTTP requests and responses
	router.Use(middleware.Recovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.CORS())
	router.Use(middleware.Logger())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, model.SuccessResponse("OK"))
	})

	// API v1 group
	v1 := router.Group("/api/v1")
	{
		// Block routes
		v1.GET("/block/:height", handler.GetBlock)
		v1.GET("/block/hash/:hash", handler.GetBlockByHash)
		v1.GET("/block/latest", handler.GetLatestBlock)
		v1.POST("/blocks", handler.GetBlocks)

		// Transaction routes
		v1.GET("/transaction/:hash", handler.GetTransaction)
		v1.POST("/transactions", handler.GetTransactions)
		v1.POST("/transaction/broadcast", handler.BroadcastTransaction)
		v1.GET("/transactions/address/:address", handler.GetTransactionsByAddress)

		// Account routes
		v1.GET("/account/:address", handler.GetAccount)
		v1.POST("/account/balance", handler.GetBalance)
		v1.POST("/account/nonce", handler.GetNonce)
		v1.POST("/account/create", handler.CreateAccount)

		// Node routes
		v1.POST("/node/info", handler.GetNodeInfo)
		v1.POST("/node/peers", handler.GetNodePeers)
		v1.POST("/node/stats", handler.GetNodeStats)

		// Network routes
		v1.POST("/network/status", handler.GetNetworkStatus)
		v1.POST("/network/topology", handler.GetNetworkTopology)
	}
}
