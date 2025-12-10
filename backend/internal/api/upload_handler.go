package api

import (
	"context"
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"streammake/internal/config"
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
	// Suivi des traitements en cours avec context pour annulation
	processingVideos map[uuid.UUID]context.CancelFunc
	processingMutex  sync.RWMutex
}

func NewUploadHandler(db *gorm.DB, cfg *config.Config) *UploadHandler {
	return &UploadHandler{
		db:               db,
		cfg:              cfg,
		processor:        video.NewProcessor(cfg.VideoStoragePath, cfg.ThumbnailStoragePath),
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

	// Créer l'entrée en base
	videoModel := models.Video{
		ID:            videoID,
		Title:         title,
		Description:   description,
		UserID:        userUUID,
		Status:        models.VideoStatusProcessing,
		CategoryID:    categoryID,
		SubCategoryID: subCategoryID,
		IsPublic:      isPublic,
		Checksum:      checksum,
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
	
	if formatCount == 0 {
		fmt.Printf("AUCUN FORMAT CRÉÉ POUR LA VIDÉO %s - La vidéo ne sera pas lisible\n", videoID)
		h.db.Model(&models.Video{}).Where("id = ?", videoID).Update("status", models.VideoStatusError)
	} else {
		fmt.Printf("Traitement terminé: %d format(s) créé(s) pour la vidéo %s\n", formatCount, videoID)
		// Mettre à jour le statut seulement si au moins un format a été créé
		h.db.Model(&models.Video{}).Where("id = ?", videoID).Update("status", models.VideoStatusReady)
	}
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

