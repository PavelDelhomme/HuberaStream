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
	VideoStoragePath    string
	ThumbnailStoragePath string
	TorrentStoragePath   string

	// Admin
	AdminEmail    string
	AdminPassword string

	// Torrent
	TorrentClientPath   string
	TorrentClientType   string
	TorrentDownloadPath string
	EnableAutoDownload  bool

	// Server
	Port string
	Env  string // development, production
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
		TorrentStoragePath:   getEnv("TORRENT_STORAGE_PATH", "./storage/torrents"),

		// Admin
		AdminEmail:    getEnv("ADMIN_EMAIL", "dev@delhomme.ovh"),
		AdminPassword: getEnv("ADMIN_PASSWORD", "AdminStreamMake2024!"),

		// Torrent
		TorrentClientPath:   getEnv("TORRENT_CLIENT_PATH", "http://localhost:9091/transmission/rpc"),
		TorrentClientType:   getEnv("TORRENT_CLIENT_TYPE", "transmission"), // transmission, qbittorrent, deluge, rtorrent
		TorrentDownloadPath: getEnv("TORRENT_DOWNLOAD_PATH", "./storage/torrents/downloads"),
		EnableAutoDownload:  getEnv("ENABLE_AUTO_DOWNLOAD", "false") == "true",

		// Server
		Port: getEnv("PORT", "8080"),
		Env:  getEnv("ENV", "development"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

