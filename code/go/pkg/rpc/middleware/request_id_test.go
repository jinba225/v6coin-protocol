package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name            string
		requestIDHeader string
		expectSameID    bool
	}{
		{
			name:            "request with X-Request-ID header",
			requestIDHeader: "client-request-id-123",
			expectSameID:    true,
		},
		{
			name:            "request without X-Request-ID header",
			requestIDHeader: "",
			expectSameID:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(RequestID())

			router.GET("/test", func(c *gin.Context) {
				requestID := c.GetString("RequestID")
				c.JSON(http.StatusOK, gin.H{"request_id": requestID})
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.requestIDHeader != "" {
				req.Header.Set("X-Request-ID", tt.requestIDHeader)
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
			}

			responseRequestID := w.Header().Get("X-Request-ID")
			if responseRequestID == "" {
				t.Error("X-Request-ID header not set in response")
			}

			if tt.expectSameID && responseRequestID != tt.requestIDHeader {
				t.Errorf("Expected X-Request-ID to be %s, got %s", tt.requestIDHeader, responseRequestID)
			}

			if !tt.expectSameID && responseRequestID == "" {
				t.Error("X-Request-ID should be generated when not provided")
			}
		})
	}
}

func TestRequestIDFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RequestID())

	router.GET("/test", func(c *gin.Context) {
		requestID := c.GetString("RequestID")
		c.JSON(http.StatusOK, gin.H{"request_id": requestID})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	requestID := w.Header().Get("X-Request-ID")
	if len(requestID) == 0 {
		t.Error("Generated request ID is empty")
	}
}

func TestRequestIDConsistency(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RequestID())

	router.GET("/test", func(c *gin.Context) {
		requestID := c.GetString("RequestID")
		responseID := c.Writer.Header().Get("X-Request-ID")
		if requestID != responseID {
			t.Errorf("Context request ID (%s) does not match response header ID (%s)", requestID, responseID)
		}
		c.JSON(http.StatusOK, gin.H{"request_id": requestID})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestRequestIDWithMultipleRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RequestID())

	router.GET("/test", func(c *gin.Context) {
		requestID := c.GetString("RequestID")
		c.JSON(http.StatusOK, gin.H{"request_id": requestID})
	})

	requestIDs := make(map[string]bool)
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		requestID := w.Header().Get("X-Request-ID")
		if requestID == "" {
			t.Error("Request ID is empty")
		}
		requestIDs[requestID] = true
	}

	if len(requestIDs) != 10 {
		t.Errorf("Expected 10 unique request IDs, got %d", len(requestIDs))
	}
}

func TestRequestIDWithCustomID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RequestID())

	router.GET("/test", func(c *gin.Context) {
		requestID := c.GetString("RequestID")
		c.JSON(http.StatusOK, gin.H{"request_id": requestID})
	})

	customID := "my-custom-request-id-12345"
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", customID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	responseID := w.Header().Get("X-Request-ID")
	if responseID != customID {
		t.Errorf("Expected X-Request-ID to be %s, got %s", customID, responseID)
	}
}
