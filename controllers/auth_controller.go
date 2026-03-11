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

// ─── REGISTER ────────────────────────────────────────────────────────────────

func Register(c *gin.Context) {
	var input struct {
		Name     string      `json:"name"`
		Email    string      `json:"email"`
		Phone    string      `json:"phone"`
		Password string      `json:"password"`
		PasswordConfirmed string      `json:"password_confirm"`
		Role     models.Role `json:"role"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.BadRequest(c, "Format request tidak valid")
		return
	}

	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.Phone = strings.TrimSpace(input.Phone)

	if input.Name == "" || input.Email == "" || input.Password == "" {
		utils.BadRequest(c, "Nama, email, dan password wajib diisi")
		return
	}
	if !utils.IsValidEmail(input.Email) {
		utils.BadRequest(c, "Format email tidak valid")
		return
	}
	if !utils.IsValidPassword(input.Password) {
		utils.BadRequest(c, "Password minimal 8 karakter dan harus mengandung huruf serta angka")
		return
	}
	if !utils.IsSamePassword(input.Password, input.PasswordConfirmed) {
		utils.BadRequest(c, "Password dan konfirmasi password tidak cocok")
		return
	}
	if input.Phone != "" && !utils.IsValidPhone(input.Phone) {
		utils.BadRequest(c, "Format nomor HP tidak valid")
		return
	}
	if input.Role != models.RoleParent && input.Role != models.RoleMentor {
		utils.BadRequest(c, "Role harus 'parent' atau 'mentor'")
		return
	}

	// Cek email sudah terdaftar
	var existing models.User
	config.DB.Where("email = ?", input.Email).First(&existing)

	if existing.ID != 0 {
		if existing.IsVerified {
			utils.Conflict(c, "Email sudah terdaftar")
			return
		}
		// Belum verifikasi: rate limit resend OTP
		now := time.Now().Unix()
		if now-existing.LastOTPSentAt < 60 {
			utils.TooManyRequests(c, "Tunggu 60 detik sebelum request OTP lagi", 60-(now-existing.LastOTPSentAt))
			return
		}
		// Update & kirim ulang OTP
		otp := utils.GenerateOTP()
		existing.OTP = otp
		existing.OTPExpired = time.Now().Add(5 * time.Minute).Unix()
		existing.LastOTPSentAt = now
		existing.OTPAttempts = 0
		config.DB.Save(&existing)

		go func(email, name, code string) {
			if err := utils.SendOTPEmail(email, name, code); err != nil {
				log.Printf("SendOTPEmail error to %s: %v", email, err)
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
		utils.InternalError(c, "Gagal memproses password")
		return
	}

	otp := utils.GenerateOTP()
	now := time.Now().Unix()

	user := models.User{
		Name:          input.Name,
		Email:         input.Email,
		Phone:         input.Phone,
		Password:      string(hash),
		Role:          input.Role,
		OTP:           otp,
		OTPExpired:    time.Now().Add(5 * time.Minute).Unix(),
		IsVerified:    false,
		LastOTPSentAt: now,
		OTPAttempts:   0,
	}
	if err := config.DB.Create(&user).Error; err != nil {
		utils.InternalError(c, "Gagal menyimpan user")
		return
	}

	// Buat profil sesuai role
	if input.Role == models.RoleMentor {
		config.DB.Create(&models.MentorProfile{UserID: user.ID})
	} else {
		config.DB.Create(&models.ParentProfile{UserID: user.ID})
	}

	go func(email, name, code string) {
		if err := utils.SendOTPEmail(email, name, code); err != nil {
			log.Printf("SendOTPEmail error to %s: %v", email, err)
		}
	}(user.Email, user.Name, otp)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Register berhasil, cek email untuk kode OTP",
		"email":   user.Email,
	})
}

// ─── VERIFY OTP ──────────────────────────────────────────────────────────────

func VerifyOTP(c *gin.Context) {
	var input struct {
		Email string `json:"email"`
		OTP   string `json:"otp"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.BadRequest(c, "Format request tidak valid")
		return
	}
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.OTP = strings.TrimSpace(input.OTP)
	if input.Email == "" || input.OTP == "" {
		utils.BadRequest(c, "Email dan OTP wajib diisi")
		return
	}

	var user models.User
	config.DB.Where("email = ?", input.Email).First(&user)
	if user.ID == 0 {
		utils.NotFound(c, "User tidak ditemukan")
		return
	}
	if user.IsVerified {
		utils.BadRequest(c, "Akun sudah terverifikasi")
		return
	}
	if user.OTPAttempts >= 5 {
		utils.TooManyRequests(c, "Terlalu banyak percobaan. Silakan minta OTP baru.", 0)
		return
	}
	// Cek expiry dulu, baru cek value
	if time.Now().Unix() > user.OTPExpired {
		utils.Unauthorized(c, "OTP sudah kadaluarsa. Silakan minta OTP baru.")
		return
	}
	if user.OTP != input.OTP {
		user.OTPAttempts++
		config.DB.Save(&user)
		remaining := 5 - user.OTPAttempts
		c.JSON(http.StatusUnauthorized, gin.H{
			"message":            fmt.Sprintf("OTP salah. Sisa percobaan: %d", remaining),
			"attempts_remaining": remaining,
		})
		return
	}

	user.IsVerified = true
	user.OTP = ""
	user.OTPExpired = 0
	user.OTPAttempts = 0
	config.DB.Save(&user)

	token, err := utils.GenerateToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		utils.InternalError(c, "Verifikasi berhasil tapi gagal generate token")
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

// ─── RESEND OTP ──────────────────────────────────────────────────────────────

func ResendOTP(c *gin.Context) {
	var input struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.BadRequest(c, "Format request tidak valid")
		return
	}
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))

	// Selalu return OK agar tidak bocorkan info email
	var user models.User
	config.DB.Where("email = ?", input.Email).First(&user)
	if user.ID == 0 || user.IsVerified {
		c.JSON(http.StatusOK, gin.H{"message": "Jika email terdaftar dan belum terverifikasi, OTP akan dikirim"})
		return
	}

	now := time.Now().Unix()
	if now-user.LastOTPSentAt < 60 {
		utils.TooManyRequests(c, "Tunggu 60 detik sebelum kirim ulang OTP", 60-(now-user.LastOTPSentAt))
		return
	}

	otp := utils.GenerateOTP()
	user.OTP = otp
	user.OTPExpired = time.Now().Add(5 * time.Minute).Unix()
	user.LastOTPSentAt = now
	user.OTPAttempts = 0
	config.DB.Save(&user)

	go func(email, name, code string) {
		if err := utils.SendOTPEmail(email, name, code); err != nil {
			log.Printf("SendOTPEmail error to %s: %v", email, err)
		}
	}(user.Email, user.Name, otp)

	c.JSON(http.StatusOK, gin.H{"message": "OTP berhasil dikirim ulang"})
}

// ─── LOGIN ────────────────────────────────────────────────────────────────────

func Login(c *gin.Context) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.BadRequest(c, "Format request tidak valid")
		return
	}
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	if input.Email == "" || input.Password == "" {
		utils.BadRequest(c, "Email dan password wajib diisi")
		return
	}

	var user models.User
	if err := config.DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		utils.Unauthorized(c, "Email atau password salah")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		utils.Unauthorized(c, "Email atau password salah")
		return
	}
	if !user.IsVerified {
		c.JSON(http.StatusForbidden, gin.H{
			"message": "Akun belum diverifikasi. Cek email untuk kode OTP.",
			"email":   user.Email,
		})
		return
	}
	if !user.IsActive {
		utils.Forbidden(c, "Akun tidak aktif. Hubungi admin.")
		return
	}

	token, err := utils.GenerateToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		utils.InternalError(c, "Gagal membuat token")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login berhasil",
		"token":   token,
		"user": gin.H{
			"id":    user.ID,
			"name":  user.Name,
			"email": user.Email,
			"phone": user.Phone,
			"role":  user.Role,
		},
	})
}

// ─── FORGOT PASSWORD ─────────────────────────────────────────────────────────

func ForgotPassword(c *gin.Context) {
	var input struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.BadRequest(c, "Format request tidak valid")
		return
	}
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))

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

	// Rate limit
	now := time.Now().Unix()
	if user.ResetTokenExpired > 0 && (user.ResetTokenExpired-now) > (15*60-60) {
		utils.TooManyRequests(c, "Tunggu sebelum request reset password lagi", 60)
		return
	}

	token, err := utils.GenerateSecureToken(32)
	if err != nil {
		utils.InternalError(c, "Gagal membuat token reset")
		return
	}
	user.ResetToken = token
	user.ResetTokenExpired = time.Now().Add(15 * time.Minute).Unix()
	user.ResetTokenUsed = false
	config.DB.Save(&user)

	go func(email, name, resetToken string) {
		if err := utils.SendForgotPasswordEmail(email, name, resetToken); err != nil {
			log.Printf("SendForgotPasswordEmail error to %s: %v", email, err)
		}
	}(user.Email, user.Name, token)

	c.JSON(http.StatusOK, successMsg)
}

// ─── VERIFY RESET TOKEN ──────────────────────────────────────────────────────

func VerifyResetToken(c *gin.Context) {
	token := strings.TrimSpace(c.Query("token"))
	if token == "" {
		utils.BadRequest(c, "Token wajib diisi")
		return
	}

	var user models.User
	if err := config.DB.Where("reset_token = ?", token).First(&user).Error; err != nil {
		utils.BadRequest(c, "Token tidak valid")
		return
	}
	if user.ResetTokenUsed {
		utils.BadRequest(c, "Token sudah digunakan")
		return
	}
	if time.Now().Unix() > user.ResetTokenExpired {
		utils.BadRequest(c, "Token sudah kadaluarsa")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Token valid", "email": user.Email})
}

// ─── RESET PASSWORD ──────────────────────────────────────────────────────────

func ResetPassword(c *gin.Context) {
	var input struct {
		Token           string `json:"token"`
		NewPassword     string `json:"new_password"`
		ConfirmPassword string `json:"confirm_password"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.BadRequest(c, "Format request tidak valid")
		return
	}
	input.Token = strings.TrimSpace(input.Token)
	if input.Token == "" {
		utils.BadRequest(c, "Token wajib diisi")
		return
	}
	if input.NewPassword != input.ConfirmPassword {
		utils.BadRequest(c, "Password baru dan konfirmasi tidak cocok")
		return
	}
	if !utils.IsValidPassword(input.NewPassword) {
		utils.BadRequest(c, "Password minimal 8 karakter dan harus mengandung huruf serta angka")
		return
	}

	var user models.User
	if err := config.DB.Where("reset_token = ?", input.Token).First(&user).Error; err != nil {
		utils.BadRequest(c, "Token tidak valid")
		return
	}
	if user.ResetTokenUsed {
		utils.BadRequest(c, "Token sudah digunakan")
		return
	}
	if time.Now().Unix() > user.ResetTokenExpired {
		utils.BadRequest(c, "Token sudah kadaluarsa")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		utils.InternalError(c, "Gagal memproses password")
		return
	}

	user.Password = string(hash)
	user.ResetToken = ""
	user.ResetTokenExpired = 0
	user.ResetTokenUsed = true
	config.DB.Save(&user)

	c.JSON(http.StatusOK, gin.H{"message": "Password berhasil direset. Silakan login dengan password baru."})
}

// ─── CHANGE PASSWORD (authenticated) ────────────────────────────────────────

func ChangePassword(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var input struct {
		OldPassword     string `json:"old_password"`
		NewPassword     string `json:"new_password"`
		ConfirmPassword string `json:"confirm_password"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.BadRequest(c, "Format request tidak valid")
		return
	}
	if input.OldPassword == "" || input.NewPassword == "" {
		utils.BadRequest(c, "Password lama dan baru wajib diisi")
		return
	}
	if input.NewPassword != input.ConfirmPassword {
		utils.BadRequest(c, "Password baru dan konfirmasi tidak cocok")
		return
	}
	if !utils.IsValidPassword(input.NewPassword) {
		utils.BadRequest(c, "Password minimal 8 karakter dan harus mengandung huruf serta angka")
		return
	}

	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		utils.NotFound(c, "User tidak ditemukan")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.OldPassword)); err != nil {
		utils.BadRequest(c, "Password lama salah")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		utils.InternalError(c, "Gagal memproses password")
		return
	}
	user.Password = string(hash)
	config.DB.Save(&user)

	c.JSON(http.StatusOK, gin.H{"message": "Password berhasil diubah"})
}

// ─── DELETE ACCOUNT ──────────────────────────────────────────────────────────

func DeleteAccount(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var input struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.BadRequest(c, "Format request tidak valid")
		return
	}

	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		utils.NotFound(c, "User tidak ditemukan")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		utils.Unauthorized(c, "Password salah")
		return
	}

	// Soft delete: nonaktifkan akun
	user.IsActive = false
	config.DB.Save(&user)

	c.JSON(http.StatusOK, gin.H{"message": "Akun berhasil dihapus"})
}