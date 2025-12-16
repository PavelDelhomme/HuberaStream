package metadata

import (
	"encoding/json"
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
	log.Printf("[WebScraper] 🔍 Recherche IMDb pour: '%s'", query)
	searchURL := fmt.Sprintf("https://www.imdb.com/find/?q=%s&s=tt&ttype=ft", url.QueryEscape(query))
	log.Printf("[WebScraper] URL de recherche IMDb: %s", searchURL)

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		log.Printf("[WebScraper] ❌ Erreur création requête IMDb: %v", err)
		return nil, err
	}
	req.Header.Set("User-Agent", s.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.9,en-US;q=0.8,en;q=0.7")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("[WebScraper] ❌ Erreur requête IMDb: %v", err)
		return nil, fmt.Errorf("erreur lors de la requête IMDb: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[WebScraper] ❌ Erreur IMDb (status %d)", resp.StatusCode)
		return nil, fmt.Errorf("erreur IMDb (status %d)", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Printf("[WebScraper] ❌ Erreur parsing HTML IMDb: %v", err)
		return nil, fmt.Errorf("erreur lors du parsing HTML: %w", err)
	}

	results := []*MovieMetadata{}

	// Chercher les résultats - essayer plusieurs sélecteurs
	selectors := []string{
		".ipc-metadata-list-summary-item",
		".find-result",
		"[data-testid='find-results-section-title'] .ipc-metadata-list-summary-item",
		".findSection .findResult",
		"li.ipc-metadata-list-summary-item",
		".ipc-metadata-list-summary-item__tc",
	}

	found := false
	for idx, selector := range selectors {
		log.Printf("[WebScraper] Essai sélecteur IMDb #%d: %s", idx+1, selector)
		count := 0
		selCount := doc.Find(selector).Length()
		log.Printf("[WebScraper] Sélecteur '%s' a trouvé %d élément(s)", selector, selCount)

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
				log.Printf("[WebScraper] ⚠️  Titre vide pour élément #%d du sélecteur '%s'", i, selector)
				return
			}
			found = true
			count++

			// Lien IMDb
			imdbLink, exists := titleLink.Attr("href")
			if !exists {
				log.Printf("[WebScraper] ⚠️  Pas de lien href pour '%s'", title)
				return
			}
			log.Printf("[WebScraper] Lien IMDb trouvé: %s", imdbLink)
			imdbID := extractIMDBID(imdbLink)
			log.Printf("[WebScraper] ✅ Résultat IMDb trouvé: '%s' (ID: %s, lien: %s)", title, imdbID, imdbLink)

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

			// Si on a un ID IMDb, récupérer plus de détails complets
			if imdbID != "" {
				log.Printf("[WebScraper] 📥 Récupération des détails complets IMDb pour ID: %s", imdbID)
				details := s.getMovieDetailsFromIMDB(imdbID)
				if details != nil {
					log.Printf("[WebScraper] ✅ Détails IMDb récupérés pour '%s':", details.Title)
					log.Printf("[WebScraper]   - Synopsis: %d caractères", len(details.Synopsis))
					log.Printf("[WebScraper]   - Réalisateur: %s", details.Director)
					log.Printf("[WebScraper]   - Scénaristes: %s", details.Writer)
					log.Printf("[WebScraper]   - Acteurs: %d acteurs", len(details.Actors))
					log.Printf("[WebScraper]   - Genres: %d genres", len(details.Genre))
					log.Printf("[WebScraper]   - Affiche: %s", details.PosterURL)
					log.Printf("[WebScraper]   - Durée: %d min", details.Runtime)
					log.Printf("[WebScraper]   - Plateforme: %s", details.Platform)

					// Fusionner tous les détails (les détails IMDb sont plus complets)
					if details.Title != "" {
						metadata.Title = details.Title
						metadata.OriginalTitle = details.OriginalTitle
					}
					if details.Director != "" {
						metadata.Director = details.Director
					}
					if details.Writer != "" {
						metadata.Writer = details.Writer
					}
					if len(details.Actors) > 0 {
						metadata.Actors = details.Actors
					}
					if len(details.Genre) > 0 {
						metadata.Genre = details.Genre
					}
					if details.Synopsis != "" {
						metadata.Synopsis = details.Synopsis
						metadata.Description = details.Synopsis
					}
					if details.PosterURL != "" {
						metadata.PosterURL = details.PosterURL
						log.Printf("[WebScraper] ✅ Affiche assignée: %s", metadata.PosterURL)
					} else {
						log.Printf("[WebScraper] ⚠️  Aucune affiche trouvée pour %s", imdbID)
					}
					if details.Runtime > 0 {
						metadata.Runtime = details.Runtime
					}
					if details.Platform != "" {
						metadata.Platform = details.Platform
					}
					if details.TrailerURL != "" {
						metadata.TrailerURL = details.TrailerURL
					}
					if details.ProductionYear > 0 {
						metadata.ProductionYear = details.ProductionYear
					}
					if details.Rating > 0 {
						metadata.Rating = details.Rating
					}
					metadata.IMDBID = imdbID
				} else {
					log.Printf("[WebScraper] ⚠️  Aucun détail récupéré pour IMDb ID: %s", imdbID)
				}
			} else {
				log.Printf("[WebScraper] ⚠️  Aucun IMDb ID trouvé pour: %s", title)
			}

			results = append(results, metadata)
		})
		if count > 0 {
			log.Printf("[WebScraper] Sélecteur '%s' a trouvé %d résultat(s)", selector, count)
		}
		if found && len(results) > 0 {
			break
		}
	}

	// TOUJOURS faire une recherche large pour être sûr de trouver les résultats
	// Même si les sélecteurs standards ont trouvé des résultats, on vérifie qu'ils ont bien un IMDb ID
	needsLargeSearch := len(results) == 0
	for _, r := range results {
		if r.IMDBID == "" {
			needsLargeSearch = true
			break
		}
	}

	if needsLargeSearch {
		log.Printf("[WebScraper] ⚠️  Recherche large pour trouver les IDs IMDb (résultats standards: %d, avec ID: %d)...", len(results), func() int {
			count := 0
			for _, r := range results {
				if r.IMDBID != "" {
					count++
				}
			}
			return count
		}())

		largeCount := doc.Find("a[href*='/title/tt']").Length()
		log.Printf("[WebScraper] Recherche large: %d liens IMDb trouvés dans la page", largeCount)

		seenIDs := make(map[string]bool)
		// Ajouter les IDs déjà trouvés
		for _, r := range results {
			if r.IMDBID != "" {
				seenIDs[r.IMDBID] = true
			}
		}

		doc.Find("a[href*='/title/tt']").Each(func(i int, sel *goquery.Selection) {
			if len(results) >= 5 {
				return
			}
			imdbLink, exists := sel.Attr("href")
			if !exists {
				return
			}
			imdbID := extractIMDBID(imdbLink)
			if imdbID == "" {
				return
			}

			// Éviter les doublons
			if seenIDs[imdbID] {
				return
			}
			seenIDs[imdbID] = true

			title := strings.TrimSpace(sel.Text())
			if title == "" {
				// Essayer de trouver le titre dans le parent ou les éléments proches
				title = strings.TrimSpace(sel.Parent().Text())
				if title == "" {
					title = strings.TrimSpace(sel.Closest("div, li, span").Text())
				}
			}
			// Nettoyer le titre (enlever les années, etc.)
			title = regexp.MustCompile(`\s*\(\d{4}\)\s*$`).ReplaceAllString(title, "")
			title = strings.TrimSpace(title)
			if title == "" {
				title = fmt.Sprintf("Film IMDb %s", imdbID)
			}

			log.Printf("[WebScraper] ✅ Résultat large trouvé: '%s' (ID: %s, lien: %s)", title, imdbID, imdbLink)

			// Créer un résultat basique
			metadata := &MovieMetadata{
				Title:         title,
				OriginalTitle: title,
				IMDBID:        imdbID,
			}

			// TOUJOURS récupérer les détails complets si on a un ID
			log.Printf("[WebScraper] 📥 Récupération FORCÉE des détails complets IMDb pour ID: %s", imdbID)
			details := s.getMovieDetailsFromIMDB(imdbID)
			if details != nil {
				log.Printf("[WebScraper] ✅ Détails IMDb récupérés (recherche large) pour '%s'", details.Title)
				metadata = details
			} else {
				log.Printf("[WebScraper] ❌ ÉCHEC récupération des détails pour IMDb ID: %s", imdbID)
			}

			results = append(results, metadata)
		})

		log.Printf("[WebScraper] Recherche large terminée: %d résultat(s) avec détails", len(results))
	}

	// FORCER la récupération des détails pour TOUS les résultats qui ont un IMDb ID
	// On récupère TOUJOURS les détails complets depuis IMDb si on a un ID
	log.Printf("[WebScraper] 🚨 POINT DE CONTRÔLE SearchMovieOnIMDB: %d résultat(s) avant vérification finale", len(results))
	log.Printf("[WebScraper] 🔄 Vérification finale SearchMovieOnIMDB: %d résultat(s) avant récupération des détails", len(results))
	for idx, result := range results {
		log.Printf("[WebScraper] 🔍 SearchMovieOnIMDB - Résultat #%d: Titre='%s', IMDb ID='%s'", idx+1, result.Title, result.IMDBID)
		if result.IMDBID != "" {
			log.Printf("[WebScraper] 🔄 Résultat #%d '%s' (ID: %s) - RÉCUPÉRATION FORCÉE des détails complets...", idx+1, result.Title, result.IMDBID)
			details := s.getMovieDetailsFromIMDB(result.IMDBID)
			if details != nil {
				log.Printf("[WebScraper] ✅ Détails complets récupérés pour '%s':", details.Title)
				log.Printf("[WebScraper]   - Synopsis: %d caractères", len(details.Synopsis))
				log.Printf("[WebScraper]   - Réalisateur: %s", details.Director)
				log.Printf("[WebScraper]   - Scénaristes: %s", details.Writer)
				log.Printf("[WebScraper]   - Acteurs: %d acteurs", len(details.Actors))
				log.Printf("[WebScraper]   - Genres: %d genres", len(details.Genre))
				log.Printf("[WebScraper]   - Affiche: %s", details.PosterURL)
				log.Printf("[WebScraper]   - Durée: %d min", details.Runtime)
				log.Printf("[WebScraper]   - Plateforme: %s", details.Platform)
				log.Printf("[WebScraper]   - Bande-annonce: %s", details.TrailerURL)

				// REMPLACER complètement avec les détails IMDb (plus complets)
				result.Title = details.Title
				result.OriginalTitle = details.OriginalTitle
				result.Director = details.Director
				result.Writer = details.Writer
				result.Actors = details.Actors
				result.Genre = details.Genre
				result.Synopsis = details.Synopsis
				result.Description = details.Synopsis // Utiliser synopsis comme description
				result.PosterURL = details.PosterURL
				result.BackdropURL = details.BackdropURL
				result.Runtime = details.Runtime
				result.Platform = details.Platform
				result.TrailerURL = details.TrailerURL
				result.ProductionYear = details.ProductionYear
				result.Rating = details.Rating
				result.IMDBID = details.IMDBID
				result.ReleaseDate = details.ReleaseDate

				log.Printf("[WebScraper] ✅ Résultat #%d COMPLÈTEMENT mis à jour avec TOUS les détails IMDb", idx+1)
			} else {
				log.Printf("[WebScraper] ❌ IMPOSSIBLE de récupérer les détails pour IMDb ID: %s", result.IMDBID)
			}
		} else {
			log.Printf("[WebScraper] ⚠️  Résultat #%d '%s' - PAS d'IMDb ID, impossible de récupérer les détails", idx+1, result.Title)
		}
	}

	log.Printf("[WebScraper] ✅ Recherche IMDb terminée: %d résultat(s) trouvé(s) pour '%s'", len(results), query)

	// DIAGNOSTIC FINAL: Vérifier que les résultats ont bien un IMDb ID
	for idx, result := range results {
		log.Printf("[WebScraper] 🔍 DIAGNOSTIC FINAL SearchMovieOnIMDB - Résultat #%d: Titre='%s', IMDb ID='%s' (VIDE=%v)", idx+1, result.Title, result.IMDBID, result.IMDBID == "")
	}

	return results, nil
}

// getMovieDetailsFromIMDB récupère les détails d'un film depuis IMDb
func (s *WebScraperService) getMovieDetailsFromIMDB(imdbID string) *MovieMetadata {
	log.Printf("[WebScraper] 🎬 DÉBUT getMovieDetailsFromIMDB pour ID: %s", imdbID)

	if imdbID == "" {
		log.Printf("[WebScraper] ❌ IMDb ID vide!")
		return nil
	}

	// Essayer d'abord la version française pour avoir le synopsis en français
	movieURL := fmt.Sprintf("https://www.imdb.com/fr/title/%s/", imdbID)
	log.Printf("[WebScraper] 📡 Requête IMDb URL: %s", movieURL)

	req, err := http.NewRequest("GET", movieURL, nil)
	if err != nil {
		log.Printf("[WebScraper] ❌ Erreur création requête IMDb: %v", err)
		return nil
	}
	req.Header.Set("User-Agent", s.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.9,en-US;q=0.8,en;q=0.7")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("[WebScraper] ❌ Erreur requête IMDb: %v", err)
		return nil
	}
	defer resp.Body.Close()

	log.Printf("[WebScraper] 📥 Réponse IMDb: status %d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		log.Printf("[WebScraper] ⚠️  Status code IMDb: %d, essai version anglaise...", resp.StatusCode)
		// Essayer la version anglaise en fallback
		movieURL = fmt.Sprintf("https://www.imdb.com/title/%s/", imdbID)
		log.Printf("[WebScraper] 📡 Requête IMDb URL (EN): %s", movieURL)
		req, _ = http.NewRequest("GET", movieURL, nil)
		req.Header.Set("User-Agent", s.userAgent)
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")
		resp, err = s.httpClient.Do(req)
		if err != nil {
			log.Printf("[WebScraper] ❌ Erreur requête IMDb (EN): %v", err)
			return nil
		}
		defer resp.Body.Close()
		log.Printf("[WebScraper] 📥 Réponse IMDb (EN): status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Printf("[WebScraper] ❌ Erreur parsing IMDb: %v", err)
		return nil
	}

	log.Printf("[WebScraper] ✅ Document IMDb parsé avec succès")

	details := &MovieMetadata{}
	details.IMDBID = imdbID

	// PRIORITÉ 1: Extraire les données JSON-LD structurées (beaucoup plus fiables que le scraping HTML)
	log.Printf("[WebScraper] 🔍 Extraction des données JSON-LD structurées...")
	jsonLDData := s.extractJSONLDFromIMDB(doc)
	if jsonLDData != nil {
		log.Printf("[WebScraper] ✅ Données JSON-LD trouvées, utilisation pour remplir les métadonnées")
		if jsonLDData.Name != "" {
			details.Title = jsonLDData.Name
			details.OriginalTitle = jsonLDData.Name
		}
		if jsonLDData.Description != "" {
			details.Synopsis = jsonLDData.Description
			details.Description = jsonLDData.Description
		}
		if len(jsonLDData.Director) > 0 {
			details.Director = jsonLDData.Director[0].Name
		}
		if len(jsonLDData.Author) > 0 {
			writers := []string{}
			for _, author := range jsonLDData.Author {
				writers = append(writers, author.Name)
			}
			details.Writer = strings.Join(writers, ", ")
		}
		if len(jsonLDData.Actor) > 0 {
			actors := []string{}
			for _, actor := range jsonLDData.Actor {
				actors = append(actors, actor.Name)
			}
			details.Actors = actors
		}
		if len(jsonLDData.Genre) > 0 {
			details.Genre = jsonLDData.Genre
		}
		if jsonLDData.Image != "" {
			details.PosterURL = jsonLDData.Image
		}
		if jsonLDData.DatePublished != "" {
			details.ReleaseDate = jsonLDData.DatePublished
		}
		if jsonLDData.AggregateRating.RatingValue > 0 {
			details.Rating = jsonLDData.AggregateRating.RatingValue
		}
		if jsonLDData.Duration != "" {
			// Format: "PT2H3M" -> convertir en minutes
			runtime := s.parseDuration(jsonLDData.Duration)
			if runtime > 0 {
				details.Runtime = runtime
			}
		}
		// Extraire l'année depuis la date de publication
		if jsonLDData.DatePublished != "" {
			year := extractYear(jsonLDData.DatePublished)
			if year > 1900 {
				details.ProductionYear = year
			}
		}
		log.Printf("[WebScraper] ✅ Métadonnées JSON-LD extraites: Titre=%s, Réalisateur=%s, Acteurs=%d, Genres=%d",
			details.Title, details.Director, len(details.Actors), len(details.Genre))
	}

	// PRIORITÉ 2: Compléter avec le scraping HTML si des données manquent
	log.Printf("[WebScraper] 🔍 Complétion avec scraping HTML si nécessaire...")

	// Titre (si pas déjà défini)
	if details.Title == "" {
		title := strings.TrimSpace(doc.Find("h1[data-testid='hero-title-block__title']").Text())
		if title == "" {
			title = strings.TrimSpace(doc.Find("h1").First().Text())
		}
		details.Title = title
		details.OriginalTitle = title
		log.Printf("[WebScraper] Titre IMDb (HTML): %s", details.Title)
	}

	// Synopsis - essayer plusieurs sélecteurs (si pas déjà défini)
	if details.Synopsis == "" {
		synopsisSelectors := []string{
			"span[data-testid='plot-xl']",
			"span[data-testid='plot-l']",
			"span[data-testid='plot']",
			".plot",
			"[class*='plot']",
			"[data-testid='plot']",
		}
		for _, sel := range synopsisSelectors {
			synopsis := strings.TrimSpace(doc.Find(sel).First().Text())
			if synopsis != "" && len(synopsis) > 20 {
				// Nettoyer le synopsis
				synopsis = regexp.MustCompile(`\s+`).ReplaceAllString(synopsis, " ")
				if len(synopsis) > 1000 {
					synopsis = synopsis[:1000] + "..."
				}
				details.Synopsis = synopsis
				details.Description = synopsis
				log.Printf("[WebScraper] Synopsis IMDb (HTML) trouvé (%d caractères)", len(synopsis))
				break
			}
		}
	}

	// Réalisateur - chercher dans la section "Réalisation" ou "Director" (si pas déjà défini)
	if details.Director == "" {
		// STRATÉGIE 1: Chercher dans les crédits principaux
		doc.Find("li[data-testid='title-pc-principal-credit']").Each(func(i int, sel *goquery.Selection) {
			labelText := strings.ToLower(strings.TrimSpace(sel.Text()))
			labelSpan := strings.ToLower(strings.TrimSpace(sel.Find("span").First().Text()))

			if strings.Contains(labelText, "réalisation") || strings.Contains(labelText, "director") ||
				strings.Contains(labelSpan, "réalisation") || strings.Contains(labelSpan, "director") {
				director := strings.TrimSpace(sel.Find("a").First().Text())
				if director != "" && len(director) > 2 {
					details.Director = director
					log.Printf("[WebScraper] ✅ Réalisateur IMDb trouvé: %s", director)
					return
				}
			}
		})

		// STRATÉGIE 2: Chercher dans tous les liens avec href contenant /name/
		if details.Director == "" {
			doc.Find("a[href*='/name/']").Each(func(i int, sel *goquery.Selection) {
				if i >= 50 { // Limiter la recherche
					return
				}
				// Chercher le contexte parent pour voir si c'est un réalisateur
				parentText := strings.ToLower(sel.Parent().Text())
				if (strings.Contains(parentText, "réalisation") || strings.Contains(parentText, "director")) &&
					!strings.Contains(parentText, "scénariste") && !strings.Contains(parentText, "writer") {
					director := strings.TrimSpace(sel.Text())
					if director != "" && len(director) > 2 && len(director) < 50 {
						details.Director = director
						log.Printf("[WebScraper] ✅ Réalisateur IMDb trouvé (stratégie 2): %s", director)
						return
					}
				}
			})
		}

		// STRATÉGIE 3: Chercher dans le texte de la page avec regex
		if details.Director == "" {
			pageText := doc.Text()
			directorPatterns := []*regexp.Regexp{
				regexp.MustCompile(`(?i)(?:Réalisation|Director)[:\s]+([A-Z][a-z]+\s+[A-Z][a-z]+)`),
				regexp.MustCompile(`(?i)(?:Réalisé par|Directed by)[:\s]+([A-Z][a-z]+\s+[A-Z][a-z]+)`),
			}
			for _, pattern := range directorPatterns {
				if matches := pattern.FindStringSubmatch(pageText); len(matches) > 1 {
					director := strings.TrimSpace(matches[1])
					if director != "" && len(director) > 2 {
						details.Director = director
						log.Printf("[WebScraper] ✅ Réalisateur IMDb trouvé (regex): %s", director)
						break
					}
				}
			}
		}
	}

	// Scénaristes/Writers - chercher dans la section "Scénaristes" ou "Writers" (si pas déjà défini)
	if details.Writer == "" {
		writers := []string{}
		seenWriters := make(map[string]bool)

		// STRATÉGIE 1: Chercher dans les crédits principaux
		doc.Find("li[data-testid='title-pc-principal-credit']").Each(func(i int, sel *goquery.Selection) {
			labelText := strings.ToLower(strings.TrimSpace(sel.Text()))
			labelSpan := strings.ToLower(strings.TrimSpace(sel.Find("span").First().Text()))

			if strings.Contains(labelText, "scénariste") || strings.Contains(labelText, "writer") ||
				strings.Contains(labelText, "scénario") || strings.Contains(labelText, "written") ||
				strings.Contains(labelSpan, "scénariste") || strings.Contains(labelSpan, "writer") {
				sel.Find("a").Each(func(j int, s *goquery.Selection) {
					writer := strings.TrimSpace(s.Text())
					if writer != "" && len(writer) > 2 && len(writer) < 50 {
						writerLower := strings.ToLower(writer)
						if !seenWriters[writerLower] {
							seenWriters[writerLower] = true
							writers = append(writers, writer)
						}
					}
				})
			}
		})

		// STRATÉGIE 2: Chercher dans tous les liens avec contexte
		if len(writers) == 0 {
			doc.Find("a[href*='/name/']").Each(func(i int, sel *goquery.Selection) {
				if i >= 100 { // Limiter
					return
				}
				parentText := strings.ToLower(sel.Parent().Text())
				if (strings.Contains(parentText, "scénariste") || strings.Contains(parentText, "writer") ||
					strings.Contains(parentText, "scénario") || strings.Contains(parentText, "written")) &&
					!strings.Contains(parentText, "director") && !strings.Contains(parentText, "réalisation") {
					writer := strings.TrimSpace(sel.Text())
					if writer != "" && len(writer) > 2 && len(writer) < 50 {
						writerLower := strings.ToLower(writer)
						if !seenWriters[writerLower] {
							seenWriters[writerLower] = true
							writers = append(writers, writer)
						}
					}
				}
			})
		}

		if len(writers) > 0 {
			details.Writer = strings.Join(writers, ", ")
			log.Printf("[WebScraper] ✅ Scénaristes IMDb (HTML) trouvés: %s", details.Writer)
		}
	}

	// Acteurs/Stars - chercher dans la section "Stars" ou "Cast" (si pas déjà défini)
	if len(details.Actors) == 0 {
		actors := []string{}
		seenActors := make(map[string]bool)

		// STRATÉGIE 1: Chercher avec data-testid
		doc.Find("a[data-testid='title-cast-item__actor']").Each(func(i int, s *goquery.Selection) {
			if i < 10 { // Limiter à 10 acteurs principaux
				actor := strings.TrimSpace(s.Text())
				if actor != "" && len(actor) > 2 && len(actor) < 50 {
					actorLower := strings.ToLower(actor)
					if !seenActors[actorLower] {
						seenActors[actorLower] = true
						actors = append(actors, actor)
					}
				}
			}
		})

		// STRATÉGIE 2: Chercher dans les crédits principaux "Stars"
		if len(actors) < 3 {
			doc.Find("li[data-testid='title-pc-principal-credit']").Each(func(i int, sel *goquery.Selection) {
				labelText := strings.ToLower(strings.TrimSpace(sel.Text()))
				labelSpan := strings.ToLower(strings.TrimSpace(sel.Find("span").First().Text()))

				if strings.Contains(labelText, "star") || strings.Contains(labelText, "avec") ||
					strings.Contains(labelSpan, "star") || strings.Contains(labelSpan, "avec") {
					sel.Find("a").Each(func(j int, s *goquery.Selection) {
						if j < 10 && len(actors) < 10 {
							actor := strings.TrimSpace(s.Text())
							if actor != "" && len(actor) > 2 && len(actor) < 50 {
								actorLower := strings.ToLower(actor)
								if !seenActors[actorLower] {
									seenActors[actorLower] = true
									actors = append(actors, actor)
								}
							}
						}
					})
				}
			})
		}

		// STRATÉGIE 3: Chercher dans la section casting
		if len(actors) < 3 {
			doc.Find(".cast_list a[href*='/name/'], .ipc-metadata-list-item__content-container a[href*='/name/']").Each(func(i int, s *goquery.Selection) {
				if i < 10 && len(actors) < 10 {
					actor := strings.TrimSpace(s.Text())
					if actor != "" && len(actor) > 2 && len(actor) < 50 {
						actorLower := strings.ToLower(actor)
						if !seenActors[actorLower] {
							seenActors[actorLower] = true
							actors = append(actors, actor)
						}
					}
				}
			})
		}

		details.Actors = actors
		if len(actors) > 0 {
			log.Printf("[WebScraper] ✅ Acteurs IMDb (HTML) trouvés (%d): %v", len(actors), actors)
		} else {
			log.Printf("[WebScraper] ⚠️  Aucun acteur trouvé (HTML)")
		}
	}

	// Genres (si pas déjà défini)
	if len(details.Genre) == 0 {
		genres := []string{}
		genreSelectors := []string{
			"a[href*='/genre/']",
			"[data-testid='genres'] a",
			".ipc-chip a[href*='/genre/']",
		}
		for _, sel := range genreSelectors {
			doc.Find(sel).Each(func(i int, s *goquery.Selection) {
				genre := strings.TrimSpace(s.Text())
				if genre != "" && len(genre) > 1 {
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
			log.Printf("[WebScraper] Genres IMDb (HTML) trouvés: %v", genres)
		}
	}

	// Année et Date de sortie (si pas déjà défini)
	if details.ProductionYear == 0 {
		yearText := doc.Find("a[href*='/releaseinfo']").First().Text()
		if yearText == "" {
			yearText = doc.Find("span[class*='TitleBlockMetaData']").Text()
		}
		if yearText == "" {
			// Chercher dans le texte de la page
			pageText := doc.Text()
			yearMatch := regexp.MustCompile(`\b(19|20)\d{2}\b`)
			if matches := yearMatch.FindString(pageText); matches != "" {
				yearText = matches
			}
		}
		year := extractYear(yearText)
		if year > 1900 {
			details.ProductionYear = year
			log.Printf("[WebScraper] ✅ Année IMDb (HTML) trouvée: %d", year)
		}

		// Date de sortie complète (format YYYY-MM-DD ou texte)
		releaseDateSelectors := []string{
			"a[href*='/releaseinfo']",
			"li[data-testid='title-details-releasedate']",
			"[data-testid='title-details-releasedate']",
			"span.ipc-metadata-list-item__list-content-item",
		}
		for _, sel := range releaseDateSelectors {
			releaseText := strings.TrimSpace(doc.Find(sel).First().Text())
			if releaseText != "" {
				// Extraire la date (format: "15 novembre 2024" ou "2024-11-15" ou "November 15, 2024")
				dateMatch := regexp.MustCompile(`(\d{4}-\d{2}-\d{2})|(\d{1,2}\s+\w+\s+\d{4})`)
				if matches := dateMatch.FindString(releaseText); matches != "" {
					details.ReleaseDate = matches
					log.Printf("[WebScraper] ✅ Date de sortie IMDb (HTML) trouvée: %s", details.ReleaseDate)
					break
				}
			}
		}
	}

	// Note/Rating (si pas déjà défini)
	if details.Rating == 0 {
		ratingText := doc.Find("span[class*='AggregateRatingButton__RatingScore']").First().Text()
		if ratingText == "" {
			ratingText = doc.Find("[data-testid='ratingGroup--imdb-rating']").Text()
		}
		rating := extractRating(ratingText)
		if rating > 0 {
			details.Rating = rating
			log.Printf("[WebScraper] Note IMDb (HTML) trouvée: %.1f", rating)
		}
	}

	// Durée/Runtime (si pas déjà défini)
	if details.Runtime == 0 {
		runtimeText := doc.Find("li[data-testid='title-techspec_runtime']").Text()
		if runtimeText == "" {
			runtimeText = doc.Find("time[datetime]").First().AttrOr("datetime", "")
		}
		// Chercher "2h 3min" ou "123 min"
		runtimeMatch := regexp.MustCompile(`(\d+)\s*h\s*(\d+)\s*min`)
		if matches := runtimeMatch.FindStringSubmatch(runtimeText); len(matches) >= 3 {
			var hours, minutes int
			fmt.Sscanf(matches[1], "%d", &hours)
			fmt.Sscanf(matches[2], "%d", &minutes)
			details.Runtime = hours*60 + minutes
			log.Printf("[WebScraper] Durée IMDb (HTML) trouvée: %d minutes (%dh %dmin)", details.Runtime, hours, minutes)
		} else {
			// Essayer format "123 min"
			runtimeMatch2 := regexp.MustCompile(`(\d+)\s*min`)
			if matches := runtimeMatch2.FindStringSubmatch(runtimeText); len(matches) >= 2 {
				fmt.Sscanf(matches[1], "%d", &details.Runtime)
				log.Printf("[WebScraper] Durée IMDb (HTML) trouvée: %d minutes", details.Runtime)
			}
		}
	}

	// Poster/Affiche - essayer plusieurs sélecteurs avec logs détaillés (si pas déjà défini)
	if details.PosterURL == "" {
		posterSelectors := []string{
			"img[data-testid='hero-image']",
			"img[data-testid='hero-image__poster']",
			"img[class*='ipc-image']",
			"img[src*='m.media-amazon.com']",
			"img[src*='media-amazon.com']",
			".ipc-media img",
			"[class*='poster'] img",
			".ipc-poster img",
			"img.ipc-image",
			"img[loading='lazy']",
		}
		log.Printf("[WebScraper] 🔍 Recherche de l'affiche IMDb pour %s...", imdbID)
		for idx, sel := range posterSelectors {
			posterURL := ""
			exists := false

			// Essayer src d'abord
			posterURL, exists = doc.Find(sel).First().Attr("src")
			if !exists || posterURL == "" {
				// Essayer data-src
				posterURL, exists = doc.Find(sel).First().Attr("data-src")
			}
			if !exists || posterURL == "" {
				// Essayer data-image-url
				posterURL, exists = doc.Find(sel).First().Attr("data-image-url")
			}
			if !exists || posterURL == "" {
				// Essayer data-lazy
				posterURL, exists = doc.Find(sel).First().Attr("data-lazy")
			}

			if posterURL != "" {
				// Nettoyer l'URL (enlever les paramètres de taille pour avoir la meilleure qualité)
				// IMDb utilise des URLs comme: https://m.media-amazon.com/images/M/..._V1_UX182_CR0,0,182,268_AL_.jpg
				// On veut: https://m.media-amazon.com/images/M/..._V1_.jpg pour la meilleure qualité
				posterURL = regexp.MustCompile(`\._[^_]+_\.`).ReplaceAllString(posterURL, "._V1_.")
				// Enlever les paramètres de taille à la fin
				posterURL = regexp.MustCompile(`_UX\d+_CR\d+,\d+,\d+,\d+_`).ReplaceAllString(posterURL, "_")
				posterURL = regexp.MustCompile(`_AL_`).ReplaceAllString(posterURL, "")
				// S'assurer qu'on a une URL complète
				if !strings.HasPrefix(posterURL, "http") {
					posterURL = "https://" + strings.TrimPrefix(posterURL, "//")
				}
				details.PosterURL = posterURL
				log.Printf("[WebScraper] ✅ Affiche IMDb trouvée avec sélecteur #%d '%s': %s", idx+1, sel, posterURL)
				break
			} else {
				log.Printf("[WebScraper] Sélecteur #%d '%s': aucune affiche trouvée", idx+1, sel)
			}
		}

		// Si toujours pas d'affiche, chercher toutes les images
		if details.PosterURL == "" {
			log.Printf("[WebScraper] ⚠️  Aucune affiche avec sélecteurs standards, recherche large...")
			doc.Find("img").Each(func(i int, sel *goquery.Selection) {
				if i >= 20 { // Limiter la recherche
					return
				}
				src, _ := sel.Attr("src")
				if src != "" && (strings.Contains(src, "media-amazon.com") || strings.Contains(src, "imdb.com")) {
					if strings.Contains(src, "poster") || strings.Contains(src, "images") {
						// Nettoyer l'URL
						src = regexp.MustCompile(`\._[^_]+_\.`).ReplaceAllString(src, "._V1_.")
						if !strings.HasPrefix(src, "http") {
							src = "https://" + strings.TrimPrefix(src, "//")
						}
						details.PosterURL = src
						log.Printf("[WebScraper] ✅ Affiche IMDb trouvée (recherche large): %s", src)
						return
					}
				}
			})

			if details.PosterURL == "" {
				log.Printf("[WebScraper] ❌ Aucune affiche trouvée (HTML) pour IMDb ID: %s", imdbID)
			}
		}
	}

	// Plateforme - chercher dans le texte (Amazon Prime Video, Netflix, etc.) (si pas déjà défini)
	if details.Platform == "" {
		platformText := doc.Text()
		platformMatch := regexp.MustCompile(`(?i)(Amazon\s+Prime\s+Video|Netflix|Disney\+|HBO|Hulu|Apple\s+TV\+|Prime\s+Video)`)
		if matches := platformMatch.FindStringSubmatch(platformText); len(matches) > 1 {
			details.Platform = matches[1]
			log.Printf("[WebScraper] ✅ Plateforme IMDb (HTML) trouvée: %s", details.Platform)
		}
	}

	// Chercher la bande annonce directement sur IMDb d'abord
	log.Printf("[WebScraper] 🔍 Recherche bande-annonce sur IMDb...")
	trailerSelectors := []string{
		"a[href*='/videoplayer/']",
		"a[data-testid='videos-slate-overlay-vertical-video']",
		"a[href*='video']",
		".ipc-video a",
		"[class*='video'] a",
	}
	for _, sel := range trailerSelectors {
		trailerLink, exists := doc.Find(sel).First().Attr("href")
		if exists && trailerLink != "" {
			if !strings.HasPrefix(trailerLink, "http") {
				trailerLink = "https://www.imdb.com" + trailerLink
			}
			details.TrailerURL = trailerLink
			log.Printf("[WebScraper] ✅ Bande annonce IMDb trouvée: %s", trailerLink)
			break
		}
	}

	// Si pas trouvé sur IMDb, chercher sur YouTube
	if details.TrailerURL == "" && details.Title != "" {
		log.Printf("[WebScraper] 🔍 Recherche bande-annonce YouTube pour '%s' (%d)...", details.Title, details.ProductionYear)
		trailerURL := s.searchYouTubeTrailer(details.Title, details.ProductionYear)
		if trailerURL != "" {
			details.TrailerURL = trailerURL
			log.Printf("[WebScraper] ✅ Bande annonce YouTube trouvée: %s", trailerURL)
		} else {
			log.Printf("[WebScraper] ⚠️  Aucune bande-annonce trouvée sur YouTube")
		}
	}

	log.Printf("[WebScraper] ✅ Détails IMDb complets extraits pour: %s (ID: %s)", details.Title, imdbID)
	log.Printf("[WebScraper] 📊 RÉSUMÉ des données récupérées:")
	log.Printf("[WebScraper]   - Titre: %s", details.Title)
	log.Printf("[WebScraper]   - Synopsis: %d caractères", len(details.Synopsis))
	log.Printf("[WebScraper]   - Réalisateur: %s", details.Director)
	log.Printf("[WebScraper]   - Scénaristes: %s", details.Writer)
	log.Printf("[WebScraper]   - Acteurs: %d", len(details.Actors))
	log.Printf("[WebScraper]   - Genres: %d", len(details.Genre))
	log.Printf("[WebScraper]   - Année: %d", details.ProductionYear)
	log.Printf("[WebScraper]   - Date sortie: %s", details.ReleaseDate)
	log.Printf("[WebScraper]   - Durée: %d min", details.Runtime)
	log.Printf("[WebScraper]   - Note: %.1f", details.Rating)
	log.Printf("[WebScraper]   - Affiche: %s", details.PosterURL)
	log.Printf("[WebScraper]   - Bande-annonce: %s", details.TrailerURL)
	log.Printf("[WebScraper]   - Plateforme: %s", details.Platform)
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
			for idx, result := range results {
				log.Printf("[WebScraper] 🔍 Résultat IMDb #%d: Titre='%s', IMDb ID='%s' (VIDE=%v)", idx+1, result.Title, result.IMDBID, result.IMDBID == "")
				key := strings.ToLower(result.Title)
				if !seenTitles[key] {
					seenTitles[key] = true
					allResults = append(allResults, result)
					log.Printf("[WebScraper] ✅ Résultat ajouté à allResults (total: %d, IMDb ID: '%s')", len(allResults), result.IMDBID)
				} else {
					log.Printf("[WebScraper] ⚠️  Résultat '%s' déjà présent (doublon ignoré)", result.Title)
				}
			}
		}

		// Limiter à 15 résultats maximum pour éviter trop de données
		if len(allResults) >= 15 {
			break
		}
	}

	log.Printf("[WebScraper] Total de %d résultat(s) unique(s) trouvé(s) pour '%s'", len(allResults), filename)

	// CRITIQUE: Vérifier immédiatement si les résultats ont un IMDb ID
	log.Printf("[WebScraper] 🔍 VÉRIFICATION IMMÉDIATE: Vérification des IMDb IDs dans les résultats...")
	for idx, result := range allResults {
		log.Printf("[WebScraper] 🔍 Résultat #%d IMMÉDIAT: Titre='%s', IMDb ID='%s' (VIDE=%v)", idx+1, result.Title, result.IMDBID, result.IMDBID == "")
	}

	log.Printf("[WebScraper] 🚨 POINT DE CONTRÔLE 1: Code continue après 'Total de'")

	// DIAGNOSTIC: Vérifier les IDs IMDb des résultats
	log.Printf("[WebScraper] 🚨 POINT DE CONTRÔLE 2: Début du diagnostic")
	for idx, result := range allResults {
		log.Printf("[WebScraper] 🔍 DIAGNOSTIC Résultat #%d: Titre='%s', IMDb ID='%s'", idx+1, result.Title, result.IMDBID)
	}
	log.Printf("[WebScraper] 🚨 POINT DE CONTRÔLE 3: Fin du diagnostic, début vérification finale")

	// FORCER la récupération des détails complets pour TOUS les résultats qui ont un IMDb ID
	// Cette étape est CRITIQUE pour avoir toutes les informations (synopsis, réalisateur, acteurs, affiche, etc.)
	log.Printf("[WebScraper] 🔄 VÉRIFICATION FINALE: %d résultat(s) avant récupération des détails complets", len(allResults))
	log.Printf("[WebScraper] 🚨 POINT DE CONTRÔLE 4: Début de la boucle de vérification finale")
	for idx, result := range allResults {
		log.Printf("[WebScraper] 🔍 Vérification résultat #%d: Titre='%s', IMDb ID='%s' (vide=%v)", idx+1, result.Title, result.IMDBID, result.IMDBID == "")
		if result.IMDBID != "" {
			log.Printf("[WebScraper] 🔄 Résultat #%d '%s' (ID: %s) - RÉCUPÉRATION FORCÉE des détails complets IMDb...", idx+1, result.Title, result.IMDBID)
			details := s.getMovieDetailsFromIMDB(result.IMDBID)
			if details != nil {
				log.Printf("[WebScraper] ✅ Détails complets récupérés pour '%s':", details.Title)
				log.Printf("[WebScraper]   - Synopsis: %d caractères", len(details.Synopsis))
				log.Printf("[WebScraper]   - Réalisateur: %s", details.Director)
				log.Printf("[WebScraper]   - Scénaristes: %s", details.Writer)
				log.Printf("[WebScraper]   - Acteurs: %d acteurs", len(details.Actors))
				log.Printf("[WebScraper]   - Genres: %d genres", len(details.Genre))
				log.Printf("[WebScraper]   - Affiche: %s", details.PosterURL)
				log.Printf("[WebScraper]   - Durée: %d min", details.Runtime)
				log.Printf("[WebScraper]   - Plateforme: %s", details.Platform)
				log.Printf("[WebScraper]   - Bande-annonce: %s", details.TrailerURL)

				// REMPLACER complètement avec les détails IMDb (plus complets)
				if details.Title != "" {
					result.Title = details.Title
					result.OriginalTitle = details.OriginalTitle
				}
				if details.Director != "" {
					result.Director = details.Director
				}
				if details.Writer != "" {
					result.Writer = details.Writer
				}
				if len(details.Actors) > 0 {
					result.Actors = details.Actors
				}
				if len(details.Genre) > 0 {
					result.Genre = details.Genre
				}
				if details.Synopsis != "" {
					result.Synopsis = details.Synopsis
					result.Description = details.Synopsis
				}
				if details.PosterURL != "" {
					result.PosterURL = details.PosterURL
				}
				if details.BackdropURL != "" {
					result.BackdropURL = details.BackdropURL
				}
				if details.Runtime > 0 {
					result.Runtime = details.Runtime
				}
				if details.Platform != "" {
					result.Platform = details.Platform
				}
				if details.TrailerURL != "" {
					result.TrailerURL = details.TrailerURL
				}
				if details.ProductionYear > 0 {
					result.ProductionYear = details.ProductionYear
				}
				if details.Rating > 0 {
					result.Rating = details.Rating
				}
				if details.ReleaseDate != "" {
					result.ReleaseDate = details.ReleaseDate
				}
				result.IMDBID = details.IMDBID // S'assurer que l'ID est bien présent
			} else {
				log.Printf("[WebScraper] ❌ ÉCHEC récupération des détails pour IMDb ID: %s", result.IMDBID)
			}
		} else {
			log.Printf("[WebScraper] ⚠️  Résultat #%d '%s' n'a pas d'IMDb ID - impossible de récupérer les détails", idx+1, result.Title)
		}
	}
	log.Printf("[WebScraper] ✅ Vérification finale terminée pour %d résultat(s)", len(allResults))

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

// Structures pour les données JSON-LD d'IMDb
type JSONLDData struct {
	Type            string          `json:"@type"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	Image           string          `json:"image"`
	DatePublished   string          `json:"datePublished"`
	Duration        string          `json:"duration"`
	Director        []Person        `json:"director"`
	Author          []Person        `json:"author"`
	Actor           []Person        `json:"actor"`
	Genre           []string        `json:"genre"`
	AggregateRating AggregateRating `json:"aggregateRating"`
}

type Person struct {
	Type string `json:"@type"`
	Name string `json:"name"`
}

type AggregateRating struct {
	Type        string  `json:"@type"`
	RatingValue float64 `json:"ratingValue"`
	BestRating  float64 `json:"bestRating"`
	WorstRating float64 `json:"worstRating"`
	RatingCount int     `json:"ratingCount"`
}

// extractJSONLDFromIMDB extrait les données JSON-LD structurées de la page IMDb
func (s *WebScraperService) extractJSONLDFromIMDB(doc *goquery.Document) *JSONLDData {
	// Chercher toutes les balises script avec type="application/ld+json"
	var jsonLDData *JSONLDData

	doc.Find("script[type='application/ld+json']").Each(func(i int, sel *goquery.Selection) {
		jsonText := strings.TrimSpace(sel.Text())
		if jsonText == "" {
			return
		}

		// Essayer de parser le JSON
		var data JSONLDData
		if err := json.Unmarshal([]byte(jsonText), &data); err != nil {
			// Si c'est un tableau, essayer de parser le premier élément
			var dataArray []JSONLDData
			if err2 := json.Unmarshal([]byte(jsonText), &dataArray); err2 == nil && len(dataArray) > 0 {
				data = dataArray[0]
			} else {
				log.Printf("[WebScraper] ⚠️  Erreur parsing JSON-LD #%d: %v", i, err)
				return
			}
		}

		// Vérifier que c'est bien un film (Movie ou VideoObject)
		if data.Type == "Movie" || data.Type == "VideoObject" || strings.Contains(strings.ToLower(data.Type), "movie") {
			jsonLDData = &data
			log.Printf("[WebScraper] ✅ Données JSON-LD trouvées: Type=%s, Titre=%s", data.Type, data.Name)
			return
		}
	})

	return jsonLDData
}

// parseDuration convertit une durée ISO 8601 (ex: "PT2H3M") en minutes
func (s *WebScraperService) parseDuration(duration string) int {
	// Format ISO 8601: PT2H3M = 2 heures 3 minutes
	// PT123M = 123 minutes
	// PT2H = 2 heures

	re := regexp.MustCompile(`PT(?:(\d+)H)?(?:(\d+)M)?`)
	matches := re.FindStringSubmatch(duration)
	if len(matches) == 0 {
		return 0
	}

	hours := 0
	minutes := 0

	if len(matches) > 1 && matches[1] != "" {
		fmt.Sscanf(matches[1], "%d", &hours)
	}
	if len(matches) > 2 && matches[2] != "" {
		fmt.Sscanf(matches[2], "%d", &minutes)
	}

	return hours*60 + minutes
}
