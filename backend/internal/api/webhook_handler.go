package api

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"streammake/internal/config"
	"streammake/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const maxWebhookBodyBytes = 1 << 20 // 1 MiB

var errWebhookPayloadTooLarge = errors.New("payload trop volumineux")

type WebhookHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewWebhookHandler(db *gorm.DB, cfg *config.Config) *WebhookHandler {
	return &WebhookHandler{db: db, cfg: cfg}
}

func (h *WebhookHandler) verifyWebhookToken(c *gin.Context) bool {
	secret := h.cfg.WebhookSecret
	if secret == "" {
		return false
	}
	tok := c.Query("token")
	if tok == "" {
		tok = c.GetHeader("X-Streammake-Webhook-Token")
	}
	if len(tok) != len(secret) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(tok), []byte(secret)) == 1
}

func (h *WebhookHandler) readBody(c *gin.Context) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxWebhookBodyBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxWebhookBodyBytes {
		return nil, errWebhookPayloadTooLarge
	}
	return body, nil
}

func parseEventType(body []byte) string {
	var env struct {
		EventType string `json:"EventType"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return "unknown"
	}
	if env.EventType == "" {
		return "unknown"
	}
	return env.EventType
}

func (h *WebhookHandler) PostRadarr(c *gin.Context) {
	h.postWebhook(c, "radarr")
}

func (h *WebhookHandler) PostSonarr(c *gin.Context) {
	h.postWebhook(c, "sonarr")
}

func (h *WebhookHandler) postWebhook(c *gin.Context, source string) {
	if !h.verifyWebhookToken(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token webhook manquant ou invalide (configurez WEBHOOK_SECRET et le même token dans Radarr/Sonarr)"})
		return
	}
	body, err := h.readBody(c)
	if errors.Is(err, errWebhookPayloadTooLarge) {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "payload trop volumineux"})
		return
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lecture du corps"})
		return
	}

	ev := models.MediaWebhookEvent{
		ID:        uuid.New(),
		Source:    source,
		EventType: parseEventType(body),
		Payload:   string(body),
		CreatedAt: time.Now().UTC(),
	}
	if err := h.db.Create(&ev).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "enregistrement"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "id": ev.ID})
}

// ListWebhookEvents liste les derniers événements (admin).
func (h *WebhookHandler) ListWebhookEvents(c *gin.Context) {
	limit := 50
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	var rows []models.MediaWebhookEvent
	if err := h.db.Order("created_at DESC").Limit(limit).Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "liste"})
		return
	}
	const maxPayloadInList = 12000
	for i := range rows {
		if len(rows[i].Payload) > maxPayloadInList {
			rows[i].Payload = rows[i].Payload[:maxPayloadInList] + "…(tronqué)"
		}
	}
	c.JSON(http.StatusOK, gin.H{"events": rows})
}

// GetWebhookSetup retourne les URL à configurer dans Radarr/Sonarr (sans exposer le secret complet).
func (h *WebhookHandler) GetWebhookSetup(c *gin.Context) {
	base := h.cfg.PublicWebhookBaseURL
	if base == "" {
		base = ""
	}
	secretConfigured := h.cfg.WebhookSecret != ""
	mask := ""
	if secretConfigured && len(h.cfg.WebhookSecret) >= 4 {
		mask = "…" + h.cfg.WebhookSecret[len(h.cfg.WebhookSecret)-4:]
	}
	c.JSON(http.StatusOK, gin.H{
		"secret_configured": secretConfigured,
		"secret_suffix":     mask,
		"public_base_url":   base,
		"paths": gin.H{
			"radarr": "/api/webhooks/radarr",
			"sonarr": "/api/webhooks/sonarr",
		},
	})
}
