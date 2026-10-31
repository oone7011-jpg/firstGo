package handlers

import (
	"net/http"
	"strconv"
	"time"

	"kir-sekolah/middleware"
	"kir-sekolah/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ItemHandler struct {
	DB *gorm.DB
}

func NewItemHandler(db *gorm.DB) *ItemHandler {
	return &ItemHandler{DB: db}
}

// List - GET /api/items
// Query params: search, room_id, condition, page, limit
func (h *ItemHandler) List(c *gin.Context) {
	query := h.DB.Model(&models.Item{}).Preload("Room")

	// Search
	if search := c.Query("search"); search != "" {
		query = query.Where("items.name LIKE ? OR items.code LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// Filter by room
	if roomID := c.Query("room_id"); roomID != "" {
		query = query.Where("items.room_id = ?", roomID)
	}

	// Filter by condition
	if condition := c.Query("condition"); condition != "" {
		query = query.Where("items.condition = ?", condition)
	}

	// Pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 500 {
		limit = 50
	}
	offset := (page - 1) * limit

	var total int64
	query.Count(&total)

	var items []models.Item
	if err := query.Order("items.code ASC").Offset(offset).Limit(limit).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Gagal mengambil data barang",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    items,
		"meta": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"total_page": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// Get - GET /api/items/:id
func (h *ItemHandler) Get(c *gin.Context) {
	id := c.Param("id")
	var item models.Item
	if err := h.DB.Preload("Room").First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, models.APIResponse{
			Success: false,
			Message: "Barang tidak ditemukan",
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    item,
	})
}

// GetByCode - GET /api/items/code/:code
func (h *ItemHandler) GetByCode(c *gin.Context) {
	code := c.Param("code")
	var item models.Item
	if err := h.DB.Preload("Room").Where("code = ?", code).First(&item).Error; err != nil {
		c.JSON(http.StatusNotFound, models.APIResponse{
			Success: false,
			Message: "Barang dengan kode '" + code + "' tidak ditemukan",
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    item,
	})
}

// Create - POST /api/items
func (h *ItemHandler) Create(c *gin.Context) {
	var req models.ItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	// Validasi condition
	if req.Condition == "" {
		req.Condition = models.KondisiBaik
	}
	if !isValidCondition(req.Condition) {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Kondisi harus salah satu: Baik, Rusak Ringan, Rusak Berat",
		})
		return
	}

	// Validasi room exists
	var room models.Room
	if err := h.DB.First(&room, req.RoomID).Error; err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Ruangan tidak ditemukan",
		})
		return
	}

	item := models.Item{
		Code:        req.Code,
		Name:        req.Name,
		RoomID:      req.RoomID,
		Condition:   req.Condition,
		Price:       req.Price,
		Description: req.Description,
	}

	// Parse acquisition date
	if req.AcquisitionDate != "" {
		if t, err := time.Parse("2006-01-02", req.AcquisitionDate); err == nil {
			item.AcquisitionDate = &t
		}
	}

	if err := h.DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusConflict, models.APIResponse{
			Success: false,
			Message: "Kode barang sudah digunakan atau error: " + err.Error(),
		})
		return
	}

	// Load room relation
	h.DB.Preload("Room").First(&item, item.ID)

	c.JSON(http.StatusCreated, models.APIResponse{
		Success: true,
		Message: "Barang berhasil dibuat",
		Data:    item,
	})
}

// Update - PUT /api/items/:id
func (h *ItemHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var item models.Item
	if err := h.DB.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, models.APIResponse{
			Success: false,
			Message: "Barang tidak ditemukan",
		})
		return
	}

	var req models.ItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	if !isValidCondition(req.Condition) && req.Condition != "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Kondisi harus salah satu: Baik, Rusak Ringan, Rusak Berat",
		})
		return
	}

	// Jika kondisi berubah, catat history
	oldCondition := item.Condition
	newCondition := req.Condition
	if newCondition == "" {
		newCondition = oldCondition
	}

	if oldCondition != newCondition {
		// Catat history perubahan kondisi
		userID := middleware.GetUserID(c)
		history := models.ItemConditionHistory{
			ItemID:       item.ID,
			OldCondition: oldCondition,
			NewCondition: newCondition,
			UserID:       userID,
			Note:         "Update melalui web/API",
		}
		h.DB.Create(&history)

		// Jika berubah jadi Rusak Berat, kirim notifikasi ke admin
		if newCondition == models.KondisiRusakBerat {
			h.notifyAdminsDamage(c, &item, oldCondition, newCondition, userID)
		}
	}

	item.Code = req.Code
	item.Name = req.Name
	item.RoomID = req.RoomID
	if req.Condition != "" {
		item.Condition = req.Condition
	}
	item.Price = req.Price
	item.Description = req.Description

	if req.AcquisitionDate != "" {
		if t, err := time.Parse("2006-01-02", req.AcquisitionDate); err == nil {
			item.AcquisitionDate = &t
		}
	}

	if err := h.DB.Save(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Gagal update barang",
		})
		return
	}

	h.DB.Preload("Room").First(&item, item.ID)

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Barang berhasil diupdate",
		Data:    item,
	})
}

// Delete - DELETE /api/items/:id
func (h *ItemHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.DB.Delete(&models.Item{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Gagal menghapus barang",
		})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Barang berhasil dihapus",
	})
}

// GetConditionHistory - GET /api/items/:id/condition-history
func (h *ItemHandler) GetConditionHistory(c *gin.Context) {
	id := c.Param("id")
	var history []models.ItemConditionHistory
	if err := h.DB.Preload("User").Where("item_id = ?", id).Order("created_at DESC").Find(&history).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Gagal mengambil riwayat kondisi",
		})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    history,
	})
}

// GetTransferHistory - GET /api/items/:id/transfer-history
func (h *ItemHandler) GetTransferHistory(c *gin.Context) {
	id := c.Param("id")
	var history []models.TransferHistory
	if err := h.DB.
		Preload("FromRoom").
		Preload("ToRoom").
		Preload("User").
		Where("item_id = ?", id).
		Order("transfer_date DESC").
		Find(&history).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Gagal mengambil riwayat pemindahan",
		})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    history,
	})
}

// isValidCondition validasi nilai kondisi
func isValidCondition(c string) bool {
	return c == models.KondisiBaik || c == models.KondisiRusakRingan || c == models.KondisiRusakBerat
}

// notifyAdminsDamage mengirim notifikasi ke semua admin saat ada barang rusak berat
func (h *ItemHandler) notifyAdminsDamage(c *gin.Context, item *models.Item, oldCondition, newCondition string, userID uint) {
	// Load room untuk info
	h.DB.First(&item.Room, item.RoomID)

	var user models.User
	h.DB.First(&user, userID)
	username := user.Username
	if user.FullName != "" {
		username = user.FullName
	}

	notif := models.Notification{
		Type:    "damage_severe",
		Title:   "⚠️ Peringatan: Barang Rusak Berat",
		Message: "Barang '" + item.Name + "' (Kode: " + item.Code + ") di ruangan " + item.Room.Name + " telah berubah dari '" + oldCondition + "' menjadi '" + newCondition + "' oleh " + username + ". Segera proses perbaikan!",
		ItemID:  &item.ID,
		UserID:  &userID,
	}
	h.DB.Create(&notif)
}
