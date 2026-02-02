// Package middleware provides HTTP middleware for the RPC server.
package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger returns a middleware that logs HTTP requests and responses.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		method := c.Request.Method
		clientIP := c.ClientIP()
		requestID := c.GetString("RequestID")

		var requestBody []byte
		if c.Request.Body != nil && method != "GET" && method != "HEAD" {
			requestBody, _ = c.GetRawData()
			c.Request.Body = io.NopCloser(bytes.NewReader(requestBody))
		}

		c.Next()

		latency := time.Since(start)

		logEntry := map[string]interface{}{
			"time":          time.Now().UTC().Format(time.RFC3339),
			"request_id":    requestID,
			"method":        method,
			"path":          path,
			"raw_query":     raw,
			"status":        c.Writer.Status(),
			"latency_ms":    latency.Milliseconds(),
			"client_ip":     clientIP,
			"user_agent":    c.Request.UserAgent(),
			"request_size":  c.Request.ContentLength,
			"response_size": c.Writer.Size(),
		}

		if len(requestBody) > 0 && len(requestBody) < 1024 {
			logEntry["request_body"] = string(requestBody)
		}

		if len(c.Errors) > 0 {
			logEntry["errors"] = c.Errors.String()
		}

		logJSON(logEntry)
	}
}

func logJSON(data map[string]interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Failed to marshal log data: %v", err)
		return
	}
	log.Println(string(jsonData))
}
