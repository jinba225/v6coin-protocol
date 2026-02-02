// Package rpc provides the RPC API server for V6Coin Protocol.
package rpc

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Server represents the RPC server.
type Server struct {
	router   *gin.Engine
	httpPort int
}

// Config holds the server configuration.
type Config struct {
	Port         int
	Mode         string // debug, release, test
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// DefaultConfig returns a default server configuration.
func DefaultConfig() *Config {
	return &Config{
		Port:         9090,
		Mode:         "release",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
}

// NewServer creates a new RPC server instance.
func NewServer(cfg *Config) *Server {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Set Gin mode based on configuration
	gin.SetMode(cfg.Mode)

	router := gin.New()

	// Add recovery middleware to recover from panics
	router.Use(gin.Recovery())

	server := &Server{
		router:   router,
		httpPort: cfg.Port,
	}

	server.setupRoutes()

	return server
}

// setupRoutes configures all API routes.
func (s *Server) setupRoutes() {
	// Health check endpoint
	s.router.GET("/health", s.healthCheck)

	// API v1 routes
	_ = s.router.Group("/api/v1")
	// TODO: Add routes for blockchain, node, and network operations
	// v1 := s.router.Group("/api/v1")
	// {
	// 	// Blockchain routes
	// 	blockchain := v1.Group("/blockchain")
	// 	{
	// 		blockchain.GET("/block/:height", h.GetBlock)
	// 		blockchain.GET("/block/latest", h.GetLatestBlock)
	// 		blockchain.GET("/transaction/:hash", h.GetTransaction)
	// 	}
	//
	// 	// Node routes
	// 	node := v1.Group("/node")
	// 	{
	// 		node.GET("/info", h.GetNodeInfo)
	// 		node.GET("/peers", h.GetPeers)
	// 	}
	//
	// 	// Network routes
	// 	network := v1.Group("/network")
	// 	{
	// 		network.GET("/stats", h.GetNetworkStats)
	// 	}
	// }
}

// healthCheck returns the server health status.
func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "v6coin-rpc",
		"time":    time.Now().UTC().Format(time.RFC3339),
	})
}

// Start starts the RPC server.
func (s *Server) Start(ctx context.Context) error {
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", s.httpPort),
		Handler:      s.router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(fmt.Sprintf("Failed to start RPC server: %v", err))
		}
	}()

	<-ctx.Done()

	// Graceful shutdown with a 5 second timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	return nil
}

// Router returns the Gin router for external registration of routes.
func (s *Server) Router() *gin.Engine {
	return s.router
}
