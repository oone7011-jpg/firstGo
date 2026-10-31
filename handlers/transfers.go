package handlers

import (
	"net/http"
	"time"

	"kir-sekolah/middleware"
	"kir-sekolah/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type TransferHandler struct {
	DB *gorm.DB
}

func NewTransferHandler(db *gorm.DB) *TransferHandler {
	return &TransferHandler{DB: db}
}

// Transfer - POST /api/transfers - memindahkan barang antar ruangan
func (h *TransferHandler) Transfer(c *gin.Context) {
	var req models.TransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	// Load item
	var item models.Item
	if err := h.DB.Preload("Room").First(&item, req.ItemID).Error; err != nil {
		c.JSON(http.StatusNotFound, models.APIResponse{
			Success: false,
			Message: "Barang tidak ditemukan",
		})
		return
	}

	// Cek apakah barang sudah ada di ruangan tujuan
	if item.RoomID == req.ToRoomID {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Barang sudah berada di ruangan tujuan",
		})
		return
	}

	// Validasi ruangan tujuan
	var toRoom models.Room
	if err := h.DB.First(&toRoom, req.ToRoomID).Error; err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Ruangan tujuan tidak ditemukan",
		})
		return
	}

	// Parse transfer date
	transferDate := time.Now()
	if req.TransferDate != "" {
		if t, err := time.Parse("2006-01-02", req.TransferDate); err == nil {
			transferDate = t
		}
	}

	userID := middleware.GetUserID(c)

	// Simpan room lama sebelum update
	oldRoomID := item.RoomID

	// Buat record transfer history
	transfer := models.TransferHistory{
		ItemID:       item.ID,
		FromRoomID:   oldRoomID,
		ToRoomID:     req.ToRoomID,
		UserID:       userID,
		Reason:       req.Reason,
		TransferDate: transferDate,
	}

	// Transaction: update item room + create transfer history
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		// Update room item
		if err := tx.Model(&item).Update("room_id", req.ToRoomID).Error; err != nil {
			return err
		}
		// Create history
		if err := tx.Create(&transfer).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Gagal memindahkan barang: " + err.Error(),
		})
		return
	}

	// Load ulang dengan relasi
	h.DB.Preload("Item").Preload("FromRoom").Preload("ToRoom").Preload("User").First(&transfer, transfer.ID)

	// Buat notifikasi ke admin
	var fromRoom models.Room
	h.DB.First(&fromRoom, oldRoomID)
	notif := models.Notification{
		Type:    "item_transfer",
		Title:   "📦 Pemindahan Barang",
		Message: "Barang '" + item.Name + "' (Kode: " + item.Code + ") dipindahkan dari " + fromRoom.Name + " ke " + toRoom.Name,
		ItemID:  &item.ID,
		UserID:  &userID,
	}
	h.DB.Create(&notif)

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Barang berhasil dipindahkan dari " + fromRoom.Name + " ke " + toRoom.Name,
		Data:    transfer,
	})
}

// List - GET /api/transfers - daftar semua riwayat pemindahan
func (h *TransferHandler) List(c *gin.Context) {
	var transfers []models.TransferHistory
	query := h.DB.
		Preload("Item").
		Preload("FromRoom").
		Preload("ToRoom").
		Preload("User").
		Order("transfer_date DESC")

	// Filter by item
	if itemID := c.Query("item_id"); itemID != "" {
		query = query.Where("item_id = ?", itemID)
	}

	// Filter by date range
	if startDate := c.Query("start_date"); startDate != "" {
		if t, err := time.Parse("2006-01-02", startDate); err == nil {
			query = query.Where("transfer_date >= ?", t)
		}
	}
	if endDate := c.Query("end_date"); endDate != "" {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			query = query.Where("transfer_date <= ?", t.Add(24*time.Hour))
		}
	}

	query.Find(&transfers)

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    transfers,
	})
}

// Get - GET /api/transfers/:id
func (h *TransferHandler) Get(c *gin.Context) {
	id := c.Param("id")
	var transfer models.TransferHistory
	if err := h.DB.
		Preload("Item").
		Preload("FromRoom").
		Preload("ToRoom").
		Preload("User").
		First(&transfer, id).Error; err != nil {
		c.JSON(http.StatusNotFound, models.APIResponse{
			Success: false,
			Message: "Riwayat pemindahan tidak ditemukan",
		})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    transfer,
	})
}
