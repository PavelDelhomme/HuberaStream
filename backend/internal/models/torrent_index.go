package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TorrentIndex représente un index de torrents configurable
type TorrentIndex struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name        string         `gorm:"not null" json:"name"`                    // Nom de l'index (ex: "1337x Main", "1337x Mirror")
	Source      TorrentSource  `gorm:"type:varchar(50);not null" json:"source"` // Type de source (1337x, etc.)
	BaseURL     string         `gorm:"type:text;not null" json:"base_url"`      // URL de base (ex: https://1337x.quest)
	SearchURL   string         `gorm:"type:text" json:"search_url"`            // URL de recherche (ex: /lmsearch?q={query}&category=lmsearch)
	Enabled     bool           `gorm:"default:true" json:"enabled"`              // Si l'index est activé
	Priority    int            `gorm:"default:0" json:"priority"`              // Priorité (plus élevé = utilisé en premier)
	IsDefault   bool           `gorm:"default:false" json:"is_default"`        // Si c'est l'index par défaut pour cette source
	Description string         `gorm:"type:text" json:"description"`          // Description de l'index
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName spécifie le nom de la table pour GORM
func (TorrentIndex) TableName() string {
	return "torrent_indexes"
}

