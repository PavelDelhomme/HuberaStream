package api

import (
	"net/http"
	"strconv"

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

// Gestion des utilisateurs
func (h *AdminHandler) ListUsers(c *gin.Context) {
	var users []models.User
	query := h.db

	// Pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	// Recherche
	if search := c.Query("search"); search != "" {
		query = query.Where("username ILIKE ? OR email ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	var total int64
	query.Model(&models.User{}).Count(&total)

	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la récupération des utilisateurs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *AdminHandler) UpdateUser(c *gin.Context) {
	id := c.Param("id")
	userID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID invalide"})
		return
	}

	var user models.User
	if err := h.db.Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Utilisateur non trouvé"})
		return
	}

	var req struct {
		Role     *models.UserRole `json:"role"`
		IsActive *bool            `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Role != nil {
		user.Role = *req.Role
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}

	if err := h.db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la mise à jour"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *AdminHandler) DeleteUser(c *gin.Context) {
	id := c.Param("id")
	userID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID invalide"})
		return
	}

	if err := h.db.Delete(&models.User{}, userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la suppression"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Utilisateur supprimé"})
}

// Statistiques
func (h *AdminHandler) GetStats(c *gin.Context) {
	var stats struct {
		TotalUsers    int64 `json:"total_users"`
		TotalVideos   int64 `json:"total_videos"`
		TotalTorrents int64 `json:"total_torrents"`
		ActiveTorrents int64 `json:"active_torrents"`
		TotalCategories int64 `json:"total_categories"`
		StorageUsed    int64 `json:"storage_used"` // TODO: Calculer la taille réelle
	}

	h.db.Model(&models.User{}).Count(&stats.TotalUsers)
	h.db.Model(&models.Video{}).Count(&stats.TotalVideos)
	h.db.Model(&models.Torrent{}).Count(&stats.TotalTorrents)
	h.db.Model(&models.Torrent{}).Where("status IN ?", []models.TorrentStatus{models.TorrentStatusDownloading, models.TorrentStatusSeeding}).Count(&stats.ActiveTorrents)
	h.db.Model(&models.Category{}).Count(&stats.TotalCategories)

	c.JSON(http.StatusOK, stats)
}

