package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"streammake/internal/config"
	"streammake/internal/models"
	"streammake/internal/video"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UploadHandler struct {
	db        *gorm.DB
	cfg       *config.Config
	processor *video.Processor
}

func NewUploadHandler(db *gorm.DB, cfg *config.Config) *UploadHandler {
	return &UploadHandler{
		db:        db,
		cfg:       cfg,
		processor: video.NewProcessor(cfg.VideoStoragePath, cfg.ThumbnailStoragePath),
	}
}

func (h *UploadHandler) UploadVideo(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr := userID.(string)
	userUUID, _ := uuid.Parse(userIDStr)

	// Récupérer le fichier
	file, err := c.FormFile("video")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Fichier vidéo requis"})
		return
	}

	// Récupérer les métadonnées
	title := strings.TrimSpace(c.PostForm("title"))
	// Valider et nettoyer le titre
	if title == "" || title == "null" || title == "Null" || title == "NULL" {
		// Utiliser le nom du fichier sans extension comme titre
		title = strings.TrimSuffix(file.Filename, filepath.Ext(file.Filename))
		if title == "" {
			title = "Vidéo sans titre"
		}
	}
	description := strings.TrimSpace(c.PostForm("description"))
	categoryIDStr := c.PostForm("category_id")
	subCategoryIDStr := c.PostForm("sub_category_id")
	isPublic := c.PostForm("is_public") != "false"

	var categoryID, subCategoryID *uuid.UUID
	if categoryIDStr != "" {
		if id, err := uuid.Parse(categoryIDStr); err == nil {
			categoryID = &id
		}
	}
	if subCategoryIDStr != "" {
		if id, err := uuid.Parse(subCategoryIDStr); err == nil {
			subCategoryID = &id
		}
	}

	// Créer l'entrée vidéo
	videoID := uuid.New()
	videoFileName := fmt.Sprintf("%s_%d%s", videoID.String(), time.Now().Unix(), filepath.Ext(file.Filename))
	videoPath := filepath.Join(h.cfg.VideoStoragePath, videoFileName)

	// Sauvegarder le fichier
	if err := c.SaveUploadedFile(file, videoPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de l'enregistrement du fichier"})
		return
	}

	// Créer l'entrée en base
	videoModel := models.Video{
		ID:            videoID,
		Title:         title,
		Description:   description,
		UserID:        userUUID,
		Status:        models.VideoStatusProcessing,
		CategoryID:    categoryID,
		SubCategoryID: subCategoryID,
		IsPublic:      isPublic,
	}

	if err := h.db.Create(&videoModel).Error; err != nil {
		os.Remove(videoPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la création de la vidéo"})
		return
	}

	// Traiter la vidéo en arrière-plan
	go h.processVideo(videoID, videoPath)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Vidéo uploadée avec succès",
		"video_id": videoID,
		"status":   "processing",
	})
}

func (h *UploadHandler) processVideo(videoID uuid.UUID, videoPath string) {
	// Récupérer le titre de la vidéo pour générer une miniature textuelle si nécessaire
	var videoModel models.Video
	h.db.Where("id = ?", videoID).First(&videoModel)
	title := videoModel.Title

	// Générer la miniature depuis la vidéo
	thumbnailPath, err := h.processor.GenerateThumbnail(videoPath, videoID)
	if err != nil {
		// Si la génération depuis la vidéo échoue, créer une miniature textuelle basée sur le titre
		thumbnailPath, err = h.generateTextThumbnail(videoID, title)
		if err != nil {
			fmt.Printf("Erreur lors de la génération de la miniature textuelle: %v\n", err)
		}
	}
	
	if thumbnailPath != "" {
		h.db.Model(&models.Video{}).Where("id = ?", videoID).Update("thumbnail_path", thumbnailPath)
	}

	// Récupérer la durée
	duration, err := h.processor.GetVideoDuration(videoPath)
	if err == nil {
		h.db.Model(&models.Video{}).Where("id = ?", videoID).Update("duration", duration)
	}

	// Créer le format progressif
	progressivePath, err := h.processor.TranscodeToProgressive(videoPath, videoID, "720p")
	if err == nil {
		format := models.VideoFormat{
			ID:       uuid.New(),
			VideoID:  videoID,
			Format:   "progressive",
			FilePath: progressivePath,
			Quality:  "720p",
		}
		h.db.Create(&format)
	}

	// Mettre à jour le statut
	h.db.Model(&models.Video{}).Where("id = ?", videoID).Update("status", models.VideoStatusReady)
}

func (h *UploadHandler) UploadThumbnail(c *gin.Context) {
	videoIDParam := c.Param("id")
	videoID, err := uuid.Parse(videoIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de vidéo invalide"})
		return
	}

	file, err := c.FormFile("thumbnail")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Fichier image requis"})
		return
	}

	thumbnailName := fmt.Sprintf("%s%s", videoID.String(), filepath.Ext(file.Filename))
	thumbnailPath := filepath.Join(h.cfg.ThumbnailStoragePath, thumbnailName)

	if err := c.SaveUploadedFile(file, thumbnailPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de l'enregistrement"})
		return
	}

	h.db.Model(&models.Video{}).Where("id = ?", videoID).Update("thumbnail_path", thumbnailPath)

	c.JSON(http.StatusOK, gin.H{
		"message":        "Miniature uploadée avec succès",
		"thumbnail_path": thumbnailPath,
	})
}

