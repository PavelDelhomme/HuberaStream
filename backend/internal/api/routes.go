package api

import (
	"log"
	"net/http"
	"streammake/internal/config"
	"streammake/internal/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRouter(db *gorm.DB, cfg *config.Config) *gin.Engine {
	router := gin.New()
	
	// Middleware de debug pour identifier où les requêtes sont bloquées
	router.Use(func(c *gin.Context) {
		if c.Request.Method == "POST" && c.Request.URL.Path == "/api/auth/login" {
			log.Printf("[DEBUG] Requête POST /api/auth/login reçue - RemoteAddr: %s, ClientIP: %s, X-Forwarded-For: %s", 
				c.Request.RemoteAddr, c.ClientIP(), c.Request.Header.Get("X-Forwarded-For"))
		}
		c.Next()
	})
	
	// Ajouter les middlewares de base (logger et recovery)
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Configuration CORS - DOIT être le premier middleware après logger/recovery
	router.Use(middleware.CORSMiddleware())

	// Initialiser les handlers
	authHandler := NewAuthHandler(db, cfg)
	videoHandler := NewVideoHandler(db, cfg)
	playlistHandler := NewPlaylistHandler(db)
	streamHandler := NewStreamHandler(db, cfg)
	adminHandler := NewAdminHandler(db, cfg)
	categoryHandler := NewCategoryHandler(db)
	torrentHandler := NewTorrentHandler(db, cfg)
	uploadHandler := NewUploadHandler(db, cfg)
	scraperHandler := NewScraperHandler(db, cfg)

	// Routes publiques
	api := router.Group("/api")
	{
		// Authentification
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
		}

		// Vidéos publiques
		videos := api.Group("/videos")
		{
			videos.GET("", videoHandler.ListVideos)
			videos.GET("/:id", videoHandler.GetVideo)
			videos.GET("/:id/stream", streamHandler.StreamVideo)
		}
	}

	// Routes protégées
	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware(cfg.JWTSecret))
	{
		// Vidéos
		videos := protected.Group("/videos")
		{
			videos.POST("", videoHandler.CreateVideo)
			videos.POST("/upload", uploadHandler.UploadVideo)
			videos.POST("/:id/thumbnail", uploadHandler.UploadThumbnail)
			videos.PUT("/:id", videoHandler.UpdateVideo)
			videos.DELETE("/:id", videoHandler.DeleteVideo)
			videos.POST("/:id/download", videoHandler.DownloadFromStream)
		}

		// Playlists
		playlists := protected.Group("/playlists")
		{
			playlists.GET("", playlistHandler.ListPlaylists)
			playlists.POST("", playlistHandler.CreatePlaylist)
			playlists.GET("/:id", playlistHandler.GetPlaylist)
			playlists.PUT("/:id", playlistHandler.UpdatePlaylist)
			playlists.DELETE("/:id", playlistHandler.DeletePlaylist)
			playlists.POST("/:id/videos/:videoId", playlistHandler.AddVideoToPlaylist)
			playlists.DELETE("/:id/videos/:videoId", playlistHandler.RemoveVideoFromPlaylist)
		}

		// Historique
		history := protected.Group("/history")
		{
			history.GET("", videoHandler.GetWatchHistory)
			history.POST("/:videoId", videoHandler.AddToWatchHistory)
		}

		// Profil utilisateur
		users := protected.Group("/users")
		{
			users.GET("/me", authHandler.GetProfile)
		}

		// Catégories (lecture pour tous)
		categories := protected.Group("/categories")
		{
			categories.GET("", categoryHandler.ListCategories)
		}
	}

	// Routes administrateur
	admin := api.Group("/admin")
	admin.Use(middleware.AuthMiddleware(cfg.JWTSecret))
	admin.Use(middleware.AdminMiddleware(db))
	{
		// Utilisateurs
		users := admin.Group("/users")
		{
			users.GET("", adminHandler.ListUsers)
			users.PUT("/:id", adminHandler.UpdateUser)
			users.DELETE("/:id", adminHandler.DeleteUser)
		}

		// Statistiques
		admin.GET("/stats", adminHandler.GetStats)

		// Catégories
		categories := admin.Group("/categories")
		{
			categories.POST("", categoryHandler.CreateCategory)
			categories.PUT("/:id", categoryHandler.UpdateCategory)
			categories.POST("/:categoryId/subcategories", categoryHandler.CreateSubCategory)
		}

		// Torrents
		torrents := admin.Group("/torrents")
		{
			torrents.GET("", torrentHandler.ListTorrents)
			torrents.POST("", torrentHandler.CreateTorrent)
			torrents.PUT("/:id", torrentHandler.UpdateTorrent)
			torrents.DELETE("/:id", torrentHandler.DeleteTorrent)
		}

		// Configurations de torrent
		torrentConfigs := admin.Group("/torrent-configs")
		{
			torrentConfigs.GET("", torrentHandler.ListTorrentConfigs)
			torrentConfigs.POST("", torrentHandler.CreateTorrentConfig)
			torrentConfigs.PUT("/:id", torrentHandler.UpdateTorrentConfig)
			torrentConfigs.DELETE("/:id", torrentHandler.DeleteTorrentConfig)
		}

		// Scraper (recherche de torrents)
		scrapers := admin.Group("/scraper")
		{
			scrapers.POST("/search", scraperHandler.Search)
			scrapers.GET("/details", scraperHandler.GetTorrentDetails)
		}
	}

	// Handler pour les routes non trouvées (debug)
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Route non trouvée",
			"path":  c.Request.URL.Path,
			"method": c.Request.Method,
		})
	})

	return router
}

