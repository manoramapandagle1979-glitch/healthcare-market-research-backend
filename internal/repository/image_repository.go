package repository

import (
	"github.com/healthcare-market-research/backend/internal/domain/image"
	"gorm.io/gorm"
)

type ImageListFilter struct {
	Search string
	Tag    string
	Page   int
	Limit  int
}

type ImageRepository interface {
	Create(img *image.Image) error
	FindByID(id uint) (*image.Image, error)
	List(filter ImageListFilter) ([]image.Image, int64, error)
	Update(img *image.Image) error
	Delete(id uint) error
}

type imageRepository struct {
	db *gorm.DB
}

func NewImageRepository(db *gorm.DB) ImageRepository {
	return &imageRepository{db: db}
}

func (r *imageRepository) Create(img *image.Image) error {
	return r.db.Create(img).Error
}

func (r *imageRepository) FindByID(id uint) (*image.Image, error) {
	var img image.Image
	if err := r.db.First(&img, id).Error; err != nil {
		return nil, err
	}
	return &img, nil
}

func (r *imageRepository) List(filter ImageListFilter) ([]image.Image, int64, error) {
	query := r.db.Model(&image.Image{})

	if filter.Search != "" {
		query = query.Where("title ILIKE ? OR alt_text ILIKE ?", "%"+filter.Search+"%", "%"+filter.Search+"%")
	}

	if filter.Tag != "" {
		query = query.Where("tags @> ?", `["`+filter.Tag+`"]`)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	offset := (filter.Page - 1) * filter.Limit

	var images []image.Image
	if err := query.Order("created_at DESC").Limit(filter.Limit).Offset(offset).Find(&images).Error; err != nil {
		return nil, 0, err
	}

	return images, total, nil
}

func (r *imageRepository) Update(img *image.Image) error {
	return r.db.Save(img).Error
}

func (r *imageRepository) Delete(id uint) error {
	return r.db.Delete(&image.Image{}, id).Error
}
