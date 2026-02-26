package controllers

import (
	"net/http"
	"time"
	"fmt"

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

	if input.Email == "" || input.Password == "" || input.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Nama, email, dan password wajib diisi",
		})
		return
	}

	var existing models.User
	config.DB.Where("email = ?", input.Email).First(&existing)

	if existing.ID != 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Email sudah terdaftar",
		})
		return
	}
	now := time.Now().Unix()

	if existing.ID != 0 {
		if now-existing.LastOTPSentAt < 60 {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"message": "Tunggu 60 detik sebelum request OTP lagi",
				"retry_after": 45,
			})
			return
		}
	}

	// hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), 14)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Gagal hash password",
		})
		return
	}
	input.Password = string(hash)

	// Generate OTP
	otp := utils.GenerateOTP()

	// Set field OTP
	input.OTP = otp
	input.OTPExpired = time.Now().Add(5 * time.Minute).Unix()
	input.IsVerified = false
	input.LastOTPSentAt = now
	input.OTPAttempts = 0

	// simpan ke DB
	if err := config.DB.Create(&input).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Gagal menyimpan user",
		})
		return
	}

	// send email (async using goroutines)
	go func() {
		err := utils.SendOTPEmail(input.Email, input.Name, otp)
			if err != nil {
				fmt.Println("Gagal kirim email:", err)
			}
	}()

	c.JSON(http.StatusOK, gin.H{
		"message": "Register berhasil, cek email untuk OTP",
		"email":   input.Email,
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

	if !user.IsVerified {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "Akun belum diverifikasi, cek email OTP",
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

func VerifyOTP(c *gin.Context) {
	var input struct {
		Email string `json:"email"`
		OTP   string `json:"otp"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Format request tidak valid",
		})
		return
	}

	var user models.User
	config.DB.Where("email = ?", input.Email).First(&user)

	if user.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "User tidak ditemukan",
		})
		return
	}

	if user.IsVerified {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Akun sudah terverifikasi",
		})
		return
	}

	if user.OTPAttempts >= 5 {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"message": "Terlalu banyak percobaan OTP, silakan request ulang",
		})
		return
	}

	if user.OTP != input.OTP {
		user.OTPAttempts += 1
		config.DB.Save(&user)

		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "OTP salah",
		})
		return
	}

	if time.Now().Unix() > user.OTPExpired {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "OTP sudah kadaluarsa",
		})
		return
	}

	user.IsVerified = true
	user.OTP = ""
	user.OTPAttempts = 0
	config.DB.Save(&user)

	// generate JWT to auto login
	token, _ := utils.GenerateToken(user.ID, user.Email)

	c.JSON(http.StatusOK, gin.H{
		"message": "Verifikasi berhasil",
		"token":   token,
	})
}

func ResendOTP(c *gin.Context) {
	var input struct {
		Email string `json:"email"`
	}

	// Bind request
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Format request tidak valid",
		})
		return
	}

	// Cari user
	var user models.User
	config.DB.Where("email = ?", input.Email).First(&user)

	if user.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "User tidak ditemukan",
		})
		return
	}

	if user.IsVerified {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Akun sudah terverifikasi",
		})
		return
	}

	now := time.Now().Unix()

	// Rate limit (60 detik)
	if now-user.LastOTPSentAt < 60 {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"message": "Tunggu 60 detik sebelum kirim ulang OTP",
			"retry_after": 60 - (now - user.LastOTPSentAt),
		})
		return
	}

	// Generate OTP baru
	otp := utils.GenerateOTP()

	// Update data user
	user.OTP = otp
	user.OTPExpired = time.Now().Add(5 * time.Minute).Unix()
	user.LastOTPSentAt = now
	user.OTPAttempts = 0

	config.DB.Save(&user)

	go func() {
		err := utils.SendOTPEmail(user.Email, user.Name, otp)
		if err != nil {
			fmt.Println("Gagal kirim email:", err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"message": "OTP berhasil dikirim ulang",
	})
}