package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"streammake/internal/config"
	"streammake/internal/models"
	"streammake/internal/scraper"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TorrentIndexHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewTorrentIndexHandler(db *gorm.DB, cfg *config.Config) *TorrentIndexHandler {
	return &TorrentIndexHandler{db: db, cfg: cfg}
}

// ListTorrentIndexes récupère tous les index de torrents
func (h *TorrentIndexHandler) ListTorrentIndexes(c *gin.Context) {
	var indexes []models.TorrentIndex
	if err := h.db.Order("priority DESC, name ASC").Find(&indexes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la récupération"})
		return
	}

	c.JSON(http.StatusOK, indexes)
}

// GetTorrentIndex récupère un index par ID
func (h *TorrentIndexHandler) GetTorrentIndex(c *gin.Context) {
	id := c.Param("id")
	indexID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID invalide"})
		return
	}

	var index models.TorrentIndex
	if err := h.db.First(&index, indexID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Index non trouvé"})
		return
	}

	c.JSON(http.StatusOK, index)
}

// CreateTorrentIndex crée un nouvel index
func (h *TorrentIndexHandler) CreateTorrentIndex(c *gin.Context) {
	var req struct {
		Name        string                `json:"name" binding:"required"`
		Source      models.TorrentSource  `json:"source" binding:"required"`
		BaseURL     string                `json:"base_url" binding:"required"`
		SearchPath  string                `json:"search_path"` // Accepté pour compatibilité
		SearchURL   string                `json:"search_url"`  // Utilisé par le modèle
		Enabled     bool                  `json:"enabled"`
		Priority    int                   `json:"priority"`
		IsDefault   bool                  `json:"is_default"`
		Description string                `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Erreur de validation: %v", err.Error())})
		return
	}

	// Si is_default est true, désactiver les autres index par défaut de la même source
	if req.IsDefault {
		h.db.Model(&models.TorrentIndex{}).
			Where("source = ? AND is_default = ?", req.Source, true).
			Update("is_default", false)
	}

	// Utiliser search_path si fourni, sinon search_url (compatibilité)
	searchURL := req.SearchPath
	if searchURL == "" {
		searchURL = req.SearchURL
	}

	index := models.TorrentIndex{
		ID:          uuid.New(),
		Name:        req.Name,
		Source:      req.Source,
		BaseURL:     req.BaseURL,
		SearchURL:   searchURL,
		Enabled:     req.Enabled,
		Priority:    req.Priority,
		IsDefault:   req.IsDefault,
		Description: req.Description,
	}

	if err := h.db.Create(&index).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la création"})
		return
	}

	c.JSON(http.StatusCreated, index)
}

// UpdateTorrentIndex met à jour un index
func (h *TorrentIndexHandler) UpdateTorrentIndex(c *gin.Context) {
	id := c.Param("id")
	indexID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID invalide"})
		return
	}

	var index models.TorrentIndex
	if err := h.db.First(&index, indexID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Index non trouvé"})
		return
	}

	var req struct {
		Name        *string               `json:"name"`
		Source      *models.TorrentSource `json:"source"`
		BaseURL     *string               `json:"base_url"`
		SearchPath  *string               `json:"search_path"` // Accepté pour compatibilité
		SearchURL   *string               `json:"search_url"`  // Utilisé par le modèle
		Enabled     *bool                 `json:"enabled"`
		Priority    *int                  `json:"priority"`
		IsDefault   *bool                 `json:"is_default"`
		Description *string               `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != nil {
		index.Name = *req.Name
	}
	if req.Source != nil {
		index.Source = *req.Source
	}
	if req.BaseURL != nil {
		index.BaseURL = *req.BaseURL
	}
	// Utiliser search_path si fourni, sinon search_url (compatibilité)
	if req.SearchPath != nil && *req.SearchPath != "" {
		index.SearchURL = *req.SearchPath
	} else if req.SearchURL != nil {
		index.SearchURL = *req.SearchURL
	}
	if req.Enabled != nil {
		index.Enabled = *req.Enabled
	}
	if req.Priority != nil {
		index.Priority = *req.Priority
	}
	if req.IsDefault != nil {
		// Si on définit comme défaut, désactiver les autres
		if *req.IsDefault {
			h.db.Model(&models.TorrentIndex{}).
				Where("source = ? AND is_default = ? AND id != ?", index.Source, true, indexID).
				Update("is_default", false)
		}
		index.IsDefault = *req.IsDefault
	}
	if req.Description != nil {
		index.Description = *req.Description
	}

	if err := h.db.Save(&index).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la mise à jour"})
		return
	}

	c.JSON(http.StatusOK, index)
}

// DeleteTorrentIndex supprime un index
func (h *TorrentIndexHandler) DeleteTorrentIndex(c *gin.Context) {
	id := c.Param("id")
	indexID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID invalide"})
		return
	}

	if err := h.db.Delete(&models.TorrentIndex{}, indexID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la suppression"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Index supprimé"})
}

// TestTorrentIndex teste un index en effectuant une recherche de test
func (h *TorrentIndexHandler) TestTorrentIndex(c *gin.Context) {
	id := c.Param("id")
	indexID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID invalide"})
		return
	}

	var index models.TorrentIndex
	if err := h.db.First(&index, indexID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Index non trouvé"})
		return
	}

	// Tester avec une recherche simple
	testQuery := "test"
	scraperInstance, err := scraper.NewScraper(index.Source)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Source non supportée"})
		return
	}

	// Si le scraper supporte les URLs personnalisées, utiliser l'index
	if scraperWithURL, ok := scraperInstance.(scraper.ScraperWithCustomURL); ok {
		scraperWithURL.SetBaseURL(index.BaseURL)
	}

	// Créer un contexte avec timeout pour le test
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	results, err := scraperInstance.Search(ctx, testQuery, nil, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Erreur lors du test",
			"message": err.Error(),
			"success": false,
		})
		return
	}

	// Limiter à 10 résultats pour l'affichage
	displayResults := results
	if len(results) > 10 {
		displayResults = results[:10]
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"message":       fmt.Sprintf("Index testé avec succès, %d résultats trouvés", len(results)),
		"results_count": len(results),
		"results":       displayResults,
		"limited":       len(results) > 10,
	})
}

