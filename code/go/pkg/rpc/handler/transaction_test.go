package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router
}

func TestGetTransaction(t *testing.T) {
	tests := []struct {
		name           string
		hashParam      string
		expectedStatus int
	}{
		{
			name:           "missing hash parameter",
			hashParam:      "",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "successful transaction retrieval",
			hashParam:      "tx123",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupTestRouter()
			router.GET("/transaction/:hash", GetTransaction)

			var req *http.Request
			if tt.hashParam != "" {
				req = httptest.NewRequest(http.MethodGet, "/transaction/"+tt.hashParam, nil)
			} else {
				req = httptest.NewRequest(http.MethodGet, "/transaction/", nil)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestGetTransactions(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
	}{
		{
			name:           "empty query params",
			queryParams:    "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "with limit parameter",
			queryParams:    "?limit=10",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "with offset parameter",
			queryParams:    "?offset=10",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "with limit and offset",
			queryParams:    "?limit=20&offset=10",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "limit exceeds maximum",
			queryParams:    "?limit=200",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupTestRouter()
			router.POST("/transactions", GetTransactions)

			req := httptest.NewRequest(http.MethodPost, "/transactions"+tt.queryParams, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestBroadcastTransaction(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
	}{
		{
			name:           "invalid JSON",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing required fields",
			requestBody:    `{"from": "2001:db8::1"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "valid transaction request",
			requestBody:    `{"from": "2001:db8::1", "to": "2001:db8::2", "amount": 1000000000}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "valid transaction with fee and data",
			requestBody:    `{"from": "2001:db8::1", "to": "2001:db8::2", "amount": 1000000000, "fee": 10000, "data": "optional"}`,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupTestRouter()
			router.POST("/transaction/broadcast", BroadcastTransaction)

			req := httptest.NewRequest(http.MethodPost, "/transaction/broadcast", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestGetTransactionsByAddress(t *testing.T) {
	tests := []struct {
		name           string
		addressParam   string
		queryParams    string
		expectedStatus int
	}{
		{
			name:           "missing address parameter",
			addressParam:   "",
			queryParams:    "",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "valid address without query params",
			addressParam:   "2001:db8::1",
			queryParams:    "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "valid address with pagination",
			addressParam:   "2001:db8::1",
			queryParams:    "?limit=10&offset=5",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupTestRouter()
			router.GET("/transactions/address/:address", GetTransactionsByAddress)

			var req *http.Request
			if tt.addressParam != "" {
				req = httptest.NewRequest(http.MethodGet, "/transactions/address/"+tt.addressParam+tt.queryParams, nil)
			} else {
				req = httptest.NewRequest(http.MethodGet, "/transactions/address/", nil)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}
