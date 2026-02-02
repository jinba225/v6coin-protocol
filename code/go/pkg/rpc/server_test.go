package rpc

import (
	"context"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Host != "0.0.0.0" {
		t.Errorf("DefaultConfig().Host = %v, want 0.0.0.0", config.Host)
	}
	if config.Port != 9090 {
		t.Errorf("DefaultConfig().Port = %v, want 9090", config.Port)
	}
	if config.Mode != "release" {
		t.Errorf("DefaultConfig().Mode = %v, want release", config.Mode)
	}
	if config.ReadTimeout != 10*time.Second {
		t.Errorf("DefaultConfig().ReadTimeout = %v, want 10s", config.ReadTimeout)
	}
	if config.WriteTimeout != 10*time.Second {
		t.Errorf("DefaultConfig().WriteTimeout = %v, want 10s", config.WriteTimeout)
	}
}

func TestNewServer(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
	}{
		{
			name:   "nil config uses defaults",
			config: nil,
		},
		{
			name: "custom config",
			config: &Config{
				Host:         "127.0.0.1",
				Port:         8080,
				Mode:         "test",
				ReadTimeout:  5 * time.Second,
				WriteTimeout: 5 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer(tt.config)

			if server == nil {
				t.Fatal("NewServer() returned nil")
			}

			if server.httpAddr == "" {
				t.Error("Server httpAddr should not be empty")
			}

			if server.router == nil {
				t.Error("Server router should not be nil")
			}
		})
	}
}

func TestServerRouter(t *testing.T) {
	config := &Config{
		Mode: "test",
	}
	server := NewServer(config)

	router := server.Router()
	if router == nil {
		t.Error("Router() returned nil")
	}

	if router != server.router {
		t.Error("Router() should return the same router instance")
	}
}

func TestServerConfig(t *testing.T) {
	tests := []struct {
		name         string
		mode         string
		expectedMode string
	}{
		{
			name:         "debug mode",
			mode:         "debug",
			expectedMode: gin.DebugMode,
		},
		{
			name:         "release mode",
			mode:         "release",
			expectedMode: gin.ReleaseMode,
		},
		{
			name:         "test mode",
			mode:         "test",
			expectedMode: gin.TestMode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Mode: tt.mode,
			}
			NewServer(config)

			if gin.Mode() != tt.expectedMode {
				t.Errorf("Expected Gin mode %s, got %s", tt.expectedMode, gin.Mode())
			}
		})
	}
}

func TestServerAddress(t *testing.T) {
	tests := []struct {
		name         string
		host         string
		port         int
		expectedAddr string
	}{
		{
			name:         "default address",
			host:         "0.0.0.0",
			port:         9090,
			expectedAddr: "0.0.0.0:9090",
		},
		{
			name:         "localhost",
			host:         "127.0.0.1",
			port:         8080,
			expectedAddr: "127.0.0.1:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Host: tt.host,
				Port: tt.port,
				Mode: "test",
			}
			server := NewServer(config)

			if server.httpAddr != tt.expectedAddr {
				t.Errorf("Expected address %s, got %s", tt.expectedAddr, server.httpAddr)
			}
		})
	}
}

func TestServerStart(t *testing.T) {
	config := &Config{
		Host: "127.0.0.1",
		Port: 9091,
		Mode: "test",
	}
	server := NewServer(config)

	ctx, cancel := context.WithCancel(context.Background())

	errChan := make(chan error, 1)
	go func() {
		errChan <- server.Start(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	cancel()

	select {
	case err := <-errChan:
		if err != nil {
			t.Errorf("Server.Start() returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Server.Start() did not return in time")
	}
}
