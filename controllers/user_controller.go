package controllers

import (
	"net/http"

	"go-potensia/config"
	"go-potensia/models"

	"github.com/gin-gonic/gin"
)

func Profile(c *gin.Context) {
	var input models.User

	// bind request
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})
		return
	}

	var user models.User

	// cek user di DB
	if err := config.DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "Email tidak ditemukan",
		})
		return
	}

	if !user.IsVerified {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "Akun belum diverifikasi, cek email OTP",
		})
		return
	}


	c.JSON(http.StatusOK, gin.H{
		"message": "Data Profil berhasil diambil",
		"user": gin.H{
			"id":    user.ID,
			"name":  user.Name,
			"email": user.Email,
			"role":  user.Role,
		},
	})
}