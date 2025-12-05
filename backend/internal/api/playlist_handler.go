package api

import (
	"net/http"

	"streammake/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PlaylistHandler struct {
	db *gorm.DB
}

func NewPlaylistHandler(db *gorm.DB) *PlaylistHandler {
	return &PlaylistHandler{db: db}
}

type CreatePlaylistRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	IsPublic    bool   `json:"is_public"`
}

func (h *PlaylistHandler) CreatePlaylist(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr := userID.(string)
	userUUID, _ := uuid.Parse(userIDStr)

	var req CreatePlaylistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	playlist := models.Playlist{
		ID:          uuid.New(),
		Name:        req.Name,
		Description: req.Description,
		UserID:      userUUID,
		IsPublic:    req.IsPublic,
	}

	if err := h.db.Create(&playlist).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la création de la playlist"})
		return
	}

	c.JSON(http.StatusCreated, playlist)
}

func (h *PlaylistHandler) ListPlaylists(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr := userID.(string)
	userUUID, _ := uuid.Parse(userIDStr)

	var playlists []models.Playlist
	if err := h.db.Preload("Videos").Where("user_id = ?", userUUID).Find(&playlists).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la récupération des playlists"})
		return
	}

	c.JSON(http.StatusOK, playlists)
}

func (h *PlaylistHandler) GetPlaylist(c *gin.Context) {
	id := c.Param("id")
	playlistID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de playlist invalide"})
		return
	}

	var playlist models.Playlist
	if err := h.db.Preload("Videos").Preload("Videos.User").Preload("Videos.Formats").Preload("Items").Preload("Items.Video").Where("id = ?", playlistID).First(&playlist).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Playlist non trouvée"})
		return
	}

	c.JSON(http.StatusOK, playlist)
}

func (h *PlaylistHandler) UpdatePlaylist(c *gin.Context) {
	id := c.Param("id")
	playlistID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de playlist invalide"})
		return
	}

	userID, _ := c.Get("user_id")
	userIDStr := userID.(string)

	var playlist models.Playlist
	if err := h.db.Where("id = ?", playlistID).First(&playlist).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Playlist non trouvée"})
		return
	}

	if playlist.UserID.String() != userIDStr {
		c.JSON(http.StatusForbidden, gin.H{"error": "Vous n'êtes pas autorisé à modifier cette playlist"})
		return
	}

	var req CreatePlaylistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	playlist.Name = req.Name
	playlist.Description = req.Description
	playlist.IsPublic = req.IsPublic

	if err := h.db.Save(&playlist).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la mise à jour"})
		return
	}

	c.JSON(http.StatusOK, playlist)
}

func (h *PlaylistHandler) DeletePlaylist(c *gin.Context) {
	id := c.Param("id")
	playlistID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de playlist invalide"})
		return
	}

	userID, _ := c.Get("user_id")
	userIDStr := userID.(string)

	var playlist models.Playlist
	if err := h.db.Where("id = ?", playlistID).First(&playlist).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Playlist non trouvée"})
		return
	}

	if playlist.UserID.String() != userIDStr {
		c.JSON(http.StatusForbidden, gin.H{"error": "Vous n'êtes pas autorisé à supprimer cette playlist"})
		return
	}

	if err := h.db.Delete(&playlist).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la suppression"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Playlist supprimée avec succès"})
}

func (h *PlaylistHandler) AddVideoToPlaylist(c *gin.Context) {
	playlistIDParam := c.Param("id")
	videoIDParam := c.Param("videoId")

	playlistID, err := uuid.Parse(playlistIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de playlist invalide"})
		return
	}

	videoID, err := uuid.Parse(videoIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de vidéo invalide"})
		return
	}

	userID, _ := c.Get("user_id")
	userIDStr := userID.(string)

	var playlist models.Playlist
	if err := h.db.Where("id = ?", playlistID).First(&playlist).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Playlist non trouvée"})
		return
	}

	if playlist.UserID.String() != userIDStr {
		c.JSON(http.StatusForbidden, gin.H{"error": "Vous n'êtes pas autorisé à modifier cette playlist"})
		return
	}

	var video models.Video
	if err := h.db.Where("id = ?", videoID).First(&video).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vidéo non trouvée"})
		return
	}

	// Compter les vidéos existantes pour déterminer la position
	var count int64
	h.db.Model(&models.PlaylistVideo{}).Where("playlist_id = ?", playlistID).Count(&count)

	playlistVideo := models.PlaylistVideo{
		ID:         uuid.New(),
		PlaylistID: playlistID,
		VideoID:    videoID,
		Position:   int(count) + 1,
	}

	if err := h.db.Create(&playlistVideo).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de l'ajout de la vidéo"})
		return
	}

	c.JSON(http.StatusOK, playlistVideo)
}

func (h *PlaylistHandler) RemoveVideoFromPlaylist(c *gin.Context) {
	playlistIDParam := c.Param("id")
	videoIDParam := c.Param("videoId")

	playlistID, err := uuid.Parse(playlistIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de playlist invalide"})
		return
	}

	videoID, err := uuid.Parse(videoIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de vidéo invalide"})
		return
	}

	userID, _ := c.Get("user_id")
	userIDStr := userID.(string)

	var playlist models.Playlist
	if err := h.db.Where("id = ?", playlistID).First(&playlist).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Playlist non trouvée"})
		return
	}

	if playlist.UserID.String() != userIDStr {
		c.JSON(http.StatusForbidden, gin.H{"error": "Vous n'êtes pas autorisé à modifier cette playlist"})
		return
	}

	if err := h.db.Where("playlist_id = ? AND video_id = ?", playlistID, videoID).Delete(&models.PlaylistVideo{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la suppression"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Vidéo retirée de la playlist"})
}

