package api

import (
	"net/http"

	"streammake/internal/config"
	"streammake/internal/models"
	"streammake/internal/scraper"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ScraperHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewScraperHandler(db *gorm.DB, cfg *config.Config) *ScraperHandler {
	return &ScraperHandler{db: db, cfg: cfg}
}

func (h *ScraperHandler) Search(c *gin.Context) {
	var req struct {
		Query        string     `json:"query" binding:"required"`
		Source       models.TorrentSource `json:"source" binding:"required"`
		CategoryID   *uuid.UUID `json:"category_id"`
		SubCategoryID *uuid.UUID `json:"sub_category_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Créer le scraper
	scraper, err := scraper.NewScraper(req.Source)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Source non supportée"})
		return
	}

	// Récupérer la catégorie si fournie
	var category *models.Category
	var subCategory *models.SubCategory

	if req.CategoryID != nil {
		var cat models.Category
		if err := h.db.First(&cat, *req.CategoryID).Error; err == nil {
			category = &cat
		}
	}

	if req.SubCategoryID != nil {
		var subCat models.SubCategory
		if err := h.db.First(&subCat, *req.SubCategoryID).Error; err == nil {
			subCategory = &subCat
		}
	}

	// Rechercher
	results, err := scraper.Search(c.Request.Context(), req.Query, category, subCategory)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la recherche"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"count":   len(results),
	})
}

func (h *ScraperHandler) GetTorrentDetails(c *gin.Context) {
	torrentURL := c.Query("url")
	sourceStr := c.Query("source")

	if torrentURL == "" || sourceStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL et source requis"})
		return
	}

	source := models.TorrentSource(sourceStr)
	scraper, err := scraper.NewScraper(source)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Source non supportée"})
		return
	}

	result, err := scraper.GetTorrentDetails(c.Request.Context(), torrentURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la récupération"})
		return
	}

	c.JSON(http.StatusOK, result)
}

