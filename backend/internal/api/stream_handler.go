package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"streammake/internal/config"
	"streammake/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type StreamHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewStreamHandler(db *gorm.DB, cfg *config.Config) *StreamHandler {
	return &StreamHandler{db: db, cfg: cfg}
}

func (h *StreamHandler) StreamVideo(c *gin.Context) {
	id := c.Param("id")
	videoID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de vidéo invalide"})
		return
	}

	format := c.DefaultQuery("format", "progressive")
	quality := c.DefaultQuery("quality", "")

	// Récupérer la vidéo et ses formats
	var video models.Video
	var videoFormat models.VideoFormat

	if err := h.db.Where("id = ?", videoID).First(&video).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vidéo non trouvée"})
		return
	}

	// Trouver le format demandé
	query := h.db.Where("video_id = ? AND format = ?", videoID, format)
	if quality != "" {
		query = query.Where("quality = ?", quality)
	}

	if err := query.First(&videoFormat).Error; err != nil {
		// Si le format n'existe pas, essayer de trouver n'importe quel format disponible
		if err := h.db.Where("video_id = ?", videoID).First(&videoFormat).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Format de vidéo non disponible"})
			return
		}
	}

	filePath := videoFormat.FilePath

	// Vérifier si le fichier existe
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Fichier vidéo non trouvé"})
		return
	}

	// Gérer les différents formats de streaming
	switch strings.ToLower(format) {
	case "hls":
		// Servir le manifeste HLS
		if videoFormat.ManifestPath != "" {
			c.File(videoFormat.ManifestPath)
		} else {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "Manifeste HLS non disponible"})
		}
	case "dash":
		// Servir le manifeste DASH
		if videoFormat.ManifestPath != "" {
			c.File(videoFormat.ManifestPath)
		} else {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "Manifeste DASH non disponible"})
		}
	case "webrtc":
		// WebRTC nécessite une implémentation plus complexe
		c.JSON(http.StatusNotImplemented, gin.H{"error": "WebRTC non encore implémenté"})
	case "progressive", "":
		// Lecture progressive - servir le fichier directement
		c.Header("Content-Type", "video/mp4")
		c.Header("Accept-Ranges", "bytes")
		c.File(filePath)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format non supporté"})
	}
}

// ServeThumbnail sert un fichier de miniature
func (h *StreamHandler) ServeThumbnail(c *gin.Context) {
	filename := c.Param("filename")
	fmt.Printf("[DEBUG ServeThumbnail] Handler appelé avec filename=%s\n", filename)

	if filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nom de fichier requis"})
		return
	}

	// Construire le chemin complet vers le fichier thumbnail
	thumbnailPath := filepath.Join(h.cfg.ThumbnailStoragePath, filename)
	fmt.Printf("[DEBUG ServeThumbnail] Recherche thumbnail: %s (chemin complet: %s, ThumbnailStoragePath: %s)\n", filename, thumbnailPath, h.cfg.ThumbnailStoragePath)

	// Vérifier si le fichier existe
	if _, err := os.Stat(thumbnailPath); os.IsNotExist(err) {
		fmt.Printf("[DEBUG ServeThumbnail] Fichier non trouvé: %s (erreur: %v)\n", thumbnailPath, err)
		// Lister les fichiers dans le dossier pour debug
		if files, err := os.ReadDir(h.cfg.ThumbnailStoragePath); err == nil {
			fmt.Printf("[DEBUG ServeThumbnail] Fichiers disponibles dans %s:\n", h.cfg.ThumbnailStoragePath)
			for _, file := range files {
				fmt.Printf("  - %s\n", file.Name())
			}
		} else {
			fmt.Printf("[DEBUG ServeThumbnail] Erreur lors de la lecture du dossier: %v\n", err)
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "Miniature non trouvée", "path": thumbnailPath})
		return
	}

	fmt.Printf("[DEBUG ServeThumbnail] Fichier trouvé, envoi: %s\n", thumbnailPath)
	// Servir le fichier avec les bons headers
	c.Header("Content-Type", "image/jpeg")
	c.Header("Cache-Control", "public, max-age=604800") // Cache 7 jours
	c.File(thumbnailPath)
}

// StreamSubtitle sert un fichier de sous-titre
func (h *StreamHandler) StreamSubtitle(c *gin.Context) {
	videoIDParam := c.Param("id")
	subtitleIDParam := c.Param("subtitleId")

	videoID, err := uuid.Parse(videoIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de vidéo invalide"})
		return
	}

	subtitleID, err := uuid.Parse(subtitleIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de sous-titre invalide"})
		return
	}

	// Vérifier que la vidéo existe
	var video models.Video
	if err := h.db.Where("id = ?", videoID).First(&video).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vidéo non trouvée"})
		return
	}

	// Récupérer le sous-titre
	var subtitle models.Subtitle
	if err := h.db.Where("id = ? AND video_id = ?", subtitleID, videoID).First(&subtitle).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sous-titre non trouvé"})
		return
	}

	// Vérifier si le fichier existe
	if _, err := os.Stat(subtitle.FilePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Fichier de sous-titre non trouvé"})
		return
	}

	// Servir le fichier avec le bon Content-Type
	if strings.HasSuffix(subtitle.FilePath, ".vtt") {
		c.Header("Content-Type", "text/vtt")
	} else if strings.HasSuffix(subtitle.FilePath, ".srt") {
		c.Header("Content-Type", "text/srt")
	} else {
		c.Header("Content-Type", "text/plain")
	}

	c.Header("Access-Control-Allow-Origin", "*")
	c.File(subtitle.FilePath)
}
