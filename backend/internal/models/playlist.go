package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Playlist struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name        string    `gorm:"not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	UserID      uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	User        User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	IsPublic    bool      `gorm:"default:false" json:"is_public"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	Videos []Video        `gorm:"many2many:playlist_videos;" json:"videos,omitempty"`
	Items  []PlaylistVideo `gorm:"foreignKey:PlaylistID" json:"items,omitempty"`
}

type PlaylistVideo struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	PlaylistID uuid.UUID `gorm:"type:uuid;not null" json:"playlist_id"`
	VideoID    uuid.UUID `gorm:"type:uuid;not null" json:"video_id"`
	Position   int       `gorm:"not null" json:"position"`
	CreatedAt  time.Time `json:"created_at"`

	Playlist Playlist `gorm:"foreignKey:PlaylistID" json:"playlist,omitempty"`
	Video    Video    `gorm:"foreignKey:VideoID" json:"video,omitempty"`
}

type WatchHistory struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	VideoID   uuid.UUID `gorm:"type:uuid;not null" json:"video_id"`
	User      User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Video     Video     `gorm:"foreignKey:VideoID" json:"video,omitempty"`
	Position  int       `gorm:"default:0" json:"position"` // Position en secondes
	WatchedAt time.Time `json:"watched_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

