package scraper

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"streammake/internal/models"
)

// CrawlerService gère le crawling automatique de 1337x
type CrawlerService struct {
	scraper     *Scraper1337x
	categories  []Category1337x
	languages   []string
	running     bool
	mu          sync.Mutex
	stopChan    chan struct{}
	resultsChan chan []*TorrentResult
}

// NewCrawlerService crée un nouveau service de crawling
func NewCrawlerService() *CrawlerService {
	return &CrawlerService{
		scraper:     New1337xScraper(),
		categories:  []Category1337x{
			CategoryPopularMovies,
			CategoryTV,
			CategoryMovieLibrary,
			CategoryAnime,
			CategoryDocumentaries,
			CategoryMovies,
		},
		languages:   []string{"FR", "EN", "ES", "DE", "IT", "JP"},
		stopChan:    make(chan struct{}),
		resultsChan: make(chan []*TorrentResult, 100),
	}
}

// Start démarre le crawling automatique
func (cs *CrawlerService) Start(ctx context.Context, interval time.Duration) error {
	cs.mu.Lock()
	if cs.running {
		cs.mu.Unlock()
		return fmt.Errorf("crawler déjà en cours d'exécution")
	}
	cs.running = true
	cs.mu.Unlock()

	// Récupérer les proxies si activé
	if cs.scraper.useProxy {
		if err := cs.scraper.FetchProxies(ctx); err != nil {
			log.Printf("Erreur lors de la récupération des proxies: %v", err)
		}
	}

	go cs.crawlLoop(ctx, interval)
	return nil
}

// Stop arrête le crawling
func (cs *CrawlerService) Stop() {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if !cs.running {
		return
	}
	cs.running = false
	close(cs.stopChan)
}

// IsRunning retourne si le crawler est en cours d'exécution
func (cs *CrawlerService) IsRunning() bool {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.running
}

// crawlLoop boucle principale de crawling
func (cs *CrawlerService) crawlLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-cs.stopChan:
			return
		case <-ticker.C:
			cs.crawlAllCategories(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// crawlAllCategories crawl toutes les catégories
func (cs *CrawlerService) crawlAllCategories(ctx context.Context) {
	for _, category := range cs.categories {
		// Crawler pour chaque langue
		for _, lang := range cs.languages {
			select {
			case <-cs.stopChan:
				return
			case <-ctx.Done():
				return
			default:
				results, err := cs.scraper.BrowseCategory(ctx, category, 1, lang)
				if err != nil {
					log.Printf("Erreur lors du crawling de %s (lang: %s): %v", category, lang, err)
					continue
				}

				if len(results) > 0 {
					select {
					case cs.resultsChan <- results:
						log.Printf("Crawlé %d torrents de %s (lang: %s)", len(results), category, lang)
					default:
						log.Printf("Channel plein, résultats ignorés")
					}
				}

				// Délai entre les requêtes pour éviter le rate limiting
				time.Sleep(2 * time.Second)
			}
		}
	}
}

// GetResults retourne les résultats du crawling
func (cs *CrawlerService) GetResults() <-chan []*TorrentResult {
	return cs.resultsChan
}

// SetCategories définit les catégories à crawler
func (cs *CrawlerService) SetCategories(categories []Category1337x) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.categories = categories
}

// SetLanguages définit les langues à crawler
func (cs *CrawlerService) SetLanguages(languages []string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.languages = languages
}

// EnableProxy active l'utilisation de proxies
func (cs *CrawlerService) EnableProxy(enable bool) {
	cs.scraper.SetUseProxy(enable)
}

