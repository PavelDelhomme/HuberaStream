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
	
	// Ajouter les middlewares de base (logger et recovery)
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Middleware de debug TRÈS TÔT pour capturer toutes les requêtes
	// IMPORTANT: Ce middleware doit être le PREMIER pour capturer toutes les requêtes
	router.Use(func(c *gin.Context) {
		// Logger TOUTES les requêtes pour debug
		if c.Request.Method == "POST" || c.Request.Method == "OPTIONS" {
			log.Printf("[EARLY-DEBUG] %s %s - RemoteAddr: %s, Origin: %s, ClientIP: %s", 
				c.Request.Method,
				c.Request.URL.Path,
				c.Request.RemoteAddr,
				c.Request.Header.Get("Origin"),
				c.ClientIP())
		}
		c.Next()
		if (c.Request.Method == "POST" || c.Request.Method == "OPTIONS") && c.Request.URL.Path == "/api/auth/login" {
			log.Printf("[EARLY-DEBUG] %s %s terminée - Status: %d", c.Request.Method, c.Request.URL.Path, c.Writer.Status())
		}
	})

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
	dvdHandler := NewDVDHandler(db, cfg)
	torrentClientsHandler := NewTorrentClientsHandler(db, cfg)
	mediaSettingsHandler := NewMediaSettingsHandler(db, cfg)
	favoriteHandler := NewFavoriteHandler(db, cfg)

	// Routes publiques
	api := router.Group("/api")
	{
		// Authentification
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
		}

		// Route continue-watching (chemin différent pour éviter le conflit avec /videos/:id)
		api.GET("/continue-watching", videoHandler.GetContinueWatching)
		
		// Catégories (publiques - accessibles sans authentification)
		api.GET("/categories", categoryHandler.ListCategories)
		
		// Vidéos publiques (avec authentification optionnelle)
		videos := api.Group("/videos")
		// Ajouter le middleware d'authentification optionnelle pour permettre l'accès aux vidéos en traitement
		videos.Use(middleware.OptionalAuthMiddleware(cfg.JWTSecret))
		{
			videos.GET("", videoHandler.ListVideos)
			videos.GET("/:id/stream", streamHandler.StreamVideo)
			videos.GET("/:id/subtitles/:subtitleId", streamHandler.StreamSubtitle)
			videos.GET("/:id/status", videoHandler.GetVideoStatus)
			videos.GET("/:id", videoHandler.GetVideo)
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
			videos.POST("/:id/thumbnail", uploadHandler.UploadThumbnail)
			videos.PUT("/:id", videoHandler.UpdateVideo)
			videos.DELETE("/:id", videoHandler.DeleteVideo)
			videos.POST("/:id/download", videoHandler.DownloadFromStream)
		}

		// Extraction DVD (admin seulement)
		dvd := protected.Group("/dvd")
		dvd.Use(middleware.AdminMiddleware(db))
		{
			dvd.GET("/detect", dvdHandler.DetectDVD)
			dvd.GET("/info", dvdHandler.GetDVDInfo)
			dvd.POST("/extract", dvdHandler.ExtractDVD)
			dvd.GET("/extract/:id/status", dvdHandler.GetExtractionStatus)
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

		// Favoris
		favorites := protected.Group("/favorites")
		{
			favorites.GET("", favoriteHandler.ListFavorites)
			favorites.POST("/:id", favoriteHandler.AddFavorite)
			favorites.DELETE("/:id", favoriteHandler.RemoveFavorite)
			favorites.GET("/:id/check", favoriteHandler.IsFavorite)
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

		// Nettoyage des vidéos invalides
		admin.GET("/videos/invalid", adminHandler.ListInvalidVideos)
		admin.POST("/videos/clean-invalid", adminHandler.CleanInvalidVideos)

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

		// Gestion des clients torrent
		torrentClients := admin.Group("/torrent-clients")
		{
			torrentClients.GET("/status", torrentClientsHandler.GetTorrentClientsStatus)
			torrentClients.GET("/current", torrentClientsHandler.GetCurrentTorrentClient)
			torrentClients.PUT("/current", torrentClientsHandler.UpdateCurrentTorrentClient)
			torrentClients.POST("/test", torrentClientsHandler.TestTorrentClient)
		}

		// Paramètres multimédias
		mediaSettings := admin.Group("/media-settings")
		{
			mediaSettings.GET("", mediaSettingsHandler.GetMediaSettings)
			mediaSettings.PUT("", mediaSettingsHandler.UpdateMediaSettings)
		}

		// Scraper (recherche de torrents)
		scrapers := admin.Group("/scraper")
		{
			scrapers.POST("/search", scraperHandler.Search)
			scrapers.GET("/details", scraperHandler.GetTorrentDetails)
		}

		// Upload vidéo (admin seulement)
		admin.POST("/videos/upload", uploadHandler.UploadVideo)
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

