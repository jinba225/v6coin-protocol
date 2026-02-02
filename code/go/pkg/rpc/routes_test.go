package rpc

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSetupRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	SetupRoutes(router)

	routes := router.Routes()

	if len(routes) == 0 {
		t.Error("No routes registered")
	}

	expectedRoutes := map[string]bool{
		"GET /health":                               false,
		"GET /api/v1/block/:height":                 false,
		"GET /api/v1/block/hash/:hash":              false,
		"GET /api/v1/block/latest":                  false,
		"POST /api/v1/blocks":                       false,
		"GET /api/v1/transaction/:hash":             false,
		"POST /api/v1/transactions":                 false,
		"POST /api/v1/transaction/broadcast":        false,
		"GET /api/v1/transactions/address/:address": false,
		"GET /api/v1/account/:address":              false,
		"POST /api/v1/account/balance":              false,
		"POST /api/v1/account/nonce":                false,
		"POST /api/v1/account/create":               false,
		"POST /api/v1/node/info":                    false,
		"POST /api/v1/node/peers":                   false,
		"POST /api/v1/node/stats":                   false,
		"POST /api/v1/network/status":               false,
		"POST /api/v1/network/topology":             false,
	}

	for _, route := range routes {
		key := route.Method + " " + route.Path
		if _, exists := expectedRoutes[key]; exists {
			expectedRoutes[key] = true
		}
	}

	for route, found := range expectedRoutes {
		if !found {
			t.Errorf("Expected route '%s' not registered", route)
		}
	}
}

func TestHealthEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestCORSHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	SetupRoutes(router)

	tests := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/block/latest"},
		{http.MethodPost, "/api/v1/transactions"},
		{http.MethodOptions, "/api/v1/transaction/broadcast"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, tt.path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
		if allowOrigin != "*" {
			t.Errorf("CORS header not set for %s %s", tt.method, tt.path)
		}
	}
}
