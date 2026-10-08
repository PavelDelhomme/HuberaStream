package config

import (
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

type Config struct {
	// Database
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	// Redis
	RedisHost string
	RedisPort string

	// JWT
	JWTSecret string

	// Storage
	VideoStoragePath     string
	ThumbnailStoragePath string

	// Admin
	AdminEmail    string
	AdminPassword string

	// Server
	Port string
	Env  string // development, production

	// Metadata
	TMDBAPIKey string
}

func Load() *Config {
	// Charger le fichier .env s'il existe
	envPath := filepath.Join(".", ".env")
	if _, err := os.Stat(envPath); err == nil {
		godotenv.Load(envPath)
	}

	return &Config{
		// Database
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "streammake"),
		DBPassword: getEnv("DB_PASSWORD", "streammake_password"),
		DBName:     getEnv("DB_NAME", "streammake_db"),

		// Redis
		RedisHost: getEnv("REDIS_HOST", "localhost"),
		RedisPort: getEnv("REDIS_PORT", "6379"),

		// JWT
		JWTSecret: getEnv("JWT_SECRET", "your-secret-key-change-in-production"),

		// Storage
		VideoStoragePath:     getEnv("VIDEO_STORAGE_PATH", "./storage/videos"),
		ThumbnailStoragePath: getEnv("THUMBNAIL_STORAGE_PATH", "./storage/thumbnails"),

		// Admin
		AdminEmail:    getEnv("ADMIN_EMAIL", "dev@delhomme.ovh"),
		AdminPassword: getEnv("ADMIN_PASSWORD", "AdminStreamMake2024!"),

		// Server
		Port: getEnv("PORT", "8080"),
		Env:  getEnv("ENV", "development"),

		// Metadata
		TMDBAPIKey: getEnv("TMDB_API_KEY", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

