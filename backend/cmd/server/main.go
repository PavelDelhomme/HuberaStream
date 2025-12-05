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

	// CRITIQUE: Le problème 403 persiste même sans SetTrustedProxies
	// Le blocage se produit AVANT les middlewares (quelques microsecondes)
	// 
	// SOLUTION EXPÉRIMENTALE: Ne PAS configurer SetTrustedProxies du tout
	// et utiliser le mode Release de Gin qui pourrait avoir un comportement différent
	
	// Ne PAS appeler SetTrustedProxies - laisser le comportement par défaut
	// En mode développement, Gin devrait accepter toutes les requêtes
	
	log.Printf("Configuration des proxies: DÉSACTIVÉE (comportement par défaut)")

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

