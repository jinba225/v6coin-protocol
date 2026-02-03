package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetNodeInfo(t *testing.T) {
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
			name:           "empty JSON",
			requestBody:    `{}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "valid request",
			requestBody:    `{"any": "data"}`,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.POST("/node/info", GetNodeInfo)

			req := httptest.NewRequest(http.MethodPost, "/node/info", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				respBody := w.Body.String()
				if !strings.Contains(respBody, "id") {
					t.Error("Response should contain 'id' field")
				}
				if !strings.Contains(respBody, "address") {
					t.Error("Response should contain 'address' field")
				}
				if !strings.Contains(respBody, "height") {
					t.Error("Response should contain 'height' field")
				}
			}
		})
	}
}

func TestGetNodePeers(t *testing.T) {
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
			name:           "empty JSON",
			requestBody:    `{}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "valid request",
			requestBody:    `{"any": "data"}`,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.POST("/node/peers", GetNodePeers)

			req := httptest.NewRequest(http.MethodPost, "/node/peers", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				respBody := w.Body.String()
				if !strings.Contains(respBody, "peers") {
					t.Error("Response should contain 'peers' field")
				}
			}
		})
	}
}

func TestGetNodeStats(t *testing.T) {
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
			name:           "empty JSON",
			requestBody:    `{}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "valid request",
			requestBody:    `{"any": "data"}`,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.POST("/node/stats", GetNodeStats)

			req := httptest.NewRequest(http.MethodPost, "/node/stats", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				respBody := w.Body.String()
				requiredFields := []string{
					"uptime", "uptimePercentage", "totalForwarded",
					"totalReceived", "averageLatency", "packetLoss",
					"connections", "activeValidators", "contributionScore",
				}
				for _, field := range requiredFields {
					if !strings.Contains(respBody, field) {
						t.Errorf("Response should contain '%s' field", field)
					}
				}
			}
		})
	}
}
