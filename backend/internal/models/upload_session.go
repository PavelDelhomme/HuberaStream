package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UploadStatus string

const (
	UploadStatusPending   UploadStatus = "pending"
	UploadStatusUploading UploadStatus = "uploading"
	UploadStatusPaused    UploadStatus = "paused"
	UploadStatusCompleted UploadStatus = "completed"
	UploadStatusError     UploadStatus = "error"
	UploadStatusCancelled UploadStatus = "cancelled"
)

// UploadSession représente une session d'upload par chunks
type UploadSession struct {
	ID            uuid.UUID   `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID        uuid.UUID   `gorm:"type:uuid;not null" json:"user_id"`
	User          User        `gorm:"foreignKey:UserID" json:"user,omitempty"`
	VideoID       *uuid.UUID  `gorm:"type:uuid" json:"video_id"` // Rempli après finalisation
	Video         *Video      `gorm:"foreignKey:VideoID" json:"video,omitempty"`
	FileName      string      `gorm:"not null" json:"file_name"`
	FileSize      int64       `gorm:"not null" json:"file_size"` // Taille totale du fichier
	ChunkSize     int64       `gorm:"not null" json:"chunk_size"` // Taille de chaque chunk (ex: 5MB)
	TotalChunks   int         `gorm:"not null" json:"total_chunks"` // Nombre total de chunks
	UploadedChunks int        `gorm:"default:0" json:"uploaded_chunks"` // Nombre de chunks uploadés
	UploadedBytes  int64      `gorm:"default:0" json:"uploaded_bytes"` // Bytes uploadés
	Status        UploadStatus `gorm:"default:'pending'" json:"status"`
	Checksum      string      `gorm:"type:varchar(64)" json:"checksum"` // SHA256 du fichier final
	TempPath      string      `gorm:"type:text" json:"temp_path"` // Chemin temporaire des chunks
	Error         string      `gorm:"type:text" json:"error"`
	Metadata      string      `gorm:"type:text" json:"metadata"` // JSON avec title, description, etc.
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// UploadChunk représente un chunk uploadé
type UploadChunk struct {
	ID            uuid.UUID   `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	SessionID     uuid.UUID   `gorm:"type:uuid;not null" json:"session_id"`
	Session       UploadSession `gorm:"foreignKey:SessionID" json:"session,omitempty"`
	ChunkIndex    int         `gorm:"not null" json:"chunk_index"` // Index du chunk (0-based)
	ChunkSize     int64       `gorm:"not null" json:"chunk_size"` // Taille réelle du chunk
	FilePath      string      `gorm:"not null" json:"file_path"` // Chemin vers le chunk sauvegardé
	Checksum      string      `gorm:"type:varchar(64)" json:"checksum"` // SHA256 du chunk
	UploadedAt    time.Time   `json:"uploaded_at"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

