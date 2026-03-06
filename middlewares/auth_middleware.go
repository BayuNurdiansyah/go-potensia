package middlewares

import (
	"net/http"
	"strings"

	"go-potensia/utils"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware validates the Bearer JWT token in the Authorization header.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Authorization header tidak ada"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Format Authorization tidak valid. Gunakan: Bearer <token>"})
			return
		}

		claims, err := utils.ParseToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Token tidak valid atau sudah kadaluarsa"})
			return
		}

		// Set user info ke context untuk digunakan di handler
		c.Set("userID", claims.UserID)
		c.Set("userEmail", claims.Email)
		c.Next()
	}
}