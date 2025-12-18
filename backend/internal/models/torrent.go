package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TorrentStatus string

const (
	TorrentStatusPending     TorrentStatus = "pending"
	TorrentStatusDownloading TorrentStatus = "downloading"
	TorrentStatusSeeding     TorrentStatus = "seeding"
	TorrentStatusCompleted   TorrentStatus = "completed"
	TorrentStatusError       TorrentStatus = "error"
	TorrentStatusPaused      TorrentStatus = "paused"
)

type TorrentSource string

const (
	TorrentSource1337x      TorrentSource = "1337x"
	TorrentSourceRARBG       TorrentSource = "rarbg"
	TorrentSourceYTS         TorrentSource = "yts"
	TorrentSourceNyaa        TorrentSource = "nyaa"
	TorrentSourcePirateBay   TorrentSource = "piratebay"
	TorrentSourceTorrentDownload TorrentSource = "torrentdownload"
	TorrentSourceCustom      TorrentSource = "custom"
)

type Torrent struct {
	ID            uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name          string         `gorm:"not null" json:"name"`
	MagnetURL     string         `gorm:"type:text" json:"magnet_url"`
	TorrentURL    string         `gorm:"type:text" json:"torrent_url"`
	Source        TorrentSource  `gorm:"type:varchar(50)" json:"source"`
	Status        TorrentStatus  `gorm:"default:'pending'" json:"status"`
	Progress      float64        `gorm:"default:0" json:"progress"` // 0-100
	Size          int64          `json:"size"`                      // Taille en bytes
	Downloaded    int64          `gorm:"default:0" json:"downloaded"`
	Uploaded      int64          `gorm:"default:0" json:"uploaded"`
	DownloadSpeed int64          `json:"download_speed"` // bytes/s
	UploadSpeed   int64          `json:"upload_speed"`   // bytes/s
	Seeders       int            `json:"seeders"`
	Leechers      int            `json:"leechers"`
	CategoryID    *uuid.UUID     `gorm:"type:uuid" json:"category_id"`
	Category      *Category      `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	SubCategoryID *uuid.UUID     `gorm:"type:uuid" json:"sub_category_id"`
	SubCategory   *SubCategory   `gorm:"foreignKey:SubCategoryID" json:"sub_category,omitempty"`
	VideoID       *uuid.UUID     `gorm:"type:uuid" json:"video_id"`
	Video         *Video         `gorm:"foreignKey:VideoID" json:"video,omitempty"`
	DownloadPath  string         `gorm:"type:text" json:"download_path"`
	Error         string         `gorm:"type:text" json:"error"`
	AddToLibrary  bool           `gorm:"default:false" json:"add_to_library"` // Marquer pour ajout à la bibliothèque
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

type TorrentConfig struct {
	ID             uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name           string         `gorm:"uniqueIndex;not null" json:"name"`
	Source         TorrentSource  `gorm:"type:varchar(50);not null" json:"source"`
	Enabled        bool           `gorm:"default:true" json:"enabled"`
	AutoDownload   bool           `gorm:"default:false" json:"auto_download"`
	SearchKeywords string         `gorm:"type:text" json:"search_keywords"` // Mots-clés séparés par virgules
	BaseURL        string         `gorm:"type:text" json:"base_url"`       // URL personnalisée du site (ex: https://1337x.quest)
	SearchURL      string         `gorm:"type:text" json:"search_url"`     // URL de recherche/liste personnalisée (ex: /watch-movie/)
	CategoryID     *uuid.UUID     `gorm:"type:uuid" json:"category_id"`
	Category       *Category      `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	SubCategoryID  *uuid.UUID     `gorm:"type:uuid" json:"sub_category_id"`
	SubCategory    *SubCategory   `gorm:"foreignKey:SubCategoryID" json:"sub_category,omitempty"`
	MinSeeders     int            `gorm:"default:1" json:"min_seeders"`
	MaxSize        int64          `json:"max_size"` // Taille max en bytes
	Quality        string         `json:"quality"`  // 1080p, 720p, etc.
	Language       string         `json:"language"`
	CheckInterval  int            `gorm:"default:3600" json:"check_interval"` // Intervalle en secondes
	LastCheck      *time.Time     `json:"last_check"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// TorrentCandidate représente un torrent trouvé par le système de recherche automatique
// qui n'a pas encore été téléchargé
type TorrentCandidate struct {
	ID             uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ConfigID       uuid.UUID      `gorm:"type:uuid;not null;index" json:"config_id"`
	Config         *TorrentConfig `gorm:"foreignKey:ConfigID" json:"config,omitempty"`
	Name           string         `gorm:"not null" json:"name"`
	MagnetURL      string         `gorm:"type:text" json:"magnet_url"`
	TorrentURL     string         `gorm:"type:text" json:"torrent_url"`
	TorrentFileURL string         `gorm:"type:text" json:"torrent_file_url"`
	Infohash       string         `gorm:"type:varchar(40);index" json:"infohash"`
	Source         TorrentSource  `gorm:"type:varchar(50);not null" json:"source"`
	Size           int64          `json:"size"` // Taille en bytes
	Seeders        int            `json:"seeders"`
	Leechers       int            `json:"leechers"`
	Quality        string         `json:"quality"`  // 1080p, 720p, etc.
	Language       string         `json:"language"`
	Category       string         `json:"category"` // Catégorie du site source
	UploadDate     string         `json:"upload_date"`
	Status         string         `gorm:"type:varchar(20);default:'pending'" json:"status"` // pending, approved, rejected, downloaded
	ApprovedForDownload bool      `gorm:"default:false" json:"approved_for_download"` // Marqué pour téléchargement automatique
	Rejected        bool          `gorm:"default:false" json:"rejected"` // Rejeté par l'utilisateur
	CategoryID     *uuid.UUID     `gorm:"type:uuid" json:"category_id"`
	CategoryModel  *Category      `gorm:"foreignKey:CategoryID" json:"category_model,omitempty"`
	SubCategoryID  *uuid.UUID     `gorm:"type:uuid" json:"sub_category_id"`
	SubCategoryModel *SubCategory `gorm:"foreignKey:SubCategoryID" json:"sub_category_model,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// TorrentLearningPattern stocke les patterns d'apprentissage pour améliorer la sélection automatique
type TorrentLearningPattern struct {
	ID                uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ConfigID          uuid.UUID      `gorm:"type:uuid;not null;index" json:"config_id"`
	Config            *TorrentConfig `gorm:"foreignKey:ConfigID" json:"config,omitempty"`
	Pattern           string         `gorm:"type:text;not null" json:"pattern"` // Pattern à rechercher dans le nom (regex ou texte)
	PatternType       string         `gorm:"type:varchar(20);default:'keyword'" json:"pattern_type"` // keyword, regex, quality, language, etc.
	Action            string         `gorm:"type:varchar(20);not null" json:"action"` // approve, reject, prefer
	Score             float64        `gorm:"default:1.0" json:"score"` // Score de confiance (0-1)
	UsageCount        int            `gorm:"default:0" json:"usage_count"` // Nombre de fois utilisé
	SuccessCount      int            `gorm:"default:0" json:"success_count"` // Nombre de fois que l'action a été confirmée
	LastUsed          *time.Time     `json:"last_used"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}
