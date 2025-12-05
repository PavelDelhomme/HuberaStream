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
	baseURL string
	client  *http.Client
}

func New1337xScraper() *Scraper1337x {
	return &Scraper1337x{
		baseURL: "https://www.1337x.to",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *Scraper1337x) Search(ctx context.Context, query string, category *models.Category, subCategory *models.SubCategory) ([]*TorrentResult, error) {
	searchURL := fmt.Sprintf("%s/search/%s/1/", s.baseURL, url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return s.parseSearchResults(string(body))
}

func (s *Scraper1337x) parseSearchResults(html string) ([]*TorrentResult, error) {
	var results []*TorrentResult

	// Pattern pour extraire les informations des torrents
	// Note: Ceci est une implémentation basique, à améliorer avec un vrai parser HTML
	re := regexp.MustCompile(`<a href="(/torrent/[^"]+)"[^>]*>([^<]+)</a>`)
	matches := re.FindAllStringSubmatch(html, -1)

	for i, match := range matches {
		if i >= 20 { // Limiter à 20 résultats
			break
		}

		if len(match) >= 3 {
			result := &TorrentResult{
				Name:      strings.TrimSpace(match[2]),
				TorrentURL: s.baseURL + match[1],
				Source:    models.TorrentSource1337x,
			}
			results = append(results, result)
		}
	}

	return results, nil
}

func (s *Scraper1337x) GetTorrentDetails(ctx context.Context, torrentURL string) (*TorrentResult, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", torrentURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := s.client.Do(req)
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

func (s *Scraper1337x) parseTorrentDetails(html, torrentURL string) (*TorrentResult, error) {
	result := &TorrentResult{
		TorrentURL: torrentURL,
		Source:     models.TorrentSource1337x,
	}

	// Extraire le magnet link
	magnetRe := regexp.MustCompile(`magnet:\?[^"'\s]+`)
	if matches := magnetRe.FindString(html); matches != "" {
		result.MagnetURL = matches
	}

	// Extraire les seeders/leechers
	seedersRe := regexp.MustCompile(`<span class="seeds">(\d+)</span>`)
	if matches := seedersRe.FindStringSubmatch(html); len(matches) > 1 {
		if val, err := strconv.Atoi(matches[1]); err == nil {
			result.Seeders = val
		}
	}

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

	return result, nil
}

