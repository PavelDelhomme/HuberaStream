package torrent

import (
	"context"
	"fmt"

	"streammake/internal/config"
	"streammake/internal/models"
)

type TorrentClient interface {
	AddTorrent(ctx context.Context, torrent *models.Torrent) error
	RemoveTorrent(ctx context.Context, torrentID string) error
	GetTorrentStatus(ctx context.Context, torrentID string) (*TorrentStatus, error)
	ListTorrents(ctx context.Context) ([]*TorrentStatus, error)
	PauseTorrent(ctx context.Context, torrentID string) error
	ResumeTorrent(ctx context.Context, torrentID string) error
}

type TorrentStatus struct {
	ID             string
	Name           string
	Status         models.TorrentStatus
	Progress       float64
	Size           int64
	Downloaded     int64
	Uploaded       int64
	DownloadSpeed  int64
	UploadSpeed    int64
	Seeders        int
	Leechers       int
	Ratio          float64
	ETA            int64 // en secondes
}

type ClientType string

const (
	ClientTransmission ClientType = "transmission"
	ClientQBittorrent  ClientType = "qbittorrent"
	ClientDeluge       ClientType = "deluge"
	ClientRTorrent     ClientType = "rtorrent"
)

func NewTorrentClient(clientType ClientType, cfg *config.Config) (TorrentClient, error) {
	// Si clientType est vide, utiliser la config
	if clientType == "" {
		clientType = ClientType(cfg.TorrentClientType)
	}

	switch clientType {
	case ClientTransmission:
		return NewTransmissionClient(cfg)
	case ClientQBittorrent:
		return NewQBittorrentClient(cfg)
	case ClientDeluge:
		return NewDelugeClient(cfg)
	case ClientRTorrent:
		return NewRTorrentClient(cfg)
	default:
		return nil, fmt.Errorf("client torrent non supporté: %s", clientType)
	}
}

