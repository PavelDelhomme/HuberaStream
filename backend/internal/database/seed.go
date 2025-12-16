package database

import (
	"log"

	"streammake/internal/config"
	"streammake/internal/models"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func SeedDatabase(db *gorm.DB, cfg *config.Config) error {
	// Créer l'utilisateur admin par défaut
	adminEmail := cfg.AdminEmail
	adminPassword := cfg.AdminPassword

	if adminEmail == "" {
		adminEmail = "dev@delhomme.ovh"
	}
	if adminPassword == "" {
		adminPassword = "AdminStreamMake2024!"
	}

	var adminUser models.User
	result := db.Where("email = ?", adminEmail).First(&adminUser)

	// Hasher le mot de passe
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	if result.Error == gorm.ErrRecordNotFound {
		// Créer l'utilisateur admin
		adminUser = models.User{
			ID:       uuid.New(),
			Username: "admin",
			Email:    adminEmail,
			Password: string(hashedPassword),
			Role:     models.RoleAdmin,
			IsActive: true,
		}

		if err := db.Create(&adminUser).Error; err != nil {
			return err
		}

		log.Printf("Utilisateur admin créé: %s", adminEmail)
	} else {
		// Mettre à jour le mot de passe et s'assurer que le rôle et l'état sont corrects
		adminUser.Password = string(hashedPassword)
		adminUser.Role = models.RoleAdmin
		adminUser.IsActive = true
		adminUser.Username = "admin" // S'assurer que le username est correct

		if err := db.Save(&adminUser).Error; err != nil {
			log.Printf("Erreur lors de la mise à jour de l'utilisateur admin: %v", err)
			return err
		}

		log.Printf("Utilisateur admin mis à jour (mot de passe réinitialisé): %s", adminEmail)
	}

	// Créer les catégories par défaut
	categories := []models.Category{
		{
			ID:          uuid.New(),
			Name:        "Anime",
			Type:        models.CategoryTypeAnime,
			Description: "Animations japonaises",
			Icon:        "🎌",
			IsActive:    true,
		},
		{
			ID:          uuid.New(),
			Name:        "Films",
			Type:        models.CategoryTypeFilm,
			Description: "Longs métrages",
			Icon:        "🎬",
			IsActive:    true,
		},
		{
			ID:          uuid.New(),
			Name:        "Séries",
			Type:        models.CategoryTypeSeries,
			Description: "Séries télévisées",
			Icon:        "📺",
			IsActive:    true,
		},
	}

	for _, category := range categories {
		var existingCategory models.Category
		if err := db.Where("name = ?", category.Name).First(&existingCategory).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := db.Create(&category).Error; err != nil {
					log.Printf("Erreur lors de la création de la catégorie %s: %v", category.Name, err)
				} else {
					log.Printf("Catégorie créée: %s", category.Name)
				}
			}
		}
	}

	// Créer des sous-catégories pour les animes
	var animeCategory models.Category
	if err := db.Where("name = ?", "Anime").First(&animeCategory).Error; err == nil {
		animeSubCategories := []models.SubCategory{
			{ID: uuid.New(), CategoryID: animeCategory.ID, Name: "Shonen", Description: "Anime shonen", IsActive: true},
			{ID: uuid.New(), CategoryID: animeCategory.ID, Name: "Seinen", Description: "Anime seinen", IsActive: true},
			{ID: uuid.New(), CategoryID: animeCategory.ID, Name: "Shojo", Description: "Anime shojo", IsActive: true},
			{ID: uuid.New(), CategoryID: animeCategory.ID, Name: "Mecha", Description: "Anime mecha", IsActive: true},
			{ID: uuid.New(), CategoryID: animeCategory.ID, Name: "Isekai", Description: "Anime isekai", IsActive: true},
		}

		for _, subCat := range animeSubCategories {
			var existing models.SubCategory
			if err := db.Where("category_id = ? AND name = ?", animeCategory.ID, subCat.Name).First(&existing).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					db.Create(&subCat)
				}
			}
		}
	}

	// Créer des sous-catégories pour les films
	var filmCategory models.Category
	if err := db.Where("name = ?", "Films").First(&filmCategory).Error; err == nil {
		filmSubCategories := []models.SubCategory{
			{ID: uuid.New(), CategoryID: filmCategory.ID, Name: "Action", Description: "Films d'action", IsActive: true},
			{ID: uuid.New(), CategoryID: filmCategory.ID, Name: "Comédie", Description: "Films comiques", IsActive: true},
			{ID: uuid.New(), CategoryID: filmCategory.ID, Name: "Drame", Description: "Films dramatiques", IsActive: true},
			{ID: uuid.New(), CategoryID: filmCategory.ID, Name: "Science-Fiction", Description: "Films de science-fiction", IsActive: true},
			{ID: uuid.New(), CategoryID: filmCategory.ID, Name: "Horreur", Description: "Films d'horreur", IsActive: true},
			{ID: uuid.New(), CategoryID: filmCategory.ID, Name: "Thriller", Description: "Films à suspense", IsActive: true},
		}

		for _, subCat := range filmSubCategories {
			var existing models.SubCategory
			if err := db.Where("category_id = ? AND name = ?", filmCategory.ID, subCat.Name).First(&existing).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					db.Create(&subCat)
				}
			}
		}
	}

	return nil
}
