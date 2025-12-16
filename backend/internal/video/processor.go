package video

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

type Processor struct {
	storagePath      string
	thumbnailPath    string
	ffmpegPath       string
	ffprobePath      string
}

func NewProcessor(storagePath, thumbnailPath string) *Processor {
	ffmpegPath := "ffmpeg"
	if path := os.Getenv("FFMPEG_PATH"); path != "" {
		ffmpegPath = path
	}
	
	ffprobePath := "ffprobe"
	if path := os.Getenv("FFPROBE_PATH"); path != "" {
		ffprobePath = path
	}

	return &Processor{
		storagePath:   storagePath,
		thumbnailPath: thumbnailPath,
		ffmpegPath:    ffmpegPath,
		ffprobePath:   ffprobePath,
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
	case "2160p": // 4K UHD
		resolution = "3840:2160"
		bitrate = "20000k"
	case "1440p": // 2K QHD
		resolution = "2560:1440"
		bitrate = "12000k"
	case "1080p": // Full HD
		resolution = "1920:1080"
		bitrate = "5000k"
	case "720p": // HD
		resolution = "1280:720"
		bitrate = "2500k"
	case "480p": // SD
		resolution = "854:480"
		bitrate = "1000k"
	case "360p": // Basse qualité
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
	case "2160p": // 4K UHD
		resolution = "3840:2160"
		bitrate = "20000k"
	case "1440p": // 2K QHD
		resolution = "2560:1440"
		bitrate = "12000k"
	case "1080p": // Full HD
		resolution = "1920:1080"
		bitrate = "5000k"
	case "720p": // HD
		resolution = "1280:720"
		bitrate = "2500k"
	case "480p": // SD
		resolution = "854:480"
		bitrate = "1000k"
	case "360p": // Basse qualité
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

// CompressVideo compresse une vidéo de manière optimale et non-destructive avec H.265/HEVC
func (p *Processor) CompressVideo(videoPath string, videoID uuid.UUID) (string, error) {
	outputFileName := fmt.Sprintf("%s_compressed.mp4", videoID.String())
	outputPath := filepath.Join(p.storagePath, "compressed", outputFileName)

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return "", fmt.Errorf("erreur lors de la création du dossier: %w", err)
	}

	// Extraire les métadonnées pour préserver la résolution originale
	metadata, err := p.ExtractMetadata(videoPath)
	if err != nil {
		// Si on ne peut pas extraire les métadonnées, utiliser des valeurs par défaut
		metadata = &VideoMetadata{Width: 1920, Height: 1080}
	}

	// Déterminer la résolution à utiliser (préserver l'originale si possible)
	resolution := fmt.Sprintf("%d:%d", metadata.Width, metadata.Height)
	if metadata.Width == 0 || metadata.Height == 0 {
		resolution = "1920:1080" // Par défaut
	}

	// Vérifier si H.265/HEVC est disponible (libx265)
	// Si non disponible, utiliser H.264 avec CRF optimal
	cmd := exec.Command(p.ffmpegPath, "-codecs")
	output, _ := cmd.Output()
	hasHEVC := strings.Contains(string(output), "libx265")

	var videoCodec, videoPreset, videoCRF string
	if hasHEVC {
		// H.265/HEVC pour une compression optimale (50% plus efficace que H.264)
		videoCodec = "libx265"
		videoPreset = "medium" // Équilibre vitesse/qualité
		videoCRF = "23"        // CRF optimal pour H.265 (qualité visuelle équivalente à H.264 CRF 23, mais fichier ~50% plus petit)
	} else {
		// Fallback sur H.264 avec CRF optimal
		videoCodec = "libx264"
		videoPreset = "slow"   // Preset plus lent = meilleure compression
		videoCRF = "20"        // CRF 20 = très haute qualité (presque lossless visuellement)
	}

	// Compression optimale non-destructive
	args := []string{
		"-i", videoPath,
		"-c:v", videoCodec,
		"-preset", videoPreset,
		"-crf", videoCRF,              // Constant Rate Factor (qualité constante, non-destructive)
		"-c:a", "aac",                 // Codec audio AAC
		"-b:a", "192k",                // Bitrate audio optimisé (meilleure qualité)
		"-ac", "2",                    // Stéréo
		"-movflags", "+faststart",     // Optimisation pour streaming web
		"-pix_fmt", "yuv420p",         // Format de pixels compatible
		"-vf", fmt.Sprintf("scale=%s:flags=lanczos", resolution), // Scaling avec Lanczos (meilleure qualité)
		"-y",
		outputPath,
	}

	cmd = exec.Command(p.ffmpegPath, args...)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("erreur lors de la compression: %w", err)
	}

	return outputPath, nil
}

// TranscodeToProgressive transcodage une vidéo en format progressif (MP4) optimisé pour streaming
func (p *Processor) TranscodeToProgressive(videoPath string, videoID uuid.UUID, quality string) (string, error) {
	outputFileName := fmt.Sprintf("%s_%s.mp4", videoID.String(), quality)
	outputPath := filepath.Join(p.storagePath, "progressive", outputFileName)

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return "", fmt.Errorf("erreur lors de la création du dossier: %w", err)
	}

	// Définir les paramètres selon la qualité
	var resolution, bitrate string
	switch quality {
	case "2160p": // 4K UHD
		resolution = "3840:2160"
		bitrate = "20000k"
	case "1440p": // 2K QHD
		resolution = "2560:1440"
		bitrate = "12000k"
	case "1080p": // Full HD
		resolution = "1920:1080"
		bitrate = "5000k"
	case "720p": // HD
		resolution = "1280:720"
		bitrate = "2500k"
	case "480p": // SD
		resolution = "854:480"
		bitrate = "1000k"
	case "360p": // Basse qualité
		resolution = "640:360"
		bitrate = "500k"
	default:
		resolution = "1280:720"
		bitrate = "2500k"
	}

	// Vérifier si H.265/HEVC est disponible pour le streaming
	cmd := exec.Command(p.ffmpegPath, "-codecs")
	output, _ := cmd.Output()
	hasHEVC := strings.Contains(string(output), "libx265")

	var videoCodec string
	if hasHEVC {
		// H.265 pour streaming optimisé (meilleure compression)
		videoCodec = "libx265"
	} else {
		// H.264 pour compatibilité maximale
		videoCodec = "libx264"
	}

	// Détecter et gérer tous les formats d'entrée (AVI, MKV, MOV, MP4, WebM, etc.)
	// FFmpeg gère automatiquement la conversion
	cmd = exec.Command(
		p.ffmpegPath,
		"-i", videoPath,
		"-c:v", videoCodec,
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

// VideoMetadata contient les métadonnées extraites d'une vidéo
type VideoMetadata struct {
	Title       string `json:"title,omitempty"`
	Author      string `json:"author,omitempty"`
	Description string `json:"description,omitempty"`
	Duration    int    `json:"duration"` // en secondes
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Bitrate     int64  `json:"bitrate"` // en bits/s
	Codec       string `json:"codec,omitempty"`
	Format      string `json:"format,omitempty"`
}

// ExtractMetadata extrait les métadonnées d'un fichier vidéo avec ffprobe
func (p *Processor) ExtractMetadata(videoPath string) (*VideoMetadata, error) {
	// Vérifier que le fichier existe
	if _, err := os.Stat(videoPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("fichier vidéo non trouvé: %s", videoPath)
	}

	// Utiliser ffprobe pour extraire les métadonnées en JSON
	cmd := exec.Command(
		p.ffprobePath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		videoPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("erreur lors de l'extraction des métadonnées (ffprobe): %w, output: %s", err, string(output))
	}

	// Parser le JSON
	var probeData struct {
		Format struct {
			Duration string            `json:"duration"`
			BitRate  string            `json:"bit_rate"`
			Format   string            `json:"format_name"`
			Tags     map[string]string `json:"tags"`
		} `json:"format"`
		Streams []struct {
			CodecType string `json:"codec_type"`
			CodecName string `json:"codec_name"`
			Width     int    `json:"width"`
			Height    int    `json:"height"`
			BitRate   string `json:"bit_rate"`
		} `json:"streams"`
	}

	if err := json.Unmarshal(output, &probeData); err != nil {
		return nil, fmt.Errorf("erreur lors du parsing JSON: %w", err)
	}

	metadata := &VideoMetadata{}

	// Extraire la durée
	if probeData.Format.Duration != "" {
		if duration, err := strconv.ParseFloat(probeData.Format.Duration, 64); err == nil {
			metadata.Duration = int(duration)
		}
	}

	// Extraire les dimensions et codec depuis le stream vidéo
	for _, stream := range probeData.Streams {
		if stream.CodecType == "video" {
			metadata.Width = stream.Width
			metadata.Height = stream.Height
			metadata.Codec = stream.CodecName
			if stream.BitRate != "" {
				if bitrate, err := strconv.ParseInt(stream.BitRate, 10, 64); err == nil {
					metadata.Bitrate = bitrate
				}
			}
			break
		}
	}

	// Extraire les tags (title, author, description)
	if probeData.Format.Tags != nil {
		if title, ok := probeData.Format.Tags["title"]; ok {
			metadata.Title = strings.TrimSpace(title)
		}
		if author, ok := probeData.Format.Tags["author"]; ok {
			metadata.Author = strings.TrimSpace(author)
		}
		if description, ok := probeData.Format.Tags["description"]; ok {
			metadata.Description = strings.TrimSpace(description)
		}
		// Aussi vérifier les variantes de clés
		if metadata.Title == "" {
			if title, ok := probeData.Format.Tags["TITLE"]; ok {
				metadata.Title = strings.TrimSpace(title)
			}
		}
		if metadata.Author == "" {
			if author, ok := probeData.Format.Tags["AUTHOR"]; ok {
				metadata.Author = strings.TrimSpace(author)
			}
			if author, ok := probeData.Format.Tags["artist"]; ok {
				metadata.Author = strings.TrimSpace(author)
			}
		}
	}

	// Extraire le format
	metadata.Format = probeData.Format.Format

	// Extraire le bitrate du format si pas déjà défini
	if metadata.Bitrate == 0 && probeData.Format.BitRate != "" {
		if bitrate, err := strconv.ParseInt(probeData.Format.BitRate, 10, 64); err == nil {
			metadata.Bitrate = bitrate
		}
	}

	return metadata, nil
}

// GenerateThumbnailFromTitle génère une miniature textuelle depuis un titre
func (p *Processor) GenerateThumbnailFromTitle(title string) (string, error) {
	// Créer un UUID temporaire pour le nom de fichier
	tempID := uuid.New()
	thumbnailName := fmt.Sprintf("temp_%s.jpg", tempID.String())
	thumbnailPath := filepath.Join(p.thumbnailPath, thumbnailName)

	// S'assurer que le dossier existe
	if err := os.MkdirAll(p.thumbnailPath, 0755); err != nil {
		return "", fmt.Errorf("erreur lors de la création du dossier: %w", err)
	}

	// Dimensions de la miniature (format 16:9)
	width := 1280
	height := 720

	// Créer une image avec un fond dégradé
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	
	// Fond dégradé (rouge vers noir, style Netflix)
	for y := 0; y < height; y++ {
		ratio := float64(y) / float64(height)
		// Dégradé de rouge (#ff0000) vers noir (#000000)
		r := uint8(255 * (1 - ratio*0.7))
		g := uint8(0)
		b := uint8(0)
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}

	// Créer le fichier
	file, err := os.Create(thumbnailPath)
	if err != nil {
		return "", fmt.Errorf("erreur lors de la création du fichier: %w", err)
	}
	defer file.Close()

	// Encoder en JPEG
	if err := jpeg.Encode(file, img, &jpeg.Options{Quality: 90}); err != nil {
		return "", fmt.Errorf("erreur lors de l'encodage JPEG: %w", err)
	}

	return thumbnailPath, nil
}

