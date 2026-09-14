package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"

	"github.com/chandangowdacbkrewops/krewops-backend/gateway/internal/response"
)

const RequestIDHeader = "X-Request-ID"

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDHeader)
		if id == "" {
			id = generateRequestID()
		}

		c.Set(response.RequestIDContextKey, id)
		c.Header(RequestIDHeader, id)
		c.Next()
	}
}

func generateRequestID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "req_unknown"
	}
	return "req_" + hex.EncodeToString(buf)
}
