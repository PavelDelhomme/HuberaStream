package api

import (
	"net/http"

	"streammake/internal/config"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MediaSettingsHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewMediaSettingsHandler(db *gorm.DB, cfg *config.Config) *MediaSettingsHandler {
	return &MediaSettingsHandler{
		db:  db,
		cfg: cfg,
	}
}

// GetMediaSettings retourne les paramètres multimédias actuels
func (h *MediaSettingsHandler) GetMediaSettings(c *gin.Context) {
	settings := gin.H{
		"video_storage_path":        h.cfg.VideoStoragePath,
		"thumbnail_storage_path":    h.cfg.ThumbnailStoragePath,
		"torrent_download_path":     h.cfg.TorrentDownloadPath,
		"enable_auto_download":      h.cfg.EnableAutoDownload,
		"default_video_quality":     "720p", // Par défaut
		"enable_hls":                true,
		"enable_dash":               false,
		"hls_segment_duration":      10,
		"max_concurrent_transcodes": 2,
		"ffmpeg_preset":             "medium",
		"ffmpeg_crf":                23,
	}

	c.JSON(http.StatusOK, settings)
}

type UpdateMediaSettingsRequest struct {
	VideoStoragePath        string `json:"video_storage_path"`
	ThumbnailStoragePath    string `json:"thumbnail_storage_path"`
	TorrentDownloadPath     string `json:"torrent_download_path"`
	EnableAutoDownload      bool   `json:"enable_auto_download"`
	DefaultVideoQuality     string `json:"default_video_quality"`
	EnableHLS               bool   `json:"enable_hls"`
	EnableDASH              bool   `json:"enable_dash"`
	HLSSegmentDuration      int    `json:"hls_segment_duration"`
	MaxConcurrentTranscodes int    `json:"max_concurrent_transcodes"`
	FFmpegPreset            string `json:"ffmpeg_preset"`
	FFmpegCRF               int    `json:"ffmpeg_crf"`
}

// UpdateMediaSettings met à jour les paramètres multimédias
func (h *MediaSettingsHandler) UpdateMediaSettings(c *gin.Context) {
	var req UpdateMediaSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Dans un vrai cas, on sauvegarderait en base de données
	// Pour l'instant, on retourne juste un succès
	c.JSON(http.StatusOK, gin.H{
		"message":  "Paramètres sauvegardés",
		"settings": req,
	})
}
