package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CategoryType string

const (
	CategoryTypeAnime CategoryType = "anime"
	CategoryTypeFilm  CategoryType = "film"
	CategoryTypeSeries CategoryType = "series"
	CategoryTypeOther CategoryType = "other"
)

type Category struct {
	ID          uuid.UUID   `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name        string      `gorm:"uniqueIndex;not null" json:"name"`
	Type        CategoryType `gorm:"type:varchar(50);not null" json:"type"`
	Description string      `gorm:"type:text" json:"description"`
	Icon        string      `json:"icon"` // URL ou nom d'icône
	IsActive    bool        `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	SubCategories []SubCategory `gorm:"foreignKey:CategoryID" json:"sub_categories,omitempty"`
	Videos        []Video       `gorm:"foreignKey:CategoryID" json:"videos,omitempty"`
}

type SubCategory struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	CategoryID  uuid.UUID `gorm:"type:uuid;not null" json:"category_id"`
	Category    Category  `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Name        string    `gorm:"not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	Videos []Video `gorm:"foreignKey:SubCategoryID" json:"videos,omitempty"`
}

