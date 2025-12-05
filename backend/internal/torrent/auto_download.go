package torrent

import (
	"context"
	"log"
	"time"

	"streammake/internal/config"
	"streammake/internal/models"
	"streammake/internal/scraper"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AutoDownloadService struct {
	db            *gorm.DB
	cfg           *config.Config
	torrentService *TorrentService
	scraper       scraper.Scraper
	ticker        *time.Ticker
}

func NewAutoDownloadService(db *gorm.DB, cfg *config.Config, torrentService *TorrentService) *AutoDownloadService {
	return &AutoDownloadService{
		db:            db,
		cfg:           cfg,
		torrentService: torrentService,
	}
}

func (s *AutoDownloadService) Start() {
	// Vérifier toutes les configurations toutes les heures
	s.ticker = time.NewTicker(1 * time.Hour)

	go func() {
		// Exécuter immédiatement au démarrage
		s.checkConfigurations(context.Background())

		for range s.ticker.C {
			s.checkConfigurations(context.Background())
		}
	}()
}

func (s *AutoDownloadService) checkConfigurations(ctx context.Context) {
	var configs []models.TorrentConfig
	if err := s.db.Where("enabled = ? AND auto_download = ?", true, true).Find(&configs).Error; err != nil {
		log.Printf("Erreur lors de la récupération des configurations: %v", err)
		return
	}

	for _, config := range configs {
		// Vérifier si on doit faire une recherche maintenant
		shouldCheck := false
		if config.LastCheck == nil {
			shouldCheck = true
		} else {
			elapsed := time.Since(*config.LastCheck)
			if elapsed >= time.Duration(config.CheckInterval)*time.Second {
				shouldCheck = true
			}
		}

		if shouldCheck {
			s.processConfig(ctx, &config)
		}
	}
}

func (s *AutoDownloadService) processConfig(ctx context.Context, config *models.TorrentConfig) {
	log.Printf("Traitement de la configuration: %s", config.Name)

	// Créer le scraper approprié
	scraper, err := scraper.NewScraper(config.Source)
	if err != nil {
		log.Printf("Erreur lors de la création du scraper: %v", err)
		return
	}

	// Construire la requête de recherche
	query := config.SearchKeywords
	if query == "" {
		log.Printf("Aucun mot-clé de recherche pour la configuration %s", config.Name)
		return
	}

	// Rechercher
	var category *models.Category
	var subCategory *models.SubCategory

	if config.CategoryID != nil {
		var cat models.Category
		if err := s.db.First(&cat, *config.CategoryID).Error; err == nil {
			category = &cat
		}
	}

	if config.SubCategoryID != nil {
		var subCat models.SubCategory
		if err := s.db.First(&subCat, *config.SubCategoryID).Error; err == nil {
			subCategory = &subCat
		}
	}

	results, err := scraper.Search(ctx, query, category, subCategory)
	if err != nil {
		log.Printf("Erreur lors de la recherche: %v", err)
		return
	}

	// Filtrer et ajouter les torrents
	for _, result := range results {
		// Vérifier les critères
		if config.MinSeeders > 0 && result.Seeders < config.MinSeeders {
			continue
		}

		if config.MaxSize > 0 && result.Size > config.MaxSize {
			continue
		}

		if config.Quality != "" && result.Quality != config.Quality {
			continue
		}

		// Vérifier si ce torrent existe déjà
		var existing models.Torrent
		magnetOrURL := result.MagnetURL
		if magnetOrURL == "" {
			magnetOrURL = result.TorrentURL
		}

		if err := s.db.Where("magnet_url = ? OR torrent_url = ?", magnetOrURL, result.TorrentURL).First(&existing).Error; err == nil {
			// Torrent déjà existant
			continue
		}

		// Créer le torrent
		torrent := &models.Torrent{
			ID:            uuid.New(),
			Name:          result.Name,
			MagnetURL:     result.MagnetURL,
			TorrentURL:    result.TorrentURL,
			Source:        config.Source,
			Status:        models.TorrentStatusPending,
			Size:          result.Size,
			Seeders:       result.Seeders,
			Leechers:      result.Leechers,
			CategoryID:    config.CategoryID,
			SubCategoryID: config.SubCategoryID,
		}

		if err := s.db.Create(torrent).Error; err != nil {
			log.Printf("Erreur lors de la création du torrent: %v", err)
			continue
		}

		// Ajouter au client torrent
		if err := s.torrentService.AddTorrent(ctx, torrent); err != nil {
			log.Printf("Erreur lors de l'ajout au client: %v", err)
			continue
		}

		log.Printf("Torrent ajouté automatiquement: %s", torrent.Name)
	}

	// Mettre à jour la date de dernière vérification
	now := time.Now()
	config.LastCheck = &now
	s.db.Save(config)
}

func (s *AutoDownloadService) Stop() {
	if s.ticker != nil {
		s.ticker.Stop()
	}
}

