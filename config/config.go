package config

import (
	"os"
	"strconv"
)

// Config menyimpan konfigurasi aplikasi
type Config struct {
	Port            string
	JWTSecret       string
	DBPath          string
	DownloadDir     string
	AdminUsername   string
	AdminPassword   string
	OperatorUsername string
	OperatorPassword string
}

// Load memuat konfigurasi dari environment variable dengan default value
func Load() *Config {
	return &Config{
		Port:             getEnv("PORT", "8080"),
		JWTSecret:        getEnv("JWT_SECRET", "kir-sekolah-secret-key-change-in-production-2024"),
		DBPath:           getEnv("DB_PATH", "./kir.db"),
		DownloadDir:      getEnv("DOWNLOAD_DIR", "./downloads"),
		AdminUsername:    getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword:    getEnv("ADMIN_PASSWORD", "admin123"),
		OperatorUsername: getEnv("OPERATOR_USERNAME", "operator"),
		OperatorPassword: getEnv("OPERATOR_PASSWORD", "operator123"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
