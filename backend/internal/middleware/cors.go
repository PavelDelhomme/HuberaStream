package middleware

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware gère les requêtes CORS de manière permissive pour le développement
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		// Répondre immédiatement aux requêtes OPTIONS (preflight)
		if c.Request.Method == "OPTIONS" {
			if origin != "" {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			} else {
				c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			}
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Accept, Origin, X-Requested-With, Access-Control-Request-Method, Access-Control-Request-Headers")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH, HEAD")
			c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Type, Authorization")
			c.Writer.Header().Set("Access-Control-Max-Age", "86400")
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		// Pour les autres requêtes, ajouter les headers CORS
		if origin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		}
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Accept, Origin, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH, HEAD")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Type, Authorization")

		// Debug: logger les requêtes POST pour identifier le problème
		if c.Request.Method == "POST" && c.Request.URL.Path == "/api/auth/login" {
			log.Printf("[CORS] POST /api/auth/login - Origin: %s, RemoteAddr: %s, X-Forwarded-For: %s, ClientIP: %s", 
				origin, c.Request.RemoteAddr, c.Request.Header.Get("X-Forwarded-For"), c.ClientIP())
		}

		// Ne PAS bloquer les requêtes - laisser passer
		c.Next()
	}
}
