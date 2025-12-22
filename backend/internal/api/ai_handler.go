package api

import (
	"fmt"
	"net/http"

	"streammake/internal/ai"
	"streammake/internal/metadata"
	"streammake/internal/models"
	"streammake/internal/torrent"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AIHandler struct {
	db                  *gorm.DB
	recommendationService *ai.RecommendationService
	detectionService      *ai.DetectionService
	seriesService         *ai.SeriesAutoDownloadService
}

func NewAIHandler(db *gorm.DB, torrentService interface{}) *AIHandler {
	recommendationService := ai.NewRecommendationService(db)
	detectionService := ai.NewDetectionService(db)
	
	handler := &AIHandler{
		db:                  db,
		recommendationService: recommendationService,
		detectionService:      detectionService,
	}
	
	// Initialiser le service de séries si le torrentService est disponible
	if ts, ok := torrentService.(*torrent.TorrentService); ok && ts != nil {
		handler.seriesService = ai.NewSeriesAutoDownloadService(db, ts)
		handler.seriesService.Start()
	}
	
	return handler
}

// GetRecommendations retourne les recommandations pour l'utilisateur connecté
func (h *AIHandler) GetRecommendations(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Non authentifié"})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID utilisateur invalide"})
		return
	}

	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		var parsedLimit int
		if _, err := fmt.Sscanf(limitStr, "%d", &parsedLimit); err == nil && parsedLimit > 0 && parsedLimit <= 50 {
			limit = parsedLimit
		}
	}

	recommendations, err := h.recommendationService.GetRecommendations(userUUID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"recommendations": recommendations,
		"count":          len(recommendations),
	})
}

// RememberMovieSelection mémorise un choix de film pour améliorer les futures détections
func (h *AIHandler) RememberMovieSelection(c *gin.Context) {
	var req struct {
		SearchKey string                      `json:"search_key" binding:"required"`
		Director  string                      `json:"director"`
		Year      int                         `json:"year"`
		Selected  *metadata.MovieMetadata     `json:"selected" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.detectionService.RememberSelection(req.SearchKey, req.Director, req.Year, req.Selected); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Sélection mémorisée avec succès"})
}

// AddSeriesToTracking ajoute une série au suivi automatique
func (h *AIHandler) AddSeriesToTracking(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Non authentifié"})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID utilisateur invalide"})
		return
	}

	var req struct {
		Title     string `json:"title" binding:"required"`
		IMDBID    string `json:"imdb_id"`
		TMDBID    int    `json:"tmdb_id"`
		PosterURL string `json:"poster_url"`
		Quality   string `json:"quality"`
		Language  string `json:"language"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.seriesService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Service de séries non disponible"})
		return
	}

	series, err := h.seriesService.AddSeriesToTracking(userUUID, req.Title, req.IMDBID, req.TMDBID, req.PosterURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, series)
}

// GetTrackedSeries retourne les séries suivies par l'utilisateur
func (h *AIHandler) GetTrackedSeries(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Non authentifié"})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID utilisateur invalide"})
		return
	}

	var series []models.SeriesTracking
	if err := h.db.Where("user_id = ?", userUUID).Find(&series).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"series": series,
		"count": len(series),
	})
}

// UpdateSeriesTracking met à jour le suivi d'une série
func (h *AIHandler) UpdateSeriesTracking(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Non authentifié"})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID utilisateur invalide"})
		return
	}

	seriesID := c.Param("id")
	seriesUUID, err := uuid.Parse(seriesID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID série invalide"})
		return
	}

	var series models.SeriesTracking
	if err := h.db.Where("id = ? AND user_id = ?", seriesUUID, userUUID).First(&series).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Série non trouvée"})
		return
	}

	var req struct {
		AutoDownload *bool   `json:"auto_download"`
		Quality      *string `json:"quality"`
		Language     *string `json:"language"`
		CurrentSeason *int   `json:"current_season"`
		CurrentEpisode *int  `json:"current_episode"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.AutoDownload != nil {
		series.AutoDownload = *req.AutoDownload
	}
	if req.Quality != nil {
		series.Quality = *req.Quality
	}
	if req.Language != nil {
		series.Language = *req.Language
	}
	if req.CurrentSeason != nil {
		series.CurrentSeason = *req.CurrentSeason
	}
	if req.CurrentEpisode != nil {
		series.CurrentEpisode = *req.CurrentEpisode
	}

	if err := h.db.Save(&series).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, series)
}

// DeleteSeriesTracking supprime le suivi d'une série
func (h *AIHandler) DeleteSeriesTracking(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Non authentifié"})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID utilisateur invalide"})
		return
	}

	seriesID := c.Param("id")
	seriesUUID, err := uuid.Parse(seriesID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID série invalide"})
		return
	}

	if err := h.db.Where("id = ? AND user_id = ?", seriesUUID, userUUID).Delete(&models.SeriesTracking{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Série retirée du suivi"})
}

// GetLearningStats retourne les statistiques d'apprentissage de l'IA
func (h *AIHandler) GetLearningStats(c *gin.Context) {
	// Compter les mémoires de détection
	var detectionMemoriesCount int64
	h.db.Model(&models.MovieDetectionMemory{}).Count(&detectionMemoriesCount)

	// Compter les patterns appris
	var patternsCount int64
	h.db.Model(&models.LearningPattern{}).Count(&patternsCount)

	// Compter les recommandations
	var recommendationsCount int64
	h.db.Model(&models.Recommendation{}).Count(&recommendationsCount)

	// Compter les séries suivies
	var seriesCount int64
	h.db.Model(&models.SeriesTracking{}).Count(&seriesCount)

	c.JSON(http.StatusOK, gin.H{
		"detection_memories": detectionMemoriesCount,
		"learning_patterns": patternsCount,
		"recommendations":   recommendationsCount,
		"tracked_series":    seriesCount,
	})
}

