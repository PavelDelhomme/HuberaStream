package scraper

import (
	"compress/gzip"
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

type Scraper1337x struct {
	baseURL      string
	client       *http.Client
	proxyManager *ProxyManager
	useProxy     bool
}

// Category représente une catégorie 1337x
type Category1337x string

const (
	CategoryPopularMovies Category1337x = "popular-movies"
	CategoryTV            Category1337x = "TV"
	CategoryMovieLibrary  Category1337x = "movie-library"
	CategoryAnime         Category1337x = "Anime"
	CategoryDocumentaries Category1337x = "Documentaries"
	CategoryMovies        Category1337x = "Movies"
)

func New1337xScraper() *Scraper1337x {
	pm := NewProxyManager()
	return &Scraper1337x{
		baseURL:      "https://1337x.quest", // Utiliser 1337x.quest par défaut
		proxyManager: pm,
		useProxy:     false, // Désactivé par défaut, activer si nécessaire
		client: &http.Client{
			Timeout: 120 * time.Second, // Augmenté à 120 secondes pour éviter les timeouts
		},
	}
}

// SetBaseURL permet de définir une URL de base personnalisée
func (s *Scraper1337x) SetBaseURL(url string) {
	s.baseURL = url
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
	// Utiliser l'URL de recherche lmsearch de 1337x.quest
	searchURL := fmt.Sprintf("%s/lmsearch?q=%s&category=lmsearch", s.baseURL, url.QueryEscape(query))
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

// FetchAndParseListing récupère et parse une URL de listing personnalisée (exporté pour auto_download)
func (s *Scraper1337x) FetchAndParseListing(ctx context.Context, url string, languageFilter string) ([]*TorrentResult, error) {
	return s.fetchAndParse(ctx, url, languageFilter)
}

// fetchAndParse récupère et parse une URL
func (s *Scraper1337x) fetchAndParse(ctx context.Context, url string, languageFilter string) ([]*TorrentResult, error) {
	// IGNORER complètement le contexte parent et créer un contexte indépendant
	// Le contexte parent peut être annulé par la requête HTTP parente, mais on veut
	// que cette requête HTTP se termine indépendamment
	reqCtx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	log.Printf("[1337x] Début de fetchAndParse pour URL: %s", url)
	log.Printf("[1337x] Contexte indépendant créé, erreur: %v", reqCtx.Err())

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

	log.Printf("[1337x] Création de la requête HTTP...")
	req, err := http.NewRequestWithContext(reqCtx, "GET", url, nil)
	if err != nil {
		log.Printf("[1337x] Erreur lors de la création de la requête: %v", err)
		return nil, fmt.Errorf("erreur lors de la création de la requête: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	// Ne pas demander de compression Brotli car Go standard ne la supporte pas nativement
	// On accepte seulement gzip et deflate que Go peut décompresser automatiquement
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	log.Printf("[1337x] Envoi de la requête HTTP...")
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[1337x] Erreur lors de la requête HTTP: %v", err)
		log.Printf("[1337x] Erreur du contexte reqCtx: %v", reqCtx.Err())
		return nil, fmt.Errorf("erreur lors de la requête: %w", err)
	}
	log.Printf("[1337x] Requête HTTP réussie, status: %d", resp.StatusCode)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("code de statut non OK: %d", resp.StatusCode)
	}

	// Vérifier si le contenu est compressé
	var reader io.Reader = resp.Body
	contentEncoding := resp.Header.Get("Content-Encoding")
	log.Printf("[1337x] Content-Encoding: %s", contentEncoding)
	
	// Si c'est du Brotli, on ne peut pas le décompresser avec Go standard
	// On doit refaire la requête sans accepter de compression
	if contentEncoding == "br" || strings.Contains(contentEncoding, "br") {
		log.Printf("[1337x] ⚠️  Contenu compressé en Brotli (non supporté), nouvelle requête sans compression...")
		resp.Body.Close()
		
		// Faire une nouvelle requête sans accepter de compression
		req2, err := http.NewRequestWithContext(reqCtx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("erreur lors de la création de la requête: %w", err)
		}
		req2.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
		req2.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
		req2.Header.Set("Accept-Language", "en-US,en;q=0.5")
		// Ne pas accepter de compression du tout
		req2.Header.Set("Accept-Encoding", "identity")
		req2.Header.Set("Connection", "keep-alive")
		req2.Header.Set("Upgrade-Insecure-Requests", "1")
		
		resp2, err := client.Do(req2)
		if err != nil {
			return nil, fmt.Errorf("erreur lors de la requête (sans compression): %w", err)
		}
		defer resp2.Body.Close()
		
		if resp2.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("code de statut non OK: %d", resp2.StatusCode)
		}
		
		contentEncoding = resp2.Header.Get("Content-Encoding")
		log.Printf("[1337x] Content-Encoding (nouvelle requête): %s", contentEncoding)
		reader = resp2.Body
	}
	
	if contentEncoding == "gzip" || strings.Contains(contentEncoding, "gzip") {
		log.Printf("[1337x] Décompression gzip...")
		gzipReader, err := gzip.NewReader(reader)
		if err != nil {
			return nil, fmt.Errorf("erreur lors de la création du reader gzip: %w", err)
		}
		defer gzipReader.Close()
		reader = gzipReader
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la lecture du corps: %w", err)
	}

	htmlContent := string(body)
	log.Printf("[1337x] Taille du HTML reçu (après décompression): %d bytes", len(htmlContent))
	
	// Sauvegarder un extrait du HTML pour debug (premiers 2000 caractères)
	if len(htmlContent) > 2000 {
		log.Printf("[1337x] Extrait HTML (premiers 2000 chars): %s...", htmlContent[:2000])
	} else {
		log.Printf("[1337x] HTML complet: %s", htmlContent)
	}

	results, err := s.parseSearchResults(htmlContent, languageFilter)
	if err != nil {
		log.Printf("[1337x] Erreur lors du parsing: %v", err)
		return nil, err
	}
	
	log.Printf("[1337x] Parsing terminé, %d résultats trouvés", len(results))
	return results, nil
}

// parseSearchResults parse les résultats de recherche avec goquery (plus robuste)
func (s *Scraper1337x) parseSearchResults(html string, languageFilter string) ([]*TorrentResult, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		log.Printf("[1337x] Erreur goquery, fallback regex: %v", err)
		// Fallback sur regex si goquery échoue
		return s.parseSearchResultsRegex(html, languageFilter)
	}

	var results []*TorrentResult

	// Essayer plusieurs sélecteurs pour trouver les résultats
	// Sélecteur 1: table.table-list tbody tr (structure standard 1337x)
	count1 := 0
	// Essayer plusieurs variantes de sélecteurs pour le tableau
	tableSelector := doc.Find("table.table-list tbody tr")
	if tableSelector.Length() == 0 {
		// Essayer sans la classe table-list
		tableSelector = doc.Find("table tbody tr")
	}
	if tableSelector.Length() == 0 {
		// Essayer avec d'autres classes
		tableSelector = doc.Find("table.tl tbody tr")
	}
	if tableSelector.Length() == 0 {
		// Essayer juste les tr dans un tableau
		tableSelector = doc.Find("tbody tr")
	}
	if tableSelector.Length() == 0 {
		// Dernier recours : tous les tr qui contiennent des liens
		tableSelector = doc.Find("tr")
	}
	
	log.Printf("[1337x] Nombre de lignes trouvées avec le sélecteur: %d", tableSelector.Length())
	
	// Si aucun résultat avec les sélecteurs de table, essayer de trouver tous les liens vers des torrents
	if tableSelector.Length() == 0 {
		log.Printf("[1337x] Aucune ligne de tableau trouvée, recherche de liens torrents dans tout le document...")
		// Chercher tous les liens qui ressemblent à des liens de torrents
		doc.Find("a[href]").Each(func(_ int, a *goquery.Selection) {
			href, exists := a.Attr("href")
			if !exists {
				return
			}
			
			// Vérifier si c'est un lien vers un torrent (contient .html, /torrent/, ou des patterns similaires)
			// Ignorer les liens de navigation, CSS, JS, etc.
			if (strings.Contains(href, ".html") && !strings.Contains(href, "javascript:") && !strings.Contains(href, "#")) ||
			   (strings.Contains(href, "/torrent/") && !strings.Contains(href, "javascript:")) ||
			   (strings.HasPrefix(href, "/") && len(href) > 15 && strings.Contains(href, "-") && 
			    !strings.Contains(href, "javascript:") && !strings.Contains(href, "#") && 
			    !strings.Contains(href, "/css/") && !strings.Contains(href, "/js/") &&
			    !strings.Contains(href, "/category") && !strings.Contains(href, "/home") &&
			    !strings.Contains(href, "/about") && !strings.Contains(href, "/contact")) {
				text := strings.TrimSpace(a.Text())
				// Ignorer les liens trop courts ou qui sont des catégories/navigation
				if len(text) > 10 && text != "" && 
				   !strings.EqualFold(text, "home") && !strings.EqualFold(text, "movies") && 
				   !strings.EqualFold(text, "tv") && !strings.EqualFold(text, "games") && 
				   !strings.EqualFold(text, "music") && !strings.EqualFold(text, "anime") &&
				   !strings.EqualFold(text, "documentaries") && !strings.EqualFold(text, "other") &&
				   !strings.EqualFold(text, "xxx") && !strings.EqualFold(text, "applications") &&
				   !strings.EqualFold(text, "top 100") && !strings.EqualFold(text, "trending") {
					result := &TorrentResult{
						Source: models.TorrentSource1337x,
						Size:   0,
					}
					result.Name = text
					if strings.HasPrefix(href, "http") {
						result.TorrentURL = href
					} else {
						result.TorrentURL = s.baseURL + href
					}
					// Extraire qualité et langue depuis le nom
					qualityRe := regexp.MustCompile(`(\d+p|4K|2160p|1440p|1080p|720p|480p|360p)`)
					if matches := qualityRe.FindStringSubmatch(result.Name); len(matches) > 1 {
						result.Quality = matches[1]
					}
					langRe := regexp.MustCompile(`\b(FR|EN|ES|DE|IT|JP|CN|KR|RU|PT|NL|PL|TR|AR|HI|TH|VI|VOSTFR|MULTI)\b`)
					if matches := langRe.FindStringSubmatch(strings.ToUpper(result.Name)); len(matches) > 1 {
						result.Language = matches[1]
					}
					if result.Name != "" && result.TorrentURL != "" {
						if languageFilter == "" || result.Language == "" || strings.EqualFold(result.Language, languageFilter) {
							results = append(results, result)
						}
					}
				}
			}
		})
		log.Printf("[1337x] Parsing par liens directs: %d résultats trouvés", len(results))
		if len(results) > 0 {
			return results, nil
		}
	}
	
	tableSelector.Each(func(i int, s_html *goquery.Selection) {
		count1++
		if i >= 50 { // Limiter à 50 résultats
			return
		}

		result := &TorrentResult{
			Source: models.TorrentSource1337x,
			Size:   0, // Initialiser explicitement à 0
		}

		// Name et TorrentURL - essayer plusieurs sélecteurs
		// Sélecteur 1: td.coll-1.name a:nth-child(2) (lien principal)
		s_html.Find("td.coll-1.name a:nth-child(2)").Each(func(_ int, a *goquery.Selection) {
			result.Name = strings.TrimSpace(a.Text())
			if href, exists := a.Attr("href"); exists {
				if strings.HasPrefix(href, "http") {
					result.TorrentURL = href
				} else {
					result.TorrentURL = s.baseURL + href
				}
			}
		})

		// Si pas trouvé, essayer d'autres sélecteurs
		if result.Name == "" {
			s_html.Find("td.name a, td.coll-1 a, a[href*='.html'], a[href*='/torrent/']").Each(func(_ int, a *goquery.Selection) {
				if result.Name == "" {
					text := strings.TrimSpace(a.Text())
					href, hasHref := a.Attr("href")
					// Ignorer les liens de navigation/catégories
					if text != "" && len(text) > 10 && hasHref && 
					   (strings.Contains(href, ".html") || strings.Contains(href, "/torrent/")) &&
					   !strings.Contains(strings.ToLower(text), "home") &&
					   !strings.Contains(strings.ToLower(text), "movies") &&
					   !strings.Contains(strings.ToLower(text), "tv") {
						result.Name = text
						if strings.HasPrefix(href, "http") {
							result.TorrentURL = href
						} else {
							result.TorrentURL = s.baseURL + href
						}
					}
				}
			})
		}

		// Seeders
		s_html.Find("td.coll-2.seeds, td.seeds, span.seeds").Each(func(_ int, td *goquery.Selection) {
			if val, err := strconv.Atoi(strings.TrimSpace(td.Text())); err == nil {
				result.Seeders = val
			}
		})

		// Leechers
		s_html.Find("td.coll-3.leeches, td.leeches, span.leeches").Each(func(_ int, td *goquery.Selection) {
			if val, err := strconv.Atoi(strings.TrimSpace(td.Text())); err == nil {
				result.Leechers = val
			}
		})

		// UploadDate
		s_html.Find("td.coll-date, td.date").Each(func(_ int, td *goquery.Selection) {
			result.UploadDate = strings.TrimSpace(td.Text())
		})

		// Size (Taille) - par classe CSS d'abord
		s_html.Find("td.coll-4.size, td.size").Each(func(_ int, td *goquery.Selection) {
			sizeText := strings.TrimSpace(td.Text())
			if sizeText != "" {
				parsedSize := formatSize(sizeText)
				if parsedSize > 0 {
					result.Size = parsedSize
					log.Printf("[1337x] Taille extraite (sélecteur 1, classe CSS): %s -> %d bytes", sizeText, parsedSize)
				} else {
					log.Printf("[1337x] ⚠️ Taille non parsée (sélecteur 1): '%s' -> 0", sizeText)
				}
			}
		})
		
		// Si la taille n'a toujours pas été trouvée, essayer de la trouver dans toutes les colonnes
		if result.Size == 0 {
			s_html.Find("td").Each(func(_ int, td *goquery.Selection) {
				text := strings.TrimSpace(td.Text())
				// Vérifier si c'est une taille (contient des unités comme GB, MB, etc.)
				if strings.Contains(strings.ToUpper(text), "GB") || 
				   strings.Contains(strings.ToUpper(text), "MB") || 
				   strings.Contains(strings.ToUpper(text), "GO") || 
				   strings.Contains(strings.ToUpper(text), "MO") ||
				   strings.Contains(strings.ToUpper(text), "TB") ||
				   strings.Contains(strings.ToUpper(text), "KB") {
					parsedSize := formatSize(text)
					if parsedSize > 0 {
						result.Size = parsedSize
						log.Printf("[1337x] Taille extraite (sélecteur 1, parsing intelligent): %s -> %d bytes", text, parsedSize)
					}
				}
			})
		}

		// Quality et Language from Name
		qualityRe := regexp.MustCompile(`(\d+p|4K|2160p|1440p|1080p|720p|480p|360p)`)
		if matches := qualityRe.FindStringSubmatch(result.Name); len(matches) > 1 {
			result.Quality = matches[1]
		}
		langRe := regexp.MustCompile(`\b(FR|EN|ES|DE|IT|JP|CN|KR|RU|PT|NL|PL|TR|AR|HI|TH|VI|VOSTFR|MULTI)\b`)
		if matches := langRe.FindStringSubmatch(strings.ToUpper(result.Name)); len(matches) > 1 {
			result.Language = matches[1]
		}

		// Category
		s_html.Find("td.coll-1.name a:nth-child(1)").Each(func(_ int, a *goquery.Selection) {
			if href, exists := a.Attr("href"); exists {
				if catMatches := regexp.MustCompile(`/cat/([^/]+)/`).FindStringSubmatch(href); len(catMatches) > 1 {
					result.Category = catMatches[1]
				}
			}
		})

		if result.Name != "" && result.TorrentURL != "" {
			if languageFilter != "" && result.Language != "" && !strings.EqualFold(result.Language, languageFilter) {
				return // Skip if language filter doesn't match
			}
			results = append(results, result)
		}
	})

	log.Printf("[1337x] Parsing goquery (sélecteur 1): %d lignes trouvées avec 'table.table-list tbody tr', %d résultats extraits", count1, len(results))

	// Si aucun résultat avec le premier sélecteur, essayer d'autres sélecteurs
	if len(results) == 0 {
		// Sélecteur 2: table tbody tr (sans classe spécifique)
		count2 := 0
		// Essayer plusieurs variantes de sélecteurs
		tableSelector2 := doc.Find("table tbody tr")
		if tableSelector2.Length() == 0 {
			tableSelector2 = doc.Find("table.tl tbody tr")
		}
		if tableSelector2.Length() == 0 {
			tableSelector2 = doc.Find("tbody tr")
		}
		if tableSelector2.Length() == 0 {
			// Essayer de trouver n'importe quel tr qui contient des liens
			tableSelector2 = doc.Find("tr")
		}
		
		tableSelector2.Each(func(i int, s_html *goquery.Selection) {
			count2++
			if i >= 50 {
				return
			}
			
			// Vérifier si cette ligne contient des liens (probablement un résultat)
			hasLink := false
			s_html.Find("a").Each(func(_ int, a *goquery.Selection) {
				if href, exists := a.Attr("href"); exists && (strings.Contains(href, "/torrent/") || strings.Contains(href, ".html") || strings.Contains(href, "/watch-")) {
					hasLink = true
				}
			})
			
			if !hasLink {
				return // Ignorer les lignes sans liens
			}
			
			result := &TorrentResult{
				Source: models.TorrentSource1337x,
			}

			// Extraire le nom et l'URL depuis n'importe quel lien
			s_html.Find("a").Each(func(_ int, a *goquery.Selection) {
				if result.Name == "" {
					text := strings.TrimSpace(a.Text())
					href, hasHref := a.Attr("href")
					if text != "" && len(text) > 3 && hasHref && (strings.Contains(href, "/torrent/") || strings.Contains(href, ".html") || strings.Contains(href, "/watch-")) {
						result.Name = text
						if strings.HasPrefix(href, "http") {
							result.TorrentURL = href
						} else {
							result.TorrentURL = s.baseURL + href
						}
					}
				}
			})

		// Extraire les données depuis les colonnes en utilisant les classes CSS
		// Structure attendue: Nom | Date | Taille | Graine (Seeders) | Sangsue (Leechers)
		
		// Seeders (Graine) - par classe CSS d'abord
		s_html.Find("td.coll-2.seeds, td.seeds, span.seeds").Each(func(_ int, td *goquery.Selection) {
			if val, err := strconv.Atoi(strings.TrimSpace(td.Text())); err == nil {
				result.Seeders = val
			}
		})
		
		// Leechers (Sangsue) - par classe CSS d'abord
		s_html.Find("td.coll-3.leeches, td.leeches, span.leeches").Each(func(_ int, td *goquery.Selection) {
			if val, err := strconv.Atoi(strings.TrimSpace(td.Text())); err == nil {
				result.Leechers = val
			}
		})
		
		// Size (Taille) - par classe CSS d'abord
		s_html.Find("td.coll-4.size, td.size").Each(func(_ int, td *goquery.Selection) {
			sizeText := strings.TrimSpace(td.Text())
			if sizeText != "" {
				parsedSize := formatSize(sizeText)
				if parsedSize > 0 {
					result.Size = parsedSize
					log.Printf("[1337x] Taille extraite (classe CSS): %s -> %d bytes", sizeText, parsedSize)
				}
			}
		})
		
		// UploadDate (Ajouté) - par classe CSS d'abord
		s_html.Find("td.coll-date, td.date").Each(func(_ int, td *goquery.Selection) {
			result.UploadDate = strings.TrimSpace(td.Text())
		})
		
		// Si les données ne sont pas trouvées par classe, essayer par position dans le tableau
		if result.Seeders == 0 && result.Leechers == 0 && result.Size == 0 {
			// Extraire toutes les colonnes et mapper par position
			cols := []string{}
			s_html.Find("td").Each(func(_ int, td *goquery.Selection) {
				text := strings.TrimSpace(td.Text())
				if text != "" {
					cols = append(cols, text)
				}
			})
			
			// Structure: [0]=Nom, [1]=Date, [2]=Taille, [3]=Graine, [4]=Sangsue
			// Mais peut varier, donc on cherche les patterns
			for _, col := range cols {
				// Date: contient "il y a", "mois", "an", "jour"
				if strings.Contains(strings.ToLower(col), "il y a") || 
				   strings.Contains(strings.ToLower(col), "mois") || 
				   strings.Contains(strings.ToLower(col), "an") ||
				   strings.Contains(strings.ToLower(col), "jour") {
					if result.UploadDate == "" {
						result.UploadDate = col
					}
				}
				// Taille: contient "Go", "Mo", "GB", "MB", etc.
				if strings.Contains(strings.ToUpper(col), "GO") || 
				   strings.Contains(strings.ToUpper(col), "MO") ||
				   strings.Contains(strings.ToUpper(col), "GB") ||
				   strings.Contains(strings.ToUpper(col), "MB") ||
				   strings.Contains(strings.ToUpper(col), "TB") ||
				   strings.Contains(strings.ToUpper(col), "KB") {
					if result.Size == 0 {
						parsedSize := formatSize(col)
						if parsedSize > 0 {
							result.Size = parsedSize
							log.Printf("[1337x] Taille extraite (parsing intelligent): %s -> %d bytes", col, parsedSize)
						}
					}
				}
				// Seeders/Leechers: nombres simples
				if val, err := strconv.Atoi(col); err == nil {
					// Si on a déjà un seeders, le suivant est probablement leechers
					if result.Seeders == 0 {
						result.Seeders = val
					} else if result.Leechers == 0 {
						result.Leechers = val
					}
				}
			}
		}

			// Quality et Language from Name
			qualityRe := regexp.MustCompile(`(\d+p|4K|2160p|1440p|1080p|720p|480p|360p)`)
			if matches := qualityRe.FindStringSubmatch(result.Name); len(matches) > 1 {
				result.Quality = matches[1]
			}
			langRe := regexp.MustCompile(`\b(FR|EN|ES|DE|IT|JP|CN|KR|RU|PT|NL|PL|TR|AR|HI|TH|VI|VOSTFR|MULTI)\b`)
			if matches := langRe.FindStringSubmatch(strings.ToUpper(result.Name)); len(matches) > 1 {
				result.Language = matches[1]
			}

			if result.Name != "" && result.TorrentURL != "" {
				if languageFilter != "" && result.Language != "" && !strings.EqualFold(result.Language, languageFilter) {
					return
				}
				results = append(results, result)
			}
		})
		log.Printf("[1337x] Parsing goquery (sélecteur 2): %d lignes trouvées avec 'table tbody tr', %d résultats extraits", count2, len(results))
	}

	// Si toujours aucun résultat, essayer le fallback regex
	if len(results) == 0 {
		log.Printf("[1337x] Aucun résultat avec goquery, utilisation du fallback regex")
		return s.parseSearchResultsRegex(html, languageFilter)
	}

	log.Printf("[1337x] Total de %d résultats trouvés après parsing", len(results))
	return results, nil
}

// parseSearchResultsRegex fallback avec regex si goquery échoue
func (s *Scraper1337x) parseSearchResultsRegex(html string, languageFilter string) ([]*TorrentResult, error) {
	var results []*TorrentResult

	// Parser amélioré utilisant la structure HTML de 1337x
	// Essayer plusieurs patterns pour trouver les lignes de résultats
	rowRe1 := regexp.MustCompile(`<tr[^>]*class="[^"]*table-list-wrap[^"]*"[^>]*>([\s\S]*?)</tr>`)
	rowRe2 := regexp.MustCompile(`<tbody[^>]*>([\s\S]*?)</tbody>`)
	// Pattern amélioré pour trouver les liens vers les pages de détails (format: /nom-du-torrent-12345678.html)
	linkRe := regexp.MustCompile(`<a[^>]*href="(/torrent/[^"]+)"[^>]*>([^<]+)</a>`)
	linkRe2 := regexp.MustCompile(`<a[^>]*href="(/(?:[^/]+-)+[0-9]+\.html)"[^>]*>([^<]+)</a>`)
	linkRe3 := regexp.MustCompile(`<a[^>]*href="(/[^"]*\.html)"[^>]*>([^<]+)</a>`)

	var rows []string

	rows1 := rowRe1.FindAllString(html, -1)
	if len(rows1) > 0 {
		rows = rows1
	} else {
		tbodyMatches := rowRe2.FindAllStringSubmatch(html, -1)
		if len(tbodyMatches) > 0 {
			trRe := regexp.MustCompile(`<tr[^>]*>([\s\S]*?)</tr>`)
			rows = trRe.FindAllString(tbodyMatches[0][1], -1)
		} else {
			// Essayer de trouver tous les liens possibles
			allLinks := linkRe.FindAllStringSubmatch(html, -1)
			if len(allLinks) == 0 {
				allLinks = linkRe2.FindAllStringSubmatch(html, -1)
			}
			if len(allLinks) == 0 {
				allLinks = linkRe3.FindAllStringSubmatch(html, -1)
			}
			
			for _, link := range allLinks {
				if len(link) >= 3 {
					href := link[1]
					name := strings.TrimSpace(link[2])
					// Filtrer les liens qui ne sont pas des torrents (ex: liens de navigation, CSS, JS)
					if name != "" && len(name) > 3 && 
					   !strings.Contains(strings.ToLower(href), "javascript:") &&
					   !strings.Contains(strings.ToLower(href), "#") &&
					   (strings.Contains(href, "/torrent/") || strings.Contains(href, ".html")) {
						result := &TorrentResult{
							Source:     models.TorrentSource1337x,
							TorrentURL: s.baseURL + href,
							Name:       name,
							Size:       0, // Initialiser explicitement
						}
						results = append(results, result)
					}
				}
			}
			if len(results) > 0 {
				log.Printf("[1337x] Parsing regex (liens directs): %d résultats trouvés", len(results))
				return results, nil
			}
		}
	}

	for i, row := range rows {
		if i >= 100 {
			break
		}

		result := s.parseRow(row)
		if result != nil {
			if languageFilter != "" && result.Language != "" && !strings.EqualFold(result.Language, languageFilter) {
				continue
			}
			results = append(results, result)
		}
	}

	return results, nil
}

// parseRow parse une ligne de résultat avec plusieurs patterns flexibles
func (s *Scraper1337x) parseRow(row string) *TorrentResult {
	result := &TorrentResult{
		Source: models.TorrentSource1337x,
	}

	// Extraire le nom et l'URL - essayer plusieurs patterns
	var nameMatches []string
	
	// Pattern 1: href="/torrent/..." ou liens vers pages de détails (ex: /alice-au-pays-des-merveilles-1951-...html)
	nameRe1 := regexp.MustCompile(`<a[^>]*href="(/torrent/[^"]+)"[^>]*>([^<]+)</a>`)
	nameMatches = nameRe1.FindStringSubmatch(row)
	
	// Pattern 2: href vers pages de détails (ex: /alice-au-pays-des-merveilles-1951-...html)
	if len(nameMatches) < 3 {
		nameRe2 := regexp.MustCompile(`<a[^>]*href="(/(?:[^/]+-)+[0-9]+\.html)"[^>]*>([^<]+)</a>`)
		nameMatches = nameRe2.FindStringSubmatch(row)
	}
	
	// Pattern 3: href="/watch-movie/..." ou autres variantes
	if len(nameMatches) < 3 {
		nameRe3 := regexp.MustCompile(`<a[^>]*href="(/watch-[^"]+)"[^>]*>([^<]+)</a>`)
		nameMatches = nameRe3.FindStringSubmatch(row)
	}
	
	// Pattern 4: href avec baseURL complète
	if len(nameMatches) < 3 {
		nameRe4 := regexp.MustCompile(`<a[^>]*href="(https?://[^"]*(?:torrent|/[^/]+-[0-9]+\.html)[^"]*)"[^>]*>([^<]+)</a>`)
		nameMatches = nameRe4.FindStringSubmatch(row)
	}
	
	if len(nameMatches) >= 3 {
		url := nameMatches[1]
		if !strings.HasPrefix(url, "http") {
			url = s.baseURL + url
		}
		result.TorrentURL = url
		result.Name = strings.TrimSpace(nameMatches[2])
	} else {
		return nil
	}

	// Extraire les seeders - essayer plusieurs patterns
	seedersRe1 := regexp.MustCompile(`<td[^>]*class="[^"]*seeds[^"]*"[^>]*>(\d+)</td>`)
	seedersRe2 := regexp.MustCompile(`<span[^>]*class="[^"]*seeds[^"]*"[^>]*>(\d+)</span>`)
	seedersRe3 := regexp.MustCompile(`seeds[^>]*>(\d+)`)
	
	if matches := seedersRe1.FindStringSubmatch(row); len(matches) > 1 {
		if val, err := strconv.Atoi(matches[1]); err == nil {
			result.Seeders = val
		}
	} else if matches := seedersRe2.FindStringSubmatch(row); len(matches) > 1 {
		if val, err := strconv.Atoi(matches[1]); err == nil {
			result.Seeders = val
		}
	} else if matches := seedersRe3.FindStringSubmatch(row); len(matches) > 1 {
		if val, err := strconv.Atoi(matches[1]); err == nil {
			result.Seeders = val
		}
	}

	// Extraire les leechers - essayer plusieurs patterns
	leechersRe1 := regexp.MustCompile(`<td[^>]*class="[^"]*leeches[^"]*"[^>]*>(\d+)</td>`)
	leechersRe2 := regexp.MustCompile(`<span[^>]*class="[^"]*leeches[^"]*"[^>]*>(\d+)</span>`)
	leechersRe3 := regexp.MustCompile(`leeches[^>]*>(\d+)`)
	
	if matches := leechersRe1.FindStringSubmatch(row); len(matches) > 1 {
		if val, err := strconv.Atoi(matches[1]); err == nil {
			result.Leechers = val
		}
	} else if matches := leechersRe2.FindStringSubmatch(row); len(matches) > 1 {
		if val, err := strconv.Atoi(matches[1]); err == nil {
			result.Leechers = val
		}
	} else if matches := leechersRe3.FindStringSubmatch(row); len(matches) > 1 {
		if val, err := strconv.Atoi(matches[1]); err == nil {
			result.Leechers = val
		}
	}

	// Extraire la date d'upload
	dateRe := regexp.MustCompile(`<td[^>]*class="[^"]*date[^"]*"[^>]*>([^<]+)</td>`)
	if matches := dateRe.FindStringSubmatch(row); len(matches) > 1 {
		result.UploadDate = strings.TrimSpace(matches[1])
	}

	// Extraire la taille - essayer plusieurs patterns
	sizeRe1 := regexp.MustCompile(`<td[^>]*class="[^"]*size[^"]*"[^>]*>([^<]+)</td>`)
	sizeRe2 := regexp.MustCompile(`Size[^>]*>([^<]+)</`)
	
	if matches := sizeRe1.FindStringSubmatch(row); len(matches) > 1 {
		parsedSize := formatSize(strings.TrimSpace(matches[1]))
		if parsedSize > 0 {
			result.Size = parsedSize
			log.Printf("[1337x] Taille extraite (parseRow, pattern 1): %s -> %d bytes", matches[1], parsedSize)
		}
	} else if matches := sizeRe2.FindStringSubmatch(row); len(matches) > 1 {
		parsedSize := formatSize(strings.TrimSpace(matches[1]))
		if parsedSize > 0 {
			result.Size = parsedSize
			log.Printf("[1337x] Taille extraite (parseRow, pattern 2): %s -> %d bytes", matches[1], parsedSize)
		}
	}
	
	// Si la taille n'a pas été trouvée, essayer de la trouver dans toutes les colonnes
	if result.Size == 0 {
		tdRe := regexp.MustCompile(`<td[^>]*>([^<]+)</td>`)
		tdMatches := tdRe.FindAllStringSubmatch(row, -1)
		for _, match := range tdMatches {
			if len(match) > 1 {
				text := strings.TrimSpace(match[1])
				// Vérifier si c'est une taille (contient des unités comme GB, MB, etc.)
				if strings.Contains(strings.ToUpper(text), "GB") || 
				   strings.Contains(strings.ToUpper(text), "MB") || 
				   strings.Contains(strings.ToUpper(text), "GO") || 
				   strings.Contains(strings.ToUpper(text), "MO") ||
				   strings.Contains(strings.ToUpper(text), "TB") ||
				   strings.Contains(strings.ToUpper(text), "KB") {
					parsedSize := formatSize(text)
					if parsedSize > 0 {
						result.Size = parsedSize
						log.Printf("[1337x] Taille extraite (parseRow, pattern générique): %s -> %d bytes", text, parsedSize)
						break
					}
				}
			}
		}
	}

	// Extraire la qualité depuis le nom (1080p, 720p, etc.)
	qualityRe := regexp.MustCompile(`(\d+p|4K|2160p|1440p|1080p|720p|480p|360p)`)
	if matches := qualityRe.FindStringSubmatch(result.Name); len(matches) > 1 {
		result.Quality = matches[1]
	}

	// Extraire la langue depuis le nom (FR, EN, etc.)
	langRe := regexp.MustCompile(`\b(FR|EN|ES|DE|IT|JP|CN|KR|RU|PT|NL|PL|TR|AR|HI|TH|VI|VOSTFR|MULTI)\b`)
	if matches := langRe.FindStringSubmatch(strings.ToUpper(result.Name)); len(matches) > 1 {
		result.Language = matches[1]
	}

	// Extraire la catégorie
	categoryRe := regexp.MustCompile(`<a[^>]*href="[^"]*/cat/([^/]+)/"`)
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

// parseTorrentDetails parse les détails d'un torrent avec patterns améliorés
func (s *Scraper1337x) parseTorrentDetails(html, torrentURL string) (*TorrentResult, error) {
	result := &TorrentResult{
		TorrentURL: torrentURL,
		Source:     models.TorrentSource1337x,
	}

	// Extraire le magnet link - essayer plusieurs patterns
	magnetRe1 := regexp.MustCompile(`href="(magnet:\?[^"]+)"`)
	magnetRe2 := regexp.MustCompile(`href='(magnet:\?[^']+)'`)
	magnetRe3 := regexp.MustCompile(`(magnet:\?[^\s"']+)`)
	
	if matches := magnetRe1.FindStringSubmatch(html); len(matches) > 1 {
		result.MagnetURL = matches[1]
	} else if matches := magnetRe2.FindStringSubmatch(html); len(matches) > 1 {
		result.MagnetURL = matches[1]
	} else if matches := magnetRe3.FindStringSubmatch(html); len(matches) > 1 {
		result.MagnetURL = matches[1]
	}

	// Extraire l'infohash depuis le magnet link ou directement depuis la page
	if result.MagnetURL != "" {
		infohashRe := regexp.MustCompile(`btih:([A-F0-9]{40})`)
		if matches := infohashRe.FindStringSubmatch(strings.ToUpper(result.MagnetURL)); len(matches) > 1 {
			result.Infohash = matches[1]
		}
	}
	
	// Essayer aussi de trouver l'infohash directement dans la page
	if result.Infohash == "" {
		infohashRe1 := regexp.MustCompile(`Infohash[^>]*>([A-F0-9]{40})`)
		infohashRe2 := regexp.MustCompile(`Infohash[^:]*:\s*([A-F0-9]{40})`)
		if matches := infohashRe1.FindStringSubmatch(strings.ToUpper(html)); len(matches) > 1 {
			result.Infohash = matches[1]
		} else if matches := infohashRe2.FindStringSubmatch(strings.ToUpper(html)); len(matches) > 1 {
			result.Infohash = matches[1]
		}
	}

	// Extraire le lien torrent (.torrent file) - essayer plusieurs patterns
	torrentFileRe1 := regexp.MustCompile(`href="([^"]+\.torrent)"`)
	torrentFileRe2 := regexp.MustCompile(`href='([^']+\.torrent)'`)
	torrentFileRe3 := regexp.MustCompile(`href="(/download/[^"]+)"`)
	
	if matches := torrentFileRe1.FindStringSubmatch(html); len(matches) > 1 {
		torrentFileURL := matches[1]
		if !strings.HasPrefix(torrentFileURL, "http") {
			torrentFileURL = s.baseURL + torrentFileURL
		}
		result.TorrentFileURL = torrentFileURL
	} else if matches := torrentFileRe2.FindStringSubmatch(html); len(matches) > 1 {
		torrentFileURL := matches[1]
		if !strings.HasPrefix(torrentFileURL, "http") {
			torrentFileURL = s.baseURL + torrentFileURL
		}
		result.TorrentFileURL = torrentFileURL
	} else if matches := torrentFileRe3.FindStringSubmatch(html); len(matches) > 1 {
		result.TorrentFileURL = s.baseURL + matches[1]
	}

	// Extraire les seeders - essayer plusieurs patterns
	seedersRe1 := regexp.MustCompile(`<span[^>]*class="[^"]*seeds[^"]*"[^>]*>(\d+)</span>`)
	seedersRe2 := regexp.MustCompile(`Semoirs[^>]*>(\d+)`)
	seedersRe3 := regexp.MustCompile(`Seeds[^>]*>(\d+)`)
	seedersRe4 := regexp.MustCompile(`seeders?[^>]*>(\d+)`)
	
	if matches := seedersRe1.FindStringSubmatch(html); len(matches) > 1 {
		if val, err := strconv.Atoi(matches[1]); err == nil {
			result.Seeders = val
		}
	} else if matches := seedersRe2.FindStringSubmatch(html); len(matches) > 1 {
		if val, err := strconv.Atoi(matches[1]); err == nil {
			result.Seeders = val
		}
	} else if matches := seedersRe3.FindStringSubmatch(html); len(matches) > 1 {
		if val, err := strconv.Atoi(matches[1]); err == nil {
			result.Seeders = val
		}
	} else if matches := seedersRe4.FindStringSubmatch(html); len(matches) > 1 {
		if val, err := strconv.Atoi(matches[1]); err == nil {
			result.Seeders = val
		}
	}

	// Extraire les leechers - essayer plusieurs patterns
	leechersRe1 := regexp.MustCompile(`<span[^>]*class="[^"]*leeches[^"]*"[^>]*>(\d+)</span>`)
	leechersRe2 := regexp.MustCompile(`Sangsues[^>]*>(\d+)`)
	leechersRe3 := regexp.MustCompile(`Leechers[^>]*>(\d+)`)
	leechersRe4 := regexp.MustCompile(`leeches?[^>]*>(\d+)`)
	
	if matches := leechersRe1.FindStringSubmatch(html); len(matches) > 1 {
		if val, err := strconv.Atoi(matches[1]); err == nil {
			result.Leechers = val
		}
	} else if matches := leechersRe2.FindStringSubmatch(html); len(matches) > 1 {
		if val, err := strconv.Atoi(matches[1]); err == nil {
			result.Leechers = val
		}
	} else if matches := leechersRe3.FindStringSubmatch(html); len(matches) > 1 {
		if val, err := strconv.Atoi(matches[1]); err == nil {
			result.Leechers = val
		}
	} else if matches := leechersRe4.FindStringSubmatch(html); len(matches) > 1 {
		if val, err := strconv.Atoi(matches[1]); err == nil {
			result.Leechers = val
		}
	}

	// Extraire la taille - essayer plusieurs patterns
	sizeRe1 := regexp.MustCompile(`Taille totale[^>]*>([^<]+)</`)
	sizeRe2 := regexp.MustCompile(`Size[^>]*>([^<]+)</`)
	sizeRe3 := regexp.MustCompile(`Taille[^>]*>([^<]+)</`)
	sizeRe4 := regexp.MustCompile(`(\d+[.,]\d+\s*[KMGT]?i?B)`)
	
	if matches := sizeRe1.FindStringSubmatch(html); len(matches) > 1 {
		result.Size = formatSize(strings.TrimSpace(matches[1]))
	} else if matches := sizeRe2.FindStringSubmatch(html); len(matches) > 1 {
		result.Size = formatSize(strings.TrimSpace(matches[1]))
	} else if matches := sizeRe3.FindStringSubmatch(html); len(matches) > 1 {
		result.Size = formatSize(strings.TrimSpace(matches[1]))
	} else if matches := sizeRe4.FindStringSubmatch(html); len(matches) > 1 {
		result.Size = formatSize(strings.TrimSpace(matches[1]))
	}

	// Extraire le nom/titre - essayer plusieurs patterns
	nameRe1 := regexp.MustCompile(`<h1[^>]*>([^<]+)</h1>`)
	nameRe2 := regexp.MustCompile(`<title>([^<]+)</title>`)
	nameRe3 := regexp.MustCompile(`<h2[^>]*>([^<]+)</h2>`)
	
	if matches := nameRe1.FindStringSubmatch(html); len(matches) > 1 {
		result.Name = strings.TrimSpace(matches[1])
	} else if matches := nameRe2.FindStringSubmatch(html); len(matches) > 1 {
		result.Name = strings.TrimSpace(matches[1])
	} else if matches := nameRe3.FindStringSubmatch(html); len(matches) > 1 {
		result.Name = strings.TrimSpace(matches[1])
	}

	// Extraire la catégorie
	categoryRe1 := regexp.MustCompile(`Catégorie[^>]*>([^<]+)</`)
	categoryRe2 := regexp.MustCompile(`Category[^>]*>([^<]+)</`)
	if matches := categoryRe1.FindStringSubmatch(html); len(matches) > 1 {
		result.Category = strings.TrimSpace(matches[1])
	} else if matches := categoryRe2.FindStringSubmatch(html); len(matches) > 1 {
		result.Category = strings.TrimSpace(matches[1])
	}

	// Extraire la qualité depuis le nom (1080p, 720p, etc.)
	qualityRe := regexp.MustCompile(`(\d+p|4K|2160p|1440p|1080p|720p|480p|360p)`)
	if matches := qualityRe.FindStringSubmatch(result.Name); len(matches) > 1 {
		result.Quality = matches[1]
	}

	// Extraire la langue depuis le nom (FR, EN, MULTI, etc.)
	langRe := regexp.MustCompile(`\b(FR|EN|ES|DE|IT|JP|CN|KR|RU|PT|NL|PL|TR|AR|HI|TH|VI|VOSTFR|MULTI|MULTi)\b`)
	if matches := langRe.FindStringSubmatch(strings.ToUpper(result.Name)); len(matches) > 1 {
		result.Language = matches[1]
	}

	return result, nil
}
