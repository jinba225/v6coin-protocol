// Package middleware provides HTTP middleware for the RPC server.
package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// Recovery returns a middleware that recovers from any panics and returns a JSON error response.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				stack := debug.Stack()
				requestID := c.GetString("RequestID")

				log.Printf("Panic recovered: %v\nRequest ID: %s\nStack:\n%s", err, requestID, stack)

				c.JSON(http.StatusInternalServerError, gin.H{
					"success":    false,
					"error":      "1006",
					"message":    "Internal server error",
					"request_id": requestID,
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}
