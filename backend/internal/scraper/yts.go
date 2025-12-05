package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"streammake/internal/models"
)

type YTSScraper struct {
	baseURL string
	client  *http.Client
}

type YTSResponse struct {
	Status string    `json:"status"`
	Data   YTSData   `json:"data"`
}

type YTSData struct {
	Movies []YTSMovie `json:"movies"`
}

type YTSMovie struct {
	Title    string      `json:"title"`
	Year     int         `json:"year"`
	Torrents []YTSTorrent `json:"torrents"`
}

type YTSTorrent struct {
	URL      string `json:"url"`
	Hash     string `json:"hash"`
	Quality  string `json:"quality"`
	Size     string `json:"size"`
	Seeds    int    `json:"seeds"`
	Peers    int    `json:"peers"`
}

func NewYTSScraper() *YTSScraper {
	return &YTSScraper{
		baseURL: "https://yts.mx/api/v2",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *YTSScraper) Search(ctx context.Context, query string, category *models.Category, subCategory *models.SubCategory) ([]*TorrentResult, error) {
	searchURL := fmt.Sprintf("%s/list_movies.json?query_term=%s&limit=20", s.baseURL, url.QueryEscape(query))

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

	var ytsResp YTSResponse
	if err := json.Unmarshal(body, &ytsResp); err != nil {
		return nil, err
	}

	var results []*TorrentResult
	for _, movie := range ytsResp.Data.Movies {
		for _, torrent := range movie.Torrents {
			result := &TorrentResult{
				Name:       fmt.Sprintf("%s (%d) [%s]", movie.Title, movie.Year, torrent.Quality),
				TorrentURL: torrent.URL,
				MagnetURL:  fmt.Sprintf("magnet:?xt=urn:btih:%s", torrent.Hash),
				Source:     models.TorrentSourceYTS,
				Quality:    torrent.Quality,
				Seeders:    torrent.Seeds,
				Leechers:   torrent.Peers,
				Size:       formatSize(torrent.Size),
			}
			results = append(results, result)
		}
	}

	return results, nil
}

func (s *YTSScraper) GetTorrentDetails(ctx context.Context, torrentURL string) (*TorrentResult, error) {
	return &TorrentResult{
		TorrentURL: torrentURL,
		Source:     models.TorrentSourceYTS,
	}, nil
}

