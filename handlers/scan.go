package handlers

import (
	"net/http"
	"time"

	"kir-sekolah/middleware"
	"kir-sekolah/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ScanHandler struct {
	DB *gorm.DB
}

func NewScanHandler(db *gorm.DB) *ScanHandler {
	return &ScanHandler{DB: db}
}

// VerifyScan - POST /api/scan/verify
// Endpoint mobile untuk verifikasi QR Code - memvalidasi apakah barang terdaftar atau tidak
func (h *ScanHandler) VerifyScan(c *gin.Context) {
	var req models.ScanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	// Cari barang berdasarkan kode
	var item models.Item
	if err := h.DB.Preload("Room").Where("code = ?", req.Code).First(&item).Error; err != nil {
		// Barang TIDAK terdaftar
		c.JSON(http.StatusOK, gin.H{
			"success":       true,
			"registered":    false,
			"message":       "QR Code tidak terdaftar di database",
			"scanned_code":  req.Code,
			"timestamp":     time.Now(),
			"scanned_by":    middleware.GetUsername(c),
			"location": gin.H{
				"latitude":  req.Latitude,
				"longitude": req.Longitude,
			},
		})
		return
	}

	// Barang TERDAFTA - kembalikan info lengkap
	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"registered":   true,
		"message":      "Barang terdaftar",
		"timestamp":    time.Now(),
		"scanned_by":   middleware.GetUsername(c),
		"item":         item,
		"actions": gin.H{
			"can_update_condition": true,
			"can_transfer":         true,
			"conditions":           []string{models.KondisiBaik, models.KondisiRusakRingan, models.KondisiRusakBerat},
		},
		"location": gin.H{
			"latitude":  req.Latitude,
			"longitude": req.Longitude,
		},
	})
}

// UpdateCondition - POST /api/scan/update-condition
// Update kondisi barang langsung dari hasil scan (untuk sidak keliling operator)
func (h *ScanHandler) UpdateCondition(c *gin.Context) {
	var req models.UpdateConditionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	// Validasi condition
	if !isValidCondition(req.Condition) {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Kondisi harus salah satu: Baik, Rusak Ringan, Rusak Berat",
		})
		return
	}

	// Cari barang by code
	var item models.Item
	if err := h.DB.Preload("Room").Where("code = ?", req.Code).First(&item).Error; err != nil {
		c.JSON(http.StatusNotFound, models.APIResponse{
			Success: false,
			Message: "Barang dengan kode '" + req.Code + "' tidak ditemukan",
		})
		return
	}

	oldCondition := item.Condition
	newCondition := req.Condition
	userID := middleware.GetUserID(c)
	username := middleware.GetUsername(c)

	// Jika kondisi sama, tidak perlu update
	if oldCondition == newCondition {
		c.JSON(http.StatusOK, models.APIResponse{
			Success: true,
			Message: "Kondisi barang tidak berubah (sudah " + oldCondition + ")",
			Data: gin.H{
				"item":            item,
				"old_condition":   oldCondition,
				"new_condition":   newCondition,
				"condition_changed": false,
			},
		})
		return
	}

	// Update kondisi
	item.Condition = newCondition
	if err := h.DB.Save(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Gagal update kondisi barang",
		})
		return
	}

	// Catat history
	history := models.ItemConditionHistory{
		ItemID:       item.ID,
		OldCondition: oldCondition,
		NewCondition: newCondition,
		UserID:       userID,
		Note:         req.Note,
		Latitude:     req.Latitude,
		Longitude:    req.Longitude,
	}
	if req.Note == "" {
		history.Note = "Update dari scan QR oleh " + username
	}
	h.DB.Create(&history)

	// Jika rusak berat, kirim notifikasi ke semua admin
	if newCondition == models.KondisiRusakBerat {
		h.notifyAdminsDamage(&item, oldCondition, newCondition, userID, username)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Kondisi barang berhasil diupdate dari '" + oldCondition + "' ke '" + newCondition + "'",
		"data": gin.H{
			"item":              item,
			"old_condition":     oldCondition,
			"new_condition":     newCondition,
			"condition_changed": true,
			"updated_by":        username,
			"updated_at":        time.Now(),
			"location": gin.H{
				"latitude":  req.Latitude,
				"longitude": req.Longitude,
			},
			"admin_notified": newCondition == models.KondisiRusakBerat,
		},
	})
}

// notifyAdminsDamage - kirim notifikasi ke semua admin saat barang rusak berat
func (h *ScanHandler) notifyAdminsDamage(item *models.Item, oldCondition, newCondition string, userID uint, username string) {
	roomName := ""
	if item.Room != nil {
		roomName = item.Room.Name
	}

	notif := models.Notification{
		Type:    "damage_severe",
		Title:   "⚠️ PERINGATAN: Barang Rusak Berat (dari Scan Mobile)",
		Message: "Barang '" + item.Name + "' (Kode: " + item.Code + ") di " + roomName + " berubah dari '" + oldCondition + "' menjadi '" + newCondition + "' melalui scan QR oleh " + username + ". Mohon segera proses perbaikan!",
		ItemID:  &item.ID,
		UserID:  &userID,
	}
	h.DB.Create(&notif)
}

// ScanHistory - GET /api/scan/history - riwayat scan yang mengubah kondisi
func (h *ScanHandler) ScanHistory(c *gin.Context) {
	var history []models.ItemConditionHistory
	query := h.DB.
		Preload("Item").
		Preload("Item.Room").
		Preload("User").
		Order("created_at DESC")

	// Filter by user
	if userID := c.Query("user_id"); userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	// Filter by date range
	if startDate := c.Query("start_date"); startDate != "" {
		if t, err := time.Parse("2006-01-02", startDate); err == nil {
			query = query.Where("created_at >= ?", t)
		}
	}
	if endDate := c.Query("end_date"); endDate != "" {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			query = query.Where("created_at <= ?", t.Add(24*time.Hour))
		}
	}

	query.Find(&history)
	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    history,
	})
}
