package middleware

import (
	"net/http"

	"streammake/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func AdminMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Non authentifié"})
			c.Abort()
			return
		}

		var user models.User
		if err := db.Where("id = ?", userID).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Utilisateur non trouvé"})
			c.Abort()
			return
		}

		if !user.IsAdmin() {
			c.JSON(http.StatusForbidden, gin.H{"error": "Accès refusé. Droits administrateur requis."})
			c.Abort()
			return
		}

		c.Set("user", user)
		c.Next()
	}
}

