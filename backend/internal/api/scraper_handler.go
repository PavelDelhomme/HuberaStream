package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"streammake/internal/config"
	"streammake/internal/models"
	"streammake/internal/scraper"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ScraperHandler struct {
	db            *gorm.DB
	cfg           *config.Config
	crawler       *scraper.CrawlerService
	crawlerCtx    context.Context
	crawlerCancel context.CancelFunc
}

func NewScraperHandler(db *gorm.DB, cfg *config.Config) *ScraperHandler {
	return &ScraperHandler{
		db:      db,
		cfg:     cfg,
		crawler: scraper.NewCrawlerService(),
	}
}

func (h *ScraperHandler) Search(c *gin.Context) {
	var req struct {
		Query         string               `json:"query" binding:"required"`
		Source        models.TorrentSource `json:"source" binding:"required"`
		CategoryID    *uuid.UUID           `json:"category_id"`
		SubCategoryID *uuid.UUID           `json:"sub_category_id"`
		IndexID       *uuid.UUID           `json:"index_id"` // Optionnel : utiliser un index spécifique
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Créer le scraper
	scraperInstance, err := scraper.NewScraper(req.Source)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Source non supportée"})
		return
	}

	// Récupérer l'index à utiliser
	var torrentIndex *models.TorrentIndex
	if req.IndexID != nil {
		// Utiliser l'index spécifié
		var index models.TorrentIndex
		if err := h.db.First(&index, *req.IndexID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Index non trouvé"})
			return
		}
		if !index.Enabled {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Index désactivé"})
			return
		}
		torrentIndex = &index
	} else {
		// Utiliser l'index par défaut pour cette source
		var index models.TorrentIndex
		if err := h.db.Where("source = ? AND is_default = ? AND enabled = ?", req.Source, true, true).
			First(&index).Error; err == nil {
			torrentIndex = &index
		}
	}

	// Configurer le scraper avec l'index si disponible
	if torrentIndex != nil {
		if scraperWithURL, ok := scraperInstance.(scraper.ScraperWithCustomURL); ok {
			scraperWithURL.SetBaseURL(torrentIndex.BaseURL)
		}
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
	results, err := scraperInstance.Search(c.Request.Context(), req.Query, category, subCategory)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Erreur lors de la recherche",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"count":   len(results),
		"index_used": func() string {
			if torrentIndex != nil {
				return torrentIndex.Name
			}
			return "default"
		}(),
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

// BrowseCategory1337x parcourt une catégorie 1337x
func (h *ScraperHandler) BrowseCategory1337x(c *gin.Context) {
	categoryStr := c.Query("category")
	pageStr := c.DefaultQuery("page", "1")
	language := c.Query("language")

	page, err := strconv.Atoi(pageStr)
	if err != nil {
		page = 1
	}

	scraper1337x := scraper.New1337xScraper()
	category := scraper.Category1337x(categoryStr)

	results, err := scraper1337x.BrowseCategory(c.Request.Context(), category, page, language)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"results":  results,
		"count":    len(results),
		"category": categoryStr,
		"page":     page,
		"language": language,
	})
}

// StartCrawler démarre le crawling automatique
func (h *ScraperHandler) StartCrawler(c *gin.Context) {
	var req struct {
		Interval   int      `json:"interval"` // en minutes
		Categories []string `json:"categories"`
		Languages  []string `json:"languages"`
		UseProxy   bool     `json:"use_proxy"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.crawler.IsRunning() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Crawler déjà en cours d'exécution"})
		return
	}

	// Convertir les catégories
	categories := make([]scraper.Category1337x, 0)
	for _, catStr := range req.Categories {
		categories = append(categories, scraper.Category1337x(catStr))
	}
	if len(categories) == 0 {
		// Par défaut, toutes les catégories
		categories = []scraper.Category1337x{
			scraper.CategoryPopularMovies,
			scraper.CategoryTV,
			scraper.CategoryMovieLibrary,
			scraper.CategoryAnime,
			scraper.CategoryDocumentaries,
			scraper.CategoryMovies,
		}
	}
	h.crawler.SetCategories(categories)

	// Définir les langues
	if len(req.Languages) > 0 {
		h.crawler.SetLanguages(req.Languages)
	}

	// Activer/désactiver proxy
	h.crawler.EnableProxy(req.UseProxy)

	// Démarrer le crawler
	interval := time.Duration(req.Interval) * time.Minute
	if interval == 0 {
		interval = 60 * time.Minute // Par défaut, toutes les heures
	}

	ctx, cancel := context.WithCancel(context.Background())
	h.crawlerCtx = ctx
	h.crawlerCancel = cancel

	if err := h.crawler.Start(ctx, interval); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":          "Crawler démarré",
		"interval_minutes": req.Interval,
		"categories":       req.Categories,
		"languages":        req.Languages,
		"use_proxy":        req.UseProxy,
	})
}

// StopCrawler arrête le crawling
func (h *ScraperHandler) StopCrawler(c *gin.Context) {
	if !h.crawler.IsRunning() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Crawler non démarré"})
		return
	}

	h.crawler.Stop()
	if h.crawlerCancel != nil {
		h.crawlerCancel()
	}

	c.JSON(http.StatusOK, gin.H{"message": "Crawler arrêté"})
}

// GetCrawlerStatus retourne le statut du crawler
func (h *ScraperHandler) GetCrawlerStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"running": h.crawler.IsRunning(),
	})
}
