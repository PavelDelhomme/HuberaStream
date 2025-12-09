package models

import (
	"time"

	"github.com/google/uuid"
)

type Favorite struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_user_video" json:"user_id"`
	VideoID   uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_user_video" json:"video_id"`
	User      User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Video     Video     `gorm:"foreignKey:VideoID" json:"video,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

