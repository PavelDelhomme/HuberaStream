package ai

import (
	"fmt"
	"log"
	"strings"
	"time"

	"streammake/internal/metadata"
	"streammake/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DetectionService améliore la détection de films avec mémoire et apprentissage
type DetectionService struct {
	db *gorm.DB
}

// NewDetectionService crée un nouveau service de détection améliorée
func NewDetectionService(db *gorm.DB) *DetectionService {
	return &DetectionService{
		db: db,
	}
}

// ImproveSearchResults améliore les résultats de recherche en utilisant la mémoire
func (s *DetectionService) ImproveSearchResults(searchKey string, director string, year int, results []*metadata.MovieMetadata) []*metadata.MovieMetadata {
	if len(results) == 0 {
		return results
	}

	// Normaliser la clé de recherche
	normalizedKey := s.normalizeSearchKey(searchKey)

	// Chercher dans la mémoire
	var memories []models.MovieDetectionMemory
	s.db.Where("search_key = ?", normalizedKey).
		Or("search_key LIKE ?", normalizedKey+"%").
		Order("selection_count DESC, last_selected DESC").
		Limit(10).
		Find(&memories)

	// Si on a une mémoire correspondante, scorer les résultats
	if len(memories) > 0 {
		scoredResults := make([]struct {
			result *metadata.MovieMetadata
			score  float64
		}, len(results))

		for i, result := range results {
			score := s.scoreResult(result, memories, director, year)
			scoredResults[i] = struct {
				result *metadata.MovieMetadata
				score  float64
			}{result, score}
		}

		// Trier par score décroissant
		for i := 0; i < len(scoredResults)-1; i++ {
			for j := i + 1; j < len(scoredResults); j++ {
				if scoredResults[i].score < scoredResults[j].score {
					scoredResults[i], scoredResults[j] = scoredResults[j], scoredResults[i]
				}
			}
		}

		// Reconstruire la liste triée
		improvedResults := make([]*metadata.MovieMetadata, len(results))
		for i, sr := range scoredResults {
			improvedResults[i] = sr.result
		}
		return improvedResults
	}

	return results
}

// RememberSelection mémorise un choix utilisateur pour améliorer les futures recherches
func (s *DetectionService) RememberSelection(searchKey string, director string, year int, selected *metadata.MovieMetadata) error {
	if selected == nil {
		return fmt.Errorf("selected metadata is nil")
	}

	normalizedKey := s.normalizeSearchKey(searchKey)

	// Chercher une mémoire existante
	var memory models.MovieDetectionMemory
	err := s.db.Where("search_key = ? AND selected_imdb_id = ?", normalizedKey, selected.IMDBID).
		First(&memory).Error

	if err == gorm.ErrRecordNotFound {
		// Créer une nouvelle mémoire
		memory = models.MovieDetectionMemory{
			ID:              uuid.New(),
			SearchKey:       normalizedKey,
			Director:        director,
			Year:            year,
			SelectedIMDBID:  selected.IMDBID,
			SelectedTitle:   selected.Title,
			SelectedDirector: selected.Director,
			SelectedYear:    selected.ProductionYear,
			SelectionCount:  1,
			LastSelected:    time.Now(),
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}
		if err := s.db.Create(&memory).Error; err != nil {
			return err
		}
		log.Printf("[AI Detection] Nouvelle mémoire créée: %s -> %s", normalizedKey, selected.Title)
	} else if err != nil {
		return err
	} else {
		// Mettre à jour la mémoire existante
		memory.SelectionCount++
		memory.LastSelected = time.Now()
		memory.UpdatedAt = time.Now()
		s.db.Save(&memory)
		log.Printf("[AI Detection] Mémoire mise à jour: %s -> %s (count: %d)", normalizedKey, selected.Title, memory.SelectionCount)
	}

	// Apprendre des patterns
	s.learnPattern(searchKey, director, year, selected)

	return nil
}

// normalizeSearchKey normalise une clé de recherche
func (s *DetectionService) normalizeSearchKey(key string) string {
	key = strings.ToLower(strings.TrimSpace(key))
	// Supprimer les caractères spéciaux
	key = strings.ReplaceAll(key, ".", "")
	key = strings.ReplaceAll(key, "-", " ")
	key = strings.ReplaceAll(key, "_", " ")
	// Normaliser les espaces multiples
	words := strings.Fields(key)
	return strings.Join(words, " ")
}

// scoreResult calcule un score pour un résultat basé sur la mémoire
func (s *DetectionService) scoreResult(result *metadata.MovieMetadata, memories []models.MovieDetectionMemory, director string, year int) float64 {
	score := 0.0

	for _, memory := range memories {
		// Correspondance exacte IMDB ID
		if result.IMDBID != "" && memory.SelectedIMDBID == result.IMDBID {
			score += 100.0 * float64(memory.SelectionCount)
			continue
		}

		// Correspondance titre (similarité)
		titleSimilarity := s.calculateSimilarity(strings.ToLower(result.Title), strings.ToLower(memory.SelectedTitle))
		if titleSimilarity > 0.8 {
			score += 50.0 * titleSimilarity * float64(memory.SelectionCount)
		}

		// Correspondance réalisateur
		if director != "" && memory.Director != "" {
			directorSimilarity := s.calculateSimilarity(strings.ToLower(director), strings.ToLower(memory.Director))
			if directorSimilarity > 0.7 {
				score += 20.0 * directorSimilarity
			}
		}

		// Correspondance année
		if year > 0 && memory.Year > 0 {
			if abs(year-memory.Year) <= 2 {
				score += 10.0
			}
		}
	}

	return score
}

// calculateSimilarity calcule une similarité simple entre deux chaînes (0-1)
func (s *DetectionService) calculateSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}

	// Similarité basée sur les mots communs
	words1 := strings.Fields(s1)
	words2 := strings.Fields(s2)

	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}

	common := 0
	for _, w1 := range words1 {
		for _, w2 := range words2 {
			if w1 == w2 {
				common++
				break
			}
		}
	}

	maxLen := len(words1)
	if len(words2) > maxLen {
		maxLen = len(words2)
	}

	return float64(common) / float64(maxLen)
}

// learnPattern apprend un pattern pour améliorer les futures détections
func (s *DetectionService) learnPattern(searchKey string, director string, year int, selected *metadata.MovieMetadata) {
	// Pattern: director + year
	if director != "" && year > 0 {
		patternKey := fmt.Sprintf("%s_%d", strings.ToLower(director), year)
		s.updatePattern("director_year", patternKey, selected)
	}

	// Pattern: title similarity
	if selected.Title != "" {
		normalizedTitle := s.normalizeSearchKey(selected.Title)
		s.updatePattern("title_similarity", normalizedTitle, selected)
	}
}

// updatePattern met à jour ou crée un pattern d'apprentissage
func (s *DetectionService) updatePattern(patternType, patternKey string, selected *metadata.MovieMetadata) {
	var pattern models.LearningPattern
	err := s.db.Where("pattern_type = ? AND pattern_key = ?", patternType, patternKey).
		First(&pattern).Error

	if err == gorm.ErrRecordNotFound {
		// Créer un nouveau pattern
		pattern = models.LearningPattern{
			ID:              uuid.New(),
			PatternType:     patternType,
			PatternKey:      patternKey,
			ExpectedIMDBID:  selected.IMDBID,
			ExpectedTitle:   selected.Title,
			Confidence:      0.5,
			SuccessCount:    1,
			FailureCount:    0,
			LastUsed:        time.Now(),
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}
		s.db.Create(&pattern)
	} else if err == nil {
		// Mettre à jour le pattern (augmenter la confiance)
		pattern.SuccessCount++
		pattern.Confidence = float64(pattern.SuccessCount) / float64(pattern.SuccessCount+pattern.FailureCount)
		pattern.LastUsed = time.Now()
		pattern.UpdatedAt = time.Now()
		s.db.Save(&pattern)
	}
}

// abs retourne la valeur absolue d'un entier
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

