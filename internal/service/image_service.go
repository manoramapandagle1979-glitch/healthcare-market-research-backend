package service

import (
	"fmt"
	"log"
	"mime/multipart"
	"strings"

	"github.com/healthcare-market-research/backend/internal/domain/image"
	"github.com/healthcare-market-research/backend/internal/repository"
)

type ImageListFilter = repository.ImageListFilter

type ImageService interface {
	Upload(file *multipart.FileHeader, title, altText string, tags []string, uploadedBy uint) (*image.Image, error)
	GetByID(id uint) (*image.Image, error)
	List(filter ImageListFilter) ([]image.Image, int64, error)
	Update(id uint, title, altText *string, tags *[]string) (*image.Image, error)
	Delete(id uint) error
}

type imageService struct {
	repo              repository.ImageRepository
	cloudflareService CloudflareImagesService
}

func NewImageService(repo repository.ImageRepository, cloudflareService CloudflareImagesService) ImageService {
	return &imageService{repo: repo, cloudflareService: cloudflareService}
}

func (s *imageService) Upload(file *multipart.FileHeader, title, altText string, tags []string, uploadedBy uint) (*image.Image, error) {
	metadata := map[string]string{
		"type":        "media_library",
		"uploaded_by": fmt.Sprintf("%d", uploadedBy),
	}

	imageURL, err := s.cloudflareService.Upload(file, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to upload image: %w", err)
	}

	objectKey, _ := s.cloudflareService.ExtractImageID(imageURL)

	if tags == nil {
		tags = []string{}
	}

	img := &image.Image{
		Title:      title,
		AltText:    altText,
		URL:        imageURL,
		ObjectKey:  objectKey,
		Size:       file.Size,
		MimeType:   strings.ToLower(file.Header.Get("Content-Type")),
		Tags:       image.Tags(tags),
		UploadedBy: &uploadedBy,
	}

	if err := s.repo.Create(img); err != nil {
		if deleteErr := s.cloudflareService.Delete(imageURL); deleteErr != nil {
			log.Printf("Warning: failed to rollback image upload: %v", deleteErr)
		}
		return nil, fmt.Errorf("failed to save image record: %w", err)
	}

	return img, nil
}

func (s *imageService) GetByID(id uint) (*image.Image, error) {
	img, err := s.repo.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("image not found: %w", err)
	}
	return img, nil
}

func (s *imageService) List(filter ImageListFilter) ([]image.Image, int64, error) {
	return s.repo.List(filter)
}

func (s *imageService) Update(id uint, title, altText *string, tags *[]string) (*image.Image, error) {
	img, err := s.repo.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("image not found: %w", err)
	}

	if title != nil {
		img.Title = *title
	}
	if altText != nil {
		img.AltText = *altText
	}
	if tags != nil {
		img.Tags = image.Tags(*tags)
	}

	if err := s.repo.Update(img); err != nil {
		return nil, fmt.Errorf("failed to update image: %w", err)
	}

	return img, nil
}

func (s *imageService) Delete(id uint) error {
	img, err := s.repo.FindByID(id)
	if err != nil {
		return fmt.Errorf("image not found: %w", err)
	}

	if err := s.cloudflareService.Delete(img.URL); err != nil {
		log.Printf("Warning: failed to delete image from R2 for image %d: %v", id, err)
	}

	if err := s.repo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete image record: %w", err)
	}

	return nil
}
