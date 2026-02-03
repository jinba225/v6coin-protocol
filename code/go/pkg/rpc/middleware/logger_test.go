package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestLogger(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		method         string
		body           string
		expectedStatus int
	}{
		{
			name:           "GET request",
			method:         http.MethodGet,
			body:           "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST request",
			method:         http.MethodPost,
			body:           `{"test": "data"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "PUT request",
			method:         http.MethodPut,
			body:           `{"update": "data"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "DELETE request",
			method:         http.MethodDelete,
			body:           "",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(Logger())

			router.GET("/test", func(c *gin.Context) {
				c.JSON(tt.expectedStatus, gin.H{"status": "ok"})
			})
			router.POST("/test", func(c *gin.Context) {
				c.JSON(tt.expectedStatus, gin.H{"status": "ok"})
			})
			router.PUT("/test", func(c *gin.Context) {
				c.JSON(tt.expectedStatus, gin.H{"status": "ok"})
			})
			router.DELETE("/test", func(c *gin.Context) {
				c.JSON(tt.expectedStatus, gin.H{"status": "ok"})
			})

			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, "/test", strings.NewReader(tt.body))
			} else {
				req = httptest.NewRequest(tt.method, "/test", nil)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestLoggerWithRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(Logger())

	router.GET("/test", func(c *gin.Context) {
		requestID := c.GetString("RequestID")
		c.JSON(http.StatusOK, gin.H{"request_id": requestID})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestLoggerWithQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(Logger())

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test?param1=value1&param2=value2", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestLoggerWithLargeBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(Logger())

	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	largeBody := strings.Repeat("a", 2000)
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(largeBody))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestLoggerWithError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(Logger())

	router.GET("/test", func(c *gin.Context) {
		c.Error(gin.Error{
			Type: gin.ErrorTypePrivate,
			Err:  http.ErrHandlerTimeout,
		})
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}
