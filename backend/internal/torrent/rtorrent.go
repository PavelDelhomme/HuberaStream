package torrent

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"streammake/internal/config"
	"streammake/internal/models"
)

type RTorrentClient struct {
	cfg     *config.Config
	baseURL string
	client  *http.Client
}

func NewRTorrentClient(cfg *config.Config) (*RTorrentClient, error) {
	baseURL := "http://localhost:80/RPC2"
	if cfg.TorrentClientPath != "" {
		baseURL = cfg.TorrentClientPath
	}

	return &RTorrentClient{
		cfg:     cfg,
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (c *RTorrentClient) AddTorrent(ctx context.Context, torrent *models.Torrent) error {
	// rTorrent utilise XML-RPC
	// Pour simplifier, on utilise une approche basique
	// En production, utiliser une bibliothèque XML-RPC
	return fmt.Errorf("rTorrent nécessite une implémentation XML-RPC complète")
}

func (c *RTorrentClient) RemoveTorrent(ctx context.Context, torrentID string) error {
	return fmt.Errorf("non implémenté")
}

func (c *RTorrentClient) GetTorrentStatus(ctx context.Context, torrentID string) (*TorrentStatus, error) {
	return nil, fmt.Errorf("non implémenté")
}

func (c *RTorrentClient) ListTorrents(ctx context.Context) ([]*TorrentStatus, error) {
	return nil, fmt.Errorf("non implémenté")
}

func (c *RTorrentClient) PauseTorrent(ctx context.Context, torrentID string) error {
	return fmt.Errorf("non implémenté")
}

func (c *RTorrentClient) ResumeTorrent(ctx context.Context, torrentID string) error {
	return fmt.Errorf("non implémenté")
}
