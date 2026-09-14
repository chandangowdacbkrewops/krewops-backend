package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/chandangowdacbkrewops/krewops-backend/gateway/internal/response"
)

func RecoveryWithEnvelope() gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(gin.DefaultErrorWriter, func(c *gin.Context, recovered interface{}) {
		response.Internal(c, "internal server error", nil)
		c.Abort()
	})
}
