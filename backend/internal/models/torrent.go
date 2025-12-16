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
	TorrentSource1337x  TorrentSource = "1337x"
	TorrentSourceRARBG  TorrentSource = "rarbg"
	TorrentSourceYTS    TorrentSource = "yts"
	TorrentSourceNyaa   TorrentSource = "nyaa"
	TorrentSourceCustom TorrentSource = "custom"
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
