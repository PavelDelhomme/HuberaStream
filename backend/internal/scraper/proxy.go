package scraper

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// ProxyManager gère la rotation des proxies
type ProxyManager struct {
	proxies    []string
	currentIdx int
	mu         sync.Mutex
	client     *http.Client
}

// NewProxyManager crée un nouveau gestionnaire de proxies
func NewProxyManager() *ProxyManager {
	return &ProxyManager{
		proxies: []string{},
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchProxies récupère une liste de proxies publics gratuits
func (pm *ProxyManager) FetchProxies(ctx context.Context) error {
	// Sources de proxies publics gratuits
	sources := []string{
		"https://api.proxyscrape.com/v2/?request=get&protocol=http&timeout=10000&country=all&ssl=all&anonymity=all",
		"https://raw.githubusercontent.com/clarketm/proxy-list/master/proxy-list-raw.txt",
		"https://raw.githubusercontent.com/TheSpeedX/PROXY-List/master/http.txt",
	}

	var allProxies []string

	for _, source := range sources {
		req, err := http.NewRequestWithContext(ctx, "GET", source, nil)
		if err != nil {
			continue
		}

		resp, err := pm.client.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			continue
		}

		// Parser les proxies (format: IP:PORT)
		lines := strings.Split(string(body), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				// Vérifier le format IP:PORT
				if strings.Contains(line, ":") {
					allProxies = append(allProxies, line)
				}
			}
		}
	}

	pm.mu.Lock()
	pm.proxies = allProxies
	pm.currentIdx = 0
	pm.mu.Unlock()

	return nil
}

// GetNextProxy retourne le prochain proxy disponible
func (pm *ProxyManager) GetNextProxy() string {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if len(pm.proxies) == 0 {
		return ""
	}

	proxy := pm.proxies[pm.currentIdx]
	pm.currentIdx = (pm.currentIdx + 1) % len(pm.proxies)
	return proxy
}

// TestProxy teste si un proxy fonctionne
func (pm *ProxyManager) TestProxy(ctx context.Context, proxyURL string) bool {
	proxy, err := url.Parse(fmt.Sprintf("http://%s", proxyURL))
	if err != nil {
		return false
	}

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxy),
		},
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.1337x.to", nil)
	if err != nil {
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}

// GetClientWithProxy retourne un client HTTP avec proxy
func (pm *ProxyManager) GetClientWithProxy(proxyURL string) (*http.Client, error) {
	if proxyURL == "" {
		return &http.Client{
			Timeout: 30 * time.Second,
		}, nil
	}

	proxy, err := url.Parse(fmt.Sprintf("http://%s", proxyURL))
	if err != nil {
		return nil, err
	}

	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxy),
		},
		Timeout: 30 * time.Second,
	}, nil
}

