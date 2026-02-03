package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORS(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name               string
		method             string
		expectedHeaders    map[string]string
		expectedStatusCode int
	}{
		{
			name:   "GET request",
			method: http.MethodGet,
			expectedHeaders: map[string]string{
				"Access-Control-Allow-Origin":   "*",
				"Access-Control-Allow-Methods":  "GET, POST, PUT, PATCH, DELETE, OPTIONS",
				"Access-Control-Allow-Headers":  "Content-Type, Authorization, X-Request-ID",
				"Access-Control-Expose-Headers": "Content-Length, X-Request-ID",
				"Access-Control-Max-Age":        "86400",
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name:   "POST request",
			method: http.MethodPost,
			expectedHeaders: map[string]string{
				"Access-Control-Allow-Origin":   "*",
				"Access-Control-Allow-Methods":  "GET, POST, PUT, PATCH, DELETE, OPTIONS",
				"Access-Control-Allow-Headers":  "Content-Type, Authorization, X-Request-ID",
				"Access-Control-Expose-Headers": "Content-Length, X-Request-ID",
				"Access-Control-Max-Age":        "86400",
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name:   "OPTIONS request",
			method: http.MethodOptions,
			expectedHeaders: map[string]string{
				"Access-Control-Allow-Origin":   "*",
				"Access-Control-Allow-Methods":  "GET, POST, PUT, PATCH, DELETE, OPTIONS",
				"Access-Control-Allow-Headers":  "Content-Type, Authorization, X-Request-ID",
				"Access-Control-Expose-Headers": "Content-Length, X-Request-ID",
				"Access-Control-Max-Age":        "86400",
			},
			expectedStatusCode: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(CORS())

			router.Any("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			})

			req := httptest.NewRequest(tt.method, "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatusCode {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatusCode, w.Code)
			}

			for header, expectedValue := range tt.expectedHeaders {
				actualValue := w.Header().Get(header)
				if actualValue != expectedValue {
					t.Errorf("Expected header %s to be %s, got %s", header, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestCORSWithRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(CORS())

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", "test-request-id")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	allowHeaders := w.Header().Get("Access-Control-Allow-Headers")
	if allowHeaders == "" {
		t.Error("Access-Control-Allow-Headers header not set")
	}
}

func TestCORSWithMultipleOrigins(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(CORS())

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	origins := []string{"http://example.com", "https://example.com", "http://localhost:3000"}
	for _, origin := range origins {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
		if allowOrigin != "*" {
			t.Errorf("Expected Access-Control-Allow-Origin to be *, got %s", allowOrigin)
		}
	}
}
