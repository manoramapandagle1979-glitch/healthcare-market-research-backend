package handler

import (
	"math"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/healthcare-market-research/backend/internal/domain/press_release"
	"github.com/healthcare-market-research/backend/internal/service"
	"github.com/healthcare-market-research/backend/pkg/response"
)

type PressReleaseHandler struct {
	service service.PressReleaseService
}

func NewPressReleaseHandler(service service.PressReleaseService) *PressReleaseHandler {
	return &PressReleaseHandler{service: service}
}

// Create godoc
// @Summary Create press release
// @Description Create a new press release
// @Tags PressReleases
// @Accept json
// @Produce json
// @Param pressRelease body press_release.CreatePressReleaseRequest true "Press release data"
// @Success 201 {object} press_release.PressReleaseResponse "Press release created successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid input or validation error"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/press-releases [post]
func (h *PressReleaseHandler) Create(c *fiber.Ctx) error {
	var req press_release.CreatePressReleaseRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body: "+err.Error())
	}

	pr, err := h.service.Create(&req)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(press_release.PressReleaseResponse{PressRelease: *pr})
}

// GetAll godoc
// @Summary Get all press releases
// @Description Get a paginated list of press releases with optional filtering
// @Tags PressReleases
// @Accept json
// @Produce json
// @Param status query string false "Filter by status: draft, review, published"
// @Param categoryId query int false "Filter by category ID"
// @Param tags query string false "Filter by tags (comma-separated)"
// @Param authorId query int false "Filter by author ID"
// @Param location query string false "Filter by location"
// @Param search query string false "Search in title, excerpt, content"
// @Param page query int false "Page number (default: 1, min: 1)"
// @Param limit query int false "Items per page (default: 20, max: 100)"
// @Success 200 {object} press_release.PressReleaseListResponse "List of press releases with pagination"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/press-releases [get]
func (h *PressReleaseHandler) GetAll(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	query := press_release.GetPressReleasesQuery{
		Status:     c.Query("status", ""),
		CategoryID: c.Query("categoryId", ""),
		Tags:       c.Query("tags", ""),
		AuthorID:   c.Query("authorId", ""),
		Location:   c.Query("location", ""),
		Search:     c.Query("search", ""),
		Page:       page,
		Limit:      limit,
	}

	pressReleases, total, err := h.service.GetAll(query)
	if err != nil {
		return response.InternalError(c, "Failed to fetch press releases")
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	return c.JSON(press_release.PressReleaseListResponse{
		PressReleases: pressReleases,
		Total:         total,
		Page:          page,
		Limit:         limit,
		TotalPages:    totalPages,
	})
}

// GetByID godoc
// @Summary Get press release by ID
// @Description Get a single press release by ID
// @Tags PressReleases
// @Accept json
// @Produce json
// @Param id path int true "Press release ID"
// @Success 200 {object} press_release.PressReleaseResponse "Press release details"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID"
// @Failure 404 {object} response.Response{error=string} "Press release not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/press-releases/{id} [get]
func (h *PressReleaseHandler) GetByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return response.BadRequest(c, "Press release ID is required")
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid press release ID format")
	}

	pr, err := h.service.GetByID(uint(id))
	if err != nil {
		return response.NotFound(c, "Press release not found")
	}

	return c.JSON(press_release.PressReleaseResponse{PressRelease: *pr})
}

// Update godoc
// @Summary Update press release
// @Description Update an existing press release
// @Tags PressReleases
// @Accept json
// @Produce json
// @Param id path int true "Press release ID"
// @Param pressRelease body press_release.UpdatePressReleaseRequest true "Press release update data"
// @Success 200 {object} press_release.PressReleaseResponse "Press release updated successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid input or validation error"
// @Failure 404 {object} response.Response{error=string} "Press release not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/press-releases/{id} [put]
func (h *PressReleaseHandler) Update(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return response.BadRequest(c, "Press release ID is required")
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid press release ID format")
	}

	var req press_release.UpdatePressReleaseRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body: "+err.Error())
	}

	pr, err := h.service.Update(uint(id), &req)
	if err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Press release not found")
		}
		return response.BadRequest(c, err.Error())
	}

	return c.JSON(press_release.PressReleaseResponse{PressRelease: *pr})
}

// Delete godoc
// @Summary Delete press release
// @Description Delete a press release by ID
// @Tags PressReleases
// @Accept json
// @Produce json
// @Param id path int true "Press release ID"
// @Success 204 "Press release deleted successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID"
// @Failure 404 {object} response.Response{error=string} "Press release not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/press-releases/{id} [delete]
func (h *PressReleaseHandler) Delete(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return response.BadRequest(c, "Press release ID is required")
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid press release ID format")
	}

	if err := h.service.Delete(uint(id)); err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Press release not found")
		}
		return response.InternalError(c, "Failed to delete press release")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// SubmitForReview godoc
// @Summary Submit press release for review
// @Description Change press release status from draft to review
// @Tags PressReleases
// @Accept json
// @Produce json
// @Param id path int true "Press release ID"
// @Success 200 {object} press_release.PressReleaseResponse "Press release submitted for review successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID or press release cannot be submitted"
// @Failure 404 {object} response.Response{error=string} "Press release not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/press-releases/{id}/submit-review [patch]
func (h *PressReleaseHandler) SubmitForReview(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return response.BadRequest(c, "Press release ID is required")
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid press release ID format")
	}

	pr, err := h.service.SubmitForReview(uint(id))
	if err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Press release not found")
		}
		return response.BadRequest(c, err.Error())
	}

	return c.JSON(press_release.PressReleaseResponse{PressRelease: *pr})
}

// Publish godoc
// @Summary Publish press release
// @Description Change press release status to published
// @Tags PressReleases
// @Accept json
// @Produce json
// @Param id path int true "Press release ID"
// @Success 200 {object} press_release.PressReleaseResponse "Press release published successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID or press release cannot be published"
// @Failure 404 {object} response.Response{error=string} "Press release not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/press-releases/{id}/publish [patch]
func (h *PressReleaseHandler) Publish(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return response.BadRequest(c, "Press release ID is required")
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid press release ID format")
	}

	pr, err := h.service.Publish(uint(id))
	if err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Press release not found")
		}
		return response.BadRequest(c, err.Error())
	}

	return c.JSON(press_release.PressReleaseResponse{PressRelease: *pr})
}

// Unpublish godoc
// @Summary Unpublish press release
// @Description Change press release status from published back to draft
// @Tags PressReleases
// @Accept json
// @Produce json
// @Param id path int true "Press release ID"
// @Success 200 {object} press_release.PressReleaseResponse "Press release unpublished successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID or press release is not published"
// @Failure 404 {object} response.Response{error=string} "Press release not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/press-releases/{id}/unpublish [patch]
func (h *PressReleaseHandler) Unpublish(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return response.BadRequest(c, "Press release ID is required")
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid press release ID format")
	}

	pr, err := h.service.Unpublish(uint(id))
	if err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Press release not found")
		}
		return response.BadRequest(c, err.Error())
	}

	return c.JSON(press_release.PressReleaseResponse{PressRelease: *pr})
}
