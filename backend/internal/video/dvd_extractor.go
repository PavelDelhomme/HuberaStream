package video

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type DVDTrack struct {
	Index     int    `json:"index"`
	Type      string `json:"type"` // video, audio, subtitle
	Language  string `json:"language"`
	Codec     string `json:"codec"`
	Duration  int    `json:"duration"` // en minutes
	Size      int64  `json:"size"`
	Title     string `json:"title,omitempty"`
	Channels  int    `json:"channels,omitempty"`
	SampleRate string `json:"sample_rate,omitempty"`
}

type DVDInfo struct {
	Title          string     `json:"title"`
	Director       string     `json:"director,omitempty"`
	Year           int        `json:"year,omitempty"`
	PosterURL      string     `json:"poster_url,omitempty"`
	Tracks         []DVDTrack `json:"tracks"`
	VideoTracks    []DVDTrack `json:"video_tracks"`
	AudioTracks    []DVDTrack `json:"audio_tracks"`
	SubtitleTracks []DVDTrack `json:"subtitle_tracks"`
}

type DVDExtractor struct {
	ffmpegPath  string
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
	var detectedDevices []string
	
	// Méthode 1: Utiliser lsblk pour détecter les lecteurs optiques (plus fiable)
	cmd := exec.Command("lsblk", "-d", "-n", "-o", "NAME,TYPE,MODEL")
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) >= 2 && fields[1] == "rom" {
				deviceName := fields[0]
				devicePath := fmt.Sprintf("/dev/%s", deviceName)
				// Vérifier que le device existe et est accessible
				if info, err := os.Stat(devicePath); err == nil && !info.IsDir() {
					// Vérifier les permissions (doit être un device block)
					if info.Mode()&os.ModeDevice != 0 {
						detectedDevices = append(detectedDevices, devicePath)
					}
				}
			}
		}
	}
	
	// Si on a trouvé des devices avec lsblk, retourner le premier
	if len(detectedDevices) > 0 {
		return detectedDevices[0], nil
	}

	// Méthode 2: Essayer différents chemins de lecteurs DVD courants
	possibleDevices := []string{
		"/dev/sr0",
		"/dev/sr1",
		"/dev/sr2",
		"/dev/sr3", // Ajouté pour les lecteurs USB
		"/dev/dvd",
		"/dev/cdrom",
		"/dev/cdrom0",
		"/dev/cdrom1",
	}

	for _, device := range possibleDevices {
		if info, err := os.Stat(device); err == nil {
			// Vérifier si c'est un device block
			if info.Mode()&os.ModeDevice != 0 {
				// Vérifier que c'est un device block (pas un dossier)
				if !info.IsDir() {
					detectedDevices = append(detectedDevices, device)
				}
			}
		}
	}

	// Méthode 3: Vérifier les points de montage
	mountPoints := []string{
		"/mnt/cdrom",
		"/media/cdrom",
		"/media/cdrom0",
		"/media/cdrom1",
	}

	for _, mountPoint := range mountPoints {
		if info, err := os.Stat(mountPoint); err == nil && info.IsDir() {
			// Vérifier s'il contient des fichiers DVD
			files, err := os.ReadDir(mountPoint)
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
					return mountPoint, nil
				}
			}
		}
	}

	// Si on a trouvé des devices, retourner le premier
	if len(detectedDevices) > 0 {
		return detectedDevices[0], nil
	}

	return "", fmt.Errorf("aucun lecteur DVD détecté. Assurez-vous qu'un DVD/CD est inséré et que le lecteur est accessible. Vérifiez aussi que le conteneur Docker a accès aux devices (/dev/sr0, /dev/dvd, /dev/cdrom, etc.)")
}

// GetDVDInfo récupère les informations du DVD (pistes audio/vidéo/sous-titres)
func (e *DVDExtractor) GetDVDInfo(devicePath string) (*DVDInfo, error) {
	// Utiliser directement le device avec ffprobe (plus fiable que dvd://)
	var ffprobeInput string
	if strings.HasPrefix(devicePath, "/dev/") {
		// Utiliser directement le device block
		ffprobeInput = devicePath
	} else if strings.Contains(devicePath, "VIDEO_TS") {
		// Chemin vers VIDEO_TS
		ffprobeInput = devicePath
	} else {
		ffprobeInput = devicePath
	}

	// Utiliser ffprobe avec des options spécifiques pour les DVD
	cmd := exec.Command(
		e.ffprobePath,
		"-v", "error",
		"-analyzeduration", "30000000", // 30 secondes d'analyse
		"-probesize", "30000000",       // 30 MB de probe
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		"-show_chapters",
		"-show_programs",
		ffprobeInput,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("erreur lors de l'analyse du DVD: %w (output: %s)", err, string(output))
	}

	var probeData struct {
		Format struct {
			Duration string            `json:"duration"`
			Tags     map[string]string `json:"tags"`
		} `json:"format"`
		Streams []struct {
			Index          int               `json:"index"`
			CodecType      string            `json:"codec_type"`
			CodecName      string            `json:"codec_name"`
			CodecLongName  string            `json:"codec_long_name"`
			Tags           map[string]string `json:"tags"`
			Duration       string            `json:"duration"`
			Width          int               `json:"width"`
			Height         int               `json:"height"`
			BitRate        string            `json:"bit_rate"`
			SampleRate     string            `json:"sample_rate"`
			Channels       int               `json:"channels"`
			ChannelLayout  string            `json:"channel_layout"`
			Disposition    map[string]int    `json:"disposition"`
		} `json:"streams"`
		Chapters []struct {
			ID        int    `json:"id"`
			TimeBase  string `json:"time_base"`
			Start     int    `json:"start"`
			StartTime string `json:"start_time"`
			End       int    `json:"end"`
			EndTime   string `json:"end_time"`
			Tags      map[string]string `json:"tags"`
		} `json:"chapters"`
	}

	if err := json.Unmarshal(output, &probeData); err != nil {
		return nil, fmt.Errorf("erreur lors du parsing des données: %w (output: %s)", err, string(output))
	}

	// Initialiser tous les tableaux pour éviter les valeurs null en JSON
	info := &DVDInfo{
		Title:          "",
		Director:       "",
		Year:           0,
		PosterURL:      "",
		Tracks:         []DVDTrack{},
		VideoTracks:    []DVDTrack{},
		AudioTracks:    []DVDTrack{},
		SubtitleTracks: []DVDTrack{},
	}

	// Extraire le titre depuis les tags du format
	if probeData.Format.Tags != nil {
		// Essayer plusieurs clés pour le titre
		if title, ok := probeData.Format.Tags["title"]; ok && title != "" {
			info.Title = title
		} else if title, ok := probeData.Format.Tags["TITLE"]; ok && title != "" {
			info.Title = title
		} else if title, ok := probeData.Format.Tags["Title"]; ok && title != "" {
			info.Title = title
		}
	}

	// Utiliser ffmpeg pour obtenir plus d'informations (titre, réalisateur, langues, détails complets)
	var ffmpegOutputStr string
	var ffmpegInput string
	
	if strings.HasPrefix(devicePath, "/dev/") {
		// Utiliser directement le device
		ffmpegInput = devicePath
	} else if strings.Contains(devicePath, "VIDEO_TS") {
		ffmpegInput = devicePath
	} else {
		ffmpegInput = devicePath
	}
	
	// Utiliser ffmpeg avec le device directement
	ffmpegCmd := exec.Command(
		e.ffmpegPath,
		"-i", ffmpegInput,
		"-t", "0.5", // Augmenter à 0.5s pour plus d'infos
		"-f", "null",
		"-",
	)
	ffmpegOutput, ffmpegErr := ffmpegCmd.CombinedOutput()
	ffmpegOutputStr = string(ffmpegOutput)
	
	// Toujours parser la sortie ffmpeg même s'il y a des erreurs (les erreurs AC3 sont normales)
	if ffmpegOutputStr != "" {
		// Si pas de titre, essayer d'extraire depuis ffmpeg output
		if info.Title == "" {
			title := e.extractTitleFromFFmpegOutput(ffmpegOutputStr)
			if title != "" {
				info.Title = title
			}
		}

		// Extraire le réalisateur depuis ffmpeg output
		director := e.extractDirectorFromFFmpegOutput(ffmpegOutputStr)
		if director != "" {
			info.Director = director
		}
	}

	// Si pas de titre, utiliser un titre par défaut
	if info.Title == "" {
		info.Title = "DVD Extraite"
	}

	// Parser les streams depuis ffmpeg output pour obtenir les langues et détails
	var streamLanguages map[int]string
	var streamDetails map[int]map[string]string
	if ffmpegOutputStr != "" {
		streamLanguages = e.parseStreamLanguagesFromFFmpeg(ffmpegOutputStr)
		streamDetails = e.parseStreamDetailsFromFFmpeg(ffmpegOutputStr)
	} else {
		streamLanguages = make(map[int]string)
		streamDetails = make(map[int]map[string]string)
	}

	// Compteurs pour les indices de pistes par type
	videoIdx := 0
	audioIdx := 0
	subtitleIdx := 0

	// Map pour stocker les langues détectées
	languageMap := make(map[string]string)

	for _, stream := range probeData.Streams {
		// Extraire la langue depuis les tags
		language := ""
		title := ""
		if stream.Tags != nil {
			if lang, ok := stream.Tags["language"]; ok && lang != "" {
				language = lang
			} else if lang, ok := stream.Tags["LANGUAGE"]; ok && lang != "" {
				language = lang
			} else if lang, ok := stream.Tags["lang"]; ok && lang != "" {
				language = lang
			}
			if t, ok := stream.Tags["title"]; ok && t != "" {
				title = t
			} else if t, ok := stream.Tags["TITLE"]; ok && t != "" {
				title = t
			}
		}

		// Si pas de langue dans les tags, utiliser celle extraite depuis ffmpeg
		if language == "" {
			if lang, ok := streamLanguages[stream.Index]; ok {
				language = lang
			} else if details, ok := streamDetails[stream.Index]; ok {
				if lang, ok := details["language"]; ok {
					language = lang
				}
			} else {
				// Essayer de la détecter depuis le codec_long_name
				lang := e.detectLanguageFromCodec(stream.CodecLongName, stream.Index)
				if lang != "" {
					language = lang
				}
			}
		}

		// Convertir la durée en minutes si disponible
		duration := 0
		if stream.Duration != "" {
			if d, err := strconv.ParseFloat(stream.Duration, 64); err == nil {
				duration = int(d / 60) // Convertir en minutes
			}
		} else if probeData.Format.Duration != "" {
			if d, err := strconv.ParseFloat(probeData.Format.Duration, 64); err == nil {
				duration = int(d / 60)
			}
		}

		track := DVDTrack{
			Index:      stream.Index,
			Codec:     stream.CodecName,
			Language:  language,
			Title:     title,
			Duration:  duration,
			Channels:  stream.Channels,
			SampleRate: stream.SampleRate,
		}

		// Enrichir avec les détails depuis ffmpeg si disponibles
		if details, ok := streamDetails[stream.Index]; ok {
			if lang, ok := details["language"]; ok && language == "" {
				language = lang
				track.Language = lang
			}
			if ch, ok := details["channels"]; ok && stream.Channels == 0 {
				if chInt, err := strconv.Atoi(ch); err == nil {
					track.Channels = chInt
				}
			}
			if sr, ok := details["sample_rate"]; ok && stream.SampleRate == "" {
				track.SampleRate = sr
			}
		}

		switch stream.CodecType {
		case "video":
			track.Type = "video"
			// Ajouter des informations supplémentaires pour la vidéo
			if stream.Width > 0 && stream.Height > 0 {
				// Enrichir avec les détails depuis ffmpeg si disponibles
				if details, ok := streamDetails[stream.Index]; ok {
					if w, ok := details["width"]; ok {
						if wInt, err := strconv.Atoi(w); err == nil && wInt > 0 {
							track.Title = fmt.Sprintf("%dx%d %s", wInt, stream.Height, stream.CodecName)
						} else {
							track.Title = fmt.Sprintf("%dx%d %s", stream.Width, stream.Height, stream.CodecName)
						}
					} else {
						track.Title = fmt.Sprintf("%dx%d %s", stream.Width, stream.Height, stream.CodecName)
					}
				} else {
					track.Title = fmt.Sprintf("%dx%d %s", stream.Width, stream.Height, stream.CodecName)
				}
			} else {
				track.Title = fmt.Sprintf("Piste vidéo principale - %s", stream.CodecName)
			}
			info.VideoTracks = append(info.VideoTracks, track)
			videoIdx++
		case "audio":
			track.Type = "audio"
			// Ajouter des informations sur l'audio
			audioInfo := stream.CodecName
			
			// Utiliser les détails depuis ffmpeg si disponibles
			if details, ok := streamDetails[stream.Index]; ok {
				if ch, ok := details["channels"]; ok {
					if chInt, err := strconv.Atoi(ch); err == nil && chInt > 0 {
						track.Channels = chInt
						audioInfo += fmt.Sprintf(" %dch", chInt)
					}
				}
				if sr, ok := details["sample_rate"]; ok && sr != "" {
					track.SampleRate = sr
					audioInfo += fmt.Sprintf(" %sHz", sr)
				}
			}
			
			// Fallback sur les valeurs de ffprobe
			if track.Channels == 0 && stream.Channels > 0 {
				track.Channels = stream.Channels
				audioInfo += fmt.Sprintf(" %dch", stream.Channels)
			}
			if track.SampleRate == "" && stream.SampleRate != "" {
				track.SampleRate = stream.SampleRate
				audioInfo += fmt.Sprintf(" %sHz", stream.SampleRate)
			}
			
			if language != "" {
				// Convertir le code langue en nom complet
				langName := e.getLanguageName(language)
				audioInfo += fmt.Sprintf(" (%s)", langName)
				track.Language = language
			} else {
				// Essayer de deviner depuis le codec ou l'index
				// Par défaut, première piste audio souvent en français ou anglais
				if audioIdx == 0 {
					audioInfo += " (Français présumé)"
				} else {
					audioInfo += " (Langue inconnue)"
				}
			}
			track.Title = audioInfo
			info.AudioTracks = append(info.AudioTracks, track)
			audioIdx++
			if language != "" {
				languageMap[language] = e.getLanguageName(language)
			}
		case "subtitle":
			track.Type = "subtitle"
			if language != "" {
				langName := e.getLanguageName(language)
				track.Title = fmt.Sprintf("%s (%s)", langName, stream.CodecName)
				track.Language = language
			} else {
				// Essayer de deviner depuis l'index
				if subtitleIdx == 0 {
					track.Title = fmt.Sprintf("Français présumé (%s)", stream.CodecName)
				} else {
					track.Title = fmt.Sprintf("Sous-titre %d (%s)", subtitleIdx+1, stream.CodecName)
				}
			}
			info.SubtitleTracks = append(info.SubtitleTracks, track)
			subtitleIdx++
			if language != "" {
				languageMap[language] = e.getLanguageName(language)
			}
		}

		info.Tracks = append(info.Tracks, track)
	}

	// Si pas de pistes audio détectées, essayer de les trouver avec ffmpeg
	if len(info.AudioTracks) == 0 {
		audioTracks := e.extractAudioTracksFromFFmpeg(devicePath)
		if len(audioTracks) > 0 {
			info.AudioTracks = audioTracks
			for _, track := range audioTracks {
				info.Tracks = append(info.Tracks, track)
			}
		}
	}

	// Si pas de sous-titres détectés, essayer de les trouver avec ffmpeg
	if len(info.SubtitleTracks) == 0 {
		subtitleTracks := e.extractSubtitleTracksFromFFmpeg(devicePath)
		if len(subtitleTracks) > 0 {
			info.SubtitleTracks = subtitleTracks
			for _, track := range subtitleTracks {
				info.Tracks = append(info.Tracks, track)
			}
		}
	}

	// S'assurer qu'il y a au moins une piste vidéo
	if len(info.VideoTracks) == 0 {
		// Créer une piste vidéo par défaut
		info.VideoTracks = []DVDTrack{
			{
				Index: 0,
				Type:  "video",
				Codec: "mpeg2video", // Codec typique des DVD
				Title: "Piste vidéo principale",
			},
		}
	}

	return info, nil
}

// extractTitleFromFFmpegOutput extrait le titre depuis la sortie de ffmpeg
func (e *DVDExtractor) extractTitleFromFFmpegOutput(outputStr string) string {
	// Patterns pour trouver le titre
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)title\s*:\s*([^\n]+)`),
		regexp.MustCompile(`(?i)"title"\s*:\s*"([^"]+)"`),
		regexp.MustCompile(`(?i)Title\s*:\s*([^\n]+)`),
		regexp.MustCompile(`(?i)DVD\s+Title\s*:\s*([^\n]+)`),
	}

	for _, pattern := range patterns {
		matches := pattern.FindStringSubmatch(outputStr)
		if len(matches) > 1 && matches[1] != "" {
			title := strings.TrimSpace(matches[1])
			if title != "" && len(title) < 200 { // Éviter les faux positifs
				return title
			}
		}
	}

	return ""
}

// extractDirectorFromFFmpegOutput extrait le réalisateur depuis la sortie de ffmpeg
func (e *DVDExtractor) extractDirectorFromFFmpegOutput(outputStr string) string {
	// Patterns pour trouver le réalisateur
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)director\s*:\s*([^\n]+)`),
		regexp.MustCompile(`(?i)"director"\s*:\s*"([^"]+)"`),
		regexp.MustCompile(`(?i)Director\s*:\s*([^\n]+)`),
		regexp.MustCompile(`(?i)by\s+([A-Z][a-z]+\s+[A-Z][a-z]+)`), // "by John Doe"
	}

	for _, pattern := range patterns {
		matches := pattern.FindStringSubmatch(outputStr)
		if len(matches) > 1 && matches[1] != "" {
			director := strings.TrimSpace(matches[1])
			if director != "" && len(director) < 100 {
				return director
			}
		}
	}

	return ""
}

// parseStreamLanguagesFromFFmpeg parse les langues depuis la sortie ffmpeg
func (e *DVDExtractor) parseStreamLanguagesFromFFmpeg(outputStr string) map[int]string {
	languages := make(map[int]string)
	
	// Pattern pour Stream #0:X(Y): Audio/Subtitle: ... (Y est souvent la langue)
	// Exemple: Stream #0:3[0x80](fra): Audio: ac3
	pattern := regexp.MustCompile(`Stream\s+#\d+:(\d+)\[.*?\]\(([a-z]{3})\):`)
	matches := pattern.FindAllStringSubmatch(outputStr, -1)
	
	for _, match := range matches {
		if len(match) >= 3 {
			streamIdx, err := strconv.Atoi(match[1])
			if err == nil {
				lang := match[2]
				if lang != "" {
					languages[streamIdx] = lang
				}
			}
		}
	}

	return languages
}

// parseStreamDetailsFromFFmpeg parse les détails complets des streams depuis ffmpeg
func (e *DVDExtractor) parseStreamDetailsFromFFmpeg(outputStr string) map[int]map[string]string {
	details := make(map[int]map[string]string)
	
	// Pattern pour Stream #0:X: Video/Audio/Subtitle avec tous les détails
	// Exemple: Stream #0:1[0x1e0]: Video: mpeg2video (Main), yuv420p(tv, top first), 720x576 [SAR 16:15 DAR 4:3], 2000 kb/s, 25 fps
	// Exemple: Stream #0:3[0x80]: Audio: ac3, 48000 Hz, stereo, fltp, 192 kb/s
	// Exemple: Stream #0:2[0x20]: Subtitle: dvd_subtitle (dvdsub)
	
	// Pattern vidéo amélioré
	videoPattern := regexp.MustCompile(`Stream\s+#\d+:(\d+)\[.*?\]:\s*Video:\s*([^,]+),.*?(\d+)x(\d+)`)
	// Pattern audio amélioré - capturer stereo, mono, 5.1, etc.
	audioPattern := regexp.MustCompile(`Stream\s+#\d+:(\d+)\[.*?\]:\s*Audio:\s*([^,]+),.*?(\d+)\s*Hz,.*?(stereo|mono|5\.1|6ch|2ch|1ch|fltp|s16),`)
	// Pattern audio alternatif si pas de "stereo/mono"
	audioPatternAlt := regexp.MustCompile(`Stream\s+#\d+:(\d+)\[.*?\]:\s*Audio:\s*([^,]+),.*?(\d+)\s*Hz`)
	subtitlePattern := regexp.MustCompile(`Stream\s+#\d+:(\d+)\[.*?\]:\s*Subtitle:\s*([^,]+)`)
	
	// Parser les pistes vidéo
	for _, match := range videoPattern.FindAllStringSubmatch(outputStr, -1) {
		if len(match) >= 5 {
			streamIdx, _ := strconv.Atoi(match[1])
			details[streamIdx] = map[string]string{
				"codec":   strings.TrimSpace(match[2]),
				"width":   match[3],
				"height":  match[4],
				"type":    "video",
			}
		}
	}
	
	// Parser les pistes audio (pattern principal)
	for _, match := range audioPattern.FindAllStringSubmatch(outputStr, -1) {
		if len(match) >= 5 {
			streamIdx, _ := strconv.Atoi(match[1])
			if details[streamIdx] == nil {
				details[streamIdx] = make(map[string]string)
			}
			details[streamIdx]["codec"] = strings.TrimSpace(match[2])
			details[streamIdx]["sample_rate"] = match[3]
			// Convertir stereo/mono en nombre de canaux
			channelStr := match[4]
			if channelStr == "stereo" || channelStr == "2ch" {
				details[streamIdx]["channels"] = "2"
			} else if channelStr == "mono" || channelStr == "1ch" {
				details[streamIdx]["channels"] = "1"
			} else if channelStr == "5.1" || channelStr == "6ch" {
				details[streamIdx]["channels"] = "6"
			}
			details[streamIdx]["type"] = "audio"
		}
	}
	
	// Parser les pistes audio (pattern alternatif si pas de channel info)
	for _, match := range audioPatternAlt.FindAllStringSubmatch(outputStr, -1) {
		if len(match) >= 4 {
			streamIdx, _ := strconv.Atoi(match[1])
			if details[streamIdx] == nil {
				details[streamIdx] = make(map[string]string)
			}
			if details[streamIdx]["codec"] == "" {
				details[streamIdx]["codec"] = strings.TrimSpace(match[2])
			}
			if details[streamIdx]["sample_rate"] == "" {
				details[streamIdx]["sample_rate"] = match[3]
			}
			if details[streamIdx]["channels"] == "" {
				// Par défaut, supposer stereo
				details[streamIdx]["channels"] = "2"
			}
			details[streamIdx]["type"] = "audio"
		}
	}
	
	// Parser les sous-titres
	for _, match := range subtitlePattern.FindAllStringSubmatch(outputStr, -1) {
		if len(match) >= 3 {
			streamIdx, _ := strconv.Atoi(match[1])
			if details[streamIdx] == nil {
				details[streamIdx] = make(map[string]string)
			}
			details[streamIdx]["codec"] = strings.TrimSpace(match[2])
			details[streamIdx]["type"] = "subtitle"
		}
	}

	return details
}

// extractAudioTracksFromFFmpeg extrait les pistes audio depuis ffmpeg
func (e *DVDExtractor) extractAudioTracksFromFFmpeg(devicePath string) []DVDTrack {
	dvdInput := "dvd://1"
	if strings.HasPrefix(devicePath, "/dev/") {
		dvdInput = fmt.Sprintf("dvd://1 -dvd_device %s", devicePath)
	}

	cmd := exec.Command(
		e.ffmpegPath,
		"-i", dvdInput,
		"-t", "0.1",
		"-f", "null",
		"-",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return []DVDTrack{}
	}

	var tracks []DVDTrack
	outputStr := string(output)
	
	// Parser les pistes audio depuis la sortie
	audioPattern := regexp.MustCompile(`Stream #(\d+):(\d+)\(([^)]+)\): Audio: ([^,]+)`)
	matches := audioPattern.FindAllStringSubmatch(outputStr, -1)
	
	for i, match := range matches {
		if len(match) >= 5 {
			language := match[3]
			codec := match[4]
			langName := e.getLanguageName(language)
			
			track := DVDTrack{
				Index:    i,
				Type:     "audio",
				Codec:    codec,
				Language: language,
				Title:    fmt.Sprintf("%s (%s)", langName, codec),
			}
			tracks = append(tracks, track)
		}
	}

	return tracks
}

// extractSubtitleTracksFromFFmpeg extrait les sous-titres depuis ffmpeg
func (e *DVDExtractor) extractSubtitleTracksFromFFmpeg(devicePath string) []DVDTrack {
	dvdInput := "dvd://1"
	if strings.HasPrefix(devicePath, "/dev/") {
		dvdInput = fmt.Sprintf("dvd://1 -dvd_device %s", devicePath)
	}

	cmd := exec.Command(
		e.ffmpegPath,
		"-i", dvdInput,
		"-t", "0.1",
		"-f", "null",
		"-",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return []DVDTrack{}
	}

	var tracks []DVDTrack
	outputStr := string(output)
	
	// Parser les sous-titres depuis la sortie
	subtitlePattern := regexp.MustCompile(`Stream #(\d+):(\d+)\(([^)]+)\): Subtitle: ([^,]+)`)
	matches := subtitlePattern.FindAllStringSubmatch(outputStr, -1)
	
	for i, match := range matches {
		if len(match) >= 5 {
			language := match[3]
			codec := match[4]
			langName := e.getLanguageName(language)
			
			track := DVDTrack{
				Index:    i,
				Type:     "subtitle",
				Codec:    codec,
				Language: language,
				Title:    fmt.Sprintf("%s (%s)", langName, codec),
			}
			tracks = append(tracks, track)
		}
	}

	return tracks
}

// detectLanguageFromCodec essaie de détecter la langue depuis le codec
func (e *DVDExtractor) detectLanguageFromCodec(codecLongName string, index int) string {
	// Patterns communs pour les langues dans les codecs
	patterns := map[string]string{
		"french":  "fra",
		"français": "fra",
		"english": "eng",
		"anglais": "eng",
		"spanish": "spa",
		"espagnol": "spa",
		"german":  "deu",
		"allemand": "deu",
		"italian": "ita",
		"italien": "ita",
	}

	codecLower := strings.ToLower(codecLongName)
	for key, lang := range patterns {
		if strings.Contains(codecLower, key) {
			return lang
		}
	}

	return ""
}

// getLanguageName convertit un code langue en nom complet
func (e *DVDExtractor) getLanguageName(code string) string {
	langMap := map[string]string{
		"fra": "Français",
		"fre": "Français",
		"fr":  "Français",
		"eng": "Anglais",
		"en":  "Anglais",
		"spa": "Espagnol",
		"es":  "Espagnol",
		"deu": "Allemand",
		"ger": "Allemand",
		"de":  "Allemand",
		"ita": "Italien",
		"it":  "Italien",
		"jpn": "Japonais",
		"ja":  "Japonais",
		"chi": "Chinois",
		"zh":  "Chinois",
		"kor": "Coréen",
		"ko":  "Coréen",
		"por": "Portugais",
		"pt":  "Portugais",
		"rus": "Russe",
		"ru":  "Russe",
		"ara": "Arabe",
		"ar":  "Arabe",
		"hin": "Hindi",
		"hi":  "Hindi",
	}

	if name, ok := langMap[strings.ToLower(code)]; ok {
		return name
	}

	return strings.ToUpper(code)
}

// ExtractDVD extrait le DVD avec les pistes sélectionnées
func (e *DVDExtractor) ExtractDVD(
	devicePath string,
	outputPath string,
	videoTrackIndex int,
	audioTrackIndices []int,
	subtitleTrackIndices []int,
	qualities []string,
) (string, error) {
	// Créer le dossier de sortie
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return "", fmt.Errorf("erreur lors de la création du dossier: %w", err)
	}

	// Construire la commande ffmpeg
	args := []string{}
	
	// Ajouter -dvd_device si c'est un device block
	if strings.HasPrefix(devicePath, "/dev/") {
		args = append(args, "-dvd_device", devicePath)
	}
	
	args = append(args, "-i", "dvd://1")
	
	// Mapper la piste vidéo (ignorer le stream dvd_nav_packet qui est index 0)
	// Les streams vidéo commencent généralement à l'index 1
	args = append(args, "-map", fmt.Sprintf("0:%d", videoTrackIndex+1))

	// Ajouter les pistes audio (les indices audio commencent après la vidéo)
	// Il faut trouver le bon index dans les streams
	for _, audioIdx := range audioTrackIndices {
		// Les streams audio sont généralement après la vidéo et les sous-titres
		// On doit mapper selon l'index réel dans les streams
		args = append(args, "-map", fmt.Sprintf("0:a:%d", audioIdx))
	}

	// Ajouter les sous-titres
	for _, subIdx := range subtitleTrackIndices {
		args = append(args, "-map", fmt.Sprintf("0:s:%d", subIdx))
	}

	// Codecs et options
	args = append(args,
		"-c:v", "libx264",
		"-preset", "medium",
		"-crf", "23",
		"-c:a", "aac",
		"-b:a", "192k",
	)
	
	// Ajouter les sous-titres si présents
	if len(subtitleTrackIndices) > 0 {
		args = append(args, "-c:s", "mov_text")
	}
	
	args = append(args,
		"-movflags", "+faststart",
		"-y",
		outputPath,
	)

	cmd := exec.Command(e.ffmpegPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("erreur lors de l'extraction: %w (output: %s)", err, string(output))
	}

	return outputPath, nil
}
