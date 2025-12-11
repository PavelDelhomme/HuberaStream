package metadata

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// WebScraperService effectue des recherches gratuites sur internet sans API
type WebScraperService struct {
	httpClient *http.Client
	userAgent  string
}

// NewWebScraperService crée un nouveau service de scraping web
func NewWebScraperService() *WebScraperService {
	return &WebScraperService{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	}
}

// SearchMovieOnAlloCine recherche un film sur AlloCiné
func (s *WebScraperService) SearchMovieOnAlloCine(query string) ([]*MovieMetadata, error) {
	// Utiliser l'API de recherche AlloCiné ou une URL plus moderne
	searchURL := fmt.Sprintf("https://www.allocine.fr/recherche/?q=%s", url.QueryEscape(query))

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, err
	}
	// User-Agent plus moderne pour éviter les erreurs 410
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Referer", "https://www.allocine.fr/")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "same-origin")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la requête AlloCiné: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("erreur AlloCiné (status %d)", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erreur lors du parsing HTML: %w", err)
	}

	results := []*MovieMetadata{}
	seenLinks := make(map[string]bool)

	// STRATÉGIE 1: Chercher directement tous les liens vers /film/ (plus fiable)
	doc.Find("a[href*='/film/']").Each(func(i int, sel *goquery.Selection) {
		if len(results) >= 5 {
			return
		}

		link, _ := sel.Attr("href")
		if link == "" {
			return
		}

		// Normaliser le lien
		if !strings.HasPrefix(link, "http") {
			link = "https://www.allocine.fr" + link
		}

		// Éviter les doublons
		if seenLinks[link] {
			return
		}
		seenLinks[link] = true

		// Extraire le titre depuis le texte du lien ou depuis l'URL
		title := strings.TrimSpace(sel.Text())
		if title == "" {
			// Essayer de trouver le titre dans le parent
			title = strings.TrimSpace(sel.Parent().Text())
		}
		if title == "" {
			// Extraire depuis l'URL si possible
			parts := strings.Split(link, "/")
			if len(parts) > 0 {
				lastPart := parts[len(parts)-1]
				title = strings.ReplaceAll(lastPart, "-", " ")
				title = strings.ReplaceAll(title, "_", " ")
			}
		}

		if title == "" {
			return
		}

		// Récupérer les détails depuis la page du film (plus complet)
		details := s.getMovieDetailsFromAlloCine(link)
		if details != nil && details.Title != "" {
			// Utiliser les détails de la page si disponibles
			metadata := details
			metadata.Title = details.Title
			if metadata.Title == "" {
				metadata.Title = title
			}
			results = append(results, metadata)
		} else {
			// Sinon créer un résultat basique
			metadata := &MovieMetadata{
				Title:         title,
				OriginalTitle: title,
			}
			results = append(results, metadata)
		}
	})

	// STRATÉGIE 2: Si pas assez de résultats, chercher avec les sélecteurs de cartes
	if len(results) < 3 {
		cardSelectors := []string{".entity-card", ".card", "[class*='card']", ".search-result"}
		for _, cardSel := range cardSelectors {
			doc.Find(cardSel).Each(func(i int, sel *goquery.Selection) {
				if len(results) >= 5 {
					return
				}

				// Chercher un lien vers un film dans cette carte
				link, _ := sel.Find("a[href*='/film/']").Attr("href")
				if link == "" {
					return
				}

				if !strings.HasPrefix(link, "http") {
					link = "https://www.allocine.fr" + link
				}

				if seenLinks[link] {
					return
				}
				seenLinks[link] = true

				// Titre
				title := strings.TrimSpace(sel.Find("a[href*='/film/']").Text())
				if title == "" {
					title = strings.TrimSpace(sel.Find("h2, h3, h4").Text())
				}
				if title == "" {
					return
				}

				// Récupérer les détails
				details := s.getMovieDetailsFromAlloCine(link)
				if details != nil {
					results = append(results, details)
				} else {
					metadata := &MovieMetadata{
						Title:         title,
						OriginalTitle: title,
					}
					results = append(results, metadata)
				}
			})

			if len(results) >= 5 {
				break
			}
		}
	}

	return results, nil
}

// getMovieDetailsFromAlloCine récupère les détails d'un film depuis sa page AlloCiné
func (s *WebScraperService) getMovieDetailsFromAlloCine(movieURL string) *MovieMetadata {
	req, err := http.NewRequest("GET", movieURL, nil)
	if err != nil {
		log.Printf("[WebScraper] Erreur création requête pour %s: %v", movieURL, err)
		return nil
	}
	req.Header.Set("User-Agent", s.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.9")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("[WebScraper] Erreur requête pour %s: %v", movieURL, err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[WebScraper] Status code %d pour %s", resp.StatusCode, movieURL)
		return nil
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Printf("[WebScraper] Erreur parsing HTML pour %s: %v", movieURL, err)
		return nil
	}

	details := &MovieMetadata{}

	// Titre - essayer plusieurs sélecteurs
	titleSelectors := []string{
		"h1.title",
		".titlebar-title",
		"h1",
		"[class*='title']",
	}
	for _, sel := range titleSelectors {
		title := strings.TrimSpace(doc.Find(sel).First().Text())
		if title != "" {
			details.Title = title
			details.OriginalTitle = title
			break
		}
	}

	// Réalisateur - essayer plusieurs sélecteurs améliorés
	directorSelectors := []string{
		".meta-body-direction .light",
		".meta-body-direction a",
		".meta-body-direction",
		"[class*='director'] .light",
		"[class*='director'] a",
		"a[href*='/personne/fichepersonne'][href*='realisateur']",
		".casting-director a",
		".casting-director .light",
		".meta-body-direction .meta-link",
	}
	for _, sel := range directorSelectors {
		director := strings.TrimSpace(doc.Find(sel).First().Text())
		// Nettoyer le texte (enlever "Réalisateur", etc.)
		director = regexp.MustCompile(`(?i)\s*(réalisateur|director|de|par)\s*:?\s*`).ReplaceAllString(director, "")
		director = strings.TrimSpace(director)
		if director != "" && len(director) > 2 && !strings.Contains(strings.ToLower(director), "réalisateur") {
			details.Director = director
			log.Printf("[WebScraper] Réalisateur trouvé: %s", director)
			break
		}
	}

	// Scénariste/Writer - chercher dans les crédits
	writerSelectors := []string{
		".meta-body-direction + .meta-body",
		"[class*='writer']",
		"[class*='scénariste']",
		".meta-body-info .meta-link",
	}
	for _, sel := range writerSelectors {
		writerText := strings.TrimSpace(doc.Find(sel).Text())
		// Chercher "Scénario" ou "Par" suivi d'un nom
		writerMatch := regexp.MustCompile(`(?i)(?:scénario|par|écrit par)\s*:?\s*([A-Z][a-z]+(?:\s+[A-Z][a-z]+)*)`)
		if matches := writerMatch.FindStringSubmatch(writerText); len(matches) > 1 {
			details.Writer = strings.TrimSpace(matches[1])
			log.Printf("[WebScraper] Scénariste trouvé: %s", details.Writer)
			break
		}
	}

	// Acteurs - essayer plusieurs sélecteurs améliorés
	actors := []string{}
	actorSelectors := []string{
		".meta-body-actor .light",
		".meta-body-actor a",
		".casting-actor .light",
		".casting-actor a",
		"[class*='actor'] .light",
		"[class*='actor'] a",
		"a[href*='/personne/fichepersonne'][href*='acteur']",
		".entity-card-actor .light",
		".entity-card-actor a",
	}
	for _, sel := range actorSelectors {
		doc.Find(sel).Each(func(i int, s *goquery.Selection) {
			if i < 10 { // Augmenter à 10 acteurs
				actor := strings.TrimSpace(s.Text())
				// Nettoyer le texte
				actor = regexp.MustCompile(`(?i)\s*(acteur|actor|voir plus|voir|...)\s*`).ReplaceAllString(actor, "")
				actor = strings.TrimSpace(actor)
				if actor != "" && len(actor) > 2 && !strings.Contains(strings.ToLower(actor), "acteur") {
					// Éviter les doublons
					found := false
					for _, existing := range actors {
						if strings.EqualFold(existing, actor) {
							found = true
							break
						}
					}
					if !found {
						actors = append(actors, actor)
					}
				}
			}
		})
		if len(actors) >= 5 { // Arrêter si on a assez d'acteurs
			break
		}
	}
	details.Actors = actors
	if len(actors) > 0 {
		log.Printf("[WebScraper] Acteurs trouvés: %v", actors)
	}

	// Genres - essayer plusieurs sélecteurs améliorés
	genres := []string{}
	genreSelectors := []string{
		".meta-body-info .genre",
		".meta-body-info a[href*='/genre/']",
		"[class*='genre'] a",
		"a[href*='/genre/']",
		".meta-body-info span",
	}
	for _, sel := range genreSelectors {
		doc.Find(sel).Each(func(i int, s *goquery.Selection) {
			genre := strings.TrimSpace(s.Text())
			if genre != "" && !regexp.MustCompile(`^\d{4}$`).MatchString(genre) && genre != "Genre" {
				// Éviter les doublons
				found := false
				for _, existing := range genres {
					if strings.EqualFold(existing, genre) {
						found = true
						break
					}
				}
				if !found {
					genres = append(genres, genre)
				}
			}
		})
		if len(genres) > 0 {
			break
		}
	}
	details.Genre = genres
	if len(genres) > 0 {
		log.Printf("[WebScraper] Genres trouvés: %v", genres)
	}

	// Année - essayer plusieurs sélecteurs
	yearSelectors := []string{
		".meta-body-info .date",
		".meta-body-info",
		"[class*='year']",
		".release-date",
	}
	for _, sel := range yearSelectors {
		yearText := doc.Find(sel).Text()
		year := extractYear(yearText)
		if year > 1900 && year < 2100 {
			details.ProductionYear = year
			log.Printf("[WebScraper] Année trouvée: %d", year)
			break
		}
	}

	// Note - essayer plusieurs sélecteurs
	ratingSelectors := []string{
		".rating-item .stareval-note",
		".rating-item",
		"[class*='rating'] .note",
		"[class*='rating']",
		".note",
	}
	for _, sel := range ratingSelectors {
		ratingText := doc.Find(sel).Text()
		rating := extractRating(ratingText)
		if rating > 0 {
			details.Rating = rating
			log.Printf("[WebScraper] Note trouvée: %.1f", rating)
			break
		}
	}

	// Synopsis - essayer plusieurs sélecteurs améliorés
	synopsisSelectors := []string{
		".content-txt .content-txt",
		".content-txt",
		"[class*='synopsis'] .content-txt",
		"[class*='synopsis']",
		"[class*='description']",
		".synopsis",
		".overview",
		"[data-testid='synopsis']",
		".synopsis-txt",
		".synopsis-text",
	}
	for _, sel := range synopsisSelectors {
		synopsis := strings.TrimSpace(doc.Find(sel).First().Text())
		// Nettoyer le synopsis
		synopsis = regexp.MustCompile(`\s+`).ReplaceAllString(synopsis, " ")
		// Enlever les mentions "Lire la suite", etc.
		synopsis = regexp.MustCompile(`(?i)\s*(lire la suite|voir plus|...)\s*$`).ReplaceAllString(synopsis, "")
		if synopsis != "" && len(synopsis) > 20 {
			if len(synopsis) > 1000 {
				synopsis = synopsis[:1000] + "..."
			}
			details.Synopsis = synopsis
			details.Description = synopsis
			log.Printf("[WebScraper] Synopsis trouvé (%d caractères)", len(synopsis))
			break
		}
	}

	// Durée/Runtime - chercher "2h 03min" ou "123 min"
	runtimeText := doc.Find(".meta-body-info, .meta-body").Text()
	runtimeMatch := regexp.MustCompile(`(\d+)\s*h\s*(\d+)\s*min`)
	if matches := runtimeMatch.FindStringSubmatch(runtimeText); len(matches) >= 3 {
		var hours, minutes int
		fmt.Sscanf(matches[1], "%d", &hours)
		fmt.Sscanf(matches[2], "%d", &minutes)
		details.Runtime = hours*60 + minutes
		log.Printf("[WebScraper] Durée trouvée: %d minutes (%dh %dmin)", details.Runtime, hours, minutes)
	} else {
		// Essayer format "123 min"
		runtimeMatch2 := regexp.MustCompile(`(\d+)\s*min`)
		if matches := runtimeMatch2.FindStringSubmatch(runtimeText); len(matches) >= 2 {
			fmt.Sscanf(matches[1], "%d", &details.Runtime)
			log.Printf("[WebScraper] Durée trouvée: %d minutes", details.Runtime)
		}
	}

	// Plateforme - chercher "Amazon Prime Video", "Netflix", etc.
	platformText := doc.Text()
	platformMatch := regexp.MustCompile(`(?i)(Amazon\s+Prime\s+Video|Netflix|Disney\+|HBO|Hulu|Apple\s+TV\+)`)
	if matches := platformMatch.FindStringSubmatch(platformText); len(matches) > 1 {
		details.Platform = matches[1]
		log.Printf("[WebScraper] Plateforme trouvée: %s", details.Platform)
	}

	// Poster - essayer plusieurs sélecteurs améliorés
	posterSelectors := []string{
		".thumbnail-img img",
		"[class*='poster'] img",
		"[class*='thumbnail'] img",
		"img[src*='media']",
		"img[src*='allocine']",
		".entity-poster img",
		"img[data-src*='media']",
	}
	for _, sel := range posterSelectors {
		posterURL, _ := doc.Find(sel).First().Attr("src")
		if posterURL == "" {
			posterURL, _ = doc.Find(sel).First().Attr("data-src")
		}
		if posterURL == "" {
			posterURL, _ = doc.Find(sel).First().Attr("data-original")
		}
		if posterURL != "" {
			// Normaliser l'URL
			if !strings.HasPrefix(posterURL, "http") {
				posterURL = "https://www.allocine.fr" + posterURL
			}
			details.PosterURL = posterURL
			log.Printf("[WebScraper] Poster trouvé: %s", posterURL)
			break
		}
	}

	// Chercher la bande annonce YouTube
	if details.Title != "" {
		trailerURL := s.searchYouTubeTrailer(details.Title, details.ProductionYear)
		if trailerURL != "" {
			details.TrailerURL = trailerURL
			log.Printf("[WebScraper] Bande annonce YouTube trouvée: %s", trailerURL)
		}
	}

	log.Printf("[WebScraper] ✅ Détails complets extraits pour: %s", details.Title)
	return details
}

// searchYouTubeTrailer recherche la bande annonce sur YouTube
func (s *WebScraperService) searchYouTubeTrailer(movieTitle string, year int) string {
	// Construire la requête de recherche
	searchQuery := fmt.Sprintf("%s bande annonce trailer %d", movieTitle, year)
	if year == 0 {
		searchQuery = fmt.Sprintf("%s bande annonce trailer", movieTitle)
	}

	searchURL := fmt.Sprintf("https://www.youtube.com/results?search_query=%s", url.QueryEscape(searchQuery))

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", s.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.9,en-US;q=0.8,en;q=0.7")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	// Chercher l'ID de la vidéo YouTube dans le HTML
	// YouTube utilise des données JSON embarquées dans le HTML
	re := regexp.MustCompile(`"videoId":"([^"]{11})"`)
	matches := re.FindStringSubmatch(string(body))
	if len(matches) > 1 {
		videoID := matches[1]
		return fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)
	}

	// Fallback: chercher dans les liens
	re2 := regexp.MustCompile(`/watch\?v=([^"&]{11})`)
	matches2 := re2.FindStringSubmatch(string(body))
	if len(matches2) > 1 {
		videoID := matches2[1]
		return fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)
	}

	return ""
}

// SearchMovieOnIMDB recherche un film sur IMDb (sans API)
func (s *WebScraperService) SearchMovieOnIMDB(query string) ([]*MovieMetadata, error) {
	searchURL := fmt.Sprintf("https://www.imdb.com/find/?q=%s&s=tt&ttype=ft", url.QueryEscape(query))

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", s.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la requête IMDb: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("erreur IMDb (status %d)", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erreur lors du parsing HTML: %w", err)
	}

	results := []*MovieMetadata{}

	// Chercher les résultats - essayer plusieurs sélecteurs
	selectors := []string{
		".ipc-metadata-list-summary-item",
		".find-result",
		"[data-testid='find-results-section-title'] .ipc-metadata-list-summary-item",
		".findSection .findResult",
	}

	found := false
	for _, selector := range selectors {
		doc.Find(selector).Each(func(i int, sel *goquery.Selection) {
			if i >= 5 { // Limiter à 5 résultats
				return
			}

			titleLink := sel.Find("a[href*='/title/']").First()
			if titleLink.Length() == 0 {
				// Essayer d'autres sélecteurs pour le titre
				titleLink = sel.Find("a").First()
			}
			title := strings.TrimSpace(titleLink.Text())
			if title == "" {
				return
			}
			found = true

			// Lien IMDb
			imdbLink, _ := titleLink.Attr("href")
			imdbID := extractIMDBID(imdbLink)

			// Année
			yearText := strings.TrimSpace(sel.Find(".ipc-metadata-list-summary-item__li").Text())
			year := extractYear(yearText)

			// Note (si disponible)
			ratingText := strings.TrimSpace(sel.Find(".ipc-rating-star").Text())
			rating := extractRating(ratingText)

			metadata := &MovieMetadata{
				Title:          title,
				OriginalTitle:  title,
				ProductionYear: year,
				Rating:         rating,
				IMDBID:         imdbID,
			}

			// Si on a un ID IMDb, récupérer plus de détails
			if imdbID != "" {
				details := s.getMovieDetailsFromIMDB(imdbID)
				if details != nil {
					metadata.Director = details.Director
					metadata.Actors = details.Actors
					metadata.Genre = details.Genre
					metadata.Synopsis = details.Synopsis
					metadata.PosterURL = details.PosterURL
				}
			}

			results = append(results, metadata)
		})

		if found && len(results) > 0 {
			break
		}
	}

	// Si aucun résultat avec les sélecteurs standards, essayer une recherche plus large
	if len(results) == 0 {
		doc.Find("a[href*='/title/tt']").Each(func(i int, sel *goquery.Selection) {
			if i >= 5 {
				return
			}
			title := strings.TrimSpace(sel.Text())
			if title != "" && len(title) > 2 {
				imdbLink, _ := sel.Attr("href")
				imdbID := extractIMDBID(imdbLink)
				metadata := &MovieMetadata{
					Title:         title,
					OriginalTitle: title,
					IMDBID:        imdbID,
				}
				results = append(results, metadata)
			}
		})
	}

	return results, nil
}

// getMovieDetailsFromIMDB récupère les détails d'un film depuis IMDb
func (s *WebScraperService) getMovieDetailsFromIMDB(imdbID string) *MovieMetadata {
	movieURL := fmt.Sprintf("https://www.imdb.com/title/%s/", imdbID)

	req, err := http.NewRequest("GET", movieURL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", s.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil
	}

	details := &MovieMetadata{}

	// Synopsis
	synopsis := strings.TrimSpace(doc.Find("span[data-testid='plot-xl']").Text())
	if synopsis == "" {
		synopsis = strings.TrimSpace(doc.Find("span[data-testid='plot-l']").Text())
	}
	details.Synopsis = synopsis

	// Réalisateur
	doc.Find("a[href*='/name/']").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		parent := s.Parent().Text()
		if strings.Contains(parent, "Director") && details.Director == "" {
			details.Director = text
		}
	})

	// Acteurs
	actors := []string{}
	doc.Find("a[data-testid='title-cast-item__actor']").Each(func(i int, s *goquery.Selection) {
		if i < 5 {
			actor := strings.TrimSpace(s.Text())
			if actor != "" {
				actors = append(actors, actor)
			}
		}
	})
	details.Actors = actors

	// Genres
	genres := []string{}
	doc.Find("a[href*='/genre/']").Each(func(i int, s *goquery.Selection) {
		genre := strings.TrimSpace(s.Text())
		if genre != "" {
			genres = append(genres, genre)
		}
	})
	details.Genre = genres

	// Poster
	posterURL, _ := doc.Find("img[data-testid='hero-image']").Attr("src")
	details.PosterURL = posterURL

	return details
}

// SearchMoviesFromFilenameWeb recherche des films sur internet (sans API)
func (s *WebScraperService) SearchMoviesFromFilenameWeb(filename string) ([]*MovieMetadata, error) {
	log.Printf("[WebScraper] ⚡ DÉBUT recherche pour '%s'", filename)

	titleVariants := ExtractTitleFromFilename(filename)

	log.Printf("[WebScraper] Variantes générées pour '%s': %v", filename, titleVariants)

	if len(titleVariants) == 0 {
		log.Printf("[WebScraper] Aucune variante générée pour '%s'", filename)
		return []*MovieMetadata{}, nil
	}

	allResults := []*MovieMetadata{}
	seenTitles := make(map[string]bool)

	// PRIORITÉ: Utiliser la première variante (Title Case, ex: "Red One") pour recherche Google
	// Google est plus flexible et trouvera plus de résultats
	// Essayer aussi "Red One film 2024" pour être plus spécifique
	if len(titleVariants) > 0 {
		// Prendre la première variante (Title Case) comme meilleure
		bestVariant := titleVariants[0]
		log.Printf("[WebScraper] 🔍 Recherche Google pour variante principale: '%s'", bestVariant)
		googleResults, err := s.SearchMovieOnGoogle(bestVariant)
		if err != nil {
			log.Printf("[WebScraper] Erreur Google pour '%s': %v", bestVariant, err)
		} else {
			log.Printf("[WebScraper] Google a trouvé %d résultat(s) pour '%s'", len(googleResults), bestVariant)
			for _, result := range googleResults {
				key := strings.ToLower(result.Title)
				if !seenTitles[key] {
					seenTitles[key] = true
					allResults = append(allResults, result)
				}
			}
		}

		// Si pas de résultats, essayer avec "film 2024" pour être plus spécifique
		if len(allResults) == 0 && bestVariant != "" {
			enhancedQuery := fmt.Sprintf("%s film 2024", bestVariant)
			log.Printf("[WebScraper] 🔍 Recherche Google améliorée: '%s'", enhancedQuery)
			googleResults, err := s.SearchMovieOnGoogle(enhancedQuery)
			if err != nil {
				log.Printf("[WebScraper] Erreur Google pour '%s': %v", enhancedQuery, err)
			} else {
				log.Printf("[WebScraper] Google amélioré a trouvé %d résultat(s) pour '%s'", len(googleResults), enhancedQuery)
				for _, result := range googleResults {
					key := strings.ToLower(result.Title)
					if !seenTitles[key] {
						seenTitles[key] = true
						allResults = append(allResults, result)
					}
				}
			}
		}
	}

	// Rechercher sur AlloCiné ET IMDb pour chaque variante (recherche plus large)
	for _, variant := range titleVariants {
		if variant == "" || len(variant) < 2 {
			continue
		}

		// Rechercher sur AlloCiné
		log.Printf("[WebScraper] Recherche AlloCiné: '%s'", variant)
		results, err := s.SearchMovieOnAlloCine(variant)
		if err != nil {
			log.Printf("[WebScraper] Erreur AlloCiné pour '%s': %v", variant, err)
		} else {
			log.Printf("[WebScraper] AlloCiné a trouvé %d résultat(s) pour '%s'", len(results), variant)
			for _, result := range results {
				key := strings.ToLower(result.Title)
				if !seenTitles[key] {
					seenTitles[key] = true
					allResults = append(allResults, result)
				}
			}
		}

		// Rechercher aussi sur IMDb (même variante)
		log.Printf("[WebScraper] Recherche IMDb: '%s'", variant)
		results, err = s.SearchMovieOnIMDB(variant)
		if err != nil {
			log.Printf("[WebScraper] Erreur IMDb pour '%s': %v", variant, err)
		} else {
			log.Printf("[WebScraper] IMDb a trouvé %d résultat(s) pour '%s'", len(results), variant)
			for _, result := range results {
				key := strings.ToLower(result.Title)
				if !seenTitles[key] {
					seenTitles[key] = true
					allResults = append(allResults, result)
				}
			}
		}

		// Limiter à 15 résultats maximum pour éviter trop de données
		if len(allResults) >= 15 {
			break
		}
	}

	log.Printf("[WebScraper] Total de %d résultat(s) unique(s) trouvé(s) pour '%s'", len(allResults), filename)

	// Limiter à 10 résultats maximum
	if len(allResults) > 10 {
		allResults = allResults[:10]
	}

	log.Printf("[WebScraper] ✅ FIN recherche - Total de %d résultat(s) unique(s) trouvé(s) pour '%s'", len(allResults), filename)

	if len(allResults) == 0 {
		log.Printf("[WebScraper] ⚠️  AUCUN résultat trouvé pour '%s' - vérifier les sélecteurs CSS", filename)
	}

	return allResults, nil
}

// SearchMovieOnGoogle recherche un film via Google (recherche large)
func (s *WebScraperService) SearchMovieOnGoogle(query string) ([]*MovieMetadata, error) {
	// Recherche Google avec "film" et "2024" pour améliorer les résultats (Red One est sorti en 2024)
	searchQuery := fmt.Sprintf("%s film 2024", query)
	searchURL := fmt.Sprintf("https://www.google.com/search?q=%s", url.QueryEscape(searchQuery))

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", s.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.9,en-US;q=0.8,en;q=0.7")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la requête Google: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("erreur Google (status %d)", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erreur lors du parsing HTML: %w", err)
	}

	results := []*MovieMetadata{}
	seenTitles := make(map[string]bool)

	// Chercher dans les snippets enrichis de Google (Knowledge Graph)
	doc.Find("div[data-attrid], .kp-blk, .kp-wholepage").Each(func(i int, sel *goquery.Selection) {
		if i >= 3 {
			return
		}

		// Titre depuis le Knowledge Graph
		title := strings.TrimSpace(sel.Find("h2, h3, span[data-attrid='title']").First().Text())
		if title == "" {
			title = strings.TrimSpace(sel.Find(".r").First().Text())
		}

		if title == "" || len(title) < 2 {
			return
		}

		// Nettoyer le titre
		title = regexp.MustCompile(`(?i)\s*(film|movie|vidéo|video|2024)\s*$`).ReplaceAllString(title, "")
		title = strings.TrimSpace(title)

		if title == "" {
			return
		}

		key := strings.ToLower(title)
		if seenTitles[key] {
			return
		}
		seenTitles[key] = true

		metadata := &MovieMetadata{
			Title:         title,
			OriginalTitle: title,
		}

		// Description/synopsis depuis le Knowledge Graph
		description := strings.TrimSpace(sel.Find("span[data-attrid='description'], .kno-rdesc, .kno-fv").First().Text())
		if description == "" {
			description = strings.TrimSpace(sel.Find(".VwiC3b, .s, .st").First().Text())
		}
		if len(description) > 1000 {
			description = description[:1000] + "..."
		}
		metadata.Description = description
		metadata.Synopsis = description

		// Extraire toutes les infos du texte
		allText := sel.Text()

		// Année
		year := extractYear(allText)
		if year > 1900 {
			metadata.ProductionYear = year
		}

		// Durée (ex: "2h 03min")
		runtimeMatch := regexp.MustCompile(`(\d+)\s*h\s*(\d+)\s*min`)
		if matches := runtimeMatch.FindStringSubmatch(allText); len(matches) >= 3 {
			var hours, minutes int
			fmt.Sscanf(matches[1], "%d", &hours)
			fmt.Sscanf(matches[2], "%d", &minutes)
			metadata.Runtime = hours*60 + minutes
		}

		// Genres (chercher "Action, Aventure, Comédie")
		genreMatch := regexp.MustCompile(`(?i)(Action|Aventure|Comédie|Comedy|Famille|Family|Drame|Drama|Thriller|Horreur|Horror|Science-Fiction|Sci-Fi)[,\s]*([A-Z][a-z]+)?[,\s]*([A-Z][a-z]+)?`)
		if matches := genreMatch.FindAllString(allText, -1); len(matches) > 0 {
			genres := []string{}
			for _, match := range matches {
				genre := strings.TrimSpace(match)
				if genre != "" && len(genre) > 2 {
					genres = append(genres, genre)
				}
			}
			if len(genres) > 0 {
				metadata.Genre = genres
			}
		}

		// Réalisateur (chercher "De Jake Kasdan" ou "Par Jake Kasdan")
		directorMatch := regexp.MustCompile(`(?i)(?:De|Par|Réalisé par|Directed by)\s+([A-Z][a-z]+\s+[A-Z][a-z]+)`)
		if matches := directorMatch.FindStringSubmatch(allText); len(matches) > 1 {
			metadata.Director = strings.TrimSpace(matches[1])
		}

		// Acteurs (chercher "avec Dwayne Johnson, Chris Evans")
		actorMatch := regexp.MustCompile(`(?i)(?:avec|with|Starring)\s+([A-Z][a-z]+\s+[A-Z][a-z]+(?:\s*,\s*[A-Z][a-z]+\s+[A-Z][a-z]+)*)`)
		if matches := actorMatch.FindStringSubmatch(allText); len(matches) > 1 {
			actorsStr := matches[1]
			actors := strings.Split(actorsStr, ",")
			metadata.Actors = []string{}
			for _, actor := range actors {
				actor = strings.TrimSpace(actor)
				if actor != "" {
					metadata.Actors = append(metadata.Actors, actor)
				}
			}
		}

		// Plateforme (Amazon Prime Video, Netflix, etc.)
		platformMatch := regexp.MustCompile(`(?i)(Amazon\s+Prime\s+Video|Netflix|Disney\+|HBO|Hulu|Apple\s+TV\+)`)
		if matches := platformMatch.FindStringSubmatch(allText); len(matches) > 1 {
			metadata.Platform = matches[1]
		}

		// Note
		rating := extractRating(allText)
		if rating > 0 {
			metadata.Rating = rating
		}

		results = append(results, metadata)
	})

	// Si pas assez de résultats, chercher dans les résultats de recherche normaux
	if len(results) < 3 {
		doc.Find("div[data-ved], .g, .tF2Cxc").Each(func(i int, sel *goquery.Selection) {
			if len(results) >= 5 {
				return
			}

			title := strings.TrimSpace(sel.Find("h3, .LC20lb, .DKV0Md").First().Text())
			if title == "" {
				return
			}

			title = regexp.MustCompile(`(?i)\s*(film|movie|vidéo|video)\s*$`).ReplaceAllString(title, "")
			title = strings.TrimSpace(title)

			if title == "" || len(title) < 2 {
				return
			}

			key := strings.ToLower(title)
			if seenTitles[key] {
				return
			}
			seenTitles[key] = true

			description := strings.TrimSpace(sel.Find(".VwiC3b, .s, .st").First().Text())
			if len(description) > 500 {
				description = description[:500] + "..."
			}

			yearText := sel.Text()
			year := extractYear(yearText)
			rating := extractRating(yearText)

			metadata := &MovieMetadata{
				Title:          title,
				OriginalTitle:  title,
				Description:    description,
				Synopsis:       description,
				ProductionYear: year,
				Rating:         rating,
			}

			results = append(results, metadata)
		})
	}

	// Si pas assez de résultats, essayer une recherche textuelle plus large
	if len(results) < 3 {
		// Chercher dans les snippets enrichis de Google
		doc.Find("div[data-attrid], .kp-blk").Each(func(i int, sel *goquery.Selection) {
			if len(results) >= 5 {
				return
			}

			title := strings.TrimSpace(sel.Find("h2, h3, span[data-attrid='title']").First().Text())
			if title == "" {
				return
			}

			key := strings.ToLower(title)
			if seenTitles[key] {
				return
			}
			seenTitles[key] = true

			description := strings.TrimSpace(sel.Find("span[data-attrid='description'], .kno-rdesc").First().Text())
			if len(description) > 500 {
				description = description[:500] + "..."
			}

			year := extractYear(sel.Text())
			rating := extractRating(sel.Text())

			metadata := &MovieMetadata{
				Title:          title,
				OriginalTitle:  title,
				Description:    description,
				Synopsis:       description,
				ProductionYear: year,
				Rating:         rating,
			}

			results = append(results, metadata)
		})
	}

	log.Printf("[WebScraper] Google a trouvé %d résultat(s) pour '%s'", len(results), query)
	return results, nil
}

// Fonctions utilitaires
func extractYear(text string) int {
	re := regexp.MustCompile(`\b(19|20)\d{2}\b`)
	matches := re.FindString(text)
	if matches != "" {
		var year int
		fmt.Sscanf(matches, "%d", &year)
		return year
	}
	return 0
}

func extractRating(text string) float64 {
	re := regexp.MustCompile(`(\d+[,.]?\d*)\s*/?\s*10`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		var rating float64
		fmt.Sscanf(strings.Replace(matches[1], ",", ".", -1), "%f", &rating)
		return rating
	}
	return 0
}

func extractIMDBID(link string) string {
	re := regexp.MustCompile(`/title/(tt\d+)/`)
	matches := re.FindStringSubmatch(link)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}
