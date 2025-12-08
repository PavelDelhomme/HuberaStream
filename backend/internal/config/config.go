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
	TorrentClientPath           string
	TorrentClientType           string
	TorrentDownloadPath         string
	EnableAutoDownload          bool
	TorrentClientTransmissionURL string
	TorrentClientTransmissionUser string
	TorrentClientTransmissionPass string
	TorrentClientQBittorrentURL string
	TorrentClientQBittorrentUser string
	TorrentClientQBittorrentPass string
	TorrentClientDelugeURL      string
	TorrentClientDelugePass     string
	TorrentClientRTorrentURL    string

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
		// Utiliser les URLs spécifiques si définies, sinon TORRENT_CLIENT_PATH, sinon défaut
		TorrentClientPath:           getTorrentClientPath(),
		TorrentClientType:           getEnv("TORRENT_CLIENT_TYPE", "transmission"), // transmission, qbittorrent, deluge, rtorrent
		TorrentDownloadPath:         getEnv("TORRENT_DOWNLOAD_PATH", "./storage/torrents/downloads"),
		EnableAutoDownload:          getEnv("ENABLE_AUTO_DOWNLOAD", "false") == "true",
		TorrentClientTransmissionURL: getEnv("TRANSMISSION_URL", "http://transmission:9091/transmission/rpc"),
		TorrentClientTransmissionUser: getEnv("TRANSMISSION_USER", "transmission"),
		TorrentClientTransmissionPass: getEnv("TRANSMISSION_PASS", "transmission"),
		TorrentClientQBittorrentURL: getEnv("QBITTORRENT_URL", "http://qbittorrent:8080"),
		TorrentClientQBittorrentUser: getEnv("QBITTORRENT_USER", "admin"),
		TorrentClientQBittorrentPass: getEnv("QBITTORRENT_PASS", "adminadmin"),
		TorrentClientDelugeURL:      getEnv("DELUGE_URL", "http://deluge:8112/json"),
		TorrentClientDelugePass:     getEnv("DELUGE_PASS", "deluge"),
		TorrentClientRTorrentURL:    getEnv("RTORRENT_URL", "http://rtorrent:5000/RPC2"),

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

// getTorrentClientPath retourne l'URL du client torrent en fonction du type et des variables d'environnement
func getTorrentClientPath() string {
	// Vérifier d'abord TORRENT_CLIENT_PATH (priorité)
	if path := os.Getenv("TORRENT_CLIENT_PATH"); path != "" {
		return path
	}

	// Sinon, utiliser les URLs spécifiques selon le type
	clientType := getEnv("TORRENT_CLIENT_TYPE", "transmission")
	switch clientType {
	case "transmission":
		if url := os.Getenv("TRANSMISSION_URL"); url != "" {
			return url
		}
		return "http://transmission:9091/transmission/rpc"
	case "qbittorrent":
		if url := os.Getenv("QBITTORRENT_URL"); url != "" {
			return url
		}
		return "http://qbittorrent:8080"
	case "deluge":
		if url := os.Getenv("DELUGE_URL"); url != "" {
			return url
		}
		return "http://deluge:8112/json"
	case "rtorrent":
		return "http://rtorrent:5000"
	default:
		return "http://transmission:9091/transmission/rpc"
	}
}

