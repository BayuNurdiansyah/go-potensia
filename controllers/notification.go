package controllers

import (
	"net/http"

	"go-potensia/config"
	"go-potensia/models"
	"go-potensia/utils"

	"github.com/gin-gonic/gin"
)

func GetNotifications(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var notifications []models.Notification
	config.DB.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(50).
		Find(&notifications)

	var unreadCount int64
	config.DB.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = false", userID).
		Count(&unreadCount)

	c.JSON(http.StatusOK, gin.H{
		"notifications": notifications,
		"unread_count":  unreadCount,
	})
}

func MarkNotificationRead(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	notifIDStr := c.Param("notif_id")

	if notifIDStr == "all" {
		config.DB.Model(&models.Notification{}).
			Where("user_id = ?", userID).
			Update("is_read", true)
		c.JSON(http.StatusOK, gin.H{"message": "Semua notifikasi ditandai sudah dibaca"})
		return
	}

	var notif models.Notification
	if err := config.DB.Where("id = ? AND user_id = ?", notifIDStr, userID).First(&notif).Error; err != nil {
		utils.NotFound(c, "Notifikasi tidak ditemukan")
		return
	}
	notif.IsRead = true
	config.DB.Save(&notif)

	c.JSON(http.StatusOK, gin.H{"message": "Notifikasi ditandai sudah dibaca"})
}

func UpdateNotificationPreferences(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var input struct {
		Reminder bool `json:"reminder"`
		Promo    bool `json:"promo"`
		Schedule bool `json:"schedule"`
		Info     bool `json:"info"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.BadRequest(c, "Format request tidak valid")
		return
	}

	config.DB.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"notif_reminder": input.Reminder,
		"notif_promo":    input.Promo,
		"notif_schedule": input.Schedule,
		"notif_info":     input.Info,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Preferensi notifikasi berhasil diperbarui"})
}

func GetNotificationPreferences(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		utils.NotFound(c, "User tidak ditemukan")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"reminder": user.NotifReminder,
		"promo":    user.NotifPromo,
		"schedule": user.NotifSchedule,
		"info":     user.NotifInfo,
	})
}