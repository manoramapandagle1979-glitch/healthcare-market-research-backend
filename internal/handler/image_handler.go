package handler

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/healthcare-market-research/backend/internal/repository"
	"github.com/healthcare-market-research/backend/internal/service"
	"github.com/healthcare-market-research/backend/pkg/response"
	"github.com/healthcare-market-research/backend/pkg/validation"
)

type ImageHandler struct {
	service service.ImageService
}

func NewImageHandler(service service.ImageService) *ImageHandler {
	return &ImageHandler{service: service}
}

type updateImageRequest struct {
	Title   *string  `json:"title,omitempty"`
	AltText *string  `json:"alt_text,omitempty"`
	Tags    *[]string `json:"tags,omitempty"`
}

// Upload godoc
// @Summary Upload image to media library
// @Description Upload an image to the general media library. Admin/editor only.
// @Tags Images
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param image formData file true "Image file (max 10MB)"
// @Param title formData string false "Image title"
// @Param alt_text formData string false "Alt text for accessibility"
// @Param tags formData string false "Comma-separated tags"
// @Success 201 {object} response.Response "Image uploaded"
// @Failure 400 {object} response.Response "Bad request"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /api/v1/images [post]
func (h *ImageHandler) Upload(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return response.Unauthorized(c, "User not authenticated")
	}

	file, err := c.FormFile("image")
	if err != nil {
		return response.BadRequest(c, "No image file provided")
	}

	if err := validation.ValidateImageFile(file); err != nil {
		return response.BadRequest(c, err.Error())
	}

	title := strings.TrimSpace(c.FormValue("title"))
	if title == "" {
		title = strings.TrimSuffix(file.Filename, getExt(file.Filename))
	}
	altText := strings.TrimSpace(c.FormValue("alt_text"))

	var tags []string
	if raw := strings.TrimSpace(c.FormValue("tags")); raw != "" {
		for _, t := range strings.Split(raw, ",") {
			if tag := strings.TrimSpace(t); tag != "" {
				tags = append(tags, tag)
			}
		}
	}

	img, err := h.service.Upload(file, title, altText, tags, userID)
	if err != nil {
		return response.InternalError(c, "Failed to upload image: "+err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(response.Response{
		Success: true,
		Data:    img,
	})
}

// List godoc
// @Summary List media library images
// @Description Paginated list of images with optional search and tag filter. Admin/editor only.
// @Tags Images
// @Security BearerAuth
// @Produce json
// @Param search query string false "Search by title or alt text"
// @Param tag query string false "Filter by tag"
// @Param page query int false "Page number (default 1)"
// @Param limit query int false "Items per page (default 20, max 100)"
// @Success 200 {object} response.Response "Image list with pagination"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden"
// @Router /api/v1/images [get]
func (h *ImageHandler) List(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if limit > 100 {
		limit = 100
	}

	filter := repository.ImageListFilter{
		Search: strings.TrimSpace(c.Query("search")),
		Tag:    strings.TrimSpace(c.Query("tag")),
		Page:   page,
		Limit:  limit,
	}

	images, total, err := h.service.List(filter)
	if err != nil {
		return response.InternalError(c, "Failed to fetch images")
	}

	return response.Success(c, fiber.Map{
		"images": images,
		"pagination": fiber.Map{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetByID godoc
// @Summary Get image by ID
// @Description Get a single media library image by ID. Admin/editor only.
// @Tags Images
// @Security BearerAuth
// @Produce json
// @Param id path int true "Image ID"
// @Success 200 {object} response.Response "Image details"
// @Failure 400 {object} response.Response "Bad request"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden"
// @Failure 404 {object} response.Response "Not found"
// @Router /api/v1/images/{id} [get]
func (h *ImageHandler) GetByID(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid image ID")
	}

	img, err := h.service.GetByID(uint(id))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return response.NotFound(c, "Image not found")
		}
		return response.InternalError(c, "Failed to fetch image")
	}

	return response.Success(c, img)
}

// Update godoc
// @Summary Update image metadata
// @Description Update title, alt text, or tags. Supports partial updates. Admin/editor only.
// @Tags Images
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Image ID"
// @Param body body updateImageRequest true "Fields to update"
// @Success 200 {object} response.Response "Updated image"
// @Failure 400 {object} response.Response "Bad request"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden"
// @Failure 404 {object} response.Response "Not found"
// @Router /api/v1/images/{id} [patch]
func (h *ImageHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid image ID")
	}

	var req updateImageRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body: "+err.Error())
	}

	if req.Title != nil {
		*req.Title = strings.TrimSpace(*req.Title)
	}
	if req.AltText != nil {
		*req.AltText = strings.TrimSpace(*req.AltText)
	}

	img, err := h.service.Update(uint(id), req.Title, req.AltText, req.Tags)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return response.NotFound(c, "Image not found")
		}
		return response.InternalError(c, "Failed to update image: "+err.Error())
	}

	return response.Success(c, img)
}

// Delete godoc
// @Summary Delete image
// @Description Permanently delete an image from both R2 and database. Admin only.
// @Tags Images
// @Security BearerAuth
// @Produce json
// @Param id path int true "Image ID"
// @Success 200 {object} response.Response "Deleted"
// @Failure 400 {object} response.Response "Bad request"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden"
// @Failure 404 {object} response.Response "Not found"
// @Router /api/v1/images/{id} [delete]
func (h *ImageHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid image ID")
	}

	if err := h.service.Delete(uint(id)); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return response.NotFound(c, "Image not found")
		}
		return response.InternalError(c, "Failed to delete image: "+err.Error())
	}

	return response.Success(c, fiber.Map{"message": "Image deleted successfully"})
}

func getExt(filename string) string {
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			return filename[i:]
		}
	}
	return ""
}
