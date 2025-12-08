package api

import (
	"net/http"
	"os"
	"path/filepath"

	"streammake/internal/config"
	"streammake/internal/models"
	"streammake/internal/video"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DVDHandler struct {
	db        *gorm.DB
	cfg       *config.Config
	extractor *video.DVDExtractor
	processor *video.Processor
}

func NewDVDHandler(db *gorm.DB, cfg *config.Config) *DVDHandler {
	return &DVDHandler{
		db:        db,
		cfg:       cfg,
		extractor: video.NewDVDExtractor(),
		processor: video.NewProcessor(cfg.VideoStoragePath, cfg.ThumbnailStoragePath),
	}
}

// DetectDVD détecte le lecteur DVD
func (h *DVDHandler) DetectDVD(c *gin.Context) {
	device, err := h.extractor.DetectDVDDevice()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Aucun lecteur DVD détecté",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"device": device,
		"message": "Lecteur DVD détecté",
	})
}

// GetDVDInfo récupère les informations du DVD
func (h *DVDHandler) GetDVDInfo(c *gin.Context) {
	devicePath := c.Query("device")
	if devicePath == "" {
		devicePath = "/dev/dvd" // Par défaut
	}

	info, err := h.extractor.GetDVDInfo(devicePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Erreur lors de la lecture du DVD",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, info)
}

type ExtractDVDRequest struct {
	DevicePath         string   `json:"device_path"`
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	CategoryID         string   `json:"category_id"`
	SubCategoryID      string   `json:"sub_category_id"`
	IsPublic           bool     `json:"is_public"`
	VideoTrackIndex    int      `json:"video_track_index"`
	AudioTrackIndices  []int    `json:"audio_track_indices"`
	SubtitleTrackIndices []int  `json:"subtitle_track_indices"`
	Qualities          []string `json:"qualities"` // ["1080p", "720p", "480p"]
}

// ExtractDVD extrait le DVD et crée une vidéo
func (h *DVDHandler) ExtractDVD(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr := userID.(string)
	userUUID, _ := uuid.Parse(userIDStr)

	var req ExtractDVDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Valider les paramètres
	if req.DevicePath == "" {
		req.DevicePath = "/dev/dvd"
	}
	if len(req.Qualities) == 0 {
		req.Qualities = []string{"1080p", "720p", "480p"}
	}
	if len(req.AudioTrackIndices) == 0 {
		req.AudioTrackIndices = []int{0} // Première piste audio par défaut
	}

	// Créer l'ID de la vidéo
	videoID := uuid.New()

	// Créer l'entrée en base
	videoModel := models.Video{
		ID:            videoID,
		Title:         req.Title,
		Description:   req.Description,
		UserID:        userUUID,
		Status:        models.VideoStatusProcessing,
		CategoryID:    nil,
		SubCategoryID: nil,
		IsPublic:      req.IsPublic,
	}

	if req.CategoryID != "" {
		catID, _ := uuid.Parse(req.CategoryID)
		videoModel.CategoryID = &catID
	}
	if req.SubCategoryID != "" {
		subCatID, _ := uuid.Parse(req.SubCategoryID)
		videoModel.SubCategoryID = &subCatID
	}

	if err := h.db.Create(&videoModel).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la création de la vidéo"})
		return
	}

	// Extraire le DVD en arrière-plan
	go h.processDVDExtraction(videoID, req)

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Extraction du DVD démarrée",
		"video_id": videoID,
		"status":   "processing",
	})
}

func (h *DVDHandler) processDVDExtraction(videoID uuid.UUID, req ExtractDVDRequest) {
	// Créer le dossier de sortie
	outputDir := filepath.Join(h.cfg.VideoStoragePath, "dvd", videoID.String())
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		h.db.Model(&models.Video{}).Where("id = ?", videoID).Update("status", models.VideoStatusError)
		return
	}

	// Extraire le DVD en HLS directement
	if err := h.extractor.ExtractDVDToHLS(
		req.DevicePath,
		videoID,
		outputDir,
		req.VideoTrackIndex,
		req.AudioTrackIndices,
		req.SubtitleTrackIndices,
		req.Qualities,
	); err != nil {
		h.db.Model(&models.Video{}).Where("id = ?", videoID).Update("status", models.VideoStatusError)
		return
	}

	// Générer la miniature depuis le premier segment
	firstSegment := filepath.Join(outputDir, "hls", videoID.String(), "1080p", "segment_000.ts")
	if thumbnailPath, err := h.processor.GenerateThumbnail(firstSegment, videoID); err == nil {
		h.db.Model(&models.Video{}).Where("id = ?", videoID).Update("thumbnail_path", thumbnailPath)
	}

	// Récupérer la durée
	if duration, err := h.processor.GetVideoDuration(firstSegment); err == nil {
		h.db.Model(&models.Video{}).Where("id = ?", videoID).Update("duration", duration)
	}

	// Mettre à jour le statut
	h.db.Model(&models.Video{}).Where("id = ?", videoID).Update("status", models.VideoStatusReady)
}

// GetExtractionStatus récupère le statut de l'extraction
func (h *DVDHandler) GetExtractionStatus(c *gin.Context) {
	videoID := c.Param("id")
	
	var video models.Video
	if err := h.db.Where("id = ?", videoID).First(&video).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vidéo non trouvée"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"video_id": video.ID,
		"status":   video.Status,
		"title":    video.Title,
	})
}

