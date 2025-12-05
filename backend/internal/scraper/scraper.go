package scraper

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"streammake/internal/models"
)

type Scraper interface {
	Search(ctx context.Context, query string, category *models.Category, subCategory *models.SubCategory) ([]*TorrentResult, error)
	GetTorrentDetails(ctx context.Context, url string) (*TorrentResult, error)
}

type TorrentResult struct {
	Name        string
	MagnetURL   string
	TorrentURL  string
	Size        int64
	Seeders     int
	Leechers    int
	UploadDate  string
	Category    string
	Source      models.TorrentSource
	Quality     string
	Language    string
}

type ScraperFactory struct{}

func NewScraper(source models.TorrentSource) (Scraper, error) {
	switch source {
	case models.TorrentSource1337x:
		return New1337xScraper(), nil
	case models.TorrentSourceRARBG:
		return NewRARBGScraper(), nil
	case models.TorrentSourceYTS:
		return NewYTSScraper(), nil
	case models.TorrentSourceNyaa:
		return NewNyaaScraper(), nil
	default:
		return nil, fmt.Errorf("source non supportée: %s", source)
	}
}

func formatSize(sizeStr string) int64 {
	// Convertir des chaînes comme "1.5 GB" en bytes
	re := regexp.MustCompile(`([\d.]+)\s*([KMGT]?i?B)`)
	matches := re.FindStringSubmatch(strings.ToUpper(sizeStr))
	if len(matches) != 3 {
		return 0
	}

	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0
	}

	unit := matches[2]
	multiplier := int64(1)

	switch unit {
	case "KB", "KIB":
		multiplier = 1024
	case "MB", "MIB":
		multiplier = 1024 * 1024
	case "GB", "GIB":
		multiplier = 1024 * 1024 * 1024
	case "TB", "TIB":
		multiplier = 1024 * 1024 * 1024 * 1024
	}

	return int64(value * float64(multiplier))
}

