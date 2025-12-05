package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"streammake/internal/models"
)

type RARBGScraper struct {
	baseURL string
	client  *http.Client
	token   string
}

type RARBGResponse struct {
	TorrentResults []RARBGTorrent `json:"torrent_results"`
}

type RARBGTorrent struct {
	Title      string `json:"title"`
	Category   string `json:"category"`
	Download   string `json:"download"`
	Seeders    string `json:"seeders"`
	Leechers   string `json:"leechers"`
	Size       string `json:"size"`
	Pubdate    string `json:"pubdate"`
	EpisodeInfo string `json:"episode_info"`
}

func NewRARBGScraper() *RARBGScraper {
	return &RARBGScraper{
		baseURL: "https://torrentapi.org/pubapi_v2.php",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *RARBGScraper) getToken(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", s.baseURL+"?get_token=get_token&app_id=StreamMake", nil)
	if err != nil {
		return err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var tokenResp struct {
		Token string `json:"token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return err
	}

	s.token = tokenResp.Token
	return nil
}

func (s *RARBGScraper) Search(ctx context.Context, query string, category *models.Category, subCategory *models.SubCategory) ([]*TorrentResult, error) {
	if s.token == "" {
		if err := s.getToken(ctx); err != nil {
			return nil, err
		}
	}

	searchURL := fmt.Sprintf("%s?mode=search&search_string=%s&token=%s&format=json_extended&app_id=StreamMake",
		s.baseURL, url.QueryEscape(query), s.token)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rarbgResp RARBGResponse
	if err := json.Unmarshal(body, &rarbgResp); err != nil {
		// Peut-être une erreur de token
		bodyStr := string(body)
		if strings.Contains(bodyStr, "token") {
			s.token = ""
			return s.Search(ctx, query, category, subCategory)
		}
		return nil, err
	}

	var results []*TorrentResult
	for _, t := range rarbgResp.TorrentResults {
		result := &TorrentResult{
			Name:       t.Title,
			TorrentURL: t.Download,
			Source:     models.TorrentSourceRARBG,
			Category:   t.Category,
		}

		if seeders, err := strconv.Atoi(t.Seeders); err == nil {
			result.Seeders = seeders
		}
		if leechers, err := strconv.Atoi(t.Leechers); err == nil {
			result.Leechers = leechers
		}

		result.Size = formatSize(t.Size)
		result.UploadDate = t.Pubdate

		results = append(results, result)
	}

	return results, nil
}

func (s *RARBGScraper) GetTorrentDetails(ctx context.Context, torrentURL string) (*TorrentResult, error) {
	// RARBG fournit directement les détails dans la recherche
	return &TorrentResult{
		TorrentURL: torrentURL,
		Source:     models.TorrentSourceRARBG,
	}, nil
}

