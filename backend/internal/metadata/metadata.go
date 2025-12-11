package metadata

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// TMDB API configuration
const (
	TMDBBaseURL      = "https://api.themoviedb.org/3"
	TMDBImageBaseURL = "https://image.tmdb.org/t/p/w500"
)

type MetadataService struct {
	tmdbAPIKey string
	httpClient *http.Client
}

type TMDBMovie struct {
	ID               int     `json:"id"`
	Title            string  `json:"title"`
	OriginalTitle    string  `json:"original_title"`
	Overview         string  `json:"overview"`
	ReleaseDate      string  `json:"release_date"`
	PosterPath       string  `json:"poster_path"`
	BackdropPath     string  `json:"backdrop_path"`
	VoteAverage      float64 `json:"vote_average"`
	VoteCount        int     `json:"vote_count"`
	GenreIDs         []int   `json:"genre_ids"`
	Popularity       float64 `json:"popularity"`
	Adult            bool    `json:"adult"`
	Video            bool    `json:"video"`
	OriginalLanguage string  `json:"original_language"`
}

type TMDBMovieDetails struct {
	TMDBMovie
	Genres              []TMDBGenre   `json:"genres"`
	ProductionCompanies []TMDBCompany `json:"production_companies"`
	Credits             TMDBCredits   `json:"credits"`
	IMDBID              string        `json:"imdb_id"`
	Runtime             int           `json:"runtime"`
	Tagline             string        `json:"tagline"`
}

type TMDBGenre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type TMDBCompany struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type TMDBCredits struct {
	Cast []TMDBCast `json:"cast"`
	Crew []TMDBCrew `json:"crew"`
}

type TMDBCast struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Character string `json:"character"`
	Order     int    `json:"order"`
}

type TMDBCrew struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Job        string `json:"job"`
	Department string `json:"department"`
}

type TMDBSearchResponse struct {
	Page         int         `json:"page"`
	Results      []TMDBMovie `json:"results"`
	TotalPages   int         `json:"total_pages"`
	TotalResults int         `json:"total_results"`
}

type MovieMetadata struct {
	Title          string   `json:"title"`
	OriginalTitle  string   `json:"original_title"`
	Description    string   `json:"description"`
	Synopsis       string   `json:"synopsis"`
	Director       string   `json:"director"`
	Actors         []string `json:"actors"`
	ReleaseDate    string   `json:"release_date"`
	ProductionYear int      `json:"production_year"`
	Genre          []string `json:"genre"`
	Rating         float64  `json:"rating"`
	IMDBID         string   `json:"imdb_id"`
	TMDBID         int      `json:"tmdb_id"`
	PosterURL      string   `json:"poster_url"`
	BackdropURL    string   `json:"backdrop_url"`
	TrailerURL     string   `json:"trailer_url"` // URL de la bande annonce YouTube
	Runtime        int      `json:"runtime"`     // en minutes
	Tagline        string   `json:"tagline"`
	Writer         string   `json:"writer"`   // Scénariste
	Platform       string   `json:"platform"` // Plateforme (Amazon Prime Video, etc.)
}

func NewMetadataService(tmdbAPIKey string) *MetadataService {
	return &MetadataService{
		tmdbAPIKey: tmdbAPIKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ExtractTitleFromFilename extrait un titre potentiel depuis le nom de fichier
func ExtractTitleFromFilename(filename string) []string {
	// IMPORTANT: Enlever l'extension AVANT tout traitement
	ext := filepath.Ext(filename)
	if ext != "" {
		filename = strings.TrimSuffix(filename, ext)
	}

	// Remplacer les underscores et tirets par des espaces AVANT le nettoyage
	filename = strings.ReplaceAll(filename, "_", " ")
	filename = strings.ReplaceAll(filename, "-", " ")
	filename = strings.ReplaceAll(filename, ".", " ")

	// Nettoyer le nom (enlever résolution, codec, etc.)
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)\d{3,4}p`),                    // 1080p, 720p, etc.
		regexp.MustCompile(`(?i)x264|x265|h264|h265|hevc`),    // Codecs
		regexp.MustCompile(`(?i)bluray|dvdrip|webrip|web-dl`), // Sources
		regexp.MustCompile(`(?i)\[.*?\]`),                     // Tags entre crochets
		regexp.MustCompile(`(?i)\(.*?\)`),                     // Tags entre parenthèses
	}

	cleaned := filename
	for _, pattern := range patterns {
		cleaned = pattern.ReplaceAllString(cleaned, " ")
	}

	// Nettoyer les espaces multiples
	cleaned = regexp.MustCompile(`\s+`).ReplaceAllString(cleaned, " ")
	cleaned = strings.TrimSpace(cleaned)

	if cleaned == "" {
		return []string{}
	}

	// Générer plusieurs variantes intelligentes
	variants := []string{}
	seen := make(map[string]bool)

	addVariant := func(v string) {
		v = strings.TrimSpace(v)
		if v != "" && len(v) >= 2 && !seen[strings.ToLower(v)] {
			seen[strings.ToLower(v)] = true
			variants = append(variants, v)
		}
	}

	// Extraire les mots pour générer les variantes
	words := strings.Fields(cleaned)

	// 1. Title Case - première lettre de chaque mot en majuscule (ex: "Red One")
	// PRIORITÉ: Cette variante est la plus importante pour la recherche
	if len(words) > 0 {
		titleCase := ""
		for i, word := range words {
			if i > 0 {
				titleCase += " "
			}
			if len(word) > 0 {
				// Title case: première lettre majuscule, reste minuscule
				// Gérer les mots en majuscules (ex: "RED" -> "Red")
				wordLower := strings.ToLower(word)
				if len(wordLower) > 0 {
					firstChar := strings.ToUpper(string(wordLower[0]))
					rest := ""
					if len(wordLower) > 1 {
						rest = wordLower[1:]
					}
					titleCase += firstChar + rest
				}
			}
		}
		// TOUJOURS ajouter Title Case EN PREMIER (pour avoir la priorité)
		// C'est crucial car addVariant utilise seen[strings.ToLower(v)] pour éviter les doublons
		if titleCase != "" {
			// Forcer l'ajout même si cleaned a déjà été ajouté
			titleCaseLower := strings.ToLower(titleCase)
			if !seen[titleCaseLower] {
				seen[titleCaseLower] = true
				variants = append(variants, titleCase)
			}
		}
		// Ajouter aussi cleaned si différent (mais après titleCase)
		if titleCase != cleaned {
			addVariant(cleaned) // Ajouter la variante originale
		}
	} else {
		// Si pas de mots, ajouter cleaned tel quel
		addVariant(cleaned)
	}

	// 3. Sentence case - première lettre du premier mot en majuscule (ex: "Red one")
	if len(words) > 0 {
		sentenceCase := strings.ToUpper(string(words[0][0])) + strings.ToLower(words[0][1:])
		for i := 1; i < len(words); i++ {
			sentenceCase += " " + strings.ToLower(words[i])
		}
		addVariant(sentenceCase)
	}

	// 4. Tout en minuscules (ex: "red one")
	addVariant(strings.ToLower(cleaned))

	// 5. Tout en majuscules (ex: "RED ONE")
	addVariant(strings.ToUpper(cleaned))

	// 6. Variante avec mots collés détectés (camelCase) - ex: "RedOne" -> "Red One"
	// Détecter les mots collés avec majuscules
	camelCase := regexp.MustCompile(`([a-z])([A-Z])`)
	if camelCase.MatchString(cleaned) {
		separated := camelCase.ReplaceAllString(cleaned, "$1 $2")
		separated = strings.TrimSpace(separated)
		if separated != cleaned {
			addVariant(separated)
			// Ajouter aussi en Title Case
			words := strings.Fields(separated)
			if len(words) > 0 {
				titleCase := ""
				for i, word := range words {
					if i > 0 {
						titleCase += " "
					}
					if len(word) > 0 {
						titleCase += strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
					}
				}
				addVariant(titleCase)
			}
		}
	}

	return variants
}

// SearchMovies recherche des films sur TMDB avec recherche flexible
func (s *MetadataService) SearchMovies(query string) ([]TMDBMovie, error) {
	if s.tmdbAPIKey == "" {
		return nil, fmt.Errorf("TMDB API key not configured")
	}

	allResults := []TMDBMovie{}
	seenIDs := make(map[int]bool)

	// Recherche principale avec le query exact
	results, err := s.searchMoviesExact(query)
	if err == nil {
		for _, movie := range results {
			if !seenIDs[movie.ID] {
				seenIDs[movie.ID] = true
				allResults = append(allResults, movie)
			}
		}
	}

	// Si pas assez de résultats, essayer une recherche partielle
	if len(allResults) < 3 {
		// Extraire les mots clés principaux (mots de 3+ caractères)
		words := strings.Fields(query)
		keywords := []string{}
		for _, word := range words {
			if len(word) >= 3 {
				keywords = append(keywords, word)
			}
		}

		// Rechercher avec les mots clés principaux
		if len(keywords) > 0 {
			partialQuery := strings.Join(keywords, " ")
			if partialQuery != query {
				results, err := s.searchMoviesExact(partialQuery)
				if err == nil {
					for _, movie := range results {
						if !seenIDs[movie.ID] {
							seenIDs[movie.ID] = true
							allResults = append(allResults, movie)
						}
					}
				}
			}
		}

		// Si toujours pas assez, essayer avec seulement le premier mot significatif
		if len(allResults) < 3 && len(keywords) > 0 {
			firstKeyword := keywords[0]
			if len(firstKeyword) >= 3 {
				results, err := s.searchMoviesExact(firstKeyword)
				if err == nil {
					for _, movie := range results {
						if !seenIDs[movie.ID] {
							seenIDs[movie.ID] = true
							allResults = append(allResults, movie)
						}
					}
				}
			}
		}
	}

	log.Printf("[MetadataService] %d résultats trouvés au total pour '%s'", len(allResults), query)
	return allResults, nil
}

// searchMoviesExact effectue une recherche exacte sur TMDB
func (s *MetadataService) searchMoviesExact(query string) ([]TMDBMovie, error) {
	// Encoder la requête
	encodedQuery := url.QueryEscape(query)
	searchURL := fmt.Sprintf("%s/search/movie?api_key=%s&query=%s&language=fr-FR&page=1",
		TMDBBaseURL, s.tmdbAPIKey, encodedQuery)

	log.Printf("[MetadataService] Recherche TMDB exacte: %s", searchURL)

	resp, err := s.httpClient.Get(searchURL)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la requête TMDB: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("erreur TMDB (status %d): %s", resp.StatusCode, string(body))
	}

	var searchResp TMDBSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("erreur lors du décodage de la réponse TMDB: %w", err)
	}

	log.Printf("[MetadataService] %d résultats trouvés pour '%s'", len(searchResp.Results), query)
	return searchResp.Results, nil
}

// GetMovieDetails récupère les détails complets d'un film
func (s *MetadataService) GetMovieDetails(tmdbID int) (*TMDBMovieDetails, error) {
	if s.tmdbAPIKey == "" {
		return nil, fmt.Errorf("TMDB API key not configured")
	}

	detailsURL := fmt.Sprintf("%s/movie/%d?api_key=%s&language=fr-FR&append_to_response=credits",
		TMDBBaseURL, tmdbID, s.tmdbAPIKey)

	log.Printf("[MetadataService] Récupération détails TMDB: %s", detailsURL)

	resp, err := s.httpClient.Get(detailsURL)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la requête TMDB: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("erreur TMDB (status %d): %s", resp.StatusCode, string(body))
	}

	var details TMDBMovieDetails
	if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
		return nil, fmt.Errorf("erreur lors du décodage de la réponse TMDB: %w", err)
	}

	return &details, nil
}

// ConvertToMovieMetadata convertit les données TMDB en MovieMetadata
func (s *MetadataService) ConvertToMovieMetadata(details *TMDBMovieDetails) *MovieMetadata {
	// Extraire le réalisateur
	director := ""
	for _, crew := range details.Credits.Crew {
		if crew.Job == "Director" {
			director = crew.Name
			break
		}
	}

	// Extraire les acteurs principaux (top 5)
	actors := []string{}
	for i, cast := range details.Credits.Cast {
		if i >= 5 {
			break
		}
		actors = append(actors, cast.Name)
	}

	// Extraire les genres
	genres := []string{}
	for _, genre := range details.Genres {
		genres = append(genres, genre.Name)
	}

	// Extraire l'année depuis la date de sortie
	year := 0
	if details.ReleaseDate != "" && len(details.ReleaseDate) >= 4 {
		if y, err := time.Parse("2006-01-02", details.ReleaseDate); err == nil {
			year = y.Year()
		}
	}

	// Construire les URLs des images
	posterURL := ""
	if details.PosterPath != "" {
		posterURL = TMDBImageBaseURL + details.PosterPath
	}

	backdropURL := ""
	if details.BackdropPath != "" {
		backdropURL = TMDBImageBaseURL + details.BackdropPath
	}

	return &MovieMetadata{
		Title:          details.Title,
		OriginalTitle:  details.OriginalTitle,
		Description:    details.Overview,
		Synopsis:       details.Overview, // TMDB fournit déjà en français si language=fr-FR
		Director:       director,
		Actors:         actors,
		ReleaseDate:    details.ReleaseDate,
		ProductionYear: year,
		Genre:          genres,
		Rating:         details.VoteAverage,
		IMDBID:         details.IMDBID,
		TMDBID:         details.ID,
		PosterURL:      posterURL,
		BackdropURL:    backdropURL,
		Runtime:        details.Runtime,
		Tagline:        details.Tagline,
	}
}

// SearchMoviesFromFilename recherche des films basés sur le nom de fichier
func (s *MetadataService) SearchMoviesFromFilename(filename string) ([]*MovieMetadata, error) {
	// Extraire les variantes du titre
	titleVariants := ExtractTitleFromFilename(filename)

	log.Printf("[MetadataService] Variantes extraites pour '%s': %v", filename, titleVariants)

	allResults := []*MovieMetadata{}
	seenIDs := make(map[int]bool)

	// Rechercher pour chaque variante (priorité aux premières variantes)
	for i, variant := range titleVariants {
		if variant == "" || len(variant) < 2 {
			continue
		}

		log.Printf("[MetadataService] Recherche variante %d/%d: '%s'", i+1, len(titleVariants), variant)

		movies, err := s.SearchMovies(variant)
		if err != nil {
			log.Printf("[MetadataService] Erreur lors de la recherche pour '%s': %v", variant, err)
			continue
		}

		// Limiter à 5 résultats par variante pour éviter trop de requêtes
		maxPerVariant := 5
		if len(movies) > maxPerVariant {
			movies = movies[:maxPerVariant]
		}

		// Récupérer les détails pour chaque résultat
		for _, movie := range movies {
			if seenIDs[movie.ID] {
				continue
			}
			seenIDs[movie.ID] = true

			details, err := s.GetMovieDetails(movie.ID)
			if err != nil {
				log.Printf("[MetadataService] Erreur lors de la récupération des détails pour ID %d: %v", movie.ID, err)
				continue
			}

			metadata := s.ConvertToMovieMetadata(details)
			allResults = append(allResults, metadata)

			// Limiter le nombre total de résultats à 10 pour éviter trop de requêtes
			if len(allResults) >= 10 {
				log.Printf("[MetadataService] Limite de 10 résultats atteinte")
				return allResults, nil
			}
		}

		// Si on a déjà des résultats avec les premières variantes, on peut s'arrêter
		if len(allResults) >= 5 && i >= 2 {
			log.Printf("[MetadataService] Suffisamment de résultats trouvés avec les premières variantes")
			break
		}
	}

	log.Printf("[MetadataService] Total de %d résultats trouvés pour '%s'", len(allResults), filename)
	return allResults, nil
}
