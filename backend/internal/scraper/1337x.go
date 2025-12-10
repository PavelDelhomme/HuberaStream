package scraper

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"streammake/internal/models"
)

type Scraper1337x struct {
	baseURL      string
	client       *http.Client
	proxyManager *ProxyManager
	useProxy     bool
}

// Category représente une catégorie 1337x
type Category1337x string

const (
	CategoryPopularMovies  Category1337x = "popular-movies"
	CategoryTV             Category1337x = "TV"
	CategoryMovieLibrary   Category1337x = "movie-library"
	CategoryAnime          Category1337x = "Anime"
	CategoryDocumentaries  Category1337x = "Documentaries"
	CategoryMovies         Category1337x = "Movies"
)

func New1337xScraper() *Scraper1337x {
	pm := NewProxyManager()
	return &Scraper1337x{
		baseURL:      "https://www.1337x.to",
		proxyManager: pm,
		useProxy:     false, // Désactivé par défaut, activer si nécessaire
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetUseProxy active/désactive l'utilisation de proxies
func (s *Scraper1337x) SetUseProxy(use bool) {
	s.useProxy = use
}

// FetchProxies récupère et teste les proxies
func (s *Scraper1337x) FetchProxies(ctx context.Context) error {
	return s.proxyManager.FetchProxies(ctx)
}

// Search recherche des torrents avec une requête
func (s *Scraper1337x) Search(ctx context.Context, query string, category *models.Category, subCategory *models.SubCategory) ([]*TorrentResult, error) {
	searchURL := fmt.Sprintf("%s/search/%s/1/", s.baseURL, url.QueryEscape(query))
	return s.fetchAndParse(ctx, searchURL, "")
}

// BrowseCategory parcourt une catégorie spécifique
func (s *Scraper1337x) BrowseCategory(ctx context.Context, category Category1337x, page int, language string) ([]*TorrentResult, error) {
	var categoryURL string
	
	switch category {
	case CategoryPopularMovies:
		categoryURL = fmt.Sprintf("%s/popular-movies", s.baseURL)
	case CategoryTV:
		categoryURL = fmt.Sprintf("%s/cat/TV/%d/", s.baseURL, page)
	case CategoryMovieLibrary:
		categoryURL = fmt.Sprintf("%s/movie-library/%d/", s.baseURL, page)
	case CategoryAnime:
		categoryURL = fmt.Sprintf("%s/cat/Anime/%d/", s.baseURL, page)
	case CategoryDocumentaries:
		categoryURL = fmt.Sprintf("%s/cat/Documentaries/%d/", s.baseURL, page)
	case CategoryMovies:
		categoryURL = fmt.Sprintf("%s/cat/Movies/%d/", s.baseURL, page)
	default:
		return nil, fmt.Errorf("catégorie non supportée: %s", category)
	}

	// Ajouter le filtre de langue si spécifié
	if language != "" {
		categoryURL += fmt.Sprintf("?lang=%s", url.QueryEscape(language))
	}

	return s.fetchAndParse(ctx, categoryURL, language)
}

// fetchAndParse récupère et parse une URL
func (s *Scraper1337x) fetchAndParse(ctx context.Context, url string, languageFilter string) ([]*TorrentResult, error) {
	var client *http.Client
	var err error

	if s.useProxy {
		proxyURL := s.proxyManager.GetNextProxy()
		client, err = s.proxyManager.GetClientWithProxy(proxyURL)
		if err != nil {
			// Fallback sur client sans proxy
			client = s.client
		}
	} else {
		client = s.client
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la requête: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("code de statut non OK: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la lecture du corps: %w", err)
	}

	return s.parseSearchResults(string(body), languageFilter)
}

// parseSearchResults parse les résultats de recherche avec un parser HTML amélioré
func (s *Scraper1337x) parseSearchResults(html string, languageFilter string) ([]*TorrentResult, error) {
	var results []*TorrentResult

	// Parser amélioré utilisant la structure HTML de 1337x
	// Les résultats sont dans des <tr> avec classe "table-list-wrap"
	rowRe := regexp.MustCompile(`<tr[^>]*class="table-list-wrap"[^>]*>([\s\S]*?)</tr>`)
	rows := rowRe.FindAllString(html, -1)

	for i, row := range rows {
		if i >= 50 { // Limiter à 50 résultats
			break
		}

		result := s.parseRow(row)
		if result != nil {
			// Filtrer par langue si spécifié
			if languageFilter != "" && result.Language != "" && !strings.EqualFold(result.Language, languageFilter) {
				continue
			}
			results = append(results, result)
		}
	}

	return results, nil
}

// parseRow parse une ligne de résultat
func (s *Scraper1337x) parseRow(row string) *TorrentResult {
	result := &TorrentResult{
		Source: models.TorrentSource1337x,
	}

	// Extraire le nom et l'URL
	nameRe := regexp.MustCompile(`<a[^>]*href="(/torrent/[^"]+)"[^>]*>([^<]+)</a>`)
	nameMatches := nameRe.FindStringSubmatch(row)
	if len(nameMatches) >= 3 {
		result.TorrentURL = s.baseURL + nameMatches[1]
		result.Name = strings.TrimSpace(nameMatches[2])
	} else {
		return nil
	}

	// Extraire les seeders
	seedersRe := regexp.MustCompile(`<td[^>]*class="seeds"[^>]*>(\d+)</td>`)
	if matches := seedersRe.FindStringSubmatch(row); len(matches) > 1 {
		if val, err := strconv.Atoi(matches[1]); err == nil {
			result.Seeders = val
		}
	}

	// Extraire les leechers
	leechersRe := regexp.MustCompile(`<td[^>]*class="leeches"[^>]*>(\d+)</td>`)
	if matches := leechersRe.FindStringSubmatch(row); len(matches) > 1 {
		if val, err := strconv.Atoi(matches[1]); err == nil {
			result.Leechers = val
		}
	}

	// Extraire la date d'upload
	dateRe := regexp.MustCompile(`<td[^>]*class="coll-date"[^>]*>([^<]+)</td>`)
	if matches := dateRe.FindStringSubmatch(row); len(matches) > 1 {
		result.UploadDate = strings.TrimSpace(matches[1])
	}

	// Extraire la taille
	sizeRe := regexp.MustCompile(`<td[^>]*class="size[^"]*"[^>]*>([^<]+)</td>`)
	if matches := sizeRe.FindStringSubmatch(row); len(matches) > 1 {
		result.Size = formatSize(strings.TrimSpace(matches[1]))
	}

	// Extraire la qualité depuis le nom (1080p, 720p, etc.)
	qualityRe := regexp.MustCompile(`(\d+p|4K|2160p|1440p|1080p|720p|480p|360p)`)
	if matches := qualityRe.FindStringSubmatch(result.Name); len(matches) > 1 {
		result.Quality = matches[1]
	}

	// Extraire la langue depuis le nom (FR, EN, etc.)
	langRe := regexp.MustCompile(`\b(FR|EN|ES|DE|IT|JP|CN|KR|RU|PT|NL|PL|TR|AR|HI|TH|VI)\b`)
	if matches := langRe.FindStringSubmatch(strings.ToUpper(result.Name)); len(matches) > 1 {
		result.Language = matches[1]
	}

	// Extraire la catégorie
	categoryRe := regexp.MustCompile(`<a[^>]*href="/cat/([^/]+)/"`)
	if matches := categoryRe.FindStringSubmatch(row); len(matches) > 1 {
		result.Category = matches[1]
	}

	return result
}

// GetTorrentDetails récupère les détails d'un torrent
func (s *Scraper1337x) GetTorrentDetails(ctx context.Context, torrentURL string) (*TorrentResult, error) {
	var client *http.Client
	var err error

	if s.useProxy {
		proxyURL := s.proxyManager.GetNextProxy()
		client, err = s.proxyManager.GetClientWithProxy(proxyURL)
		if err != nil {
			client = s.client
		}
	} else {
		client = s.client
	}

	req, err := http.NewRequestWithContext(ctx, "GET", torrentURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return s.parseTorrentDetails(string(body), torrentURL)
}

// parseTorrentDetails parse les détails d'un torrent
func (s *Scraper1337x) parseTorrentDetails(html, torrentURL string) (*TorrentResult, error) {
	result := &TorrentResult{
		TorrentURL: torrentURL,
		Source:     models.TorrentSource1337x,
	}

	// Extraire le magnet link
	magnetRe := regexp.MustCompile(`href="(magnet:\?[^"]+)"`)
	if matches := magnetRe.FindStringSubmatch(html); len(matches) > 1 {
		result.MagnetURL = matches[1]
	}

	// Extraire les seeders
	seedersRe := regexp.MustCompile(`<span class="seeds">(\d+)</span>`)
	if matches := seedersRe.FindStringSubmatch(html); len(matches) > 1 {
		if val, err := strconv.Atoi(matches[1]); err == nil {
			result.Seeders = val
		}
	}

	// Extraire les leechers
	leechersRe := regexp.MustCompile(`<span class="leeches">(\d+)</span>`)
	if matches := leechersRe.FindStringSubmatch(html); len(matches) > 1 {
		if val, err := strconv.Atoi(matches[1]); err == nil {
			result.Leechers = val
		}
	}

	// Extraire la taille
	sizeRe := regexp.MustCompile(`Size[^>]*>([^<]+)</li>`)
	if matches := sizeRe.FindStringSubmatch(html); len(matches) > 1 {
		result.Size = formatSize(matches[1])
	}

	// Extraire le nom
	nameRe := regexp.MustCompile(`<h1[^>]*>([^<]+)</h1>`)
	if matches := nameRe.FindStringSubmatch(html); len(matches) > 1 {
		result.Name = strings.TrimSpace(matches[1])
	}

	// Extraire la qualité
	qualityRe := regexp.MustCompile(`(\d+p|4K|2160p|1440p|1080p|720p|480p|360p)`)
	if matches := qualityRe.FindStringSubmatch(result.Name); len(matches) > 1 {
		result.Quality = matches[1]
	}

	// Extraire la langue
	langRe := regexp.MustCompile(`\b(FR|EN|ES|DE|IT|JP|CN|KR|RU|PT|NL|PL|TR|AR|HI|TH|VI)\b`)
	if matches := langRe.FindStringSubmatch(strings.ToUpper(result.Name)); len(matches) > 1 {
		result.Language = matches[1]
	}

	return result, nil
}
