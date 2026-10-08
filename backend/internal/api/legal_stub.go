package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const legalCatalogMessage = "Hubera Stream ne propose que le catalogue légal Pluto TV et Arte.tv. Pas de client torrent, pas de magnet, pas d’index pirate."

// LegalCatalogOnly répond aux anciennes routes torrent / *arr : plus de client, uniquement le catalogue légal.
func LegalCatalogOnly(c *gin.Context) {
	c.JSON(http.StatusGone, gin.H{
		"error":   legalCatalogMessage,
		"catalog": "/app/",
		"sources": []string{"Pluto TV", "Arte.tv"},
	})
}
