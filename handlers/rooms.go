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

type RoomHandler struct {
	DB *gorm.DB
}

func NewRoomHandler(db *gorm.DB) *RoomHandler {
	return &RoomHandler{DB: db}
}

// List - GET /api/rooms
func (h *RoomHandler) List(c *gin.Context) {
	var rooms []models.Room
	query := h.DB.Order("code ASC")

	// Search by name
	if search := c.Query("search"); search != "" {
		query = query.Where("name LIKE ? OR code LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	query.Find(&rooms)

	// Include item count untuk setiap room
	type RoomWithCount struct {
		models.Room
		ItemCount int64 `json:"item_count"`
	}
	var result []RoomWithCount
	for _, room := range rooms {
		var count int64
		h.DB.Model(&models.Item{}).Where("room_id = ?", room.ID).Count(&count)
		result = append(result, RoomWithCount{
			Room:      room,
			ItemCount: count,
		})
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    result,
	})
}

// Get - GET /api/rooms/:id
func (h *RoomHandler) Get(c *gin.Context) {
	id := c.Param("id")
	var room models.Room
	if err := h.DB.First(&room, id).Error; err != nil {
		c.JSON(http.StatusNotFound, models.APIResponse{
			Success: false,
			Message: "Ruangan tidak ditemukan",
		})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    room,
	})
}

// Create - POST /api/rooms
func (h *RoomHandler) Create(c *gin.Context) {
	var req models.RoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	room := models.Room{
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
	}

	if err := h.DB.Create(&room).Error; err != nil {
		c.JSON(http.StatusConflict, models.APIResponse{
			Success: false,
			Message: "Kode ruangan sudah digunakan atau error: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, models.APIResponse{
		Success: true,
		Message: "Ruangan berhasil dibuat",
		Data:    room,
	})
}

// Update - PUT /api/rooms/:id
func (h *RoomHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var room models.Room
	if err := h.DB.First(&room, id).Error; err != nil {
		c.JSON(http.StatusNotFound, models.APIResponse{
			Success: false,
			Message: "Ruangan tidak ditemukan",
		})
		return
	}

	var req models.RoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	room.Code = req.Code
	room.Name = req.Name
	room.Description = req.Description

	if err := h.DB.Save(&room).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Gagal update ruangan",
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Ruangan berhasil diupdate",
		Data:    room,
	})
}

// Delete - DELETE /api/rooms/:id
func (h *RoomHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	// Cek apakah masih ada barang di ruangan ini
	var count int64
	h.DB.Model(&models.Item{}).Where("room_id = ?", id).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Tidak dapat menghapus ruangan yang masih memiliki " + strconv.FormatInt(count, 10) + " barang",
		})
		return
	}

	if err := h.DB.Delete(&models.Room{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Gagal menghapus ruangan",
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Ruangan berhasil dihapus",
	})
}

// GetItems - GET /api/rooms/:id/items - daftar barang di ruangan tertentu
func (h *RoomHandler) GetItems(c *gin.Context) {
	id := c.Param("id")
	var items []models.Item
	if err := h.DB.Where("room_id = ?", id).Order("code ASC").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Gagal mengambil daftar barang",
		})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    items,
	})
}

// unused but kept for reference
var _ = time.Now
var _ = middleware.GetUserID
