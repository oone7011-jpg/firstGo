package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"kir-sekolah/models"

	"github.com/jung-kurt/gofpdf"
	"github.com/skip2/go-qrcode"
	"gorm.io/gorm"
)

type QRService struct {
	DB          *gorm.DB
	DownloadDir string
}

func NewQRService(db *gorm.DB, downloadDir string) *QRService {
	return &QRService{DB: db, DownloadDir: downloadDir}
}

// GenerateQRCodePNG membuat file PNG QR Code untuk satu barang
// File disimpan ke disk dan path-nya dikembalikan
func (s *QRService) GenerateQRCodePNG(code string) (string, error) {
	// Buat QR Code dengan ukuran 256x256
	fileName := fmt.Sprintf("qr_%s.png", sanitizeFilename(code))
	filePath := filepath.Join(s.DownloadDir, fileName)

	// QR Code berisi URL endpoint mobile untuk scan
	// Format: kode_barang (bisa juga URL seperti https://app.kir-sekolah.id/scan?code=INV-001)
	qrContent := code
	err := qrcode.WriteFile(qrContent, qrcode.Medium, 256, filePath)
	if err != nil {
		return "", err
	}

	return filePath, nil
}

// GenerateQRCodeBytes membuat QR Code dan mengembalikan sebagai byte[]
func (s *QRService) GenerateQRCodeBytes(code string) ([]byte, error) {
	png, err := qrcode.Encode(code, qrcode.Medium, 256)
	if err != nil {
		return nil, err
	}
	return png, nil
}

// GenerateBulkLabelsPDF membuat PDF berisi label QR Code untuk multiple barang
// Layout: 3 kolom x 8 baris = 24 label per halaman A4
func (s *QRService) GenerateBulkLabelsPDF(itemIDs []uint) (string, error) {
	// Load items
	var items []models.Item
	if err := s.DB.Preload("Room").Where("id IN ?", itemIDs).Find(&items).Error; err != nil {
		return "", err
	}
	if len(items) == 0 {
		return "", fmt.Errorf("tidak ada barang yang dipilih")
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 10)
	pdf.AddPage()

	// Layout: 3 kolom x 8 baris = 24 labels per A4 page
	const (
		cols         = 3
		rows         = 8
		labelW       = 60.0
		labelH       = 35.0
		marginX      = 15.0
		marginY      = 15.0
		spacingX     = 5.0
		spacingY     = 2.0
		qrSize       = 22.0
	)

	currentCol := 0
	currentRow := 0

	for _, item := range items {
		if currentCol == 0 && currentRow == 0 && pdf.PageNo() > 1 {
			// Sudah penuh, halaman baru sudah dibuat
		}

		x := marginX + float64(currentCol)*(labelW+spacingX)
		y := marginY + float64(currentRow)*(labelH+spacingY)

		// Border label
		pdf.SetDrawColor(200, 200, 200)
		pdf.SetLineWidth(0.2)
		pdf.Rect(x, y, labelW, labelH, "D")

		// Generate QR Code sementara
		qrBytes, err := s.GenerateQRCodeBytes(item.Code)
		if err != nil {
			continue
		}

		// Simpan QR code ke temporary file agar bisa diembed
		tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("qr_%s.png", sanitizeFilename(item.Code)))
		if err := os.WriteFile(tmpFile, qrBytes, 0644); err != nil {
			continue
		}

		// Posisi QR di kiri label
		qrX := x + 2
		qrY := y + (labelH-qrSize)/2
		pdf.ImageOptions(tmpFile, qrX, qrY, qrSize, qrSize, false, gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}, 0, "")

		// Teks di kanan QR
		textX := qrX + qrSize + 3
		textY := y + 5
		pdf.SetFont("Arial", "B", 7)
		pdf.SetXY(textX, textY)
		pdf.MultiCell(labelW-qrSize-7, 3, "KIR SEKOLAH", "", "L", false)
		pdf.SetFont("Arial", "B", 8)
		pdf.SetXY(textX, textY+4)
		pdf.MultiCell(labelW-qrSize-7, 3.5, truncate(item.Name, 25), "", "L", false)
		pdf.SetFont("Arial", "", 7)
		pdf.SetXY(textX, textY+11)
		pdf.MultiCell(labelW-qrSize-7, 3, "Kode: "+item.Code, "", "L", false)
		if item.Room != nil {
			pdf.SetXY(textX, textY+14)
			pdf.MultiCell(labelW-qrSize-7, 3, truncate(item.Room.Name, 25), "", "L", false)
		}

		// Next cell
		currentCol++
		if currentCol >= cols {
			currentCol = 0
			currentRow++
			if currentRow >= rows {
				currentRow = 0
				pdf.AddPage()
			}
		}

		// Hapus temp file
		os.Remove(tmpFile)
	}

	// Save PDF
	outputFile := filepath.Join(s.DownloadDir, "qr_labels_bulk.pdf")
	if err := pdf.OutputFileAndClose(outputFile); err != nil {
		return "", err
	}

	return outputFile, nil
}

// GenerateSingleLabelPDF membuat PDF berisi 1 label QR Code besar (untuk printing)
func (s *QRService) GenerateSingleLabelPDF(itemID uint) (string, error) {
	var item models.Item
	if err := s.DB.Preload("Room").First(&item, itemID).Error; err != nil {
		return "", err
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// Label 80mm x 50mm di tengah halaman
	labelW := 80.0
	labelH := 50.0
	pageW, pageH := pdf.GetPageSize()
	x := (pageW - labelW) / 2
	y := (pageH - labelH) / 2

	// Border
	pdf.SetDrawColor(0, 0, 0)
	pdf.SetLineWidth(0.5)
	pdf.Rect(x, y, labelW, labelH, "D")

	// QR Code di kiri
	qrBytes, err := s.GenerateQRCodeBytes(item.Code)
	if err != nil {
		return "", err
	}
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("qr_single_%s.png", sanitizeFilename(item.Code)))
	if err := os.WriteFile(tmpFile, qrBytes, 0644); err != nil {
		return "", err
	}
	defer os.Remove(tmpFile)

	qrSize := 35.0
	pdf.ImageOptions(tmpFile, x+5, y+(labelH-qrSize)/2, qrSize, qrSize, false, gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}, 0, "")

	// Teks
	textX := x + qrSize + 10
	textY := y + 8
	pdf.SetFont("Arial", "B", 12)
	pdf.SetXY(textX, textY)
	pdf.MultiCell(labelW-qrSize-15, 5, "KIR SEKOLAH", "", "L", false)

	pdf.SetFont("Arial", "B", 11)
	pdf.SetXY(textX, textY+8)
	pdf.MultiCell(labelW-qrSize-15, 5, truncate(item.Name, 30), "", "L", false)

	pdf.SetFont("Arial", "", 10)
	pdf.SetXY(textX, textY+18)
	pdf.MultiCell(labelW-qrSize-15, 4, "Kode: "+item.Code, "", "L", false)
	if item.Room != nil {
		pdf.SetXY(textX, textY+23)
		pdf.MultiCell(labelW-qrSize-15, 4, truncate(item.Room.Name, 30), "", "L", false)
	}

	outputFile := filepath.Join(s.DownloadDir, fmt.Sprintf("qr_label_%s.pdf", sanitizeFilename(item.Code)))
	if err := pdf.OutputFileAndClose(outputFile); err != nil {
		return "", err
	}

	return outputFile, nil
}

func sanitizeFilename(s string) string {
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "\\", "-")
	return s
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
