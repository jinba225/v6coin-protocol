package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetNetworkStatus(t *testing.T) {
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
			router.POST("/network/status", GetNetworkStatus)

			req := httptest.NewRequest(http.MethodPost, "/network/status", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				respBody := w.Body.String()
				requiredFields := []string{
					"totalPeers", "activePeers", "inactivePeers",
					"averageUptime", "networkStatus", "message",
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

func TestGetNetworkTopology(t *testing.T) {
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
			router.POST("/network/topology", GetNetworkTopology)

			req := httptest.NewRequest(http.MethodPost, "/network/topology", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				respBody := w.Body.String()
				if !strings.Contains(respBody, "nodes") {
					t.Error("Response should contain 'nodes' field")
				}
				if !strings.Contains(respBody, "edges") {
					t.Error("Response should contain 'edges' field")
				}
			}
		})
	}
}
