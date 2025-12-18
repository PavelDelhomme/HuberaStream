package scraper

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"streammake/internal/models"

	"github.com/PuerkitoBio/goquery"
)

type PirateBayScraper struct {
	baseURL string
	client  *http.Client
}

func NewPirateBayScraper() *PirateBayScraper {
	return &PirateBayScraper{
		baseURL: "https://thepiratebay.org",
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (s *PirateBayScraper) SetBaseURL(url string) {
	s.baseURL = url
}

func (s *PirateBayScraper) Search(ctx context.Context, query string, category *models.Category, subCategory *models.SubCategory) ([]*TorrentResult, error) {
	// The Pirate Bay utilise category:201 pour les films/vidéos
	// Format: /search.php?q=query&category=201
	searchURL := fmt.Sprintf("%s/search.php?q=%s&category=201", s.baseURL, url.QueryEscape(query))
	return s.fetchAndParse(ctx, searchURL)
}

func (s *PirateBayScraper) GetTorrentDetails(ctx context.Context, torrentURL string) (*TorrentResult, error) {
	results, err := s.fetchAndParse(ctx, torrentURL)
	if err != nil {
		return nil, err
	}
	if len(results) > 0 {
		return results[0], nil
	}
	return nil, fmt.Errorf("aucun résultat trouvé")
}

func (s *PirateBayScraper) fetchAndParse(ctx context.Context, url string) ([]*TorrentResult, error) {
	log.Printf("[PirateBay] Début de fetchAndParse pour URL: %s", url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la création de la requête: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := s.client.Do(req)
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

	htmlContent := string(body)
	log.Printf("[PirateBay] Taille du HTML reçu: %d bytes", len(htmlContent))

	results, err := s.parseSearchResults(htmlContent)
	if err != nil {
		log.Printf("[PirateBay] Erreur lors du parsing: %v", err)
		return nil, err
	}

	log.Printf("[PirateBay] Parsing terminé, %d résultats trouvés", len(results))
	return results, nil
}

func (s *PirateBayScraper) parseSearchResults(html string) ([]*TorrentResult, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("erreur goquery: %w", err)
	}

	var results []*TorrentResult

	// The Pirate Bay utilise un tableau avec la classe "table"
	doc.Find("table#searchResult tr").Each(func(i int, tr *goquery.Selection) {
		if i == 0 {
			return // Ignorer l'en-tête
		}

		result := &TorrentResult{
			Source: models.TorrentSourcePirateBay,
		}

		// Nom et lien
		tr.Find("td div.detName a").Each(func(_ int, a *goquery.Selection) {
			result.Name = strings.TrimSpace(a.Text())
			if href, exists := a.Attr("href"); exists {
				if strings.HasPrefix(href, "http") {
					result.TorrentURL = href
				} else {
					result.TorrentURL = s.baseURL + href
				}
			}
		})

		// Magnet link
		tr.Find("td a[href^='magnet:']").Each(func(_ int, a *goquery.Selection) {
			if href, exists := a.Attr("href"); exists {
				result.MagnetURL = href
				// Extraire l'infohash
				infohashRe := regexp.MustCompile(`btih:([A-F0-9]{40})`)
				if matches := infohashRe.FindStringSubmatch(strings.ToUpper(href)); len(matches) > 1 {
					result.Infohash = matches[1]
				}
			}
		})

		// Taille
		tr.Find("td").Each(func(j int, td *goquery.Selection) {
			text := strings.TrimSpace(td.Text())
			// La taille est généralement dans une colonne spécifique
			if strings.Contains(strings.ToUpper(text), "GB") || 
			   strings.Contains(strings.ToUpper(text), "MB") ||
			   strings.Contains(strings.ToUpper(text), "KB") {
				result.Size = formatSize(text)
			}
		})

		// Seeders et Leechers (format: "SE LE" ou "S L")
		seedersLeechers := tr.Find("td").Last().Text()
		re := regexp.MustCompile(`(\d+)\s+(\d+)`)
		if matches := re.FindStringSubmatch(seedersLeechers); len(matches) >= 3 {
			if val, err := strconv.Atoi(matches[1]); err == nil {
				result.Seeders = val
			}
			if val, err := strconv.Atoi(matches[2]); err == nil {
				result.Leechers = val
			}
		}

		// Qualité depuis le nom
		qualityRe := regexp.MustCompile(`(\d+p|4K|2160p|1440p|1080p|720p|480p|360p)`)
		if matches := qualityRe.FindStringSubmatch(result.Name); len(matches) > 1 {
			result.Quality = matches[1]
		}

		// Langue depuis le nom
		langRe := regexp.MustCompile(`\b(FR|EN|ES|DE|IT|JP|CN|KR|RU|PT|NL|PL|TR|AR|HI|TH|VI|VOSTFR|MULTI)\b`)
		if matches := langRe.FindStringSubmatch(strings.ToUpper(result.Name)); len(matches) > 1 {
			result.Language = matches[1]
		}

		if result.Name != "" && (result.MagnetURL != "" || result.TorrentURL != "") {
			results = append(results, result)
		}
	})

	return results, nil
}

