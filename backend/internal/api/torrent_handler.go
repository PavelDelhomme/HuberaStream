package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"streammake/internal/config"
	"streammake/internal/models"
	"streammake/internal/scraper"
	"streammake/internal/torrent"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TorrentHandler struct {
	db                 *gorm.DB
	cfg                *config.Config
	torrentService     *torrent.TorrentService
	autoDownloadService *torrent.AutoDownloadService
}

func NewTorrentHandler(db *gorm.DB, cfg *config.Config) *TorrentHandler {
	// Initialiser le service torrent
	torrentService, err := torrent.NewTorrentService(db, cfg, torrent.ClientType(cfg.TorrentClientType))
	if err != nil {
		// Log l'erreur mais continue sans service torrent
		// Le service sera disponible quand le client torrent sera configuré
	}

	handler := &TorrentHandler{
		db:             db,
		cfg:            cfg,
		torrentService: torrentService,
	}

	// Démarrer le service de téléchargement automatique si le service torrent est disponible
	if torrentService != nil && cfg.EnableAutoDownload {
		autoDownloadService := torrent.NewAutoDownloadService(db, cfg, torrentService)
		autoDownloadService.Start()
		handler.autoDownloadService = autoDownloadService
	}

	return handler
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

	// Mettre à jour le statut des torrents actifs depuis le client torrent
	if h.torrentService != nil {
		ctx := c.Request.Context()
		for i := range torrents {
			torrent := &torrents[i]
			// Ne mettre à jour que les torrents actifs (pending, downloading, seeding)
			if torrent.Status == models.TorrentStatusPending || 
			   torrent.Status == models.TorrentStatusDownloading || 
			   torrent.Status == models.TorrentStatusSeeding {
				// Trouver le torrent dans le client
				clientTorrentID, err := h.findTorrentInClient(ctx, torrent)
				if err == nil {
					// Récupérer le statut depuis le client
					status, err := h.torrentService.GetTorrentStatus(ctx, clientTorrentID)
					if err == nil {
						log.Printf("Mise à jour du torrent '%s': Progress=%.2f%%, Size=%d, Downloaded=%d, Speed=%d/s", 
							torrent.Name, status.Progress, status.Size, status.Downloaded, status.DownloadSpeed)
						
						// Mettre à jour les infos du torrent
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
						
						// Sauvegarder les modifications
						if err := h.db.Save(torrent).Error; err != nil {
							log.Printf("Erreur lors de la sauvegarde du torrent: %v", err)
						} else {
							log.Printf("Torrent '%s' mis à jour avec succès: Progress=%.2f%%, Downloaded=%d bytes", 
								torrent.Name, torrent.Progress, torrent.Downloaded)
						}
					} else {
						log.Printf("Erreur lors de la récupération du statut pour '%s': %v", torrent.Name, err)
					}
				} else {
					log.Printf("Torrent '%s' non trouvé dans le client: %v", torrent.Name, err)
				}
			}
		}
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

	// Trouver l'ID du torrent dans le client torrent en listant tous les torrents
	// et en cherchant celui qui correspond (par nom ou magnet URL)
	clientTorrentID, err := h.findTorrentInClient(c.Request.Context(), &torrent)
	if err != nil {
		log.Printf("Erreur lors de la recherche du torrent dans le client: %v", err)
		// Si le torrent n'est pas dans le client mais qu'on peut l'ajouter, le faire
		if torrent.Status == models.TorrentStatusPending || torrent.Status == models.TorrentStatusError {
			log.Printf("Tentative d'ajout du torrent au client...")
			if addErr := h.torrentService.AddTorrent(c.Request.Context(), &torrent); addErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Torrent non trouvé dans le client et impossible de l'ajouter: %v", addErr)})
				return
			}
			// Réessayer après l'ajout
			time.Sleep(1 * time.Second)
			clientTorrentID, err = h.findTorrentInClient(c.Request.Context(), &torrent)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Torrent ajouté mais toujours non trouvé: %v", err)})
				return
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Torrent non trouvé dans le client: %v. Le torrent doit être en téléchargement pour être mis en pause.", err)})
			return
		}
	}

	if err := h.torrentService.PauseTorrent(c.Request.Context(), clientTorrentID); err != nil {
		log.Printf("Erreur lors de la pause du torrent: %v", err)
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

	// Trouver l'ID du torrent dans le client torrent
	clientTorrentID, err := h.findTorrentInClient(c.Request.Context(), &torrent)
	if err != nil {
		log.Printf("Erreur lors de la recherche du torrent dans le client: %v", err)
		// Si le torrent n'est pas dans le client mais qu'on peut l'ajouter, le faire
		if torrent.Status == models.TorrentStatusPending || torrent.Status == models.TorrentStatusError {
			log.Printf("Tentative d'ajout du torrent au client...")
			if addErr := h.torrentService.AddTorrent(c.Request.Context(), &torrent); addErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Torrent non trouvé dans le client et impossible de l'ajouter: %v", addErr)})
				return
			}
			// Réessayer après l'ajout
			time.Sleep(1 * time.Second)
			clientTorrentID, err = h.findTorrentInClient(c.Request.Context(), &torrent)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Torrent ajouté mais toujours non trouvé: %v", err)})
				return
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Torrent non trouvé dans le client: %v. Le torrent doit être en téléchargement pour être repris.", err)})
			return
		}
	}

	if err := h.torrentService.ResumeTorrent(c.Request.Context(), clientTorrentID); err != nil {
		log.Printf("Erreur lors de la reprise du torrent: %v", err)
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

	// Trouver l'ID du torrent dans le client torrent
	clientTorrentID, err := h.findTorrentInClient(c.Request.Context(), &torrent)
	if err != nil {
		// Si le torrent n'est pas trouvé dans le client, retourner le statut de la DB
		c.JSON(http.StatusOK, torrent)
		return
	}

	// Récupérer le statut depuis le client torrent
	status, err := h.torrentService.GetTorrentStatus(c.Request.Context(), clientTorrentID)
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

	// Ajouter les statistiques des candidats pour chaque configuration
	type ConfigWithStats struct {
		models.TorrentConfig
		CandidatesStats struct {
			Total      int64 `json:"total"`
			Pending    int64 `json:"pending"`
			Approved   int64 `json:"approved"`
			Rejected   int64 `json:"rejected"`
			Downloaded int64 `json:"downloaded"`
		} `json:"candidates_stats"`
	}

	var configsWithStats []ConfigWithStats
	for _, config := range configs {
		configWithStats := ConfigWithStats{
			TorrentConfig: config,
		}

		// Compter les candidats par statut
		h.db.Model(&models.TorrentCandidate{}).Where("config_id = ?", config.ID).Count(&configWithStats.CandidatesStats.Total)
		h.db.Model(&models.TorrentCandidate{}).Where("config_id = ? AND status = ?", config.ID, "pending").Count(&configWithStats.CandidatesStats.Pending)
		h.db.Model(&models.TorrentCandidate{}).Where("config_id = ? AND status = ?", config.ID, "approved").Count(&configWithStats.CandidatesStats.Approved)
		h.db.Model(&models.TorrentCandidate{}).Where("config_id = ? AND status = ?", config.ID, "rejected").Count(&configWithStats.CandidatesStats.Rejected)
		h.db.Model(&models.TorrentCandidate{}).Where("config_id = ? AND status = ?", config.ID, "downloaded").Count(&configWithStats.CandidatesStats.Downloaded)

		configsWithStats = append(configsWithStats, configWithStats)
	}

	c.JSON(http.StatusOK, configsWithStats)
}

func (h *TorrentHandler) CreateTorrentConfig(c *gin.Context) {
	var req struct {
		Name           string               `json:"name" binding:"required"`
		Source         models.TorrentSource `json:"source" binding:"required"`
		Enabled        bool                 `json:"enabled"`
		AutoDownload   bool                 `json:"auto_download"`
		SearchKeywords string               `json:"search_keywords"`
		BaseURL        string               `json:"base_url"`        // URL de base personnalisée
		SearchURL      string               `json:"search_url"`      // URL de recherche/liste personnalisée
		CategoryID     *uuid.UUID           `json:"category_id"`
		SubCategoryID  *uuid.UUID           `json:"sub_category_id"`
		MinSeeders     int                  `json:"min_seeders"`
		MaxSize        int64                `json:"max_size"`
		Quality        string               `json:"quality"`
		Language       string               `json:"language"`
		CheckInterval  int                  `json:"check_interval"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Erreur de validation: %v", err.Error())})
		return
	}

	config := models.TorrentConfig{
		ID:             uuid.New(),
		Name:           req.Name,
		Source:         req.Source,
		Enabled:        req.Enabled,
		AutoDownload:   req.AutoDownload,
		SearchKeywords: req.SearchKeywords,
		BaseURL:        req.BaseURL,
		SearchURL:      req.SearchURL,
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
		Name           *string      `json:"name"`
		Source         *models.TorrentSource `json:"source"`
		Enabled        *bool        `json:"enabled"`
		AutoDownload   *bool        `json:"auto_download"`
		SearchKeywords *string      `json:"search_keywords"`
		BaseURL        *string      `json:"base_url"`
		SearchURL      *string      `json:"search_url"`
		CategoryID     *uuid.UUID   `json:"category_id"`
		SubCategoryID *uuid.UUID   `json:"sub_category_id"`
		MinSeeders     *int         `json:"min_seeders"`
		MaxSize        *int64       `json:"max_size"`
		Quality        *string      `json:"quality"`
		Language       *string      `json:"language"`
		CheckInterval  *int         `json:"check_interval"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != nil {
		config.Name = *req.Name
	}
	if req.Source != nil {
		config.Source = *req.Source
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
	if req.BaseURL != nil {
		config.BaseURL = *req.BaseURL
	}
	if req.SearchURL != nil {
		config.SearchURL = *req.SearchURL
	}
	if req.CategoryID != nil {
		config.CategoryID = req.CategoryID
	}
	if req.SubCategoryID != nil {
		config.SubCategoryID = req.SubCategoryID
	}
	if req.MinSeeders != nil {
		config.MinSeeders = *req.MinSeeders
	}
	if req.MaxSize != nil {
		config.MaxSize = *req.MaxSize
	}
	if req.Quality != nil {
		config.Quality = *req.Quality
	}
	if req.Language != nil {
		config.Language = *req.Language
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

// Gestion des candidats de torrent
func (h *TorrentHandler) ListTorrentCandidates(c *gin.Context) {
	var candidates []models.TorrentCandidate
	query := h.db.Preload("Config").Preload("CategoryModel").Preload("SubCategoryModel")

	// Filtres
	if configID := c.Query("config_id"); configID != "" {
		configUUID, err := uuid.Parse(configID)
		if err == nil {
			query = query.Where("config_id = ?", configUUID)
		}
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if approved := c.Query("approved"); approved != "" {
		if approved == "true" {
			query = query.Where("approved_for_download = ?", true)
		} else {
			query = query.Where("approved_for_download = ?", false)
		}
	}
	if rejected := c.Query("rejected"); rejected != "" {
		if rejected == "true" {
			query = query.Where("rejected = ?", true)
		} else {
			query = query.Where("rejected = ?", false)
		}
	}

	// Pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset := (page - 1) * limit

	var total int64
	query.Model(&models.TorrentCandidate{}).Count(&total)

	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&candidates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la récupération"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"candidates": candidates,
		"total":      total,
		"page":       page,
		"limit":      limit,
	})
}

func (h *TorrentHandler) ApproveTorrentCandidate(c *gin.Context) {
	id := c.Param("id")
	candidateID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID invalide"})
		return
	}

	var candidate models.TorrentCandidate
	if err := h.db.First(&candidate, candidateID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Candidat non trouvé"})
		return
	}

	// Approuver pour téléchargement
	candidate.ApprovedForDownload = true
	candidate.Status = "approved"
	candidate.Rejected = false

	if err := h.db.Save(&candidate).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la mise à jour"})
		return
	}

	c.JSON(http.StatusOK, candidate)
}

// DisapproveTorrentCandidate désapprouve un candidat (le remet en attente)
func (h *TorrentHandler) DisapproveTorrentCandidate(c *gin.Context) {
	id := c.Param("id")
	candidateID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID invalide"})
		return
	}

	var candidate models.TorrentCandidate
	if err := h.db.First(&candidate, candidateID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Candidat non trouvé"})
		return
	}

	// Désapprouver (remettre en attente)
	candidate.ApprovedForDownload = false
	candidate.Status = "pending"
	candidate.Rejected = false

	if err := h.db.Save(&candidate).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la mise à jour"})
		return
	}

	c.JSON(http.StatusOK, candidate)
}

func (h *TorrentHandler) RejectTorrentCandidate(c *gin.Context) {
	id := c.Param("id")
	candidateID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID invalide"})
		return
	}

	var candidate models.TorrentCandidate
	if err := h.db.First(&candidate, candidateID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Candidat non trouvé"})
		return
	}

	// Rejeter
	candidate.Rejected = true
	candidate.ApprovedForDownload = false
	candidate.Status = "rejected"

	if err := h.db.Save(&candidate).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la mise à jour"})
		return
	}

	c.JSON(http.StatusOK, candidate)
}

func (h *TorrentHandler) DownloadTorrentCandidate(c *gin.Context) {
	id := c.Param("id")
	candidateID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID invalide"})
		return
	}

	var candidate models.TorrentCandidate
	if err := h.db.First(&candidate, candidateID).Error; err != nil {
		log.Printf("Erreur lors de la récupération du candidat %s: %v", candidateID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Candidat non trouvé"})
		return
	}

	// Vérifier si ce torrent existe déjà (utiliser gorm.ErrRecordNotFound pour vérifier)
	var existing models.Torrent
	magnetOrURL := candidate.MagnetURL
	if magnetOrURL == "" {
		magnetOrURL = candidate.TorrentURL
	}

	err = h.db.Where("magnet_url = ? OR torrent_url = ?", magnetOrURL, candidate.TorrentURL).First(&existing).Error
	if err == nil {
		// Torrent existe déjà
		candidate.Status = "downloaded"
		h.db.Save(&candidate)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ce torrent existe déjà", "torrent": existing})
		return
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		// Autre erreur que "record not found"
		log.Printf("Erreur lors de la vérification du torrent existant: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la vérification"})
		return
	}

	// Vérifier que le service torrent est disponible
	if h.torrentService == nil {
		log.Printf("Service torrent non disponible")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Service torrent non disponible"})
		return
	}

	// Si on n'a pas de magnet URL mais qu'on a une TorrentURL (page de détails),
	// récupérer les détails pour obtenir le magnet URL
	magnetURL := candidate.MagnetURL
	if magnetURL == "" && candidate.TorrentURL != "" {
		log.Printf("Pas de magnet URL, récupération des détails depuis: %s", candidate.TorrentURL)
		// Vérifier si c'est une URL de page de détails (contient .html ou /torrent/)
		if strings.Contains(candidate.TorrentURL, ".html") || strings.Contains(candidate.TorrentURL, "/torrent/") {
			// Créer un scraper pour récupérer les détails
			scraperInstance, err := scraper.NewScraper(candidate.Source)
			if err == nil {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				
				details, err := scraperInstance.GetTorrentDetails(ctx, candidate.TorrentURL)
				if err == nil && details != nil && details.MagnetURL != "" {
					// Vérifier que le magnet URL est bien formé
					magnetPreviewLen := len(details.MagnetURL)
					if magnetPreviewLen > 100 {
						magnetPreviewLen = 100
					}
					if !strings.HasPrefix(details.MagnetURL, "magnet:") {
						log.Printf("Magnet URL invalide récupéré (ne commence pas par 'magnet:'): %s", details.MagnetURL[:magnetPreviewLen])
					} else if !strings.Contains(strings.ToUpper(details.MagnetURL), "BTIH:") {
						log.Printf("Magnet URL invalide récupéré (pas d'infohash): %s", details.MagnetURL[:magnetPreviewLen])
					} else {
						magnetURL = details.MagnetURL
						magnetPreview := magnetURL
						if len(magnetPreview) > 50 {
							magnetPreview = magnetPreview[:50] + "..."
						}
						log.Printf("Magnet URL récupéré: %s", magnetPreview)
						// Mettre à jour le candidat avec le magnet URL
						candidate.MagnetURL = magnetURL
						if details.Size > 0 {
							candidate.Size = details.Size
						}
						if details.Seeders > 0 {
							candidate.Seeders = details.Seeders
						}
						if details.Leechers > 0 {
							candidate.Leechers = details.Leechers
						}
						h.db.Save(&candidate)
					}
				} else {
					log.Printf("Erreur lors de la récupération des détails: %v", err)
				}
			}
		}
	}

	// Si toujours pas de magnet URL, on ne peut pas télécharger
	if magnetURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Impossible de récupérer le lien magnet ou le fichier torrent"})
		return
	}

	// Créer le torrent
	torrent := &models.Torrent{
		ID:            uuid.New(),
		Name:          candidate.Name,
		MagnetURL:     magnetURL,
		TorrentURL:    candidate.TorrentURL,
		Source:        candidate.Source,
		Status:        models.TorrentStatusPending,
		Size:          candidate.Size,
		Seeders:       candidate.Seeders,
		Leechers:      candidate.Leechers,
		CategoryID:    candidate.CategoryID,
		SubCategoryID: candidate.SubCategoryID,
	}

	if err := h.db.Create(torrent).Error; err != nil {
		log.Printf("Erreur lors de la création du torrent: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la création du torrent"})
		return
	}

	// Ajouter au client torrent
	ctx := c.Request.Context()
	magnetPreview := torrent.MagnetURL
	if len(magnetPreview) > 50 {
		magnetPreview = magnetPreview[:50] + "..."
	}
	log.Printf("Tentative d'ajout du torrent '%s' au client (magnet: %s, download_path: %s)", torrent.Name, magnetPreview, h.cfg.TorrentDownloadPath)
	if err := h.torrentService.AddTorrent(ctx, torrent); err != nil {
		log.Printf("Erreur détaillée lors de l'ajout au client torrent: %v", err)
		log.Printf("  - Nom du torrent: %s", torrent.Name)
		log.Printf("  - Magnet URL: %s", torrent.MagnetURL)
		log.Printf("  - Torrent URL: %s", torrent.TorrentURL)
		log.Printf("  - Chemin de téléchargement: %s", h.cfg.TorrentDownloadPath)
		// Mettre à jour le statut du torrent en erreur
		torrent.Status = models.TorrentStatusError
		torrent.Error = err.Error()
		h.db.Save(torrent)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Erreur lors de l'ajout au client: %v", err), "torrent": torrent})
		return
	}

	// Mettre à jour le statut du torrent
	torrent.Status = models.TorrentStatusDownloading
	h.db.Save(torrent)

	// Marquer le candidat comme téléchargé
	candidate.Status = "downloaded"
	h.db.Save(&candidate)

	c.JSON(http.StatusOK, torrent)
}

func (h *TorrentHandler) DeleteTorrentCandidate(c *gin.Context) {
	id := c.Param("id")
	candidateID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID invalide"})
		return
	}

	if err := h.db.Delete(&models.TorrentCandidate{}, candidateID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la suppression"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Candidat supprimé"})
}

// DeleteAllTorrentCandidatesByConfig supprime tous les candidats d'une configuration
func (h *TorrentHandler) DeleteAllTorrentCandidatesByConfig(c *gin.Context) {
	configID := c.Param("configId")
	configUUID, err := uuid.Parse(configID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de configuration invalide"})
		return
	}

	// Supprimer définitivement tous les candidats de cette configuration (hard delete)
	result := h.db.Unscoped().Where("config_id = ?", configUUID).Delete(&models.TorrentCandidate{})
	if result.Error != nil {
		log.Printf("Erreur lors de la suppression des candidats: %v", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la suppression"})
		return
	}

	log.Printf("Suppression de %d candidat(s) pour la configuration %s", result.RowsAffected, configID)
	c.JSON(http.StatusOK, gin.H{
		"message":      fmt.Sprintf("%d candidat(s) supprimé(s)", result.RowsAffected),
		"deleted_count": result.RowsAffected,
	})
}

// TriggerConfigSearch déclenche manuellement une recherche pour une configuration
func (h *TorrentHandler) TriggerConfigSearch(c *gin.Context) {
	id := c.Param("id")
	configID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID invalide"})
		return
	}

	var config models.TorrentConfig
	if err := h.db.First(&config, configID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Configuration non trouvée"})
		return
	}

	if !config.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "La configuration n'est pas activée"})
		return
	}

	// Créer un service temporaire pour cette recherche si nécessaire
	if h.autoDownloadService == nil {
		if h.torrentService == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Service torrent non disponible"})
			return
		}
		h.autoDownloadService = torrent.NewAutoDownloadService(h.db, h.cfg, h.torrentService)
	}

	// Exécuter la recherche dans une goroutine avec un contexte indépendant
	go func() {
		// Créer un contexte avec un timeout de 5 minutes pour la recherche
		// Utiliser context.Background() pour éviter que le contexte soit annulé
		// par la requête HTTP parente
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		
		// Log pour debug
		log.Printf("Démarrage de la recherche pour la configuration: %s", config.Name)
		log.Printf("Contexte créé, erreur: %v", ctx.Err())
		
		err := h.autoDownloadService.ProcessConfig(ctx, &config)
		if err != nil {
			log.Printf("Erreur lors du traitement de la configuration %s: %v", config.Name, err)
		} else {
			log.Printf("Recherche terminée avec succès pour la configuration: %s", config.Name)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"message": "Recherche déclenchée",
		"config":  config.Name,
	})
}

// Gestion des patterns d'apprentissage
func (h *TorrentHandler) ListLearningPatterns(c *gin.Context) {
	var patterns []models.TorrentLearningPattern
	query := h.db

	if configID := c.Query("config_id"); configID != "" {
		configUUID, err := uuid.Parse(configID)
		if err == nil {
			query = query.Where("config_id = ?", configUUID)
		}
	}

	if err := query.Preload("Config").Order("score DESC, usage_count DESC").Find(&patterns).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la récupération"})
		return
	}

	c.JSON(http.StatusOK, patterns)
}

func (h *TorrentHandler) CreateLearningPattern(c *gin.Context) {
	var req struct {
		ConfigID   uuid.UUID `json:"config_id" binding:"required"`
		Pattern    string    `json:"pattern" binding:"required"`
		PatternType string   `json:"pattern_type" binding:"required"`
		Action     string    `json:"action" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Action != "approve" && req.Action != "reject" && req.Action != "prefer" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Action doit être 'approve', 'reject' ou 'prefer'"})
		return
	}

	pattern := models.TorrentLearningPattern{
		ID:          uuid.New(),
		ConfigID:    req.ConfigID,
		Pattern:     req.Pattern,
		PatternType: req.PatternType,
		Action:      req.Action,
		Score:       1.0,
	}

	if err := h.db.Create(&pattern).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la création"})
		return
	}

	c.JSON(http.StatusCreated, pattern)
}

func (h *TorrentHandler) DeleteLearningPattern(c *gin.Context) {
	id := c.Param("id")
	patternID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID invalide"})
		return
	}

	if err := h.db.Delete(&models.TorrentLearningPattern{}, patternID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la suppression"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Pattern supprimé"})
}

// LearnFromCandidate crée ou met à jour un pattern d'apprentissage basé sur l'action de l'utilisateur
func (h *TorrentHandler) LearnFromCandidate(c *gin.Context) {
	candidateID := c.Param("id")
	candUUID, err := uuid.Parse(candidateID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID invalide"})
		return
	}

	var candidate models.TorrentCandidate
	if err := h.db.First(&candidate, candUUID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Candidat non trouvé"})
		return
	}

	var req struct {
		Action      string `json:"action" binding:"required"` // approve, reject
		PatternType string `json:"pattern_type"`              // keyword, quality, language, etc.
		Pattern     string `json:"pattern"`                   // Si vide, extraire automatiquement
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Déterminer le pattern si non fourni
	pattern := req.Pattern
	patternType := req.PatternType

	if pattern == "" {
		// Extraire automatiquement un pattern basé sur le candidat
		if candidate.Quality != "" {
			pattern = candidate.Quality
			patternType = "quality"
		} else if candidate.Language != "" {
			pattern = candidate.Language
			patternType = "language"
		} else {
			// Extraire des mots-clés du nom (premiers mots significatifs)
			words := strings.Fields(candidate.Name)
			if len(words) > 0 {
				// Prendre les 2-3 premiers mots comme pattern
				if len(words) >= 3 {
					pattern = strings.Join(words[:3], " ")
				} else {
					pattern = strings.Join(words, " ")
				}
				patternType = "keyword"
			}
		}
	}

	if pattern == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Impossible d'extraire un pattern"})
		return
	}

	// Chercher un pattern existant
	var existingPattern models.TorrentLearningPattern
	err = h.db.Where("config_id = ? AND pattern = ? AND pattern_type = ?", candidate.ConfigID, pattern, patternType).First(&existingPattern).Error

	if err == nil {
		// Pattern existant : augmenter le score et le compteur de succès
		existingPattern.SuccessCount++
		existingPattern.UsageCount++
		// Augmenter le score (max 1.0)
		if existingPattern.Score < 1.0 {
			existingPattern.Score += 0.1
			if existingPattern.Score > 1.0 {
				existingPattern.Score = 1.0
			}
		}
		now := time.Now()
		existingPattern.LastUsed = &now
		h.db.Save(&existingPattern)
		c.JSON(http.StatusOK, gin.H{"message": "Pattern mis à jour", "pattern": existingPattern})
	} else {
		// Nouveau pattern
		newPattern := models.TorrentLearningPattern{
			ID:          uuid.New(),
			ConfigID:    candidate.ConfigID,
			Pattern:     pattern,
			PatternType: patternType,
			Action:      req.Action,
			Score:       0.8, // Score initial
			UsageCount:  1,
			SuccessCount: 1,
		}
		now := time.Now()
		newPattern.LastUsed = &now

		if err := h.db.Create(&newPattern).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la création"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "Pattern créé", "pattern": newPattern})
	}
}

// findTorrentInClient trouve l'ID du torrent dans le client torrent en listant tous les torrents
// et en cherchant celui qui correspond (par nom ou magnet URL)
func (h *TorrentHandler) findTorrentInClient(ctx context.Context, torrent *models.Torrent) (string, error) {
	if h.torrentService == nil {
		return "", fmt.Errorf("service torrent non disponible")
	}

	// Si le torrent n'a pas encore été ajouté au client (statut pending), l'ajouter d'abord
	// Mais seulement s'il a un MagnetURL ou TorrentURL valide
	if torrent.Status == models.TorrentStatusPending {
		if torrent.MagnetURL == "" && torrent.TorrentURL == "" {
			return "", fmt.Errorf("torrent sans magnet URL ni torrent URL, impossible de l'ajouter au client")
		}
		log.Printf("Torrent '%s' en statut pending, tentative d'ajout au client...", torrent.Name)
		if err := h.torrentService.AddTorrent(ctx, torrent); err != nil {
			log.Printf("Erreur lors de l'ajout du torrent au client: %v", err)
			// Ne pas retourner d'erreur ici, on va quand même chercher dans la liste
			// au cas où le torrent aurait été ajouté manuellement
		} else {
			// Attendre un peu pour que le client traite l'ajout
			time.Sleep(2 * time.Second)
		}
	}

	// Utiliser une méthode publique du service pour lister les torrents
	allTorrents, err := h.torrentService.ListAllTorrents(ctx)
	if err != nil {
		log.Printf("Erreur lors de la liste des torrents: %v", err)
		return "", fmt.Errorf("erreur lors de la liste des torrents: %w", err)
	}

	log.Printf("Recherche du torrent '%s' parmi %d torrents du client", torrent.Name, len(allTorrents))

	// Normaliser les noms pour la comparaison (remplacer + par espaces, URL decode, etc.)
	normalizeName := func(name string) string {
		// Remplacer + par espaces (encodage URL)
		name = strings.ReplaceAll(name, "+", " ")
		// Décoder les autres caractères URL-encodés si nécessaire
		// Pour l'instant, on fait juste le remplacement de base
		return strings.ToLower(strings.TrimSpace(name))
	}
	
	torrentNameNormalized := normalizeName(torrent.Name)

	// Chercher le torrent correspondant par nom exact (plus fiable)
	for _, t := range allTorrents {
		if t.Name == torrent.Name {
			log.Printf("Torrent trouvé par nom exact: %s (ID: %s)", t.Name, t.ID)
			return t.ID, nil
		}
		// Comparaison normalisée (gère les + et les espaces)
		if normalizeName(t.Name) == torrentNameNormalized {
			log.Printf("Torrent trouvé par nom normalisé: %s (ID: %s) [recherché: %s]", t.Name, t.ID, torrent.Name)
			return t.ID, nil
		}
	}

	// Si pas trouvé par nom exact, essayer par nom partiel (au cas où le nom a été modifié)
	for _, t := range allTorrents {
		tNameNormalized := normalizeName(t.Name)
		// Recherche partielle (au moins 50% du nom correspond)
		if len(torrentNameNormalized) > 10 {
			// Prendre les 30 premiers caractères pour la recherche partielle
			searchPrefix := torrentNameNormalized
			if len(searchPrefix) > 30 {
				searchPrefix = searchPrefix[:30]
			}
			if strings.Contains(tNameNormalized, searchPrefix) {
				log.Printf("Torrent trouvé par nom partiel: %s (ID: %s) [recherché: %s]", t.Name, t.ID, torrent.Name)
				return t.ID, nil
			}
		}
	}

	// Si pas trouvé par nom, essayer par magnet URL (extraire l'infohash)
	if torrent.MagnetURL != "" {
		// Extraire l'infohash du magnet URL
		infohashRe := regexp.MustCompile(`btih:([A-F0-9]{40})`)
		matches := infohashRe.FindStringSubmatch(strings.ToUpper(torrent.MagnetURL))
		if len(matches) > 1 {
			infohash := matches[1]
			log.Printf("Recherche par infohash: %s", infohash)
			// Pour qBittorrent, l'ID est le hash
			// Pour Transmission, on doit chercher par nom
			// Pour Deluge, l'ID est aussi le hash
			for _, t := range allTorrents {
				if strings.ToUpper(t.ID) == infohash {
					log.Printf("Torrent trouvé par infohash: %s (ID: %s)", t.Name, t.ID)
					return t.ID, nil
				}
			}
		}
	}

	// Log tous les noms de torrents pour déboguer
	log.Printf("Torrents disponibles dans le client:")
	for i, t := range allTorrents {
		if i < 10 { // Limiter à 10 pour ne pas surcharger les logs
			log.Printf("  - %s (ID: %s)", t.Name, t.ID)
		}
	}

	return "", fmt.Errorf("torrent '%s' non trouvé dans le client (cherché parmi %d torrents)", torrent.Name, len(allTorrents))
}

// min retourne le minimum de deux entiers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
