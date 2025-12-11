package torrent

import (
	"context"
	"fmt"
	"time"

	"streammake/internal/config"
	"streammake/internal/models"

	"gorm.io/gorm"
)

type TorrentService struct {
	db            *gorm.DB
	cfg           *config.Config
	client        TorrentClient
	clientType    ClientType
	updateTicker  *time.Ticker
	postProcessor *PostProcessor
}

func NewTorrentService(db *gorm.DB, cfg *config.Config, clientType ClientType) (*TorrentService, error) {
	// Si clientType est vide, utiliser la config
	if clientType == "" {
		clientType = ClientType(cfg.TorrentClientType)
	}

	client, err := NewTorrentClient(clientType, cfg)
	if err != nil {
		return nil, err
	}

	postProcessor := NewPostProcessor(db, cfg)

	service := &TorrentService{
		db:            db,
		cfg:           cfg,
		client:        client,
		clientType:    clientType,
		postProcessor: postProcessor,
	}

	// Démarrer le service de mise à jour périodique
	service.startUpdateService()
	
	// Démarrer le service de post-traitement
	service.startPostProcessingService()

	return service, nil
}

func (s *TorrentService) AddTorrent(ctx context.Context, torrent *models.Torrent) error {
	// Ajouter au client torrent
	if err := s.client.AddTorrent(ctx, torrent); err != nil {
		return fmt.Errorf("erreur lors de l'ajout au client: %w", err)
	}

	// Mettre à jour le statut
	torrent.Status = models.TorrentStatusDownloading
	if err := s.db.Save(torrent).Error; err != nil {
		return err
	}

	return nil
}

func (s *TorrentService) UpdateTorrentStatus(ctx context.Context, torrent *models.Torrent) error {
	// Obtenir le statut depuis le client
	// Note: On doit mapper l'ID du torrent dans notre DB vers l'ID du client
	// Pour simplifier, on utilise le hash ou un identifiant unique
	torrentID := torrent.ID.String()
	
	status, err := s.client.GetTorrentStatus(ctx, torrentID)
	if err != nil {
		// Le torrent n'existe peut-être pas encore dans le client
		return nil
	}

	// Mettre à jour le torrent
	torrent.Status = status.Status
	torrent.Progress = status.Progress
	torrent.Size = status.Size
	torrent.Downloaded = status.Downloaded
	torrent.Uploaded = status.Uploaded
	torrent.DownloadSpeed = status.DownloadSpeed
	torrent.UploadSpeed = status.UploadSpeed
	torrent.Seeders = status.Seeders
	torrent.Leechers = status.Leechers

	return s.db.Save(torrent).Error
}

func (s *TorrentService) startUpdateService() {
	s.updateTicker = time.NewTicker(10 * time.Second) // Mise à jour toutes les 10 secondes

	go func() {
		for range s.updateTicker.C {
			// Vérifier d'abord s'il y a des torrents actifs avant de faire des requêtes
			var count int64
			if err := s.db.Model(&models.Torrent{}).Where("status IN ?", []models.TorrentStatus{
				models.TorrentStatusPending,
				models.TorrentStatusDownloading,
				models.TorrentStatusSeeding,
			}).Count(&count).Error; err == nil && count > 0 {
				s.updateAllTorrents()
			}
		}
	}()
}

func (s *TorrentService) startPostProcessingService() {
	postProcessTicker := time.NewTicker(30 * time.Second) // Vérifier toutes les 30 secondes

	go func() {
		for range postProcessTicker.C {
			// Vérifier d'abord s'il y a des torrents terminés à traiter avant de faire des requêtes
			var count int64
			if err := s.db.Model(&models.Torrent{}).Where("status = ? AND video_id IS NULL", models.TorrentStatusCompleted).Count(&count).Error; err == nil && count > 0 {
				ctx := context.Background()
				// Traiter silencieusement, log seulement les erreurs critiques
				_ = s.postProcessor.ProcessCompletedTorrents(ctx)
			}
		}
	}()
}

func (s *TorrentService) updateAllTorrents() {
	ctx := context.Background()

	var torrents []models.Torrent
	if err := s.db.Where("status IN ?", []models.TorrentStatus{
		models.TorrentStatusPending,
		models.TorrentStatusDownloading,
		models.TorrentStatusSeeding,
	}).Find(&torrents).Error; err != nil {
		// Log seulement les erreurs critiques
		return
	}

	// Si aucun torrent actif, ne rien faire
	if len(torrents) == 0 {
		return
	}

	// Mettre à jour les statuts silencieusement (sans logs)
	for _, torrent := range torrents {
		_ = s.UpdateTorrentStatus(ctx, &torrent) // Ignorer les erreurs non critiques
	}

	// Vérifier les torrents terminés et les marquer comme completed si nécessaire
	// Seulement si on a des torrents en téléchargement
	var downloadingTorrents []models.Torrent
	if err := s.db.Where("status = ?", models.TorrentStatusDownloading).Find(&downloadingTorrents).Error; err == nil && len(downloadingTorrents) > 0 {
		for _, torrent := range downloadingTorrents {
			status, err := s.GetTorrentStatus(ctx, torrent.ID.String())
			if err == nil && status.Progress >= 100 && status.Status == models.TorrentStatusSeeding {
				torrent.Status = models.TorrentStatusCompleted
				s.db.Save(&torrent)
			}
		}
	}
}

func (s *TorrentService) Stop() {
	if s.updateTicker != nil {
		s.updateTicker.Stop()
	}
}

func (s *TorrentService) PauseTorrent(ctx context.Context, torrentID string) error {
	return s.client.PauseTorrent(ctx, torrentID)
}

func (s *TorrentService) ResumeTorrent(ctx context.Context, torrentID string) error {
	return s.client.ResumeTorrent(ctx, torrentID)
}

func (s *TorrentService) RemoveTorrent(ctx context.Context, torrentID string) error {
	return s.client.RemoveTorrent(ctx, torrentID)
}

func (s *TorrentService) GetTorrentStatus(ctx context.Context, torrentID string) (*TorrentStatus, error) {
	return s.client.GetTorrentStatus(ctx, torrentID)
}

