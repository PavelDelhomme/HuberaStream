package api

import (
	"log"
	"net/http"
	"streammake/internal/config"
	"streammake/internal/middleware"
	"strings"

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
		// Logger TOUTES les requêtes pour debug (y compris GET)
		if strings.Contains(c.Request.URL.Path, "/api/videos/") || strings.Contains(c.Request.URL.Path, "/api/thumbnails/") {
			log.Printf("[EARLY-DEBUG] %s %s - RemoteAddr: %s, Origin: %s, ClientIP: %s",
				c.Request.Method,
				c.Request.URL.Path,
				c.Request.RemoteAddr,
				c.Request.Header.Get("Origin"),
				c.ClientIP())
		}
		c.Next()
		if strings.Contains(c.Request.URL.Path, "/api/videos/") || strings.Contains(c.Request.URL.Path, "/api/thumbnails/") {
			log.Printf("[EARLY-DEBUG] %s %s terminée - Status: %d", c.Request.Method, c.Request.URL.Path, c.Writer.Status())
		}
	})

	// Configuration CORS - DOIT être le premier middleware après logger/recovery
	router.Use(middleware.CORSMiddleware())

	// Initialiser les handlers
	authHandler := NewAuthHandler(db, cfg)
	uploadHandler := NewUploadHandler(db, cfg)
	videoHandler := NewVideoHandler(db, cfg, uploadHandler)
	chunkUploadHandler := NewChunkUploadHandler(db, cfg, uploadHandler)
	playlistHandler := NewPlaylistHandler(db)
	streamHandler := NewStreamHandler(db, cfg)
	adminHandler := NewAdminHandler(db, cfg)
	categoryHandler := NewCategoryHandler(db)
	torrentHandler := NewTorrentHandler(db, cfg)
	scraperHandler := NewScraperHandler(db, cfg)
	dvdHandler := NewDVDHandler(db, cfg, uploadHandler) // Passer uploadHandler pour traitement automatique
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

		// Servir les thumbnails (publiques) - utiliser un handler dédié
		api.GET("/thumbnails/:filename", streamHandler.ServeThumbnail)

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
			videos.POST("/:id/cancel", videoHandler.CancelVideoProcessing)
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

		// IMPORTANT: Routes spécifiques AVANT le groupe pour éviter les conflits
		// Toutes les routes spécifiques /videos/* doivent être AVANT le groupe videosAdmin

		// Recherche de métadonnées basée sur le nom de fichier (sans upload)
		admin.POST("/videos/search-metadata", uploadHandler.SearchMetadataFromFilename)

		// Récupération des détails complets d'un film depuis TMDB
		admin.POST("/videos/metadata-details", uploadHandler.GetMetadataDetails)

		// Extraction de métadonnées (route spécifique, DOIT être la première route /videos)
		admin.POST("/videos/extract-metadata", uploadHandler.ExtractVideoMetadata)

		// Upload vidéo (route spécifique)
		admin.POST("/videos/upload", uploadHandler.UploadVideo)

		// Nettoyage des vidéos invalides
		admin.GET("/videos/invalid", adminHandler.ListInvalidVideos)
		admin.POST("/videos/clean-invalid", adminHandler.CleanInvalidVideos)

		// Groupe pour les routes avec paramètres (après les routes spécifiques)
		videosAdmin := admin.Group("/videos")
		{
			// Routes avec paramètres
			videosAdmin.POST("/:id/extract-metadata", uploadHandler.ExtractVideoMetadataFromID)
		}

		// Génération de miniature depuis titre
		admin.GET("/thumbnails/generate", uploadHandler.GenerateThumbnailFromTitle)

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
			torrents.GET("/:id/status", torrentHandler.GetTorrentStatus)
			torrents.POST("/:id/pause", torrentHandler.PauseTorrent)
			torrents.POST("/:id/resume", torrentHandler.ResumeTorrent)
			torrents.POST("/:id/toggle-library", torrentHandler.ToggleAddToLibrary)
			torrents.PUT("/:id", torrentHandler.UpdateTorrent)
			torrents.DELETE("/:id", torrentHandler.DeleteTorrent)
			torrents.POST("/download-1337x", torrentHandler.DownloadFrom1337x)
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
			scrapers.GET("/browse-1337x", scraperHandler.BrowseCategory1337x)
			scrapers.POST("/crawler/start", scraperHandler.StartCrawler)
			scrapers.POST("/crawler/stop", scraperHandler.StopCrawler)
			scrapers.GET("/crawler/status", scraperHandler.GetCrawlerStatus)
		}

		// Upload par chunks (avec pause/reprise)
		chunks := admin.Group("/uploads")
		{
			chunks.POST("/init", chunkUploadHandler.InitUploadSession)
			chunks.POST("/:session_id/chunk", chunkUploadHandler.UploadChunk)
			chunks.GET("/:session_id/status", chunkUploadHandler.GetUploadStatus)
			chunks.POST("/:session_id/pause", chunkUploadHandler.PauseUpload)
			chunks.POST("/:session_id/resume", chunkUploadHandler.ResumeUpload)
			chunks.POST("/:session_id/finalize", chunkUploadHandler.FinalizeUpload)
			chunks.POST("/:session_id/cancel", chunkUploadHandler.CancelUpload)
		}
	}

	// Handler pour les routes non trouvées (debug)
	router.NoRoute(func(c *gin.Context) {
		log.Printf("[NoRoute] Route non trouvée: %s %s", c.Request.Method, c.Request.URL.Path)
		c.JSON(http.StatusNotFound, gin.H{
			"error":  "Route non trouvée",
			"path":   c.Request.URL.Path,
			"method": c.Request.Method,
		})
	})

	return router
}
