package handler

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/healthcare-market-research/backend/internal/domain/report"
	"github.com/healthcare-market-research/backend/internal/domain/user"
	"github.com/healthcare-market-research/backend/internal/repository"
	"github.com/healthcare-market-research/backend/internal/service"
	"github.com/healthcare-market-research/backend/pkg/response"
)

type ReportHandler struct {
	service service.ReportService
}

func NewReportHandler(service service.ReportService) *ReportHandler {
	return &ReportHandler{service: service}
}

// Helper functions

// isAdminOrEditor checks if the current user is an admin or editor
func isAdminOrEditor(c *fiber.Ctx) bool {
	userCtx := c.Locals("user")
	if userCtx == nil {
		return false
	}
	currentUser, ok := userCtx.(*user.User)
	if !ok {
		return false
	}
	return currentUser.Role == "admin" || currentUser.Role == "editor"
}

// stripAdminFields removes sensitive admin fields from reports for public API responses
func stripAdminFields(reports []report.Report) []report.Report {
	for i := range reports {
		reports[i].CreatedBy = nil
		reports[i].UpdatedBy = nil
		reports[i].InternalNotes = ""
		reports[i].WorkflowStatus = ""
		reports[i].ScheduledPublishAt = nil
		reports[i].ApprovedBy = nil
		reports[i].ApprovedAt = nil
	}
	return reports
}

// getUserID safely extracts user ID from fiber context
func getUserID(c *fiber.Ctx) (uint, error) {
	userCtx := c.Locals("user")
	if userCtx == nil {
		return 0, fiber.NewError(fiber.StatusUnauthorized, "User not authenticated")
	}
	currentUser, ok := userCtx.(*user.User)
	if !ok {
		return 0, fiber.NewError(fiber.StatusUnauthorized, "Invalid user context")
	}
	return currentUser.ID, nil
}

// GetAll godoc
// @Summary Get all reports with optional filters
// @Description Get a paginated list of healthcare market research reports with optional filters for status, category, geography, access type, and search. Admin users can use additional filters for user tracking, workflow status, and date ranges. Admin-only fields are automatically included in responses for authenticated admin/editor users.
// @Tags Reports
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1, min: 1)"
// @Param limit query int false "Items per page (default: 20, max: 100)"
// @Param status query string false "Filter by status (draft or published)"
// @Param category query string false "Filter by category slug"
// @Param geography query string false "Filter by geography (comma-separated, e.g., 'North America,Europe')"
// @Param accessType query string false "Filter by access type (free or paid)"
// @Param search query string false "Search in title, summary, and description"
// @Param created_by query int false "Admin only: Filter by creator user ID"
// @Param updated_by query int false "Admin only: Filter by last updater user ID"
// @Param workflow_status query string false "Admin only: Filter by workflow status (draft, pending_review, approved, rejected, scheduled, published, archived)"
// @Param created_after query string false "Admin only: Filter by created date (ISO 8601, e.g., 2024-01-01T00:00:00Z)"
// @Param created_before query string false "Admin only: Filter by created date (ISO 8601)"
// @Param updated_after query string false "Admin only: Filter by updated date (ISO 8601)"
// @Param updated_before query string false "Admin only: Filter by updated date (ISO 8601)"
// @Param published_after query string false "Admin only: Filter by published date (ISO 8601)"
// @Param published_before query string false "Admin only: Filter by published date (ISO 8601)"
// @Success 200 {object} response.Response{data=[]report.Report,meta=response.Meta} "List of reports with pagination metadata. Admin fields (created_by, updated_by, internal_notes, workflow_status, scheduled_publish_at, approved_by, approved_at) are included only for authenticated admin/editor users."
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

	// Check if any filters are provided
	status := c.Query("status")
	category := c.Query("category")
	geographyParam := c.Query("geography")
	accessType := c.Query("accessType")
	search := c.Query("search")

	// Admin filters
	var createdBy *uint
	if createdByStr := c.Query("created_by"); createdByStr != "" {
		id, err := strconv.ParseUint(createdByStr, 10, 32)
		if err == nil {
			val := uint(id)
			createdBy = &val
		}
	}

	var updatedBy *uint
	if updatedByStr := c.Query("updated_by"); updatedByStr != "" {
		id, err := strconv.ParseUint(updatedByStr, 10, 32)
		if err == nil {
			val := uint(id)
			updatedBy = &val
		}
	}

	workflowStatus := c.Query("workflow_status")

	// Date range parsing
	var createdAfter, createdBefore, updatedAfter, updatedBefore, publishedAfter, publishedBefore *time.Time
	if after := c.Query("created_after"); after != "" {
		if t, err := time.Parse(time.RFC3339, after); err == nil {
			createdAfter = &t
		}
	}
	if before := c.Query("created_before"); before != "" {
		if t, err := time.Parse(time.RFC3339, before); err == nil {
			createdBefore = &t
		}
	}
	if after := c.Query("updated_after"); after != "" {
		if t, err := time.Parse(time.RFC3339, after); err == nil {
			updatedAfter = &t
		}
	}
	if before := c.Query("updated_before"); before != "" {
		if t, err := time.Parse(time.RFC3339, before); err == nil {
			updatedBefore = &t
		}
	}
	if after := c.Query("published_after"); after != "" {
		if t, err := time.Parse(time.RFC3339, after); err == nil {
			publishedAfter = &t
		}
	}
	if before := c.Query("published_before"); before != "" {
		if t, err := time.Parse(time.RFC3339, before); err == nil {
			publishedBefore = &t
		}
	}

	var reports []report.Report
	var total int64
	var err error

	// If any filters are provided, use the filtered endpoint
	hasFilters := status != "" || category != "" || geographyParam != "" || accessType != "" || search != "" ||
		createdBy != nil || updatedBy != nil || workflowStatus != "" ||
		createdAfter != nil || createdBefore != nil || updatedAfter != nil || updatedBefore != nil ||
		publishedAfter != nil || publishedBefore != nil

	if hasFilters {
		// Parse geography into array
		var geography []string
		if geographyParam != "" {
			geography = strings.Split(geographyParam, ",")
			// Trim whitespace from each geography
			for i := range geography {
				geography[i] = strings.TrimSpace(geography[i])
			}
		}

		filters := repository.ReportFilters{
			Status:          status,
			Category:        category,
			Geography:       geography,
			AccessType:      accessType,
			Search:          search,
			CreatedBy:       createdBy,
			UpdatedBy:       updatedBy,
			WorkflowStatus:  workflowStatus,
			CreatedAfter:    createdAfter,
			CreatedBefore:   createdBefore,
			UpdatedAfter:    updatedAfter,
			UpdatedBefore:   updatedBefore,
			PublishedAfter:  publishedAfter,
			PublishedBefore: publishedBefore,
			Page:            page,
			Limit:           limit,
		}

		reports, total, err = h.service.GetAllWithFilters(filters)
	} else {
		// No filters, use the default GetAll
		reports, total, err = h.service.GetAll(page, limit)
	}

	if err != nil {
		return response.InternalError(c, "Failed to fetch reports")
	}

	// Strip admin fields if user is not admin/editor
	responseData := reports
	if !isAdminOrEditor(c) {
		responseData = stripAdminFields(reports)
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	meta := &response.Meta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}

	return response.SuccessWithMeta(c, responseData, meta)
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
// @Description Create a new healthcare market research report. Automatically sets created_by and updated_by to the authenticated user, and workflow_status to 'draft'. Requires admin or editor role.
// @Tags Reports
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param report body report.Report true "Report data"
// @Success 201 {object} response.Response{data=report.Report} "Created report with auto-populated admin tracking fields"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid input"
// @Failure 401 {object} response.Response{error=string} "Unauthorized - authentication required"
// @Failure 403 {object} response.Response{error=string} "Forbidden - requires admin or editor role"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/reports [post]
func (h *ReportHandler) Create(c *fiber.Ctx) error {
	// Get authenticated user ID
	userID, err := getUserID(c)
	if err != nil {
		return err
	}

	var req report.Report
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	// Validate required fields based on frontend expectations
	if req.Title == "" {
		return response.BadRequest(c, "Title is required (minimum 10 characters)")
	}
	if req.Slug == "" {
		return response.BadRequest(c, "Slug is required")
	}
	if req.CategoryID == 0 {
		return response.BadRequest(c, "Category ID is required")
	}
	if req.Summary == "" {
		return response.BadRequest(c, "Summary is required (minimum 50 characters)")
	}
	if len(req.Geography) == 0 {
		return response.BadRequest(c, "At least one geography is required")
	}

	// Set default values if not provided
	if req.Status == "" {
		req.Status = "draft"
	}
	if req.AccessType == "" {
		req.AccessType = "paid"
	}

	if err := h.service.Create(&req, userID); err != nil {
		return response.InternalError(c, "Failed to create report")
	}

	return c.Status(fiber.StatusCreated).JSON(response.Response{
		Success: true,
		Data:    req,
	})
}

// Update godoc
// @Summary Update a report
// @Description Update an existing healthcare market research report by ID. Automatically updates updated_by field. If workflow_status changes to 'approved', sets approved_by and approved_at. Creates version history when status changes to 'published'. Requires admin or editor role.
// @Tags Reports
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Report ID"
// @Param report body report.Report true "Updated report data"
// @Success 200 {object} response.Response{data=report.Report} "Updated report with auto-updated admin tracking fields"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid input"
// @Failure 401 {object} response.Response{error=string} "Unauthorized - authentication required"
// @Failure 403 {object} response.Response{error=string} "Forbidden - requires admin or editor role"
// @Failure 404 {object} response.Response{error=string} "Report not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/reports/{id} [put]
func (h *ReportHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid report ID")
	}

	// Get user ID from context (set by auth middleware)
	userCtx := c.Locals("user")
	if userCtx == nil {
		return response.Unauthorized(c, "User not authenticated")
	}
	currentUser, ok := userCtx.(*user.User)
	if !ok {
		return response.Unauthorized(c, "Invalid user context")
	}

	var req report.Report
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	// Validate required fields
	if req.Title == "" {
		return response.BadRequest(c, "Title is required (minimum 10 characters)")
	}
	if req.Slug == "" {
		return response.BadRequest(c, "Slug is required")
	}
	if req.CategoryID == 0 {
		return response.BadRequest(c, "Category ID is required")
	}
	if req.Summary == "" {
		return response.BadRequest(c, "Summary is required (minimum 50 characters)")
	}
	if len(req.Geography) == 0 {
		return response.BadRequest(c, "At least one geography is required")
	}

	// Pass user ID to service for version history
	if err := h.service.Update(uint(id), &req, currentUser.ID); err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Report not found")
		}
		return response.InternalError(c, "Failed to update report")
	}

	return response.Success(c, req)
}

// Delete godoc
// @Summary Delete a report
// @Description Delete a healthcare market research report by ID. Permanently removes the report and all associated data. Requires admin role.
// @Tags Reports
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Report ID"
// @Success 200 {object} response.Response{data=map[string]string} "Report deleted successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID"
// @Failure 401 {object} response.Response{error=string} "Unauthorized - authentication required"
// @Failure 403 {object} response.Response{error=string} "Forbidden - requires admin role"
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

// Workflow Management Handlers

// SubmitForReview godoc
// @Summary Submit report for review
// @Description Submit a report for review (changes workflow_status from 'draft' or 'rejected' to 'pending_review'). Can only be called on reports with workflow_status of 'draft' or 'rejected'. Updates the updated_by field to the authenticated user. Requires admin or editor role.
// @Tags Reports - Workflow
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Report ID"
// @Success 200 {object} response.Response{data=string} "Report submitted for review successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID or invalid workflow transition (e.g., report is not in draft/rejected status)"
// @Failure 401 {object} response.Response{error=string} "Unauthorized - authentication required"
// @Failure 403 {object} response.Response{error=string} "Forbidden - requires admin or editor role"
// @Failure 404 {object} response.Response{error=string} "Report not found"
// @Router /api/v1/reports/{id}/submit-review [post]
func (h *ReportHandler) SubmitForReview(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid report ID")
	}

	userID, err := getUserID(c)
	if err != nil {
		return err
	}

	if err := h.service.SubmitForReview(uint(id), userID); err != nil {
		return response.BadRequest(c, err.Error())
	}

	return response.Success(c, "Report submitted for review")
}

// ApproveReport godoc
// @Summary Approve report
// @Description Approve a report (changes workflow_status from 'pending_review' to 'approved'). Can only be called on reports with workflow_status of 'pending_review'. Automatically sets approved_by to the authenticated user and approved_at to current timestamp. Updates updated_by field. Requires admin role only.
// @Tags Reports - Workflow
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Report ID"
// @Success 200 {object} response.Response{data=string} "Report approved successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID or invalid workflow transition (e.g., report is not in pending_review status)"
// @Failure 401 {object} response.Response{error=string} "Unauthorized - authentication required"
// @Failure 403 {object} response.Response{error=string} "Forbidden - requires admin role"
// @Failure 404 {object} response.Response{error=string} "Report not found"
// @Router /api/v1/reports/{id}/approve [post]
func (h *ReportHandler) ApproveReport(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid report ID")
	}

	userID, err := getUserID(c)
	if err != nil {
		return err
	}

	if err := h.service.ApproveReport(uint(id), userID); err != nil {
		return response.BadRequest(c, err.Error())
	}

	return response.Success(c, "Report approved")
}

// RejectReport godoc
// @Summary Reject report
// @Description Reject a report with a reason (changes workflow_status from 'pending_review' to 'rejected'). Can only be called on reports with workflow_status of 'pending_review'. Automatically appends rejection reason with timestamp to internal_notes field. Updates updated_by field. Requires admin role only.
// @Tags Reports - Workflow
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Report ID"
// @Param body body object{reason=string} true "Rejection details - reason field is required and will be appended to internal_notes"
// @Success 200 {object} response.Response{data=string} "Report rejected successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID, missing reason, or invalid workflow transition (e.g., report is not in pending_review status)"
// @Failure 401 {object} response.Response{error=string} "Unauthorized - authentication required"
// @Failure 403 {object} response.Response{error=string} "Forbidden - requires admin role"
// @Failure 404 {object} response.Response{error=string} "Report not found"
// @Router /api/v1/reports/{id}/reject [post]
func (h *ReportHandler) RejectReport(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid report ID")
	}

	userID, err := getUserID(c)
	if err != nil {
		return err
	}

	var body struct {
		Reason string `json:"reason"`
	}
	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if body.Reason == "" {
		return response.BadRequest(c, "Rejection reason is required")
	}

	if err := h.service.RejectReport(uint(id), userID, body.Reason); err != nil {
		return response.BadRequest(c, err.Error())
	}

	return response.Success(c, "Report rejected")
}

// SchedulePublish godoc
// @Summary Schedule report for publishing
// @Description Schedule an approved report for future publishing (changes workflow_status from 'approved' to 'scheduled'). Can only be called on reports with workflow_status of 'approved'. Sets scheduled_publish_at to the provided timestamp. The scheduled time must be in the future. Requires admin role only.
// @Tags Reports - Workflow
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Report ID"
// @Param body body object{scheduled_at=string} true "Schedule details - scheduled_at must be a future timestamp in RFC3339 format (e.g., '2024-02-01T10:00:00Z')"
// @Success 200 {object} response.Response{data=string} "Report scheduled for publishing successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID, missing/invalid scheduled_at, scheduled time is in the past, or invalid workflow transition (e.g., report is not in approved status)"
// @Failure 401 {object} response.Response{error=string} "Unauthorized - authentication required"
// @Failure 403 {object} response.Response{error=string} "Forbidden - requires admin role"
// @Failure 404 {object} response.Response{error=string} "Report not found"
// @Router /api/v1/reports/{id}/schedule [post]
func (h *ReportHandler) SchedulePublish(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid report ID")
	}

	var body struct {
		ScheduledAt time.Time `json:"scheduled_at"`
	}
	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if body.ScheduledAt.IsZero() {
		return response.BadRequest(c, "Scheduled publish time is required")
	}

	if err := h.service.SchedulePublish(uint(id), body.ScheduledAt); err != nil {
		return response.BadRequest(c, err.Error())
	}

	return response.Success(c, "Report scheduled for publishing")
}
