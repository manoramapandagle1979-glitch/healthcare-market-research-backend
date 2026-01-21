package service

import (
	"fmt"
	"time"

	"github.com/healthcare-market-research/backend/internal/cache"
	"github.com/healthcare-market-research/backend/internal/domain/press_release"
	"github.com/healthcare-market-research/backend/internal/repository"
	"github.com/gosimple/slug"
)

type PressReleaseService interface {
	Create(req *press_release.CreatePressReleaseRequest) (*press_release.PressRelease, error)
	GetAll(query press_release.GetPressReleasesQuery) ([]press_release.PressRelease, int64, error)
	GetByID(id uint) (*press_release.PressRelease, error)
	Update(id uint, req *press_release.UpdatePressReleaseRequest) (*press_release.PressRelease, error)
	Delete(id uint) error
	SubmitForReview(id uint) (*press_release.PressRelease, error)
	Publish(id uint) (*press_release.PressRelease, error)
	Unpublish(id uint) (*press_release.PressRelease, error)
}

type pressReleaseService struct {
	repo repository.PressReleaseRepository
}

func NewPressReleaseService(repo repository.PressReleaseRepository) PressReleaseService {
	return &pressReleaseService{repo: repo}
}

func (s *pressReleaseService) Create(req *press_release.CreatePressReleaseRequest) (*press_release.PressRelease, error) {
	// Validate status
	if req.Status != press_release.StatusDraft && req.Status != press_release.StatusReview && req.Status != press_release.StatusPublished {
		return nil, fmt.Errorf("invalid status: must be 'draft', 'review', or 'published'")
	}

	// Generate slug from title
	prSlug := slug.Make(req.Title)

	// Parse publish date
	publishDate, err := time.Parse(time.RFC3339, req.PublishDate)
	if err != nil {
		return nil, fmt.Errorf("invalid publishDate format: must be ISO 8601 (RFC3339)")
	}

	// Create press release
	pr := &press_release.PressRelease{
		Title:       req.Title,
		Slug:        prSlug,
		Excerpt:     req.Excerpt,
		Content:     req.Content,
		CategoryID:  req.CategoryID,
		Tags:        req.Tags,
		AuthorID:    req.AuthorID,
		Status:      req.Status,
		PublishDate: &publishDate,
		Location:    req.Location,
	}

	// Set metadata if provided
	if req.Metadata != nil {
		pr.Metadata = *req.Metadata
	}

	if err := s.repo.Create(pr); err != nil {
		return nil, err
	}

	// Invalidate caches
	cache.DeletePattern("press_releases:*")

	return pr, nil
}

func (s *pressReleaseService) GetAll(query press_release.GetPressReleasesQuery) ([]press_release.PressRelease, int64, error) {
	// Only cache non-filtered, non-search queries
	shouldCache := query.Status == "" && query.CategoryID == "" && query.Tags == "" &&
		query.AuthorID == "" && query.Location == "" && query.Search == ""

	if shouldCache {
		cacheKey := fmt.Sprintf("press_releases:list:%d:%d", query.Page, query.Limit)

		type result struct {
			PressReleases []press_release.PressRelease `json:"pressReleases"`
			Total         int64                        `json:"total"`
		}

		var res result

		err := cache.GetOrSet(cacheKey, &res, 5*time.Minute, func() (interface{}, error) {
			pressReleases, total, err := s.repo.GetAll(query)
			if err != nil {
				return nil, err
			}
			return result{PressReleases: pressReleases, Total: total}, nil
		})

		if err != nil {
			return nil, 0, err
		}

		return res.PressReleases, res.Total, nil
	}

	// Don't cache filtered/search results
	return s.repo.GetAll(query)
}

func (s *pressReleaseService) GetByID(id uint) (*press_release.PressRelease, error) {
	cacheKey := fmt.Sprintf("press_release:id:%d", id)

	var pr press_release.PressRelease

	err := cache.GetOrSet(cacheKey, &pr, 10*time.Minute, func() (interface{}, error) {
		return s.repo.GetByID(id)
	})

	if err != nil {
		return nil, err
	}

	return &pr, nil
}

func (s *pressReleaseService) Update(id uint, req *press_release.UpdatePressReleaseRequest) (*press_release.PressRelease, error) {
	// Check if press release exists
	_, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Build updates map
	updates := make(map[string]interface{})

	if req.Title != nil {
		updates["title"] = *req.Title
		// Update slug if title changed
		updates["slug"] = slug.Make(*req.Title)
	}

	if req.Excerpt != nil {
		updates["excerpt"] = *req.Excerpt
	}

	if req.Content != nil {
		updates["content"] = *req.Content
	}

	if req.CategoryID != nil {
		updates["category_id"] = *req.CategoryID
	}

	if req.Tags != nil {
		updates["tags"] = *req.Tags
	}

	if req.AuthorID != nil {
		updates["author_id"] = *req.AuthorID
	}

	if req.Status != nil {
		// Validate status
		if *req.Status != press_release.StatusDraft && *req.Status != press_release.StatusReview && *req.Status != press_release.StatusPublished {
			return nil, fmt.Errorf("invalid status: must be 'draft', 'review', or 'published'")
		}
		updates["status"] = *req.Status
	}

	if req.PublishDate != nil {
		publishDate, err := time.Parse(time.RFC3339, *req.PublishDate)
		if err != nil {
			return nil, fmt.Errorf("invalid publishDate format: must be ISO 8601 (RFC3339)")
		}
		updates["publish_date"] = publishDate
	}

	if req.Location != nil {
		updates["location"] = *req.Location
	}

	if req.Metadata != nil {
		updates["metadata"] = *req.Metadata
	}

	// Update press release
	if err := s.repo.Update(id, updates); err != nil {
		return nil, err
	}

	// Invalidate caches
	cache.DeletePattern("press_releases:*")
	cache.Delete(fmt.Sprintf("press_release:id:%d", id))

	// Fetch and return updated press release
	return s.repo.GetByID(id)
}

func (s *pressReleaseService) Delete(id uint) error {
	// Check if press release exists
	_, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	if err := s.repo.Delete(id); err != nil {
		return err
	}

	// Invalidate caches
	cache.DeletePattern("press_releases:*")
	cache.Delete(fmt.Sprintf("press_release:id:%d", id))

	return nil
}

func (s *pressReleaseService) SubmitForReview(id uint) (*press_release.PressRelease, error) {
	// Check if press release exists
	existingPR, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Check if already in review or published
	if existingPR.Status == press_release.StatusReview {
		return nil, fmt.Errorf("press release is already in review")
	}
	if existingPR.Status == press_release.StatusPublished {
		return nil, fmt.Errorf("cannot submit published press release for review")
	}

	if err := s.repo.SubmitForReview(id); err != nil {
		return nil, err
	}

	// Invalidate caches
	cache.DeletePattern("press_releases:*")
	cache.Delete(fmt.Sprintf("press_release:id:%d", id))

	// Fetch and return updated press release
	return s.repo.GetByID(id)
}

func (s *pressReleaseService) Publish(id uint) (*press_release.PressRelease, error) {
	// Check if press release exists
	existingPR, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Check if already published
	if existingPR.Status == press_release.StatusPublished {
		return nil, fmt.Errorf("press release is already published")
	}

	if err := s.repo.Publish(id); err != nil {
		return nil, err
	}

	// Invalidate caches
	cache.DeletePattern("press_releases:*")
	cache.Delete(fmt.Sprintf("press_release:id:%d", id))

	// Fetch and return updated press release
	return s.repo.GetByID(id)
}

func (s *pressReleaseService) Unpublish(id uint) (*press_release.PressRelease, error) {
	// Check if press release exists
	existingPR, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Check if currently published
	if existingPR.Status != press_release.StatusPublished {
		return nil, fmt.Errorf("press release is not published")
	}

	if err := s.repo.Unpublish(id); err != nil {
		return nil, err
	}

	// Invalidate caches
	cache.DeletePattern("press_releases:*")
	cache.Delete(fmt.Sprintf("press_release:id:%d", id))

	// Fetch and return updated press release
	return s.repo.GetByID(id)
}
