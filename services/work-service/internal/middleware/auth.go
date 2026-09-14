package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const AuthUserIDKey = "auth_user_id"

func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{"error": "authorization header required"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.JSON(401, gin.H{"error": "invalid authorization header"})
			c.Abort()
			return
		}

		token, err := jwt.Parse(parts[1], func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(401, gin.H{"error": "invalid or expired token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(401, gin.H{"error": "invalid token claims"})
			c.Abort()
			return
		}

		sub, ok := claims["sub"].(string)
		if !ok || sub == "" {
			c.JSON(401, gin.H{"error": "invalid token subject"})
			c.Abort()
			return
		}

		c.Set(AuthUserIDKey, sub)
		c.Next()
	}
}

func GetAuthUserID(c *gin.Context) (string, bool) {
	value, exists := c.Get(AuthUserIDKey)
	if !exists {
		return "", false
	}

	userID, ok := value.(string)
	if !ok || userID == "" {
		return "", false
	}

	return userID, true
}
