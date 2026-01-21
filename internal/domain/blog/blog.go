package blog

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// BlogStatus represents the status of a blog post
type BlogStatus string

const (
	StatusDraft     BlogStatus = "draft"
	StatusReview    BlogStatus = "review"
	StatusPublished BlogStatus = "published"
)

// BlogMetadata contains SEO metadata for a blog post
type BlogMetadata struct {
	MetaTitle       string   `json:"metaTitle,omitempty"`
	MetaDescription string   `json:"metaDescription,omitempty"`
	Keywords        []string `json:"keywords,omitempty"`
}

// Value implements the driver.Valuer interface for GORM
func (m BlogMetadata) Value() (driver.Value, error) {
	return json.Marshal(m)
}

// Scan implements the sql.Scanner interface for GORM
func (m *BlogMetadata) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, &m)
}

// Blog represents a blog post in the database
type Blog struct {
	ID          uint         `json:"id" gorm:"primaryKey"`
	Title       string       `json:"title" gorm:"type:varchar(200);not null"`
	Slug        string       `json:"slug" gorm:"type:varchar(250);uniqueIndex;not null"`
	Excerpt     string       `json:"excerpt" gorm:"type:varchar(500);not null"`
	Content     string       `json:"content" gorm:"type:text;not null"`
	CategoryID  uint         `json:"categoryId" gorm:"not null;index"`
	Tags        string       `json:"tags" gorm:"type:varchar(500)"`
	AuthorID    uint         `json:"authorId" gorm:"not null;index"`
	Status      BlogStatus   `json:"status" gorm:"type:varchar(20);default:'draft';index"`
	PublishDate *time.Time   `json:"publishDate,omitempty" gorm:"index"`
	Location    string       `json:"location,omitempty" gorm:"type:varchar(255)"`
	Metadata    BlogMetadata `json:"metadata" gorm:"type:jsonb"`
	ReviewedBy  *uint        `json:"reviewedBy,omitempty" gorm:"index"`
	ReviewedAt  *time.Time   `json:"reviewedAt,omitempty"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`
}

// TableName specifies the table name for GORM
func (Blog) TableName() string {
	return "blogs"
}

// CreateBlogRequest is the request body for creating a new blog
type CreateBlogRequest struct {
	Title       string        `json:"title" validate:"required,min=10,max=200"`
	Excerpt     string        `json:"excerpt" validate:"required,min=50,max=500"`
	Content     string        `json:"content" validate:"required,min=100"`
	CategoryID  uint          `json:"categoryId" validate:"required"`
	Tags        string        `json:"tags"`
	AuthorID    uint          `json:"authorId" validate:"required"`
	Status      BlogStatus    `json:"status" validate:"required,oneof=draft review published"`
	PublishDate string        `json:"publishDate" validate:"required"`
	Location    string        `json:"location,omitempty"`
	Metadata    *BlogMetadata `json:"metadata,omitempty"`
}

// UpdateBlogRequest is the request body for updating a blog
type UpdateBlogRequest struct {
	Title       *string       `json:"title,omitempty" validate:"omitempty,min=10,max=200"`
	Excerpt     *string       `json:"excerpt,omitempty" validate:"omitempty,min=50,max=500"`
	Content     *string       `json:"content,omitempty" validate:"omitempty,min=100"`
	CategoryID  *uint         `json:"categoryId,omitempty"`
	Tags        *string       `json:"tags,omitempty"`
	AuthorID    *uint         `json:"authorId,omitempty"`
	Status      *BlogStatus   `json:"status,omitempty" validate:"omitempty,oneof=draft review published"`
	PublishDate *string       `json:"publishDate,omitempty"`
	Location    *string       `json:"location,omitempty"`
	Metadata    *BlogMetadata `json:"metadata,omitempty"`
}

// GetBlogsQuery represents query parameters for filtering blogs
type GetBlogsQuery struct {
	Status     string
	CategoryID string
	Tags       string
	AuthorID   string
	Location   string
	Search     string
	Page       int
	Limit      int
}

// BlogListResponse represents a list of blogs with pagination
type BlogListResponse struct {
	Blogs      []Blog `json:"blogs"`
	Total      int64  `json:"total"`
	Page       int    `json:"page"`
	Limit      int    `json:"limit"`
	TotalPages int    `json:"totalPages"`
}

// BlogResponse represents a single blog response
type BlogResponse struct {
	Blog Blog `json:"blog"`
}
