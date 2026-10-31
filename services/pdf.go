package services

import (
        "fmt"
        "path/filepath"
        "time"

        "kir-sekolah/models"

        "github.com/jung-kurt/gofpdf"
        "gorm.io/gorm"
)

type PDFService struct {
        DB          *gorm.DB
        DownloadDir string
}

func NewPDFService(db *gorm.DB, downloadDir string) *PDFService {
        return &PDFService{DB: db, DownloadDir: downloadDir}
}

// GenerateKIRReport membuat laporan PDF Kartu Inventaris Ruang
// Filter: roomID (opsional), condition (opsional)
func (s *PDFService) GenerateKIRReport(roomID uint, condition string) (string, error) {
        // Gunakan Join ke rooms agar bisa order by rooms.code
        query := s.DB.Joins("LEFT JOIN rooms ON rooms.id = items.room_id").
                Preload("Room").
                Order("rooms.code ASC, items.code ASC")
        if roomID > 0 {
                query = query.Where("items.room_id = ?", roomID)
        }
        if condition != "" {
                query = query.Where("items.condition = ?", condition)
        }

        var items []models.Item
        if err := query.Find(&items).Error; err != nil {
                return "", err
        }

        pdf := gofpdf.New("P", "mm", "A4", "")
        pdf.SetAutoPageBreak(true, 15)
        pdf.SetMargins(15, 15, 15)

        // Cover / Header
        s.addReportHeader(pdf, roomID, condition, len(items))

        // Summary per ruangan
        roomSummary := s.getRoomSummary(items)
        s.addSummaryPage(pdf, roomSummary, len(items))

        // Detail per ruangan
        currentRoomID := uint(0)
        for _, item := range items {
                if item.RoomID != currentRoomID {
                        currentRoomID = item.RoomID
                        pdf.AddPage()
                        s.addRoomHeader(pdf, item.Room)
                }
                s.addItemRow(pdf, &item)
        }

        // Footer summary
        pdf.AddPage()
        s.addConditionSummary(pdf, items)

        // Save
        outputFile := filepath.Join(s.DownloadDir, fmt.Sprintf("laporan_kir_%s.pdf", time.Now().Format("20060102_150405")))
        if err := pdf.OutputFileAndClose(outputFile); err != nil {
                return "", err
        }

        return outputFile, nil
}

func (s *PDFService) addReportHeader(pdf *gofpdf.Fpdf, roomID uint, condition string, totalItems int) {
        pdf.AddPage()
        pdf.SetFont("Arial", "B", 18)
        pdf.Cell(0, 10, "KARTU INVENTARIS RUANG (KIR)")
        pdf.Ln(8)

        pdf.SetFont("Arial", "B", 14)
        pdf.Cell(0, 7, "Laporan Inventaris Sekolah")
        pdf.Ln(8)

        pdf.SetFont("Arial", "", 10)
        pdf.Cell(0, 5, fmt.Sprintf("Tanggal Cetak: %s", time.Now().Format("02 Januari 2006 15:04:05")))
        pdf.Ln(5)

        // Filter info
        filterInfo := "Filter: Semua Ruangan"
        if roomID > 0 {
                var room models.Room
                s.DB.First(&room, roomID)
                if room.ID > 0 {
                        filterInfo = "Filter: Ruangan " + room.Name + " (" + room.Code + ")"
                }
        }
        if condition != "" {
                filterInfo += " | Kondisi: " + condition
        }
        pdf.Cell(0, 5, filterInfo)
        pdf.Ln(5)
        pdf.Cell(0, 5, fmt.Sprintf("Total Barang: %d", totalItems))
        pdf.Ln(10)

        // Garis pemisah
        pdf.SetDrawColor(0, 0, 0)
        pdf.SetLineWidth(0.5)
        pdf.Line(15, pdf.GetY(), 195, pdf.GetY())
        pdf.Ln(5)
}

func (s *PDFService) addSummaryPage(pdf *gofpdf.Fpdf, roomSummary []roomSummaryItem, totalItems int) {
        pdf.AddPage()
        pdf.SetFont("Arial", "B", 14)
        pdf.Cell(0, 8, "Ringkasan per Ruangan")
        pdf.Ln(10)

        // Header tabel
        pdf.SetFont("Arial", "B", 9)
        pdf.SetFillColor(220, 220, 220)
        pdf.CellFormat(20, 7, "Kode", "1", 0, "C", true, 0, "")
        pdf.CellFormat(70, 7, "Nama Ruangan", "1", 0, "L", true, 0, "")
        pdf.CellFormat(25, 7, "Jumlah", "1", 0, "C", true, 0, "")
        pdf.CellFormat(25, 7, "Baik", "1", 0, "C", true, 0, "")
        pdf.CellFormat(25, 7, "Rusak Ringan", "1", 0, "C", true, 0, "")
        pdf.CellFormat(20, 7, "Rusak Berat", "1", 0, "C", true, 0, "")
        pdf.Ln(-1)

        pdf.SetFont("Arial", "", 9)
        totalBaik, totalRusakRingan, totalRusakBerat := 0, 0, 0
        for _, rs := range roomSummary {
                pdf.CellFormat(20, 6, rs.Code, "1", 0, "C", false, 0, "")
                pdf.CellFormat(70, 6, truncate(rs.Name, 35), "1", 0, "L", false, 0, "")
                pdf.CellFormat(25, 6, fmt.Sprintf("%d", rs.Total), "1", 0, "C", false, 0, "")
                pdf.CellFormat(25, 6, fmt.Sprintf("%d", rs.Baik), "1", 0, "C", false, 0, "")
                pdf.CellFormat(25, 6, fmt.Sprintf("%d", rs.RusakRingan), "1", 0, "C", false, 0, "")
                pdf.CellFormat(20, 6, fmt.Sprintf("%d", rs.RusakBerat), "1", 0, "C", false, 0, "")
                pdf.Ln(-1)
                totalBaik += rs.Baik
                totalRusakRingan += rs.RusakRingan
                totalRusakBerat += rs.RusakBerat
        }

        // Total row
        pdf.SetFont("Arial", "B", 9)
        pdf.SetFillColor(240, 240, 240)
        pdf.CellFormat(20, 7, "", "1", 0, "C", true, 0, "")
        pdf.CellFormat(70, 7, "TOTAL", "1", 0, "L", true, 0, "")
        pdf.CellFormat(25, 7, fmt.Sprintf("%d", totalItems), "1", 0, "C", true, 0, "")
        pdf.CellFormat(25, 7, fmt.Sprintf("%d", totalBaik), "1", 0, "C", true, 0, "")
        pdf.CellFormat(25, 7, fmt.Sprintf("%d", totalRusakRingan), "1", 0, "C", true, 0, "")
        pdf.CellFormat(20, 7, fmt.Sprintf("%d", totalRusakBerat), "1", 0, "C", true, 0, "")
        pdf.Ln(-1)
}

type roomSummaryItem struct {
        Code        string
        Name        string
        Total       int
        Baik        int
        RusakRingan int
        RusakBerat  int
}

func (s *PDFService) getRoomSummary(items []models.Item) []roomSummaryItem {
        roomMap := make(map[uint]*roomSummaryItem)
        for _, item := range items {
                if item.Room == nil {
                        continue
                }
                if _, ok := roomMap[item.RoomID]; !ok {
                        roomMap[item.RoomID] = &roomSummaryItem{
                                Code: item.Room.Code,
                                Name: item.Room.Name,
                        }
                }
                rs := roomMap[item.RoomID]
                rs.Total++
                switch item.Condition {
                case models.KondisiBaik:
                        rs.Baik++
                case models.KondisiRusakRingan:
                        rs.RusakRingan++
                case models.KondisiRusakBerat:
                        rs.RusakBerat++
                }
        }
        result := make([]roomSummaryItem, 0, len(roomMap))
        for _, rs := range roomMap {
                result = append(result, *rs)
        }
        return result
}

func (s *PDFService) addRoomHeader(pdf *gofpdf.Fpdf, room *models.Room) {
        if room == nil {
                return
        }
        pdf.SetFont("Arial", "B", 13)
        pdf.SetFillColor(230, 240, 255)
        pdf.CellFormat(0, 8, fmt.Sprintf("RUANGAN: %s (%s)", room.Name, room.Code), "1", 0, "L", true, 0, "")
        pdf.Ln(10)

        // Header tabel
        pdf.SetFont("Arial", "B", 9)
        pdf.SetFillColor(220, 220, 220)
        pdf.CellFormat(25, 7, "Kode", "1", 0, "C", true, 0, "")
        pdf.CellFormat(60, 7, "Nama Barang", "1", 0, "L", true, 0, "")
        pdf.CellFormat(30, 7, "Kondisi", "1", 0, "C", true, 0, "")
        pdf.CellFormat(25, 7, "Tanggal Perolehan", "1", 0, "C", true, 0, "")
        pdf.CellFormat(25, 7, "Harga (Rp)", "1", 0, "R", true, 0, "")
        pdf.CellFormat(20, 7, "Keterangan", "1", 0, "L", true, 0, "")
        pdf.Ln(-1)
}

func (s *PDFService) addItemRow(pdf *gofpdf.Fpdf, item *models.Item) {
        pdf.SetFont("Arial", "", 8)

        // Kondisi color coding
        bgR, bgG, bgB := 255, 255, 255
        switch item.Condition {
        case models.KondisiBaik:
                bgR, bgG, bgB = 220, 255, 220
        case models.KondisiRusakRingan:
                bgR, bgG, bgB = 255, 245, 200
        case models.KondisiRusakBerat:
                bgR, bgG, bgB = 255, 200, 200
        }
        pdf.SetFillColor(bgR, bgG, bgB)

        acqDate := "-"
        if item.AcquisitionDate != nil {
                acqDate = item.AcquisitionDate.Format("02/01/2006")
        }

        pdf.CellFormat(25, 6, item.Code, "1", 0, "C", true, 0, "")
        pdf.CellFormat(60, 6, truncate(item.Name, 30), "1", 0, "L", true, 0, "")
        pdf.CellFormat(30, 6, item.Condition, "1", 0, "C", true, 0, "")
        pdf.CellFormat(25, 6, acqDate, "1", 0, "C", true, 0, "")
        pdf.CellFormat(25, 6, formatRupiah(item.Price), "1", 0, "R", true, 0, "")
        pdf.CellFormat(20, 6, truncate(item.Description, 15), "1", 0, "L", true, 0, "")
        pdf.Ln(-1)
}

func (s *PDFService) addConditionSummary(pdf *gofpdf.Fpdf, items []models.Item) {
        pdf.SetFont("Arial", "B", 14)
        pdf.Cell(0, 8, "Ringkasan Kondisi Barang")
        pdf.Ln(10)

        countBaik, countRusakRingan, countRusakBerat := 0, 0, 0
        totalValue := 0.0
        for _, item := range items {
                switch item.Condition {
                case models.KondisiBaik:
                        countBaik++
                case models.KondisiRusakRingan:
                        countRusakRingan++
                case models.KondisiRusakBerat:
                        countRusakBerat++
                }
                totalValue += item.Price
        }

        pdf.SetFont("Arial", "", 11)
        pdf.CellFormat(80, 8, "Total Barang Kondisi Baik", "1", 0, "L", false, 0, "")
        pdf.CellFormat(40, 8, fmt.Sprintf(": %d", countBaik), "1", 0, "L", false, 0, "")
        pdf.Ln(8)

        pdf.CellFormat(80, 8, "Total Barang Rusak Ringan", "1", 0, "L", false, 0, "")
        pdf.CellFormat(40, 8, fmt.Sprintf(": %d", countRusakRingan), "1", 0, "L", false, 0, "")
        pdf.Ln(8)

        pdf.CellFormat(80, 8, "Total Barang Rusak Berat", "1", 0, "L", false, 0, "")
        pdf.CellFormat(40, 8, fmt.Sprintf(": %d", countRusakBerat), "1", 0, "L", false, 0, "")
        pdf.Ln(8)

        pdf.CellFormat(80, 8, "TOTAL NILAI ASET", "1", 0, "L", false, 0, "")
        pdf.CellFormat(40, 8, ": Rp "+formatRupiah(totalValue), "1", 0, "L", false, 0, "")
        pdf.Ln(15)

        pdf.SetFont("Arial", "I", 9)
        pdf.MultiCell(0, 5, "Catatan: Laporan ini dicetak otomatis oleh Sistem KIR Sekolah. Untuk barang dengan kondisi 'Rusak Berat' harap segera diproses perbaikan atau penghapusan.", "", "L", false)
}

// GenerateTransferReportPDF membuat laporan PDF riwayat pemindahan barang
func (s *PDFService) GenerateTransferReportPDF(startDate, endDate string) (string, error) {
        query := s.DB.
                Preload("Item").
                Preload("FromRoom").
                Preload("ToRoom").
                Preload("User").
                Order("transfer_date DESC")

        if startDate != "" {
                if t, err := time.Parse("2006-01-02", startDate); err == nil {
                        query = query.Where("transfer_date >= ?", t)
                }
        }
        if endDate != "" {
                if t, err := time.Parse("2006-01-02", endDate); err == nil {
                        query = query.Where("transfer_date <= ?", t.Add(24*time.Hour))
                }
        }

        var transfers []models.TransferHistory
        if err := query.Find(&transfers).Error; err != nil {
                return "", err
        }

        pdf := gofpdf.New("L", "mm", "A4", "")
        pdf.SetAutoPageBreak(true, 15)
        pdf.SetMargins(15, 15, 15)
        pdf.AddPage()

        pdf.SetFont("Arial", "B", 16)
        pdf.Cell(0, 8, "LAPORAN RIWAYAT PEMINDAHAN BARANG")
        pdf.Ln(8)

        pdf.SetFont("Arial", "", 10)
        pdf.Cell(0, 5, fmt.Sprintf("Tanggal Cetak: %s", time.Now().Format("02 Januari 2006 15:04:05")))
        pdf.Ln(5)
        dateRange := "Semua periode"
        if startDate != "" || endDate != "" {
                dateRange = startDate + " s/d " + endDate
        }
        pdf.Cell(0, 5, "Periode: "+dateRange)
        pdf.Ln(5)
        pdf.Cell(0, 5, fmt.Sprintf("Total Pemindahan: %d", len(transfers)))
        pdf.Ln(10)

        // Header
        pdf.SetFont("Arial", "B", 9)
        pdf.SetFillColor(220, 220, 220)
        pdf.CellFormat(25, 7, "Tanggal", "1", 0, "C", true, 0, "")
        pdf.CellFormat(25, 7, "Kode Barang", "1", 0, "C", true, 0, "")
        pdf.CellFormat(50, 7, "Nama Barang", "1", 0, "L", true, 0, "")
        pdf.CellFormat(40, 7, "Dari Ruangan", "1", 0, "L", true, 0, "")
        pdf.CellFormat(40, 7, "Ke Ruangan", "1", 0, "L", true, 0, "")
        pdf.CellFormat(30, 7, "Oleh", "1", 0, "L", true, 0, "")
        pdf.CellFormat(50, 7, "Alasan", "1", 0, "L", true, 0, "")
        pdf.Ln(-1)

        pdf.SetFont("Arial", "", 8)
        for _, t := range transfers {
                fromRoom := "-"
                if t.FromRoom != nil {
                        fromRoom = t.FromRoom.Name
                }
                toRoom := "-"
                if t.ToRoom != nil {
                        toRoom = t.ToRoom.Name
                }
                itemName := "-"
                itemCode := "-"
                if t.Item != nil {
                        itemName = truncate(t.Item.Name, 25)
                        itemCode = t.Item.Code
                }
                userName := "-"
                if t.User != nil {
                        if t.User.FullName != "" {
                                userName = t.User.FullName
                        } else {
                                userName = t.User.Username
                        }
                }

                pdf.CellFormat(25, 6, t.TransferDate.Format("02/01/2006"), "1", 0, "C", false, 0, "")
                pdf.CellFormat(25, 6, itemCode, "1", 0, "C", false, 0, "")
                pdf.CellFormat(50, 6, itemName, "1", 0, "L", false, 0, "")
                pdf.CellFormat(40, 6, truncate(fromRoom, 20), "1", 0, "L", false, 0, "")
                pdf.CellFormat(40, 6, truncate(toRoom, 20), "1", 0, "L", false, 0, "")
                pdf.CellFormat(30, 6, truncate(userName, 15), "1", 0, "L", false, 0, "")
                pdf.CellFormat(50, 6, truncate(t.Reason, 25), "1", 0, "L", false, 0, "")
                pdf.Ln(-1)
        }

        outputFile := filepath.Join(s.DownloadDir, fmt.Sprintf("laporan_pemindahan_%s.pdf", time.Now().Format("20060102_150405")))
        if err := pdf.OutputFileAndClose(outputFile); err != nil {
                return "", err
        }
        return outputFile, nil
}

// formatRupiah format angka ke format Rupiah (1.234.567)
func formatRupiah(amount float64) string {
        if amount == 0 {
                return "0"
        }
        isNegative := amount < 0
        if isNegative {
                amount = -amount
        }
        // Convert ke int untuk memformat
        intAmount := int64(amount)
        result := ""
        counter := 0
        for intAmount > 0 {
                if counter == 3 {
                        result = "." + result
                        counter = 0
                }
                digit := intAmount % 10
                result = string(rune('0'+digit)) + result
                intAmount /= 10
                counter++
        }
        if result == "" {
                result = "0"
        }
        if isNegative {
                result = "-" + result
        }
        return result
}
