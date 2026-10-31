package handlers

import (
	"net/http"
	"path/filepath"
	"strconv"

	"kir-sekolah/models"
	"kir-sekolah/services"

	"github.com/gin-gonic/gin"
)

type ReportHandler struct {
	PDFService   *services.PDFService
	ExcelService *services.ExcelService
	QRService    *services.QRService
}

func NewReportHandler(pdf *services.PDFService, excel *services.ExcelService, qr *services.QRService) *ReportHandler {
	return &ReportHandler{PDFService: pdf, ExcelService: excel, QRService: qr}
}

// GenerateKIRReportPDF - GET /api/reports/kir/pdf
// Query: room_id, condition
func (h *ReportHandler) GenerateKIRReportPDF(c *gin.Context) {
	roomID, _ := strconv.ParseUint(c.Query("room_id"), 10, 32)
	condition := c.Query("condition")

	filePath, err := h.PDFService.GenerateKIRReport(uint(roomID), condition)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Gagal generate PDF: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "PDF berhasil dibuat",
		Data: gin.H{
			"file_name": filepath.Base(filePath),
			"file_path": filePath,
			"download_url": "/api/reports/download/" + filepath.Base(filePath),
		},
	})
}

// GenerateFinanceReportExcel - GET /api/reports/finance/excel
// Query: year, month
func (h *ReportHandler) GenerateFinanceReportExcel(c *gin.Context) {
	year, _ := strconv.Atoi(c.DefaultQuery("year", "0"))
	month, _ := strconv.Atoi(c.DefaultQuery("month", "0"))

	filePath, err := h.ExcelService.GenerateMonthlyFinanceReport(year, month)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Gagal generate Excel: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Excel berhasil dibuat",
		Data: gin.H{
			"file_name":     filepath.Base(filePath),
			"file_path":     filePath,
			"download_url":  "/api/reports/download/" + filepath.Base(filePath),
		},
	})
}

// GenerateTransferReportPDF - GET /api/reports/transfers/pdf
// Query: start_date, end_date
func (h *ReportHandler) GenerateTransferReportPDF(c *gin.Context) {
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	filePath, err := h.PDFService.GenerateTransferReportPDF(startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Gagal generate PDF: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "PDF laporan pemindahan berhasil dibuat",
		Data: gin.H{
			"file_name":    filepath.Base(filePath),
			"file_path":    filePath,
			"download_url": "/api/reports/download/" + filepath.Base(filePath),
		},
	})
}

// GenerateQRCodePNG - GET /api/qrcode/:code/png
func (h *ReportHandler) GenerateQRCodePNG(c *gin.Context) {
	code := c.Param("code")

	// Cari barang dulu
	var item models.Item
	if err := h.QRService.DB.Where("code = ?", code).First(&item).Error; err != nil {
		c.JSON(http.StatusNotFound, models.APIResponse{
			Success: false,
			Message: "Barang tidak ditemukan",
		})
		return
	}

	filePath, err := h.QRService.GenerateQRCodePNG(item.Code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Gagal generate QR code",
		})
		return
	}

	c.Header("Content-Type", "image/png")
	c.File(filePath)
}

// GenerateSingleLabelPDF - GET /api/qrcode/items/:id/label-pdf
func (h *ReportHandler) GenerateSingleLabelPDF(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	filePath, err := h.QRService.GenerateSingleLabelPDF(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Gagal generate label PDF: " + err.Error(),
		})
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", "inline; filename="+filepath.Base(filePath))
	c.File(filePath)
}

// GenerateBulkLabelsPDF - POST /api/qrcode/bulk-labels
// Body: { "item_ids": [1, 2, 3, ...] }
func (h *ReportHandler) GenerateBulkLabelsPDF(c *gin.Context) {
	var req struct {
		ItemIDs []uint `json:"item_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	if len(req.ItemIDs) == 0 {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Item IDs tidak boleh kosong",
		})
		return
	}

	filePath, err := h.QRService.GenerateBulkLabelsPDF(req.ItemIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Gagal generate label PDF: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Label PDF berhasil dibuat",
		Data: gin.H{
			"file_name":    filepath.Base(filePath),
			"file_path":    filePath,
			"download_url": "/api/reports/download/" + filepath.Base(filePath),
		},
	})
}

// DownloadFile - GET /api/reports/download/:filename
func (h *ReportHandler) DownloadFile(c *gin.Context) {
	filename := c.Param("filename")
	filePath := filepath.Join(h.QRService.DownloadDir, filename)
	c.File(filePath)
}
