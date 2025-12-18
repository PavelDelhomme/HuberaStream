package database

import (
	"fmt"
	"log"

	"streammake/internal/config"
	"streammake/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Initialize(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		cfg.DBHost,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
		cfg.DBPort,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error), // Ne logger que les erreurs (pas les "record not found")
	})

	if err != nil {
		return nil, fmt.Errorf("erreur lors de la connexion à la base de données: %w", err)
	}

	// Auto-migrer les modèles
	if err := autoMigrate(db); err != nil {
		return nil, fmt.Errorf("erreur lors de la migration: %w", err)
	}

	log.Println("Base de données initialisée avec succès")
	return db, nil
}

func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.Video{},
		&models.Playlist{},
		&models.PlaylistVideo{},
		&models.WatchHistory{},
		&models.VideoFormat{},
		&models.Category{},
		&models.SubCategory{},
		&models.Torrent{},
		&models.TorrentConfig{},
		&models.UploadSession{},
		&models.UploadChunk{},
	)
}
