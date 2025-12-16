package api

import (
	"net/http"

	"streammake/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CategoryHandler struct {
	db *gorm.DB
}

func NewCategoryHandler(db *gorm.DB) *CategoryHandler {
	return &CategoryHandler{db: db}
}

func (h *CategoryHandler) ListCategories(c *gin.Context) {
	var categories []models.Category
	if err := h.db.Preload("SubCategories").Where("is_active = ?", true).Find(&categories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la récupération des catégories"})
		return
	}
	c.JSON(http.StatusOK, categories)
}

func (h *CategoryHandler) CreateCategory(c *gin.Context) {
	var req struct {
		Name        string              `json:"name" binding:"required"`
		Type        models.CategoryType `json:"type" binding:"required"`
		Description string              `json:"description"`
		Icon        string              `json:"icon"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	category := models.Category{
		ID:          uuid.New(),
		Name:        req.Name,
		Type:        req.Type,
		Description: req.Description,
		Icon:        req.Icon,
		IsActive:    true,
	}

	if err := h.db.Create(&category).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la création"})
		return
	}

	c.JSON(http.StatusCreated, category)
}

func (h *CategoryHandler) UpdateCategory(c *gin.Context) {
	id := c.Param("id")
	categoryID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID invalide"})
		return
	}

	var category models.Category
	if err := h.db.Where("id = ?", categoryID).First(&category).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Catégorie non trouvée"})
		return
	}

	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		Icon        *string `json:"icon"`
		IsActive    *bool   `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != nil {
		category.Name = *req.Name
	}
	if req.Description != nil {
		category.Description = *req.Description
	}
	if req.Icon != nil {
		category.Icon = *req.Icon
	}
	if req.IsActive != nil {
		category.IsActive = *req.IsActive
	}

	if err := h.db.Save(&category).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la mise à jour"})
		return
	}

	c.JSON(http.StatusOK, category)
}

func (h *CategoryHandler) CreateSubCategory(c *gin.Context) {
	categoryIDParam := c.Param("categoryId")
	categoryID, err := uuid.Parse(categoryIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de catégorie invalide"})
		return
	}

	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	subCategory := models.SubCategory{
		ID:          uuid.New(),
		CategoryID:  categoryID,
		Name:        req.Name,
		Description: req.Description,
		IsActive:    true,
	}

	if err := h.db.Create(&subCategory).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la création"})
		return
	}

	c.JSON(http.StatusCreated, subCategory)
}
