package services

import (
        "fmt"
        "path/filepath"
        "time"

        "kir-sekolah/models"

        "github.com/xuri/excelize/v2"
        "gorm.io/gorm"
)

type ExcelService struct {
        DB          *gorm.DB
        DownloadDir string
}

func NewExcelService(db *gorm.DB, downloadDir string) *ExcelService {
        return &ExcelService{DB: db, DownloadDir: downloadDir}
}

// GenerateMonthlyFinanceReport membuat laporan Excel keuangan bulanan
// year: tahun, month: bulan (1-12)
func (s *ExcelService) GenerateMonthlyFinanceReport(year int, month int) (string, error) {
        if year == 0 {
                year = time.Now().Year()
        }
        if month == 0 {
                month = int(time.Now().Month())
        }

        // Get all items dengan harga
        var items []models.Item
        if err := s.DB.Preload("Room").Order("room_id ASC, code ASC").Find(&items).Error; err != nil {
                return "", err
        }

        f := excelize.NewFile()
        defer f.Close()

        // ====== SHEET 1: Ringkasan ======
        sheetSummary := "Ringkasan"
        f.SetSheetName(f.GetSheetName(0), sheetSummary)

        // Title
        f.SetCellValue(sheetSummary, "A1", fmt.Sprintf("LAPORAN KEUANGAN INVENTARIS SEKOLAH"))
        f.SetCellValue(sheetSummary, "A2", fmt.Sprintf("Periode: %s %d", time.Month(month).String(), year))
        f.SetCellValue(sheetSummary, "A3", fmt.Sprintf("Tanggal Cetak: %s", time.Now().Format("02/01/2006 15:04:05")))

        // Style title
        styleTitle, _ := f.NewStyle(&excelize.Style{
                Font:      &excelize.Font{Bold: true, Size: 16, Color: "#1F4E79"},
                Alignment: &excelize.Alignment{Horizontal: "center"},
        })
        f.SetCellStyle(sheetSummary, "A1", "F1", styleTitle)
        f.MergeCell(sheetSummary, "A1", "F1")

        styleSubtitle, _ := f.NewStyle(&excelize.Style{
                Font:      &excelize.Font{Italic: true, Size: 11, Color: "#555555"},
                Alignment: &excelize.Alignment{Horizontal: "center"},
        })
        f.SetCellStyle(sheetSummary, "A2", "F2", styleSubtitle)
        f.MergeCell(sheetSummary, "A2", "F2")
        f.SetCellStyle(sheetSummary, "A3", "F3", styleSubtitle)
        f.MergeCell(sheetSummary, "A3", "F3")

        // Summary table
        f.SetCellValue(sheetSummary, "A5", "RINGKASAN PER KONDISI")
        styleHeaderSummary, _ := f.NewStyle(&excelize.Style{
                Font: &excelize.Font{Bold: true, Size: 12, Color: "#FFFFFF"},
                Fill: excelize.Fill{Type: "pattern", Color: []string{"#1F4E79"}, Pattern: 1},
                Alignment: &excelize.Alignment{Horizontal: "center"},
                Border: []excelize.Border{
                        {Type: "left", Color: "#000000", Style: 1},
                        {Type: "top", Color: "#000000", Style: 1},
                        {Type: "right", Color: "#000000", Style: 1},
                        {Type: "bottom", Color: "#000000", Style: 1},
                },
        })
        f.SetCellStyle(sheetSummary, "A5", "D5", styleHeaderSummary)
        f.MergeCell(sheetSummary, "A5", "D5")

        // Header
        headers := []string{"Kondisi", "Jumlah Barang", "Total Nilai (Rp)", "Persentase"}
        for i, h := range headers {
                cell, _ := excelize.CoordinatesToCellName(i+1, 6)
                f.SetCellValue(sheetSummary, cell, h)
        }
        styleHeaderTable, _ := f.NewStyle(&excelize.Style{
                Font: &excelize.Font{Bold: true},
                Fill: excelize.Fill{Type: "pattern", Color: []string{"#B4C6E7"}, Pattern: 1},
                Alignment: &excelize.Alignment{Horizontal: "center"},
                Border: []excelize.Border{
                        {Type: "left", Color: "#000000", Style: 1},
                        {Type: "top", Color: "#000000", Style: 1},
                        {Type: "right", Color: "#000000", Style: 1},
                        {Type: "bottom", Color: "#000000", Style: 1},
                },
        })
        f.SetCellStyle(sheetSummary, "A6", "D6", styleHeaderTable)

        // Calculate summary
        countBaik, countRusakRingan, countRusakBerat := 0, 0, 0
        valueBaik, valueRusakRingan, valueRusakBerat := 0.0, 0.0, 0.0
        for _, item := range items {
                switch item.Condition {
                case models.KondisiBaik:
                        countBaik++
                        valueBaik += item.Price
                case models.KondisiRusakRingan:
                        countRusakRingan++
                        valueRusakRingan += item.Price
                case models.KondisiRusakBerat:
                        countRusakBerat++
                        valueRusakBerat += item.Price
                }
        }
        totalItems := countBaik + countRusakRingan + countRusakBerat
        totalValue := valueBaik + valueRusakRingan + valueRusakBerat

        type conditionRow struct {
                name  string
                count int
                value float64
        }
        rows := []conditionRow{
                {"Baik", countBaik, valueBaik},
                {"Rusak Ringan", countRusakRingan, valueRusakRingan},
                {"Rusak Berat", countRusakBerat, valueRusakBerat},
        }

        styleCell, _ := f.NewStyle(&excelize.Style{
                Alignment: &excelize.Alignment{Horizontal: "center"},
                Border: []excelize.Border{
                        {Type: "left", Color: "#000000", Style: 1},
                        {Type: "top", Color: "#000000", Style: 1},
                        {Type: "right", Color: "#000000", Style: 1},
                        {Type: "bottom", Color: "#000000", Style: 1},
                },
        })
        styleCurrency, _ := f.NewStyle(&excelize.Style{
                NumFmt:     2,
                Alignment:  &excelize.Alignment{Horizontal: "right"},
                Border: []excelize.Border{
                        {Type: "left", Color: "#000000", Style: 1},
                        {Type: "top", Color: "#000000", Style: 1},
                        {Type: "right", Color: "#000000", Style: 1},
                        {Type: "bottom", Color: "#000000", Style: 1},
                },
        })

        for i, r := range rows {
                row := 7 + i
                f.SetCellValue(sheetSummary, fmt.Sprintf("A%d", row), r.name)
                f.SetCellValue(sheetSummary, fmt.Sprintf("B%d", row), r.count)
                f.SetCellValue(sheetSummary, fmt.Sprintf("C%d", row), r.value)
                f.SetCellValue(sheetSummary, fmt.Sprintf("D%d", row), fmt.Sprintf("%.1f%%", percent(r.count, totalItems)))
                f.SetCellStyle(sheetSummary, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), styleCell)
                f.SetCellStyle(sheetSummary, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), styleCell)
                f.SetCellStyle(sheetSummary, fmt.Sprintf("C%d", row), fmt.Sprintf("C%d", row), styleCurrency)
                f.SetCellStyle(sheetSummary, fmt.Sprintf("D%d", row), fmt.Sprintf("D%d", row), styleCell)
        }

        // Total row
        totalRow := 10
        f.SetCellValue(sheetSummary, fmt.Sprintf("A%d", totalRow), "TOTAL")
        f.SetCellValue(sheetSummary, fmt.Sprintf("B%d", totalRow), totalItems)
        f.SetCellValue(sheetSummary, fmt.Sprintf("C%d", totalRow), totalValue)
        f.SetCellValue(sheetSummary, fmt.Sprintf("D%d", totalRow), "100%")
        styleTotal, _ := f.NewStyle(&excelize.Style{
                Font: &excelize.Font{Bold: true},
                Fill: excelize.Fill{Type: "pattern", Color: []string{"#FFE699"}, Pattern: 1},
                Alignment: &excelize.Alignment{Horizontal: "center"},
                Border: []excelize.Border{
                        {Type: "left", Color: "#000000", Style: 1},
                        {Type: "top", Color: "#000000", Style: 1},
                        {Type: "right", Color: "#000000", Style: 1},
                        {Type: "bottom", Color: "#000000", Style: 1},
                },
        })
        f.SetCellStyle(sheetSummary, fmt.Sprintf("A%d", totalRow), fmt.Sprintf("A%d", totalRow), styleTotal)
        f.SetCellStyle(sheetSummary, fmt.Sprintf("B%d", totalRow), fmt.Sprintf("B%d", totalRow), styleTotal)
        styleTotalCur, _ := f.NewStyle(&excelize.Style{
                Font:      &excelize.Font{Bold: true},
                Fill:      excelize.Fill{Type: "pattern", Color: []string{"#FFE699"}, Pattern: 1},
                NumFmt:    2,
                Alignment: &excelize.Alignment{Horizontal: "right"},
                Border: []excelize.Border{
                        {Type: "left", Color: "#000000", Style: 1},
                        {Type: "top", Color: "#000000", Style: 1},
                        {Type: "right", Color: "#000000", Style: 1},
                        {Type: "bottom", Color: "#000000", Style: 1},
                },
        })
        f.SetCellStyle(sheetSummary, fmt.Sprintf("C%d", totalRow), fmt.Sprintf("C%d", totalRow), styleTotalCur)
        f.SetCellStyle(sheetSummary, fmt.Sprintf("D%d", totalRow), fmt.Sprintf("D%d", totalRow), styleTotal)

        // Set column widths
        f.SetColWidth(sheetSummary, "A", "A", 25)
        f.SetColWidth(sheetSummary, "B", "B", 18)
        f.SetColWidth(sheetSummary, "C", "C", 25)
        f.SetColWidth(sheetSummary, "D", "D", 15)

        // ====== SHEET 2: Detail per Ruangan ======
        sheetRoom := "Per Ruangan"
        f.NewSheet(sheetRoom)

        f.SetCellValue(sheetRoom, "A1", "RINGKASAN NILAI ASET PER RUANGAN")
        styleTitle2, _ := f.NewStyle(&excelize.Style{
                Font:      &excelize.Font{Bold: true, Size: 14, Color: "#1F4E79"},
                Alignment: &excelize.Alignment{Horizontal: "center"},
        })
        f.SetCellStyle(sheetRoom, "A1", "F1", styleTitle2)
        f.MergeCell(sheetRoom, "A1", "F1")

        // Headers
        roomHeaders := []string{"Kode Ruang", "Nama Ruangan", "Jumlah Barang", "Kondisi Baik", "Kondisi Rusak Ringan", "Total Nilai (Rp)"}
        for i, h := range roomHeaders {
                cell, _ := excelize.CoordinatesToCellName(i+1, 3)
                f.SetCellValue(sheetRoom, cell, h)
        }
        f.SetCellStyle(sheetRoom, "A3", "F3", styleHeaderTable)

        // Aggregate by room
        roomAgg := make(map[uint]*roomAggregate)
        for _, item := range items {
                if item.Room == nil {
                        continue
                }
                if _, ok := roomAgg[item.RoomID]; !ok {
                        roomAgg[item.RoomID] = &roomAggregate{
                                Code: item.Room.Code,
                                Name: item.Room.Name,
                        }
                }
                ra := roomAgg[item.RoomID]
                ra.Count++
                ra.TotalValue += item.Price
                switch item.Condition {
                case models.KondisiBaik:
                        ra.Baik++
                case models.KondisiRusakRingan:
                        ra.RusakRingan++
                case models.KondisiRusakBerat:
                        ra.RusakBerat++
                }
        }

        // Sort rooms by code
        sortedRooms := make([]*roomAggregate, 0, len(roomAgg))
        for _, ra := range roomAgg {
                sortedRooms = append(sortedRooms, ra)
        }
        // simple bubble sort by code
        for i := 0; i < len(sortedRooms); i++ {
                for j := i + 1; j < len(sortedRooms); j++ {
                        if sortedRooms[i].Code > sortedRooms[j].Code {
                                sortedRooms[i], sortedRooms[j] = sortedRooms[j], sortedRooms[i]
                        }
                }
        }

        for i, ra := range sortedRooms {
                row := 4 + i
                f.SetCellValue(sheetRoom, fmt.Sprintf("A%d", row), ra.Code)
                f.SetCellValue(sheetRoom, fmt.Sprintf("B%d", row), ra.Name)
                f.SetCellValue(sheetRoom, fmt.Sprintf("C%d", row), ra.Count)
                f.SetCellValue(sheetRoom, fmt.Sprintf("D%d", row), ra.Baik)
                f.SetCellValue(sheetRoom, fmt.Sprintf("E%d", row), ra.RusakRingan)
                f.SetCellValue(sheetRoom, fmt.Sprintf("F%d", row), ra.TotalValue)
                f.SetCellStyle(sheetRoom, fmt.Sprintf("A%d", row), fmt.Sprintf("E%d", row), styleCell)
                f.SetCellStyle(sheetRoom, fmt.Sprintf("F%d", row), fmt.Sprintf("F%d", row), styleCurrency)
        }

        // Total row for rooms
        totalRoomRow := 4 + len(sortedRooms)
        f.SetCellValue(sheetRoom, fmt.Sprintf("A%d", totalRoomRow), "TOTAL")
        f.SetCellValue(sheetRoom, fmt.Sprintf("B%d", totalRoomRow), "")
        f.SetCellValue(sheetRoom, fmt.Sprintf("C%d", totalRoomRow), totalItems)
        f.SetCellValue(sheetRoom, fmt.Sprintf("D%d", totalRoomRow), countBaik)
        f.SetCellValue(sheetRoom, fmt.Sprintf("E%d", totalRoomRow), countRusakRingan)
        f.SetCellValue(sheetRoom, fmt.Sprintf("F%d", totalRoomRow), totalValue)
        f.SetCellStyle(sheetRoom, fmt.Sprintf("A%d", totalRoomRow), fmt.Sprintf("E%d", totalRoomRow), styleTotal)
        f.SetCellStyle(sheetRoom, fmt.Sprintf("F%d", totalRoomRow), fmt.Sprintf("F%d", totalRoomRow), styleTotalCur)

        f.SetColWidth(sheetRoom, "A", "A", 15)
        f.SetColWidth(sheetRoom, "B", "B", 30)
        f.SetColWidth(sheetRoom, "C", "C", 15)
        f.SetColWidth(sheetRoom, "D", "D", 15)
        f.SetColWidth(sheetRoom, "E", "E", 18)
        f.SetColWidth(sheetRoom, "F", "F", 25)

        // ====== SHEET 3: Detail Item ======
        sheetDetail := "Detail Barang"
        f.NewSheet(sheetDetail)

        detailHeaders := []string{"Kode", "Nama Barang", "Ruangan", "Kondisi", "Tanggal Perolehan", "Harga (Rp)", "Keterangan"}
        for i, h := range detailHeaders {
                cell, _ := excelize.CoordinatesToCellName(i+1, 1)
                f.SetCellValue(sheetDetail, cell, h)
        }
        f.SetCellStyle(sheetDetail, "A1", "G1", styleHeaderTable)

        for i, item := range items {
                row := 2 + i
                roomName := ""
                roomCode := ""
                if item.Room != nil {
                        roomName = item.Room.Name
                        roomCode = item.Room.Code
                }
                acqDate := ""
                if item.AcquisitionDate != nil {
                        acqDate = item.AcquisitionDate.Format("02/01/2006")
                }
                f.SetCellValue(sheetDetail, fmt.Sprintf("A%d", row), item.Code)
                f.SetCellValue(sheetDetail, fmt.Sprintf("B%d", row), item.Name)
                f.SetCellValue(sheetDetail, fmt.Sprintf("C%d", row), roomCode+" - "+roomName)
                f.SetCellValue(sheetDetail, fmt.Sprintf("D%d", row), item.Condition)
                f.SetCellValue(sheetDetail, fmt.Sprintf("E%d", row), acqDate)
                f.SetCellValue(sheetDetail, fmt.Sprintf("F%d", row), item.Price)
                f.SetCellValue(sheetDetail, fmt.Sprintf("G%d", row), item.Description)

                // Warna kondisi
                var color string
                switch item.Condition {
                case models.KondisiBaik:
                        color = "#C6EFCE"
                case models.KondisiRusakRingan:
                        color = "#FFEB9C"
                case models.KondisiRusakBerat:
                        color = "#FFC7CE"
                }
                condStyle, _ := f.NewStyle(&excelize.Style{
                        Fill:      excelize.Fill{Type: "pattern", Color: []string{color}, Pattern: 1},
                        Alignment: &excelize.Alignment{Horizontal: "center"},
                        Border: []excelize.Border{
                                {Type: "left", Color: "#CCCCCC", Style: 1},
                                {Type: "top", Color: "#CCCCCC", Style: 1},
                                {Type: "right", Color: "#CCCCCC", Style: 1},
                                {Type: "bottom", Color: "#CCCCCC", Style: 1},
                        },
                })
                f.SetCellStyle(sheetDetail, fmt.Sprintf("D%d", row), fmt.Sprintf("D%d", row), condStyle)
                f.SetCellStyle(sheetDetail, fmt.Sprintf("F%d", row), fmt.Sprintf("F%d", row), styleCurrency)
        }

        f.SetColWidth(sheetDetail, "A", "A", 15)
        f.SetColWidth(sheetDetail, "B", "B", 35)
        f.SetColWidth(sheetDetail, "C", "C", 30)
        f.SetColWidth(sheetDetail, "D", "D", 18)
        f.SetColWidth(sheetDetail, "E", "E", 18)
        f.SetColWidth(sheetDetail, "F", "F", 18)
        f.SetColWidth(sheetDetail, "G", "G", 30)

        // Set active sheet to summary
        f.SetActiveSheet(0)

        // Save
        outputFile := filepath.Join(s.DownloadDir, fmt.Sprintf("laporan_keuangan_%04d_%02d.xlsx", year, month))
        if err := f.SaveAs(outputFile); err != nil {
                return "", err
        }

        return outputFile, nil
}

type roomAggregate struct {
        Code        string
        Name        string
        Count       int
        Baik        int
        RusakRingan int
        RusakBerat  int
        TotalValue  float64
}

func percent(part, total int) float64 {
        if total == 0 {
                return 0
        }
        return float64(part) / float64(total) * 100
}
