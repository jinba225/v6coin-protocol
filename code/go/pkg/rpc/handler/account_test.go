package handler

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetAccount(t *testing.T) {
	tests := []struct {
		name           string
		addressParam   string
		expectedStatus int
	}{
		{
			name:           "missing address parameter",
			addressParam:   "",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid IPv4 address",
			addressParam:   "192.168.1.1",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid IPv6 address format",
			addressParam:   "invalid-address",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "valid IPv6 address",
			addressParam:   "2001:db8::1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "valid IPv6 full address",
			addressParam:   "2001:0db8:0000:0000:0000:0000:0000:0001",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupTestRouter()
			router.GET("/account/:address", GetAccount)

			var req *http.Request
			if tt.addressParam != "" {
				req = httptest.NewRequest(http.MethodGet, "/account/"+tt.addressParam, nil)
			} else {
				req = httptest.NewRequest(http.MethodGet, "/account/", nil)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestGetBalance(t *testing.T) {
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
			name:           "missing address field",
			requestBody:    `{}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid IPv4 address",
			requestBody:    `{"address": "192.168.1.1"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "valid request",
			requestBody:    `{"address": "2001:db8::1"}`,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupTestRouter()
			router.POST("/account/balance", GetBalance)

			req := httptest.NewRequest(http.MethodPost, "/account/balance", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestGetNonce(t *testing.T) {
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
			name:           "missing address field",
			requestBody:    `{}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid address format",
			requestBody:    `{"address": "invalid"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "valid request",
			requestBody:    `{"address": "2001:db8::1"}`,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupTestRouter()
			router.POST("/account/nonce", GetNonce)

			req := httptest.NewRequest(http.MethodPost, "/account/nonce", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestCreateAccount(t *testing.T) {
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
			name:           "missing password field",
			requestBody:    `{}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty password",
			requestBody:    `{"password": ""}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "valid request",
			requestBody:    `{"password": "testpassword"}`,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupTestRouter()
			router.POST("/account/create", CreateAccount)

			req := httptest.NewRequest(http.MethodPost, "/account/create", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.name == "valid request" && w.Code == http.StatusOK {
				respBody := w.Body.String()
				if !strings.Contains(respBody, "address") {
					t.Error("Response should contain 'address' field")
				}
			}
		})
	}
}

func TestValidateIPv6Address(t *testing.T) {
	tests := []struct {
		name      string
		address   string
		expectNil bool
	}{
		{
			name:      "nil address",
			address:   "",
			expectNil: true,
		},
		{
			name:      "IPv4 address",
			address:   "192.168.1.1",
			expectNil: true,
		},
		{
			name:      "invalid format",
			address:   "not-an-ip",
			expectNil: true,
		},
		{
			name:      "valid IPv6 compressed",
			address:   "2001:db8::1",
			expectNil: false,
		},
		{
			name:      "valid IPv6 full",
			address:   "2001:0db8:0000:0000:0000:0000:0000:0001",
			expectNil: false,
		},
		{
			name:      "valid IPv6 loopback",
			address:   "::1",
			expectNil: false,
		},
		{
			name:      "valid IPv6 unspecified",
			address:   "::",
			expectNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.address)
			isNil := ip == nil || ip.To4() != nil

			if isNil != tt.expectNil {
				t.Errorf("ParseIP(%s) nil expectation = %v, got %v", tt.address, tt.expectNil, isNil)
			}
		})
	}
}
