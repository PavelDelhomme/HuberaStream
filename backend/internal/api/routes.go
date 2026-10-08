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
	dvdHandler := NewDVDHandler(db, cfg, uploadHandler) // Passer uploadHandler pour traitement automatique
	mediaSettingsHandler := NewMediaSettingsHandler(db, cfg)
	favoriteHandler := NewFavoriteHandler(db, cfg)
	aiHandler := NewAIHandler(db)

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

		api.POST("/webhooks/radarr", LegalCatalogOnly)
		api.POST("/webhooks/sonarr", LegalCatalogOnly)

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
			dvd.POST("/extract/:id/cancel", dvdHandler.CancelExtraction)
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

		// IA - Recommandations
		ai := protected.Group("/ai")
		{
			ai.GET("/recommendations", aiHandler.GetRecommendations)
			ai.POST("/remember-selection", aiHandler.RememberMovieSelection)
			ai.GET("/learning-stats", aiHandler.GetLearningStats)
		}

		// IA - Suivi de séries
		series := protected.Group("/series")
		{
			series.GET("", aiHandler.GetTrackedSeries)
			series.POST("", aiHandler.AddSeriesToTracking)
			series.PUT("/:id", aiHandler.UpdateSeriesTracking)
			series.DELETE("/:id", aiHandler.DeleteSeriesTracking)
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

		// Anciennes routes torrent / *arr : stub légal (plus de client)
		admin.Any("/torrents", LegalCatalogOnly)
		admin.Any("/torrents/*path", LegalCatalogOnly)
		admin.Any("/torrent-configs", LegalCatalogOnly)
		admin.Any("/torrent-configs/*path", LegalCatalogOnly)
		admin.Any("/torrent-candidates", LegalCatalogOnly)
		admin.Any("/torrent-candidates/*path", LegalCatalogOnly)
		admin.Any("/torrent-learning-patterns", LegalCatalogOnly)
		admin.Any("/torrent-learning-patterns/*path", LegalCatalogOnly)
		admin.Any("/torrent-clients", LegalCatalogOnly)
		admin.Any("/torrent-clients/*path", LegalCatalogOnly)
		admin.Any("/torrent-indexes", LegalCatalogOnly)
		admin.Any("/torrent-indexes/*path", LegalCatalogOnly)
		admin.Any("/scraper", LegalCatalogOnly)
		admin.Any("/scraper/*path", LegalCatalogOnly)
		admin.Any("/media-stack/status", LegalCatalogOnly)
		admin.Any("/webhooks/setup", LegalCatalogOnly)
		admin.Any("/webhooks/events", LegalCatalogOnly)

		// Paramètres multimédias
		mediaSettings := admin.Group("/media-settings")
		{
			mediaSettings.GET("", mediaSettingsHandler.GetMediaSettings)
			mediaSettings.PUT("", mediaSettingsHandler.UpdateMediaSettings)
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
