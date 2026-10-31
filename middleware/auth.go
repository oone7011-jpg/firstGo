package middleware

import (
	"net/http"
	"strings"

	"kir-sekolah/config"
	"kir-sekolah/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Claims - JWT claims
type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken menghasilkan JWT token untuk user
func GenerateToken(cfg *config.Config, user models.User) (string, error) {
	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: "kir-sekolah",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWTSecret))
}

// ParseToken memparse dan memvalidasi JWT token
func ParseToken(cfg *config.Config, tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}
	return claims, nil
}

// AuthMiddleware - middleware untuk validasi JWT token
func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, models.APIResponse{
				Success: false,
				Message: "Authorization header required",
			})
			c.Abort()
			return
		}

		// Format: "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, models.APIResponse{
				Success: false,
				Message: "Invalid authorization format, use: Bearer <token>",
			})
			c.Abort()
			return
		}

		claims, err := ParseToken(cfg, parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, models.APIResponse{
				Success: false,
				Message: "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// Set user info di context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Next()
	}
}

// AdminOnly - middleware untuk route yang hanya bisa diakses admin
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists || role.(string) != models.RoleAdmin {
			c.JSON(http.StatusForbidden, models.APIResponse{
				Success: false,
				Message: "Access denied: admin role required",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// GetUserID - helper untuk ambil user_id dari context
func GetUserID(c *gin.Context) uint {
	if v, exists := c.Get("user_id"); exists {
		return v.(uint)
	}
	return 0
}

// GetUsername - helper untuk ambil username dari context
func GetUsername(c *gin.Context) string {
	if v, exists := c.Get("username"); exists {
		return v.(string)
	}
	return ""
}

// GetRole - helper untuk ambil role dari context
func GetRole(c *gin.Context) string {
	if v, exists := c.Get("role"); exists {
		return v.(string)
	}
	return ""
}
