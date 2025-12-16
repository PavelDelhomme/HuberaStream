package api

import (
	"fmt"
	"net/http"
	"strconv"

	"streammake/internal/config"
	"streammake/internal/models"
	"streammake/internal/scraper"
	"streammake/internal/torrent"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TorrentHandler struct {
	db             *gorm.DB
	cfg            *config.Config
	torrentService *torrent.TorrentService
}

func NewTorrentHandler(db *gorm.DB, cfg *config.Config) *TorrentHandler {
	// Initialiser le service torrent
	torrentService, err := torrent.NewTorrentService(db, cfg, torrent.ClientType(cfg.TorrentClientType))
	if err != nil {
		// Log l'erreur mais continue sans service torrent
		// Le service sera disponible quand le client torrent sera configuré
	}

	return &TorrentHandler{
		db:             db,
		cfg:            cfg,
		torrentService: torrentService,
	}
}

func (h *TorrentHandler) ListTorrents(c *gin.Context) {
	var torrents []models.Torrent
	query := h.db.Preload("Category").Preload("SubCategory").Preload("Video")

	// Filtres
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if source := c.Query("source"); source != "" {
		query = query.Where("source = ?", source)
	}

	// Pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	var total int64
	query.Model(&models.Torrent{}).Count(&total)

	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&torrents).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la récupération"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"torrents": torrents,
		"total":    total,
		"page":     page,
		"limit":    limit,
	})
}

func (h *TorrentHandler) CreateTorrent(c *gin.Context) {
	var req struct {
		Name          string               `json:"name" binding:"required"`
		MagnetURL     string               `json:"magnet_url"`
		TorrentURL    string               `json:"torrent_url"`
		Source        models.TorrentSource `json:"source"`
		CategoryID    *uuid.UUID           `json:"category_id"`
		SubCategoryID *uuid.UUID           `json:"sub_category_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.MagnetURL == "" && req.TorrentURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Magnet URL ou Torrent URL requis"})
		return
	}

	torrent := models.Torrent{
		ID:            uuid.New(),
		Name:          req.Name,
		MagnetURL:     req.MagnetURL,
		TorrentURL:    req.TorrentURL,
		Source:        req.Source,
		Status:        models.TorrentStatusPending,
		CategoryID:    req.CategoryID,
		SubCategoryID: req.SubCategoryID,
	}

	if err := h.db.Create(&torrent).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la création"})
		return
	}

	// Démarrer le téléchargement si le service est disponible
	if h.torrentService != nil {
		if err := h.torrentService.AddTorrent(c.Request.Context(), &torrent); err != nil {
			// Log l'erreur mais ne pas échouer la création
			// Le torrent sera créé en base et pourra être ajouté manuellement plus tard
		}
	}

	c.JSON(http.StatusCreated, torrent)
}

func (h *TorrentHandler) UpdateTorrent(c *gin.Context) {
	id := c.Param("id")
	torrentID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID invalide"})
		return
	}

	var torrent models.Torrent
	if err := h.db.Where("id = ?", torrentID).First(&torrent).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Torrent non trouvé"})
		return
	}

	var req struct {
		Status *models.TorrentStatus `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Status != nil {
		torrent.Status = *req.Status
	}

	if err := h.db.Save(&torrent).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la mise à jour"})
		return
	}

	c.JSON(http.StatusOK, torrent)
}

func (h *TorrentHandler) DeleteTorrent(c *gin.Context) {
	id := c.Param("id")
	torrentID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID invalide"})
		return
	}

	var torrent models.Torrent
	if err := h.db.Where("id = ?", torrentID).First(&torrent).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Torrent non trouvé"})
		return
	}

	// Supprimer le torrent du client si le service est disponible
	if h.torrentService != nil {
		// Le service torrent gère la suppression
		_ = h.torrentService.RemoveTorrent(c.Request.Context(), torrentID.String())
	}

	if err := h.db.Delete(&torrent).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la suppression"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Torrent supprimé"})
}

// PauseTorrent met en pause un torrent
func (h *TorrentHandler) PauseTorrent(c *gin.Context) {
	id := c.Param("id")
	torrentID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID invalide"})
		return
	}

	var torrent models.Torrent
	if err := h.db.Where("id = ?", torrentID).First(&torrent).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Torrent non trouvé"})
		return
	}

	if h.torrentService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Service torrent non disponible"})
		return
	}

	if err := h.torrentService.PauseTorrent(c.Request.Context(), torrentID.String()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	torrent.Status = models.TorrentStatusPaused
	h.db.Save(&torrent)

	c.JSON(http.StatusOK, torrent)
}

// ResumeTorrent reprend un torrent
func (h *TorrentHandler) ResumeTorrent(c *gin.Context) {
	id := c.Param("id")
	torrentID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID invalide"})
		return
	}

	var torrent models.Torrent
	if err := h.db.Where("id = ?", torrentID).First(&torrent).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Torrent non trouvé"})
		return
	}

	if h.torrentService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Service torrent non disponible"})
		return
	}

	if err := h.torrentService.ResumeTorrent(c.Request.Context(), torrentID.String()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	torrent.Status = models.TorrentStatusDownloading
	h.db.Save(&torrent)

	c.JSON(http.StatusOK, torrent)
}

// GetTorrentStatus récupère le statut d'un torrent depuis le client
func (h *TorrentHandler) GetTorrentStatus(c *gin.Context) {
	id := c.Param("id")
	torrentID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID invalide"})
		return
	}

	var torrent models.Torrent
	if err := h.db.Where("id = ?", torrentID).First(&torrent).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Torrent non trouvé"})
		return
	}

	if h.torrentService == nil {
		c.JSON(http.StatusOK, torrent)
		return
	}

	// Récupérer le statut depuis le client torrent
	status, err := h.torrentService.GetTorrentStatus(c.Request.Context(), torrentID.String())
	if err != nil {
		c.JSON(http.StatusOK, torrent) // Retourner le statut de la DB si erreur
		return
	}

	// Mettre à jour le torrent avec les infos du client
	torrent.Progress = status.Progress
	torrent.Size = status.Size
	torrent.Downloaded = status.Downloaded
	torrent.Uploaded = status.Uploaded
	torrent.DownloadSpeed = status.DownloadSpeed
	torrent.UploadSpeed = status.UploadSpeed
	torrent.Seeders = status.Seeders
	torrent.Leechers = status.Leechers

	// Mettre à jour le statut
	switch status.Status {
	case models.TorrentStatusDownloading:
		torrent.Status = models.TorrentStatusDownloading
	case models.TorrentStatusSeeding:
		torrent.Status = models.TorrentStatusSeeding
	case models.TorrentStatusPaused:
		torrent.Status = models.TorrentStatusPaused
	}

	// Si le torrent est terminé (100%) et en seeding, le marquer comme completed
	if status.Progress >= 100 && status.Status == models.TorrentStatusSeeding {
		torrent.Status = models.TorrentStatusCompleted
	}

	h.db.Save(&torrent)

	c.JSON(http.StatusOK, torrent)
}

// DownloadFrom1337x télécharge un torrent depuis 1337x
func (h *TorrentHandler) DownloadFrom1337x(c *gin.Context) {
	var req struct {
		TorrentURL    string     `json:"torrent_url" binding:"required"`
		CategoryID    *uuid.UUID `json:"category_id"`
		SubCategoryID *uuid.UUID `json:"sub_category_id"`
		AddToLibrary  bool       `json:"add_to_library"` // Nouveau champ pour marquer pour la bibliothèque
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Récupérer les détails du torrent depuis 1337x
	scraper1337x := scraper.New1337xScraper()
	result, err := scraper1337x.GetTorrentDetails(c.Request.Context(), req.TorrentURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la récupération des détails"})
		return
	}

	// Créer le torrent
	torrent := models.Torrent{
		ID:            uuid.New(),
		Name:          result.Name,
		MagnetURL:     result.MagnetURL,
		TorrentURL:    req.TorrentURL,
		Source:        models.TorrentSource1337x,
		Status:        models.TorrentStatusPending,
		CategoryID:    req.CategoryID,
		SubCategoryID: req.SubCategoryID,
		Size:          result.Size,
		Seeders:       result.Seeders,
		Leechers:      result.Leechers,
		AddToLibrary:  req.AddToLibrary, // Marquer pour la bibliothèque si demandé
	}

	if err := h.db.Create(&torrent).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la création"})
		return
	}

	// Démarrer le téléchargement
	if h.torrentService != nil {
		if err := h.torrentService.AddTorrent(c.Request.Context(), &torrent); err != nil {
			torrent.Status = models.TorrentStatusError
			torrent.Error = err.Error()
			h.db.Save(&torrent)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "torrent": torrent})
			return
		}
		torrent.Status = models.TorrentStatusDownloading
		h.db.Save(&torrent)
	}

	c.JSON(http.StatusCreated, torrent)
}

// ToggleAddToLibrary bascule le flag "add_to_library" d'un torrent
func (h *TorrentHandler) ToggleAddToLibrary(c *gin.Context) {
	id := c.Param("id")
	torrentID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID invalide"})
		return
	}

	var torrent models.Torrent
	if err := h.db.Where("id = ?", torrentID).First(&torrent).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Torrent non trouvé"})
		return
	}

	// Basculer le flag
	torrent.AddToLibrary = !torrent.AddToLibrary
	if err := h.db.Save(&torrent).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la mise à jour"})
		return
	}

	// Si on active le flag et que le torrent n'est pas encore téléchargé, démarrer le téléchargement
	if torrent.AddToLibrary && torrent.Status == models.TorrentStatusPending && h.torrentService != nil {
		if err := h.torrentService.AddTorrent(c.Request.Context(), &torrent); err != nil {
			torrent.Status = models.TorrentStatusError
			torrent.Error = err.Error()
			h.db.Save(&torrent)
			c.JSON(http.StatusOK, gin.H{
				"torrent": torrent,
				"message": "Flag activé mais erreur lors du démarrage du téléchargement",
				"error":   err.Error(),
			})
			return
		}
		torrent.Status = models.TorrentStatusDownloading
		h.db.Save(&torrent)
	}

	c.JSON(http.StatusOK, gin.H{
		"torrent": torrent,
		"message": fmt.Sprintf("Flag 'add_to_library' mis à jour: %v", torrent.AddToLibrary),
	})
}

// Gestion des configurations de torrent
func (h *TorrentHandler) ListTorrentConfigs(c *gin.Context) {
	var configs []models.TorrentConfig
	if err := h.db.Preload("Category").Preload("SubCategory").Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la récupération"})
		return
	}
	c.JSON(http.StatusOK, configs)
}

func (h *TorrentHandler) CreateTorrentConfig(c *gin.Context) {
	var req struct {
		Name           string               `json:"name" binding:"required"`
		Source         models.TorrentSource `json:"source" binding:"required"`
		Enabled        bool                 `json:"enabled"`
		AutoDownload   bool                 `json:"auto_download"`
		SearchKeywords string               `json:"search_keywords"`
		CategoryID     *uuid.UUID           `json:"category_id"`
		SubCategoryID  *uuid.UUID           `json:"sub_category_id"`
		MinSeeders     int                  `json:"min_seeders"`
		MaxSize        int64                `json:"max_size"`
		Quality        string               `json:"quality"`
		Language       string               `json:"language"`
		CheckInterval  int                  `json:"check_interval"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config := models.TorrentConfig{
		ID:             uuid.New(),
		Name:           req.Name,
		Source:         req.Source,
		Enabled:        req.Enabled,
		AutoDownload:   req.AutoDownload,
		SearchKeywords: req.SearchKeywords,
		CategoryID:     req.CategoryID,
		SubCategoryID:  req.SubCategoryID,
		MinSeeders:     req.MinSeeders,
		MaxSize:        req.MaxSize,
		Quality:        req.Quality,
		Language:       req.Language,
		CheckInterval:  req.CheckInterval,
	}

	if err := h.db.Create(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la création"})
		return
	}

	c.JSON(http.StatusCreated, config)
}

func (h *TorrentHandler) UpdateTorrentConfig(c *gin.Context) {
	id := c.Param("id")
	configID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID invalide"})
		return
	}

	var config models.TorrentConfig
	if err := h.db.Where("id = ?", configID).First(&config).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Configuration non trouvée"})
		return
	}

	var req struct {
		Enabled        *bool   `json:"enabled"`
		AutoDownload   *bool   `json:"auto_download"`
		SearchKeywords *string `json:"search_keywords"`
		MinSeeders     *int    `json:"min_seeders"`
		CheckInterval  *int    `json:"check_interval"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Enabled != nil {
		config.Enabled = *req.Enabled
	}
	if req.AutoDownload != nil {
		config.AutoDownload = *req.AutoDownload
	}
	if req.SearchKeywords != nil {
		config.SearchKeywords = *req.SearchKeywords
	}
	if req.MinSeeders != nil {
		config.MinSeeders = *req.MinSeeders
	}
	if req.CheckInterval != nil {
		config.CheckInterval = *req.CheckInterval
	}

	if err := h.db.Save(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la mise à jour"})
		return
	}

	c.JSON(http.StatusOK, config)
}

func (h *TorrentHandler) DeleteTorrentConfig(c *gin.Context) {
	id := c.Param("id")
	configID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID invalide"})
		return
	}

	if err := h.db.Delete(&models.TorrentConfig{}, configID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la suppression"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Configuration supprimée"})
}
