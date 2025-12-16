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
	// Pour les DVD, utiliser dvd://1 pour analyser le premier titre
	// Si devicePath est un device block, utiliser dvd://1 avec -dvd_device
	dvdInput := "dvd://1"
	
	// Si c'est un device block, spécifier le device
	if strings.HasPrefix(devicePath, "/dev/") {
		dvdInput = fmt.Sprintf("dvd://1 -dvd_device %s", devicePath)
	} else if strings.HasSuffix(devicePath, "VIDEO_TS") || strings.Contains(devicePath, "VIDEO_TS") {
		// Si c'est un chemin vers VIDEO_TS, utiliser directement
		dvdInput = devicePath
	}

	// Utiliser ffprobe avec des options spécifiques pour les DVD
	cmd := exec.Command(
		"sh", "-c",
		fmt.Sprintf("%s -v error -analyzeduration 10000000 -probesize 10000000 -print_format json -show_format -show_streams -show_chapters %s", 
			e.ffprobePath, dvdInput),
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Essayer avec le device directement
		cmd2 := exec.Command(
			e.ffprobePath,
			"-v", "error",
			"-analyzeduration", "10000000",
			"-probesize", "10000000",
			"-print_format", "json",
			"-show_format",
			"-show_streams",
			"-show_chapters",
			devicePath,
		)
		output, err = cmd2.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("erreur lors de l'analyse du DVD: %w (output: %s)", err, string(output))
		}
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

	// Si pas de titre, essayer d'extraire depuis ffmpeg output
	var ffmpegOutputStr string
	if err == nil {
		ffmpegOutputStr = string(ffmpegOutput)
		if info.Title == "" {
			// Utiliser ffmpeg pour obtenir plus d'informations
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

		// Si pas de langue dans les tags, essayer de la détecter depuis le codec_long_name
		if language == "" {
			lang := e.detectLanguageFromCodec(stream.CodecLongName, stream.Index)
			if lang != "" {
				language = lang
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

		switch stream.CodecType {
		case "video":
			track.Type = "video"
			// Ajouter des informations supplémentaires pour la vidéo
			if stream.Width > 0 && stream.Height > 0 {
				track.Title = fmt.Sprintf("%dx%d %s", stream.Width, stream.Height, stream.CodecName)
			} else {
				track.Title = fmt.Sprintf("Piste vidéo %d - %s", videoIdx+1, stream.CodecName)
			}
			info.VideoTracks = append(info.VideoTracks, track)
			videoIdx++
		case "audio":
			track.Type = "audio"
			// Ajouter des informations sur l'audio
			audioInfo := stream.CodecName
			if stream.Channels > 0 {
				audioInfo += fmt.Sprintf(" %dch", stream.Channels)
			}
			if stream.SampleRate != "" {
				audioInfo += fmt.Sprintf(" %sHz", stream.SampleRate)
			}
			if language != "" {
				// Convertir le code langue en nom complet
				langName := e.getLanguageName(language)
				audioInfo += fmt.Sprintf(" (%s)", langName)
				track.Language = language
			} else {
				audioInfo += " (Unknown)"
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
				track.Title = fmt.Sprintf("Sous-titre %d (%s)", subtitleIdx+1, stream.CodecName)
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
	dvdInput := "dvd://1"
	if strings.HasPrefix(devicePath, "/dev/") {
		dvdInput = fmt.Sprintf("dvd://1 -dvd_device %s", devicePath)
	}

	args := []string{
		"-i", dvdInput,
		"-map", fmt.Sprintf("0:v:%d", videoTrackIndex),
	}

	// Ajouter les pistes audio
	for _, audioIdx := range audioTrackIndices {
		args = append(args, "-map", fmt.Sprintf("0:a:%d", audioIdx))
	}

	// Ajouter les sous-titres
	for _, subIdx := range subtitleTrackIndices {
		args = append(args, "-map", fmt.Sprintf("0:s:%d", subIdx))
	}

	// Codecs
	args = append(args,
		"-c:v", "libx264",
		"-c:a", "aac",
		"-c:s", "mov_text",
		"-y",
		outputPath,
	)

	cmd := exec.Command(e.ffmpegPath, args...)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("erreur lors de l'extraction: %w", err)
	}

	return outputPath, nil
}
