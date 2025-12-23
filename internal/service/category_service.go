package service

import (
	"fmt"
	"time"

	"github.com/healthcare-market-research/backend/internal/cache"
	"github.com/healthcare-market-research/backend/internal/domain/category"
	"github.com/healthcare-market-research/backend/internal/repository"
)

type CategoryService interface {
	GetAll(page, limit int) ([]category.Category, int64, error)
	GetBySlug(slug string) (*category.Category, error)
}

type categoryService struct {
	repo repository.CategoryRepository
}

func NewCategoryService(repo repository.CategoryRepository) CategoryService {
	return &categoryService{repo: repo}
}

func (s *categoryService) GetAll(page, limit int) ([]category.Category, int64, error) {
	cacheKey := fmt.Sprintf("categories:list:%d:%d", page, limit)
	totalKey := "categories:total"

	type result struct {
		Categories []category.Category `json:"categories"`
		Total      int64               `json:"total"`
	}

	var res result

	// Use singleflight-protected cache-aside pattern
	err := cache.GetOrSet(cacheKey, &res, 10*time.Minute, func() (interface{}, error) {
		categories, total, err := s.repo.GetAll(page, limit)
		if err != nil {
			return nil, err
		}

		// Also cache total separately
		cache.Set(totalKey, total, 10*time.Minute)

		return result{Categories: categories, Total: total}, nil
	})

	if err != nil {
		return nil, 0, err
	}

	return res.Categories, res.Total, nil
}

func (s *categoryService) GetBySlug(slug string) (*category.Category, error) {
	cacheKey := fmt.Sprintf("category:slug:%s", slug)

	var cat category.Category

	// Use singleflight-protected cache-aside pattern
	err := cache.GetOrSet(cacheKey, &cat, 30*time.Minute, func() (interface{}, error) {
		return s.repo.GetBySlug(slug)
	})

	if err != nil {
		return nil, err
	}

	return &cat, nil
}
