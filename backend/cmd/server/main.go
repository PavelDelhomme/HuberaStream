package main

import (
	"context"
	"log"
	"os"
	"time"

	"streammake/internal/api"
	"streammake/internal/database"
	"streammake/internal/config"
	"streammake/internal/torrent"
)

func main() {
	// Charger la configuration
	cfg := config.Load()

	// Initialiser la base de données
	db, err := database.Initialize(cfg)
	if err != nil {
		log.Fatalf("Erreur lors de l'initialisation de la base de données: %v", err)
	}

	// Seed la base de données (créer admin et catégories par défaut)
	if err := database.SeedDatabase(db, cfg); err != nil {
		log.Printf("Avertissement lors du seed de la base de données: %v", err)
	}

	// Créer les dossiers de stockage si nécessaire
	if err := os.MkdirAll(cfg.VideoStoragePath, 0755); err != nil {
		log.Fatalf("Erreur lors de la création du dossier de stockage vidéo: %v", err)
	}
	if err := os.MkdirAll(cfg.ThumbnailStoragePath, 0755); err != nil {
		log.Fatalf("Erreur lors de la création du dossier de stockage thumbnails: %v", err)
	}

	// Configurer automatiquement qBittorrent si nécessaire
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		
		// Attendre un peu que qBittorrent démarre
		time.Sleep(10 * time.Second)
		
		if cfg.TorrentClientQBittorrentURL != "" && cfg.TorrentClientQBittorrentPass != "" {
			log.Printf("🔧 Configuration automatique de qBittorrent...")
			if err := torrent.ConfigureQBittorrentFromDocker(
				ctx,
				"streammake-qbittorrent",
				cfg.TorrentClientQBittorrentUser,
				cfg.TorrentClientQBittorrentPass,
			); err != nil {
				log.Printf("⚠️  Erreur lors de la configuration automatique de qBittorrent: %v", err)
				log.Printf("   Vous pouvez configurer qBittorrent manuellement via http://localhost:3306")
			} else {
				log.Printf("✅ qBittorrent configuré automatiquement avec succès!")
				log.Printf("   Redémarrez qBittorrent pour que les changements prennent effet")
			}
		}
	}()

	// Initialiser le routeur API
	router := api.SetupRouter(db, cfg)

	// CONFIGURATION DES PROXIES DE CONFIANCE
	// D'après le rapport de diagnostic:
	// - Backend IP: 172.18.0.4
	// - Nginx IP: 172.18.0.6
	// - X-Forwarded-For peut être 172.19.0.1 (réseau différent)
	// - Une requête depuis Nginx (172.18.0.6) a réussi (ligne 87 du rapport)
	//
	// Le warning "You trusted all proxies" apparaît quand SetTrustedProxies n'est pas configuré
	// mais que Gin détecte des headers X-Forwarded-For. Il faut configurer explicitement.
	
	trustedProxies := []string{
		"172.18.0.6/32",   // IP exacte de Nginx (d'après le rapport)
		"172.18.0.0/16",  // Réseau Docker complet (172.18.x.x)
		"172.19.0.0/16",  // Réseau Docker alternatif (pour X-Forwarded-For)
		"172.17.0.0/16",  // Réseau Docker bridge par défaut
		"127.0.0.1",      // Localhost
		"::1",            // Localhost IPv6
	}
	
	if err := router.SetTrustedProxies(trustedProxies); err != nil {
		log.Printf("Avertissement: Erreur lors de la configuration des proxies: %v", err)
	}
	
	// Utiliser X-Forwarded-For comme plateforme de confiance
	router.TrustedPlatform = "X-Forwarded-For"
	
	log.Printf("Configuration des proxies: IPs de confiance = %v, TrustedPlatform = %s", trustedProxies, router.TrustedPlatform)

	// Démarrer le serveur
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Serveur démarré sur le port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Erreur lors du démarrage du serveur: %v", err)
	}
}

