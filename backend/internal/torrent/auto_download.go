package torrent

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"streammake/internal/config"
	"streammake/internal/models"
	"streammake/internal/scraper"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AutoDownloadService struct {
	db             *gorm.DB
	cfg            *config.Config
	torrentService *TorrentService
	scraper        scraper.Scraper
	ticker         *time.Ticker
}

func NewAutoDownloadService(db *gorm.DB, cfg *config.Config, torrentService *TorrentService) *AutoDownloadService {
	return &AutoDownloadService{
		db:             db,
		cfg:            cfg,
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
	// Vérifier toutes les configurations activées (pas seulement auto_download)
	var configs []models.TorrentConfig
	if err := s.db.Where("enabled = ?", true).Find(&configs).Error; err != nil {
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

	// Télécharger les candidats approuvés pour auto-download
	s.downloadApprovedCandidates(ctx)
}

// ProcessConfig traite une configuration (exporté pour déclenchement manuel)
func (s *AutoDownloadService) ProcessConfig(ctx context.Context, config *models.TorrentConfig) error {
	return s.processConfig(ctx, config)
}

func (s *AutoDownloadService) processConfig(ctx context.Context, config *models.TorrentConfig) error {
	log.Printf("Traitement de la configuration: %s", config.Name)
	
	// Créer un contexte indépendant pour toute la durée du traitement
	// Le contexte parent peut être annulé, mais on veut que le traitement se termine
	processCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	
	log.Printf("Contexte indépendant créé pour la configuration: %s", config.Name)

	// Créer le scraper approprié
	scraperInstance, err := scraper.NewScraper(config.Source)
	if err != nil {
		log.Printf("Erreur lors de la création du scraper: %v", err)
		return err
	}

	// Récupérer l'index de torrent à utiliser (priorité: config.BaseURL > index par défaut)
	baseURL := config.BaseURL
	
	if baseURL == "" {
		// Récupérer l'index par défaut pour cette source
		var index models.TorrentIndex
		if err := s.db.Where("source = ? AND is_default = ? AND enabled = ?", config.Source, true, true).
			First(&index).Error; err == nil {
			baseURL = index.BaseURL
			log.Printf("Utilisation de l'index par défaut: %s (%s)", index.Name, baseURL)
		} else {
			// Fallback sur l'URL par défaut
			if config.Source == models.TorrentSource1337x {
				baseURL = "https://1337x.quest"
				log.Printf("Utilisation de l'URL par défaut: %s", baseURL)
			}
		}
	} else {
		log.Printf("Utilisation de l'URL de la configuration: %s", baseURL)
	}

	// Si c'est un scraper qui supporte les URLs personnalisées, la configurer
	if scraperWithURL, ok := scraperInstance.(scraper.ScraperWithCustomURL); ok {
		if baseURL != "" {
			scraperWithURL.SetBaseURL(baseURL)
		}
	}
	
	// Support spécial pour 1337x.proxyninja.net (utilise le scraper 1337x)
	if config.Source == models.TorrentSource1337x && baseURL == "" {
		// Vérifier si on veut utiliser proxyninja
		if config.BaseURL == "" {
			// Par défaut, utiliser 1337x.quest, mais on peut aussi utiliser proxyninja
			baseURL = "https://1337x.quest"
		}
		if scraperWithURL, ok := scraperInstance.(scraper.ScraperWithCustomURL); ok {
			scraperWithURL.SetBaseURL(baseURL)
		}
	}

	var results []*scraper.TorrentResult

	// Si on a une SearchURL personnalisée, l'utiliser pour parser la page de listing
	if config.SearchURL != "" {
		// Construire l'URL complète (baseURL déjà défini plus haut)
		if baseURL == "" {
			if config.Source == models.TorrentSource1337x {
				baseURL = "https://1337x.quest"
			}
		}
		searchURL := baseURL + config.SearchURL
		
		// Parser la page de listing directement
		if scraperWithURL, ok := scraperInstance.(scraper.ScraperWithCustomURL); ok {
			var err error
			log.Printf("Appel FetchAndParseListing avec URL: %s", searchURL)
			results, err = scraperWithURL.FetchAndParseListing(processCtx, searchURL, config.Language)
			if err != nil {
				log.Printf("Erreur lors du parsing de l'URL personnalisée: %v", err)
				return err
			}
			log.Printf("FetchAndParseListing réussi, %d résultats trouvés", len(results))
		} else {
			log.Printf("Le scraper %s ne supporte pas les URLs personnalisées", config.Source)
			return fmt.Errorf("le scraper %s ne supporte pas les URLs personnalisées", config.Source)
		}
	} else {
		// Utiliser la recherche par mots-clés
		query := config.SearchKeywords
		if query == "" {
			log.Printf("Aucun mot-clé de recherche ni URL personnalisée pour la configuration %s", config.Name)
			return fmt.Errorf("aucun mot-clé de recherche ni URL personnalisée pour la configuration %s", config.Name)
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

		var err error
		log.Printf("Appel Search avec query: %s", query)
		results, err = scraperInstance.Search(processCtx, query, category, subCategory)
		if err != nil {
			log.Printf("Erreur lors de la recherche: %v", err)
			return err
		}
		log.Printf("Search réussi, %d résultats trouvés", len(results))
	}

	// Filtrer et stocker les candidats
	log.Printf("Traitement de %d résultats pour créer des candidats...", len(results))
	candidatesCreated := 0
	candidatesSkipped := 0
	
	for i, result := range results {
		log.Printf("[%d/%d] Traitement du résultat: %s (seeders: %d, size: %d, quality: %s, category: %s)", i+1, len(results), result.Name, result.Seeders, result.Size, result.Quality, result.Category)
		
		// Filtrer les fichiers audio/musique - on ne veut que des vidéos
		if isAudioFile(result) {
			log.Printf("  ⏭️  Ignoré: fichier audio/musique détecté (nom: %s, catégorie: %s)", result.Name, result.Category)
			candidatesSkipped++
			continue
		}
		
		// Vérifier les critères (mais créer quand même le candidat pour permettre à l'utilisateur de le voir)
		// On créera le candidat même si les critères ne sont pas remplis, mais on le marquera comme "rejected"
		shouldRejectDueToFilters := false
		rejectionReason := ""
		
		if config.MinSeeders > 0 && result.Seeders < config.MinSeeders {
			log.Printf("  ⚠️  Seeders (%d) < minimum requis (%d) - candidat créé mais marqué comme rejeté", result.Seeders, config.MinSeeders)
			shouldRejectDueToFilters = true
			rejectionReason = fmt.Sprintf("Seeders insuffisants (%d < %d)", result.Seeders, config.MinSeeders)
		}

		if config.MaxSize > 0 && result.Size > config.MaxSize {
			log.Printf("  ⚠️  Taille (%d) > maximum autorisé (%d) - candidat créé mais marqué comme rejeté", result.Size, config.MaxSize)
			shouldRejectDueToFilters = true
			if rejectionReason != "" {
				rejectionReason += ", "
			}
			rejectionReason += fmt.Sprintf("Taille trop grande (%d > %d)", result.Size, config.MaxSize)
		}

		if config.Quality != "" && result.Quality != config.Quality {
			log.Printf("  ⚠️  Qualité (%s) != qualité requise (%s) - candidat créé mais marqué comme rejeté", result.Quality, config.Quality)
			shouldRejectDueToFilters = true
			if rejectionReason != "" {
				rejectionReason += ", "
			}
			rejectionReason += fmt.Sprintf("Qualité non correspondante (%s != %s)", result.Quality, config.Quality)
		}

		// Vérifier si ce torrent existe déjà (en tant que torrent ou candidat)
		magnetOrURL := result.MagnetURL
		if magnetOrURL == "" {
			magnetOrURL = result.TorrentURL
		}

		// Vérifier dans les torrents existants
		var existingTorrent models.Torrent
		err := s.db.Where("magnet_url = ? OR torrent_url = ?", magnetOrURL, result.TorrentURL).First(&existingTorrent).Error
		if err == nil {
			log.Printf("  ⏭️  Ignoré: torrent déjà existant (ID: %s)", existingTorrent.ID)
			candidatesSkipped++
			continue
		}
		// Ignorer l'erreur "record not found" - c'est normal si le torrent n'existe pas
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("  ⚠️  Erreur lors de la vérification du torrent existant: %v", err)
		}

		// Vérifier dans les candidats existants (uniquement ceux non supprimés)
		var existingCandidate models.TorrentCandidate
		infohash := extractInfohash(result.MagnetURL)
		if infohash != "" {
			err := s.db.Where("infohash = ? AND config_id = ? AND deleted_at IS NULL", infohash, config.ID).First(&existingCandidate).Error
			if err == nil {
				log.Printf("  🔄 Candidat déjà existant (infohash: %s), mise à jour...", infohash[:8]+"...")
				// Candidat déjà existant, mettre à jour les infos
				existingCandidate.Seeders = result.Seeders
				existingCandidate.Leechers = result.Leechers
				existingCandidate.Size = result.Size
				if err := s.db.Save(&existingCandidate).Error; err != nil {
					log.Printf("  ❌ Erreur lors de la mise à jour du candidat: %v", err)
				} else {
					log.Printf("  ✅ Candidat mis à jour: %s", existingCandidate.Name)
				}
				candidatesSkipped++
				continue
			}
			// Ignorer l'erreur "record not found" - c'est normal si le candidat n'existe pas
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				log.Printf("  ⚠️  Erreur lors de la vérification du candidat existant (infohash): %v", err)
			}
		} else {
			// Fallback: vérifier par URL (uniquement ceux non supprimés)
			err := s.db.Where("torrent_url = ? AND config_id = ? AND deleted_at IS NULL", result.TorrentURL, config.ID).First(&existingCandidate).Error
			if err == nil {
				log.Printf("  🔄 Candidat déjà existant (par URL), mise à jour...")
				existingCandidate.Seeders = result.Seeders
				existingCandidate.Leechers = result.Leechers
				existingCandidate.Size = result.Size
				if err := s.db.Save(&existingCandidate).Error; err != nil {
					log.Printf("  ❌ Erreur lors de la mise à jour du candidat: %v", err)
				} else {
					log.Printf("  ✅ Candidat mis à jour: %s", existingCandidate.Name)
				}
				candidatesSkipped++
				continue
			}
			// Ignorer l'erreur "record not found" - c'est normal si le candidat n'existe pas
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				log.Printf("  ⚠️  Erreur lors de la vérification du candidat existant (URL): %v", err)
			}
		}

		// Créer le candidat
		log.Printf("  ➕ Création d'un nouveau candidat...")
		candidate := &models.TorrentCandidate{
			ID:             uuid.New(),
			ConfigID:       config.ID,
			Name:           result.Name,
			MagnetURL:      result.MagnetURL,
			TorrentURL:     result.TorrentURL,
			TorrentFileURL: result.TorrentFileURL,
			Infohash:       infohash,
			Source:         config.Source,
			Size:           result.Size,
			Seeders:        result.Seeders,
			Leechers:       result.Leechers,
			Quality:        result.Quality,
			Language:       result.Language,
			Category:       result.Category,
			UploadDate:     result.UploadDate,
			Status:         "pending",
			CategoryID:     config.CategoryID,
			SubCategoryID:  config.SubCategoryID,
		}

		// Appliquer les patterns d'apprentissage
		shouldApprove, shouldReject := s.applyLearningPatterns(config.ID, result)
		
		// Si le candidat ne respecte pas les filtres, le marquer comme rejeté
		if shouldRejectDueToFilters {
			candidate.Rejected = true
			candidate.Status = "rejected"
			log.Printf("  🚫 Candidat marqué comme rejeté (filtres: %s)", rejectionReason)
		} else if shouldReject {
			candidate.Rejected = true
			candidate.Status = "rejected"
			log.Printf("  🚫 Candidat marqué comme rejeté (pattern d'apprentissage)")
		} else if shouldApprove || config.AutoDownload {
			candidate.ApprovedForDownload = true
			candidate.Status = "approved"
			log.Printf("  ✅ Candidat marqué comme approuvé (auto-download: %v)", config.AutoDownload)
		}

		if err := s.db.Create(candidate).Error; err != nil {
			log.Printf("  ❌ Erreur lors de la création du candidat: %v", err)
			candidatesSkipped++
			continue
		}

		log.Printf("  ✅ Candidat créé avec succès: %s (seeders: %d, status: %s)", candidate.Name, candidate.Seeders, candidate.Status)
		candidatesCreated++
	}
	
	log.Printf("Résumé: %d candidats créés, %d ignorés/sautés sur %d résultats", candidatesCreated, candidatesSkipped, len(results))

	// Mettre à jour la date de dernière vérification
	now := time.Now()
	config.LastCheck = &now
	s.db.Save(config)
	
	return nil
}

// downloadApprovedCandidates télécharge les candidats approuvés pour auto-download
func (s *AutoDownloadService) downloadApprovedCandidates(ctx context.Context) {
	var candidates []models.TorrentCandidate
	if err := s.db.Where("approved_for_download = ? AND status = ? AND rejected = ?", true, "approved", false).Find(&candidates).Error; err != nil {
		log.Printf("Erreur lors de la récupération des candidats approuvés: %v", err)
		return
	}

	for _, candidate := range candidates {
		// Vérifier si ce torrent existe déjà
		var existing models.Torrent
		magnetOrURL := candidate.MagnetURL
		if magnetOrURL == "" {
			magnetOrURL = candidate.TorrentURL
		}

		if err := s.db.Where("magnet_url = ? OR torrent_url = ?", magnetOrURL, candidate.TorrentURL).First(&existing).Error; err == nil {
			// Torrent déjà existant, marquer le candidat comme téléchargé
			candidate.Status = "downloaded"
			s.db.Save(&candidate)
			continue
		}

		// Créer le torrent
		torrent := &models.Torrent{
			ID:            uuid.New(),
			Name:          candidate.Name,
			MagnetURL:     candidate.MagnetURL,
			TorrentURL:    candidate.TorrentURL,
			Source:        candidate.Source,
			Status:        models.TorrentStatusPending,
			Size:          candidate.Size,
			Seeders:       candidate.Seeders,
			Leechers:      candidate.Leechers,
			CategoryID:    candidate.CategoryID,
			SubCategoryID: candidate.SubCategoryID,
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

		// Marquer le candidat comme téléchargé
		candidate.Status = "downloaded"
		s.db.Save(&candidate)

		log.Printf("Torrent téléchargé automatiquement: %s", torrent.Name)
	}
}

// isAudioFile vérifie si un résultat est un fichier audio/musique
func isAudioFile(result *scraper.TorrentResult) bool {
	name := strings.ToLower(result.Name)
	category := strings.ToLower(result.Category)
	
	// Vérifier la catégorie
	if category == "music" || category == "audio" || strings.Contains(category, "music") {
		return true
	}
	
	// Vérifier les extensions audio dans le nom
	audioExtensions := []string{".mp3", ".flac", ".wav", ".aac", ".ogg", ".m4a", ".wma", ".opus", ".ape", ".dsd"}
	for _, ext := range audioExtensions {
		if strings.Contains(name, ext) {
			return true
		}
	}
	
	// Vérifier les mots-clés audio dans le nom
	audioKeywords := []string{
		" mp3 ", " flac ", " wav ", " aac ", " ogg ", " m4a ", " wma ",
		"320kbps", "192kbps", "128kbps", "lossless", "v0", "v2",
		"album", "ep", "single", "mixtape", "soundtrack", "ost",
		"instrumental", "instrumentale", "acapella", "remix", "dj mix",
		"version instrumentale", "instrumental version", "inst.",
	}
	for _, keyword := range audioKeywords {
		if strings.Contains(name, keyword) {
			return true
		}
	}
	
	// Vérifier les patterns spécifiques (ex: "Artist - Album (Year) [Format]")
	// Si le nom contient des patterns typiques de musique mais pas de qualité vidéo
	if !hasVideoQuality(name) && hasMusicPattern(name) {
		return true
	}
	
	return false
}

// hasVideoQuality vérifie si le nom contient une qualité vidéo
func hasVideoQuality(name string) bool {
	videoQualities := []string{"1080p", "720p", "480p", "360p", "2160p", "4k", "bluray", "dvdrip", "webrip", "web-dl", "hdtv", "bdrip", "brrip"}
	for _, quality := range videoQualities {
		if strings.Contains(name, quality) {
			return true
		}
	}
	return false
}

// hasMusicPattern vérifie si le nom contient des patterns typiques de musique
func hasMusicPattern(name string) bool {
	// Patterns typiques de noms de fichiers audio
	musicPatterns := []string{
		" - ", // Format "Artist - Title"
		" [",  // Format avec tags [FLAC], [MP3], etc.
		" (",  // Format avec année ou info (2015), (Instrumental), etc.
	}
	
	patternCount := 0
	for _, pattern := range musicPatterns {
		if strings.Contains(name, pattern) {
			patternCount++
		}
	}
	
	// Si on a plusieurs patterns et pas de qualité vidéo, c'est probablement de la musique
	return patternCount >= 2
}

// extractInfohash extrait l'infohash d'un lien magnet
func extractInfohash(magnetURL string) string {
	if magnetURL == "" {
		return ""
	}
	// Chercher btih: dans le magnet link
	re := regexp.MustCompile(`btih:([A-F0-9]{40})`)
	matches := re.FindStringSubmatch(strings.ToUpper(magnetURL))
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// applyLearningPatterns applique les patterns d'apprentissage pour déterminer si un candidat doit être approuvé ou rejeté
func (s *AutoDownloadService) applyLearningPatterns(configID uuid.UUID, result *scraper.TorrentResult) (shouldApprove, shouldReject bool) {
	var patterns []models.TorrentLearningPattern
	if err := s.db.Where("config_id = ? AND deleted_at IS NULL", configID).Find(&patterns).Error; err != nil {
		return false, false
	}

	approveScore := 0.0
	rejectScore := 0.0

	for _, pattern := range patterns {
		matches := false

		switch pattern.PatternType {
		case "keyword":
			// Recherche de mot-clé dans le nom (insensible à la casse)
			if strings.Contains(strings.ToLower(result.Name), strings.ToLower(pattern.Pattern)) {
				matches = true
			}
		case "regex":
			// Recherche par regex
			re, err := regexp.Compile(pattern.Pattern)
			if err == nil {
				if re.MatchString(result.Name) {
					matches = true
				}
			}
		case "quality":
			// Correspondance de qualité
			if result.Quality == pattern.Pattern {
				matches = true
			}
		case "language":
			// Correspondance de langue
			if result.Language == pattern.Pattern {
				matches = true
			}
		case "seeders":
			// Comparaison de seeders
			minSeeders, err := strconv.Atoi(pattern.Pattern)
			if err == nil && result.Seeders >= minSeeders {
				matches = true
			}
		}

		if matches {
			// Mettre à jour le compteur d'utilisation
			pattern.UsageCount++
			now := time.Now()
			pattern.LastUsed = &now
			s.db.Save(&pattern)

			// Ajouter au score selon l'action
			if pattern.Action == "approve" {
				approveScore += pattern.Score
			} else if pattern.Action == "reject" {
				rejectScore += pattern.Score
			}
		}
	}

	// Si le score de rejet est supérieur, rejeter
	if rejectScore > 0.5 {
		return false, true
	}

	// Si le score d'approbation est supérieur à un seuil, approuver
	if approveScore > 0.7 {
		return true, false
	}

	return false, false
}

func (s *AutoDownloadService) Stop() {
	if s.ticker != nil {
		s.ticker.Stop()
	}
}
