package models

import (
	"time"

	"github.com/google/uuid"
)

// MediaWebhookEvent enregistre un appel webhook Radarr/Sonarr (audit, debug, futures automatisations).
type MediaWebhookEvent struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Source    string    `json:"source" gorm:"size:32;not null;index"`
	EventType string    `json:"event_type" gorm:"size:128;index"`
	Payload   string    `json:"payload" gorm:"type:jsonb"` // JSON brut
	CreatedAt time.Time `json:"created_at"`
}

func (MediaWebhookEvent) TableName() string {
	return "media_webhook_events"
}
