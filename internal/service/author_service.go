package service

import (
	"fmt"
	"time"

	"github.com/healthcare-market-research/backend/internal/cache"
	"github.com/healthcare-market-research/backend/internal/domain/author"
	"github.com/healthcare-market-research/backend/internal/repository"
)

type AuthorService interface {
	GetAll(page, limit int, search string) ([]author.Author, int64, error)
	GetByID(id uint) (*author.Author, error)
	Create(author *author.Author) error
	Update(id uint, author *author.Author) error
	Delete(id uint) error
}

type authorService struct {
	repo repository.AuthorRepository
}

func NewAuthorService(repo repository.AuthorRepository) AuthorService {
	return &authorService{repo: repo}
}

func (s *authorService) GetAll(page, limit int, search string) ([]author.Author, int64, error) {
	// Cache only non-search queries
	if search == "" {
		cacheKey := fmt.Sprintf("authors:list:%d:%d", page, limit)

		type result struct {
			Authors []author.Author `json:"authors"`
			Total   int64           `json:"total"`
		}

		var res result

		// Use cache-aside pattern
		err := cache.GetOrSet(cacheKey, &res, 10*time.Minute, func() (interface{}, error) {
			authors, total, err := s.repo.GetAll(page, limit, search)
			if err != nil {
				return nil, err
			}
			return result{Authors: authors, Total: total}, nil
		})

		if err != nil {
			return nil, 0, err
		}

		return res.Authors, res.Total, nil
	}

	// Don't cache search results
	return s.repo.GetAll(page, limit, search)
}

func (s *authorService) GetByID(id uint) (*author.Author, error) {
	cacheKey := fmt.Sprintf("author:id:%d", id)

	var auth author.Author

	// Use cache-aside pattern
	err := cache.GetOrSet(cacheKey, &auth, 30*time.Minute, func() (interface{}, error) {
		return s.repo.GetByID(id)
	})

	if err != nil {
		return nil, err
	}

	return &auth, nil
}

func (s *authorService) Create(auth *author.Author) error {
	err := s.repo.Create(auth)
	if err != nil {
		return err
	}

	// Invalidate list caches
	cache.DeletePattern("authors:list:*")

	return nil
}

func (s *authorService) Update(id uint, auth *author.Author) error {
	// Check if author exists
	_, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	auth.ID = id
	err = s.repo.Update(auth)
	if err != nil {
		return err
	}

	// Invalidate caches
	cache.DeletePattern("authors:list:*")
	cache.Delete(fmt.Sprintf("author:id:%d", id))

	return nil
}

func (s *authorService) Delete(id uint) error {
	// Check if author is referenced in any reports
	isReferenced, err := s.repo.IsReferencedInReports(id)
	if err != nil {
		return err
	}

	if isReferenced {
		return fmt.Errorf("cannot delete author: author is referenced in one or more reports")
	}

	err = s.repo.Delete(id)
	if err != nil {
		return err
	}

	// Invalidate caches
	cache.DeletePattern("authors:list:*")
	cache.Delete(fmt.Sprintf("author:id:%d", id))

	return nil
}
