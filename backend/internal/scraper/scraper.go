package scraper

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"streammake/internal/models"
)

type Scraper interface {
	Search(ctx context.Context, query string, category *models.Category, subCategory *models.SubCategory) ([]*TorrentResult, error)
	GetTorrentDetails(ctx context.Context, url string) (*TorrentResult, error)
}

// ScraperWithCustomURL est une interface pour les scrapers qui supportent les URLs personnalisées
type ScraperWithCustomURL interface {
	Scraper
	SetBaseURL(url string)
	FetchAndParseListing(ctx context.Context, url string, languageFilter string) ([]*TorrentResult, error)
}

type TorrentResult struct {
	Name          string
	MagnetURL     string
	TorrentURL    string
	TorrentFileURL string // URL du fichier .torrent
	Infohash     string  // Infohash du torrent
	Size          int64
	Seeders       int
	Leechers      int
	UploadDate    string
	Category      string
	Source        models.TorrentSource
	Quality       string
	Language      string
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
	case models.TorrentSourcePirateBay:
		return NewPirateBayScraper(), nil
	case models.TorrentSourceTorrentDownload:
		return NewTorrentDownloadScraper(), nil
	default:
		return nil, fmt.Errorf("source non supportée: %s", source)
	}
}

func formatSize(sizeStr string) int64 {
	if sizeStr == "" {
		return 0
	}
	
	// Nettoyer la chaîne : remplacer les virgules par des points et supprimer les espaces
	sizeStr = strings.ReplaceAll(sizeStr, ",", ".")
	sizeStr = strings.TrimSpace(sizeStr)
	
	// Supprimer les caractères non numériques sauf les points et les unités
	// Exemples: "709.23 Mo", "1,47 Go", "1.5 GB", "8 MB", "1 B"
	
	// Pattern amélioré pour capturer: "709.23 Mo", "1,47 Go", "1.5 GB", "8 MB", "1 B", etc.
	// Essayer d'abord avec unité explicite
	re := regexp.MustCompile(`([\d.]+)\s*([KMGT]?i?[OB]?)`)
	matches := re.FindStringSubmatch(strings.ToUpper(sizeStr))
	
	// Si pas de match, essayer de trouver juste un nombre suivi d'une unité dans le texte
	if len(matches) < 2 {
		// Pattern alternatif: chercher n'importe quel nombre suivi d'une unité
		re2 := regexp.MustCompile(`(\d+\.?\d*)\s*(KB|MB|GB|TB|KIB|MIB|GIB|TIB|KO|MO|GO|TO|K|M|G|T|B)`)
		matches = re2.FindStringSubmatch(strings.ToUpper(sizeStr))
	}
	
	if len(matches) < 2 {
		log.Printf("[formatSize] ⚠️ Impossible de parser la taille: '%s'", sizeStr)
		return 0
	}

	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		log.Printf("[formatSize] ⚠️ Erreur de conversion numérique: '%s' -> %v", matches[1], err)
		return 0
	}

	unit := ""
	if len(matches) >= 3 {
		unit = strings.TrimSpace(matches[2])
	}
	
	multiplier := int64(1)

	switch unit {
	case "KB", "KIB", "KO", "K":
		multiplier = 1024
	case "MB", "MIB", "MO", "M":
		multiplier = 1024 * 1024
	case "GB", "GIB", "GO", "G":
		multiplier = 1024 * 1024 * 1024
	case "TB", "TIB", "TO", "T":
		multiplier = 1024 * 1024 * 1024 * 1024
	case "B", "":
		// Si pas d'unité ou "B" (bytes), assumer bytes
		multiplier = 1
	default:
		// Si unité inconnue, assumer bytes
		log.Printf("[formatSize] ⚠️ Unité inconnue: '%s', assumant bytes", unit)
		multiplier = 1
	}

	result := int64(value * float64(multiplier))
	if result == 0 && value > 0 {
		log.Printf("[formatSize] ⚠️ Résultat 0 pour valeur %f et unité %s (multiplier: %d)", value, unit, multiplier)
	}
	
	return result
}
