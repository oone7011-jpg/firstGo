package main

import (
	"fmt"
	"log"

	"kir-sekolah/config"
	"kir-sekolah/database"
	"kir-sekolah/handlers"
	"kir-sekolah/middleware"
	"kir-sekolah/services"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// Load config
	cfg := config.Load()

	// Init database
	if err := database.Init(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Init services
	pdfService := services.NewPDFService(database.DB, cfg.DownloadDir)
	excelService := services.NewExcelService(database.DB, cfg.DownloadDir)
	qrService := services.NewQRService(database.DB, cfg.DownloadDir)

	// Init handlers
	authHandler := handlers.NewAuthHandler(database.DB, cfg)
	roomHandler := handlers.NewRoomHandler(database.DB)
	itemHandler := handlers.NewItemHandler(database.DB)
	transferHandler := handlers.NewTransferHandler(database.DB)
	scanHandler := handlers.NewScanHandler(database.DB)
	notificationHandler := handlers.NewNotificationHandler(database.DB)
	reportHandler := handlers.NewReportHandler(pdfService, excelService, qrService)

	// Setup gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// CORS for mobile app access
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
	}))

	// === Health check ===
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":     "ok",
			"service":    "KIR Sekolah API",
			"version":    "1.0.0",
			"timestamp":  fmt.Sprintf("%v", c.Request.URL.Path),
		})
	})

	// === Public routes (no auth) ===
	api := r.Group("/api")
	{
		api.POST("/auth/login", authHandler.Login)
	}

	// === Protected routes ===
	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware(cfg))
	{
		// Auth
		protected.GET("/auth/me", authHandler.Me)

		// Users (admin only)
		users := protected.Group("/users")
		users.Use(middleware.AdminOnly())
		{
			users.GET("", authHandler.ListUsers)
			users.POST("", authHandler.CreateUser)
			users.DELETE("/:id", authHandler.DeleteUser)
		}

		// Rooms
		rooms := protected.Group("/rooms")
		{
			rooms.GET("", roomHandler.List)
			rooms.GET("/:id", roomHandler.Get)
			rooms.POST("", roomHandler.Create)
			rooms.PUT("/:id", roomHandler.Update)
			rooms.DELETE("/:id", roomHandler.Delete)
			rooms.GET("/:id/items", roomHandler.GetItems)
		}

		// Items
		items := protected.Group("/items")
		{
			items.GET("", itemHandler.List)
			items.GET("/:id", itemHandler.Get)
			items.GET("/code/:code", itemHandler.GetByCode)
			items.POST("", itemHandler.Create)
			items.PUT("/:id", itemHandler.Update)
			items.DELETE("/:id", itemHandler.Delete)
			items.GET("/:id/condition-history", itemHandler.GetConditionHistory)
			items.GET("/:id/transfer-history", itemHandler.GetTransferHistory)
		}

		// Transfers (riwayat pemindahan)
		transfers := protected.Group("/transfers")
		{
			transfers.GET("", transferHandler.List)
			transfers.GET("/:id", transferHandler.Get)
			transfers.POST("", transferHandler.Transfer)
		}

		// Scan (mobile endpoints)
		scan := protected.Group("/scan")
		{
			scan.POST("/verify", scanHandler.VerifyScan)
			scan.POST("/update-condition", scanHandler.UpdateCondition)
			scan.GET("/history", scanHandler.ScanHistory)
		}

		// Notifications
		notifications := protected.Group("/notifications")
		{
			notifications.GET("", notificationHandler.List)
			notifications.GET("/unread-count", notificationHandler.GetUnreadCount)
			notifications.PUT("/:id/read", notificationHandler.MarkAsRead)
			notifications.PUT("/read-all", notificationHandler.MarkAllAsRead)
			notifications.DELETE("/:id", notificationHandler.Delete)
		}

		// Reports
		reports := protected.Group("/reports")
		{
			reports.GET("/kir/pdf", reportHandler.GenerateKIRReportPDF)
			reports.GET("/finance/excel", reportHandler.GenerateFinanceReportExcel)
			reports.GET("/transfers/pdf", reportHandler.GenerateTransferReportPDF)
			reports.GET("/download/:filename", reportHandler.DownloadFile)
		}

		// QR Code
		qr := protected.Group("/qrcode")
		{
			qr.GET("/:code/png", reportHandler.GenerateQRCodePNG)
			qr.GET("/items/:id/label-pdf", reportHandler.GenerateSingleLabelPDF)
			qr.POST("/bulk-labels", reportHandler.GenerateBulkLabelsPDF)
		}
	}

	// Start server
	addr := ":" + cfg.Port
	log.Printf("🚀 Server KIR Sekolah berjalan di http://localhost:%s", cfg.Port)
	log.Printf("📖 API Documentation: http://localhost:%s/health", cfg.Port)
	log.Printf("🔐 Default login: admin/admin123 atau operator/operator123")
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
