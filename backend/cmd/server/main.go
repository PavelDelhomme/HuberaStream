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
	// IMPORTANT: SetTrustedProxies doit être appelé AVANT SetupRouter dans Gin v1.9+
	// Mais ici on le fait après car SetupRouter retourne le router
	// Autoriser tous les proxies dans le réseau Docker (développement uniquement)
	// En production, spécifier les IPs exactes des proxies
	router.SetTrustedProxies([]string{"0.0.0.0/0", "::/0"}) // Accepter tous les proxies pour le développement
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

