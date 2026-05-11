package video

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type DVDTrack struct {
	Index      int    `json:"index"`
	Type       string `json:"type"` // video, audio, subtitle
	Language   string `json:"language"`
	Codec      string `json:"codec"`
	Duration   int    `json:"duration"` // en minutes
	Size       int64  `json:"size"`
	Title      string `json:"title,omitempty"`
	Channels   int    `json:"channels,omitempty"`
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

	log.Printf("[DVD Detection] Démarrage de la détection du lecteur DVD...")

	// Méthode 1: Utiliser lsblk pour détecter les lecteurs optiques (plus fiable)
	cmd := exec.Command("lsblk", "-d", "-n", "-o", "NAME,TYPE,MODEL")
	output, err := cmd.Output()
	if err == nil {
		log.Printf("[DVD Detection] lsblk output: %s", string(output))
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) >= 2 && fields[1] == "rom" {
				deviceName := fields[0]
				devicePath := fmt.Sprintf("/dev/%s", deviceName)
				log.Printf("[DVD Detection] Lecteur optique détecté par lsblk: %s", devicePath)
				// Vérifier que le device existe et est accessible
				if info, err := os.Stat(devicePath); err == nil && !info.IsDir() {
					// Vérifier les permissions (doit être un device block)
					if info.Mode()&os.ModeDevice != 0 {
						detectedDevices = append(detectedDevices, devicePath)
						log.Printf("[DVD Detection] Device validé: %s", devicePath)
					} else {
						log.Printf("[DVD Detection] Device %s n'est pas un device block", devicePath)
					}
				} else {
					log.Printf("[DVD Detection] Device %s non accessible: %v", devicePath, err)
				}
			}
		}
	} else {
		log.Printf("[DVD Detection] Erreur lors de l'exécution de lsblk: %v", err)
	}

	// Si on a trouvé des devices avec lsblk, retourner le premier
	if len(detectedDevices) > 0 {
		log.Printf("[DVD Detection] Device détecté via lsblk: %s", detectedDevices[0])
		return detectedDevices[0], nil
	}

	// Méthode 2: Essayer différents chemins de lecteurs DVD courants
	// Linux
	possibleDevices := []string{
		"/dev/sr0",
		"/dev/sr1",
		"/dev/sr2",
		"/dev/sr3",
		"/dev/sr4",
		"/dev/cdrom",
		"/dev/cdrom0",
		"/dev/cdrom1",
		"/dev/dvd",
		"/dev/dvdrw",
	}
	
	// macOS (si on est sur macOS ou dans Docker Desktop sur Mac)
	macDevices := []string{
		"/dev/disk1",
		"/dev/disk2",
		"/dev/disk3",
		"/dev/disk4",
		"/dev/rdisk1",
		"/dev/rdisk2",
		"/dev/rdisk3",
		"/dev/rdisk4",
	}
	possibleDevices = append(possibleDevices, macDevices...)
	
	// Windows/WSL (si applicable)
	windowsDevices := []string{
		"/mnt/d",
		"/mnt/e",
		"/mnt/f",
	}
	possibleDevices = append(possibleDevices, windowsDevices...)

	log.Printf("[DVD Detection] Vérification des devices standards...")
	for _, device := range possibleDevices {
		if info, err := os.Stat(device); err == nil {
			log.Printf("[DVD Detection] Device trouvé: %s", device)
			// Vérifier si c'est un device block
			if info.Mode()&os.ModeDevice != 0 {
				// Vérifier que c'est un device block (pas un dossier)
				if !info.IsDir() {
					detectedDevices = append(detectedDevices, device)
					log.Printf("[DVD Detection] Device validé: %s", device)
				} else {
					log.Printf("[DVD Detection] Device %s est un dossier, ignoré", device)
				}
			} else {
				log.Printf("[DVD Detection] Device %s n'est pas un device block", device)
			}
		} else {
			log.Printf("[DVD Detection] Device %s non accessible: %v", device, err)
		}
	}

	// Méthode 3: Vérifier les points de montage (Linux, macOS, Windows)
	// Utiliser mount ou df pour trouver tous les points de montage
	mountPoints := e.FindMountPoints()
	
	log.Printf("[DVD Detection] Points de montage trouvés: %v", mountPoints)
	
	for _, mountPoint := range mountPoints {
		if info, err := os.Stat(mountPoint); err == nil && info.IsDir() {
			log.Printf("[DVD Detection] Vérification du point de montage: %s", mountPoint)
			// Vérifier s'il contient des fichiers DVD
			files, err := os.ReadDir(mountPoint)
			if err == nil && len(files) > 0 {
				// Vérifier la présence de fichiers typiques d'un DVD
				hasVideoTS := false
				for _, file := range files {
					if file.Name() == "VIDEO_TS" || file.Name() == "video_ts" {
						hasVideoTS = true
						log.Printf("[DVD Detection] DVD trouvé au point de montage: %s", mountPoint)
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

// FindMountPoints trouve tous les points de montage possibles pour les DVD/CD
// Fonctionne sur Linux, macOS et Windows (WSL)
func (e *DVDExtractor) FindMountPoints() []string {
	var mountPoints []string

	// Points de montage Linux standards
	linuxMountPoints := []string{
		"/mnt/cdrom",
		"/mnt/cdrom0",
		"/mnt/cdrom1",
		"/media/cdrom",
		"/media/cdrom0",
		"/media/cdrom1",
		"/run/media",
		"/media",
	}

	// Points de montage macOS
	macMountPoints := []string{
		"/Volumes",
	}

	// Points de montage Windows/WSL
	windowsMountPoints := []string{
		"/mnt/d",
		"/mnt/e",
		"/mnt/f",
	}

	// Ajouter tous les points de montage standards
	mountPoints = append(mountPoints, linuxMountPoints...)
	mountPoints = append(mountPoints, macMountPoints...)
	mountPoints = append(mountPoints, windowsMountPoints...)

	// Méthode 1: Utiliser la commande 'mount' pour trouver les points de montage
	cmd := exec.Command("mount")
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, line := range lines {
			// Parser la sortie de mount pour trouver les devices optiques
			// Format typique: /dev/sr0 on /run/media/pactivisme/KILL type iso9660
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				device := fields[0]
				mountPoint := fields[2]
				
				// Vérifier si c'est un device optique (sr*, cdrom, disk* sur macOS)
				if strings.HasPrefix(device, "/dev/sr") || 
				   strings.Contains(device, "cdrom") ||
				   (strings.HasPrefix(device, "/dev/disk") && strings.Contains(line, "iso9660")) {
					// Vérifier que le point de montage n'est pas déjà dans la liste
					found := false
					for _, mp := range mountPoints {
						if mp == mountPoint {
							found = true
							break
						}
					}
					if !found {
						mountPoints = append(mountPoints, mountPoint)
						log.Printf("[DVD Detection] Point de montage trouvé via mount: %s (device: %s)", mountPoint, device)
					}
				}
			}
		}
	}

	// Méthode 2: Utiliser 'df' pour trouver les systèmes de fichiers montés
	cmd = exec.Command("df", "-T")
	output, err = cmd.Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, line := range lines {
			// Chercher les systèmes de fichiers iso9660 (DVD/CD) ou udf
			if strings.Contains(line, "iso9660") || strings.Contains(line, "udf") {
				fields := strings.Fields(line)
				if len(fields) >= 7 {
					mountPoint := fields[6]
					// Vérifier que le point de montage n'est pas déjà dans la liste
					found := false
					for _, mp := range mountPoints {
						if mp == mountPoint {
							found = true
							break
						}
					}
					if !found {
						mountPoints = append(mountPoints, mountPoint)
						log.Printf("[DVD Detection] Point de montage trouvé via df: %s", mountPoint)
					}
				}
			}
		}
	}

	// Méthode 3: Pour Linux, scanner /run/media et /media pour trouver les sous-dossiers
	// (ex: /run/media/pactivisme/KILL)
	scanDirs := []string{"/run/media", "/media", "/Volumes"}
	for _, scanDir := range scanDirs {
		if info, err := os.Stat(scanDir); err == nil && info.IsDir() {
			entries, err := os.ReadDir(scanDir)
			if err == nil {
				for _, entry := range entries {
					if entry.IsDir() {
						fullPath := filepath.Join(scanDir, entry.Name())
						// Vérifier si c'est un sous-dossier (ex: /run/media/user/DVD_NAME)
						subEntries, err := os.ReadDir(fullPath)
						if err == nil {
							for _, subEntry := range subEntries {
								if subEntry.IsDir() {
									subPath := filepath.Join(fullPath, subEntry.Name())
									// Vérifier que ce n'est pas déjà dans la liste
									found := false
									for _, mp := range mountPoints {
										if mp == subPath {
											found = true
											break
										}
									}
									if !found {
										mountPoints = append(mountPoints, subPath)
										log.Printf("[DVD Detection] Point de montage trouvé via scan: %s", subPath)
									}
								}
							}
						}
						// Ajouter aussi le dossier direct (pour macOS /Volumes/DVD_NAME)
						found := false
						for _, mp := range mountPoints {
							if mp == fullPath {
								found = true
								break
							}
						}
						if !found {
							mountPoints = append(mountPoints, fullPath)
						}
					}
				}
			}
		}
	}

	return mountPoints
}

// mountDVD tente de monter le DVD à un point de montage temporaire
// Retourne le point de montage si le montage réussit, sinon retourne une chaîne vide
func (e *DVDExtractor) mountDVD(devicePath string) string {
	// Créer un point de montage temporaire
	mountPoint := filepath.Join("/tmp", fmt.Sprintf("dvd_mount_%d", time.Now().Unix()))
	
	// Créer le répertoire de montage
	if err := os.MkdirAll(mountPoint, 0755); err != nil {
		log.Printf("[DVD Extraction] ⚠️  Impossible de créer le point de montage %s: %v", mountPoint, err)
		return ""
	}
	
	// Essayer de monter le DVD
	// Utiliser mount avec l'option -t iso9660 ou -t udf pour les DVD
	cmd := exec.Command("mount", "-t", "iso9660", devicePath, mountPoint)
	if err := cmd.Run(); err != nil {
		// Essayer avec udf
		cmd = exec.Command("mount", "-t", "udf", devicePath, mountPoint)
		if err := cmd.Run(); err != nil {
			// Essayer sans spécifier le type de système de fichiers
			cmd = exec.Command("mount", devicePath, mountPoint)
			if err := cmd.Run(); err != nil {
				log.Printf("[DVD Extraction] ⚠️  Impossible de monter le DVD %s à %s: %v", devicePath, mountPoint, err)
				// Nettoyer le répertoire de montage
				os.Remove(mountPoint)
				return ""
			}
		}
	}
	
	// Vérifier que le montage a réussi en vérifiant la présence de VIDEO_TS
	videoTSPath := filepath.Join(mountPoint, "VIDEO_TS")
	if info, err := os.Stat(videoTSPath); err == nil && info.IsDir() {
		log.Printf("[DVD Extraction] ✅ DVD monté avec succès à: %s", mountPoint)
		return mountPoint
	}
	
	// Si VIDEO_TS n'existe pas, le montage a peut-être échoué
	log.Printf("[DVD Extraction] ⚠️  DVD monté mais VIDEO_TS non trouvé à: %s", mountPoint)
	// Démontage
	exec.Command("umount", mountPoint).Run()
	os.Remove(mountPoint)
	return ""
}

// analyzeVOBFiles analyse les fichiers VOB du DVD pour identifier les titres et langues
// Les fichiers VOB sont organisés comme VTS_XX_Y_Z.VOB où :
// - XX = numéro du VTS (Video Title Set)
// - Y = numéro du titre dans ce VTS
// - Z = numéro de partie (les VOB sont limités à ~1GB)
func (e *DVDExtractor) analyzeVOBFiles(videoTSPath string) (map[int][]string, error) {
	vobFiles := make(map[int][]string)
	
	// Lire le répertoire VIDEO_TS
	files, err := os.ReadDir(videoTSPath)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la lecture de VIDEO_TS: %w", err)
	}
	
	// Pattern pour matcher les fichiers VOB: VTS_XX_Y_Z.VOB
	vobPattern := regexp.MustCompile(`VTS_(\d+)_(\d+)_(\d+)\.VOB`)
	
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		
		fileName := file.Name()
		matches := vobPattern.FindStringSubmatch(fileName)
		if len(matches) == 4 {
			vtsNum, _ := strconv.Atoi(matches[1])
			titleNum, _ := strconv.Atoi(matches[2])
			partNum, _ := strconv.Atoi(matches[3])
			
			// Créer une clé unique pour chaque titre (VTS + titre)
			titleKey := vtsNum*1000 + titleNum
			
			fullPath := filepath.Join(videoTSPath, fileName)
			vobFiles[titleKey] = append(vobFiles[titleKey], fullPath)
			
			log.Printf("[DVD VOB Analysis] 📁 Fichier VOB trouvé: %s (VTS=%d, Titre=%d, Partie=%d)", 
				fileName, vtsNum, titleNum, partNum)
		}
	}
	
	// Trier les fichiers VOB par partie pour chaque titre
	for titleKey, files := range vobFiles {
		// Trier par nom de fichier (qui contient le numéro de partie)
		// Cela garantit l'ordre correct: VTS_01_1_0.VOB, VTS_01_1_1.VOB, etc.
		sortedFiles := make([]string, len(files))
		copy(sortedFiles, files)
		// Trier par nom de fichier
		for i := 0; i < len(sortedFiles)-1; i++ {
			for j := i + 1; j < len(sortedFiles); j++ {
				if sortedFiles[i] > sortedFiles[j] {
					sortedFiles[i], sortedFiles[j] = sortedFiles[j], sortedFiles[i]
				}
			}
		}
		vobFiles[titleKey] = sortedFiles
		log.Printf("[DVD VOB Analysis] 📊 Titre %d: %d fichier(s) VOB", titleKey, len(sortedFiles))
	}
	
	return vobFiles, nil
}

// analyzeIFOFiles analyse les fichiers IFO du DVD pour extraire toutes les langues et pistes
// Les fichiers IFO contiennent les métadonnées complètes du DVD (comme HandBrake les utilise)
// VIDEO_TS.IFO = informations générales du DVD
// VTS_XX_0.IFO = informations pour chaque Video Title Set (VTS)
func (e *DVDExtractor) analyzeIFOFiles(videoTSPath string) (map[int]*IFOInfo, error) {
	ifoInfos := make(map[int]*IFOInfo)
	
	// Lire le répertoire VIDEO_TS
	files, err := os.ReadDir(videoTSPath)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la lecture de VIDEO_TS: %w", err)
	}
	
	// Pattern pour matcher les fichiers IFO: VTS_XX_0.IFO et VIDEO_TS.IFO
	ifoPattern := regexp.MustCompile(`VTS_(\d+)_0\.IFO`)
	
	// D'abord, analyser VIDEO_TS.IFO (informations générales)
	videoTSIFO := filepath.Join(videoTSPath, "VIDEO_TS.IFO")
	if info, err := os.Stat(videoTSIFO); err == nil && !info.IsDir() {
		log.Printf("[DVD IFO Analysis] 📄 Analyse de VIDEO_TS.IFO...")
		ifoInfo := e.parseIFOFile(videoTSIFO, 0)
		if ifoInfo != nil {
			ifoInfos[0] = ifoInfo
			log.Printf("[DVD IFO Analysis] ✅ VIDEO_TS.IFO analysé: %d piste(s) audio, %d sous-titre(s)", 
				len(ifoInfo.AudioTracks), len(ifoInfo.SubtitleTracks))
		}
	}
	
	// Ensuite, analyser tous les VTS_XX_0.IFO
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		
		fileName := file.Name()
		matches := ifoPattern.FindStringSubmatch(fileName)
		if len(matches) == 2 {
			vtsNum, _ := strconv.Atoi(matches[1])
			fullPath := filepath.Join(videoTSPath, fileName)
			
			log.Printf("[DVD IFO Analysis] 📄 Analyse de %s (VTS=%d)...", fileName, vtsNum)
			ifoInfo := e.parseIFOFile(fullPath, vtsNum)
			if ifoInfo != nil {
				ifoInfos[vtsNum] = ifoInfo
				log.Printf("[DVD IFO Analysis] ✅ %s analysé: %d piste(s) audio, %d sous-titre(s)", 
					fileName, len(ifoInfo.AudioTracks), len(ifoInfo.SubtitleTracks))
			}
		}
	}
	
	return ifoInfos, nil
}

// IFOInfo contient les informations extraites d'un fichier IFO
type IFOInfo struct {
	VTSNumber      int
	AudioTracks    []DVDTrack
	SubtitleTracks []DVDTrack
	VideoTracks    []DVDTrack
}

// parseIFOFile parse un fichier IFO pour extraire les langues et pistes
// Utilise lsdvd ou dvdbackup pour lire les métadonnées IFO
func (e *DVDExtractor) parseIFOFile(ifoPath string, vtsNum int) *IFOInfo {
	// Utiliser lsdvd pour analyser le fichier IFO
	// lsdvd peut analyser directement un fichier IFO si on lui donne le device
	// Mais on peut aussi utiliser dvdbackup ou dvdread pour extraire les infos
	
	// Méthode 1: Utiliser lsdvd avec le device parent (si disponible)
	// Méthode 2: Utiliser dvdbackup -I pour lire les infos IFO
	// Méthode 3: Utiliser ffprobe sur le fichier IFO
	
	// Pour l'instant, on utilise ffprobe pour analyser le fichier IFO
	// ffprobe peut lire les fichiers IFO et extraire les streams
	cmd := exec.Command(
		e.ffprobePath,
		"-v", "error",
		"-hide_banner",
		"-analyzeduration", "10000000",
		"-probesize", "10000000",
		"-print_format", "json",
		"-show_streams",
		"-show_format",
		ifoPath,
	)
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	if err != nil {
		log.Printf("[DVD IFO Analysis] ⚠️  Erreur ffprobe sur %s: %v", ifoPath, err)
		// Essayer avec lsdvd si disponible
		return e.parseIFOWithLsdvd(ifoPath, vtsNum)
	}
	
	output := stdout.Bytes()
	var probeData struct {
		Streams []struct {
			Index         int               `json:"index"`
			CodecType     string            `json:"codec_type"`
			CodecName     string            `json:"codec_name"`
			CodecLongName string            `json:"codec_long_name"`
			Tags          map[string]string `json:"tags"`
			Channels      int               `json:"channels"`
			SampleRate    string            `json:"sample_rate"`
		} `json:"streams"`
	}
	
	if err := json.Unmarshal(output, &probeData); err != nil {
		log.Printf("[DVD IFO Analysis] ⚠️  Erreur parsing JSON pour %s: %v", ifoPath, err)
		return e.parseIFOWithLsdvd(ifoPath, vtsNum)
	}
	
	info := &IFOInfo{
		VTSNumber:      vtsNum,
		AudioTracks:    []DVDTrack{},
		SubtitleTracks: []DVDTrack{},
		VideoTracks:    []DVDTrack{},
	}
	
	// Parser les streams depuis ffprobe
	for _, stream := range probeData.Streams {
		language := ""
		if stream.Tags != nil {
			if lang, ok := stream.Tags["language"]; ok && lang != "" {
				language = lang
			} else if lang, ok := stream.Tags["LANGUAGE"]; ok && lang != "" {
				language = lang
			}
		}
		
		track := DVDTrack{
			Index:    stream.Index,
			Codec:    stream.CodecName,
			Language: language,
		}
		
		switch stream.CodecType {
		case "video":
			track.Type = "video"
			track.Title = fmt.Sprintf("Vidéo - %s", stream.CodecName)
			info.VideoTracks = append(info.VideoTracks, track)
		case "audio":
			track.Type = "audio"
			track.Channels = stream.Channels
			track.SampleRate = stream.SampleRate
			langName := e.getLanguageName(language)
			if langName == "" && language != "" {
				langName = strings.ToUpper(language)
			} else if langName == "" {
				langName = "Langue inconnue"
			}
			track.Title = fmt.Sprintf("%s - %s", langName, stream.CodecName)
			if stream.Channels > 0 {
				if stream.Channels == 1 {
					track.Title += " (Mono)"
				} else if stream.Channels == 2 {
					track.Title += " (Stéréo)"
				} else if stream.Channels == 6 {
					track.Title += " (5.1)"
				} else {
					track.Title += fmt.Sprintf(" (%d canaux)", stream.Channels)
				}
			}
			info.AudioTracks = append(info.AudioTracks, track)
		case "subtitle":
			track.Type = "subtitle"
			langName := e.getLanguageName(language)
			if langName == "" && language != "" {
				langName = strings.ToUpper(language)
			} else if langName == "" {
				langName = "Langue inconnue"
			}
			track.Title = fmt.Sprintf("%s - %s", langName, stream.CodecName)
			info.SubtitleTracks = append(info.SubtitleTracks, track)
		}
	}
	
	return info
}

// parseIFOWithLsdvd utilise lsdvd pour analyser un fichier IFO
func (e *DVDExtractor) parseIFOWithLsdvd(ifoPath string, vtsNum int) *IFOInfo {
	// lsdvd ne peut pas analyser directement un fichier IFO
	// On retourne nil pour utiliser les autres méthodes
	return nil
}

// GetDVDInfo récupère les informations du DVD (pistes audio/vidéo/sous-titres)
// Utilise lsdvd pour une analyse complète de la structure, puis ffprobe/ffmpeg pour les détails
func (e *DVDExtractor) GetDVDInfo(devicePath string) (*DVDInfo, error) {
	// D'abord, essayer d'obtenir le label du volume depuis le système de fichiers
	volumeLabel := e.getVolumeLabel(devicePath)
	log.Printf("[DVD Info] 📀 Label du volume: %s", volumeLabel)

	// Essayer de monter le DVD pour accéder aux fichiers VOB
	mountPoint := ""
	if strings.HasPrefix(devicePath, "/dev/") {
		// Chercher un point de montage existant
		mountPoints := e.FindMountPoints()
		for _, mp := range mountPoints {
			videoTSPath := filepath.Join(mp, "VIDEO_TS")
			if info, err := os.Stat(videoTSPath); err == nil && info.IsDir() {
				// Vérifier quel device est monté ici
				cmd := exec.Command("findmnt", "-n", "-o", "SOURCE", mp)
				if output, err := cmd.Output(); err == nil {
					mountedDevice := strings.TrimSpace(string(output))
					if mountedDevice == devicePath || strings.Contains(mountedDevice, filepath.Base(devicePath)) {
						mountPoint = mp
						log.Printf("[DVD Info] ✅ DVD déjà monté à: %s", mountPoint)
						break
					}
				}
			}
		}
		
		// Si pas monté, essayer de monter
		if mountPoint == "" {
			log.Printf("[DVD Info] 🔄 Tentative de montage du DVD...")
			mountPoint = e.mountDVD(devicePath)
		}
	} else if strings.Contains(devicePath, "VIDEO_TS") {
		// C'est déjà un chemin VIDEO_TS
		mountPoint = filepath.Dir(filepath.Dir(devicePath))
		log.Printf("[DVD Info] ✅ Chemin VIDEO_TS fourni: %s", devicePath)
	}

	// Analyser les fichiers VOB et IFO si le DVD est monté (comme HandBrake)
	var vobFiles map[int][]string
	var ifoInfos map[int]*IFOInfo
	if mountPoint != "" {
		videoTSPath := filepath.Join(mountPoint, "VIDEO_TS")
		if info, err := os.Stat(videoTSPath); err == nil && info.IsDir() {
			log.Printf("[DVD Info] 📁 Analyse automatique des fichiers DVD dans: %s", videoTSPath)
			
			// Analyser les fichiers VOB pour identifier les titres
			vobFiles, _ = e.analyzeVOBFiles(videoTSPath)
			if len(vobFiles) > 0 {
				log.Printf("[DVD Info] ✅ %d titre(s) VOB trouvé(s)", len(vobFiles))
			}
			
			// Analyser les fichiers IFO pour extraire toutes les langues (comme HandBrake)
			log.Printf("[DVD Info] 📄 Analyse automatique des fichiers IFO pour détecter toutes les langues...")
			ifoInfos, _ = e.analyzeIFOFiles(videoTSPath)
			if len(ifoInfos) > 0 {
				totalAudio := 0
				totalSubtitles := 0
				for vts, ifoInfo := range ifoInfos {
					totalAudio += len(ifoInfo.AudioTracks)
					totalSubtitles += len(ifoInfo.SubtitleTracks)
					log.Printf("[DVD Info] 📊 VTS_%02d_0.IFO: %d piste(s) audio, %d sous-titre(s)", 
						vts, len(ifoInfo.AudioTracks), len(ifoInfo.SubtitleTracks))
				}
				log.Printf("[DVD Info] ✅ Total: %d piste(s) audio, %d sous-titre(s) détectées depuis les fichiers IFO", 
					totalAudio, totalSubtitles)
			}
		}
	}

	// Ensuite, essayer d'utiliser lsdvd pour obtenir la structure complète du DVD
	lsdvdInfo := e.analyzeDVDWithLsdvd(devicePath)

	// STRATÉGIE COMME HANDBRAKE : Utiliser les fichiers VOB si le DVD est monté
	// Sinon, utiliser directement le device avec ffprobe
	var ffprobeInput string
	var videoTSPathForProbe string
	
	// Si le DVD est monté, utiliser les fichiers VOB pour l'analyse
	if mountPoint != "" {
		videoTSPathForProbe = filepath.Join(mountPoint, "VIDEO_TS")
		// Utiliser le premier fichier VOB pour l'analyse (ou VIDEO_TS.IFO si disponible)
		ifoPath := filepath.Join(videoTSPathForProbe, "VIDEO_TS.IFO")
		if _, err := os.Stat(ifoPath); err == nil {
			// Utiliser VIDEO_TS.IFO pour l'analyse (comme HandBrake)
			ffprobeInput = ifoPath
			log.Printf("[DVD Info] ✅ Utilisation VIDEO_TS.IFO pour l'analyse: %s", ifoPath)
		} else {
			// Sinon, analyser les fichiers VOB
			vobFiles, _ := e.analyzeVOBFiles(videoTSPathForProbe)
			if len(vobFiles) > 0 {
				// Prendre le premier fichier VOB du premier titre
				for _, files := range vobFiles {
					if len(files) > 0 {
						ffprobeInput = files[0]
						log.Printf("[DVD Info] ✅ Utilisation premier fichier VOB pour l'analyse: %s", ffprobeInput)
						break
					}
				}
			}
			if ffprobeInput == "" {
				ffprobeInput = videoTSPathForProbe
				log.Printf("[DVD Info] ⚠️  Utilisation dossier VIDEO_TS pour l'analyse: %s", videoTSPathForProbe)
			}
		}
	} else if strings.HasPrefix(devicePath, "/dev/") {
		// Si pas monté, utiliser directement le device block
		ffprobeInput = devicePath
		log.Printf("[DVD Info] ⚠️  DVD non monté, utilisation directe du device: %s", devicePath)
	} else if strings.Contains(devicePath, "VIDEO_TS") {
		// Chemin vers VIDEO_TS
		ffprobeInput = devicePath
		videoTSPathForProbe = devicePath
	} else {
		ffprobeInput = devicePath
	}

	// Utiliser ffprobe avec des options spécifiques pour les DVD
	// Utiliser -v error pour réduire les warnings, et séparer stdout/stderr
	// Utiliser -loglevel error pour forcer les erreurs vers stderr uniquement
	cmd := exec.Command(
		e.ffprobePath,
		"-v", "error", // Réduire les messages d'erreur/warning
		"-loglevel", "error", // Forcer les erreurs vers stderr uniquement
		"-hide_banner", // Cacher la bannière
		"-analyzeduration", "30000000", // 30 secondes d'analyse
		"-probesize", "30000000", // 30 MB de probe
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		"-show_chapters",
		"-show_programs",
		ffprobeInput,
	)
	
	// Capturer stdout et stderr séparément pour éviter la pollution JSON
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	if err != nil {
		// Log les erreurs stderr mais ne pas les inclure dans le JSON
		log.Printf("[DVD Info] Erreurs ffprobe (stderr): %s", stderr.String())
		return nil, fmt.Errorf("erreur lors de l'analyse du DVD: %w (stderr: %s)", err, stderr.String())
	}
	
	output := stdout.Bytes()

	// Nettoyer le JSON en enlevant les messages d'erreur AC3 qui polluent la sortie
	outputStr := string(output)
	
	// Log pour debug
	if len(outputStr) > 300 {
		log.Printf("[DVD Info] Output brut (premiers 300 caractères): %s", outputStr[:300])
	} else {
		log.Printf("[DVD Info] Output brut: %s", outputStr)
	}
	
	// NOUVELLE STRATÉGIE : Gérer les deux cas (avec et sans erreur AC3)
	// Cas 1: JSON propre : { "programs": [ ] ou { "streams": [ ]
	// Cas 2: JSON pollué : { [ac3 @ ...] error ... "programs": [ ]
	
	// Trouver le premier { dans la sortie
	firstBrace := strings.Index(outputStr, "{")
	if firstBrace < 0 {
		return nil, fmt.Errorf("impossible de trouver l'accolade ouvrante '{' dans la sortie ffprobe")
	}
	log.Printf("[DVD Info] Premier { trouvé à la position %d", firstBrace)
	
	// Vérifier si le JSON est propre (commence directement par { suivi de "programs" ou "streams")
	// Chercher "programs" ou "streams" dans les 50 premiers caractères après {
	afterBrace := outputStr[firstBrace:]
	programsIdx := strings.Index(afterBrace, `"programs"`)
	streamsIdx := strings.Index(afterBrace, `"streams"`)
	
	var jsonStartPos int = -1
	
	// Si on trouve "programs" ou "streams" dans les 50 premiers caractères, le JSON est probablement propre
	if programsIdx >= 0 && programsIdx < 50 {
		// JSON propre : commencer directement après {
		jsonStartPos = firstBrace
		log.Printf("[DVD Info] ✅ JSON propre détecté: 'programs' trouvé à la position %d (après {)", programsIdx)
	} else if streamsIdx >= 0 && streamsIdx < 50 {
		// JSON propre : commencer directement après {
		jsonStartPos = firstBrace
		log.Printf("[DVD Info] ✅ JSON propre détecté: 'streams' trouvé à la position %d (après {)", streamsIdx)
	} else {
		// JSON pollué : chercher "programs" ou "streams" après les erreurs AC3
		log.Printf("[DVD Info] ⚠️ JSON pollué détecté, recherche de 'programs' ou 'streams' après les erreurs...")
		
		// Chercher le premier "programs" ou "streams" qui vient APRÈS le premier {
		var targetIdx int = -1
		if programsIdx >= 0 && (streamsIdx < 0 || programsIdx < streamsIdx) {
			targetIdx = programsIdx
		} else if streamsIdx >= 0 {
			targetIdx = streamsIdx
		}
		
		if targetIdx >= 0 {
			jsonStartPos = firstBrace + targetIdx
			log.Printf("[DVD Info] ✅ 'programs'/'streams' trouvé à la position %d (après nettoyage)", jsonStartPos)
		} else {
			// Dernier recours : chercher n'importe quel "programs" ou "streams"
			allProgramsIdx := strings.Index(outputStr, `"programs"`)
			allStreamsIdx := strings.Index(outputStr, `"streams"`)
			
			if allProgramsIdx >= 0 && (allStreamsIdx < 0 || allProgramsIdx < allStreamsIdx) {
				jsonStartPos = allProgramsIdx
				log.Printf("[DVD Info] ⚠️ Utilisation de 'programs' trouvé à la position %d (dernier recours)", jsonStartPos)
			} else if allStreamsIdx >= 0 {
				jsonStartPos = allStreamsIdx
				log.Printf("[DVD Info] ⚠️ Utilisation de 'streams' trouvé à la position %d (dernier recours)", jsonStartPos)
			} else {
				return nil, fmt.Errorf("impossible de trouver 'programs' ou 'streams' dans la sortie ffprobe")
			}
		}
	}
	
	// Reconstruire le JSON : { + "programs"/"streams" + tout ce qui suit
	// Cela supprime automatiquement tous les messages d'erreur entre { et "programs"/"streams"
	cleanedJSON := "{" + outputStr[jsonStartPos:]
	
	// Chercher la fin du JSON (dernière accolade fermante) au cas où
	jsonEnd := strings.LastIndex(cleanedJSON, "}")
	if jsonEnd > 0 && jsonEnd < len(cleanedJSON)-1 {
		// Il y a du texte après le JSON, l'enlever
		cleanedJSON = cleanedJSON[:jsonEnd+1]
		log.Printf("[DVD Info] Texte après JSON supprimé")
	}
	
	outputStr = cleanedJSON
	
	log.Printf("[DVD Info] JSON nettoyé : %d caractères supprimés entre { et début JSON", jsonStartPos-firstBrace-1)
	
	if len(outputStr) > 200 {
		log.Printf("[DVD Info] JSON nettoyé (premiers 200 caractères): %s", outputStr[:200])
	} else {
		log.Printf("[DVD Info] JSON nettoyé: %s", outputStr)
	}
	
	var probeData struct {
		Format struct {
			Duration string            `json:"duration"`
			Tags     map[string]string `json:"tags"`
		} `json:"format"`
		Streams []struct {
			Index         int               `json:"index"`
			CodecType     string            `json:"codec_type"`
			CodecName     string            `json:"codec_name"`
			CodecLongName string            `json:"codec_long_name"`
			Tags          map[string]string `json:"tags"`
			Duration      string            `json:"duration"`
			Width         int               `json:"width"`
			Height        int               `json:"height"`
			BitRate       string            `json:"bit_rate"`
			SampleRate    string            `json:"sample_rate"`
			Channels      int               `json:"channels"`
			ChannelLayout string            `json:"channel_layout"`
			Disposition   map[string]int    `json:"disposition"`
		} `json:"streams"`
		Chapters []struct {
			ID        int               `json:"id"`
			TimeBase  string            `json:"time_base"`
			Start     int               `json:"start"`
			StartTime string            `json:"start_time"`
			End       int               `json:"end"`
			EndTime   string            `json:"end_time"`
			Tags      map[string]string `json:"tags"`
		} `json:"chapters"`
	}

	// Parser le JSON nettoyé
	// Vérifier d'abord que le JSON nettoyé ne contient pas de messages d'erreur
	if strings.Contains(outputStr, "[ac3 @") {
		log.Printf("[DVD Info] ⚠️ ATTENTION: Le JSON nettoyé contient encore '[ac3 @', ce qui ne devrait pas arriver")
		// Essayer de nettoyer à nouveau
		ac3Pos := strings.Index(outputStr, "[ac3 @")
		if ac3Pos > 0 && ac3Pos < len(outputStr) {
			// Trouver le prochain "programs" après [ac3 @
			nextPrograms := strings.Index(outputStr[ac3Pos:], `"programs"`)
			if nextPrograms > 0 {
				outputStr = "{" + outputStr[ac3Pos+nextPrograms:]
				log.Printf("[DVD Info] JSON re-nettoyé après détection de [ac3 @")
			}
		}
	}
	
	// Parser le JSON nettoyé
	if err := json.Unmarshal([]byte(outputStr), &probeData); err != nil {
		// Log détaillé pour debug
		log.Printf("[DVD Info] ❌ ERREUR parsing JSON: %v", err)
		previewLen := 500
		if len(outputStr) < previewLen {
			previewLen = len(outputStr)
		}
		log.Printf("[DVD Info] JSON nettoyé (premiers %d caractères): %s", previewLen, outputStr[:previewLen])
		
		// Essayer une dernière fois de nettoyer le JSON
		// Si le JSON commence toujours par { [ac3 @, on doit trouver le vrai début
		if strings.HasPrefix(outputStr, "{ [ac3 @") || strings.Contains(outputStr[:100], "[ac3 @") {
			log.Printf("[DVD Info] ⚠️ JSON contient encore [ac3 @ au début, tentative de nettoyage final")
			// Trouver le premier "programs" ou "streams" qui vient après [ac3 @
			ac3Pos := strings.Index(outputStr, "[ac3 @")
			if ac3Pos >= 0 {
				// Chercher "programs" ou "streams" après [ac3 @
				afterAc3 := outputStr[ac3Pos:]
				programsIdx := strings.Index(afterAc3, `"programs"`)
				streamsIdx := strings.Index(afterAc3, `"streams"`)
				
				var startIdx int = -1
				if programsIdx >= 0 && (streamsIdx < 0 || programsIdx < streamsIdx) {
					startIdx = ac3Pos + programsIdx
				} else if streamsIdx >= 0 {
					startIdx = ac3Pos + streamsIdx
				}
				
				if startIdx > 0 {
					outputStr = "{" + outputStr[startIdx:]
					log.Printf("[DVD Info] JSON re-nettoyé, nouveau début à la position %d", startIdx)
					// Réessayer de parser
					if err2 := json.Unmarshal([]byte(outputStr), &probeData); err2 != nil {
						log.Printf("[DVD Info] ❌ ERREUR parsing JSON après re-nettoyage: %v", err2)
						previewLen2 := 500
						if len(outputStr) < previewLen2 {
							previewLen2 = len(outputStr)
						}
						log.Printf("[DVD Info] JSON après re-nettoyage (premiers %d caractères): %s", previewLen2, outputStr[:previewLen2])
						previewLen3 := 1000
						if len(outputStr) < previewLen3 {
							previewLen3 = len(outputStr)
						}
						return nil, fmt.Errorf("erreur lors du parsing des données: %w (output: %s)", err2, outputStr[:previewLen3])
					}
					log.Printf("[DVD Info] ✅ JSON parsé avec succès après re-nettoyage")
				} else {
					previewLen4 := 1000
					if len(outputStr) < previewLen4 {
						previewLen4 = len(outputStr)
					}
					return nil, fmt.Errorf("erreur lors du parsing des données: %w (output: %s)", err, outputStr[:previewLen4])
				}
			} else {
				previewLen5 := 1000
				if len(outputStr) < previewLen5 {
					previewLen5 = len(outputStr)
				}
				return nil, fmt.Errorf("erreur lors du parsing des données: %w (output: %s)", err, outputStr[:previewLen5])
			}
		} else {
			previewLen6 := 1000
			if len(outputStr) < previewLen6 {
				previewLen6 = len(outputStr)
			}
			return nil, fmt.Errorf("erreur lors du parsing des données: %w (output: %s)", err, outputStr[:previewLen6])
		}
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
	ffmpegOutput, _ := ffmpegCmd.CombinedOutput() // Ignorer les erreurs (AC3 errors sont normales)
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

	// Enrichir avec les informations de lsdvd et le label du volume
	// Priorité : label du volume > lsdvd > ffprobe > défaut
	if volumeLabel != "" && volumeLabel != "unknown" && volumeLabel != "Unknown" {
		info.Title = volumeLabel
	} else if lsdvdInfo != nil && lsdvdInfo.Title != "" {
		// Titre : priorité à lsdvd si ffprobe n'a rien trouvé
		if info.Title == "" || info.Title == "DVD Extraite" {
			info.Title = lsdvdInfo.Title
		}
	}

	// Si pas de titre, utiliser un titre par défaut
	if info.Title == "" {
		info.Title = "DVD Extraite"
	}

	// PRIORITÉ : Utiliser les informations des fichiers IFO (plus précises, comme HandBrake)
	// Utiliser les IFO AVANT de parser les streams ffprobe pour avoir toutes les pistes
	if len(ifoInfos) > 0 {
		// Trouver le VTS principal (généralement celui avec le plus de pistes audio)
		var mainVTS int = 0
		maxAudioTracks := 0
		for vts, ifoInfo := range ifoInfos {
			if len(ifoInfo.AudioTracks) > maxAudioTracks {
				maxAudioTracks = len(ifoInfo.AudioTracks)
				mainVTS = vts
			}
		}
		
		mainIFO := ifoInfos[mainVTS]
		if mainIFO != nil {
			log.Printf("[DVD Info] 🎯 Utilisation PRIORITAIRE des informations du VTS_%02d_0.IFO (principal)", mainVTS)
			
			// Utiliser les pistes audio des IFO EN PRIORITÉ (plus précises pour les langues)
			// Les IFO contiennent TOUTES les pistes disponibles, pas seulement celles détectées par ffprobe
			if len(mainIFO.AudioTracks) > 0 {
				info.AudioTracks = mainIFO.AudioTracks
				log.Printf("[DVD Info] ✅ %d piste(s) audio chargée(s) depuis IFO (priorité)", len(info.AudioTracks))
				// Ajouter aussi à Tracks pour la liste complète
				for _, track := range mainIFO.AudioTracks {
					info.Tracks = append(info.Tracks, track)
				}
			}
			
			// Utiliser les sous-titres des IFO EN PRIORITÉ
			if len(mainIFO.SubtitleTracks) > 0 {
				info.SubtitleTracks = mainIFO.SubtitleTracks
				log.Printf("[DVD Info] ✅ %d sous-titre(s) chargé(s) depuis IFO (priorité)", len(info.SubtitleTracks))
				// Ajouter aussi à Tracks pour la liste complète
				for _, track := range mainIFO.SubtitleTracks {
					info.Tracks = append(info.Tracks, track)
				}
			}
			
			// Utiliser aussi les pistes vidéo des IFO si disponibles
			if len(mainIFO.VideoTracks) > 0 {
				info.VideoTracks = mainIFO.VideoTracks
				log.Printf("[DVD Info] ✅ %d piste(s) vidéo chargée(s) depuis IFO (priorité)", len(info.VideoTracks))
				// Ajouter aussi à Tracks pour la liste complète
				for _, track := range mainIFO.VideoTracks {
					info.Tracks = append(info.Tracks, track)
				}
			}
		}
	}
	
	// Enrichir avec les informations de lsdvd si disponibles (fallback)
	if lsdvdInfo != nil {
		if len(lsdvdInfo.AudioTracks) > 0 && len(info.AudioTracks) == 0 {
			// Si lsdvd a trouvé des pistes audio mais pas ffprobe/IFO, utiliser celles de lsdvd
			info.AudioTracks = lsdvdInfo.AudioTracks
			log.Printf("[DVD Info] ✅ %d piste(s) audio chargée(s) depuis lsdvd (fallback)", len(info.AudioTracks))
		} else if len(lsdvdInfo.AudioTracks) > 0 && len(ifoInfos) == 0 {
			// Fusionner les informations : lsdvd pour la structure, ffprobe pour les détails techniques
			info.AudioTracks = e.mergeAudioTracks(lsdvdInfo.AudioTracks, info.AudioTracks)
			log.Printf("[DVD Info] ✅ %d piste(s) audio fusionnée(s) (lsdvd + ffprobe)", len(info.AudioTracks))
		}
		if len(lsdvdInfo.SubtitleTracks) > 0 && len(info.SubtitleTracks) == 0 {
			// Si lsdvd a trouvé des sous-titres mais pas ffprobe/IFO, utiliser ceux de lsdvd
			info.SubtitleTracks = lsdvdInfo.SubtitleTracks
			log.Printf("[DVD Info] ✅ %d sous-titre(s) chargé(s) depuis lsdvd (fallback)", len(info.SubtitleTracks))
		} else if len(lsdvdInfo.SubtitleTracks) > 0 && len(ifoInfos) == 0 {
			// Fusionner les informations
			info.SubtitleTracks = e.mergeSubtitleTracks(lsdvdInfo.SubtitleTracks, info.SubtitleTracks)
			log.Printf("[DVD Info] ✅ %d sous-titre(s) fusionné(s) (lsdvd + ffprobe)", len(info.SubtitleTracks))
		}
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

	// Vérifier si les IFO ont déjà été chargés (priorité)
	// Si les IFO ont été chargés, on ne doit PAS ajouter les pistes audio/sous-titres depuis ffprobe
	hasIFOData := len(ifoInfos) > 0
	if hasIFOData {
		log.Printf("[DVD Info] 📊 IFO chargés: %d piste(s) audio, %d sous-titre(s) depuis IFO (priorité sur ffprobe)", 
			len(info.AudioTracks), len(info.SubtitleTracks))
	}

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
			Codec:      stream.CodecName,
			Language:   language,
			Title:      title,
			Duration:   duration,
			Channels:   stream.Channels,
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
			// Toujours ajouter les pistes vidéo depuis ffprobe (les IFO n'ont souvent pas de pistes vidéo)
			info.VideoTracks = append(info.VideoTracks, track)
			info.Tracks = append(info.Tracks, track)
			videoIdx++
		case "audio":
			// Si les IFO ont déjà été chargés, NE PAS ajouter les pistes audio depuis ffprobe
			// (les IFO sont plus précis pour les langues et contiennent TOUTES les pistes)
			if !hasIFOData {
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
				info.Tracks = append(info.Tracks, track)
				audioIdx++
				if language != "" {
					languageMap[language] = e.getLanguageName(language)
				}
			} else {
				log.Printf("[DVD Info] ⏭️  Piste audio stream %d ignorée (déjà chargée depuis IFO)", stream.Index)
			}
		case "subtitle":
			// Si les IFO ont déjà été chargés, NE PAS ajouter les sous-titres depuis ffprobe
			// (les IFO sont plus précis pour les langues et contiennent TOUS les sous-titres)
			if !hasIFOData {
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
				info.Tracks = append(info.Tracks, track)
				subtitleIdx++
				if language != "" {
					languageMap[language] = e.getLanguageName(language)
				}
			} else {
				log.Printf("[DVD Info] ⏭️  Sous-titre stream %d ignoré (déjà chargé depuis IFO)", stream.Index)
			}
		}
	}

	// Si pas de pistes audio détectées, essayer plusieurs méthodes
	if len(info.AudioTracks) == 0 {
		// Méthode 1: Extraire depuis ffmpeg directement
		audioTracks := e.extractAudioTracksFromFFmpeg(devicePath)
		if len(audioTracks) > 0 {
			info.AudioTracks = audioTracks
			for _, track := range audioTracks {
				info.Tracks = append(info.Tracks, track)
			}
		} else {
			// Méthode 2: Vérifier si l'audio est intégré dans la piste vidéo
			// Certains DVDs ont l'audio dans le même stream que la vidéo
			audioTracksFromVideo := e.extractAudioFromVideoStream(devicePath, probeData.Streams)
			if len(audioTracksFromVideo) > 0 {
				info.AudioTracks = audioTracksFromVideo
				for _, track := range audioTracksFromVideo {
					info.Tracks = append(info.Tracks, track)
				}
			} else {
				// Méthode 3: Créer une piste audio par défaut si la vidéo a des canaux audio
				// Vérifier si la piste vidéo a des informations audio
				for _, stream := range probeData.Streams {
					if stream.CodecType == "video" && (stream.Channels > 0 || stream.SampleRate != "") {
						// La piste vidéo contient probablement de l'audio intégré
						track := DVDTrack{
							Index:      len(info.AudioTracks),
							Type:       "audio",
							Codec:      "ac3", // Par défaut pour DVD
							Language:   "fra", // Français présumé
							Title:      "Audio intégré (Français présumé - ac3)",
							Channels:   2, // Stéréo par défaut
							SampleRate: "48000",
						}
						info.AudioTracks = append(info.AudioTracks, track)
						info.Tracks = append(info.Tracks, track)
						break
					}
				}
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

	// Améliorer le titre avec le label du volume si disponible
	if info.Title == "" || info.Title == "DVD Extraite" {
		if volumeLabel != "" && volumeLabel != "unknown" && volumeLabel != "Unknown" {
			info.Title = volumeLabel
			log.Printf("[DVD Info] 📝 Titre mis à jour depuis le label du volume: %s", volumeLabel)
		}
	}

	// Si on a analysé les fichiers VOB, ajouter des informations supplémentaires
	if len(vobFiles) > 0 {
		// Trouver le titre avec le plus de fichiers VOB (généralement le film principal)
		var bestTitleKey int
		maxFiles := 0
		for titleKey, files := range vobFiles {
			if len(files) > maxFiles {
				maxFiles = len(files)
				bestTitleKey = titleKey
			}
		}
		if maxFiles > 0 {
			log.Printf("[DVD Info] 🎬 Titre principal identifié: VTS %d avec %d fichier(s) VOB", 
				bestTitleKey/1000, maxFiles)
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

	// Pattern amélioré pour Stream #0:X[ID](lang): Type: ...
	// Exemples:
	// Stream #0:3[0x80](fra): Audio: ac3
	// Stream #0:2[0x20](eng): Subtitle: dvd_subtitle
	// Pattern principal avec langue entre parenthèses
	pattern := regexp.MustCompile(`Stream\s+#\d+:(\d+)\[.*?\]\(([a-z]{2,3})\):`)
	matches := pattern.FindAllStringSubmatch(outputStr, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			streamIdx, err := strconv.Atoi(match[1])
			if err == nil {
				lang := strings.ToLower(strings.TrimSpace(match[2]))
				if lang != "" && len(lang) >= 2 {
					languages[streamIdx] = lang
				}
			}
		}
	}

	// Pattern alternatif : chercher "language:" dans les métadonnées
	// Exemple: Metadata: language: fra
	langMetadataPattern := regexp.MustCompile(`Stream\s+#\d+:(\d+).*?language:\s*([a-z]{2,3})`)
	langMatches := langMetadataPattern.FindAllStringSubmatch(outputStr, -1)

	for _, match := range langMatches {
		if len(match) >= 3 {
			streamIdx, err := strconv.Atoi(match[1])
			if err == nil {
				lang := strings.ToLower(strings.TrimSpace(match[2]))
				if lang != "" && len(lang) >= 2 {
					// Ne pas écraser si déjà trouvé
					if _, exists := languages[streamIdx]; !exists {
						languages[streamIdx] = lang
					}
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
				"codec":  strings.TrimSpace(match[2]),
				"width":  match[3],
				"height": match[4],
				"type":   "video",
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

// extractAudioFromVideoStream vérifie si l'audio est intégré dans les streams vidéo
func (e *DVDExtractor) extractAudioFromVideoStream(devicePath string, streams []struct {
	Index         int               `json:"index"`
	CodecType     string            `json:"codec_type"`
	CodecName     string            `json:"codec_name"`
	CodecLongName string            `json:"codec_long_name"`
	Tags          map[string]string `json:"tags"`
	Duration      string            `json:"duration"`
	Width         int               `json:"width"`
	Height        int               `json:"height"`
	BitRate       string            `json:"bit_rate"`
	SampleRate    string            `json:"sample_rate"`
	Channels      int               `json:"channels"`
	ChannelLayout string            `json:"channel_layout"`
	Disposition   map[string]int    `json:"disposition"`
}) []DVDTrack {
	var tracks []DVDTrack

	// Analyser la sortie ffmpeg pour trouver tous les streams audio, même ceux qui ne sont pas détectés par ffprobe
	var ffmpegInput string
	if strings.HasPrefix(devicePath, "/dev/") {
		ffmpegInput = devicePath
	} else {
		ffmpegInput = devicePath
	}

	// Utiliser ffmpeg avec une analyse plus approfondie
	cmd := exec.Command(
		e.ffmpegPath,
		"-analyzeduration", "10000000", // 10 secondes
		"-probesize", "10000000", // 10 MB
		"-i", ffmpegInput,
		"-t", "1", // Analyser 1 seconde
		"-f", "null",
		"-",
	)
	output, _ := cmd.CombinedOutput() // Ignorer les erreurs
	outputStr := string(output)

	// Parser tous les streams audio depuis la sortie ffmpeg
	// Pattern pour capturer tous les streams audio, même ceux intégrés
	// Exemple: Stream #0:1[0x1e0]: Video: mpeg2video
	//          Stream #0:2[0x80](fra): Audio: ac3
	audioPattern := regexp.MustCompile(`Stream\s+#\d+:(\d+)\[.*?\]\(?([a-z]{2,3})?\)?:\s*Audio:\s*([^,]+)`)
	matches := audioPattern.FindAllStringSubmatch(outputStr, -1)

	for i, match := range matches {
		if len(match) >= 4 {
			// streamIdx n'est pas utilisé, on utilise i comme index
			language := strings.ToLower(strings.TrimSpace(match[2]))
			codec := strings.TrimSpace(match[3])

			if language == "" {
				language = "fra" // Français présumé
			}

			langName := e.getLanguageName(language)
			if langName == "" {
				langName = "Français présumé"
			}

			track := DVDTrack{
				Index:    i,
				Type:     "audio",
				Codec:    codec,
				Language: language,
				Title:    fmt.Sprintf("%s - %s", langName, codec),
				Channels: 2, // Par défaut
			}
			tracks = append(tracks, track)
		}
	}

	return tracks
}

// extractAudioTracksFromFFmpeg extrait les pistes audio depuis ffmpeg
func (e *DVDExtractor) extractAudioTracksFromFFmpeg(devicePath string) []DVDTrack {
	// Utiliser directement le device path au lieu de dvd://1
	var ffmpegInput string
	if strings.HasPrefix(devicePath, "/dev/") {
		ffmpegInput = devicePath
	} else if strings.Contains(devicePath, "VIDEO_TS") {
		ffmpegInput = devicePath
	} else {
		ffmpegInput = devicePath
	}

	cmd := exec.Command(
		e.ffmpegPath,
		"-i", ffmpegInput,
		"-t", "0.5", // Augmenter pour plus d'infos
		"-f", "null",
		"-",
	)
	output, _ := cmd.CombinedOutput() // Ignorer les erreurs (AC3 errors sont normales)
	outputStr := string(output)

	// Même si ffmpeg retourne une erreur (AC3 errors sont normales), parser la sortie
	var tracks []DVDTrack

	// Pattern amélioré pour capturer les pistes audio avec langue
	// Exemples:
	// Stream #0:3[0x80](fra): Audio: ac3, 48000 Hz, stereo, fltp, 192 kb/s
	// Stream #0:4[0x81](eng): Audio: ac3, 48000 Hz, 5.1(side), fltp, 448 kb/s
	audioPattern := regexp.MustCompile(`Stream\s+#\d+:(\d+)\[.*?\]\(([a-z]{3})\):\s*Audio:\s*([^,]+),.*?(\d+)\s*Hz,.*?(stereo|mono|5\.1|6ch|2ch|1ch|fltp|s16|surround|quad|hexagonal|octagonal)?`)
	matches := audioPattern.FindAllStringSubmatch(outputStr, -1)

	// Si pas de match avec langue, essayer sans langue
	if len(matches) == 0 {
		audioPatternNoLang := regexp.MustCompile(`Stream\s+#\d+:(\d+)\[.*?\]:\s*Audio:\s*([^,]+),.*?(\d+)\s*Hz`)
		matches = audioPatternNoLang.FindAllStringSubmatch(outputStr, -1)
	}

	for i, match := range matches {
		audioIndex := i // Index pour la piste audio (0, 1, 2...)
		language := ""
		codec := ""
		sampleRate := ""
		channels := 0

		if len(match) >= 4 {
			// Avec langue
			if len(match) >= 6 {
				language = match[2]
				codec = strings.TrimSpace(match[3])
				sampleRate = match[4]
				channelStr := match[5]

				// Convertir channel string en nombre
				if channelStr == "stereo" || channelStr == "2ch" {
					channels = 2
				} else if channelStr == "mono" || channelStr == "1ch" {
					channels = 1
				} else if channelStr == "5.1" || channelStr == "6ch" {
					channels = 6
				} else {
					channels = 2 // Par défaut stereo
				}
			}

			langName := e.getLanguageName(language)
			if langName == "" && language != "" {
				langName = strings.ToUpper(language)
			} else if langName == "" {
				langName = "Français présumé"
				language = "fra"
			}

			// Créer un titre descriptif
			title := fmt.Sprintf("%s - %s", langName, codec)
			if channels > 0 {
				if channels == 1 {
					title += " (Mono)"
				} else if channels == 2 {
					title += " (Stéréo)"
				} else if channels == 6 {
					title += " (5.1)"
				} else {
					title += fmt.Sprintf(" (%d canaux)", channels)
				}
			}
			if sampleRate != "" {
				title += fmt.Sprintf(" %sHz", sampleRate)
			}

			track := DVDTrack{
				Index:      audioIndex, // Utiliser audioIndex
				Type:       "audio",
				Codec:      codec,
				Language:   language,
				Title:      title,
				Channels:   channels,
				SampleRate: sampleRate,
			}
			tracks = append(tracks, track)
		}
	}

	return tracks
}

// extractSubtitleTracksFromFFmpeg extrait les sous-titres depuis ffmpeg
func (e *DVDExtractor) extractSubtitleTracksFromFFmpeg(devicePath string) []DVDTrack {
	// Utiliser directement le device path au lieu de dvd://1
	var ffmpegInput string
	if strings.HasPrefix(devicePath, "/dev/") {
		ffmpegInput = devicePath
	} else {
		ffmpegInput = devicePath
	}
	
	cmd := exec.Command(
		e.ffmpegPath,
		"-i", ffmpegInput,
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
		"french":   "fra",
		"français": "fra",
		"english":  "eng",
		"anglais":  "eng",
		"spanish":  "spa",
		"espagnol": "spa",
		"german":   "deu",
		"allemand": "deu",
		"italian":  "ita",
		"italien":  "ita",
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

// DVDTitleInfo représente les informations d'un titre DVD
type DVDTitleInfo struct {
	TitleNumber    int
	Length         float64 // En secondes
	AudioTracks    []DVDTrack
	SubtitleTracks []DVDTrack
}

// analyzeDVDWithLsdvd utilise lsdvd pour analyser la structure complète du DVD
// Utilise la même logique que HandBrake : trouve le titre le plus long et extrait ses pistes
func (e *DVDExtractor) analyzeDVDWithLsdvd(devicePath string) *DVDInfo {
	// lsdvd peut analyser directement un device block
	// Utiliser -x pour obtenir plus d'informations XML
	cmd := exec.Command("lsdvd", "-x", devicePath)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// Si lsdvd échoue, essayer sans -x
	if err != nil || outputStr == "" {
		cmd = exec.Command("lsdvd", devicePath)
		output, err = cmd.CombinedOutput()
		if err != nil {
			// lsdvd peut échouer, ce n'est pas grave, on utilisera ffprobe
			return nil
		}
		outputStr = string(output)
		if outputStr == "" {
			return nil
		}
	}

	// Analyser tous les titres pour trouver le plus long (comme HandBrake)
	titles := e.parseDVDTitles(outputStr)
	if len(titles) == 0 {
		// Pas de titres trouvés, utiliser l'ancienne méthode
		return e.parseDVDInfoLegacy(outputStr)
	}

	// Trouver le titre le plus long (généralement le film principal)
	longestTitle := titles[0]
	for _, title := range titles {
		if title.Length > longestTitle.Length {
			longestTitle = title
		}
	}

	// Retourner les informations du titre le plus long
	info := &DVDInfo{
		Title:          "",
		AudioTracks:    longestTitle.AudioTracks,
		SubtitleTracks: longestTitle.SubtitleTracks,
		VideoTracks:    []DVDTrack{},
	}

	return info
}

// parseDVDTitles parse tous les titres du DVD depuis la sortie de lsdvd
func (e *DVDExtractor) parseDVDTitles(outputStr string) []DVDTitleInfo {
	var titles []DVDTitleInfo
	lines := strings.Split(outputStr, "\n")

	var currentTitleInfo *DVDTitleInfo

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Détecter un nouveau titre
		// Format: Title: 01, Length: 01:23:45.000 (chapters: 20, cells: 20)
		titleMatch := regexp.MustCompile(`Title:\s*(\d+),.*?Length:\s*(\d+):(\d+):(\d+)\.(\d+)`)
		if matches := titleMatch.FindStringSubmatch(line); len(matches) >= 6 {
			// Sauvegarder le titre précédent s'il existe
			if currentTitleInfo != nil {
				titles = append(titles, *currentTitleInfo)
			}

			// Nouveau titre
			titleNum, _ := strconv.Atoi(matches[1])
			hours, _ := strconv.Atoi(matches[2])
			minutes, _ := strconv.Atoi(matches[3])
			seconds, _ := strconv.Atoi(matches[4])
			ms, _ := strconv.Atoi(matches[5])

			length := float64(hours*3600+minutes*60+seconds) + float64(ms)/1000.0

			currentTitleInfo = &DVDTitleInfo{
				TitleNumber:    titleNum,
				Length:         length,
				AudioTracks:    []DVDTrack{},
				SubtitleTracks: []DVDTrack{},
			}
			continue
		}

		// Si on est dans un titre, parser les pistes audio
		if currentTitleInfo != nil {
			// Piste audio : Audio: 0, Language: fra, Format: ac3, Frequency: 48000, Channels: 2
			if strings.HasPrefix(line, "Audio:") || strings.Contains(line, "Audio:") {
				audioMatch := regexp.MustCompile(`Audio:\s*(\d+).*?Language:\s*([a-z]{2,3})`)
				if matches := audioMatch.FindStringSubmatch(line); len(matches) >= 3 {
					index, _ := strconv.Atoi(matches[1])
					language := strings.ToLower(strings.TrimSpace(matches[2]))

					// Extraire le format/codec
					codec := "ac3" // Par défaut
					if formatMatch := regexp.MustCompile(`Format:\s*([^,]+)`).FindStringSubmatch(line); len(formatMatch) >= 2 {
						codec = strings.TrimSpace(formatMatch[1])
					}

					// Extraire la fréquence
					sampleRate := "48000" // Par défaut
					if freqMatch := regexp.MustCompile(`Frequency:\s*(\d+)`).FindStringSubmatch(line); len(freqMatch) >= 2 {
						sampleRate = freqMatch[1]
					}

					// Extraire les canaux
					channels := 2 // Par défaut stéréo
					if chMatch := regexp.MustCompile(`Channels:\s*(\d+)`).FindStringSubmatch(line); len(chMatch) >= 2 {
						if ch, err := strconv.Atoi(chMatch[1]); err == nil {
							channels = ch
						}
					}

					langName := e.getLanguageName(language)
					if langName == "" {
						langName = strings.ToUpper(language)
					}

					title := fmt.Sprintf("%s - %s", langName, codec)
					if channels > 0 {
						if channels == 1 {
							title += " (Mono)"
						} else if channels == 2 {
							title += " (Stéréo)"
						} else if channels == 6 {
							title += " (5.1)"
						} else {
							title += fmt.Sprintf(" (%d canaux)", channels)
						}
					}
					if sampleRate != "" {
						title += fmt.Sprintf(" %sHz", sampleRate)
					}

					track := DVDTrack{
						Index:      index,
						Type:       "audio",
						Codec:      codec,
						Language:   language,
						Title:      title,
						Channels:   channels,
						SampleRate: sampleRate,
					}
					currentTitleInfo.AudioTracks = append(currentTitleInfo.AudioTracks, track)
				}
			}

			// Piste sous-titre : Subtitle: 0, Language: fra
			if strings.HasPrefix(line, "Subtitle:") || strings.Contains(line, "Subtitle:") {
				subtitleMatch := regexp.MustCompile(`Subtitle:\s*(\d+).*?Language:\s*([a-z]{2,3})`)
				if matches := subtitleMatch.FindStringSubmatch(line); len(matches) >= 3 {
					index, _ := strconv.Atoi(matches[1])
					language := strings.ToLower(strings.TrimSpace(matches[2]))

					langName := e.getLanguageName(language)
					if langName == "" {
						langName = strings.ToUpper(language)
					}

					track := DVDTrack{
						Index:    index,
						Type:     "subtitle",
						Codec:    "dvd_subtitle",
						Language: language,
						Title:    fmt.Sprintf("%s - dvd_subtitle", langName),
					}
					currentTitleInfo.SubtitleTracks = append(currentTitleInfo.SubtitleTracks, track)
				}
			}
		}
	}

	// Ajouter le dernier titre
	if currentTitleInfo != nil {
		titles = append(titles, *currentTitleInfo)
	}

	return titles
}

// parseDVDInfoLegacy parse les informations DVD avec l'ancienne méthode (sans titres)
func (e *DVDExtractor) parseDVDInfoLegacy(outputStr string) *DVDInfo {

	info := &DVDInfo{
		AudioTracks:    []DVDTrack{},
		SubtitleTracks: []DVDTrack{},
		VideoTracks:    []DVDTrack{},
	}

	// Parser la sortie de lsdvd
	lines := strings.Split(outputStr, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Extraire le titre du DVD - plusieurs formats possibles
		if strings.HasPrefix(line, "Disc Title:") {
			title := strings.TrimPrefix(line, "Disc Title:")
			title = strings.TrimSpace(title)
			if title != "" && title != "unknown" && title != "Unknown" {
				info.Title = title
			}
			continue
		}
		// Format alternatif
		if strings.HasPrefix(line, "Title:") && !strings.Contains(line, "Length:") {
			titleMatch := regexp.MustCompile(`Title:\s*(.+)`)
			if matches := titleMatch.FindStringSubmatch(line); len(matches) >= 2 {
				title := strings.TrimSpace(matches[1])
				if title != "" && title != "unknown" && title != "Unknown" {
					info.Title = title
				}
			}
			continue
		}

		// Parser les pistes audio - plusieurs formats possibles
		// Format 1: Audio: 0, Language: fra, Format: ac3, Frequency: 48000, Quantization: 16bit, Channels: 2
		// Format 2: Audio: 0, Language: fra, Format: ac3, Frequency: 48000, Channels: 2
		// Format 3: Audio: 0, Language: fra, Format: ac3
		if strings.HasPrefix(line, "Audio:") || strings.Contains(line, "Audio:") {
			// Pattern plus flexible
			audioMatch := regexp.MustCompile(`Audio:\s*(\d+).*?Language:\s*([a-z]{2,3})`)
			if matches := audioMatch.FindStringSubmatch(line); len(matches) >= 3 {
				index, _ := strconv.Atoi(matches[1])
				language := strings.ToLower(matches[2])

				// Extraire le format/codec
				codec := "ac3" // Par défaut
				if formatMatch := regexp.MustCompile(`Format:\s*([^,]+)`).FindStringSubmatch(line); len(formatMatch) >= 2 {
					codec = strings.TrimSpace(formatMatch[1])
				}

				// Extraire la fréquence
				sampleRate := "48000" // Par défaut
				if freqMatch := regexp.MustCompile(`Frequency:\s*(\d+)`).FindStringSubmatch(line); len(freqMatch) >= 2 {
					sampleRate = freqMatch[1]
				}

				// Extraire les canaux
				channels := 2 // Par défaut stéréo
				if chMatch := regexp.MustCompile(`Channels:\s*(\d+)`).FindStringSubmatch(line); len(chMatch) >= 2 {
					if ch, err := strconv.Atoi(chMatch[1]); err == nil {
						channels = ch
					}
				}

				langName := e.getLanguageName(language)
				if langName == "" {
					langName = strings.ToUpper(language)
				}

				title := fmt.Sprintf("%s - %s", langName, codec)
				if channels > 0 {
					if channels == 1 {
						title += " (Mono)"
					} else if channels == 2 {
						title += " (Stéréo)"
					} else if channels == 6 {
						title += " (5.1)"
					} else {
						title += fmt.Sprintf(" (%d canaux)", channels)
					}
				}
				if sampleRate != "" {
					title += fmt.Sprintf(" %sHz", sampleRate)
				}

				track := DVDTrack{
					Index:      index,
					Type:       "audio",
					Codec:      codec,
					Language:   language,
					Title:      title,
					Channels:   channels,
					SampleRate: sampleRate,
				}
				info.AudioTracks = append(info.AudioTracks, track)
			} else {
				// Pattern alternatif sans langue explicite
				audioMatchAlt := regexp.MustCompile(`Audio:\s*(\d+)`)
				if matches := audioMatchAlt.FindStringSubmatch(line); len(matches) >= 2 {
					index, _ := strconv.Atoi(matches[1])
					codec := "ac3"
					if formatMatch := regexp.MustCompile(`Format:\s*([^,]+)`).FindStringSubmatch(line); len(formatMatch) >= 2 {
						codec = strings.TrimSpace(formatMatch[1])
					}

					// Utiliser "Inconnu" si pas de langue
					title := fmt.Sprintf("Piste %d - %s", index+1, codec)
					track := DVDTrack{
						Index:    index,
						Type:     "audio",
						Codec:    codec,
						Language: "",
						Title:    title,
						Channels: 2,
					}
					info.AudioTracks = append(info.AudioTracks, track)
				}
			}
			continue
		}

		// Parser les sous-titres
		if strings.HasPrefix(line, "Subtitle:") {
			subtitleMatch := regexp.MustCompile(`Subtitle:\s*(\d+).*?Language:\s*([a-z]{2,3})`)
			if matches := subtitleMatch.FindStringSubmatch(line); len(matches) >= 3 {
				index, _ := strconv.Atoi(matches[1])
				language := strings.ToLower(matches[2])

				langName := e.getLanguageName(language)
				if langName == "" {
					langName = strings.ToUpper(language)
				}

				track := DVDTrack{
					Index:    index,
					Type:     "subtitle",
					Language: language,
					Codec:    "dvd_subtitle",
					Title:    fmt.Sprintf("%s - dvd_subtitle", langName),
				}
				info.SubtitleTracks = append(info.SubtitleTracks, track)
			}
			continue
		}
	}

	return info
}

// mergeAudioTracks fusionne les pistes audio de lsdvd et ffprobe
func (e *DVDExtractor) mergeAudioTracks(lsdvdTracks, ffprobeTracks []DVDTrack) []DVDTrack {
	ffprobeMap := make(map[int]DVDTrack)
	for _, track := range ffprobeTracks {
		ffprobeMap[track.Index] = track
	}

	merged := make([]DVDTrack, 0)
	for _, lsdvdTrack := range lsdvdTracks {
		if ffprobeTrack, exists := ffprobeMap[lsdvdTrack.Index]; exists {
			mergedTrack := lsdvdTrack
			if ffprobeTrack.SampleRate != "" && mergedTrack.SampleRate == "" {
				mergedTrack.SampleRate = ffprobeTrack.SampleRate
			}
			if ffprobeTrack.Channels > 0 && mergedTrack.Channels == 0 {
				mergedTrack.Channels = ffprobeTrack.Channels
			}
			if ffprobeTrack.Codec != "" && mergedTrack.Codec == "" {
				mergedTrack.Codec = ffprobeTrack.Codec
			}
			langName := e.getLanguageName(mergedTrack.Language)
			if langName == "" {
				langName = strings.ToUpper(mergedTrack.Language)
			}
			title := fmt.Sprintf("%s - %s", langName, mergedTrack.Codec)
			if mergedTrack.Channels > 0 {
				if mergedTrack.Channels == 1 {
					title += " (Mono)"
				} else if mergedTrack.Channels == 2 {
					title += " (Stéréo)"
				} else if mergedTrack.Channels == 6 {
					title += " (5.1)"
				} else {
					title += fmt.Sprintf(" (%d canaux)", mergedTrack.Channels)
				}
			}
			if mergedTrack.SampleRate != "" {
				title += fmt.Sprintf(" %sHz", mergedTrack.SampleRate)
			}
			mergedTrack.Title = title
			merged = append(merged, mergedTrack)
		} else {
			merged = append(merged, lsdvdTrack)
		}
	}

	for _, ffprobeTrack := range ffprobeTracks {
		found := false
		for _, mergedTrack := range merged {
			if mergedTrack.Index == ffprobeTrack.Index {
				found = true
				break
			}
		}
		if !found {
			merged = append(merged, ffprobeTrack)
		}
	}

	return merged
}

// mergeSubtitleTracks fusionne les sous-titres de lsdvd et ffprobe
func (e *DVDExtractor) mergeSubtitleTracks(lsdvdTracks, ffprobeTracks []DVDTrack) []DVDTrack {
	ffprobeMap := make(map[int]DVDTrack)
	for _, track := range ffprobeTracks {
		ffprobeMap[track.Index] = track
	}

	merged := make([]DVDTrack, 0)
	for _, lsdvdTrack := range lsdvdTracks {
		if ffprobeTrack, exists := ffprobeMap[lsdvdTrack.Index]; exists {
			mergedTrack := lsdvdTrack
			if ffprobeTrack.Codec != "" {
				mergedTrack.Codec = ffprobeTrack.Codec
			}
			langName := e.getLanguageName(mergedTrack.Language)
			if langName == "" {
				langName = strings.ToUpper(mergedTrack.Language)
			}
			mergedTrack.Title = fmt.Sprintf("%s - %s", langName, mergedTrack.Codec)
			merged = append(merged, mergedTrack)
		} else {
			merged = append(merged, lsdvdTrack)
		}
	}

	for _, ffprobeTrack := range ffprobeTracks {
		found := false
		for _, mergedTrack := range merged {
			if mergedTrack.Index == ffprobeTrack.Index {
				found = true
				break
			}
		}
		if !found {
			merged = append(merged, ffprobeTrack)
		}
	}

	return merged
}

// getVolumeLabel récupère le label du volume depuis le système de fichiers
func (e *DVDExtractor) getVolumeLabel(devicePath string) string {
	// Méthode 1: Utiliser blkid pour obtenir le label
	cmd := exec.Command("blkid", "-s", "LABEL", "-o", "value", devicePath)
	output, err := cmd.Output()
	if err == nil {
		label := strings.TrimSpace(string(output))
		if label != "" && label != "unknown" && label != "Unknown" {
			return label
		}
	}

	// Méthode 2: Utiliser lsblk pour obtenir le label
	cmd = exec.Command("lsblk", "-n", "-o", "LABEL", devicePath)
	output, err = cmd.Output()
	if err == nil {
		label := strings.TrimSpace(string(output))
		if label != "" && label != "unknown" && label != "Unknown" {
			return label
		}
	}

	// Méthode 3: Essayer de monter le DVD et lire le label
	// Chercher les points de montage possibles
	possibleMountPoints := []string{
		"/mnt/cdrom",
		"/media/cdrom",
		"/media/cdrom0",
		"/media/cdrom1",
		"/run/media",
	}

	for _, mountPoint := range possibleMountPoints {
		if info, err := os.Stat(mountPoint); err == nil && info.IsDir() {
			// Vérifier si c'est un point de montage actif
			cmd = exec.Command("findmnt", "-n", "-o", "SOURCE", mountPoint)
			output, err = cmd.Output()
			if err == nil {
				source := strings.TrimSpace(string(output))
				if strings.Contains(source, devicePath) || strings.Contains(devicePath, source) {
					// C'est bien monté, essayer de lire le label depuis le système de fichiers
					// Utiliser udevadm ou directement depuis /proc/mounts
					cmd = exec.Command("udevadm", "info", "-q", "name", "-n", devicePath)
					output, err = cmd.Output()
					if err == nil {
						// Essayer de lire le label depuis le système de fichiers monté
						labelFile := filepath.Join(mountPoint, ".discinfo")
						if data, err := os.ReadFile(labelFile); err == nil {
							lines := strings.Split(string(data), "\n")
							if len(lines) > 0 && lines[0] != "" {
								return strings.TrimSpace(lines[0])
							}
						}
					}
				}
			}
		}
	}

	// Méthode 4: Essayer de lire depuis /dev/disk/by-label
	cmd = exec.Command("ls", "-la", "/dev/disk/by-label/")
	output, err = cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, devicePath) {
				// Extraire le label depuis le lien symbolique
				parts := strings.Fields(line)
				if len(parts) >= 9 {
					// Le label est le nom du fichier
					label := filepath.Base(parts[8])
					if label != "" && label != "unknown" && label != "Unknown" {
						return label
					}
				}
			}
		}
	}

	return ""
}

// findLongestDVDTitle trouve le titre le plus long dans le DVD (comme HandBrake)
func (e *DVDExtractor) findLongestDVDTitle(devicePath string) (int, error) {
	// Utiliser lsdvd pour lister tous les titres
	cmd := exec.Command("lsdvd", devicePath)
	output, err := cmd.Output()
	if err != nil {
		// Si lsdvd échoue, retourner le titre 1 par défaut
		return 1, nil
	}

	outputStr := string(output)
	longestDuration := 0
	longestTitle := 1

	// Parser la sortie de lsdvd pour trouver le titre le plus long
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		// Chercher les lignes "Title: X, Length: HH:MM:SS"
		if strings.Contains(line, "Title:") && strings.Contains(line, "Length:") {
			// Extraire le numéro de titre
			titleMatch := regexp.MustCompile(`Title:\s*(\d+)`)
			titleMatches := titleMatch.FindStringSubmatch(line)
			if len(titleMatches) < 2 {
				continue
			}
			titleNum, err := strconv.Atoi(titleMatches[1])
			if err != nil {
				continue
			}

			// Extraire la durée
			durationMatch := regexp.MustCompile(`Length:\s*(\d+):(\d+):(\d+)`)
			durationMatches := durationMatch.FindStringSubmatch(line)
			if len(durationMatches) >= 4 {
				hours, _ := strconv.Atoi(durationMatches[1])
				minutes, _ := strconv.Atoi(durationMatches[2])
				seconds, _ := strconv.Atoi(durationMatches[3])
				totalSeconds := hours*3600 + minutes*60 + seconds

				if totalSeconds > longestDuration {
					longestDuration = totalSeconds
					longestTitle = titleNum
				}
			}
		}
	}

	return longestTitle, nil
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

	// Trouver le titre le plus long (comme HandBrake)
	log.Printf("[DVD Extraction] Recherche du titre le plus long...")
	longestTitle, err := e.findLongestDVDTitle(devicePath)
	if err != nil {
		log.Printf("[DVD Extraction] Erreur lors de la recherche du titre le plus long: %v, utilisation du titre 1 par défaut", err)
		// En cas d'erreur, utiliser le titre 1 par défaut
		longestTitle = 1
	} else {
		log.Printf("[DVD Extraction] Titre le plus long trouvé: %d", longestTitle)
	}

	// Construire la commande ffmpeg
	log.Printf("[DVD Extraction] 🔧 DÉBUT construction args - devicePath: %s, longestTitle: %d", devicePath, longestTitle)
	args := []string{}

	// Ajouter des options pour améliorer la lecture du DVD
	// Ces options permettent à ffmpeg de mieux analyser et lire le DVD
	args = append(args,
		"-analyzeduration", "10000000", // 10 secondes d'analyse
		"-probesize", "10000000",       // 10 MB de probe
		"-thread_queue_size", "512",    // Taille de la queue pour la lecture
	)

	// STRATÉGIE COMME HANDBRAKE :
	// 1. Essayer de monter le DVD et utiliser les fichiers VIDEO_TS (plus fiable)
	// 2. Sinon, utiliser dvdread://N avec le titre spécifique (comme HandBrake)
	// 3. En dernier recours, utiliser le device directement
	
	var input string
	mountPoint := ""
	
	// Étape 1: Vérifier si le DVD est déjà monté
	if strings.HasPrefix(devicePath, "/dev/") {
		// Chercher un point de montage pour ce device
		mountPoints := e.FindMountPoints()
		for _, mp := range mountPoints {
			// Vérifier si ce point de montage contient VIDEO_TS
			videoTSPath := filepath.Join(mp, "VIDEO_TS")
			if info, err := os.Stat(videoTSPath); err == nil && info.IsDir() {
				// Vérifier quel device est monté ici
				cmd := exec.Command("findmnt", "-n", "-o", "SOURCE", mp)
				if output, err := cmd.Output(); err == nil {
					mountedDevice := strings.TrimSpace(string(output))
					if mountedDevice == devicePath || strings.Contains(mountedDevice, filepath.Base(devicePath)) {
						mountPoint = mp
						log.Printf("[DVD Extraction] ✅ DVD déjà monté à: %s", mountPoint)
						break
					}
				}
			}
		}
		
		// Si pas monté, essayer de monter le DVD
		if mountPoint == "" {
			log.Printf("[DVD Extraction] 🔄 Tentative de montage du DVD...")
			mountPoint = e.mountDVD(devicePath)
			if mountPoint != "" {
				log.Printf("[DVD Extraction] ✅ DVD monté avec succès à: %s", mountPoint)
			}
		}
		
		// Si on a un point de montage, utiliser les fichiers VOB directement (comme HandBrake)
		if mountPoint != "" {
			videoTSPath := filepath.Join(mountPoint, "VIDEO_TS")
			
			// Analyser les fichiers VOB pour trouver ceux du titre le plus long
			vobFiles, err := e.analyzeVOBFiles(videoTSPath)
			if err == nil && len(vobFiles) > 0 {
				// Trouver les fichiers VOB du titre le plus long
				// Le titre le plus long correspond généralement au VTS principal
				// On cherche le VTS qui contient le plus de fichiers VOB (généralement le film principal)
				var bestTitleKey int
				maxFiles := 0
				for titleKey, files := range vobFiles {
					if len(files) > maxFiles {
						maxFiles = len(files)
						bestTitleKey = titleKey
					}
				}
				
				if maxFiles > 0 {
					// Utiliser concat pour joindre tous les fichiers VOB du titre (comme HandBrake)
					// ffmpeg peut lire plusieurs fichiers VOB en séquence avec concat
					vobList := vobFiles[bestTitleKey]
					
					// Créer un fichier concat pour ffmpeg
					concatFile := filepath.Join(filepath.Dir(outputPath), "concat_list.txt")
					concatContent := ""
					for _, vobFile := range vobList {
						// Échapper les caractères spéciaux pour le format concat
						escapedPath := strings.ReplaceAll(vobFile, "'", "'\\''")
						concatContent += fmt.Sprintf("file '%s'\n", escapedPath)
					}
					
					if err := os.WriteFile(concatFile, []byte(concatContent), 0644); err == nil {
						// Utiliser le fichier concat avec ffmpeg
						input = fmt.Sprintf("concat:%s", concatFile)
						log.Printf("[DVD Extraction] ✅ Utilisation concat avec %d fichier(s) VOB du titre %d", len(vobList), bestTitleKey)
						log.Printf("[DVD Extraction] 📁 Fichiers VOB: %v", vobList)
					} else {
						// Fallback: utiliser le premier fichier VOB
						input = vobList[0]
						log.Printf("[DVD Extraction] ⚠️  Erreur création concat, utilisation du premier VOB: %s", input)
					}
				} else {
					// Aucun fichier VOB trouvé, utiliser VIDEO_TS directement
					input = videoTSPath
					log.Printf("[DVD Extraction] ⚠️  Aucun fichier VOB trouvé, utilisation du dossier VIDEO_TS: %s", videoTSPath)
				}
			} else {
				// Si l'analyse VOB échoue, utiliser VIDEO_TS directement
				input = videoTSPath
				log.Printf("[DVD Extraction] ⚠️  Analyse VOB échouée, utilisation du dossier VIDEO_TS: %s", videoTSPath)
			}
		} else {
		// Si le montage échoue, essayer d'utiliser directement le device
		// Mais d'abord, essayer de lire les fichiers VOB depuis le device avec libdvdread
		// Si ça ne marche pas, utiliser le device directement
		log.Printf("[DVD Extraction] ⚠️  DVD non monté, tentative d'utilisation directe du device: %s", devicePath)
		// Essayer d'utiliser directement le device - ffmpeg peut parfois le lire
		// Mais on ne peut pas utiliser dvd://1 car le protocole n'est pas reconnu
		input = devicePath
		log.Printf("[DVD Extraction] ⚠️  Utilisation directe du device (sans montage): %s", devicePath)
		}
	} else if strings.Contains(devicePath, "VIDEO_TS") {
		// Si c'est déjà un chemin VIDEO_TS, analyser les fichiers VOB
		videoTSPath := devicePath
		vobFiles, err := e.analyzeVOBFiles(videoTSPath)
		if err == nil && len(vobFiles) > 0 {
			// Trouver le titre avec le plus de fichiers VOB
			var bestTitleKey int
			maxFiles := 0
			for titleKey, files := range vobFiles {
				if len(files) > maxFiles {
					maxFiles = len(files)
					bestTitleKey = titleKey
				}
			}
			
			if maxFiles > 0 {
				vobList := vobFiles[bestTitleKey]
				concatFile := filepath.Join(filepath.Dir(outputPath), "concat_list.txt")
				concatContent := ""
				for _, vobFile := range vobList {
					escapedPath := strings.ReplaceAll(vobFile, "'", "'\\''")
					concatContent += fmt.Sprintf("file '%s'\n", escapedPath)
				}
				
				if err := os.WriteFile(concatFile, []byte(concatContent), 0644); err == nil {
					input = fmt.Sprintf("concat:%s", concatFile)
					log.Printf("[DVD Extraction] ✅ Utilisation concat avec %d fichier(s) VOB depuis VIDEO_TS", len(vobList))
				} else {
					input = vobList[0]
					log.Printf("[DVD Extraction] ⚠️  Erreur création concat, utilisation du premier VOB: %s", input)
				}
			} else {
				input = videoTSPath
				log.Printf("[DVD Extraction] ⚠️  Aucun fichier VOB trouvé, utilisation du dossier VIDEO_TS: %s", videoTSPath)
			}
		} else {
			input = videoTSPath
			log.Printf("[DVD Extraction] Utilisation du dossier VIDEO_TS: %s", videoTSPath)
		}
	} else {
		// Point de montage direct
		videoTSPath := filepath.Join(devicePath, "VIDEO_TS")
		if info, err := os.Stat(videoTSPath); err == nil && info.IsDir() {
			// Analyser les fichiers VOB
			vobFiles, err := e.analyzeVOBFiles(videoTSPath)
			if err == nil && len(vobFiles) > 0 {
				var bestTitleKey int
				maxFiles := 0
				for titleKey, files := range vobFiles {
					if len(files) > maxFiles {
						maxFiles = len(files)
						bestTitleKey = titleKey
					}
				}
				
				if maxFiles > 0 {
					vobList := vobFiles[bestTitleKey]
					concatFile := filepath.Join(filepath.Dir(outputPath), "concat_list.txt")
					concatContent := ""
					for _, vobFile := range vobList {
						escapedPath := strings.ReplaceAll(vobFile, "'", "'\\''")
						concatContent += fmt.Sprintf("file '%s'\n", escapedPath)
					}
					
					if err := os.WriteFile(concatFile, []byte(concatContent), 0644); err == nil {
						input = fmt.Sprintf("concat:%s", concatFile)
						log.Printf("[DVD Extraction] ✅ Utilisation concat avec %d fichier(s) VOB", len(vobList))
					} else {
						input = vobList[0]
						log.Printf("[DVD Extraction] ⚠️  Erreur création concat, utilisation du premier VOB: %s", input)
					}
				} else {
					input = videoTSPath
					log.Printf("[DVD Extraction] ⚠️  Aucun fichier VOB trouvé, utilisation du dossier VIDEO_TS: %s", videoTSPath)
				}
			} else {
				input = videoTSPath
				log.Printf("[DVD Extraction] Utilisation du dossier VIDEO_TS: %s", videoTSPath)
			}
		} else {
			input = devicePath
			log.Printf("[DVD Extraction] Utilisation du point de montage: %s", devicePath)
		}
	}
	
	// Vérification AVANT d'ajouter -i : s'assurer que input n'est JAMAIS dvd://
	if strings.HasPrefix(input, "dvd://") {
		log.Printf("[DVD Extraction] ❌ ERREUR CRITIQUE: input est dvd://%s, remplacement FORCÉ par %s", input, devicePath)
		input = devicePath
	}
	
	args = append(args, "-i", input)
	log.Printf("[DVD Extraction] 🔧 Après ajout -i: args=%v", args)
	log.Printf("[DVD Extraction] 🔧 Input utilisé: %s (devicePath=%s)", input, devicePath)
	
	// Vérification finale : s'assurer que input n'est pas dvd:// dans args
	for i, arg := range args {
		if arg == "-i" && i+1 < len(args) && strings.HasPrefix(args[i+1], "dvd://") {
			log.Printf("[DVD Extraction] ❌ ERREUR: args[%d] contient encore dvd://%s, remplacement par %s", i+1, args[i+1], devicePath)
			args[i+1] = devicePath
			log.Printf("[DVD Extraction] ✅ CORRECTION: args[%d] remplacé par %s", i+1, devicePath)
			break
		}
	}

	// D'abord, lister les streams disponibles pour diagnostic et mapping correct
	log.Printf("[DVD Extraction] 🔍 Diagnostic: Liste des streams disponibles...")
	
	// Vérification CRITIQUE : s'assurer que input n'est JAMAIS dvd:// avant ffprobe
	probeInput := input
	if strings.HasPrefix(probeInput, "dvd://") {
		log.Printf("[DVD Extraction] ❌ ERREUR CRITIQUE: probeInput est dvd://%s, remplacement FORCÉ par %s", probeInput, devicePath)
		probeInput = devicePath
	}
	
	probeArgs := []string{"-v", "error", "-show_streams", "-of", "json"}
	
	// Utiliser -analyzeduration et -probesize pour mieux lire le DVD
	probeArgs = append(probeArgs, "-analyzeduration", "10000000", "-probesize", "10000000")
	probeArgs = append(probeArgs, probeInput)
	
	probeCmd := exec.Command(e.ffprobePath, probeArgs...)
	// Plus besoin de DVD_DEVICE car on utilise directement le device
	log.Printf("[DVD Extraction] 🔧 Commande ffprobe: %s %s (input vérifié: %s)", e.ffprobePath, strings.Join(probeArgs, " "), probeInput)
	probeOutput, probeErr := probeCmd.Output()
	
	var availableVideoStreams []int
	var availableAudioStreams []int
	var availableSubtitleStreams []int
	
	if probeErr == nil {
		var probeData struct {
			Streams []struct {
				Index      int    `json:"index"`
				CodecType  string `json:"codec_type"`
				CodecName  string `json:"codec_name"`
				CodecLongName string `json:"codec_long_name"`
			} `json:"streams"`
		}
		if err := json.Unmarshal(probeOutput, &probeData); err == nil {
			log.Printf("[DVD Extraction] 📊 Streams disponibles: %d", len(probeData.Streams))
			for _, stream := range probeData.Streams {
				log.Printf("[DVD Extraction]   Stream %d: type=%s, codec=%s (%s)", 
					stream.Index, stream.CodecType, stream.CodecName, stream.CodecLongName)
				switch stream.CodecType {
				case "video":
					availableVideoStreams = append(availableVideoStreams, stream.Index)
				case "audio":
					availableAudioStreams = append(availableAudioStreams, stream.Index)
				case "subtitle":
					availableSubtitleStreams = append(availableSubtitleStreams, stream.Index)
				}
			}
			log.Printf("[DVD Extraction] 📊 Streams par type: video=%d, audio=%d, subtitle=%d", 
				len(availableVideoStreams), len(availableAudioStreams), len(availableSubtitleStreams))
		}
	} else {
		log.Printf("[DVD Extraction] ⚠️ Impossible de lister les streams (non bloquant): %v", probeErr)
	}

	// Mapper la piste vidéo
	// Utiliser le premier stream vidéo disponible si videoTrackIndex est 0
	videoStreamIdx := 0
	if len(availableVideoStreams) > 0 {
		// Utiliser l'index réel du stream vidéo dans la liste
		if videoTrackIndex < len(availableVideoStreams) {
			videoStreamIdx = availableVideoStreams[videoTrackIndex]
		} else {
			videoStreamIdx = availableVideoStreams[0]
			log.Printf("[DVD Extraction] ⚠️ videoTrackIndex %d hors limites, utilisation du premier stream vidéo (index %d)", 
				videoTrackIndex, videoStreamIdx)
		}
	} else {
		// Par défaut, utiliser l'index 1 (après dvd_nav_packet)
		videoStreamIdx = 1
		log.Printf("[DVD Extraction] ⚠️ Aucun stream vidéo détecté, utilisation de l'index par défaut: %d", videoStreamIdx)
	}
	
	videoMap := fmt.Sprintf("0:%d", videoStreamIdx)
	log.Printf("[DVD Extraction] 🎬 Mapping piste vidéo: %s (stream index=%d)", videoMap, videoStreamIdx)
	args = append(args, "-map", videoMap)

	// Ajouter les pistes audio (vérifier d'abord qu'elles existent)
	if len(audioTrackIndices) > 0 {
		if len(availableAudioStreams) == 0 {
			log.Printf("[DVD Extraction] ⚠️ Aucun stream audio détecté par ffprobe, mais des pistes audio sont demandées")
			log.Printf("[DVD Extraction] 🔍 Tentative de détection audio avec ffmpeg directement...")
			// Essayer de détecter les streams audio avec ffmpeg directement
			// Parfois ffprobe ne détecte pas les streams audio avec dvd://, mais ffmpeg les voit
			// On va utiliser une approche plus permissive : mapper par type plutôt que par index
			// Utiliser -map 0:a pour mapper tous les streams audio, ou -map 0:a:0 pour le premier
			log.Printf("[DVD Extraction] 🎵 Mapping audio par type (0:a:%d) - ffmpeg détectera les streams", audioTrackIndices[0])
			args = append(args, "-map", fmt.Sprintf("0:a:%d", audioTrackIndices[0]))
			// Si ffmpeg ne trouve pas de stream audio, il échouera, mais au moins on aura essayé
		} else {
			// Mapper les streams audio disponibles
			for _, audioIdx := range audioTrackIndices {
				if audioIdx < len(availableAudioStreams) {
					audioStreamIdx := availableAudioStreams[audioIdx]
					audioMap := fmt.Sprintf("0:%d", audioStreamIdx)
					log.Printf("[DVD Extraction] 🎵 Mapping piste audio: %s (stream index=%d, audioIdx=%d)", 
						audioMap, audioStreamIdx, audioIdx)
					args = append(args, "-map", audioMap)
				} else {
					log.Printf("[DVD Extraction] ⚠️ audioIdx %d hors limites (disponibles: %d), utilisation du mapping par type", 
						audioIdx, len(availableAudioStreams))
					// Essayer quand même avec le mapping par type
					args = append(args, "-map", fmt.Sprintf("0:a:%d", audioIdx))
				}
			}
		}
	} else {
		// Aucune piste audio sélectionnée : créer une vidéo muette
		// Ne pas mapper de piste audio
		log.Printf("[DVD Extraction] ℹ️ Aucune piste audio sélectionnée, vidéo muette")
	}

	// Ajouter les sous-titres (vérifier d'abord qu'ils existent)
	for _, subIdx := range subtitleTrackIndices {
		if len(availableSubtitleStreams) > 0 && subIdx < len(availableSubtitleStreams) {
			subtitleStreamIdx := availableSubtitleStreams[subIdx]
			subtitleMap := fmt.Sprintf("0:%d", subtitleStreamIdx)
			log.Printf("[DVD Extraction] 📝 Mapping sous-titre: %s (stream index=%d, subIdx=%d)", 
				subtitleMap, subtitleStreamIdx, subIdx)
			args = append(args, "-map", subtitleMap)
		} else {
			log.Printf("[DVD Extraction] ⚠️ subIdx %d hors limites ou aucun sous-titre disponible, ignoré", subIdx)
		}
	}

	// Codecs et options
	args = append(args,
		"-c:v", "libx264",
		"-preset", "medium",
		"-crf", "23",
	)

	// Ajouter l'audio seulement si des pistes audio sont sélectionnées
	if len(audioTrackIndices) > 0 {
		args = append(args, "-c:a", "aac", "-b:a", "192k")
	} else {
		// Vidéo muette : ne pas encoder d'audio
		args = append(args, "-an") // -an = no audio
	}

	// Ajouter les sous-titres si présents
	if len(subtitleTrackIndices) > 0 {
		args = append(args, "-c:s", "mov_text")
	}

	args = append(args,
		"-movflags", "+faststart",
		"-y",
		outputPath,
	)

	// DEBUG: Vérifier que args ne contient pas -dvd_device
	log.Printf("[DVD Extraction] 🔧 AVANT vérification -dvd_device: args=%v", args)
	for i, arg := range args {
		if arg == "-dvd_device" {
			log.Printf("[DVD Extraction] ❌ ERREUR: -dvd_device trouvé dans args à l'index %d! Args complets: %v", i, args)
			return "", fmt.Errorf("ERREUR: -dvd_device ne doit pas être dans les arguments")
		}
	}
	log.Printf("[DVD Extraction] ✅ Vérification OK: -dvd_device n'est PAS dans args")
	
	cmd := exec.Command(e.ffmpegPath, args...)
	
	// Plus besoin de DVD_DEVICE car on utilise directement le device
	// ffmpeg détectera automatiquement les streams du DVD
	log.Printf("[DVD Extraction] 🔧 Commande complète: %s %s", e.ffmpegPath, strings.Join(args, " "))
	
	log.Printf("[DVD Extraction] 🔧 Commande créée: %s avec args: %v", e.ffmpegPath, args)
	
	// Log de la commande complète pour debug
	log.Printf("[DVD Extraction] Commande ffmpeg finale: %s %s", e.ffmpegPath, strings.Join(args, " "))
	log.Printf("[DVD Extraction] DEBUG Args complets: %v", args)
	
	// Capturer stderr séparément pour voir les warnings
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	
	log.Printf("[DVD Extraction] 🚀 Démarrage de l'extraction ffmpeg...")
	startTime := time.Now()
	
	// Capturer aussi stdout pour voir les messages de progression
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	
	err = cmd.Run()
	duration := time.Since(startTime)
	outputStr := string(stderr.Bytes())
	stdoutStr := string(stdout.Bytes())
	
	log.Printf("[DVD Extraction] ⏱️ Extraction terminée en %v", duration)
	
	// Log la sortie complète pour diagnostic
	if len(outputStr) > 0 {
		log.Printf("[DVD Extraction] 📋 Sortie stderr (premiers 1000 caractères): %s", 
			func() string {
				if len(outputStr) > 1000 {
					return outputStr[:1000] + "..."
				}
				return outputStr
			}())
	}
	if len(stdoutStr) > 0 {
		log.Printf("[DVD Extraction] 📋 Sortie stdout (premiers 1000 caractères): %s", 
			func() string {
				if len(stdoutStr) > 1000 {
					return stdoutStr[:1000] + "..."
				}
				return stdoutStr
			}())
	}
	
	if err != nil {
		log.Printf("[DVD Extraction] ❌ ERREUR ffmpeg: %v", err)
		log.Printf("[DVD Extraction] Sortie stderr ffmpeg complète (premiers 5000 caractères): %s", 
			func() string {
				if len(outputStr) > 5000 {
					return outputStr[:5000] + "..."
				}
				return outputStr
			}())
		return "", fmt.Errorf("erreur lors de l'extraction: %w (output: %s)", err, outputStr)
	}

	log.Printf("[DVD Extraction] ✅ Extraction ffmpeg réussie")
	
	// Vérifier que le fichier existe et a une taille valide
	if fileInfo, err := os.Stat(outputPath); err != nil {
		log.Printf("[DVD Extraction] ❌ ERREUR: Le fichier extrait n'existe pas: %s", outputPath)
		return "", fmt.Errorf("fichier extrait non trouvé: %w", err)
	} else {
		log.Printf("[DVD Extraction] 📁 Fichier extrait: %s, taille: %d bytes (%.2f MB)", 
			outputPath, fileInfo.Size(), float64(fileInfo.Size())/1024/1024)
		
		// Vérifier la taille minimale (au moins 5 MB pour un extrait valide - un DVD de quelques secondes fait déjà plusieurs MB)
		if fileInfo.Size() < 5*1024*1024 {
			log.Printf("[DVD Extraction] ❌ ERREUR: Fichier très petit (%d bytes, %.2f MB), l'extraction a probablement échoué", 
				fileInfo.Size(), float64(fileInfo.Size())/1024/1024)
			log.Printf("[DVD Extraction] 🔍 Vérification avec ffprobe pour confirmer...")
			
			// Vérifier avec ffprobe que le fichier est valide
			probeCmd := exec.Command(e.ffprobePath, "-v", "error", "-show_entries", "format=duration,size", 
				"-of", "json", outputPath)
			probeOutput, probeErr := probeCmd.Output()
			if probeErr != nil {
				log.Printf("[DVD Extraction] ❌ ERREUR: Fichier invalide selon ffprobe: %v", probeErr)
				log.Printf("[DVD Extraction] Sortie ffprobe: %s", string(probeOutput))
				return "", fmt.Errorf("fichier extrait trop petit (%d bytes) et invalide selon ffprobe: %w", fileInfo.Size(), probeErr)
			}
			
			var probeData struct {
				Format struct {
					Duration string `json:"duration"`
					Size     string `json:"size"`
				} `json:"format"`
			}
			if err := json.Unmarshal(probeOutput, &probeData); err == nil {
				if duration, err := strconv.ParseFloat(probeData.Format.Duration, 64); err == nil {
					if duration < 1.0 {
						log.Printf("[DVD Extraction] ❌ ERREUR: Durée invalide (%.2f secondes), fichier corrompu", duration)
						return "", fmt.Errorf("fichier extrait invalide: durée trop courte (%.2f secondes) et taille trop petite (%d bytes)", duration, fileInfo.Size())
					}
					log.Printf("[DVD Extraction] ⚠️ Fichier petit mais durée valide: %.2f secondes", duration)
				} else {
					log.Printf("[DVD Extraction] ❌ ERREUR: Impossible de parser la durée: %v", err)
					return "", fmt.Errorf("fichier extrait invalide: taille trop petite (%d bytes) et durée non parsable", fileInfo.Size())
				}
			} else {
				log.Printf("[DVD Extraction] ❌ ERREUR: Impossible de parser la réponse ffprobe: %v", err)
				return "", fmt.Errorf("fichier extrait invalide: taille trop petite (%d bytes) et JSON invalide", fileInfo.Size())
			}
		} else {
			// Fichier de taille raisonnable, vérifier avec ffprobe
			log.Printf("[DVD Extraction] 🔍 Vérification de la validité du fichier avec ffprobe...")
			probeCmd := exec.Command(e.ffprobePath, "-v", "error", "-show_entries", "format=duration,size", 
				"-of", "json", outputPath)
			probeOutput, probeErr := probeCmd.Output()
			if probeErr != nil {
				log.Printf("[DVD Extraction] ❌ ERREUR: Fichier invalide selon ffprobe: %v", probeErr)
				log.Printf("[DVD Extraction] Sortie ffprobe: %s", string(probeOutput))
				return "", fmt.Errorf("fichier extrait invalide (ffprobe échoue): %w", probeErr)
			}
			
			var probeData struct {
				Format struct {
					Duration string `json:"duration"`
					Size     string `json:"size"`
				} `json:"format"`
			}
			if err := json.Unmarshal(probeOutput, &probeData); err == nil {
				if duration, err := strconv.ParseFloat(probeData.Format.Duration, 64); err == nil {
					log.Printf("[DVD Extraction] ✅ Fichier valide: durée=%.2f secondes (%.2f minutes)", 
						duration, duration/60)
				} else {
					log.Printf("[DVD Extraction] ⚠️ Impossible de parser la durée: %v", err)
				}
			}
			log.Printf("[DVD Extraction] ✅ Fichier extrait validé avec succès")
		}
	}
	
	return outputPath, nil
}
