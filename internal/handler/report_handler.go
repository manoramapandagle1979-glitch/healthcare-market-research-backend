package handler

import (
	"math"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/healthcare-market-research/backend/internal/domain/report"
	"github.com/healthcare-market-research/backend/internal/service"
	"github.com/healthcare-market-research/backend/pkg/response"
)

type ReportHandler struct {
	service service.ReportService
}

func NewReportHandler(service service.ReportService) *ReportHandler {
	return &ReportHandler{service: service}
}

// GetAll godoc
// @Summary Get all reports
// @Description Get a paginated list of all published healthcare market research reports
// @Tags Reports
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1, min: 1)"
// @Param limit query int false "Items per page (default: 20, max: 100)"
// @Success 200 {object} response.Response{data=[]report.ReportWithRelations,meta=response.Meta} "List of reports with pagination metadata"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/reports [get]
func (h *ReportHandler) GetAll(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	reports, total, err := h.service.GetAll(page, limit)
	if err != nil {
		return response.InternalError(c, "Failed to fetch reports")
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	meta := &response.Meta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}

	return response.SuccessWithMeta(c, reports, meta)
}

// GetBySlug godoc
// @Summary Get report by slug
// @Description Get a single healthcare market research report by its unique slug
// @Tags Reports
// @Accept json
// @Produce json
// @Param slug path string true "Report slug"
// @Success 200 {object} response.Response{data=report.ReportWithRelations} "Report details with relations"
// @Failure 400 {object} response.Response{error=string} "Bad request - slug is required"
// @Failure 404 {object} response.Response{error=string} "Report not found"
// @Router /api/v1/reports/{slug} [get]
func (h *ReportHandler) GetBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return response.BadRequest(c, "Slug is required")
	}

	report, err := h.service.GetBySlug(slug)
	if err != nil {
		return response.NotFound(c, "Report not found")
	}

	return response.Success(c, report)
}

// GetByCategorySlug godoc
// @Summary Get reports by category slug
// @Description Get a paginated list of reports belonging to a specific category
// @Tags Reports
// @Accept json
// @Produce json
// @Param slug path string true "Category slug"
// @Param page query int false "Page number (default: 1, min: 1)"
// @Param limit query int false "Items per page (default: 20, max: 100)"
// @Success 200 {object} response.Response{data=[]report.ReportWithRelations,meta=response.Meta} "List of reports in category with pagination metadata"
// @Failure 400 {object} response.Response{error=string} "Bad request - category slug is required"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/categories/{slug}/reports [get]
func (h *ReportHandler) GetByCategorySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return response.BadRequest(c, "Category slug is required")
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	reports, total, err := h.service.GetByCategorySlug(slug, page, limit)
	if err != nil {
		return response.InternalError(c, "Failed to fetch reports")
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	meta := &response.Meta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}

	return response.SuccessWithMeta(c, reports, meta)
}

// Search godoc
// @Summary Search reports
// @Description Search for healthcare market research reports by query text (searches in title, description, and summary)
// @Tags Reports
// @Accept json
// @Produce json
// @Param q query string true "Search query text"
// @Param page query int false "Page number (default: 1, min: 1)"
// @Param limit query int false "Items per page (default: 20, max: 100)"
// @Success 200 {object} response.Response{data=[]report.ReportWithRelations,meta=response.Meta} "Search results with pagination metadata"
// @Failure 400 {object} response.Response{error=string} "Bad request - search query is required"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/search [get]
func (h *ReportHandler) Search(c *fiber.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return response.BadRequest(c, "Search query is required")
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	reports, total, err := h.service.Search(query, page, limit)
	if err != nil {
		return response.InternalError(c, "Failed to search reports")
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	meta := &response.Meta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}

	return response.SuccessWithMeta(c, reports, meta)
}

// Create godoc
// @Summary Create a new report
// @Description Create a new healthcare market research report
// @Tags Reports
// @Accept json
// @Produce json
// @Param report body report.Report true "Report data"
// @Success 201 {object} response.Response{data=report.Report} "Created report"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid input"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/reports [post]
func (h *ReportHandler) Create(c *fiber.Ctx) error {
	var req report.Report
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	// Validate required fields
	if req.Title == "" || req.Slug == "" || req.CategoryID == 0 {
		return response.BadRequest(c, "Title, slug, and category_id are required")
	}

	if err := h.service.Create(&req); err != nil {
		return response.InternalError(c, "Failed to create report")
	}

	return c.Status(fiber.StatusCreated).JSON(response.Response{
		Success: true,
		Data:    req,
	})
}

// Update godoc
// @Summary Update a report
// @Description Update an existing healthcare market research report by ID
// @Tags Reports
// @Accept json
// @Produce json
// @Param id path int true "Report ID"
// @Param report body report.Report true "Updated report data"
// @Success 200 {object} response.Response{data=report.Report} "Updated report"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid input"
// @Failure 404 {object} response.Response{error=string} "Report not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/reports/{id} [put]
func (h *ReportHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid report ID")
	}

	var req report.Report
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	// Validate required fields
	if req.Title == "" || req.Slug == "" || req.CategoryID == 0 {
		return response.BadRequest(c, "Title, slug, and category_id are required")
	}

	if err := h.service.Update(uint(id), &req); err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Report not found")
		}
		return response.InternalError(c, "Failed to update report")
	}

	return response.Success(c, req)
}

// Delete godoc
// @Summary Delete a report
// @Description Delete a healthcare market research report by ID
// @Tags Reports
// @Accept json
// @Produce json
// @Param id path int true "Report ID"
// @Success 200 {object} response.Response{data=map[string]string} "Report deleted successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID"
// @Failure 404 {object} response.Response{error=string} "Report not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/reports/{id} [delete]
func (h *ReportHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid report ID")
	}

	if err := h.service.Delete(uint(id)); err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Report not found")
		}
		return response.InternalError(c, "Failed to delete report")
	}

	return response.Success(c, fiber.Map{
		"message": "Report deleted successfully",
	})
}
