package api

import (
	"streammake/internal/config"
	"streammake/internal/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRouter(db *gorm.DB, cfg *config.Config) *gin.Engine {
	router := gin.Default()

	// Configuration CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:80"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Initialiser les handlers
	authHandler := NewAuthHandler(db, cfg)
	videoHandler := NewVideoHandler(db, cfg)
	playlistHandler := NewPlaylistHandler(db)
	streamHandler := NewStreamHandler(db, cfg)

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
	}

	return router
}

