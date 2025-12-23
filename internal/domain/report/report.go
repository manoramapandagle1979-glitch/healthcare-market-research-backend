package report

import (
	"time"
)

type Report struct {
	ID              uint       `json:"id" gorm:"primaryKey"`
	CategoryID      uint       `json:"category_id" gorm:"index;not null"`
	SubCategoryID   *uint      `json:"sub_category_id,omitempty" gorm:"index"`
	MarketSegmentID *uint      `json:"market_segment_id,omitempty" gorm:"index"`
	Title           string     `json:"title" gorm:"type:varchar(500);not null"`
	Slug            string     `json:"slug" gorm:"type:varchar(500);uniqueIndex;not null"`
	Description     string     `json:"description" gorm:"type:text"`
	Summary         string     `json:"summary" gorm:"type:text"`
	ThumbnailURL    string     `json:"thumbnail_url" gorm:"type:varchar(500)"`
	Price           float64    `json:"price" gorm:"type:decimal(10,2)"`
	Currency        string     `json:"currency" gorm:"type:varchar(3);default:'USD'"`
	PageCount       int        `json:"page_count" gorm:"default:0"`
	PublishedAt     *time.Time `json:"published_at" gorm:"index"`
	IsPublished     bool       `json:"is_published" gorm:"default:false;index"`
	IsFeatured      bool       `json:"is_featured" gorm:"default:false"`
	ViewCount       int        `json:"view_count" gorm:"default:0"`
	DownloadCount   int        `json:"download_count" gorm:"default:0"`
	MetaTitle       string     `json:"meta_title" gorm:"type:varchar(255)"`
	MetaDescription string     `json:"meta_description" gorm:"type:varchar(500)"`
	MetaKeywords    string     `json:"meta_keywords" gorm:"type:varchar(500)"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type ChartMetadata struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	ReportID    uint      `json:"report_id" gorm:"index;not null"`
	Title       string    `json:"title" gorm:"type:varchar(500);not null"`
	ChartType   string    `json:"chart_type" gorm:"type:varchar(50)"` // bar, line, pie, etc.
	Description string    `json:"description" gorm:"type:text"`
	DataPoints  int       `json:"data_points" gorm:"default:0"`
	Order       int       `json:"order" gorm:"default:0"`
	IsActive    bool      `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Report      *Report   `json:"report,omitempty" gorm:"foreignKey:ReportID"`
}

type ReportWithRelations struct {
	Report
	CategoryName      string           `json:"category_name,omitempty"`
	SubCategoryName   string           `json:"sub_category_name,omitempty"`
	MarketSegmentName string           `json:"market_segment_name,omitempty"`
	Charts            []ChartMetadata  `json:"charts,omitempty"`
}
