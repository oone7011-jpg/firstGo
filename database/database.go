package database

import (
	"log"
	"os"
	"path/filepath"

	"kir-sekolah/config"
	"kir-sekolah/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Init menginisialisasi database SQLite dan melakukan migrasi schema
func Init(cfg *config.Config) error {
	// Pastikan direktori untuk database ada
	dbDir := filepath.Dir(cfg.DBPath)
	if dbDir != "." && dbDir != "" {
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			return err
		}
	}

	// Pastikan direktori download ada
	if err := os.MkdirAll(cfg.DownloadDir, 0755); err != nil {
		return err
	}

	var err error
	DB, err = gorm.Open(sqlite.Open(cfg.DBPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return err
	}

	// Enable foreign keys
	if err := DB.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		return err
	}

	// Auto migrate semua model
	if err := DB.AutoMigrate(
		&models.User{},
		&models.Room{},
		&models.Item{},
		&models.TransferHistory{},
		&models.Notification{},
		&models.ItemConditionHistory{},
	); err != nil {
		return err
	}

	// Seed default users
	if err := seedDefaultUsers(cfg); err != nil {
		return err
	}

	// Seed sample data jika database kosong
	if err := seedSampleData(); err != nil {
		log.Printf("Warning: seed sample data: %v", err)
	}

	log.Println("✓ Database initialized successfully")
	return nil
}

// seedDefaultUsers membuat user admin dan operator default jika belum ada
func seedDefaultUsers(cfg *config.Config) error {
	var userCount int64
	DB.Model(&models.User{}).Count(&userCount)
	if userCount > 0 {
		return nil
	}

	users := []models.User{
		{
			Username: cfg.AdminUsername,
			Role:     models.RoleAdmin,
			FullName: "Administrator Sistem",
		},
		{
			Username: cfg.OperatorUsername,
			Role:     models.RoleOperator,
			FullName: "Operator Inventaris",
		},
	}

	for i := range users {
		hashed, err := bcrypt.GenerateFromPassword([]byte(cfg.AdminPassword), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		if users[i].Username == cfg.OperatorUsername {
			hashed, err = bcrypt.GenerateFromPassword([]byte(cfg.OperatorPassword), bcrypt.DefaultCost)
			if err != nil {
				return err
			}
		}
		users[i].Password = string(hashed)
		if err := DB.Create(&users[i]).Error; err != nil {
			return err
		}
	}

	log.Println("✓ Default users created (admin/operator)")
	return nil
}

// seedSampleData membuat sample ruangan dan barang jika database kosong
func seedSampleData() error {
	var roomCount int64
	DB.Model(&models.Room{}).Count(&roomCount)
	if roomCount > 0 {
		return nil
	}

	// Sample rooms
	rooms := []models.Room{
		{Code: "R001", Name: "Ruang Kelas 7A", Description: "Kelas tingkat 7A lantai 1"},
		{Code: "R002", Name: "Ruang Kelas 8B", Description: "Kelas tingkat 8B lantai 2"},
		{Code: "R003", Name: "Laboratorium IPA", Description: "Lab IPA lengkap"},
		{Code: "R004", Name: "Perpustakaan", Description: "Ruang perpustakaan utama"},
		{Code: "R005", Name: "Ruang Guru", Description: "Ruang kerja guru"},
		{Code: "R006", Name: "Ruang Kepala Sekolah", Description: "Kantor kepala sekolah"},
	}
	for i := range rooms {
		if err := DB.Create(&rooms[i]).Error; err != nil {
			return err
		}
	}

	// Sample items
	items := []models.Item{
		{Code: "INV-001", Name: "Meja Belajar Siswa", RoomID: 1, Condition: models.KondisiBaik, Price: 350000, Description: "Meja kayu jati"},
		{Code: "INV-002", Name: "Kursi Siswa", RoomID: 1, Condition: models.KondisiBaik, Price: 150000, Description: "Kursi besi"},
		{Code: "INV-003", Name: "Papan Tulis Whiteboard", RoomID: 1, Condition: models.KondisiBaik, Price: 500000},
		{Code: "INV-004", Name: "Proyektor Epson", RoomID: 3, Condition: models.KondisiBaik, Price: 3500000, Description: "Proyektor untuk presentasi"},
		{Code: "INV-005", Name: "Mikroskop Olympus", RoomID: 3, Condition: models.KondisiRusakRingan, Price: 2500000, Description: "Lensa sedikit bermasalah"},
		{Code: "INV-006", Name: "Tabung Reaksi (Set 10)", RoomID: 3, Condition: models.KondisiBaik, Price: 200000},
		{Code: "INV-007", Name: "Rak Buku Besar", RoomID: 4, Condition: models.KondisiBaik, Price: 1200000},
		{Code: "INV-008", Name: "Komputer PC Guru", RoomID: 5, Condition: models.KondisiBaik, Price: 5000000, Description: "Intel i5, RAM 8GB"},
		{Code: "INV-009", Name: "Printer Epson L3110", RoomID: 5, Condition: models.KondisiRusakBerat, Price: 1800000, Description: "Tidak bisa menyala"},
		{Code: "INV-010", Name: "AC Split 1 PK", RoomID: 6, Condition: models.KondisiBaik, Price: 3000000},
	}
	for i := range items {
		if err := DB.Create(&items[i]).Error; err != nil {
			return err
		}
	}

	log.Println("✓ Sample data created (6 rooms, 10 items)")
	return nil
}
