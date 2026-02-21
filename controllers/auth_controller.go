package controllers

import (
	"net/http"

	"go-potensia/config"
	"go-potensia/models"
	"go-potensia/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func Register(c *gin.Context) {
	var input models.User

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	// hash password
	hash, _ := bcrypt.GenerateFromPassword([]byte(input.Password), 14)
	input.Password = string(hash)

	// simpan ke DB
	config.DB.Create(&input)

	token, _ := utils.GenerateToken(input.ID, input.Email)

	c.JSON(http.StatusOK, gin.H{
		"message": "Register success",
		"token":   token,
		"user":    input,
	})
}

var jwtKey = []byte("SECRET_KEY_BEBAS") // nanti pindah ke env

func Login(c *gin.Context) {
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

	// compare password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "Password salah",
		})
		return
	}

	// validasi role
	if user.Role != input.Role {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "Role tidak sesuai, harap ganti role",
		})
		return
	}

	// generate JWT
	token, err := utils.GenerateToken(user.ID, user.Email)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Gagal generate token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login berhasil",
		"token":   token,
		"user": gin.H{
			"id":    user.ID,
			"name":  user.Name,
			"email": user.Email,
			"role":  user.Role,
		},
	})
}