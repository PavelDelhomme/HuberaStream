package config

import (
	"os"
	"path/filepath"
	"strings"

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
	TorrentStoragePath   string

	// Admin
	AdminEmail    string
	AdminPassword string

	// Torrent
	TorrentClientPath             string
	TorrentClientType             string
	TorrentDownloadPath           string
	EnableAutoDownload            bool
	TorrentClientTransmissionURL  string
	TorrentClientTransmissionUser string
	TorrentClientTransmissionPass string
	TorrentClientQBittorrentURL   string
	TorrentClientQBittorrentUser  string
	TorrentClientQBittorrentPass  string
	TorrentClientDelugeURL        string
	TorrentClientDelugePass       string
	TorrentClientRTorrentURL      string

	// Server
	Port string
	Env  string // development, production

	// Metadata
	TMDBAPIKey string

	// Intégration *arr (Prowlarr / Radarr / Sonarr) — URLs internes Docker ou vides
	ProwlarrURL string
	RadarrURL   string
	SonarrURL   string

	// Webhooks Radarr/Sonarr → StreamMake
	WebhookSecret         string
	PublicWebhookBaseURL  string // ex. https://stream.delhomme.ovh (sans /api) pour affichage admin
	EnableWebhookImport   bool   // tente d’importer le fichier vidéo si le chemin est visible par le backend
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
		TorrentClientPath:             getTorrentClientPath(),
		TorrentClientType:             getEnv("TORRENT_CLIENT_TYPE", "transmission"), // transmission, qbittorrent, deluge, rtorrent
		TorrentDownloadPath:           getTorrentDownloadPath(),
		EnableAutoDownload:            getEnv("ENABLE_AUTO_DOWNLOAD", "false") == "true",
		TorrentClientTransmissionURL:  getEnv("TRANSMISSION_URL", "http://transmission:9091/transmission/rpc"),
		TorrentClientTransmissionUser: getEnv("TRANSMISSION_USER", "transmission"),
		TorrentClientTransmissionPass: getEnv("TRANSMISSION_PASS", "transmission"),
		TorrentClientQBittorrentURL:   getEnv("QBITTORRENT_URL", "http://qbittorrent:8080"),
		TorrentClientQBittorrentUser:  getEnv("QBITTORRENT_USER", "admin"),
		TorrentClientQBittorrentPass:  getEnv("QBITTORRENT_PASS", "adminadmin"),
		TorrentClientDelugeURL:        getEnv("DELUGE_URL", "http://deluge:8112/json"),
		TorrentClientDelugePass:       getEnv("DELUGE_PASS", "deluge"),
		TorrentClientRTorrentURL:      getEnv("RTORRENT_URL", "http://rtorrent:5000/RPC2"),

		// Server
		Port: getEnv("PORT", "8080"),
		Env:  getEnv("ENV", "development"),

		// Metadata
		TMDBAPIKey: getEnv("TMDB_API_KEY", ""),

		// *arr (optionnel ; activer les services avec docker compose --profile arr)
		ProwlarrURL: strings.TrimSpace(getEnv("PROWLARR_URL", "")),
		RadarrURL:   strings.TrimSpace(getEnv("RADARR_URL", "")),
		SonarrURL:   strings.TrimSpace(getEnv("SONARR_URL", "")),

		WebhookSecret:        strings.TrimSpace(getEnv("WEBHOOK_SECRET", "")),
		PublicWebhookBaseURL: strings.TrimSpace(strings.TrimSuffix(getEnv("PUBLIC_WEBHOOK_BASE_URL", ""), "/")),
		EnableWebhookImport:  getEnv("WEBHOOK_IMPORT_ENABLED", "false") == "true",
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getAbsolutePath convertit un chemin relatif en chemin absolu
func getAbsolutePath(path string) string {
	// Si le chemin est déjà absolu, le retourner tel quel
	if filepath.IsAbs(path) {
		return path
	}
	// Sinon, le convertir en chemin absolu
	absPath, err := filepath.Abs(path)
	if err != nil {
		// En cas d'erreur, retourner le chemin original
		return path
	}
	return absPath
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

// getTorrentDownloadPath retourne le chemin de téléchargement pour les torrents
// Ce chemin doit être tel que le client torrent le voit dans son conteneur
func getTorrentDownloadPath() string {
	path := getEnv("TORRENT_DOWNLOAD_PATH", "/downloads")
	// Si le chemin commence par /downloads, le garder tel quel (c'est le chemin dans le conteneur Transmission)
	if strings.HasPrefix(path, "/downloads") {
		return path
	}
	// Sinon, utiliser getAbsolutePath pour les autres cas
	return getAbsolutePath(path)
}
