package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"streammake/internal/config"
	"streammake/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AdminHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewAdminHandler(db *gorm.DB, cfg *config.Config) *AdminHandler {
	return &AdminHandler{db: db, cfg: cfg}
}

func (h *AdminHandler) GetStats(c *gin.Context) {
	var stats struct {
		TotalUsers     int64 `json:"total_users"`
		TotalVideos    int64 `json:"total_videos"`
		TotalTorrents  int64 `json:"total_torrents"`
		ActiveTorrents int64 `json:"active_torrents"`
		TotalCategories int64 `json:"total_categories"`
	}

	h.db.Model(&models.User{}).Count(&stats.TotalUsers)
	h.db.Model(&models.Video{}).Count(&stats.TotalVideos)
	h.db.Model(&models.Torrent{}).Count(&stats.TotalTorrents)
	h.db.Model(&models.Torrent{}).Where("status IN ?", []string{"downloading", "seeding"}).Count(&stats.ActiveTorrents)
	h.db.Model(&models.Category{}).Count(&stats.TotalCategories)

	c.JSON(http.StatusOK, stats)
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	var users []models.User
	query := h.db

	// Pagination
	page := 1
	limit := 20
	if p := c.Query("page"); p != "" {
		if parsed, err := parseUint(p); err == nil {
			page = parsed
		}
	}
	if l := c.Query("limit"); l != "" {
		if parsed, err := parseUint(l); err == nil {
			limit = parsed
		}
	}
	offset := (page - 1) * limit

	// Recherche
	if search := c.Query("search"); search != "" {
		query = query.Where("username ILIKE ? OR email ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	var total int64
	query.Model(&models.User{}).Count(&total)

	query.Offset(offset).Limit(limit).Find(&users)

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *AdminHandler) UpdateUser(c *gin.Context) {
	userIDParam := c.Param("id")
	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID utilisateur invalide"})
		return
	}

	var req struct {
		IsActive *bool   `json:"is_active"`
		Role     *string `json:"role"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := h.db.Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Utilisateur non trouvé"})
		return
	}

	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}
	if req.Role != nil {
		user.Role = models.UserRole(*req.Role)
	}

	h.db.Save(&user)

	c.JSON(http.StatusOK, gin.H{
		"message": "Utilisateur mis à jour",
		"user":    user,
	})
}

func (h *AdminHandler) DeleteUser(c *gin.Context) {
	userIDParam := c.Param("id")
	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID utilisateur invalide"})
		return
	}

	if err := h.db.Delete(&models.User{}, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la suppression"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Utilisateur supprimé"})
}

// ListInvalidVideos liste les vidéos invalides sans les supprimer
func (h *AdminHandler) ListInvalidVideos(c *gin.Context) {
	var invalidVideos []gin.H

	// Trouver toutes les vidéos
	var videos []models.Video
	h.db.Preload("User").Find(&videos)

	for _, video := range videos {
		var reasons []string

		// Vérifier si le titre est null ou vide
		if video.Title == "" || video.Title == "null" || video.Title == "Null" || video.Title == "NULL" {
			reasons = append(reasons, "titre null ou vide")
		}

		// Vérifier si la vidéo n'a aucun format
		var formatCount int64
		h.db.Model(&models.VideoFormat{}).Where("video_id = ?", video.ID).Count(&formatCount)
		if formatCount == 0 {
			reasons = append(reasons, "aucun format disponible")
		}

		// Vérifier si le statut est "error"
		if video.Status == models.VideoStatusError {
			reasons = append(reasons, "statut error")
		}

		if len(reasons) > 0 {
			invalidVideos = append(invalidVideos, gin.H{
				"id":          video.ID,
				"title":       video.Title,
				"status":      video.Status,
				"user":        video.User.Username,
				"created_at":  video.CreatedAt,
				"reasons":     reasons,
				"format_count": formatCount,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"invalid_videos": invalidVideos,
		"count":         len(invalidVideos),
	})
}

// CleanInvalidVideos supprime les vidéos invalides (sans format, titre null, etc.)
func (h *AdminHandler) CleanInvalidVideos(c *gin.Context) {
	var deletedCount int
	var errors []string
	var deletedVideos []gin.H

	// Trouver toutes les vidéos
	var videos []models.Video
	h.db.Preload("User").Find(&videos)

	for _, video := range videos {
		shouldDelete := false
		var reasons []string

		// Vérifier si le titre est null ou vide
		if video.Title == "" || video.Title == "null" || video.Title == "Null" || video.Title == "NULL" {
			shouldDelete = true
			reasons = append(reasons, "titre null ou vide")
		}

		// Vérifier si la vidéo n'a aucun format
		var formatCount int64
		h.db.Model(&models.VideoFormat{}).Where("video_id = ?", video.ID).Count(&formatCount)
		if formatCount == 0 {
			shouldDelete = true
			reasons = append(reasons, "aucun format disponible")
		}

		// Vérifier si le statut est "error"
		if video.Status == models.VideoStatusError {
			shouldDelete = true
			reasons = append(reasons, "statut error")
		}

		if shouldDelete {
			// Enregistrer les infos de la vidéo avant suppression
			deletedVideos = append(deletedVideos, gin.H{
				"id":      video.ID.String(),
				"title":   video.Title,
				"status":  string(video.Status),
				"user":    video.User.Username,
				"reasons": reasons,
			})

			// Supprimer les formats associés
			h.db.Where("video_id = ?", video.ID).Delete(&models.VideoFormat{})
			
			// Supprimer les fichiers vidéo
			if videoPath := h.getVideoFilePath(video.ID); videoPath != "" {
				if err := os.Remove(videoPath); err != nil && !os.IsNotExist(err) {
					errors = append(errors, fmt.Sprintf("Erreur suppression fichier vidéo %s: %v", video.ID, err))
				}
			}

			// Supprimer les fichiers de format
			h.cleanupVideoFormats(video.ID)

			// Supprimer la miniature
			if video.ThumbnailPath != "" {
				if err := os.Remove(video.ThumbnailPath); err != nil && !os.IsNotExist(err) {
					// Ignorer les erreurs de suppression de miniature
				}
			}

			// Supprimer la vidéo de la base
			if err := h.db.Delete(&video).Error; err != nil {
				errors = append(errors, fmt.Sprintf("Erreur suppression vidéo %s: %v", video.ID, err))
			} else {
				deletedCount++
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Nettoyage terminé",
		"deleted_count": deletedCount,
		"deleted_videos": deletedVideos,
		"errors":        errors,
	})
}

// getVideoFilePath retourne le chemin du fichier vidéo original
func (h *AdminHandler) getVideoFilePath(videoID uuid.UUID) string {
	// Chercher dans le dossier de stockage
	pattern := filepath.Join(h.cfg.VideoStoragePath, fmt.Sprintf("%s_*", videoID.String()))
	matches, err := filepath.Glob(pattern)
	if err == nil && len(matches) > 0 {
		return matches[0]
	}
	return ""
}

// cleanupVideoFormats supprime tous les fichiers de format associés à une vidéo
func (h *AdminHandler) cleanupVideoFormats(videoID uuid.UUID) {
	var formats []models.VideoFormat
	h.db.Where("video_id = ?", videoID).Find(&formats)

	for _, format := range formats {
		// Supprimer le fichier de format
		if format.FilePath != "" {
			if err := os.Remove(format.FilePath); err != nil && !os.IsNotExist(err) {
				// Ignorer les erreurs
			}
		}

		// Pour HLS/DASH, supprimer le dossier entier
		if format.Format == "hls" || format.Format == "dash" {
			if format.FilePath != "" {
				if err := os.RemoveAll(format.FilePath); err != nil && !os.IsNotExist(err) {
					// Ignorer les erreurs
				}
			}
			if format.ManifestPath != "" {
				if err := os.Remove(format.ManifestPath); err != nil && !os.IsNotExist(err) {
					// Ignorer les erreurs
				}
			}
		}
	}
}

func parseUint(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}
