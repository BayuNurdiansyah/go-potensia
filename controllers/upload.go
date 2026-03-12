package controllers

import (
	"net/http"

	"go-potensia/config"
	"go-potensia/models"
	"go-potensia/utils"

	"github.com/gin-gonic/gin"
)

// UploadAvatar godoc
// POST /upload/avatar
// Upload foto profil untuk user yang sedang login (mentor atau parent).
// Content-Type: multipart/form-data
// Field name: "avatar"
func UploadAvatar(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	// 1. Parse multipart form (limit 5MB + buffer)
	if err := c.Request.ParseMultipartForm(5 << 20); err != nil {
		utils.BadRequest(c, "Gagal memproses form upload")
		return
	}

	fileHeader, err := c.FormFile("avatar")
	if err != nil {
		utils.BadRequest(c, "Field 'avatar' tidak ditemukan dalam request")
		return
	}

	// 2. Validate extension & size before opening
	if err := utils.ValidateAvatarFile(fileHeader); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	// 3. Open file
	file, err := fileHeader.Open()
	if err != nil {
		utils.InternalError(c, "Gagal membuka file")
		return
	}
	defer file.Close()

	// 4. Get current user to determine folder and delete old avatar
	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		utils.NotFound(c, "User tidak ditemukan")
		return
	}

	// Folder = role (mentor / parent)
	folder := string(user.Role)

	// 5. Upload ke Supabase Storage
	storage := utils.NewSupabaseStorage()
	publicURL, err := storage.UploadAvatar(file, fileHeader, folder, userID)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": err.Error()})
		return
	}

	// 6. Hapus avatar lama (async, tidak block response)
	if user.AvatarURL != nil && *user.AvatarURL != "" {
		go func(oldURL string) {
			_ = storage.DeleteAvatar(oldURL)
		}(*user.AvatarURL)
	}

	// 7. Update avatar_url di tabel users
	if err := config.DB.Model(&user).Update("avatar_url", publicURL).Error; err != nil {
		utils.InternalError(c, "Gagal menyimpan URL avatar")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Foto profil berhasil diperbarui",
		"avatar_url": publicURL,
	})
}

// DeleteAvatar godoc
// DELETE /upload/avatar
// Hapus foto profil user yang sedang login.
func DeleteAvatar(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		utils.NotFound(c, "User tidak ditemukan")
		return
	}

	if user.AvatarURL == nil || *user.AvatarURL == "" {
		utils.BadRequest(c, "Tidak ada foto profil untuk dihapus")
		return
	}

	// Hapus dari Supabase Storage
	storage := utils.NewSupabaseStorage()
	if err := storage.DeleteAvatar(*user.AvatarURL); err != nil {
		// Log tapi tetap lanjut hapus dari DB
		// (file mungkin sudah tidak ada di storage)
		_ = err
	}

	// Kosongkan avatar_url di DB
	if err := config.DB.Model(&user).Update("avatar_url", nil).Error; err != nil {
		utils.InternalError(c, "Gagal menghapus foto profil")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Foto profil berhasil dihapus"})
}