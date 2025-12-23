package category

import (
	"time"
)

type Category struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"type:varchar(255);not null"`
	Slug        string    `json:"slug" gorm:"type:varchar(255);uniqueIndex;not null"`
	Description string    `json:"description" gorm:"type:text"`
	ImageURL    string    `json:"image_url" gorm:"type:varchar(500)"`
	IsActive    bool      `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type SubCategory struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	CategoryID  uint      `json:"category_id" gorm:"index;not null"`
	Name        string    `json:"name" gorm:"type:varchar(255);not null"`
	Slug        string    `json:"slug" gorm:"type:varchar(255);uniqueIndex;not null"`
	Description string    `json:"description" gorm:"type:text"`
	IsActive    bool      `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Category    *Category `json:"category,omitempty" gorm:"foreignKey:CategoryID"`
}

type MarketSegment struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	SubCategoryID uint      `json:"sub_category_id" gorm:"index;not null"`
	Name          string    `json:"name" gorm:"type:varchar(255);not null"`
	Slug          string    `json:"slug" gorm:"type:varchar(255);uniqueIndex;not null"`
	Description   string    `json:"description" gorm:"type:text"`
	IsActive      bool      `json:"is_active" gorm:"default:true"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	SubCategory   *SubCategory `json:"sub_category,omitempty" gorm:"foreignKey:SubCategoryID"`
}
