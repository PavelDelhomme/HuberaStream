package models

import (
	"time"

	"github.com/google/uuid"
)

// UserPreference stocke les préférences d'un utilisateur apprises par l'IA
type UserPreference struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	User      User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	
	// Préférences apprises
	PreferredGenres      string  `gorm:"type:text" json:"preferred_genres"`        // Genres préférés (séparés par virgules)
	PreferredDirectors   string  `gorm:"type:text" json:"preferred_directors"`      // Réalisateurs préférés
	PreferredActors       string  `gorm:"type:text" json:"preferred_actors"`        // Acteurs préférés
	PreferredYears        string  `gorm:"type:text" json:"preferred_years"`        // Périodes préférées (ex: "2020-2024")
	MinRating             float64 `gorm:"default:0" json:"min_rating"`              // Note minimale préférée
	PreferredQuality      string  `gorm:"type:varchar(20)" json:"preferred_quality"` // Qualité préférée (1080p, 720p, etc.)
	PreferredLanguage     string  `gorm:"type:varchar(10)" json:"preferred_language"` // Langue préférée
	
	// Statistiques d'apprentissage
	TotalWatched         int       `gorm:"default:0" json:"total_watched"`         // Nombre de vidéos regardées
	TotalLiked           int       `gorm:"default:0" json:"total_liked"`           // Nombre de vidéos aimées
	LastUpdated          time.Time `json:"last_updated"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// MovieDetectionMemory stocke la mémoire des détections de films pour améliorer les résultats futurs
type MovieDetectionMemory struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	
	// Clé de recherche (normalisée)
	SearchKey string `gorm:"type:varchar(255);not null;index" json:"search_key"` // Ex: "kill", "kill bill"
	
	// Contexte de la recherche
	Director  string `gorm:"type:varchar(255)" json:"director"`  // Réalisateur si connu
	Year      int    `json:"year"`                               // Année si connue
	
	// Résultat sélectionné (ce qui a été choisi par l'utilisateur)
	SelectedIMDBID    string    `gorm:"type:varchar(20);index" json:"selected_imdb_id"`
	SelectedTitle     string    `gorm:"type:varchar(255)" json:"selected_title"`
	SelectedDirector  string    `gorm:"type:varchar(255)" json:"selected_director"`
	SelectedYear      int       `json:"selected_year"`
	
	// Métriques d'apprentissage
	SelectionCount    int       `gorm:"default:1" json:"selection_count"`    // Nombre de fois sélectionné
	LastSelected      time.Time `json:"last_selected"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// Recommendation stocke les recommandations générées par l'IA
type Recommendation struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	User      User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	
	// Vidéo recommandée
	VideoID   uuid.UUID `gorm:"type:uuid;not null;index" json:"video_id"`
	Video     Video     `gorm:"foreignKey:VideoID" json:"video,omitempty"`
	
	// Score et raison
	Score     float64 `gorm:"not null" json:"score"`              // Score de recommandation (0-100)
	Reason    string  `gorm:"type:text" json:"reason"`            // Raison de la recommandation
	Category  string  `gorm:"type:varchar(50)" json:"category"`  // Type: "similar", "trending", "new_release", etc.
	
	// Statut
	Shown     bool      `gorm:"default:false" json:"shown"`      // A été montré à l'utilisateur
	Clicked   bool      `gorm:"default:false" json:"clicked"`     // L'utilisateur a cliqué
	Watched   bool      `gorm:"default:false" json:"watched"`    // L'utilisateur a regardé
	Liked     *bool     `gorm:"type:boolean" json:"liked"`       // L'utilisateur a aimé (null = pas encore évalué)
	
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SeriesTracking suit les séries pour l'auto-téléchargement
type SeriesTracking struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	User      User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	
	// Informations de la série
	Title         string `gorm:"type:varchar(255);not null" json:"title"`
	IMDBID        string `gorm:"type:varchar(20);index" json:"imdb_id"`
	TMDBID        int    `json:"tmdb_id"`
	PosterURL     string `gorm:"type:text" json:"poster_url"`
	
	// Suivi
	CurrentSeason int    `gorm:"default:1" json:"current_season"`      // Saison actuelle
	CurrentEpisode int   `gorm:"default:1" json:"current_episode"`    // Épisode actuel
	TotalSeasons  int    `json:"total_seasons"`                       // Nombre total de saisons
	TotalEpisodes int    `json:"total_episodes"`                       // Nombre total d'épisodes
	
	// Configuration auto-téléchargement
	AutoDownload  bool   `gorm:"default:true" json:"auto_download"`    // Auto-télécharger les nouveaux épisodes
	Quality       string `gorm:"type:varchar(20)" json:"quality"`      // Qualité préférée
	Language      string `gorm:"type:varchar(10)" json:"language"`    // Langue préférée
	
	// Dernière vérification
	LastChecked   *time.Time `json:"last_checked"`
	LastDownloaded *time.Time `json:"last_downloaded"`
	
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// SeriesEpisodeTracking suit les épisodes téléchargés/regardés
type SeriesEpisodeTracking struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	SeriesID  uuid.UUID `gorm:"type:uuid;not null;index" json:"series_id"`
	Series    SeriesTracking `gorm:"foreignKey:SeriesID" json:"series,omitempty"`
	
	Season    int    `gorm:"not null" json:"season"`
	Episode   int    `gorm:"not null" json:"episode"`
	Title     string `gorm:"type:varchar(255)" json:"title"`
	
	// Statut
	VideoID   *uuid.UUID `gorm:"type:uuid;index" json:"video_id"`    // ID de la vidéo si téléchargée
	Video     *Video     `gorm:"foreignKey:VideoID" json:"video,omitempty"`
	Downloaded bool      `gorm:"default:false" json:"downloaded"`
	Watched    bool      `gorm:"default:false" json:"watched"`
	
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// LearningPattern stocke les patterns appris pour améliorer les détections
type LearningPattern struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	
	// Pattern
	PatternType string `gorm:"type:varchar(50);not null;index" json:"pattern_type"` // "director_year", "title_similarity", etc.
	PatternKey  string `gorm:"type:varchar(255);not null;index" json:"pattern_key"` // Clé du pattern
	
	// Résultat attendu
	ExpectedIMDBID string `gorm:"type:varchar(20)" json:"expected_imdb_id"`
	ExpectedTitle  string `gorm:"type:varchar(255)" json:"expected_title"`
	
	// Confiance
	Confidence     float64   `gorm:"default:0.5" json:"confidence"`    // 0-1
	SuccessCount   int       `gorm:"default:0" json:"success_count"`   // Nombre de succès
	FailureCount   int       `gorm:"default:0" json:"failure_count"`   // Nombre d'échecs
	
	LastUsed       time.Time `json:"last_used"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

