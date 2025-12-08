package middleware

import (
	"net/http"

	"streammake/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AdminMiddleware vérifie que l'utilisateur est un administrateur
func AdminMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Non authentifié"})
			c.Abort()
			return
		}

		userIDStr := userID.(string)
		userUUID, err := uuid.Parse(userIDStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "ID utilisateur invalide"})
			c.Abort()
			return
		}

		var user models.User
		if err := db.Where("id = ?", userUUID).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Utilisateur non trouvé"})
			c.Abort()
			return
		}

		if user.Role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Accès refusé. Administrateur requis."})
			c.Abort()
			return
		}

		c.Next()
	}
}
