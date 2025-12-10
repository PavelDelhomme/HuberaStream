package torrent

import (
	"context"
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"streammake/internal/config"
	"streammake/internal/models"
	"streammake/internal/video"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PostProcessor struct {
	db        *gorm.DB
	cfg       *config.Config
	processor *video.Processor
}

func NewPostProcessor(db *gorm.DB, cfg *config.Config) *PostProcessor {
	return &PostProcessor{
		db:        db,
		cfg:       cfg,
		processor: video.NewProcessor(cfg.VideoStoragePath, cfg.ThumbnailStoragePath),
	}
}

// ProcessCompletedTorrents traite les torrents terminés
func (p *PostProcessor) ProcessCompletedTorrents(ctx context.Context) error {
	var completedTorrents []models.Torrent
	if err := p.db.Where("status = ? AND video_id IS NULL", models.TorrentStatusCompleted).
		Find(&completedTorrents).Error; err != nil {
		return fmt.Errorf("erreur lors de la récupération des torrents terminés: %w", err)
	}

	for _, torrent := range completedTorrents {
		if err := p.processTorrent(ctx, &torrent); err != nil {
			log.Printf("Erreur lors du traitement du torrent %s: %v", torrent.ID, err)
			// Continuer avec les autres torrents même en cas d'erreur
		}
	}

	return nil
}

// processTorrent traite un torrent terminé et crée une vidéo
func (p *PostProcessor) processTorrent(ctx context.Context, torrent *models.Torrent) error {
	// Trouver les fichiers vidéo dans le dossier de téléchargement
	downloadPath := p.cfg.TorrentDownloadPath
	if torrent.DownloadPath != "" {
		downloadPath = torrent.DownloadPath
	}

	videoFiles, err := p.findVideoFiles(downloadPath)
	if err != nil {
		return fmt.Errorf("erreur lors de la recherche de fichiers vidéo: %w", err)
	}

	if len(videoFiles) == 0 {
		log.Printf("Aucun fichier vidéo trouvé pour le torrent %s dans %s", torrent.ID, downloadPath)
		return nil
	}

	// Traiter chaque fichier vidéo trouvé
	for _, videoFile := range videoFiles {
		if err := p.createVideoFromFile(ctx, torrent, videoFile); err != nil {
			log.Printf("Erreur lors de la création de la vidéo depuis %s: %v", videoFile, err)
			// Continuer avec les autres fichiers
		}
	}

	return nil
}

// findVideoFiles trouve les fichiers vidéo dans un répertoire
func (p *PostProcessor) findVideoFiles(rootPath string) ([]string, error) {
	var videoFiles []string
	videoExtensions := []string{".mp4", ".mkv", ".avi", ".mov", ".wmv", ".flv", ".webm", ".m4v"}

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Ignorer les erreurs de lecture
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		for _, videoExt := range videoExtensions {
			if ext == videoExt {
				videoFiles = append(videoFiles, path)
				break
			}
		}

		return nil
	})

	return videoFiles, err
}

// createVideoFromFile crée une vidéo depuis un fichier téléchargé
func (p *PostProcessor) createVideoFromFile(ctx context.Context, torrent *models.Torrent, videoFilePath string) error {
	// Générer un titre depuis le nom du fichier
	title := strings.TrimSuffix(filepath.Base(videoFilePath), filepath.Ext(videoFilePath))
	title = strings.ReplaceAll(title, "_", " ")
	title = strings.ReplaceAll(title, ".", " ")
	title = strings.TrimSpace(title)
	if title == "" {
		title = torrent.Name
	}

	// Calculer le checksum
	file, err := os.Open(videoFilePath)
	if err != nil {
		return fmt.Errorf("erreur lors de l'ouverture du fichier: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return fmt.Errorf("erreur lors du calcul du checksum: %w", err)
	}
	checksum := fmt.Sprintf("%x", hash.Sum(nil))

	// Créer l'entrée vidéo
	videoID := uuid.New()
	videoModel := models.Video{
		ID:            videoID,
		Title:         title,
		Description:   fmt.Sprintf("Téléchargé depuis torrent: %s", torrent.Name),
		UserID:        uuid.MustParse("00000000-0000-0000-0000-000000000000"), // Admin par défaut
		Status:        models.VideoStatusProcessing,
		CategoryID:    torrent.CategoryID,
		SubCategoryID: torrent.SubCategoryID,
		IsPublic:      true,
		Checksum:      checksum,
	}

	if err := p.db.Create(&videoModel).Error; err != nil {
		return fmt.Errorf("erreur lors de la création de la vidéo: %w", err)
	}

	// Copier le fichier vers le dossier de stockage vidéo
	videoFileName := fmt.Sprintf("%s_%d%s", videoID.String(), time.Now().Unix(), filepath.Ext(videoFilePath))
	videoStoragePath := filepath.Join(p.cfg.VideoStoragePath, videoFileName)

	if err := p.copyFile(videoFilePath, videoStoragePath); err != nil {
		p.db.Delete(&videoModel)
		return fmt.Errorf("erreur lors de la copie du fichier: %w", err)
	}

	// Lier le torrent à la vidéo
	torrent.VideoID = &videoID
	if err := p.db.Save(torrent).Error; err != nil {
		log.Printf("Erreur lors de la liaison du torrent à la vidéo: %v", err)
	}

	// Lancer le traitement vidéo en arrière-plan
	go p.processVideo(ctx, videoID, videoStoragePath)

	log.Printf("Vidéo créée depuis torrent %s: %s (ID: %s)", torrent.ID, title, videoID)
	return nil
}

// copyFile copie un fichier
func (p *PostProcessor) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// processVideo traite une vidéo (transcodage, miniatures, etc.)
func (p *PostProcessor) processVideo(ctx context.Context, videoID uuid.UUID, videoPath string) {
	var videoModel models.Video
	if err := p.db.Where("id = ?", videoID).First(&videoModel).Error; err != nil {
		log.Printf("Erreur: vidéo %s non trouvée", videoID)
		return
	}

	title := videoModel.Title
	description := videoModel.Description

	// Générer la miniature
	var thumbnailPath string
	var err error

	// Générer la miniature depuis la vidéo
	thumbnailPath, err = p.processor.GenerateThumbnail(videoPath, videoID)
	if err != nil {
		log.Printf("Erreur lors de la génération de la miniature depuis la vidéo: %v", err)
	}

	// Si échec, créer une miniature textuelle
	if thumbnailPath == "" || err != nil {
		thumbnailPath, err = p.generateTextThumbnail(videoID, title, description)
		if err != nil {
			log.Printf("Erreur lors de la génération de la miniature textuelle: %v", err)
		}
	}

	if thumbnailPath != "" {
		p.db.Model(&models.Video{}).Where("id = ?", videoID).Update("thumbnail_path", thumbnailPath)
	}

	// Récupérer la durée
	duration, err := p.processor.GetVideoDuration(videoPath)
	if err == nil {
		p.db.Model(&models.Video{}).Where("id = ?", videoID).Update("duration", duration)
	}

	// Transcodage en plusieurs qualités
	qualities := []string{"360p", "480p", "720p", "1080p"}
	for _, quality := range qualities {
		select {
		case <-ctx.Done():
			log.Printf("Traitement de la vidéo %s annulé", videoID)
			p.db.Model(&models.Video{}).Where("id = ?", videoID).Update("status", models.VideoStatusCancelled)
			return
		default:
		}

		// Transcodage progressif
		outputPath, err := p.processor.TranscodeToProgressive(videoPath, videoID, quality)
		if err != nil {
			log.Printf("Erreur lors du transcodage %s: %v", quality, err)
			continue
		}

		// Créer l'entrée VideoFormat
		format := models.VideoFormat{
			ID:       uuid.New(),
			VideoID:  videoID,
			Format:   "progressive",
			Quality:  quality,
			FilePath: outputPath,
		}
		p.db.Create(&format)
	}

	// Transcodage HLS
	select {
	case <-ctx.Done():
		log.Printf("Traitement de la vidéo %s annulé avant HLS", videoID)
		p.db.Model(&models.Video{}).Where("id = ?", videoID).Update("status", models.VideoStatusCancelled)
		return
	default:
	}

	hlsManifest, hlsDir, err := p.processor.TranscodeToHLS(videoPath, videoID, "720p")
	if err == nil {
		format := models.VideoFormat{
			ID:       uuid.New(),
			VideoID:  videoID,
			Format:   "hls",
			Quality:  "720p",
			FilePath: hlsDir,
		}
		p.db.Create(&format)
	}

	// Vérifier si des formats ont été créés
	var formatCount int64
	p.db.Model(&models.VideoFormat{}).Where("video_id = ?", videoID).Count(&formatCount)

	if formatCount > 0 {
		p.db.Model(&models.Video{}).Where("id = ?", videoID).Update("status", models.VideoStatusReady)
		log.Printf("Vidéo %s traitée avec succès (%d formats)", videoID, formatCount)
	} else {
		p.db.Model(&models.Video{}).Where("id = ?", videoID).Update("status", models.VideoStatusError)
		log.Printf("Aucun format créé pour la vidéo %s", videoID)
	}
}

// generateTextThumbnail génère une miniature textuelle
func (p *PostProcessor) generateTextThumbnail(videoID uuid.UUID, title string, description string) (string, error) {
	width := 1280
	height := 720

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fond dégradé rouge/noir
	for y := 0; y < height; y++ {
		ratio := float64(y) / float64(height)
		r := uint8(255 * (1 - ratio*0.7))
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{r, 0, 0, 255})
		}
	}

	if title == "" || title == "null" {
		title = "Vidéo sans titre"
	}

	thumbnailName := fmt.Sprintf("%s.jpg", videoID.String())
	thumbnailPath := filepath.Join(p.cfg.ThumbnailStoragePath, thumbnailName)

	if err := os.MkdirAll(p.cfg.ThumbnailStoragePath, 0755); err != nil {
		return "", fmt.Errorf("erreur lors de la création du dossier: %w", err)
	}

	file, err := os.Create(thumbnailPath)
	if err != nil {
		return "", fmt.Errorf("erreur lors de la création du fichier: %w", err)
	}
	defer file.Close()

	if err := jpeg.Encode(file, img, &jpeg.Options{Quality: 90}); err != nil {
		return "", fmt.Errorf("erreur lors de l'encodage JPEG: %w", err)
	}

	return thumbnailPath, nil
}

