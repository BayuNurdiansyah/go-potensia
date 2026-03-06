package middlewares

import (
	"strings"

	"go-potensia/models"
	"go-potensia/utils"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware validates the Bearer JWT token.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			utils.Unauthorized(c, "Authorization header tidak ada")
			c.Abort()
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			utils.Unauthorized(c, "Format Authorization tidak valid. Gunakan: Bearer <token>")
			c.Abort()
			return
		}

		claims, err := utils.ParseToken(parts[1])
		if err != nil {
			utils.Unauthorized(c, "Token tidak valid atau sudah kadaluarsa")
			c.Abort()
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("userEmail", claims.Email)
		c.Set("userRole", claims.Role)
		c.Next()
	}
}

// RequireRole returns a middleware that restricts access to specific roles.
func RequireRole(roles ...models.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get("userRole")
		if !exists {
			utils.Forbidden(c, "Role tidak ditemukan")
			c.Abort()
			return
		}
		userRole := models.Role(roleVal.(string))
		for _, r := range roles {
			if userRole == r {
				c.Next()
				return
			}
		}
		utils.Forbidden(c, "Akses ditolak untuk role ini")
		c.Abort()
	}
}