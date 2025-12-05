package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type VideoStatus string

const (
	VideoStatusPending    VideoStatus = "pending"
	VideoStatusProcessing VideoStatus = "processing"
	VideoStatusReady      VideoStatus = "ready"
	VideoStatusError      VideoStatus = "error"
)

type Video struct {
	ID          uuid.UUID   `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Title       string      `gorm:"not null" json:"title"`
	Description string      `gorm:"type:text" json:"description"`
	UserID      uuid.UUID   `gorm:"type:uuid;not null" json:"user_id"`
	User        User        `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Status      VideoStatus `gorm:"default:'pending'" json:"status"`
	Duration    int         `json:"duration"` // en secondes
	Views       int         `gorm:"default:0" json:"views"`
	ThumbnailPath string    `json:"thumbnail_path"`
	OriginalURL   string    `json:"original_url"` // URL source si téléchargé depuis streaming
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	Formats      []VideoFormat `gorm:"foreignKey:VideoID" json:"formats,omitempty"`
	Tags         []VideoTag    `gorm:"many2many:video_tags;" json:"tags,omitempty"`
	Playlists    []Playlist    `gorm:"many2many:playlist_videos;" json:"playlists,omitempty"`
}

type VideoFormat struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	VideoID      uuid.UUID `gorm:"type:uuid;not null" json:"video_id"`
	Format       string    `gorm:"not null" json:"format"` // hls, dash, webrtc, progressive
	FilePath     string    `gorm:"not null" json:"file_path"`
	ManifestPath string    `json:"manifest_path"` // Pour HLS/DASH
	Quality      string    `json:"quality"`      // 1080p, 720p, 480p, etc.
	Bitrate      int       `json:"bitrate"`
	Width        int       `json:"width"`
	Height       int       `json:"height"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type VideoTag struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name      string    `gorm:"uniqueIndex;not null" json:"name"`
	Videos    []Video   `gorm:"many2many:video_tags;" json:"videos,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

