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

	// Filtre par catégorie
	if categoryID := c.Query("category_id"); categoryID != "" {
		catUUID, err := uuid.Parse(categoryID)
		if err == nil {
			query = query.Where("category_id = ?", catUUID)
		}
	}

	// Recherche
	if search := c.Query("search"); search != "" {
		query = query.Where("title ILIKE ? OR description ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// Trier par vues pour les tendances
	sortBy := c.DefaultQuery("sort", "created_at")
	if sortBy == "views" {
		query = query.Order("views DESC")
	} else {
		query = query.Order("created_at DESC")
	}

	var total int64
	query.Model(&models.Video{}).Count(&total)

	if err := query.Offset(offset).Limit(limit).Find(&videos).Error; err != nil {
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

// GetContinueWatching récupère les vidéos en cours de visionnage
// Peut être appelé depuis les routes publiques ou protégées
func (h *VideoHandler) GetContinueWatching(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		// Pas d'utilisateur connecté, retourner un tableau vide
		c.JSON(http.StatusOK, gin.H{"videos": []interface{}{}})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		c.JSON(http.StatusOK, gin.H{"videos": []interface{}{}})
		return
	}

	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"videos": []interface{}{}})
		return
	}

	var history []models.WatchHistory
	// Récupérer les vidéos avec position > 0 et < 90% de la durée
	if err := h.db.Preload("Video").Preload("Video.User").
		Where("user_id = ? AND position > 0", userUUID).
		Order("watched_at DESC").
		Limit(20).
		Find(&history).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la récupération"})
		return
	}

	// Filtrer pour ne garder que les vidéos non terminées
	var continueWatching []models.WatchHistory
	for _, h := range history {
		if h.Video.Duration > 0 && float64(h.Position) < float64(h.Video.Duration)*0.9 {
			continueWatching = append(continueWatching, h)
		}
	}

	c.JSON(http.StatusOK, gin.H{"videos": continueWatching})
}

func (h *VideoHandler) GetVideo(c *gin.Context) {
	id := c.Param("id")
	
	// Si c'est "continue-watching", rediriger vers le bon handler
	if id == "continue-watching" {
		h.GetContinueWatching(c)
		return
	}
	
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
	userUUID, _ := uuid.Parse(userIDStr)

	var video models.Video
	if err := h.db.Where("id = ?", videoID).First(&video).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vidéo non trouvée"})
		return
	}

	// Vérifier les permissions (propriétaire ou admin)
	if video.UserID != userUUID {
		// Vérifier si admin
		var user models.User
		if err := h.db.Where("id = ?", userUUID).First(&user).Error; err != nil || user.Role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Permission refusée"})
			return
		}
	}

	var req struct {
		Title         string     `json:"title"`
		Description   string     `json:"description"`
		CategoryID    *uuid.UUID `json:"category_id"`
		SubCategoryID *uuid.UUID `json:"sub_category_id"`
		IsPublic      *bool      `json:"is_public"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Title != "" {
		video.Title = req.Title
	}
	if req.Description != "" {
		video.Description = req.Description
	}
	if req.CategoryID != nil {
		video.CategoryID = req.CategoryID
	}
	if req.SubCategoryID != nil {
		video.SubCategoryID = req.SubCategoryID
	}
	if req.IsPublic != nil {
		video.IsPublic = *req.IsPublic
	}

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
	userUUID, _ := uuid.Parse(userIDStr)

	var video models.Video
	if err := h.db.Where("id = ?", videoID).First(&video).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vidéo non trouvée"})
		return
	}

	// Vérifier les permissions (propriétaire ou admin)
	if video.UserID != userUUID {
		var user models.User
		if err := h.db.Where("id = ?", userUUID).First(&user).Error; err != nil || user.Role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Permission refusée"})
			return
		}
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
