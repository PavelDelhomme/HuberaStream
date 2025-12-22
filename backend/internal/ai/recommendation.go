package ai

import (
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"streammake/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RecommendationService gère les recommandations intelligentes
type RecommendationService struct {
	db *gorm.DB
}

// NewRecommendationService crée un nouveau service de recommandation
func NewRecommendationService(db *gorm.DB) *RecommendationService {
	return &RecommendationService{
		db: db,
	}
}

// GetRecommendations génère des recommandations pour un utilisateur
func (s *RecommendationService) GetRecommendations(userID uuid.UUID, limit int) ([]*models.Recommendation, error) {
	if limit <= 0 {
		limit = 10
	}

	// Charger les préférences de l'utilisateur
	preferences, err := s.loadOrCreatePreferences(userID)
	if err != nil {
		return nil, err
	}

	// Obtenir l'historique de visionnage
	watchHistory, err := s.getWatchHistory(userID)
	if err != nil {
		return nil, err
	}

	// Si pas d'historique, retourner des recommandations basiques (tendances, nouveautés)
	if len(watchHistory) == 0 {
		return s.getDefaultRecommendations(userID, limit)
	}

	// Analyser les préférences depuis l'historique
	s.analyzePreferences(userID, watchHistory, preferences)

	// Générer des recommandations basées sur les préférences
	recommendations := s.generateRecommendations(userID, preferences, watchHistory, limit)

	// Sauvegarder les recommandations
	for _, rec := range recommendations {
		if err := s.db.Create(rec).Error; err != nil {
			log.Printf("[AI] Erreur lors de la sauvegarde de la recommandation: %v", err)
		}
	}

	return recommendations, nil
}

// loadOrCreatePreferences charge ou crée les préférences d'un utilisateur
func (s *RecommendationService) loadOrCreatePreferences(userID uuid.UUID) (*models.UserPreference, error) {
	var preferences models.UserPreference
	if err := s.db.Where("user_id = ?", userID).First(&preferences).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Créer des préférences par défaut
			preferences = models.UserPreference{
				ID:              uuid.New(),
				UserID:          userID,
				MinRating:       0,
				PreferredQuality: "1080p",
				PreferredLanguage: "fr",
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			}
			if err := s.db.Create(&preferences).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return &preferences, nil
}

// getWatchHistory récupère l'historique de visionnage d'un utilisateur
func (s *RecommendationService) getWatchHistory(userID uuid.UUID) ([]models.WatchHistory, error) {
	var history []models.WatchHistory
	if err := s.db.Where("user_id = ?", userID).
		Preload("Video").
		Order("watched_at DESC").
		Limit(100).
		Find(&history).Error; err != nil {
		return nil, err
	}
	return history, nil
}

// analyzePreferences analyse l'historique pour mettre à jour les préférences
func (s *RecommendationService) analyzePreferences(userID uuid.UUID, history []models.WatchHistory, preferences *models.UserPreference) {
	genreCount := make(map[string]int)
	directorCount := make(map[string]int)
	actorCount := make(map[string]int)
	yearCount := make(map[int]int)
	ratingSum := 0.0
	ratingCount := 0

	for _, entry := range history {
		if entry.Video.ID == uuid.Nil {
			continue
		}

		// Compter les genres
		if entry.Video.Genre != "" {
			genres := strings.Split(entry.Video.Genre, ",")
			for _, genre := range genres {
				genre = strings.TrimSpace(genre)
				if genre != "" {
					genreCount[genre]++
				}
			}
		}

		// Compter les réalisateurs
		if entry.Video.Director != "" {
			directorCount[entry.Video.Director]++
		}

		// Compter les acteurs
		if entry.Video.Actors != "" {
			actors := strings.Split(entry.Video.Actors, ",")
			for _, actor := range actors {
				actor = strings.TrimSpace(actor)
				if actor != "" {
					actorCount[actor]++
				}
			}
		}

		// Compter les années
		if entry.Video.ProductionYear > 1900 {
			yearCount[entry.Video.ProductionYear]++
		}

		// Calculer la note moyenne
		if entry.Video.Rating > 0 {
			ratingSum += entry.Video.Rating
			ratingCount++
		}
	}

	// Mettre à jour les préférences
	preferences.TotalWatched = len(history)
	
	// Top 5 genres
	preferences.PreferredGenres = s.getTopN(genreCount, 5)
	
	// Top 5 réalisateurs
	preferences.PreferredDirectors = s.getTopN(directorCount, 5)
	
	// Top 10 acteurs
	preferences.PreferredActors = s.getTopN(actorCount, 10)
	
	// Note minimale (moyenne - 1)
	if ratingCount > 0 {
		avgRating := ratingSum / float64(ratingCount)
		preferences.MinRating = math.Max(0, avgRating-1.0)
	}

	// Période préférée (plage d'années la plus regardée)
	if len(yearCount) > 0 {
		minYear, maxYear := s.getYearRange(yearCount)
		preferences.PreferredYears = fmt.Sprintf("%d-%d", minYear, maxYear)
	}

	preferences.LastUpdated = time.Now()
	preferences.UpdatedAt = time.Now()
	s.db.Save(preferences)
}

// getTopN retourne les N éléments les plus fréquents
func (s *RecommendationService) getTopN(countMap map[string]int, n int) string {
	type item struct {
		key   string
		count int
	}
	items := make([]item, 0, len(countMap))
	for k, v := range countMap {
		items = append(items, item{k, v})
	}

	// Trier par count décroissant
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			if items[i].count < items[j].count {
				items[i], items[j] = items[j], items[i]
			}
		}
	}

	// Prendre les N premiers
	result := make([]string, 0, n)
	for i := 0; i < n && i < len(items); i++ {
		result = append(result, items[i].key)
	}
	return strings.Join(result, ", ")
}

// getYearRange retourne la plage d'années la plus fréquente
func (s *RecommendationService) getYearRange(yearCount map[int]int) (int, int) {
	if len(yearCount) == 0 {
		return 2000, time.Now().Year()
	}

	minYear := 9999
	maxYear := 0
	for year := range yearCount {
		if year < minYear {
			minYear = year
		}
		if year > maxYear {
			maxYear = year
		}
	}
	return minYear, maxYear
}

// generateRecommendations génère des recommandations basées sur les préférences
func (s *RecommendationService) generateRecommendations(userID uuid.UUID, preferences *models.UserPreference, history []models.WatchHistory, limit int) []*models.Recommendation {
	// Obtenir toutes les vidéos disponibles
	var videos []models.Video
	query := s.db.Where("status = ? AND is_public = ?", models.VideoStatusReady, true)
	
	// Filtrer par note minimale
	if preferences.MinRating > 0 {
		query = query.Where("rating >= ?", preferences.MinRating)
	}
	
	// Filtrer par années préférées
	if preferences.PreferredYears != "" {
		// Parser la plage d'années (ex: "2020-2024")
		parts := strings.Split(preferences.PreferredYears, "-")
		if len(parts) == 2 {
			var minYear, maxYear int
			fmt.Sscanf(parts[0], "%d", &minYear)
			fmt.Sscanf(parts[1], "%d", &maxYear)
			if minYear > 0 && maxYear > 0 {
				query = query.Where("production_year >= ? AND production_year <= ?", minYear, maxYear)
			}
		}
	}
	
	// Exclure les vidéos déjà regardées
	if len(history) > 0 {
		watchedIDs := make([]uuid.UUID, len(history))
		for i, h := range history {
			watchedIDs[i] = h.VideoID
		}
		query = query.Where("id NOT IN ?", watchedIDs)
	}
	
	query.Limit(limit * 3).Find(&videos) // Récupérer 3x plus pour avoir du choix

	// Scorer chaque vidéo
	scored := make([]struct {
		video models.Video
		score float64
		reason string
	}, 0, len(videos))

	for _, video := range videos {
		score, reason := s.scoreVideo(video, preferences, history)
		scored = append(scored, struct {
			video models.Video
			score float64
			reason string
		}{video, score, reason})
	}

	// Trier par score décroissant
	for i := 0; i < len(scored)-1; i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[i].score < scored[j].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	// Créer les recommandations
	recommendations := make([]*models.Recommendation, 0, limit)
	for i := 0; i < limit && i < len(scored); i++ {
		rec := &models.Recommendation{
			ID:        uuid.New(),
			UserID:    userID,
			VideoID:   scored[i].video.ID,
			Score:     scored[i].score,
			Reason:    scored[i].reason,
			Category:  s.determineCategory(scored[i].video, preferences),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		recommendations = append(recommendations, rec)
	}

	return recommendations
}

// scoreVideo calcule un score de recommandation pour une vidéo
func (s *RecommendationService) scoreVideo(video models.Video, preferences *models.UserPreference, history []models.WatchHistory) (float64, string) {
	score := 50.0 // Score de base
	reasons := []string{}

	// Bonus pour les genres préférés
	if preferences.PreferredGenres != "" && video.Genre != "" {
		preferredGenres := strings.Split(strings.ToLower(preferences.PreferredGenres), ",")
		videoGenres := strings.Split(strings.ToLower(video.Genre), ",")
		matches := 0
		for _, pref := range preferredGenres {
			pref = strings.TrimSpace(pref)
			for _, vg := range videoGenres {
				if strings.Contains(strings.TrimSpace(vg), pref) {
					matches++
					break
				}
			}
		}
		if matches > 0 {
			score += float64(matches) * 10
			reasons = append(reasons, fmt.Sprintf("Genres similaires (%d)", matches))
		}
	}

	// Bonus pour les réalisateurs préférés
	if preferences.PreferredDirectors != "" && video.Director != "" {
		preferredDirectors := strings.Split(strings.ToLower(preferences.PreferredDirectors), ",")
		videoDirector := strings.ToLower(video.Director)
		for _, pref := range preferredDirectors {
			if strings.Contains(videoDirector, strings.TrimSpace(pref)) {
				score += 15
				reasons = append(reasons, "Réalisateur apprécié")
				break
			}
		}
	}

	// Bonus pour la note
	if video.Rating > 0 {
		score += video.Rating * 2
		reasons = append(reasons, fmt.Sprintf("Note: %.1f/10", video.Rating))
	}

	// Bonus pour les nouveautés (dernières 2 années)
	currentYear := time.Now().Year()
	if video.ProductionYear >= currentYear-2 {
		score += 10
		reasons = append(reasons, "Nouveauté")
	}

	// Bonus pour les tendances (nombre de vues)
	if video.Views > 100 {
		score += math.Log10(float64(video.Views)) * 5
		reasons = append(reasons, "Populaire")
	}

	// Limiter le score à 100
	if score > 100 {
		score = 100
	}

	reason := strings.Join(reasons, ", ")
	if reason == "" {
		reason = "Recommandation générale"
	}

	return score, reason
}

// determineCategory détermine la catégorie de recommandation
func (s *RecommendationService) determineCategory(video models.Video, preferences *models.UserPreference) string {
	currentYear := time.Now().Year()
	
	if video.ProductionYear >= currentYear-1 {
		return "new_release"
	}
	
	if video.Views > 1000 {
		return "trending"
	}
	
	if video.Director != "" && preferences.PreferredDirectors != "" {
		preferredDirectors := strings.Split(strings.ToLower(preferences.PreferredDirectors), ",")
		videoDirector := strings.ToLower(video.Director)
		for _, pref := range preferredDirectors {
			if strings.Contains(videoDirector, strings.TrimSpace(pref)) {
				return "similar"
			}
		}
	}
	
	return "general"
}

// getDefaultRecommendations retourne des recommandations par défaut (tendances, nouveautés)
func (s *RecommendationService) getDefaultRecommendations(userID uuid.UUID, limit int) ([]*models.Recommendation, error) {
	var videos []models.Video
	
	// Récupérer les vidéos les plus populaires et récentes
	s.db.Where("status = ? AND is_public = ?", models.VideoStatusReady, true).
		Order("views DESC, production_year DESC").
		Limit(limit).
		Find(&videos)

	recommendations := make([]*models.Recommendation, 0, len(videos))
	for _, video := range videos {
		rec := &models.Recommendation{
			ID:        uuid.New(),
			UserID:    userID,
			VideoID:   video.ID,
			Score:     70.0,
			Reason:    "Populaire et récent",
			Category:  "trending",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		recommendations = append(recommendations, rec)
	}

	return recommendations, nil
}

