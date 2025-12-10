package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"streammake/internal/config"
	"streammake/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type VideoHandler struct {
	db            *gorm.DB
	cfg           *config.Config
	uploadHandler *UploadHandler // Référence pour accéder aux traitements en cours
}

func NewVideoHandler(db *gorm.DB, cfg *config.Config, uploadHandler *UploadHandler) *VideoHandler {
	return &VideoHandler{
		db:            db,
		cfg:           cfg,
		uploadHandler: uploadHandler,
	}
}

type CreateVideoRequest struct {
	Title         string     `json:"title" binding:"required"`
	Description   string     `json:"description"`
	Tags          []string   `json:"tags"`
	CategoryID    *uuid.UUID `json:"category_id"`
	SubCategoryID *uuid.UUID `json:"sub_category_id"`
	IsPublic      bool       `json:"is_public"`
}

func (h *VideoHandler) CreateVideo(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr := userID.(string)
	userUUID, _ := uuid.Parse(userIDStr)

	var req CreateVideoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Valider le titre (ne doit pas être "null" ou vide)
	title := strings.TrimSpace(req.Title)
	if title == "" || title == "null" || title == "Null" || title == "NULL" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Le titre de la vidéo ne peut pas être vide ou 'null'"})
		return
	}

	video := models.Video{
		ID:            uuid.New(),
		Title:         title,
		Description:   req.Description,
		UserID:        userUUID,
		Status:        models.VideoStatusPending,
		CategoryID:    req.CategoryID,
		SubCategoryID: req.SubCategoryID,
		IsPublic:      req.IsPublic,
	}

	if err := h.db.Create(&video).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la création de la vidéo"})
		return
	}

	c.JSON(http.StatusCreated, video)
}

func (h *VideoHandler) ListVideos(c *gin.Context) {
	var videos []models.Video
	query := h.db.Preload("User").Preload("Formats")
	
	// Si pas de paramètre status, afficher seulement les vidéos prêtes (pour les non-admins)
	// Les admins peuvent voir toutes les vidéos
	userID, exists := c.Get("user_id")
	isAdmin := false
	if exists {
		var user models.User
		if err := h.db.Where("id = ?", userID).First(&user).Error; err == nil {
			isAdmin = user.Role == models.RoleAdmin
		}
	}
	
	// Filtre par statut si fourni
	statusFilter := c.Query("status")
	if statusFilter == "all" && isAdmin {
		// Les admins peuvent voir toutes les vidéos si status=all - ne pas appliquer de filtre
		// (laisser query tel quel pour récupérer toutes les vidéos)
	} else if statusFilter != "" && statusFilter != "all" {
		// Filtrer par statut spécifique
		query = query.Where("status = ?", statusFilter)
	} else if !isAdmin {
		// Les non-admins voient seulement les vidéos prêtes
		query = query.Where("status = ?", models.VideoStatusReady)
	} else if isAdmin && statusFilter == "" {
		// Par défaut, les admins voient toutes les vidéos (pas seulement les prêtes)
		// Ne pas appliquer de filtre
	}

	// Pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	// Filtre par catégorie
	if categoryID := c.Query("category_id"); categoryID != "" {
		catUUID, err := uuid.Parse(categoryID)
		if err == nil {
			query = query.Where("category_id = ?", catUUID)
		}
	}

	// Filtre par utilisateur (pour afficher les vidéos d'un utilisateur spécifique)
	if userIDParam := c.Query("user_id"); userIDParam != "" {
		userUUID, err := uuid.Parse(userIDParam)
		if err == nil {
			query = query.Where("user_id = ?", userUUID)
		}
	}

	// Recherche
	if search := c.Query("search"); search != "" {
		query = query.Where("title ILIKE ? OR description ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// Trier par vues pour les tendances
	sortBy := c.DefaultQuery("sort", "created_at")
	if sortBy == "views" {
		query = query.Order("views DESC")
	} else {
		query = query.Order("created_at DESC")
	}

	var total int64
	query.Model(&models.Video{}).Count(&total)

	if err := query.Offset(offset).Limit(limit).Find(&videos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la récupération des vidéos"})
		return
	}

	// Filtrer les vidéos avec titre "null" ou vide SEULEMENT pour les non-admins
	// Les admins doivent voir toutes les vidéos pour pouvoir les gérer
	var validVideos []models.Video
	if isAdmin && statusFilter == "all" {
		// Pour les admins avec status=all, afficher toutes les vidéos
		validVideos = videos
	} else {
		// Pour les non-admins ou sans status=all, filtrer les vidéos avec titre "null" ou vide
		for _, video := range videos {
			if video.Title != "" && video.Title != "null" && video.Title != "Null" && video.Title != "NULL" {
				validVideos = append(validVideos, video)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"videos": validVideos,
		"total":  int64(len(validVideos)),
		"page":   page,
		"limit":  limit,
	})
}

// GetContinueWatching récupère les vidéos en cours de visionnage
// Peut être appelé depuis les routes publiques ou protégées
func (h *VideoHandler) GetContinueWatching(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		// Pas d'utilisateur connecté, retourner un tableau vide
		c.JSON(http.StatusOK, gin.H{"videos": []interface{}{}})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		c.JSON(http.StatusOK, gin.H{"videos": []interface{}{}})
		return
	}

	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"videos": []interface{}{}})
		return
	}

	var history []models.WatchHistory
	// Récupérer les vidéos avec position > 0 et < 90% de la durée
	if err := h.db.Preload("Video").Preload("Video.User").
		Where("user_id = ? AND position > 0", userUUID).
		Order("watched_at DESC").
		Limit(20).
		Find(&history).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la récupération"})
		return
	}

	// Filtrer pour ne garder que les vidéos non terminées
	var continueWatching []models.WatchHistory
	for _, h := range history {
		if h.Video.Duration > 0 && float64(h.Position) < float64(h.Video.Duration)*0.9 {
			continueWatching = append(continueWatching, h)
		}
	}

	c.JSON(http.StatusOK, gin.H{"videos": continueWatching})
}

func (h *VideoHandler) GetVideo(c *gin.Context) {
	id := c.Param("id")
	fmt.Printf("[DEBUG GetVideo] Handler appelé avec id=%s\n", id)
	
	// Si c'est "continue-watching", rediriger vers le bon handler
	if id == "continue-watching" {
		h.GetContinueWatching(c)
		return
	}
	
	videoID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de vidéo invalide"})
		return
	}

	var video models.Video
	fmt.Printf("[DEBUG GetVideo] Récupération vidéo %s\n", videoID)
	
	// Essayer d'abord de récupérer la vidéo (sans soft-deleted)
	if err := h.db.Where("id = ?", videoID).First(&video).Error; err != nil {
		// Si la vidéo n'est pas trouvée, essayer avec Unscoped pour voir si elle est soft-deleted
		if err := h.db.Unscoped().Where("id = ?", videoID).First(&video).Error; err != nil {
			fmt.Printf("[DEBUG GetVideo] Erreur lors de la récupération de la vidéo %s: %v\n", videoID, err)
			c.JSON(http.StatusNotFound, gin.H{"error": "Vidéo non trouvée"})
			return
		}
		// Si la vidéo est soft-deleted, vérifier si elle est en traitement
		// Si elle est en traitement, permettre quand même l'accès
		if video.DeletedAt.Valid && video.Status != models.VideoStatusProcessing {
			fmt.Printf("[DEBUG GetVideo] Vidéo %s est soft-deleted et n'est pas en traitement\n", videoID)
			c.JSON(http.StatusNotFound, gin.H{"error": "Vidéo non trouvée"})
			return
		}
		// Si elle est en traitement même si soft-deleted, continuer
		fmt.Printf("[DEBUG GetVideo] Vidéo %s est soft-deleted mais en traitement, accès autorisé\n", videoID)
	}
	
	fmt.Printf("[DEBUG GetVideo] Vidéo trouvée en base: ID=%s, Title=%s, Status=%s, IsPublic=%v, DeletedAt=%v\n", 
		video.ID, video.Title, video.Status, video.IsPublic, video.DeletedAt.Valid)
	
	// Charger les relations manuellement après avoir récupéré la vidéo
	h.db.Model(&video).Association("Formats").Find(&video.Formats)
	h.db.Model(&video).Association("User").Find(&video.User)
	h.db.Model(&video).Association("Subtitles").Find(&video.Subtitles)
	h.db.Model(&video).Association("Tags").Find(&video.Tags)
	
	fmt.Printf("[DEBUG GetVideo] Relations chargées: Formats=%d\n", len(video.Formats))

	// Vérifier les permissions d'accès
	// Récupérer user_id depuis le contexte (peut être vide si route publique sans token)
	userID, exists := c.Get("user_id")
	isAdmin := false
	isOwner := false
	
	if exists {
		userIDStr, ok := userID.(string)
		if ok && userIDStr != "" { // Vérifier que userIDStr n'est pas vide
			userUUID, err := uuid.Parse(userIDStr)
			if err == nil {
				// Vérifier si c'est le propriétaire
				if video.UserID == userUUID {
					isOwner = true
				}
				
				// Vérifier si c'est un admin
				var user models.User
				if err := h.db.Where("id = ?", userUUID).First(&user).Error; err == nil {
					isAdmin = user.Role == models.RoleAdmin
				}
			}
		}
	}

	// Si la vidéo est en traitement, permettre l'accès même sans authentification
	// car le frontend a besoin de récupérer les infos pour afficher l'écran de traitement
	if video.Status == models.VideoStatusProcessing {
		// Permettre l'accès pour les vidéos en traitement (même sans auth, même si pas publique)
		// Cela permet au frontend d'afficher l'écran de traitement
		// Continuer pour retourner la vidéo
	} else if video.Status != models.VideoStatusReady {
		// Pour les vidéos en erreur, seuls propriétaire et admin peuvent les voir
		if !isOwner && !isAdmin {
			c.JSON(http.StatusNotFound, gin.H{"error": "Vidéo non trouvée"})
			return
		}
		// Pour les vidéos en erreur, retourner quand même la vidéo aux admins/propriétaires
		// pour qu'ils puissent voir les détails de l'erreur
	}

	// Si la vidéo n'est pas publique, seuls le propriétaire et les admins peuvent la voir
	// MAIS: Pour les vidéos en traitement, on a déjà permis l'accès ci-dessus
	// ET: Pour les vidéos prêtes mais non publiques, permettre l'accès au propriétaire et aux admins
	if !video.IsPublic && !isOwner && !isAdmin {
		// Si la vidéo est en traitement, on a déjà permis l'accès ci-dessus
		if video.Status != models.VideoStatusProcessing {
			fmt.Printf("[DEBUG GetVideo] Accès refusé: vidéo non publique, pas propriétaire/admin (Status: %s, IsPublic: %v, isOwner: %v, isAdmin: %v)\n", 
				video.Status, video.IsPublic, isOwner, isAdmin)
			c.JSON(http.StatusNotFound, gin.H{"error": "Vidéo non trouvée"})
			return
		}
	}

	// Incrémenter les vues seulement si la vidéo est prête et publique
	if video.Status == models.VideoStatusReady && video.IsPublic {
		h.db.Model(&video).Update("views", video.Views+1)
	}

	c.JSON(http.StatusOK, video)
}

// GetVideoStatus retourne le statut de traitement d'une vidéo
func (h *VideoHandler) GetVideoStatus(c *gin.Context) {
	id := c.Param("id")
	videoID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de vidéo invalide"})
		return
	}

	var video models.Video
	if err := h.db.Where("id = ?", videoID).First(&video).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vidéo non trouvée"})
		return
	}

	// Vérifier les permissions (propriétaire ou admin)
	// Pour les vidéos en traitement, permettre l'accès même sans authentification
	// car le frontend a besoin de récupérer le statut pour afficher l'écran de traitement
	userID, exists := c.Get("user_id")
	isOwner := false
	isAdmin := false
	
	if exists {
		userIDStr, ok := userID.(string)
		if ok && userIDStr != "" { // Vérifier que userIDStr n'est pas vide
			userUUID, err := uuid.Parse(userIDStr)
			if err == nil {
				if video.UserID == userUUID {
					isOwner = true
				}
				var user models.User
				if err := h.db.Where("id = ?", userUUID).First(&user).Error; err == nil {
					isAdmin = user.Role == models.RoleAdmin
				}
			}
		}
	}

	// Permettre l'accès au statut pour les vidéos en traitement même sans authentification
	// Cela permet au frontend d'afficher l'écran de traitement
	// Pour les vidéos en traitement, permettre l'accès même si pas propriétaire/admin
	fmt.Printf("[DEBUG GetVideoStatus] Vidéo %s: Status=%s, VideoStatusProcessing=%s, isOwner=%v, isAdmin=%v\n", 
		videoID, video.Status, models.VideoStatusProcessing, isOwner, isAdmin)
	
	if video.Status == models.VideoStatusProcessing {
		// Permettre l'accès au statut pour les vidéos en traitement (même sans auth)
		// Cela permet au frontend d'afficher l'écran de traitement
		fmt.Printf("[DEBUG GetVideoStatus] Accès autorisé pour vidéo en traitement %s\n", videoID)
	} else if !isOwner && !isAdmin {
		// Pour les autres statuts, vérifier les permissions
		fmt.Printf("[DEBUG GetVideoStatus] Accès refusé pour vidéo %s (statut: %s, isOwner: %v, isAdmin: %v)\n", 
			videoID, video.Status, isOwner, isAdmin)
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission refusée"})
		return
	}

	// Compter les formats créés
	var formatCount int64
	h.db.Model(&models.VideoFormat{}).Where("video_id = ?", videoID).Count(&formatCount)

	// Calculer le pourcentage de progression (approximatif)
	progress := 0
	if video.Status == models.VideoStatusReady {
		progress = 100
	} else if video.Status == models.VideoStatusProcessing {
		// Estimation basée sur le nombre de formats (5 formats max: 4 progressifs + 1 HLS)
		// Étapes du traitement :
		// - 0-10% : Génération miniature et récupération durée
		// - 10-80% : Génération des 4 formats progressifs (360p, 480p, 720p, 1080p) = 4/5 = 80%
		// - 80-95% : Génération du format HLS = 1/5 = 20%
		// - 95-100% : Finalisation
		
		// Si aucun format n'a été créé, on est au début du traitement
		if formatCount == 0 {
			// Vérifier si la miniature existe (indique qu'on a passé l'étape initiale)
			if video.ThumbnailPath != "" {
				progress = 10 // Miniature créée, début du transcodage
			} else {
				progress = 5 // Début du traitement (génération miniature, analyse, etc.)
			}
		} else {
			// Calculer la progression basée sur les formats créés
			// 5 formats max : 4 progressifs + 1 HLS
			// Chaque format progressif = 15% (4 * 15% = 60%)
			// Le format HLS = 25% (car plus long)
			// Total = 85% pour tous les formats, puis 15% pour la finalisation
			
			// Compter les formats progressifs et HLS séparément
			var progressiveCount int64
			var hlsCount int64
			h.db.Model(&models.VideoFormat{}).
				Where("video_id = ? AND format = ?", videoID, "progressive").
				Count(&progressiveCount)
			h.db.Model(&models.VideoFormat{}).
				Where("video_id = ? AND format = ?", videoID, "hls").
				Count(&hlsCount)
			
			// Calculer la progression
			// 0-10% : Initialisation (miniature, durée)
			// 10-70% : Formats progressifs (4 formats * 15% = 60%)
			// 70-95% : Format HLS (25%)
			// 95-100% : Finalisation
			
			baseProgress := 10 // Étape initiale terminée
			progressiveProgress := int((float64(progressiveCount) / 4.0) * 60.0) // Max 60%
			hlsProgress := int((float64(hlsCount) / 1.0) * 25.0) // Max 25%
			
			progress = baseProgress + progressiveProgress + hlsProgress
			
			// Ne pas dépasser 95% tant que le statut n'est pas "ready"
			if progress > 95 {
				progress = 95
			}
			
			// S'assurer qu'on a au moins 10% si un format a été créé
			if progress < 10 && formatCount > 0 {
				progress = 10
			}
		}
	} else if video.Status == models.VideoStatusError || video.Status == models.VideoStatusCancelled {
		progress = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"video_id": videoID,
		"status":   string(video.Status),
		"progress": progress,
		"formats_count": formatCount,
		"thumbnail_path": video.ThumbnailPath,
		"title": video.Title,
		"description": video.Description,
	})
}

func (h *VideoHandler) UpdateVideo(c *gin.Context) {
	id := c.Param("id")
	videoID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de vidéo invalide"})
		return
	}

	userID, _ := c.Get("user_id")
	userIDStr := userID.(string)
	userUUID, _ := uuid.Parse(userIDStr)

	var video models.Video
	if err := h.db.Where("id = ?", videoID).First(&video).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vidéo non trouvée"})
		return
	}

	// Vérifier les permissions (propriétaire ou admin)
	if video.UserID != userUUID {
		// Vérifier si admin
		var user models.User
		if err := h.db.Where("id = ?", userUUID).First(&user).Error; err != nil || user.Role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Permission refusée"})
			return
		}
	}

	var req struct {
		Title         string      `json:"title"`
		Description   string      `json:"description"`
		CategoryID    *uuid.UUID  `json:"category_id"`
		SubCategoryID *uuid.UUID  `json:"sub_category_id"`
		IsPublic      *bool       `json:"is_public"`
		Status        *string     `json:"status"` // Pour permettre le changement de statut
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Title != "" {
		video.Title = req.Title
	}
	if req.Description != "" {
		video.Description = req.Description
	}
	if req.CategoryID != nil {
		video.CategoryID = req.CategoryID
	}
	if req.SubCategoryID != nil {
		video.SubCategoryID = req.SubCategoryID
	}
	if req.IsPublic != nil {
		video.IsPublic = *req.IsPublic
	}

	// Permettre le changement de statut (admin seulement)
	if req.Status != nil {
		status := models.VideoStatus(*req.Status)
		// Valider le statut
		if status == models.VideoStatusPending || 
		   status == models.VideoStatusProcessing || 
		   status == models.VideoStatusReady || 
		   status == models.VideoStatusError {
			video.Status = status
		}
	}

	if err := h.db.Save(&video).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la mise à jour"})
		return
	}

	c.JSON(http.StatusOK, video)
}

func (h *VideoHandler) DeleteVideo(c *gin.Context) {
	id := c.Param("id")
	videoID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de vidéo invalide"})
		return
	}

	userID, _ := c.Get("user_id")
	userIDStr := userID.(string)
	userUUID, _ := uuid.Parse(userIDStr)

	var video models.Video
	if err := h.db.Where("id = ?", videoID).First(&video).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vidéo non trouvée"})
		return
	}

	// Vérifier les permissions (propriétaire ou admin)
	if video.UserID != userUUID {
		var user models.User
		if err := h.db.Where("id = ?", userUUID).First(&user).Error; err != nil || user.Role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Permission refusée"})
			return
		}
	}

	if err := h.db.Delete(&video).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la suppression"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Vidéo supprimée avec succès"})
}

func (h *VideoHandler) DownloadFromStream(c *gin.Context) {
	id := c.Param("id")
	videoID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de vidéo invalide"})
		return
	}

	var req struct {
		URL string `json:"url" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var video models.Video
	if err := h.db.Where("id = ?", videoID).First(&video).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vidéo non trouvée"})
		return
	}

	// Mettre à jour le statut et l'URL originale
	video.Status = models.VideoStatusProcessing
	video.OriginalURL = req.URL
	h.db.Save(&video)

	// Démarrer le téléchargement en arrière-plan
	// TODO: Implémenter le service de téléchargement

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Téléchargement démarré",
		"video_id": video.ID,
	})
}

func (h *VideoHandler) GetWatchHistory(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr := userID.(string)
	userUUID, _ := uuid.Parse(userIDStr)

	var history []models.WatchHistory
	if err := h.db.Preload("Video").Preload("Video.User").Where("user_id = ?", userUUID).Order("watched_at DESC").Limit(50).Find(&history).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la récupération de l'historique"})
		return
	}

	c.JSON(http.StatusOK, history)
}

func (h *VideoHandler) AddToWatchHistory(c *gin.Context) {
	videoIDParam := c.Param("videoId")
	videoID, err := uuid.Parse(videoIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de vidéo invalide"})
		return
	}

	userID, _ := c.Get("user_id")
	userIDStr := userID.(string)
	userUUID, _ := uuid.Parse(userIDStr)

	var req struct {
		Position int `json:"position"`
	}
	c.ShouldBindJSON(&req)

	// Mettre à jour ou créer l'entrée d'historique
	var history models.WatchHistory
	h.db.Where("user_id = ? AND video_id = ?", userUUID, videoID).FirstOrCreate(&history, models.WatchHistory{
		UserID:   userUUID,
		VideoID:  videoID,
		Position: req.Position,
	})

	history.Position = req.Position
	h.db.Save(&history)

	c.JSON(http.StatusOK, history)
}

// CancelVideoProcessing annule le traitement d'une vidéo en cours
func (h *VideoHandler) CancelVideoProcessing(c *gin.Context) {
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

	// Vérifier que la vidéo est en cours de traitement
	if video.Status != models.VideoStatusProcessing {
		c.JSON(http.StatusBadRequest, gin.H{"error": "La vidéo n'est pas en cours de traitement"})
		return
	}

	// Vérifier les permissions (propriétaire ou admin)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentification requise"})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentification requise"})
		return
	}

	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "ID utilisateur invalide"})
		return
	}

	// Vérifier si c'est le propriétaire ou un admin
	isOwner := video.UserID == userUUID
	var user models.User
	isAdmin := false
		if h.db.Where("id = ?", userUUID).First(&user).Error == nil {
		isAdmin = user.Role == models.RoleAdmin
	}

	if !isOwner && !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Vous n'avez pas la permission d'annuler cette vidéo"})
		return
	}

	// Annuler le traitement si une goroutine est en cours
	if h.uploadHandler != nil {
		h.uploadHandler.processingMutex.Lock()
		cancel, exists := h.uploadHandler.processingVideos[videoID]
		h.uploadHandler.processingMutex.Unlock()

		if exists && cancel != nil {
			// Annuler le context pour arrêter le traitement
			cancel()
			fmt.Printf("Traitement de la vidéo %s annulé par l'utilisateur %s\n", videoID, userUUID)
		}
	}

	// Mettre à jour le statut de la vidéo
	h.db.Model(&models.Video{}).Where("id = ?", videoID).Update("status", models.VideoStatusCancelled)

	c.JSON(http.StatusOK, gin.H{
		"message": "Traitement de la vidéo annulé avec succès",
		"video_id": videoID,
		"status":   "cancelled",
	})
}
