package torrent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"streammake/internal/config"
	"streammake/internal/models"
)

type QBittorrentClient struct {
	cfg      *config.Config
	baseURL  string
	username string
	password string
	client   *http.Client
	sid      string
}

type QBittorrentTorrent struct {
	Hash            string  `json:"hash"`
	Name            string  `json:"name"`
	State           string  `json:"state"` // downloading, seeding, paused, etc.
	Progress        float64 `json:"progress"`
	Size            int64   `json:"size"`
	Downloaded      int64   `json:"downloaded"`
	Uploaded        int64   `json:"uploaded"`
	DlSpeed         int64   `json:"dlspeed"`
	UpSpeed         int64   `json:"upspeed"`
	NumSeeds        int     `json:"num_seeds"`
	NumLeechers     int     `json:"num_leechers"`
	Ratio           float64 `json:"ratio"`
	Eta             int64   `json:"eta"`
}

func NewQBittorrentClient(cfg *config.Config) (*QBittorrentClient, error) {
	baseURL := "http://localhost:8080"
	if cfg.TorrentClientPath != "" {
		baseURL = cfg.TorrentClientPath
	}

	client := &QBittorrentClient{
		cfg:      cfg,
		baseURL:  baseURL,
		username: "admin",
		password: "adminadmin",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Se connecter
	if err := client.login(context.Background()); err != nil {
		return nil, fmt.Errorf("erreur lors de la connexion: %w", err)
	}

	return client, nil
}

func (c *QBittorrentClient) login(ctx context.Context) error {
	data := url.Values{}
	data.Set("username", c.username)
	data.Set("password", c.password)

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v2/auth/login", strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("échec de la connexion, code: %d", resp.StatusCode)
	}

	// Le cookie de session est dans les headers
	cookies := resp.Cookies()
	for _, cookie := range cookies {
		if cookie.Name == "SID" {
			c.sid = cookie.Value
			break
		}
	}

	return nil
}

func (c *QBittorrentClient) makeRequest(ctx context.Context, method, endpoint string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+endpoint, body)
	if err != nil {
		return nil, err
	}

	if c.sid != "" {
		req.AddCookie(&http.Cookie{Name: "SID", Value: c.sid})
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	return c.client.Do(req)
}

func (c *QBittorrentClient) AddTorrent(ctx context.Context, torrent *models.Torrent) error {
	var torrentData string
	if torrent.MagnetURL != "" {
		torrentData = torrent.MagnetURL
	} else if torrent.TorrentURL != "" {
		// TODO: Télécharger le fichier .torrent
		return fmt.Errorf("téléchargement de fichier .torrent non encore implémenté")
	} else {
		return fmt.Errorf("magnet URL ou torrent URL requis")
	}

	data := url.Values{}
	data.Set("urls", torrentData)
	data.Set("savepath", c.cfg.TorrentDownloadPath)

	resp, err := c.makeRequest(ctx, "POST", "/api/v2/torrents/add", strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("erreur lors de l'ajout: %s", string(body))
	}

	return nil
}

func (c *QBittorrentClient) RemoveTorrent(ctx context.Context, torrentID string) error {
	data := url.Values{}
	data.Set("hashes", torrentID)
	data.Set("deleteFiles", "false")

	resp, err := c.makeRequest(ctx, "POST", "/api/v2/torrents/delete", strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("erreur lors de la suppression, code: %d", resp.StatusCode)
	}

	return nil
}

func (c *QBittorrentClient) GetTorrentStatus(ctx context.Context, torrentID string) (*TorrentStatus, error) {
	torrents, err := c.ListTorrents(ctx)
	if err != nil {
		return nil, err
	}

	for _, t := range torrents {
		if t.ID == torrentID {
			return t, nil
		}
	}

	return nil, fmt.Errorf("torrent non trouvé")
}

func (c *QBittorrentClient) ListTorrents(ctx context.Context) ([]*TorrentStatus, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v2/torrents/info", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var qbTorrents []QBittorrentTorrent
	if err := json.NewDecoder(resp.Body).Decode(&qbTorrents); err != nil {
		return nil, err
	}

	var statuses []*TorrentStatus
	for _, t := range qbTorrents {
		status := &TorrentStatus{
			ID:            t.Hash,
			Name:          t.Name,
			Progress:      t.Progress * 100,
			Size:          t.Size,
			Downloaded:    t.Downloaded,
			Uploaded:      t.Uploaded,
			DownloadSpeed: t.DlSpeed,
			UploadSpeed:   t.UpSpeed,
			Seeders:       t.NumSeeds,
			Leechers:      t.NumLeechers,
			Ratio:         t.Ratio,
			ETA:           t.Eta,
		}

		// Convertir le statut qBittorrent
		switch t.State {
		case "downloading":
			status.Status = models.TorrentStatusDownloading
		case "seeding":
			status.Status = models.TorrentStatusSeeding
		case "pausedDL", "pausedUP":
			status.Status = models.TorrentStatusPaused
		case "completed":
			status.Status = models.TorrentStatusCompleted
		default:
			status.Status = models.TorrentStatusPending
		}

		statuses = append(statuses, status)
	}

	return statuses, nil
}

func (c *QBittorrentClient) PauseTorrent(ctx context.Context, torrentID string) error {
	data := url.Values{}
	data.Set("hashes", torrentID)

	resp, err := c.makeRequest(ctx, "POST", "/api/v2/torrents/pause", strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("erreur lors de la pause, code: %d", resp.StatusCode)
	}

	return nil
}

func (c *QBittorrentClient) ResumeTorrent(ctx context.Context, torrentID string) error {
	data := url.Values{}
	data.Set("hashes", torrentID)

	resp, err := c.makeRequest(ctx, "POST", "/api/v2/torrents/resume", strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("erreur lors de la reprise, code: %d", resp.StatusCode)
	}

	return nil
}

