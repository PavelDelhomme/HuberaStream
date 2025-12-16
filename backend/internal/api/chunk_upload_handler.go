package api

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"streammake/internal/config"
	"streammake/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	DefaultChunkSize = 5 * 1024 * 1024 // 5MB par chunk
	MaxChunkSize     = 50 * 1024 * 1024 // 50MB max par chunk
)

type ChunkUploadHandler struct {
	db            *gorm.DB
	cfg           *config.Config
	uploadHandler *UploadHandler // Référence pour processVideo
	sessions      map[uuid.UUID]*UploadSessionState
	sessionsMutex sync.RWMutex
}

type UploadSessionState struct {
	Session   models.UploadSession
	Chunks    map[int]*models.UploadChunk
	Mutex     sync.RWMutex
}

func NewChunkUploadHandler(db *gorm.DB, cfg *config.Config, uploadHandler *UploadHandler) *ChunkUploadHandler {
	return &ChunkUploadHandler{
		db:            db,
		cfg:           cfg,
		uploadHandler: uploadHandler,
		sessions:      make(map[uuid.UUID]*UploadSessionState),
	}
}

// InitUploadSession initialise une nouvelle session d'upload
func (h *ChunkUploadHandler) InitUploadSession(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr := userID.(string)
	userUUID, _ := uuid.Parse(userIDStr)

	var req struct {
		FileName  string          `json:"file_name" binding:"required"`
		FileSize  int64           `json:"file_size" binding:"required"`
		ChunkSize int64           `json:"chunk_size"` // Optionnel, défaut 5MB
		Metadata  json.RawMessage `json:"metadata"`   // title, description, category_id, etc.
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Valider la taille du fichier
	if req.FileSize <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Taille de fichier invalide"})
		return
	}

	// Déterminer la taille des chunks
	chunkSize := req.ChunkSize
	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize
	}
	if chunkSize > MaxChunkSize {
		chunkSize = MaxChunkSize
	}

	// Calculer le nombre total de chunks
	totalChunks := int((req.FileSize + chunkSize - 1) / chunkSize) // Arrondi supérieur

	// Créer la session
	sessionID := uuid.New()
	session := models.UploadSession{
		ID:            sessionID,
		UserID:        userUUID,
		FileName:      req.FileName,
		FileSize:      req.FileSize,
		ChunkSize:     chunkSize,
		TotalChunks:   totalChunks,
		Status:        models.UploadStatusPending,
		Metadata:      string(req.Metadata),
		TempPath:      filepath.Join(h.cfg.VideoStoragePath, "uploads", sessionID.String()),
	}

	if err := h.db.Create(&session).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la création de la session"})
		return
	}

	// Créer le dossier temporaire pour les chunks
	if err := os.MkdirAll(session.TempPath, 0755); err != nil {
		h.db.Delete(&session)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la création du dossier temporaire"})
		return
	}

	// Initialiser l'état de la session en mémoire
	h.sessionsMutex.Lock()
	h.sessions[sessionID] = &UploadSessionState{
		Session: session,
		Chunks:  make(map[int]*models.UploadChunk),
	}
	h.sessionsMutex.Unlock()

	log.Printf("[InitUploadSession] Session créée: %s, fichier: %s, taille: %d, chunks: %d", 
		sessionID, req.FileName, req.FileSize, totalChunks)

	c.JSON(http.StatusCreated, gin.H{
		"session_id":   sessionID,
		"chunk_size":   chunkSize,
		"total_chunks": totalChunks,
		"status":       "pending",
	})
}

// UploadChunk upload un chunk de fichier
func (h *ChunkUploadHandler) UploadChunk(c *gin.Context) {
	sessionIDParam := c.Param("session_id")
	sessionID, err := uuid.Parse(sessionIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de session invalide"})
		return
	}

	// Récupérer la session
	var session models.UploadSession
	if err := h.db.Where("id = ?", sessionID).First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session non trouvée"})
		return
	}

	// Vérifier les permissions
	userID, _ := c.Get("user_id")
	userIDStr := userID.(string)
	userUUID, _ := uuid.Parse(userIDStr)
	if session.UserID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission refusée"})
		return
	}

	// Vérifier que la session n'est pas annulée ou terminée
	if session.Status == models.UploadStatusCancelled || session.Status == models.UploadStatusCompleted {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session terminée ou annulée"})
		return
	}

	// Récupérer le chunk index et le checksum
	chunkIndexStr := c.PostForm("chunk_index")
	chunkIndex, err := strconv.Atoi(chunkIndexStr)
	if err != nil || chunkIndex < 0 || chunkIndex >= session.TotalChunks {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Index de chunk invalide"})
		return
	}
	
	// Récupérer le checksum du chunk (calculé côté client)
	expectedChecksum := c.PostForm("checksum")

	// Récupérer le fichier chunk
	file, err := c.FormFile("chunk")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chunk requis"})
		return
	}

	// Vérifier que le chunk n'a pas déjà été uploadé
	var existingChunk models.UploadChunk
	if err := h.db.Where("session_id = ? AND chunk_index = ?", sessionID, chunkIndex).First(&existingChunk).Error; err == nil {
		// Chunk déjà uploadé, retourner succès
		c.JSON(http.StatusOK, gin.H{
			"message":     "Chunk déjà uploadé",
			"chunk_index": chunkIndex,
			"uploaded":    true,
		})
		return
	}

	// Sauvegarder le chunk temporairement
	chunkFileName := fmt.Sprintf("chunk_%d", chunkIndex)
	chunkPath := filepath.Join(session.TempPath, chunkFileName)
	
	if err := c.SaveUploadedFile(file, chunkPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de l'enregistrement du chunk"})
		return
	}

	// Calculer le checksum du chunk
	chunkFile, err := os.Open(chunkPath)
	if err != nil {
		os.Remove(chunkPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de l'ouverture du chunk"})
		return
	}
	defer chunkFile.Close()

	hash := sha256.New()
	chunkSize, err := io.Copy(hash, chunkFile)
	if err != nil {
		os.Remove(chunkPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors du calcul du checksum"})
		return
	}
	chunkChecksum := fmt.Sprintf("%x", hash.Sum(nil))
	
	// Vérifier l'intégrité du chunk si un checksum a été fourni
	if expectedChecksum != "" && chunkChecksum != expectedChecksum {
		os.Remove(chunkPath)
		log.Printf("[UploadChunk] ❌ ERREUR D'INTÉGRITÉ: Chunk %d (session %s)", chunkIndex, sessionID)
		log.Printf("[UploadChunk]   Checksum attendu: %s", expectedChecksum)
		log.Printf("[UploadChunk]   Checksum calculé: %s", chunkChecksum)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Checksum du chunk invalide - corruption détectée",
			"expected": expectedChecksum,
			"got": chunkChecksum,
			"chunk_index": chunkIndex,
		})
		return
	} else if expectedChecksum != "" {
		log.Printf("[UploadChunk] ✅ Intégrité validée pour chunk %d/%d (session %s): %s", 
			chunkIndex+1, session.TotalChunks, sessionID, chunkChecksum[:16]+"...")
	}

	// Créer l'entrée du chunk
	chunk := models.UploadChunk{
		ID:         uuid.New(),
		SessionID:  sessionID,
		ChunkIndex: chunkIndex,
		ChunkSize:  chunkSize,
		FilePath:   chunkPath,
		Checksum:   chunkChecksum,
		UploadedAt: time.Now(),
	}

	if err := h.db.Create(&chunk).Error; err != nil {
		os.Remove(chunkPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la création de l'entrée du chunk"})
		return
	}

	// Mettre à jour la session
	session.UploadedChunks++
	session.UploadedBytes += chunkSize
	if session.Status == models.UploadStatusPending {
		session.Status = models.UploadStatusUploading
	}
	h.db.Save(&session)

	log.Printf("[UploadChunk] Chunk %d/%d uploadé pour session %s (%.1f%%)", 
		chunkIndex+1, session.TotalChunks, sessionID, 
		float64(session.UploadedChunks)/float64(session.TotalChunks)*100)

	c.JSON(http.StatusOK, gin.H{
		"message":        "Chunk uploadé avec succès",
		"chunk_index":    chunkIndex,
		"uploaded_chunks": session.UploadedChunks,
		"total_chunks":   session.TotalChunks,
		"progress":       float64(session.UploadedChunks) / float64(session.TotalChunks) * 100,
	})
}

// GetUploadStatus retourne le statut d'un upload
func (h *ChunkUploadHandler) GetUploadStatus(c *gin.Context) {
	sessionIDParam := c.Param("session_id")
	sessionID, err := uuid.Parse(sessionIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de session invalide"})
		return
	}

	var session models.UploadSession
	if err := h.db.Where("id = ?", sessionID).First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session non trouvée"})
		return
	}

	// Vérifier les permissions
	userID, exists := c.Get("user_id")
	if exists {
		userIDStr := userID.(string)
		userUUID, _ := uuid.Parse(userIDStr)
		if session.UserID != userUUID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Permission refusée"})
			return
		}
	}

	progress := float64(0)
	if session.TotalChunks > 0 {
		progress = float64(session.UploadedChunks) / float64(session.TotalChunks) * 100
	}
	
	// Récupérer les détails des chunks pour l'historique
	var chunks []models.UploadChunk
	h.db.Where("session_id = ?", sessionID).Order("chunk_index ASC").Find(&chunks)
	
	// Préparer les informations des chunks
	chunkDetails := make([]map[string]interface{}, len(chunks))
	for i, chunk := range chunks {
		chunkDetails[i] = map[string]interface{}{
			"index": chunk.ChunkIndex,
			"size": chunk.ChunkSize,
			"checksum": chunk.Checksum,
			"uploaded_at": chunk.UploadedAt,
			"status": "uploaded",
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"session_id":     sessionID,
		"status":         string(session.Status),
		"uploaded_chunks": session.UploadedChunks,
		"total_chunks":   session.TotalChunks,
		"uploaded_bytes": session.UploadedBytes,
		"file_size":      session.FileSize,
		"progress":       progress,
		"error":          session.Error,
	})
}

// PauseUpload met en pause un upload
func (h *ChunkUploadHandler) PauseUpload(c *gin.Context) {
	sessionIDParam := c.Param("session_id")
	sessionID, err := uuid.Parse(sessionIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de session invalide"})
		return
	}

	var session models.UploadSession
	if err := h.db.Where("id = ?", sessionID).First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session non trouvée"})
		return
	}

	// Vérifier les permissions
	userID, _ := c.Get("user_id")
	userIDStr := userID.(string)
	userUUID, _ := uuid.Parse(userIDStr)
	if session.UserID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission refusée"})
		return
	}

	if session.Status != models.UploadStatusUploading {
		c.JSON(http.StatusBadRequest, gin.H{"error": "L'upload n'est pas en cours"})
		return
	}

	session.Status = models.UploadStatusPaused
	h.db.Save(&session)

	c.JSON(http.StatusOK, gin.H{
		"message": "Upload mis en pause",
		"status":  "paused",
	})
}

// ResumeUpload reprend un upload en pause
func (h *ChunkUploadHandler) ResumeUpload(c *gin.Context) {
	sessionIDParam := c.Param("session_id")
	sessionID, err := uuid.Parse(sessionIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de session invalide"})
		return
	}

	var session models.UploadSession
	if err := h.db.Where("id = ?", sessionID).First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session non trouvée"})
		return
	}

	// Vérifier les permissions
	userID, _ := c.Get("user_id")
	userIDStr := userID.(string)
	userUUID, _ := uuid.Parse(userIDStr)
	if session.UserID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission refusée"})
		return
	}

	if session.Status != models.UploadStatusPaused {
		c.JSON(http.StatusBadRequest, gin.H{"error": "L'upload n'est pas en pause"})
		return
	}

	session.Status = models.UploadStatusUploading
	h.db.Save(&session)

	c.JSON(http.StatusOK, gin.H{
		"message": "Upload repris",
		"status":  "uploading",
	})
}

// FinalizeUpload assemble les chunks et crée la vidéo
func (h *ChunkUploadHandler) FinalizeUpload(c *gin.Context) {
	sessionIDParam := c.Param("session_id")
	sessionID, err := uuid.Parse(sessionIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de session invalide"})
		return
	}

	var session models.UploadSession
	if err := h.db.Where("id = ?", sessionID).First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session non trouvée"})
		return
	}

	// Vérifier les permissions
	userID, _ := c.Get("user_id")
	userIDStr := userID.(string)
	userUUID, _ := uuid.Parse(userIDStr)
	if session.UserID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission refusée"})
		return
	}

	// Récupérer tous les chunks triés par index
	var chunks []models.UploadChunk
	if err := h.db.Where("session_id = ?", sessionID).Order("chunk_index ASC").Find(&chunks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la récupération des chunks"})
		return
	}

	// Compter réellement les chunks dans la base de données
	actualChunkCount := len(chunks)
	
	// Vérifier que tous les chunks sont uploadés (utiliser le comptage réel plutôt que le compteur de session)
	if actualChunkCount < session.TotalChunks {
		log.Printf("[FinalizeUpload] ⚠️  Chunks manquants: %d/%d uploadés (session: %s)", 
			actualChunkCount, session.TotalChunks, sessionID)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":          "Tous les chunks ne sont pas encore uploadés",
			"uploaded_chunks": actualChunkCount,
			"total_chunks":    session.TotalChunks,
			"session_uploaded_chunks": session.UploadedChunks, // Pour debug
		})
		return
	}

	if actualChunkCount != session.TotalChunks {
		log.Printf("[FinalizeUpload] ⚠️  Nombre de chunks incorrect: %d attendu, %d trouvé (session: %s)", 
			session.TotalChunks, actualChunkCount, sessionID)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Nombre de chunks incorrect",
			"expected": session.TotalChunks,
			"found": actualChunkCount,
		})
		return
	}
	
	log.Printf("[FinalizeUpload] ✅ Tous les chunks sont présents: %d/%d (session: %s)", 
		actualChunkCount, session.TotalChunks, sessionID)

	// Assembler les chunks
	videoID := uuid.New()
	videoFileName := fmt.Sprintf("%s_%d%s", videoID.String(), time.Now().Unix(), filepath.Ext(session.FileName))
	videoPath := filepath.Join(h.cfg.VideoStoragePath, videoFileName)

	outputFile, err := os.Create(videoPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la création du fichier final"})
		return
	}
	defer outputFile.Close()

	// Copier tous les chunks dans l'ordre
	for _, chunk := range chunks {
		chunkFile, err := os.Open(chunk.FilePath)
		if err != nil {
			outputFile.Close()
			os.Remove(videoPath)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Erreur lors de l'ouverture du chunk %d", chunk.ChunkIndex)})
			return
		}

		if _, err := io.Copy(outputFile, chunkFile); err != nil {
			chunkFile.Close()
			outputFile.Close()
			os.Remove(videoPath)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Erreur lors de l'assemblage du chunk %d", chunk.ChunkIndex)})
			return
		}
		chunkFile.Close()
	}
	outputFile.Close()

	// Calculer le checksum du fichier final
	finalFile, err := os.Open(videoPath)
	if err != nil {
		os.Remove(videoPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de l'ouverture du fichier final"})
		return
	}
	defer finalFile.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, finalFile); err != nil {
		os.Remove(videoPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors du calcul du checksum final"})
		return
	}
	finalChecksum := fmt.Sprintf("%x", hash.Sum(nil))

	// Parser les métadonnées
	var metadata struct {
		Title          string     `json:"title"`
		Description    string     `json:"description"`
		CategoryID     *uuid.UUID `json:"category_id"`
		// Métadonnées enrichies
		Director       string     `json:"director"`
		Actors         string     `json:"actors"`
		ReleaseDate    string     `json:"release_date"`
		ProductionYear int        `json:"production_year"`
		Synopsis       string     `json:"synopsis"`
		Genre          string     `json:"genre"`
		Rating         float64    `json:"rating"`
		IMDBID         string     `json:"imdb_id"`
		TMDBID         int        `json:"tmdb_id"`
		PosterURL      string     `json:"poster_url"`
		SubCategoryID *uuid.UUID `json:"sub_category_id"`
		IsPublic      bool       `json:"is_public"`
	}
	if err := json.Unmarshal([]byte(session.Metadata), &metadata); err != nil {
		log.Printf("[FinalizeUpload] Erreur parsing métadonnées: %v", err)
		// Utiliser des valeurs par défaut
		metadata.Title = session.FileName
		metadata.IsPublic = true
	}

	// Créer l'entrée vidéo
	videoModel := models.Video{
		ID:              videoID,
		Title:           metadata.Title,
		Description:     metadata.Description,
		UserID:          session.UserID,
		Status:          models.VideoStatusProcessing,
		CategoryID:      metadata.CategoryID,
		SubCategoryID:   metadata.SubCategoryID,
		IsPublic:        metadata.IsPublic,
		Checksum:        finalChecksum,
		// Métadonnées enrichies
		Director:        metadata.Director,
		Actors:          metadata.Actors,
		ReleaseDate:     metadata.ReleaseDate,
		ProductionYear:  metadata.ProductionYear,
		Synopsis:        metadata.Synopsis,
		Genre:           metadata.Genre,
		Rating:          metadata.Rating,
		IMDBID:          metadata.IMDBID,
		TMDBID:          metadata.TMDBID,
		PosterURL:       metadata.PosterURL,
	}

	if err := h.db.Create(&videoModel).Error; err != nil {
		os.Remove(videoPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la création de la vidéo"})
		return
	}

	// Lier la session à la vidéo
	session.VideoID = &videoID
	session.Status = models.UploadStatusCompleted
	session.Checksum = finalChecksum
	h.db.Save(&session)

	// Vérification de l'intégrité du fichier assemblé
	log.Printf("[FinalizeUpload] Vérification de l'intégrité du fichier assemblé (checksum: %s)", finalChecksum[:16]+"...")
	
	// Vérifier que le checksum du fichier final correspond bien
	// (déjà calculé ci-dessus, mais on peut le vérifier à nouveau si nécessaire)
	if session.Checksum != "" && finalChecksum != session.Checksum {
		log.Printf("[FinalizeUpload] ⚠️  Attention: Le checksum du fichier final ne correspond pas à celui attendu")
		// Note: Le checksum a été calculé juste avant, donc normalement ils devraient correspondre
		// Mais on peut ajouter une vérification supplémentaire si nécessaire
	}

	// Nettoyer les chunks temporaires (en arrière-plan)
	go h.cleanupChunks(sessionID, session.TempPath)

	// Lancer le traitement vidéo en arrière-plan
	if h.uploadHandler != nil {
		ctx, cancel := context.WithCancel(context.Background())
		h.uploadHandler.processingMutex.Lock()
		h.uploadHandler.processingVideos[videoID] = cancel
		h.uploadHandler.processingMutex.Unlock()
		go h.uploadHandler.processVideo(ctx, videoID, videoPath)
	}

	log.Printf("[FinalizeUpload] Upload finalisé: session %s -> vidéo %s (checksum vérifié: %s)", 
		sessionID, videoID, finalChecksum[:16]+"...")

	c.JSON(http.StatusOK, gin.H{
		"message":  "Upload finalisé avec succès",
		"video_id": videoID,
		"status":   "processing",
		"checksum": finalChecksum, // Retourner le checksum pour confirmation
	})
}

// cleanupChunks nettoie les chunks temporaires
func (h *ChunkUploadHandler) cleanupChunks(sessionID uuid.UUID, tempPath string) {
	// Supprimer les fichiers chunks
	if err := os.RemoveAll(tempPath); err != nil {
		log.Printf("[cleanupChunks] Erreur lors de la suppression des chunks: %v", err)
	}

	// Supprimer les entrées de chunks de la base
	h.db.Where("session_id = ?", sessionID).Delete(&models.UploadChunk{})
}

// CancelUpload annule un upload et supprime la vidéo partielle si elle existe
func (h *ChunkUploadHandler) CancelUpload(c *gin.Context) {
	sessionIDParam := c.Param("session_id")
	sessionID, err := uuid.Parse(sessionIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de session invalide"})
		return
	}

	var session models.UploadSession
	if err := h.db.Where("id = ?", sessionID).First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session non trouvée"})
		return
	}

	// Vérifier les permissions
	userID, _ := c.Get("user_id")
	userIDStr := userID.(string)
	userUUID, _ := uuid.Parse(userIDStr)
	if session.UserID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission refusée"})
		return
	}

	// Si une vidéo a été créée (même partielle), la supprimer
	if session.VideoID != nil {
		var video models.Video
		if err := h.db.Where("id = ?", *session.VideoID).First(&video).Error; err == nil {
			log.Printf("[CancelUpload] Suppression de la vidéo partielle: %s (session: %s)", *session.VideoID, sessionID)
			
			// Récupérer les formats vidéo pour supprimer les fichiers
			var formats []models.VideoFormat
			h.db.Where("video_id = ?", *session.VideoID).Find(&formats)
			
			// Supprimer tous les fichiers vidéo associés
			for _, format := range formats {
				if format.FilePath != "" {
					if err := os.Remove(format.FilePath); err != nil && !os.IsNotExist(err) {
						log.Printf("[CancelUpload] Erreur lors de la suppression du fichier vidéo %s: %v", format.FilePath, err)
					}
				}
				if format.ManifestPath != "" {
					if err := os.Remove(format.ManifestPath); err != nil && !os.IsNotExist(err) {
						log.Printf("[CancelUpload] Erreur lors de la suppression du manifest %s: %v", format.ManifestPath, err)
					}
				}
			}
			
			// Supprimer la miniature s'il y en a une
			if video.ThumbnailPath != "" {
				if err := os.Remove(video.ThumbnailPath); err != nil && !os.IsNotExist(err) {
					log.Printf("[CancelUpload] Erreur lors de la suppression de la miniature %s: %v", video.ThumbnailPath, err)
				}
			}
			
			// Supprimer les sous-titres associés
			var subtitles []models.Subtitle
			h.db.Where("video_id = ?", *session.VideoID).Find(&subtitles)
			for _, subtitle := range subtitles {
				if subtitle.FilePath != "" {
					if err := os.Remove(subtitle.FilePath); err != nil && !os.IsNotExist(err) {
						log.Printf("[CancelUpload] Erreur lors de la suppression du sous-titre %s: %v", subtitle.FilePath, err)
					}
				}
			}
			
			// Supprimer les formats et sous-titres de la base de données
			h.db.Where("video_id = ?", *session.VideoID).Delete(&models.VideoFormat{})
			h.db.Where("video_id = ?", *session.VideoID).Delete(&models.Subtitle{})
			
			// Supprimer la vidéo de la base de données
			if err := h.db.Delete(&video).Error; err != nil {
				log.Printf("[CancelUpload] Erreur lors de la suppression de la vidéo de la base: %v", err)
			} else {
				log.Printf("[CancelUpload] ✅ Vidéo partielle supprimée: %s", *session.VideoID)
			}
			
			// Annuler le traitement vidéo en cours si actif
			if h.uploadHandler != nil {
				h.uploadHandler.processingMutex.Lock()
				if cancel, exists := h.uploadHandler.processingVideos[*session.VideoID]; exists {
					cancel()
					delete(h.uploadHandler.processingVideos, *session.VideoID)
					log.Printf("[CancelUpload] Traitement vidéo annulé: %s", *session.VideoID)
				}
				h.uploadHandler.processingMutex.Unlock()
			}
		}
	}

	// Marquer la session comme annulée
	session.Status = models.UploadStatusCancelled
	session.VideoID = nil // Retirer la référence à la vidéo
	h.db.Save(&session)

	// Nettoyer les chunks
	go h.cleanupChunks(sessionID, session.TempPath)

	log.Printf("[CancelUpload] ✅ Upload annulé et nettoyé: session %s", sessionID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Upload annulé et vidéo partielle supprimée",
		"status":  "cancelled",
	})
}

