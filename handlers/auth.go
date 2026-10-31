package handlers

import (
	"net/http"

	"kir-sekolah/config"
	"kir-sekolah/middleware"
	"kir-sekolah/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewAuthHandler(db *gorm.DB, cfg *config.Config) *AuthHandler {
	return &AuthHandler{DB: db, Cfg: cfg}
}

// Login - POST /api/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	var user models.User
	if err := h.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, models.APIResponse{
			Success: false,
			Message: "Username atau password salah",
		})
		return
	}

	// Validasi password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, models.APIResponse{
			Success: false,
			Message: "Username atau password salah",
		})
		return
	}

	// Generate JWT token
	token, err := middleware.GenerateToken(h.Cfg, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Gagal membuat token: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Login berhasil",
		Data: models.LoginResponse{
			Token: token,
			User:  user,
		},
	})
}

// Me - GET /api/auth/me - mengambil info user yang sedang login
func (h *AuthHandler) Me(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var user models.User
	if err := h.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, models.APIResponse{
			Success: false,
			Message: "User tidak ditemukan",
		})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    user,
	})
}

// CreateUser - POST /api/users - hanya admin
func (h *AuthHandler) CreateUser(c *gin.Context) {
	var req models.UserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	// Validasi role
	if req.Role != models.RoleAdmin && req.Role != models.RoleOperator {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Role harus 'admin' atau 'operator'",
		})
		return
	}

	// Hash password
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Gagal hash password",
		})
		return
	}

	user := models.User{
		Username: req.Username,
		Password: string(hashed),
		Role:     req.Role,
		FullName: req.FullName,
	}

	if err := h.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusConflict, models.APIResponse{
			Success: false,
			Message: "Username sudah digunakan atau error: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, models.APIResponse{
		Success: true,
		Message: "User berhasil dibuat",
		Data:    user,
	})
}

// ListUsers - GET /api/users - hanya admin
func (h *AuthHandler) ListUsers(c *gin.Context) {
	var users []models.User
	h.DB.Order("created_at DESC").Find(&users)
	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    users,
	})
}

// DeleteUser - DELETE /api/users/:id - hanya admin
func (h *AuthHandler) DeleteUser(c *gin.Context) {
	id := c.Param("id")

	// Cegah admin menghapus diri sendiri
	currentUserID := middleware.GetUserID(c)
	if currentUserID > 0 && c.Param("id") == currentUserIDToString(currentUserID) {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Tidak dapat menghapus user yang sedang login",
		})
		return
	}

	if err := h.DB.Delete(&models.User{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Gagal menghapus user",
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "User berhasil dihapus",
	})
}

func currentUserIDToString(id uint) string {
	return intToString(int(id))
}

func intToString(i int) string {
	if i == 0 {
		return "0"
	}
	result := ""
	for i > 0 {
		result = string(rune('0'+i%10)) + result
		i /= 10
	}
	return result
}
