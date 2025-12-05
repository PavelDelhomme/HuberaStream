package main

import (
	"log"
	"os"

	"streammake/internal/api"
	"streammake/internal/database"
	"streammake/internal/config"
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

	// Initialiser le routeur API
	router := api.SetupRouter(db, cfg)

	// Configurer Gin pour accepter les requêtes proxy (nécessaire pour Nginx)
	// CRITIQUE: Dans Gin v1.9.1, il y a un bug connu où SetTrustedProxies peut ne pas fonctionner correctement
	// Solution de contournement: Désactiver complètement la vérification des proxies en mode développement
	// En production, utiliser une liste spécifique de proxies de confiance
	if cfg.Env == "development" {
		// En développement, accepter tous les proxies (solution de contournement)
		// Note: nil devrait fonctionner mais ne fonctionne pas dans certaines versions
		router.SetTrustedProxies(nil)
		log.Printf("Mode développement: Tous les proxies acceptés (SetTrustedProxies(nil))")
	} else {
		// En production, spécifier uniquement les IPs exactes des proxies
		router.SetTrustedProxies([]string{
			"172.19.0.0/16",  // Réseau Docker par défaut
			"172.17.0.0/16",  // Réseau Docker bridge
			"127.0.0.1",      // Localhost IPv4
			"::1",            // Localhost IPv6
		})
		log.Printf("Mode production: Proxies spécifiques configurés")
	}
	router.TrustedPlatform = "X-Forwarded-For" // Utiliser X-Forwarded-For pour identifier l'IP du client

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

