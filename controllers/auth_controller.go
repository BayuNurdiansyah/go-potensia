package controllers

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"go-potensia/config"
	"go-potensia/models"
	"go-potensia/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// ────────────────────────────────────────────────────────────────────────────
// Register
// ────────────────────────────────────────────────────────────────────────────

func Register(c *gin.Context) {
	var input struct {
		Name     string      `json:"name"`
		Email    string      `json:"email"`
		Password string      `json:"password"`
		Role     models.Role `json:"role"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Format request tidak valid"})
		return
	}

	// Trim whitespace
	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.TrimSpace(strings.ToLower(input.Email))

	// Validasi field wajib
	if input.Name == "" || input.Email == "" || input.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Nama, email, dan password wajib diisi"})
		return
	}

	// Validasi format email
	if !utils.IsValidEmail(input.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Format email tidak valid"})
		return
	}

	// Validasi password
	if !utils.IsValidPassword(input.Password) {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Password minimal 8 karakter dan harus mengandung huruf serta angka",
		})
		return
	}

	// Validasi role
	if input.Role == "" {
		input.Role = models.RoleStudent
	}
	if input.Role != models.RoleStudent && input.Role != models.RoleMentor {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Role tidak valid"})
		return
	}

	// Cek apakah email sudah digunakan
	var existing models.User
	config.DB.Where("email = ?", input.Email).First(&existing)

	if existing.ID != 0 {
		if existing.IsVerified {
			c.JSON(http.StatusConflict, gin.H{"message": "Email sudah terdaftar"})
			return
		}

		// Akun belum verified — cek rate limit sebelum kirim ulang OTP
		now := time.Now().Unix()
		if now-existing.LastOTPSentAt < 60 {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"message":     "Tunggu 60 detik sebelum request OTP lagi",
				"retry_after": 60 - (now - existing.LastOTPSentAt),
			})
			return
		}

		// Kirim ulang OTP untuk akun yang belum terverifikasi
		otp := utils.GenerateOTP()
		existing.OTP = otp
		existing.OTPExpired = time.Now().Add(5 * time.Minute).Unix()
		existing.LastOTPSentAt = now
		existing.OTPAttempts = 0
		config.DB.Save(&existing)

		go func(email, name, otpCode string) {
			if err := utils.SendOTPEmail(email, name, otpCode); err != nil {
				log.Printf("ERROR Send OTP Email to %s: %v", email, err)
			}
		}(existing.Email, existing.Name, otp)

		c.JSON(http.StatusOK, gin.H{
			"message": "Email sudah terdaftar tapi belum diverifikasi. OTP baru telah dikirim.",
			"email":   existing.Email,
		})
		return
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memproses password"})
		return
	}

	// Generate OTP
	otp := utils.GenerateOTP()
	now := time.Now().Unix()

	user := models.User{
		Name:          input.Name,
		Email:         input.Email,
		Password:      string(hash),
		Role:          input.Role,
		OTP:           otp,
		OTPExpired:    time.Now().Add(5 * time.Minute).Unix(),
		IsVerified:    false,
		LastOTPSentAt: now,
		OTPAttempts:   0,
	}

	if err := config.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal menyimpan data user"})
		return
	}

	// Kirim OTP via email secara async
	go func(email, name, otpCode string) {
		if err := utils.SendOTPEmail(email, name, otpCode); err != nil {
			log.Printf("ERROR SendOTPEmail to %s: %v", email, err)
		}
	}(user.Email, user.Name, otp)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Register berhasil, cek email untuk kode OTP",
		"email":   user.Email,
	})
}

// ────────────────────────────────────────────────────────────────────────────
// VerifyOTP
// ────────────────────────────────────────────────────────────────────────────

func VerifyOTP(c *gin.Context) {
	var input struct {
		Email string `json:"email"`
		OTP   string `json:"otp"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Format request tidak valid"})
		return
	}

	input.Email = strings.TrimSpace(strings.ToLower(input.Email))
	input.OTP = strings.TrimSpace(input.OTP)

	if input.Email == "" || input.OTP == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Email dan OTP wajib diisi"})
		return
	}

	var user models.User
	config.DB.Where("email = ?", input.Email).First(&user)

	if user.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "User tidak ditemukan"})
		return
	}

	if user.IsVerified {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Akun sudah terverifikasi"})
		return
	}

	// Cek rate limit percobaan OTP
	if user.OTPAttempts >= 5 {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"message": "Terlalu banyak percobaan. Silakan minta OTP baru.",
		})
		return
	}

	// Cek expiry terlebih dahulu (sebelum cek value OTP)
	if time.Now().Unix() > user.OTPExpired {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "OTP sudah kadaluarsa. Silakan minta OTP baru.",
		})
		return
	}

	// Cek value OTP
	if user.OTP != input.OTP {
		user.OTPAttempts++
		config.DB.Save(&user)

		remaining := 5 - user.OTPAttempts
		c.JSON(http.StatusUnauthorized, gin.H{
			"message":   fmt.Sprintf("OTP salah. Sisa percobaan: %d", remaining),
			"attempts_remaining": remaining,
		})
		return
	}

	// OTP valid — verifikasi akun
	user.IsVerified = true
	user.OTP = ""
	user.OTPExpired = 0
	user.OTPAttempts = 0
	config.DB.Save(&user)

	// Auto login — generate JWT
	token, err := utils.GenerateToken(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Verifikasi berhasil tapi gagal generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Verifikasi berhasil",
		"token":   token,
		"user": gin.H{
			"id":    user.ID,
			"name":  user.Name,
			"email": user.Email,
			"role":  user.Role,
		},
	})
}

// ────────────────────────────────────────────────────────────────────────────
// ResendOTP
// ────────────────────────────────────────────────────────────────────────────

func ResendOTP(c *gin.Context) {
	var input struct {
		Email string `json:"email"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Format request tidak valid"})
		return
	}

	input.Email = strings.TrimSpace(strings.ToLower(input.Email))
	if input.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Email wajib diisi"})
		return
	}

	var user models.User
	config.DB.Where("email = ?", input.Email).First(&user)

	if user.ID == 0 {
		// Jangan bocorkan apakah email ada atau tidak (security best practice)
		c.JSON(http.StatusOK, gin.H{"message": "Jika email terdaftar, OTP akan dikirim"})
		return
	}

	if user.IsVerified {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Akun sudah terverifikasi"})
		return
	}

	now := time.Now().Unix()

	// Rate limit: 60 detik
	if now-user.LastOTPSentAt < 60 {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"message":     "Tunggu 60 detik sebelum kirim ulang OTP",
			"retry_after": 60 - (now - user.LastOTPSentAt),
		})
		return
	}

	otp := utils.GenerateOTP()
	user.OTP = otp
	user.OTPExpired = time.Now().Add(5 * time.Minute).Unix()
	user.LastOTPSentAt = now
	user.OTPAttempts = 0
	config.DB.Save(&user)

	go func(email, name, otpCode string) {
		if err := utils.SendOTPEmail(email, name, otpCode); err != nil {
			log.Printf("ERROR SendOTPEmail to %s: %v", email, err)
		}
	}(user.Email, user.Name, otp)

	c.JSON(http.StatusOK, gin.H{"message": "OTP berhasil dikirim ulang"})
}

// ────────────────────────────────────────────────────────────────────────────
// Login
// ────────────────────────────────────────────────────────────────────────────

func Login(c *gin.Context) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Format request tidak valid"})
		return
	}

	input.Email = strings.TrimSpace(strings.ToLower(input.Email))

	if input.Email == "" || input.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Email dan password wajib diisi"})
		return
	}

	var user models.User
	if err := config.DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		// Gunakan pesan generik agar tidak bocorkan info (timing-safe alternative not needed here)
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Email atau password salah"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Email atau password salah"})
		return
	}

	if !user.IsVerified {
		c.JSON(http.StatusForbidden, gin.H{
			"message": "Akun belum diverifikasi. Cek email untuk kode OTP.",
			"email":   user.Email,
		})
		return
	}

	token, err := utils.GenerateToken(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal membuat token"})
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

// ────────────────────────────────────────────────────────────────────────────
// ForgotPassword — Step 1: request reset link
// ────────────────────────────────────────────────────────────────────────────

func ForgotPassword(c *gin.Context) {
	var input struct {
		Email string `json:"email"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Format request tidak valid"})
		return
	}

	input.Email = strings.TrimSpace(strings.ToLower(input.Email))
	if input.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Email wajib diisi"})
		return
	}

	// Selalu return 200 agar tidak bocorkan apakah email terdaftar atau tidak
	successMsg := gin.H{"message": "Jika email terdaftar dan terverifikasi, link reset password akan dikirim"}

	var user models.User
	if err := config.DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusOK, successMsg)
		return
	}

	if !user.IsVerified {
		c.JSON(http.StatusOK, successMsg)
		return
	}

	// Rate limit: 60 detik antar request
	now := time.Now().Unix()
	if user.ResetTokenExpired > 0 && now-(user.ResetTokenExpired-15*60) < 60 {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"message":     "Tunggu 60 detik sebelum request reset password lagi",
			"retry_after": int64(60) - (now - (user.ResetTokenExpired - 15*60)),
		})
		return
	}

	// Generate secure token (32 bytes = 64 hex chars)
	token, err := utils.GenerateSecureToken(32)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal membuat token reset"})
		return
	}

	user.ResetToken = token
	user.ResetTokenExpired = time.Now().Add(15 * time.Minute).Unix()
	user.ResetTokenUsed = false
	config.DB.Save(&user)

	go func(email, name, resetToken string) {
		if err := utils.SendForgotPasswordEmail(email, name, resetToken); err != nil {
			log.Printf("ERROR SendForgotPasswordEmail to %s: %v", email, err)
		}
	}(user.Email, user.Name, token)

	c.JSON(http.StatusOK, successMsg)
}

// ────────────────────────────────────────────────────────────────────────────
// VerifyResetToken — Step 2: validate token before showing reset form
// ────────────────────────────────────────────────────────────────────────────

func VerifyResetToken(c *gin.Context) {
	token := strings.TrimSpace(c.Query("token"))
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Token wajib diisi"})
		return
	}

	var user models.User
	if err := config.DB.Where("reset_token = ?", token).First(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Token tidak valid"})
		return
	}

	if user.ResetTokenUsed {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Token sudah digunakan"})
		return
	}

	if time.Now().Unix() > user.ResetTokenExpired {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Token sudah kadaluarsa"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Token valid",
		"email":   user.Email,
	})
}

// ────────────────────────────────────────────────────────────────────────────
// ResetPassword — Step 3: set new password
// ────────────────────────────────────────────────────────────────────────────

func ResetPassword(c *gin.Context) {
	var input struct {
		Token           string `json:"token"`
		NewPassword     string `json:"new_password"`
		ConfirmPassword string `json:"confirm_password"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Format request tidak valid"})
		return
	}

	input.Token = strings.TrimSpace(input.Token)
	if input.Token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Token wajib diisi"})
		return
	}

	if input.NewPassword == "" || input.ConfirmPassword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Password baru dan konfirmasi password wajib diisi"})
		return
	}

	if input.NewPassword != input.ConfirmPassword {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Password baru dan konfirmasi password tidak cocok"})
		return
	}

	if !utils.IsValidPassword(input.NewPassword) {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Password minimal 8 karakter dan harus mengandung huruf serta angka",
		})
		return
	}

	var user models.User
	if err := config.DB.Where("reset_token = ?", input.Token).First(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Token tidak valid"})
		return
	}

	if user.ResetTokenUsed {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Token sudah digunakan"})
		return
	}

	if time.Now().Unix() > user.ResetTokenExpired {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Token sudah kadaluarsa"})
		return
	}

	// Hash password baru
	hash, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memproses password"})
		return
	}

	// Update password dan invalidate token
	user.Password = string(hash)
	user.ResetToken = ""
	user.ResetTokenExpired = 0
	user.ResetTokenUsed = true
	config.DB.Save(&user)

	c.JSON(http.StatusOK, gin.H{"message": "Password berhasil direset. Silakan login dengan password baru."})
}