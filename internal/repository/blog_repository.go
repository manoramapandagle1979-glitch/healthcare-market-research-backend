package repository

import (
	"strings"

	"github.com/healthcare-market-research/backend/internal/domain/blog"
	"gorm.io/gorm"
)

type BlogRepository interface {
	Create(b *blog.Blog) error
	GetAll(query blog.GetBlogsQuery) ([]blog.Blog, int64, error)
	GetByID(id uint) (*blog.Blog, error)
	GetBySlug(slug string) (*blog.Blog, error)
	Update(id uint, updates map[string]interface{}) error
	Delete(id uint) error
	SubmitForReview(id uint) error
	Publish(id uint) error
	Unpublish(id uint) error
}

type blogRepository struct {
	db *gorm.DB
}

func NewBlogRepository(db *gorm.DB) BlogRepository {
	return &blogRepository{db: db}
}

func (r *blogRepository) Create(b *blog.Blog) error {
	return r.db.Create(b).Error
}

func (r *blogRepository) GetAll(query blog.GetBlogsQuery) ([]blog.Blog, int64, error) {
	var blogs []blog.Blog
	var total int64

	db := r.db.Model(&blog.Blog{})

	// Apply filters
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}

	if query.CategoryID != "" {
		db = db.Where("category_id = ?", query.CategoryID)
	}

	if query.Tags != "" {
		// Split comma-separated tags and search for any of them
		tags := strings.Split(query.Tags, ",")
		tagConditions := make([]string, 0, len(tags))
		tagValues := make([]interface{}, 0, len(tags))

		for _, tag := range tags {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tagConditions = append(tagConditions, "tags LIKE ?")
				tagValues = append(tagValues, "%"+tag+"%")
			}
		}

		if len(tagConditions) > 0 {
			db = db.Where(strings.Join(tagConditions, " OR "), tagValues...)
		}
	}

	if query.AuthorID != "" {
		db = db.Where("author_id = ?", query.AuthorID)
	}

	if query.Location != "" {
		db = db.Where("location ILIKE ?", "%"+query.Location+"%")
	}

	if query.Search != "" {
		searchPattern := "%" + query.Search + "%"
		db = db.Where("title ILIKE ? OR excerpt ILIKE ? OR content ILIKE ?", searchPattern, searchPattern, searchPattern)
	}

	// Count total before pagination
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	offset := (query.Page - 1) * query.Limit
	db = db.Order("created_at DESC").Offset(offset).Limit(query.Limit)

	// Fetch blogs
	if err := db.Find(&blogs).Error; err != nil {
		return nil, 0, err
	}

	return blogs, total, nil
}

func (r *blogRepository) GetByID(id uint) (*blog.Blog, error) {
	var b blog.Blog
	if err := r.db.First(&b, id).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *blogRepository) GetBySlug(slug string) (*blog.Blog, error) {
	var b blog.Blog
	if err := r.db.Where("slug = ?", slug).First(&b).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *blogRepository) Update(id uint, updates map[string]interface{}) error {
	return r.db.Model(&blog.Blog{}).Where("id = ?", id).Updates(updates).Error
}

func (r *blogRepository) Delete(id uint) error {
	return r.db.Delete(&blog.Blog{}, id).Error
}

func (r *blogRepository) SubmitForReview(id uint) error {
	return r.db.Model(&blog.Blog{}).Where("id = ?", id).Update("status", blog.StatusReview).Error
}

func (r *blogRepository) Publish(id uint) error {
	return r.db.Model(&blog.Blog{}).Where("id = ?", id).Update("status", blog.StatusPublished).Error
}

func (r *blogRepository) Unpublish(id uint) error {
	return r.db.Model(&blog.Blog{}).Where("id = ?", id).Update("status", blog.StatusDraft).Error
}
