package video

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

type DVDTrack struct {
	Index     int    `json:"index"`
	Type      string `json:"type"` // video, audio, subtitle
	Language  string `json:"language"`
	Codec     string `json:"codec"`
	Duration  int    `json:"duration"` // en secondes
	Size      int64  `json:"size"`
	Title     string `json:"title,omitempty"`
}

type DVDInfo struct {
	Title       string     `json:"title"`
	Tracks      []DVDTrack `json:"tracks"`
	VideoTracks []DVDTrack `json:"video_tracks"`
	AudioTracks []DVDTrack `json:"audio_tracks"`
	SubtitleTracks []DVDTrack `json:"subtitle_tracks"`
}

type DVDExtractor struct {
	ffmpegPath string
	ffprobePath string
}

func NewDVDExtractor() *DVDExtractor {
	ffmpegPath := "ffmpeg"
	if path := os.Getenv("FFMPEG_PATH"); path != "" {
		ffmpegPath = path
	}
	
	ffprobePath := "ffprobe"
	if path := os.Getenv("FFPROBE_PATH"); path != "" {
		ffprobePath = path
	}

	return &DVDExtractor{
		ffmpegPath:  ffmpegPath,
		ffprobePath: ffprobePath,
	}
}

// DetectDVDDevice détecte le lecteur DVD disponible
func (e *DVDExtractor) DetectDVDDevice() (string, error) {
	// Méthode 1: Utiliser lsblk pour détecter les lecteurs optiques
	cmd := exec.Command("lsblk", "-d", "-n", "-o", "NAME,TYPE")
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) >= 2 && fields[1] == "rom" {
				deviceName := fields[0]
				devicePath := fmt.Sprintf("/dev/%s", deviceName)
				// Vérifier que le device existe et est accessible
				if info, err := os.Stat(devicePath); err == nil && !info.IsDir() {
					// Vérifier avec file si disponible
					fileCmd := exec.Command("file", devicePath)
					fileOutput, fileErr := fileCmd.Output()
					if fileErr == nil {
						outputStr := strings.ToLower(string(fileOutput))
						if strings.Contains(outputStr, "iso") || 
						   strings.Contains(outputStr, "dvd") ||
						   strings.Contains(outputStr, "cd-rom") ||
						   strings.Contains(outputStr, "block special") {
							return devicePath, nil
						}
					} else {
						// Si file n'est pas disponible, accepter le device
						return devicePath, nil
					}
				}
			}
		}
	}

	// Méthode 2: Essayer différents chemins de lecteurs DVD courants
	possibleDevices := []string{
		"/dev/dvd",
		"/dev/cdrom",
		"/dev/sr0",
		"/dev/sr1",
		"/dev/sr2",
		"/dev/cdrom0",
		"/dev/cdrom1",
		"/mnt/cdrom",
		"/media/cdrom",
		"/media/cdrom0",
		"/media/cdrom1",
	}

	for _, device := range possibleDevices {
		if info, err := os.Stat(device); err == nil {
			// Vérifier si c'est un fichier de device ou un dossier monté
			if info.IsDir() {
				// C'est un point de montage, vérifier s'il contient des fichiers DVD
				files, err := os.ReadDir(device)
				if err == nil && len(files) > 0 {
					// Vérifier la présence de fichiers typiques d'un DVD
					hasVideoTS := false
					for _, file := range files {
						if file.Name() == "VIDEO_TS" || file.Name() == "video_ts" {
							hasVideoTS = true
							break
						}
					}
					if hasVideoTS {
						return device, nil
					}
				}
			} else {
				// C'est un device file, vérifier avec la commande file
				cmd := exec.Command("file", device)
				output, err := cmd.Output()
				if err == nil {
					outputStr := strings.ToLower(string(output))
					if strings.Contains(outputStr, "iso") || 
					   strings.Contains(outputStr, "dvd") ||
					   strings.Contains(outputStr, "cd-rom") ||
					   strings.Contains(outputStr, "block special") {
						return device, nil
					}
				}
				// Si file n'est pas disponible, vérifier les permissions
				if info.Mode()&os.ModeDevice != 0 {
					return device, nil
				}
			}
		}
	}

	return "", fmt.Errorf("aucun lecteur DVD détecté. Assurez-vous qu'un DVD/CD est inséré et que le lecteur est accessible. Vérifiez aussi que le conteneur Docker a accès aux devices (/dev/dvd, /dev/cdrom, etc.)")
}

// GetDVDInfo récupère les informations du DVD (pistes audio/vidéo/sous-titres)
func (e *DVDExtractor) GetDVDInfo(devicePath string) (*DVDInfo, error) {
	// Utiliser ffprobe pour analyser le DVD
	cmd := exec.Command(
		e.ffprobePath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		devicePath,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("erreur lors de l'analyse du DVD: %w", err)
	}

	var probeData struct {
		Format struct {
			Tags map[string]string `json:"tags"`
		} `json:"format"`
		Streams []struct {
			Index     int               `json:"index"`
			CodecType string            `json:"codec_type"`
			CodecName string            `json:"codec_name"`
			Tags      map[string]string `json:"tags"`
			Duration  string            `json:"duration"`
			Width     int               `json:"width"`
			Height    int               `json:"height"`
		} `json:"streams"`
	}

	if err := json.Unmarshal(output, &probeData); err != nil {
		return nil, fmt.Errorf("erreur lors du parsing des données: %w", err)
	}

	// Initialiser tous les tableaux pour éviter les valeurs null en JSON
	info := &DVDInfo{
		Title:          probeData.Format.Tags["title"],
		Tracks:         []DVDTrack{},
		VideoTracks:    []DVDTrack{},
		AudioTracks:    []DVDTrack{},
		SubtitleTracks: []DVDTrack{},
	}

	// Si pas de titre, utiliser un titre par défaut
	if info.Title == "" {
		info.Title = "DVD Extraite"
	}

	for _, stream := range probeData.Streams {
		track := DVDTrack{
			Index: stream.Index,
			Codec: stream.CodecName,
		}

		// Extraire la langue depuis les tags si disponible
		language := ""
		if stream.Tags != nil {
			if lang, ok := stream.Tags["language"]; ok {
				language = lang
			}
		}

		switch stream.CodecType {
		case "video":
			track.Type = "video"
			info.VideoTracks = append(info.VideoTracks, track)
		case "audio":
			track.Type = "audio"
			track.Language = language
			info.AudioTracks = append(info.AudioTracks, track)
		case "subtitle":
			track.Type = "subtitle"
			track.Language = language
			info.SubtitleTracks = append(info.SubtitleTracks, track)
		}

		info.Tracks = append(info.Tracks, track)
	}

	// S'assurer qu'il y a au moins une piste vidéo
	if len(info.VideoTracks) == 0 {
		// Créer une piste vidéo par défaut
		info.VideoTracks = []DVDTrack{
			{
				Index: 0,
				Type:  "video",
				Codec: "unknown",
			},
		}
	}

	return info, nil
}

// ExtractDVD extrait le DVD avec les pistes sélectionnées
func (e *DVDExtractor) ExtractDVD(
	devicePath string,
	outputPath string,
	videoTrackIndex int,
	audioTrackIndices []int,
	subtitleTrackIndices []int,
	progressCallback func(progress float64),
) error {
	// Construire la commande ffmpeg
	args := []string{
		"-i", devicePath,
		"-map", fmt.Sprintf("0:v:%d", videoTrackIndex),
		"-c:v", "libx264",
		"-preset", "medium",
		"-crf", "23",
	}

	// Ajouter les pistes audio
	for _, audioIdx := range audioTrackIndices {
		args = append(args, "-map", fmt.Sprintf("0:a:%d", audioIdx))
		args = append(args, "-c:a", "aac", "-b:a", "192k")
	}

	// Ajouter les sous-titres
	for _, subIdx := range subtitleTrackIndices {
		args = append(args, "-map", fmt.Sprintf("0:s:%d", subIdx))
		args = append(args, "-c:s", "mov_text")
	}

	// Format de sortie
	args = append(args,
		"-movflags", "+faststart",
		"-y", // Écraser si existe
		outputPath,
	)

	cmd := exec.Command(e.ffmpegPath, args...)
	
	// Capturer la progression
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("erreur lors de la création du pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("erreur lors du démarrage de l'extraction: %w", err)
	}

	// Lire la progression (simplifié - dans un vrai cas, il faudrait parser la sortie)
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stderr.Read(buf)
			if n > 0 && progressCallback != nil {
				// Parser la progression depuis la sortie ffmpeg
				// Pour simplifier, on envoie une progression estimée
				progressCallback(0.5) // 50% pour l'instant
			}
			if err != nil {
				break
			}
		}
	}()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("erreur lors de l'extraction: %w", err)
	}

	if progressCallback != nil {
		progressCallback(1.0) // 100%
	}

	return nil
}

// ExtractDVDToHLS extrait le DVD et le convertit directement en HLS
func (e *DVDExtractor) ExtractDVDToHLS(
	devicePath string,
	videoID uuid.UUID,
	outputDir string,
	videoTrackIndex int,
	audioTrackIndices []int,
	subtitleTrackIndices []int,
	qualities []string,
) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("erreur lors de la création du dossier: %w", err)
	}

	// Extraire d'abord en fichier temporaire
	tempPath := filepath.Join(outputDir, "temp_extract.mkv")
	if err := e.ExtractDVD(devicePath, tempPath, videoTrackIndex, audioTrackIndices, subtitleTrackIndices, nil); err != nil {
		return err
	}
	defer os.Remove(tempPath)

	// Ensuite transcoder en HLS pour chaque qualité
	// Utiliser le storage path parent pour le processor
	storagePath := filepath.Dir(outputDir)
	thumbnailPath := filepath.Join(storagePath, "thumbnails")
	processor := NewProcessor(storagePath, thumbnailPath)
	
	for _, quality := range qualities {
		_, _, err := processor.TranscodeToHLS(tempPath, videoID, quality)
		if err != nil {
			return fmt.Errorf("erreur lors du transcodage HLS %s: %w", quality, err)
		}
	}

	return nil
}

