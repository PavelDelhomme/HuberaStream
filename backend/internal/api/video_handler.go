package api

import (
	"net/http"
	"strconv"

	"streammake/internal/config"
	"streammake/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type VideoHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewVideoHandler(db *gorm.DB, cfg *config.Config) *VideoHandler {
	return &VideoHandler{db: db, cfg: cfg}
}

type CreateVideoRequest struct {
	Title         string     `json:"title" binding:"required"`
	Description   string     `json:"description"`
	Tags          []string   `json:"tags"`
	CategoryID    *uuid.UUID `json:"category_id"`
	SubCategoryID *uuid.UUID `json:"sub_category_id"`
	IsPublic      bool       `json:"is_public"`
}

func (h *VideoHandler) CreateVideo(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr := userID.(string)
	userUUID, _ := uuid.Parse(userIDStr)

	var req CreateVideoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	video := models.Video{
		ID:            uuid.New(),
		Title:         req.Title,
		Description:   req.Description,
		UserID:        userUUID,
		Status:        models.VideoStatusPending,
		CategoryID:    req.CategoryID,
		SubCategoryID: req.SubCategoryID,
		IsPublic:      req.IsPublic,
	}

	if err := h.db.Create(&video).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la création de la vidéo"})
		return
	}

	c.JSON(http.StatusCreated, video)
}

func (h *VideoHandler) ListVideos(c *gin.Context) {
	var videos []models.Video
	query := h.db.Preload("User").Preload("Formats").Where("status = ?", models.VideoStatusReady)

	// Pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	// Recherche
	if search := c.Query("search"); search != "" {
		query = query.Where("title ILIKE ? OR description ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	var total int64
	query.Model(&models.Video{}).Count(&total)

	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&videos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la récupération des vidéos"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"videos": videos,
		"total":  total,
		"page":   page,
		"limit":  limit,
	})
}

func (h *VideoHandler) GetVideo(c *gin.Context) {
	id := c.Param("id")
	videoID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de vidéo invalide"})
		return
	}

	var video models.Video
	if err := h.db.Preload("User").Preload("Formats").Preload("Tags").Where("id = ?", videoID).First(&video).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vidéo non trouvée"})
		return
	}

	// Incrémenter les vues
	h.db.Model(&video).Update("views", video.Views+1)

	c.JSON(http.StatusOK, video)
}

func (h *VideoHandler) UpdateVideo(c *gin.Context) {
	id := c.Param("id")
	videoID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de vidéo invalide"})
		return
	}

	userID, _ := c.Get("user_id")
	userIDStr := userID.(string)

	var video models.Video
	if err := h.db.Where("id = ?", videoID).First(&video).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vidéo non trouvée"})
		return
	}

	// Vérifier que l'utilisateur est le propriétaire
	if video.UserID.String() != userIDStr {
		c.JSON(http.StatusForbidden, gin.H{"error": "Vous n'êtes pas autorisé à modifier cette vidéo"})
		return
	}

	var req CreateVideoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	video.Title = req.Title
	video.Description = req.Description

	if err := h.db.Save(&video).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la mise à jour"})
		return
	}

	c.JSON(http.StatusOK, video)
}

func (h *VideoHandler) DeleteVideo(c *gin.Context) {
	id := c.Param("id")
	videoID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de vidéo invalide"})
		return
	}

	userID, _ := c.Get("user_id")
	userIDStr := userID.(string)

	var video models.Video
	if err := h.db.Where("id = ?", videoID).First(&video).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vidéo non trouvée"})
		return
	}

	// Vérifier que l'utilisateur est le propriétaire
	if video.UserID.String() != userIDStr {
		c.JSON(http.StatusForbidden, gin.H{"error": "Vous n'êtes pas autorisé à supprimer cette vidéo"})
		return
	}

	if err := h.db.Delete(&video).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la suppression"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Vidéo supprimée avec succès"})
}

func (h *VideoHandler) DownloadFromStream(c *gin.Context) {
	id := c.Param("id")
	videoID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de vidéo invalide"})
		return
	}

	var req struct {
		URL string `json:"url" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var video models.Video
	if err := h.db.Where("id = ?", videoID).First(&video).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vidéo non trouvée"})
		return
	}

	// Mettre à jour le statut et l'URL originale
	video.Status = models.VideoStatusProcessing
	video.OriginalURL = req.URL
	h.db.Save(&video)

	// Démarrer le téléchargement en arrière-plan
	// TODO: Implémenter le service de téléchargement

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Téléchargement démarré",
		"video_id": video.ID,
	})
}

func (h *VideoHandler) GetWatchHistory(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr := userID.(string)
	userUUID, _ := uuid.Parse(userIDStr)

	var history []models.WatchHistory
	if err := h.db.Preload("Video").Preload("Video.User").Where("user_id = ?", userUUID).Order("watched_at DESC").Limit(50).Find(&history).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la récupération de l'historique"})
		return
	}

	c.JSON(http.StatusOK, history)
}

func (h *VideoHandler) AddToWatchHistory(c *gin.Context) {
	videoIDParam := c.Param("videoId")
	videoID, err := uuid.Parse(videoIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de vidéo invalide"})
		return
	}

	userID, _ := c.Get("user_id")
	userIDStr := userID.(string)
	userUUID, _ := uuid.Parse(userIDStr)

	var req struct {
		Position int `json:"position"`
	}
	c.ShouldBindJSON(&req)

	// Mettre à jour ou créer l'entrée d'historique
	var history models.WatchHistory
	h.db.Where("user_id = ? AND video_id = ?", userUUID, videoID).FirstOrCreate(&history, models.WatchHistory{
		UserID:   userUUID,
		VideoID:  videoID,
		Position: req.Position,
	})

	history.Position = req.Position
	h.db.Save(&history)

	c.JSON(http.StatusOK, history)
}

