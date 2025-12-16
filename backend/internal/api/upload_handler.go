package api

import (
	"context"
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"streammake/internal/config"
	"streammake/internal/metadata"
	"streammake/internal/models"
	"streammake/internal/video"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UploadHandler struct {
	db        *gorm.DB
	cfg       *config.Config
	processor *video.Processor
	metadataService *metadata.MetadataService
	// Suivi des traitements en cours avec context pour annulation
	processingVideos map[uuid.UUID]context.CancelFunc
	processingMutex  sync.RWMutex
}

func NewUploadHandler(db *gorm.DB, cfg *config.Config) *UploadHandler {
	return &UploadHandler{
		db:               db,
		cfg:              cfg,
		processor:        video.NewProcessor(cfg.VideoStoragePath, cfg.ThumbnailStoragePath),
		metadataService:  metadata.NewMetadataService(cfg.TMDBAPIKey),
		processingVideos: make(map[uuid.UUID]context.CancelFunc),
	}
}

func (h *UploadHandler) UploadVideo(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr := userID.(string)
	userUUID, _ := uuid.Parse(userIDStr)

	// Récupérer le fichier
	file, err := c.FormFile("video")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Fichier vidéo requis"})
		return
	}

	// Récupérer les métadonnées
	title := strings.TrimSpace(c.PostForm("title"))
	// Valider et nettoyer le titre
	if title == "" || title == "null" || title == "Null" || title == "NULL" {
		// Utiliser le nom du fichier sans extension comme titre
		title = strings.TrimSuffix(file.Filename, filepath.Ext(file.Filename))
		if title == "" {
			title = "Vidéo sans titre"
		}
	}
	description := strings.TrimSpace(c.PostForm("description"))
	categoryIDStr := c.PostForm("category_id")
	subCategoryIDStr := c.PostForm("sub_category_id")
	isPublic := c.PostForm("is_public") != "false"
	
	// Métadonnées enrichies
	director := strings.TrimSpace(c.PostForm("director"))
	actors := strings.TrimSpace(c.PostForm("actors"))
	releaseDate := strings.TrimSpace(c.PostForm("release_date"))
	productionYearStr := strings.TrimSpace(c.PostForm("production_year"))
	synopsis := strings.TrimSpace(c.PostForm("synopsis"))
	genre := strings.TrimSpace(c.PostForm("genre"))
	ratingStr := strings.TrimSpace(c.PostForm("rating"))
	imdbID := strings.TrimSpace(c.PostForm("imdb_id"))
	tmdbIDStr := strings.TrimSpace(c.PostForm("tmdb_id"))
	posterURL := strings.TrimSpace(c.PostForm("poster_url"))
	backdropURL := strings.TrimSpace(c.PostForm("backdrop_url"))
	trailerURL := strings.TrimSpace(c.PostForm("trailer_url"))
	runtimeStr := strings.TrimSpace(c.PostForm("runtime"))
	writer := strings.TrimSpace(c.PostForm("writer"))
	platform := strings.TrimSpace(c.PostForm("platform"))
	
	// Convertir les valeurs numériques
	var productionYear int
	if productionYearStr != "" {
		fmt.Sscanf(productionYearStr, "%d", &productionYear)
	}
	var rating float64
	if ratingStr != "" {
		fmt.Sscanf(ratingStr, "%f", &rating)
	}
	var tmdbID int
	if tmdbIDStr != "" {
		fmt.Sscanf(tmdbIDStr, "%d", &tmdbID)
	}
	var runtime int
	if runtimeStr != "" {
		fmt.Sscanf(runtimeStr, "%d", &runtime)
	}

	var categoryID, subCategoryID *uuid.UUID
	if categoryIDStr != "" {
		if id, err := uuid.Parse(categoryIDStr); err == nil {
			categoryID = &id
		}
	}
	if subCategoryIDStr != "" {
		if id, err := uuid.Parse(subCategoryIDStr); err == nil {
			subCategoryID = &id
		}
	}

	// Créer l'entrée vidéo
	videoID := uuid.New()
	videoFileName := fmt.Sprintf("%s_%d%s", videoID.String(), time.Now().Unix(), filepath.Ext(file.Filename))
	videoPath := filepath.Join(h.cfg.VideoStoragePath, videoFileName)

	// Sauvegarder le fichier d'abord
	if err := c.SaveUploadedFile(file, videoPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de l'enregistrement du fichier"})
		return
	}

	// Calculer le checksum SHA256 du fichier sauvegardé
	savedFile, err := os.Open(videoPath)
	if err != nil {
		os.Remove(videoPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de l'ouverture du fichier sauvegardé"})
		return
	}
	defer savedFile.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, savedFile); err != nil {
		os.Remove(videoPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors du calcul du checksum"})
		return
	}
	checksum := fmt.Sprintf("%x", hash.Sum(nil))

	// Créer l'entrée en base avec métadonnées enrichies
	videoModel := models.Video{
		ID:              videoID,
		Title:           title,
		Description:     description,
		UserID:          userUUID,
		Status:          models.VideoStatusProcessing,
		CategoryID:      categoryID,
		SubCategoryID:   subCategoryID,
		IsPublic:        isPublic,
		Checksum:        checksum,
		// Métadonnées enrichies
		Director:        director,
		Actors:          actors,
		ReleaseDate:     releaseDate,
		ProductionYear:  productionYear,
		Synopsis:        synopsis,
		Genre:           genre,
		Rating:          rating,
		IMDBID:          imdbID,
		TMDBID:          tmdbID,
		PosterURL:       posterURL,
		BackdropURL:     backdropURL,
		TrailerURL:      trailerURL,
		Runtime:         runtime,
		Writer:          writer,
		Platform:        platform,
	}

	if err := h.db.Create(&videoModel).Error; err != nil {
		os.Remove(videoPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la création de la vidéo"})
		return
	}

	// Créer un context avec annulation pour le traitement
	ctx, cancel := context.WithCancel(context.Background())
	
	// Enregistrer la fonction d'annulation
	h.processingMutex.Lock()
	h.processingVideos[videoID] = cancel
	h.processingMutex.Unlock()

	// Traiter la vidéo en arrière-plan avec le context
	go h.processVideo(ctx, videoID, videoPath)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Vidéo uploadée avec succès",
		"video_id": videoID,
		"status":   "processing",
	})
}

func (h *UploadHandler) processVideo(ctx context.Context, videoID uuid.UUID, videoPath string) {
	// Nettoyer la fonction d'annulation à la fin du traitement
	defer func() {
		h.processingMutex.Lock()
		delete(h.processingVideos, videoID)
		h.processingMutex.Unlock()
	}()

	// Vérifier si l'annulation a été demandée avant de commencer
	select {
	case <-ctx.Done():
		fmt.Printf("Traitement de la vidéo %s annulé avant le début\n", videoID)
		h.db.Model(&models.Video{}).Where("id = ?", videoID).Update("status", models.VideoStatusCancelled)
		return
	default:
	}

	// Récupérer le titre de la vidéo pour générer une miniature textuelle si nécessaire
	var videoModel models.Video
	h.db.Where("id = ?", videoID).First(&videoModel)
	title := videoModel.Title
	description := videoModel.Description

	// Extraire les métadonnées AVANT de générer la miniature pour avoir plus d'infos
	metadata, err := h.processor.ExtractMetadata(videoPath)
	if err == nil && metadata != nil {
		// Mettre à jour la vidéo avec les métadonnées extraites
		updates := make(map[string]interface{})
		if metadata.Title != "" && videoModel.Title == "" {
			updates["title"] = metadata.Title
			title = metadata.Title
		}
		if metadata.Description != "" && videoModel.Description == "" {
			updates["description"] = metadata.Description
			description = metadata.Description
		}
		if metadata.Duration > 0 && videoModel.Duration == 0 {
			updates["duration"] = metadata.Duration
		}
		if len(updates) > 0 {
			h.db.Model(&videoModel).Updates(updates)
			fmt.Printf("Métadonnées extraites et mises à jour pour la vidéo %s\n", videoID)
		}
	}

	// Générer la miniature depuis la vidéo (essayer plusieurs positions pour trouver une bonne frame)
	thumbnailPath, err := h.processor.GenerateThumbnail(videoPath, videoID)
	if err != nil {
		fmt.Printf("Erreur lors de la génération de la miniature depuis la vidéo: %v\n", err)
		// Si la génération depuis la vidéo échoue, créer une miniature textuelle améliorée
		// basée sur le titre, la description et les métadonnées
		thumbnailPath, err = h.generateTextThumbnail(videoID, title, description)
		if err != nil {
			fmt.Printf("Erreur lors de la génération de la miniature textuelle: %v\n", err)
		} else {
			fmt.Printf("Miniature textuelle générée avec succès pour la vidéo %s (titre: %s)\n", videoID, title)
		}
	} else {
		fmt.Printf("Miniature générée depuis la vidéo avec succès pour %s\n", videoID)
	}
	
	// Vérifier l'annulation après génération de la miniature
	select {
	case <-ctx.Done():
		fmt.Printf("Traitement de la vidéo %s annulé après génération de la miniature\n", videoID)
		h.db.Model(&models.Video{}).Where("id = ?", videoID).Update("status", models.VideoStatusCancelled)
		return
	default:
	}
	
	if thumbnailPath != "" {
		h.db.Model(&models.Video{}).Where("id = ?", videoID).Update("thumbnail_path", thumbnailPath)
	}

	// Récupérer la durée
	duration, err := h.processor.GetVideoDuration(videoPath)
	if err == nil {
		h.db.Model(&models.Video{}).Where("id = ?", videoID).Update("duration", duration)
	} else {
		fmt.Printf("Erreur lors de la récupération de la durée: %v\n", err)
	}

	// Compression optimale non-destructive de la vidéo originale
	// Cela réduit la taille du fichier stocké de ~50% avec H.265/HEVC
	fmt.Printf("🎬 Démarrage de la compression optimale pour la vidéo %s...\n", videoID)
	compressedPath, err := h.processor.CompressVideo(videoPath, videoID)
	if err != nil {
		fmt.Printf("⚠️  Erreur lors de la compression optimale: %v (utilisation de la vidéo originale)\n", err)
		compressedPath = videoPath // Utiliser l'originale si la compression échoue
	} else {
		fmt.Printf("✅ Compression optimale terminée: %s (utilisée pour le transcodage)\n", compressedPath)
		// Remplacer le chemin de la vidéo par la version compressée pour le transcodage
		videoPath = compressedPath
	}

	// Qualités à générer (du plus bas au plus haut pour permettre le fallback)
	qualities := []string{"360p", "480p", "720p", "1080p"}
	
	// Générer les formats progressifs (MP4) pour toutes les qualités
	for _, quality := range qualities {
		// Vérifier l'annulation avant chaque transcodage
		select {
		case <-ctx.Done():
			fmt.Printf("Traitement de la vidéo %s annulé pendant le transcodage\n", videoID)
			h.db.Model(&models.Video{}).Where("id = ?", videoID).Update("status", models.VideoStatusCancelled)
			// Nettoyer les formats partiels créés
			h.cleanupPartialFormats(videoID)
			return
		default:
		}

		progressivePath, err := h.processor.TranscodeToProgressive(videoPath, videoID, quality)
		if err != nil {
			fmt.Printf("Erreur lors du transcodage progressif %s: %v\n", quality, err)
			continue
		}
		
		format := models.VideoFormat{
			ID:       uuid.New(),
			VideoID:  videoID,
			Format:   "progressive",
			FilePath: progressivePath,
			Quality:  quality,
		}
		if err := h.db.Create(&format).Error; err != nil {
			fmt.Printf("Erreur lors de la création du format progressif %s: %v\n", quality, err)
		} else {
			fmt.Printf("Format progressif %s créé avec succès\n", quality)
		}
	}

	// Vérifier l'annulation avant le transcodage HLS
	select {
	case <-ctx.Done():
		fmt.Printf("Traitement de la vidéo %s annulé avant le transcodage HLS\n", videoID)
		h.db.Model(&models.Video{}).Where("id = ?", videoID).Update("status", models.VideoStatusCancelled)
		return
	default:
	}

	// Générer le format HLS (pour le streaming adaptatif)
	// On génère seulement 720p en HLS pour commencer (peut être étendu)
	hlsManifest, hlsDir, err := h.processor.TranscodeToHLS(videoPath, videoID, "720p")
	if err != nil {
		fmt.Printf("Erreur lors du transcodage HLS: %v\n", err)
	} else {
		format := models.VideoFormat{
			ID:           uuid.New(),
			VideoID:      videoID,
			Format:       "hls",
			FilePath:     hlsDir,
			ManifestPath: hlsManifest,
			Quality:      "720p",
		}
		if err := h.db.Create(&format).Error; err != nil {
			fmt.Printf("Erreur lors de la création du format HLS: %v\n", err)
		} else {
			fmt.Printf("Format HLS créé avec succès\n")
		}
	}

	// Vérifier l'annulation une dernière fois avant de finaliser
	select {
	case <-ctx.Done():
		fmt.Printf("Traitement de la vidéo %s annulé juste avant la finalisation\n", videoID)
		h.db.Model(&models.Video{}).Where("id = ?", videoID).Update("status", models.VideoStatusCancelled)
		h.cleanupPartialFormats(videoID)
		return
	default:
	}

	// Vérifier qu'au moins un format a été créé
	var formatCount int64
	h.db.Model(&models.VideoFormat{}).Where("video_id = ?", videoID).Count(&formatCount)
	
	// Compter les formats par type pour le log
	var progressiveCount int64
	var hlsCount int64
	h.db.Model(&models.VideoFormat{}).
		Where("video_id = ? AND format = ?", videoID, "progressive").
		Count(&progressiveCount)
	h.db.Model(&models.VideoFormat{}).
		Where("video_id = ? AND format = ?", videoID, "hls").
		Count(&hlsCount)
	
	fmt.Printf("Traitement terminé pour la vidéo %s: %d format(s) total (%d progressifs, %d HLS)\n", 
		videoID, formatCount, progressiveCount, hlsCount)
	
	// Vérification de l'intégrité du fichier original après traitement
	// Récupérer le checksum stocké en base
	var storedVideo models.Video
	if err := h.db.Where("id = ?", videoID).First(&storedVideo).Error; err == nil && storedVideo.Checksum != "" {
		fmt.Printf("🔍 Vérification de l'intégrité du fichier original pour la vidéo %s...\n", videoID)
		currentChecksum, err := h.calculateFileChecksum(videoPath)
		if err != nil {
			fmt.Printf("⚠️  Erreur lors de la vérification d'intégrité: %v\n", err)
		} else if currentChecksum != storedVideo.Checksum {
			fmt.Printf("❌ ERREUR D'INTÉGRITÉ DÉTECTÉE pour la vidéo %s!\n", videoID)
			fmt.Printf("   Checksum attendu: %s\n", storedVideo.Checksum)
			fmt.Printf("   Checksum actuel:  %s\n", currentChecksum)
			fmt.Printf("   Le fichier a été corrompu pendant le traitement!\n")
			h.db.Model(&models.Video{}).Where("id = ?", videoID).Update("status", models.VideoStatusError)
			return
		} else {
			fmt.Printf("✅ Vérification d'intégrité réussie pour la vidéo %s (checksum: %s)\n", videoID, currentChecksum[:16]+"...")
		}
	}
	
	if formatCount == 0 {
		fmt.Printf("❌ AUCUN FORMAT CRÉÉ POUR LA VIDÉO %s - La vidéo ne sera pas lisible\n", videoID)
		h.db.Model(&models.Video{}).Where("id = ?", videoID).Update("status", models.VideoStatusError)
	} else {
		fmt.Printf("✅ Traitement terminé avec succès: %d format(s) créé(s) pour la vidéo %s\n", formatCount, videoID)
		// Mettre à jour le statut à "ready" seulement si au moins un format a été créé
		// Utiliser UpdateColumns pour forcer la mise à jour même si d'autres champs n'ont pas changé
		if err := h.db.Model(&models.Video{}).
			Where("id = ?", videoID).
			UpdateColumns(map[string]interface{}{
				"status": models.VideoStatusReady,
			}).Error; err != nil {
			fmt.Printf("❌ Erreur lors de la mise à jour du statut à 'ready' pour la vidéo %s: %v\n", videoID, err)
		} else {
			fmt.Printf("✅ Statut mis à jour à 'ready' pour la vidéo %s\n", videoID)
		}
	}
}

// calculateFileChecksum calcule le checksum SHA256 d'un fichier
func (h *UploadHandler) calculateFileChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("erreur lors de l'ouverture du fichier: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("erreur lors du calcul du checksum: %w", err)
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// cleanupPartialFormats supprime les formats partiels créés lors d'une annulation
func (h *UploadHandler) cleanupPartialFormats(videoID uuid.UUID) {
	var formats []models.VideoFormat
	h.db.Where("video_id = ?", videoID).Find(&formats)
	
	for _, format := range formats {
		// Supprimer le fichier si c'est un format progressif
		if format.Format == "progressive" && format.FilePath != "" {
			if err := os.Remove(format.FilePath); err != nil {
				fmt.Printf("Erreur lors de la suppression du fichier %s: %v\n", format.FilePath, err)
			}
		}
		// Supprimer le dossier HLS si c'est un format HLS
		if format.Format == "hls" && format.FilePath != "" {
			if err := os.RemoveAll(format.FilePath); err != nil {
				fmt.Printf("Erreur lors de la suppression du dossier HLS %s: %v\n", format.FilePath, err)
			}
		}
		// Supprimer l'entrée en base
		h.db.Delete(&format)
	}
}

func (h *UploadHandler) UploadThumbnail(c *gin.Context) {
	videoIDParam := c.Param("id")
	videoID, err := uuid.Parse(videoIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de vidéo invalide"})
		return
	}

	file, err := c.FormFile("thumbnail")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Fichier image requis"})
		return
	}

	thumbnailName := fmt.Sprintf("%s%s", videoID.String(), filepath.Ext(file.Filename))
	thumbnailPath := filepath.Join(h.cfg.ThumbnailStoragePath, thumbnailName)

	if err := c.SaveUploadedFile(file, thumbnailPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de l'enregistrement"})
		return
	}

	h.db.Model(&models.Video{}).Where("id = ?", videoID).Update("thumbnail_path", thumbnailPath)

	c.JSON(http.StatusOK, gin.H{
		"message":        "Miniature uploadée avec succès",
		"thumbnail_path": thumbnailPath,
	})
}

// generateTextThumbnail génère une miniature textuelle améliorée basée sur le titre, description et métadonnées
func (h *UploadHandler) generateTextThumbnail(videoID uuid.UUID, title string, description string) (string, error) {
	// Dimensions de la miniature (format 16:9)
	width := 1280
	height := 720

	// Créer une image avec un fond dégradé
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	
	// Fond dégradé (rouge vers noir, style Netflix)
	for y := 0; y < height; y++ {
		ratio := float64(y) / float64(height)
		// Dégradé de rouge (#ff0000) vers noir (#000000)
		r := uint8(255 * (1 - ratio*0.7)) // Réduire plus rapidement pour un dégradé plus sombre
		g := uint8(0)
		b := uint8(0)
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}

	// Si pas de titre valide, utiliser un texte par défaut
	if title == "" || title == "null" || title == "Null" || title == "NULL" {
		title = "Vidéo sans titre"
	}

	// Créer le chemin de la miniature
	thumbnailName := fmt.Sprintf("%s.jpg", videoID.String())
	thumbnailPath := filepath.Join(h.cfg.ThumbnailStoragePath, thumbnailName)

	// S'assurer que le dossier existe
	if err := os.MkdirAll(h.cfg.ThumbnailStoragePath, 0755); err != nil {
		return "", fmt.Errorf("erreur lors de la création du dossier: %w", err)
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

	// Note: Pour ajouter du texte sur l'image, on pourrait utiliser une bibliothèque comme
	// github.com/golang/freetype, mais cela nécessiterait une police TTF.
	// Pour l'instant, on génère une image avec un fond dégradé rouge/noir style Netflix.
	// Le texte peut être ajouté plus tard si nécessaire avec une bibliothèque de rendu de texte.

	return thumbnailPath, nil
}

// GenerateThumbnailFromTitle génère une miniature depuis un titre (sans fichier vidéo)
func (h *UploadHandler) GenerateThumbnailFromTitle(c *gin.Context) {
	title := c.Query("title")
	if title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Paramètre 'title' requis"})
		return
	}

	thumbnailPath, err := h.processor.GenerateThumbnailFromTitle(title)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Erreur lors de la génération: %v", err)})
		return
	}

	// Retourner l'URL de la miniature (relative au serveur)
	// Le frontend pourra l'afficher via /api/thumbnails/...
	thumbnailName := filepath.Base(thumbnailPath)
	c.JSON(http.StatusOK, gin.H{
		"thumbnail_path": thumbnailPath,
		"thumbnail_url":  fmt.Sprintf("/api/thumbnails/%s", thumbnailName),
		"thumbnail_name": thumbnailName,
	})
}

// ExtractVideoMetadata extrait les métadonnées d'un fichier vidéo uploadé
func (h *UploadHandler) ExtractVideoMetadata(c *gin.Context) {
	log.Printf("[ExtractVideoMetadata] ✅ Handler appelé pour %s", c.Request.URL.Path)
	log.Printf("[ExtractVideoMetadata] Début de l'extraction des métadonnées")
	
	// Récupérer le fichier
	file, err := c.FormFile("video")
	if err != nil {
		log.Printf("[ExtractVideoMetadata] Erreur lors de la récupération du fichier: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Fichier vidéo requis"})
		return
	}

	log.Printf("[ExtractVideoMetadata] Fichier reçu: %s (taille: %d)", file.Filename, file.Size)

	// Limiter la taille du fichier pour l'extraction de métadonnées (100MB max)
	const maxMetadataFileSize = 100 * 1024 * 1024 // 100MB
	if file.Size > maxMetadataFileSize {
		log.Printf("[ExtractVideoMetadata] Fichier trop volumineux pour l'extraction de métadonnées: %d bytes", file.Size)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Le fichier est trop volumineux pour l'extraction de métadonnées (max: 100MB, votre fichier: %.2fMB). Utilisez la recherche de métadonnées basée sur le nom de fichier.", float64(file.Size)/(1024*1024)),
		})
		return
	}

	// Sauvegarder temporairement le fichier
	tempDir := filepath.Join(h.cfg.VideoStoragePath, "temp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		log.Printf("[ExtractVideoMetadata] Erreur lors de la création du dossier temporaire: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la création du dossier temporaire"})
		return
	}

	tempPath := filepath.Join(tempDir, fmt.Sprintf("metadata_%d_%s", time.Now().Unix(), file.Filename))
	log.Printf("[ExtractVideoMetadata] Sauvegarde temporaire dans: %s", tempPath)
	
	if err := c.SaveUploadedFile(file, tempPath); err != nil {
		log.Printf("[ExtractVideoMetadata] Erreur lors de l'enregistrement temporaire: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de l'enregistrement temporaire"})
		return
	}

	log.Printf("[ExtractVideoMetadata] Fichier sauvegardé, début de l'extraction des métadonnées")

	// Nettoyer le fichier temporaire après extraction
	defer func() {
		if err := os.Remove(tempPath); err != nil {
			log.Printf("[ExtractVideoMetadata] Erreur lors de la suppression du fichier temporaire: %v", err)
		} else {
			log.Printf("[ExtractVideoMetadata] Fichier temporaire supprimé: %s", tempPath)
		}
	}()

	// Extraire les métadonnées
	metadata, err := h.processor.ExtractMetadata(tempPath)
	if err != nil {
		log.Printf("[ExtractVideoMetadata] Erreur lors de l'extraction des métadonnées: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Erreur lors de l'extraction: %v", err)})
		return
	}

	log.Printf("[ExtractVideoMetadata] Métadonnées extraites avec succès: titre=%s, durée=%d, dimensions=%dx%d", 
		metadata.Title, metadata.Duration, metadata.Width, metadata.Height)
	
	c.JSON(http.StatusOK, metadata)
}

// SearchMetadataFromFilename recherche des métadonnées basées sur le nom de fichier (sans upload)
func (h *UploadHandler) SearchMetadataFromFilename(c *gin.Context) {
	var req struct {
		Filename string `json:"filename" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nom de fichier requis"})
		return
	}
	
	log.Printf("[SearchMetadataFromFilename] 🔍 Recherche de métadonnées pour: %s", req.Filename)
	log.Printf("[SearchMetadataFromFilename] ⚠️  CODE MODIFIÉ - Utilisation du scraper web UNIQUEMENT (PAS DE TMDB)")
	
	// Utiliser UNIQUEMENT le scraper web gratuit (sans API) - PAS DE TMDB
	webScraper := metadata.NewWebScraperService()
	log.Printf("[SearchMetadataFromFilename] ⚡ Appel du scraper web...")
	
	results, err := webScraper.SearchMoviesFromFilenameWeb(req.Filename)
	
	if err != nil {
		log.Printf("[SearchMetadataFromFilename] ❌ Erreur scraper web: %v", err)
	} else {
		log.Printf("[SearchMetadataFromFilename] ✅ Scraper web a trouvé %d résultat(s)", len(results))
	}
	
	if len(results) == 0 {
		log.Printf("[SearchMetadataFromFilename] ⚠️  Aucun résultat trouvé pour: %s", req.Filename)
		
		// Proposer un résultat basique basé sur le nom de fichier nettoyé
		titleVariants := metadata.ExtractTitleFromFilename(req.Filename)
		cleanTitle := ""
		if len(titleVariants) > 0 {
			// Prendre la première variante en Title Case
			for _, variant := range titleVariants {
				words := strings.Fields(variant)
				if len(words) > 0 {
					titleCase := ""
					for i, word := range words {
						if i > 0 {
							titleCase += " "
						}
						if len(word) > 0 {
							titleCase += strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
						}
					}
					cleanTitle = titleCase
					break
				}
			}
			if cleanTitle == "" {
				cleanTitle = titleVariants[0]
			}
		} else {
			cleanTitle = strings.TrimSuffix(req.Filename, filepath.Ext(req.Filename))
			cleanTitle = strings.ReplaceAll(cleanTitle, "_", " ")
			cleanTitle = strings.ReplaceAll(cleanTitle, "-", " ")
		}
		
		basicResult := &metadata.MovieMetadata{
			Title:         cleanTitle,
			OriginalTitle: cleanTitle,
			Description:   fmt.Sprintf("Vidéo: %s", cleanTitle),
			Synopsis:      fmt.Sprintf("Vidéo uploadée: %s", cleanTitle),
		}
		
		log.Printf("[SearchMetadataFromFilename] 📝 Proposition d'un résultat basique: %s", cleanTitle)
		c.JSON(http.StatusOK, gin.H{
			"results": []*metadata.MovieMetadata{basicResult},
			"count":   1,
			"message": "Résultat basique (aucun résultat trouvé sur internet)",
		})
		return
	}
	
	log.Printf("[SearchMetadataFromFilename] %d résultat(s) trouvé(s) pour: %s", len(results), req.Filename)
	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"count": len(results),
	})
}

// GetMetadataDetails récupère les détails complets d'un film depuis TMDB
func (h *UploadHandler) GetMetadataDetails(c *gin.Context) {
	var req struct {
		TMDBID int `json:"tmdb_id" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "TMDB ID requis"})
		return
	}
	
	log.Printf("[GetMetadataDetails] Récupération des détails pour TMDB ID: %d", req.TMDBID)
	
	details, err := h.metadataService.GetMovieDetails(req.TMDBID)
	if err != nil {
		log.Printf("[GetMetadataDetails] Erreur: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Erreur: %v", err)})
		return
	}
	
	metadata := h.metadataService.ConvertToMovieMetadata(details)
	c.JSON(http.StatusOK, metadata)
}

// ExtractVideoMetadataFromID extrait les métadonnées d'une vidéo déjà uploadée (via son ID)
func (h *UploadHandler) ExtractVideoMetadataFromID(c *gin.Context) {
	videoIDParam := c.Param("id")
	videoID, err := uuid.Parse(videoIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de vidéo invalide"})
		return
	}

	log.Printf("[ExtractVideoMetadataFromID] Extraction des métadonnées pour la vidéo %s", videoID)

	// Récupérer la vidéo
	var video models.Video
	if err := h.db.Where("id = ?", videoID).First(&video).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vidéo non trouvée"})
		return
	}

	// Vérifier que la vidéo a un fichier (chercher dans les formats ou utiliser l'ID)
	// Construire le chemin du fichier original
	// Le fichier original est généralement stocké avec le nom: {videoID}_{timestamp}.{ext}
	// Chercher dans le répertoire de stockage
	videoPath := ""
	
	// Essayer de trouver le fichier original dans le répertoire de stockage
	// Le fichier peut être nommé avec l'ID de la vidéo
	videoDir := h.cfg.VideoStoragePath
	entries, err := os.ReadDir(videoDir)
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasPrefix(entry.Name(), videoID.String()) {
				videoPath = filepath.Join(videoDir, entry.Name())
				break
			}
		}
	}
	
	// Si pas trouvé, essayer avec le nom de fichier original si disponible
	if videoPath == "" {
		// Chercher dans les formats pour trouver le fichier original
		var formats []models.VideoFormat
		h.db.Where("video_id = ?", videoID).Find(&formats)
		if len(formats) > 0 {
			// Utiliser le premier format pour trouver le répertoire
			formatPath := formats[0].FilePath
			if formatPath != "" {
				// Le fichier original devrait être dans le même répertoire
				formatDir := filepath.Dir(formatPath)
				entries, err := os.ReadDir(formatDir)
				if err == nil {
					for _, entry := range entries {
						if !entry.IsDir() && strings.HasPrefix(entry.Name(), videoID.String()) {
							videoPath = filepath.Join(formatDir, entry.Name())
							break
						}
					}
				}
			}
		}
	}
	
	if videoPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Fichier vidéo original non trouvé"})
		return
	}
	if _, err := os.Stat(videoPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Fichier vidéo non trouvé"})
		return
	}

	log.Printf("[ExtractVideoMetadataFromID] Fichier trouvé: %s", videoPath)

	// Extraire les métadonnées
	metadata, err := h.processor.ExtractMetadata(videoPath)
	if err != nil {
		log.Printf("[ExtractVideoMetadataFromID] Erreur lors de l'extraction des métadonnées: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Erreur lors de l'extraction: %v", err)})
		return
	}

	log.Printf("[ExtractVideoMetadataFromID] Métadonnées extraites avec succès: titre=%s, durée=%d, dimensions=%dx%d", 
		metadata.Title, metadata.Duration, metadata.Width, metadata.Height)

	// Mettre à jour la vidéo avec les métadonnées extraites
	updates := make(map[string]interface{})
	if metadata.Title != "" && video.Title == "" {
		updates["title"] = metadata.Title
	}
	if metadata.Description != "" && video.Description == "" {
		updates["description"] = metadata.Description
	}
	// Note: Le champ Author n'existe pas dans le modèle Video actuel
	// On peut l'ajouter à la description si nécessaire
	if metadata.Author != "" && !strings.Contains(video.Description, metadata.Author) {
		if video.Description == "" {
			updates["description"] = fmt.Sprintf("Par %s", metadata.Author)
		} else {
			updates["description"] = fmt.Sprintf("Par %s\n\n%s", metadata.Author, video.Description)
		}
	}
	if metadata.Duration > 0 && video.Duration == 0 {
		updates["duration"] = metadata.Duration
	}
	
	// Appliquer les mises à jour
	if len(updates) > 0 {
		if err := h.db.Model(&video).Updates(updates).Error; err != nil {
			log.Printf("[ExtractVideoMetadataFromID] Erreur lors de la mise à jour de la vidéo: %v", err)
			// Ne pas échouer si la mise à jour échoue, on retourne quand même les métadonnées
		} else {
			log.Printf("[ExtractVideoMetadataFromID] Vidéo mise à jour avec succès")
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"metadata": metadata,
		"video": video,
		"message": "Métadonnées extraites et vidéo mise à jour",
	})
}

