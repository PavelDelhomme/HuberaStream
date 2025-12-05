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

type NyaaScraper struct {
	baseURL string
	client  *http.Client
}

func NewNyaaScraper() *NyaaScraper {
	return &NyaaScraper{
		baseURL: "https://nyaa.si",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *NyaaScraper) Search(ctx context.Context, query string, category *models.Category, subCategory *models.SubCategory) ([]*TorrentResult, error) {
	searchURL := fmt.Sprintf("%s/?q=%s&s=seeders&o=desc", s.baseURL, url.QueryEscape(query))

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

func (s *NyaaScraper) parseSearchResults(html string) ([]*TorrentResult, error) {
	var results []*TorrentResult

	// Parser le tableau de résultats Nyaa
	// Format HTML de Nyaa avec des lignes <tr>
	rowRe := regexp.MustCompile(`<tr[^>]*>([\s\S]*?)</tr>`)
	rows := rowRe.FindAllString(html, -1)

	for i, row := range rows {
		if i == 0 { // Skip header
			continue
		}
		if i > 20 { // Limiter à 20 résultats
			break
		}

		result := s.parseRow(row)
		if result != nil {
			results = append(results, result)
		}
	}

	return results, nil
}

func (s *NyaaScraper) parseRow(row string) *TorrentResult {
	// Extraire le titre et le lien
	titleRe := regexp.MustCompile(`<a[^>]*href="(/view/[^"]+)"[^>]*>([^<]+)</a>`)
	titleMatch := titleRe.FindStringSubmatch(row)
	if len(titleMatch) < 3 {
		return nil
	}

	result := &TorrentResult{
		Name:      strings.TrimSpace(titleMatch[2]),
		TorrentURL: s.baseURL + titleMatch[1],
		Source:    models.TorrentSourceNyaa,
	}

	// Extraire le magnet
	magnetRe := regexp.MustCompile(`href="(magnet:[^"]+)"`)
	if matches := magnetRe.FindStringSubmatch(row); len(matches) > 1 {
		result.MagnetURL = matches[1]
	}

	// Extraire seeders/leechers
	seedersRe := regexp.MustCompile(`<td[^>]*>(\d+)</td>`)
	seedersMatches := seedersRe.FindAllStringSubmatch(row, -1)
	if len(seedersMatches) >= 1 {
		if val, err := strconv.Atoi(seedersMatches[0][1]); err == nil {
			result.Seeders = val
		}
	}
	if len(seedersMatches) >= 2 {
		if val, err := strconv.Atoi(seedersMatches[1][1]); err == nil {
			result.Leechers = val
		}
	}

	// Extraire la taille
	sizeRe := regexp.MustCompile(`<td[^>]*>([\d.]+\s*[KMGT]?i?B)</td>`)
	if matches := sizeRe.FindStringSubmatch(row); len(matches) > 1 {
		result.Size = formatSize(matches[1])
	}

	return result
}

func (s *NyaaScraper) GetTorrentDetails(ctx context.Context, torrentURL string) (*TorrentResult, error) {
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

	// Extraire le magnet link de la page
	magnetRe := regexp.MustCompile(`href="(magnet:[^"]+)"`)
	matches := magnetRe.FindString(string(body))

	result := &TorrentResult{
		TorrentURL: torrentURL,
		Source:     models.TorrentSourceNyaa,
	}

	if matches != "" {
		result.MagnetURL = strings.Trim(matches, `href="`)
		result.MagnetURL = strings.Trim(result.MagnetURL, `"`)
	}

	return result, nil
}

