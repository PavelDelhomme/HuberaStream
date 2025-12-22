package ai

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"time"

	"streammake/internal/models"
	"streammake/internal/scraper"
	"streammake/internal/torrent"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SeriesAutoDownloadService gère l'auto-téléchargement intelligent de séries
type SeriesAutoDownloadService struct {
	db             *gorm.DB
	torrentService *torrent.TorrentService
	scraper        scraper.Scraper
	ticker         *time.Ticker
}

// NewSeriesAutoDownloadService crée un nouveau service d'auto-téléchargement de séries
func NewSeriesAutoDownloadService(db *gorm.DB, torrentService *torrent.TorrentService) *SeriesAutoDownloadService {
	return &SeriesAutoDownloadService{
		db:             db,
		torrentService: torrentService,
	}
}

// Start démarre le service d'auto-téléchargement
func (s *SeriesAutoDownloadService) Start() {
	// Vérifier toutes les 6 heures
	s.ticker = time.NewTicker(6 * time.Hour)

	go func() {
		// Exécuter immédiatement au démarrage
		s.checkSeries(context.Background())

		for range s.ticker.C {
			s.checkSeries(context.Background())
		}
	}()
}

// Stop arrête le service
func (s *SeriesAutoDownloadService) Stop() {
	if s.ticker != nil {
		s.ticker.Stop()
	}
}

// checkSeries vérifie toutes les séries suivies pour de nouveaux épisodes
func (s *SeriesAutoDownloadService) checkSeries(ctx context.Context) {
	var series []models.SeriesTracking
	if err := s.db.Where("auto_download = ?", true).Find(&series).Error; err != nil {
		log.Printf("[AI Series] Erreur lors de la récupération des séries: %v", err)
		return
	}

	log.Printf("[AI Series] Vérification de %d série(s) suivie(s)", len(series))

	for _, ser := range series {
		s.checkSeriesEpisodes(ctx, &ser)
	}
}

// checkSeriesEpisodes vérifie et télécharge les nouveaux épisodes d'une série
func (s *SeriesAutoDownloadService) checkSeriesEpisodes(ctx context.Context, series *models.SeriesTracking) {
	log.Printf("[AI Series] Vérification de la série: %s (Saison %d, Épisode %d)", series.Title, series.CurrentSeason, series.CurrentEpisode)

	// Chercher les nouveaux épisodes sur les trackers
	searchQuery := s.buildSearchQuery(series)
	
	// Utiliser le scraper 1337x par défaut
	scraperInstance, err := scraper.NewScraper(models.TorrentSource1337x)
	if err != nil {
		log.Printf("[AI Series] Erreur lors de la création du scraper: %v", err)
		return
	}

	// Rechercher les épisodes
	results, err := scraperInstance.Search(ctx, searchQuery, nil, nil)
	if err != nil {
		log.Printf("[AI Series] Erreur lors de la recherche: %v", err)
		return
	}

	// Limiter à 20 résultats
	if len(results) > 20 {
		results = results[:20]
	}

	// Filtrer et télécharger les nouveaux épisodes
	newEpisodes := s.filterNewEpisodes(series, results)
	
	for _, episode := range newEpisodes {
		if err := s.downloadEpisode(ctx, series, episode); err != nil {
			log.Printf("[AI Series] Erreur lors du téléchargement de l'épisode: %v", err)
			continue
		}
	}

	// Mettre à jour la date de dernière vérification
	now := time.Now()
	series.LastChecked = &now
	s.db.Save(series)
}

// buildSearchQuery construit une requête de recherche pour une série
func (s *SeriesAutoDownloadService) buildSearchQuery(series *models.SeriesTracking) string {
	// Format: "Series Name S01E01" ou "Series Name Season 1 Episode 1"
	query := series.Title
	
	// Ajouter la saison et l'épisode suivants
	nextSeason := series.CurrentSeason
	nextEpisode := series.CurrentEpisode + 1
	
	// Si on dépasse les épisodes de la saison, passer à la saison suivante
	// (On suppose 10 épisodes par saison par défaut, mais on peut améliorer ça)
	if nextEpisode > 20 { // Limite de sécurité
		nextSeason++
		nextEpisode = 1
	}
	
	// Format: "Series Name S02E03" ou "Series Name Season 2 Episode 3"
	query += fmt.Sprintf(" S%02dE%02d", nextSeason, nextEpisode)
	
	// Ajouter la qualité si spécifiée
	if series.Quality != "" {
		query += " " + series.Quality
	}
	
	return query
}

// filterNewEpisodes filtre les résultats pour trouver les nouveaux épisodes
func (s *SeriesAutoDownloadService) filterNewEpisodes(series *models.SeriesTracking, results []*scraper.TorrentResult) []*scraper.TorrentResult {
	newEpisodes := []*scraper.TorrentResult{}
	
	nextSeason := series.CurrentSeason
	nextEpisode := series.CurrentEpisode + 1
	if nextEpisode > 20 {
		nextSeason++
		nextEpisode = 1
	}

	// Patterns pour détecter les épisodes
	patterns := []*regexp.Regexp{
		regexp.MustCompile(fmt.Sprintf(`(?i)S%02dE%02d`, nextSeason, nextEpisode)),
		regexp.MustCompile(fmt.Sprintf(`(?i)Season\s*%d\s*Episode\s*%d`, nextSeason, nextEpisode)),
		regexp.MustCompile(fmt.Sprintf(`(?i)S%02d\.?E%02d`, nextSeason, nextEpisode)),
	}

	for _, result := range results {
		// Vérifier si c'est un épisode qui nous intéresse
		isMatch := false
		for _, pattern := range patterns {
			if pattern.MatchString(result.Name) {
				isMatch = true
				break
			}
		}

		if !isMatch {
			continue
		}

		// Vérifier si l'épisode n'est pas déjà téléchargé
		var existing models.SeriesEpisodeTracking
		err := s.db.Where("series_id = ? AND season = ? AND episode = ?", series.ID, nextSeason, nextEpisode).
			First(&existing).Error
		
		if err == gorm.ErrRecordNotFound {
			// Nouvel épisode trouvé
			newEpisodes = append(newEpisodes, result)
		}
	}

	return newEpisodes
}

// downloadEpisode télécharge un épisode
func (s *SeriesAutoDownloadService) downloadEpisode(ctx context.Context, series *models.SeriesTracking, episode *scraper.TorrentResult) error {
	log.Printf("[AI Series] Téléchargement de l'épisode: %s", episode.Name)

	// Extraire la saison et l'épisode depuis le nom
	season, ep := s.extractSeasonEpisode(episode.Name)
	if season == 0 || ep == 0 {
		return fmt.Errorf("impossible d'extraire la saison/épisode de: %s", episode.Name)
	}

	// Vérifier si le torrent existe déjà
	var existing models.Torrent
	magnetOrURL := episode.MagnetURL
	if magnetOrURL == "" {
		magnetOrURL = episode.TorrentURL
	}

	if err := s.db.Where("magnet_url = ? OR torrent_url = ?", magnetOrURL, episode.TorrentURL).
		First(&existing).Error; err == nil {
		log.Printf("[AI Series] Torrent déjà existant: %s", episode.Name)
		return nil
	}

	// Créer le torrent
	torrent := &models.Torrent{
		ID:         uuid.New(),
		Name:       episode.Name,
		MagnetURL:  episode.MagnetURL,
		TorrentURL: episode.TorrentURL,
		Source:      models.TorrentSource1337x,
		Status:     models.TorrentStatusPending,
		Size:       episode.Size,
		Seeders:    episode.Seeders,
		Leechers:   episode.Leechers,
	}

	if err := s.db.Create(torrent).Error; err != nil {
		return err
	}

	// Ajouter au client torrent
	if s.torrentService != nil {
		if err := s.torrentService.AddTorrent(ctx, torrent); err != nil {
			return err
		}
	}

	// Créer le tracking de l'épisode
	episodeTracking := &models.SeriesEpisodeTracking{
		ID:         uuid.New(),
		SeriesID:   series.ID,
		Season:     season,
		Episode:    ep,
		Title:      episode.Name,
		VideoID:    nil, // Sera mis à jour après traitement
		Downloaded: true,
		Watched:    false,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	s.db.Create(episodeTracking)

	// Mettre à jour la série
	series.CurrentSeason = season
	series.CurrentEpisode = ep
	now := time.Now()
	series.LastDownloaded = &now
	s.db.Save(series)

	log.Printf("[AI Series] ✅ Épisode téléchargé: %s S%02dE%02d", series.Title, season, ep)
	return nil
}

// extractSeasonEpisode extrait la saison et l'épisode depuis un nom de fichier
func (s *SeriesAutoDownloadService) extractSeasonEpisode(name string) (season, episode int) {
	// Pattern: S01E01, S1E1, Season 1 Episode 1, etc.
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)S(\d+)E(\d+)`),
		regexp.MustCompile(`(?i)Season\s*(\d+)\s*Episode\s*(\d+)`),
		regexp.MustCompile(`(?i)S(\d+)\.?E(\d+)`),
	}

	for _, pattern := range patterns {
		matches := pattern.FindStringSubmatch(name)
		if len(matches) >= 3 {
			if s, err := strconv.Atoi(matches[1]); err == nil {
				if e, err := strconv.Atoi(matches[2]); err == nil {
					return s, e
				}
			}
		}
	}

	return 0, 0
}

// AddSeriesToTracking ajoute une série au suivi
func (s *SeriesAutoDownloadService) AddSeriesToTracking(userID uuid.UUID, title string, imdbID string, tmdbID int, posterURL string) (*models.SeriesTracking, error) {
	// Vérifier si la série est déjà suivie
	var existing models.SeriesTracking
	if err := s.db.Where("user_id = ? AND (title = ? OR imdb_id = ?)", userID, title, imdbID).
		First(&existing).Error; err == nil {
		// Mettre à jour si nécessaire
		existing.AutoDownload = true
		s.db.Save(&existing)
		return &existing, nil
	}

	series := &models.SeriesTracking{
		ID:            uuid.New(),
		UserID:        userID,
		Title:         title,
		IMDBID:        imdbID,
		TMDBID:        tmdbID,
		PosterURL:     posterURL,
		CurrentSeason: 1,
		CurrentEpisode: 0,
		AutoDownload:  true,
		Quality:       "1080p",
		Language:      "fr",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.db.Create(series).Error; err != nil {
		return nil, err
	}

	log.Printf("[AI Series] Nouvelle série ajoutée au suivi: %s", title)
	return series, nil
}

