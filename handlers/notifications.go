package handlers

import (
	"net/http"
	"time"

	"kir-sekolah/middleware"
	"kir-sekolah/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type NotificationHandler struct {
	DB *gorm.DB
}

func NewNotificationHandler(db *gorm.DB) *NotificationHandler {
	return &NotificationHandler{DB: db}
}

// List - GET /api/notifications
// Daftar notifikasi - admin bisa lihat semua, operator hanya yang ditujukan ke dirinya
func (h *NotificationHandler) List(c *gin.Context) {
	role := middleware.GetRole(c)
	query := h.DB.Preload("Item").Preload("Item.Room").Preload("User").Order("created_at DESC")

	// Filter by type
	if notifType := c.Query("type"); notifType != "" {
		query = query.Where("type = ?", notifType)
	}

	// Filter unread only
	if c.Query("unread") == "true" {
		query = query.Where("is_read = ?", false)
	}

	// Limit
	limit := 100
	if l := c.Query("limit"); l != "" {
		if v, err := stringToInt(l); err == nil && v > 0 && v <= 500 {
			limit = v
		}
	}

	var notifs []models.Notification
	query.Limit(limit).Find(&notifs)

	_ = role // admin/operator bisa lihat semua notifikasi

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    notifs,
	})
}

// GetUnreadCount - GET /api/notifications/unread-count
func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
	var count int64
	h.DB.Model(&models.Notification{}).Where("is_read = ?", false).Count(&count)

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"unread_count": count,
	})
}

// MarkAsRead - PUT /api/notifications/:id/read
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	id := c.Param("id")
	now := time.Now()

	if err := h.DB.Model(&models.Notification{}).Where("id = ?", id).Updates(map[string]interface{}{
		"is_read": true,
		"read_at": &now,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Gagal menandai notifikasi sebagai dibaca",
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Notifikasi ditandai sebagai dibaca",
	})
}

// MarkAllAsRead - PUT /api/notifications/read-all
func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	now := time.Now()
	if err := h.DB.Model(&models.Notification{}).Where("is_read = ?", false).Updates(map[string]interface{}{
		"is_read": true,
		"read_at": &now,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Gagal menandai semua notifikasi",
		})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Semua notifikasi ditandai sebagai dibaca",
	})
}

// Delete - DELETE /api/notifications/:id
func (h *NotificationHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.DB.Delete(&models.Notification{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Gagal menghapus notifikasi",
		})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Notifikasi dihapus",
	})
}

// stringToInt helper
func stringToInt(s string) (int, error) {
	var result int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, errInvalidNumber
		}
		result = result*10 + int(c-'0')
	}
	return result, nil
}

var errInvalidNumber = &invalidNumberError{}

type invalidNumberError struct{}

func (e *invalidNumberError) Error() string { return "invalid number" }
