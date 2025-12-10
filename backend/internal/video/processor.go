package video

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/google/uuid"
)

type Processor struct {
	storagePath      string
	thumbnailPath    string
	ffmpegPath       string
}

func NewProcessor(storagePath, thumbnailPath string) *Processor {
	ffmpegPath := "ffmpeg"
	if path := os.Getenv("FFMPEG_PATH"); path != "" {
		ffmpegPath = path
	}

	return &Processor{
		storagePath:   storagePath,
		thumbnailPath: thumbnailPath,
		ffmpegPath:    ffmpegPath,
	}
}

// GenerateThumbnail génère une miniature à partir d'une vidéo
// Essaie plusieurs positions temporelles pour trouver une bonne frame
func (p *Processor) GenerateThumbnail(videoPath string, videoID uuid.UUID) (string, error) {
	thumbnailName := fmt.Sprintf("%s.jpg", videoID.String())
	thumbnailPath := filepath.Join(p.thumbnailPath, thumbnailName)

	// Essayer plusieurs positions temporelles pour trouver une bonne frame
	// Positions à essayer : 1s, 5s, 10s, 30s, 60s
	timePositions := []string{"00:00:01", "00:00:05", "00:00:10", "00:00:30", "00:01:00"}
	
	var lastErr error
	for _, timePos := range timePositions {
		cmd := exec.Command(
			p.ffmpegPath,
			"-i", videoPath,
			"-ss", timePos,
			"-vframes", "1",
			"-q:v", "2", // Haute qualité
			"-vf", "scale=1280:720:force_original_aspect_ratio=decrease,pad=1280:720:(ow-iw)/2:(oh-ih)/2", // Format 16:9
			thumbnailPath,
		)

		if err := cmd.Run(); err != nil {
			lastErr = err
			continue // Essayer la position suivante
		}

		// Vérifier que le fichier a été créé et n'est pas vide
		if info, err := os.Stat(thumbnailPath); err == nil && info.Size() > 0 {
			return thumbnailPath, nil
		}
	}

	// Si toutes les tentatives ont échoué, retourner la dernière erreur
	return "", fmt.Errorf("erreur lors de la génération de la miniature après plusieurs tentatives: %w", lastErr)
}

// GetVideoDuration retourne la durée d'une vidéo en secondes
func (p *Processor) GetVideoDuration(videoPath string) (int, error) {
	cmd := exec.Command(
		p.ffmpegPath,
		"-i", videoPath,
		"-hide_banner",
		"-show_entries", "format=duration",
		"-v", "quiet",
		"-of", "csv=p=0",
	)

	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("erreur lors de la récupération de la durée: %w", err)
	}

	var duration float64
	if _, err := fmt.Sscanf(string(output), "%f", &duration); err != nil {
		return 0, fmt.Errorf("erreur lors du parsing de la durée: %w", err)
	}

	return int(duration), nil
}

// TranscodeToHLS transcodage une vidéo en format HLS
func (p *Processor) TranscodeToHLS(videoPath string, videoID uuid.UUID, quality string) (string, string, error) {
	outputDir := filepath.Join(p.storagePath, "hls", videoID.String())
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", "", fmt.Errorf("erreur lors de la création du dossier: %w", err)
	}

	manifestPath := filepath.Join(outputDir, "playlist.m3u8")
	segmentPattern := filepath.Join(outputDir, "segment_%03d.ts")

	// Définir les paramètres selon la qualité
	var resolution, bitrate string
	switch quality {
	case "1080p":
		resolution = "1920:1080"
		bitrate = "5000k"
	case "720p":
		resolution = "1280:720"
		bitrate = "2500k"
	case "480p":
		resolution = "854:480"
		bitrate = "1000k"
	case "360p":
		resolution = "640:360"
		bitrate = "500k"
	default:
		resolution = "1280:720"
		bitrate = "2500k"
	}

	// Détecter et gérer tous les formats d'entrée pour HLS
	cmd := exec.Command(
		p.ffmpegPath,
		"-i", videoPath,
		"-c:v", "libx264",              // Codec vidéo H.264
		"-preset", "medium",             // Équilibre vitesse/qualité
		"-crf", "23",                    // Qualité constante
		"-c:a", "aac",                   // Codec audio AAC
		"-b:a", "128k",                  // Bitrate audio
		"-b:v", bitrate,                 // Bitrate vidéo
		"-s", resolution,                // Résolution
		"-pix_fmt", "yuv420p",           // Format de pixels compatible
		"-hls_time", "10",               // Durée des segments (10 secondes)
		"-hls_playlist_type", "vod",     // Type de playlist (Video On Demand)
		"-hls_segment_filename", segmentPattern,
		"-y",                            // Écraser si existe
		manifestPath,
	)

	if err := cmd.Run(); err != nil {
		return "", "", fmt.Errorf("erreur lors du transcodage HLS: %w", err)
	}

	return manifestPath, outputDir, nil
}

// TranscodeToDASH transcodage une vidéo en format DASH
func (p *Processor) TranscodeToDASH(videoPath string, videoID uuid.UUID, quality string) (string, string, error) {
	outputDir := filepath.Join(p.storagePath, "dash", videoID.String())
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", "", fmt.Errorf("erreur lors de la création du dossier: %w", err)
	}

	manifestPath := filepath.Join(outputDir, "manifest.mpd")

	// Définir les paramètres selon la qualité
	var resolution, bitrate string
	switch quality {
	case "1080p":
		resolution = "1920:1080"
		bitrate = "5000k"
	case "720p":
		resolution = "1280:720"
		bitrate = "2500k"
	case "480p":
		resolution = "854:480"
		bitrate = "1000k"
	case "360p":
		resolution = "640:360"
		bitrate = "500k"
	default:
		resolution = "1280:720"
		bitrate = "2500k"
	}

	cmd := exec.Command(
		p.ffmpegPath,
		"-i", videoPath,
		"-c:v", "libx264",
		"-c:a", "aac",
		"-b:v", bitrate,
		"-s", resolution,
		"-f", "dash",
		"-use_timeline", "1",
		"-use_template", "1",
		manifestPath,
	)

	if err := cmd.Run(); err != nil {
		return "", "", fmt.Errorf("erreur lors du transcodage DASH: %w", err)
	}

	return manifestPath, outputDir, nil
}

// TranscodeToProgressive transcodage une vidéo en format progressif (MP4)
func (p *Processor) TranscodeToProgressive(videoPath string, videoID uuid.UUID, quality string) (string, error) {
	outputFileName := fmt.Sprintf("%s_%s.mp4", videoID.String(), quality)
	outputPath := filepath.Join(p.storagePath, "progressive", outputFileName)

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return "", fmt.Errorf("erreur lors de la création du dossier: %w", err)
	}

	// Définir les paramètres selon la qualité
	var resolution, bitrate string
	switch quality {
	case "1080p":
		resolution = "1920:1080"
		bitrate = "5000k"
	case "720p":
		resolution = "1280:720"
		bitrate = "2500k"
	case "480p":
		resolution = "854:480"
		bitrate = "1000k"
	case "360p":
		resolution = "640:360"
		bitrate = "500k"
	default:
		resolution = "1280:720"
		bitrate = "2500k"
	}

	// Détecter et gérer tous les formats d'entrée (AVI, MKV, MOV, MP4, WebM, etc.)
	// FFmpeg gère automatiquement la conversion
	cmd := exec.Command(
		p.ffmpegPath,
		"-i", videoPath,
		"-c:v", "libx264",           // Codec vidéo H.264 (compatible web)
		"-preset", "medium",          // Équilibre vitesse/qualité
		"-crf", "23",                 // Qualité constante (23 = bon équilibre)
		"-c:a", "aac",                // Codec audio AAC (compatible web)
		"-b:a", "128k",               // Bitrate audio
		"-b:v", bitrate,              // Bitrate vidéo selon la qualité
		"-s", resolution,             // Résolution
		"-movflags", "+faststart",    // Optimisation pour streaming web
		"-pix_fmt", "yuv420p",        // Format de pixels compatible (nécessaire pour web)
		"-y",                         // Écraser le fichier de sortie s'il existe
		outputPath,
	)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("erreur lors du transcodage progressif: %w", err)
	}

	return outputPath, nil
}

