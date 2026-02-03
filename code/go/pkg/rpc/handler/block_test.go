package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jinba225/v6coin-protocol/pkg/consensus"
)

type mockBlockchainService struct {
	getBlock         func([]byte) *consensus.Block
	getBlockByHeight func(uint64) []*consensus.Block
	getChainHead     func() *consensus.Block
	getCurrentHeight func() uint64
}

func (m *mockBlockchainService) GetBlock(hash []byte) *consensus.Block {
	if m.getBlock != nil {
		return m.getBlock(hash)
	}
	return nil
}

func (m *mockBlockchainService) GetBlockByHeight(height uint64) []*consensus.Block {
	if m.getBlockByHeight != nil {
		return m.getBlockByHeight(height)
	}
	return nil
}

func (m *mockBlockchainService) GetChainHead() *consensus.Block {
	if m.getChainHead != nil {
		return m.getChainHead()
	}
	return nil
}

func (m *mockBlockchainService) GetCurrentHeight() uint64 {
	if m.getCurrentHeight != nil {
		return m.getCurrentHeight()
	}
	return 0
}

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router
}

func TestGetBlock(t *testing.T) {
	tests := []struct {
		name           string
		heightParam    string
		mockService    *mockBlockchainService
		expectedStatus int
	}{
		{
			name:           "missing height parameter",
			heightParam:    "",
			mockService:    &mockBlockchainService{},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid height format",
			heightParam:    "abc",
			mockService:    &mockBlockchainService{},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "block not found",
			heightParam: "999",
			mockService: &mockBlockchainService{
				getBlockByHeight: func(uint64) []*consensus.Block {
					return []*consensus.Block{}
				},
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:        "successful block retrieval",
			heightParam: "1",
			mockService: &mockBlockchainService{
				getBlockByHeight: func(uint64) []*consensus.Block {
					header := &consensus.BlockHeader{
						Height:      1,
						Timestamp:   1234567890,
						ValidatorID: "validator1",
					}
					block := &consensus.Block{
						Header: header,
					}
					return []*consensus.Block{block}
				},
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetBlockchainService(tt.mockService)

			router := setupRouter()
			router.GET("/block/:height", GetBlock)

			var req *http.Request
			if tt.heightParam != "" {
				req = httptest.NewRequest(http.MethodGet, "/block/"+tt.heightParam, nil)
			} else {
				req = httptest.NewRequest(http.MethodGet, "/block/", nil)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestGetBlockByHash(t *testing.T) {
	tests := []struct {
		name           string
		hashParam      string
		mockService    *mockBlockchainService
		expectedStatus int
	}{
		{
			name:           "missing hash parameter",
			hashParam:      "",
			mockService:    &mockBlockchainService{},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid hash format",
			hashParam:      "not-a-hash",
			mockService:    &mockBlockchainService{},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "block not found",
			hashParam: "aabbccdd",
			mockService: &mockBlockchainService{
				getBlock: func([]byte) *consensus.Block {
					return nil
				},
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:      "successful block retrieval",
			hashParam: "aabbccdd",
			mockService: &mockBlockchainService{
				getBlock: func([]byte) *consensus.Block {
					header := &consensus.BlockHeader{
						Height:      1,
						Timestamp:   1234567890,
						ValidatorID: "validator1",
					}
					return &consensus.Block{
						Header: header,
					}
				},
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetBlockchainService(tt.mockService)

			router := setupRouter()
			router.GET("/block/hash/:hash", GetBlockByHash)

			var req *http.Request
			if tt.hashParam != "" {
				req = httptest.NewRequest(http.MethodGet, "/block/hash/"+tt.hashParam, nil)
			} else {
				req = httptest.NewRequest(http.MethodGet, "/block/hash/", nil)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestGetLatestBlock(t *testing.T) {
	tests := []struct {
		name           string
		mockService    *mockBlockchainService
		expectedStatus int
	}{
		{
			name: "service unavailable",
			mockService: &mockBlockchainService{
				getChainHead: func() *consensus.Block {
					return nil
				},
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "successful latest block retrieval",
			mockService: &mockBlockchainService{
				getChainHead: func() *consensus.Block {
					header := &consensus.BlockHeader{
						Height:      100,
						Timestamp:   1234567890,
						ValidatorID: "validator1",
					}
					return &consensus.Block{
						Header: header,
					}
				},
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetBlockchainService(tt.mockService)

			router := setupRouter()
			router.GET("/block/latest", GetLatestBlock)

			req := httptest.NewRequest(http.MethodGet, "/block/latest", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestGetBlocks(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		mockService    *mockBlockchainService
		expectedStatus int
	}{
		{
			name:           "empty request",
			requestBody:    "",
			mockService:    &mockBlockchainService{},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "valid request with height",
			requestBody: `{"height": 50, "count": 10}`,
			mockService: &mockBlockchainService{
				getCurrentHeight: func() uint64 { return 100 },
				getBlockByHeight: func(uint64) []*consensus.Block {
					header := &consensus.BlockHeader{
						Height: 50,
					}
					block := &consensus.Block{
						Header: header,
					}
					return []*consensus.Block{block}
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "request with height exceeding current height",
			requestBody: `{"height": 150}`,
			mockService: &mockBlockchainService{
				getCurrentHeight: func() uint64 { return 100 },
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetBlockchainService(tt.mockService)

			router := setupRouter()
			router.POST("/blocks", GetBlocks)

			var req *http.Request
			if tt.requestBody != "" {
				req = httptest.NewRequest(http.MethodPost, "/blocks", strings.NewReader(tt.requestBody))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(http.MethodPost, "/blocks", nil)
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestUint64FromString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    uint64
		wantErr bool
	}{
		{
			name:    "valid number",
			input:   "123",
			want:    123,
			wantErr: false,
		},
		{
			name:    "zero",
			input:   "0",
			want:    0,
			wantErr: false,
		},
		{
			name:    "large number",
			input:   "18446744073709551615",
			want:    18446744073709551615,
			wantErr: false,
		},
		{
			name:    "invalid string",
			input:   "abc",
			want:    0,
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			want:    0,
			wantErr: false,
		},
		{
			name:    "string with special characters",
			input:   "12a34",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := uint64FromString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("uint64FromString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("uint64FromString() = %v, want %v", got, tt.want)
			}
			if err != nil && !errors.Is(err, ErrInvalidParameter) {
				t.Errorf("uint64FromString() error = %v, want ErrInvalidParameter", err)
			}
		})
	}
}

func TestIpv6ToString(t *testing.T) {
	tests := []struct {
		name string
		ip   []byte
		want string
	}{
		{
			name: "nil IP",
			ip:   nil,
			want: "",
		},
		{
			name: "valid IPv6",
			ip:   []byte{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
			want: "2001:db8::1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ipv6ToString(tt.ip)
			if got != tt.want {
				t.Errorf("ipv6ToString() = %v, want %v", got, tt.want)
			}
		})
	}
}
