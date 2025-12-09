package api

import (
	"net/http"

	"streammake/internal/config"
	"streammake/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FavoriteHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewFavoriteHandler(db *gorm.DB, cfg *config.Config) *FavoriteHandler {
	return &FavoriteHandler{db: db, cfg: cfg}
}

// ListFavorites récupère tous les favoris de l'utilisateur connecté
func (h *FavoriteHandler) ListFavorites(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr := userID.(string)
	userUUID, _ := uuid.Parse(userIDStr)

	var favorites []models.Favorite
	if err := h.db.Preload("Video").Preload("Video.User").Preload("Video.Category").
		Where("user_id = ?", userUUID).
		Order("created_at DESC").
		Find(&favorites).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la récupération des favoris"})
		return
	}

	// Transformer en format simple pour le frontend
	videos := make([]models.Video, 0, len(favorites))
	for _, fav := range favorites {
		if fav.Video.ID != uuid.Nil {
			videos = append(videos, fav.Video)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"favorites": favorites,
		"videos":    videos,
		"total":     len(videos),
	})
}

// AddFavorite ajoute une vidéo aux favoris
func (h *FavoriteHandler) AddFavorite(c *gin.Context) {
	videoIDParam := c.Param("id")
	videoID, err := uuid.Parse(videoIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de vidéo invalide"})
		return
	}

	userID, _ := c.Get("user_id")
	userIDStr := userID.(string)
	userUUID, _ := uuid.Parse(userIDStr)

	// Vérifier que la vidéo existe
	var video models.Video
	if err := h.db.Where("id = ?", videoID).First(&video).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vidéo non trouvée"})
		return
	}

	// Vérifier si le favori existe déjà
	var existingFavorite models.Favorite
	if err := h.db.Where("user_id = ? AND video_id = ?", userUUID, videoID).First(&existingFavorite).Error; err == nil {
		c.JSON(http.StatusOK, gin.H{"message": "Vidéo déjà dans les favoris", "favorite": existingFavorite})
		return
	}

	// Créer le favori
	favorite := models.Favorite{
		ID:      uuid.New(),
		UserID:  userUUID,
		VideoID: videoID,
	}

	if err := h.db.Create(&favorite).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de l'ajout aux favoris"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Vidéo ajoutée aux favoris",
		"favorite": favorite,
	})
}

// RemoveFavorite retire une vidéo des favoris
func (h *FavoriteHandler) RemoveFavorite(c *gin.Context) {
	videoIDParam := c.Param("id")
	videoID, err := uuid.Parse(videoIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de vidéo invalide"})
		return
	}

	userID, _ := c.Get("user_id")
	userIDStr := userID.(string)
	userUUID, _ := uuid.Parse(userIDStr)

	// Supprimer le favori
	result := h.db.Where("user_id = ? AND video_id = ?", userUUID, videoID).Delete(&models.Favorite{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la suppression du favori"})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Favori non trouvé"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Vidéo retirée des favoris"})
}

// IsFavorite vérifie si une vidéo est dans les favoris
func (h *FavoriteHandler) IsFavorite(c *gin.Context) {
	videoIDParam := c.Param("id")
	videoID, err := uuid.Parse(videoIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de vidéo invalide"})
		return
	}

	userID, _ := c.Get("user_id")
	userIDStr := userID.(string)
	userUUID, _ := uuid.Parse(userIDStr)

	var favorite models.Favorite
	isFavorite := h.db.Where("user_id = ? AND video_id = ?", userUUID, videoID).First(&favorite).Error == nil

	c.JSON(http.StatusOK, gin.H{
		"is_favorite": isFavorite,
		"favorite":    favorite,
	})
}

