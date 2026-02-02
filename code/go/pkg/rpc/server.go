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
	httpAddr string
}

// Config holds the server configuration.
type Config struct {
	Host         string
	Port         int
	Mode         string // debug, release, test
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// DefaultConfig returns a default server configuration.
func DefaultConfig() *Config {
	return &Config{
		Host:         "0.0.0.0",
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

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	server := &Server{
		router:   router,
		httpAddr: addr,
	}

	SetupRoutes(router)

	return server
}

// Start starts the RPC server.
func (s *Server) Start(ctx context.Context) error {
	httpServer := &http.Server{
		Addr:         s.httpAddr,
		Handler:      s.router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(fmt.Sprintf("Failed to start RPC server: %v", err))
		}
	}()

	<-ctx.Done()

	// Graceful shutdown with a 5 second timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	return nil
}

// Router returns the Gin router for external registration of routes.
func (s *Server) Router() *gin.Engine {
	return s.router
}
