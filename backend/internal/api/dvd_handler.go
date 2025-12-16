package api

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"streammake/internal/config"
	"streammake/internal/metadata"
	"streammake/internal/models"
	"streammake/internal/video"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DVDHandler struct {
	db              *gorm.DB
	cfg             *config.Config
	extractor       *video.DVDExtractor
	processor       *video.Processor
	metadataService *metadata.WebScraperService
	uploadHandler   *UploadHandler // Référence pour processVideo
}

func NewDVDHandler(db *gorm.DB, cfg *config.Config, uploadHandler *UploadHandler) *DVDHandler {
	return &DVDHandler{
		db:              db,
		cfg:             cfg,
		extractor:       video.NewDVDExtractor(),
		processor:       video.NewProcessor(cfg.VideoStoragePath, cfg.ThumbnailStoragePath),
		metadataService: metadata.NewWebScraperService(),
		uploadHandler:   uploadHandler,
	}
}

// DetectDVD détecte le lecteur DVD
func (h *DVDHandler) DetectDVD(c *gin.Context) {
	device, err := h.extractor.DetectDVDDevice()
	if err != nil {
		// Retourner 200 avec un message d'erreur plutôt que 404
		// pour que le frontend puisse gérer l'erreur correctement
		// Inclure aussi une liste de devices à essayer manuellement
		possibleDevices := []string{"/dev/sr0", "/dev/sr1", "/dev/sr2", "/dev/dvd", "/dev/cdrom"}
		var availableDevices []string
		for _, d := range possibleDevices {
			if info, statErr := os.Stat(d); statErr == nil && !info.IsDir() {
				if info.Mode()&os.ModeDevice != 0 {
					availableDevices = append(availableDevices, d)
				}
			}
		}
		
		c.JSON(http.StatusOK, gin.H{
			"device": "",
			"error": "Aucun lecteur DVD détecté automatiquement",
			"message": err.Error(),
			"success": false,
			"available_devices": availableDevices,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"device": device,
		"message": "Lecteur DVD détecté",
		"success": true,
	})
}

// GetDVDInfo récupère les informations du DVD
func (h *DVDHandler) GetDVDInfo(c *gin.Context) {
	devicePath := c.Query("device")
	if devicePath == "" {
		devicePath = "/dev/dvd" // Par défaut
	}

	info, err := h.extractor.GetDVDInfo(devicePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Erreur lors de la lecture du DVD",
			"message": err.Error(),
		})
		return
	}

	// Si on a un titre et/ou réalisateur, chercher automatiquement une miniature
	// Essayer plusieurs variantes de recherche
	searchQueries := []string{}
	
	if info.Title != "" && info.Title != "DVD Extraite" {
		searchQueries = append(searchQueries, info.Title)
		if info.Director != "" {
			searchQueries = append(searchQueries, info.Title+" "+info.Director)
		}
	}
	
	// Si pas de titre mais qu'on a le device, essayer de chercher "KILL" ou autres titres connus
	if info.Title == "" || info.Title == "DVD Extraite" {
		// Chercher avec des termes génériques basés sur le device
		searchQueries = append(searchQueries, "KILL film", "KILL NIKHIL NAGESH BHAT")
	}
	
	// Chercher les métadonnées pour obtenir le poster
	for _, searchQuery := range searchQueries {
		if searchQuery == "" {
			continue
		}
		
		metadataResults, err := h.metadataService.SearchMoviesFromFilenameWeb(searchQuery)
		if err == nil && len(metadataResults) > 0 {
			// Prendre le premier résultat (le plus pertinent)
			bestMatch := metadataResults[0]
			if bestMatch.PosterURL != "" {
				info.PosterURL = bestMatch.PosterURL
			}
			// Enrichir aussi les autres métadonnées si manquantes
			if info.Title == "" || info.Title == "DVD Extraite" {
				if bestMatch.Title != "" {
					info.Title = bestMatch.Title
				}
			}
			if info.Director == "" && bestMatch.Director != "" {
				info.Director = bestMatch.Director
			}
			if info.Year == 0 && bestMatch.ProductionYear > 0 {
				info.Year = bestMatch.ProductionYear
			}
			break // On a trouvé, on arrête
		}
	}

	c.JSON(http.StatusOK, info)
}

type ExtractDVDRequest struct {
	DevicePath         string   `json:"device_path"`
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	CategoryID         string   `json:"category_id"`
	SubCategoryID      string   `json:"sub_category_id"`
	IsPublic           bool     `json:"is_public"`
	VideoTrackIndex    int      `json:"video_track_index"`
	AudioTrackIndices  []int    `json:"audio_track_indices"`
	SubtitleTrackIndices []int  `json:"subtitle_track_indices"`
	Qualities          []string `json:"qualities"` // ["1080p", "720p", "480p"]
}

// ExtractDVD extrait le DVD et crée une vidéo
func (h *DVDHandler) ExtractDVD(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr := userID.(string)
	userUUID, _ := uuid.Parse(userIDStr)

	var req ExtractDVDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Valider les paramètres
	if req.DevicePath == "" {
		req.DevicePath = "/dev/dvd"
	}
	if len(req.Qualities) == 0 {
		req.Qualities = []string{"1080p", "720p", "480p"}
	}
	// Si aucune piste audio n'est sélectionnée, on peut extraire sans audio (vidéo muette)
	// Ne pas forcer la première piste audio par défaut

	// Créer l'ID de la vidéo
	videoID := uuid.New()

	// Créer l'entrée en base
	videoModel := models.Video{
		ID:            videoID,
		Title:         req.Title,
		Description:   req.Description,
		UserID:        userUUID,
		Status:        models.VideoStatusProcessing,
		CategoryID:    nil,
		SubCategoryID: nil,
		IsPublic:      req.IsPublic,
	}

	if req.CategoryID != "" {
		catID, _ := uuid.Parse(req.CategoryID)
		videoModel.CategoryID = &catID
	}
	if req.SubCategoryID != "" {
		subCatID, _ := uuid.Parse(req.SubCategoryID)
		videoModel.SubCategoryID = &subCatID
	}

	if err := h.db.Create(&videoModel).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la création de la vidéo"})
		return
	}

	// Extraire le DVD en arrière-plan
	go h.processDVDExtraction(videoID, req)

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Extraction du DVD démarrée",
		"video_id": videoID,
		"status":   "processing",
	})
}

func (h *DVDHandler) processDVDExtraction(videoID uuid.UUID, req ExtractDVDRequest) {
	log.Printf("[DVD Extraction] Démarrage extraction pour vidéo %s", videoID)
	
	// Créer le dossier de sortie
	outputDir := filepath.Join(h.cfg.VideoStoragePath, "dvd", videoID.String())
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Printf("[DVD Extraction] ERREUR: Impossible de créer le dossier de sortie: %v", err)
		h.db.Model(&models.Video{}).Where("id = ?", videoID).Update("status", models.VideoStatusError)
		return
	}
	log.Printf("[DVD Extraction] Dossier de sortie créé: %s", outputDir)

	// Extraire le DVD en MP4 brut
	outputPath := filepath.Join(outputDir, "video.mp4")
	log.Printf("[DVD Extraction] Démarrage extraction DVD depuis %s vers %s", req.DevicePath, outputPath)
	log.Printf("[DVD Extraction] Paramètres: video_track=%d, audio_tracks=%v, subtitle_tracks=%v, qualities=%v", 
		req.VideoTrackIndex, req.AudioTrackIndices, req.SubtitleTrackIndices, req.Qualities)
	
	extractedPath, err := h.extractor.ExtractDVD(
		req.DevicePath,
		outputPath,
		req.VideoTrackIndex,
		req.AudioTrackIndices,
		req.SubtitleTrackIndices,
		req.Qualities,
	)
	if err != nil {
		log.Printf("[DVD Extraction] ERREUR lors de l'extraction: %v", err)
		h.db.Model(&models.Video{}).Where("id = ?", videoID).Update("status", models.VideoStatusError)
		return
	}
	
	log.Printf("[DVD Extraction] Extraction terminée, fichier: %s", extractedPath)

	// Vérifier que le fichier extrait existe
	if _, err := os.Stat(extractedPath); os.IsNotExist(err) {
		log.Printf("[DVD Extraction] ERREUR: Le fichier extrait n'existe pas: %s", extractedPath)
		h.db.Model(&models.Video{}).Where("id = ?", videoID).Update("status", models.VideoStatusError)
		return
	}
	
	// Vérifier la taille du fichier
	if fileInfo, err := os.Stat(extractedPath); err == nil {
		log.Printf("[DVD Extraction] Fichier extrait: %s, taille: %d bytes", extractedPath, fileInfo.Size())
		if fileInfo.Size() == 0 {
			log.Printf("[DVD Extraction] ERREUR: Le fichier extrait est vide")
			h.db.Model(&models.Video{}).Where("id = ?", videoID).Update("status", models.VideoStatusError)
			return
		}
	}

	// Calculer le checksum du fichier extrait
	checksum, err := h.calculateFileChecksum(outputPath)
	if err == nil && checksum != "" {
		h.db.Model(&models.Video{}).Where("id = ?", videoID).Update("checksum", checksum)
	}

	// Utiliser le chemin extrait au lieu de outputPath
	videoPathToProcess := extractedPath
	if videoPathToProcess == "" {
		videoPathToProcess = outputPath
	}
	
	log.Printf("[DVD Extraction] Démarrage traitement vidéo: %s", videoPathToProcess)
	
	// Lancer le traitement vidéo complet (compression optimale + transcodage) automatiquement
	// C'est le même traitement que pour les uploads : compression H.265/HEVC + transcodage multi-qualités
	if h.uploadHandler != nil {
		ctx, cancel := context.WithCancel(context.Background())
		h.uploadHandler.processingMutex.Lock()
		h.uploadHandler.processingVideos[videoID] = cancel
		h.uploadHandler.processingMutex.Unlock()
		
		// Utiliser le même processVideo que pour les uploads
		// Cela va automatiquement :
		// 1. Compresser la vidéo avec H.265/HEVC (réduction ~50%)
		// 2. Générer les miniatures
		// 3. Transcoder en plusieurs qualités (360p, 480p, 720p, 1080p)
		// 4. Générer les formats HLS pour streaming adaptatif
		log.Printf("[DVD Extraction] Lancement processVideo pour vidéo %s", videoID)
		go h.uploadHandler.processVideo(ctx, videoID, videoPathToProcess)
	} else {
		// Fallback si uploadHandler n'est pas disponible
		// Générer au moins la miniature et la durée
		if thumbnailPath, err := h.processor.GenerateThumbnail(outputPath, videoID); err == nil {
			h.db.Model(&models.Video{}).Where("id = ?", videoID).Update("thumbnail_path", thumbnailPath)
		}
		if duration, err := h.processor.GetVideoDuration(outputPath); err == nil {
			h.db.Model(&models.Video{}).Where("id = ?", videoID).Update("duration", duration)
		}
		h.db.Model(&models.Video{}).Where("id = ?", videoID).Update("status", models.VideoStatusReady)
	}
}

// calculateFileChecksum calcule le checksum SHA256 d'un fichier
func (h *DVDHandler) calculateFileChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// GetExtractionStatus récupère le statut de l'extraction
func (h *DVDHandler) GetExtractionStatus(c *gin.Context) {
	videoID := c.Param("id")
	
	var video models.Video
	if err := h.db.Where("id = ?", videoID).First(&video).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vidéo non trouvée"})
		return
	}

	// Compter les formats créés pour calculer la progression
	var formatCount int64
	h.db.Model(&models.VideoFormat{}).Where("video_id = ?", videoID).Count(&formatCount)
	
	// Calculer la progression basée sur les formats créés
	// 0-30%: Extraction, 30-50%: Compression, 50-90%: Transcodage (4 qualités), 90-100%: Finalisation
	progress := 0
	if video.Status == models.VideoStatusProcessing {
		// Si on a des formats, on est au moins à 30% (compression terminée)
		if formatCount > 0 {
			// 30% base + (formatCount / 4 qualités) * 60%
			progress = 30 + int(float64(formatCount)/4.0*60)
			if progress > 90 {
				progress = 90
			}
		} else {
			// Pas encore de formats, on est dans l'extraction (0-30%)
			progress = 5 // Minimum pour montrer qu'on a commencé
		}
	} else if video.Status == models.VideoStatusReady {
		progress = 100
	}

	c.JSON(http.StatusOK, gin.H{
		"video_id": video.ID,
		"status":   video.Status,
		"title":    video.Title,
		"progress": progress,
		"formats":  formatCount,
	})
}

// CancelExtraction annule l'extraction d'un DVD en cours
func (h *DVDHandler) CancelExtraction(c *gin.Context) {
	videoIDParam := c.Param("id")
	videoID, err := uuid.Parse(videoIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de vidéo invalide"})
		return
	}

	// Vérifier que la vidéo existe
	var video models.Video
	if err := h.db.Where("id = ?", videoID).First(&video).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vidéo non trouvée"})
		return
	}

	// Vérifier que c'est bien en cours de traitement
	if video.Status != models.VideoStatusProcessing {
		c.JSON(http.StatusBadRequest, gin.H{"error": "L'extraction n'est pas en cours"})
		return
	}

	// Annuler le traitement si disponible
	if h.uploadHandler != nil {
		h.uploadHandler.processingMutex.Lock()
		if cancel, exists := h.uploadHandler.processingVideos[videoID]; exists {
			cancel() // Annuler le contexte
			delete(h.uploadHandler.processingVideos, videoID)
		}
		h.uploadHandler.processingMutex.Unlock()
	}

	// Mettre à jour le statut
	h.db.Model(&models.Video{}).Where("id = ?", videoID).Update("status", models.VideoStatusCancelled)

	// Nettoyer les formats partiels
	if h.uploadHandler != nil {
		h.uploadHandler.cleanupPartialFormats(videoID)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Extraction annulée avec succès",
		"video_id": videoID,
		"status":   "cancelled",
	})
}

