package repository

import (
	"fmt"
	"strings"
	"time"

	"github.com/healthcare-market-research/backend/internal/domain/report"
	"gorm.io/gorm"
)

// ReportFilters contains all possible filter options for reports
type ReportFilters struct {
	// Public filters
	Status      string   // 'draft' or 'published'
	Category    string   // Category slug
	Geography   []string // Array of geography strings
	Search      string   // Full-text search query
	AccessType  string   // 'free' or 'paid'
	Page        int
	Limit       int

	// Admin filters
	CreatedBy       *uint      // Filter by creator user ID
	UpdatedBy       *uint      // Filter by last updater user ID
	WorkflowStatus  string     // Filter by workflow status
	CreatedAfter    *time.Time // Date range start
	CreatedBefore   *time.Time // Date range end
	UpdatedAfter    *time.Time
	UpdatedBefore   *time.Time
	PublishedAfter  *time.Time
	PublishedBefore *time.Time
	IncludeDrafts   bool       // For admin, show drafts
}

type ReportRepository interface {
	GetAll(page, limit int) ([]report.Report, int64, error)
	GetAllWithFilters(filters ReportFilters) ([]report.Report, int64, error)
	GetBySlug(slug string) (*report.ReportWithRelations, error)
	GetByID(id uint) (*report.Report, error)
	GetByCategorySlug(categorySlug string, page, limit int) ([]report.Report, int64, error)
	Search(query string, page, limit int) ([]report.Report, int64, error)
	GetChartsByReportID(reportID uint) ([]report.ChartMetadata, error)
	Create(report *report.Report) error
	Update(report *report.Report) error
	Delete(id uint) error
	// Version history methods
	CreateVersion(version *report.ReportVersion) error
	GetVersionsByReportID(reportID uint) ([]report.ReportVersion, error)
	GetLatestVersionNumber(reportID uint) (int, error)
}

type reportRepository struct {
	db *gorm.DB
}

func NewReportRepository(db *gorm.DB) ReportRepository {
	return &reportRepository{db: db}
}

func (r *reportRepository) GetAll(page, limit int) ([]report.Report, int64, error) {
	var reports []report.Report
	var total int64

	offset := (page - 1) * limit

	// Default: only show published reports
	countSQL := "SELECT COUNT(*) FROM reports WHERE status = 'published'"
	if err := r.db.Raw(countSQL).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	querySQL := `
		SELECT *
		FROM reports
		WHERE status = 'published'
		ORDER BY COALESCE(publish_date, updated_at) DESC
		LIMIT ? OFFSET ?
	`

	err := r.db.Raw(querySQL, limit, offset).Scan(&reports).Error

	return reports, total, err
}

func (r *reportRepository) GetAllWithFilters(filters ReportFilters) ([]report.Report, int64, error) {
	var reports []report.Report
	var total int64

	offset := (filters.Page - 1) * filters.Limit

	// Build WHERE clause dynamically
	var conditions []string
	var args []interface{}
	var countArgs []interface{}

	// Status filter
	if filters.Status != "" {
		conditions = append(conditions, "r.status = ?")
		args = append(args, filters.Status)
		countArgs = append(countArgs, filters.Status)
	} else {
		// Default to published only for public API
		conditions = append(conditions, "r.status = 'published'")
	}

	// Category filter
	if filters.Category != "" {
		conditions = append(conditions, "c.slug = ?")
		args = append(args, filters.Category)
		countArgs = append(countArgs, filters.Category)
	}

	// Geography filter - check if any of the provided geographies are in the report's geography JSON array
	if len(filters.Geography) > 0 {
		geographyConditions := make([]string, len(filters.Geography))
		for i, geo := range filters.Geography {
			geographyConditions[i] = "r.geography::jsonb ? ?"
			args = append(args, geo)
			countArgs = append(countArgs, geo)
		}
		conditions = append(conditions, fmt.Sprintf("(%s)", strings.Join(geographyConditions, " OR ")))
	}

	// Access type filter
	if filters.AccessType != "" {
		conditions = append(conditions, "r.access_type = ?")
		args = append(args, filters.AccessType)
		countArgs = append(countArgs, filters.AccessType)
	}

	// Search filter
	if filters.Search != "" {
		searchPattern := "%" + filters.Search + "%"
		conditions = append(conditions, "(r.title ILIKE ? OR r.summary ILIKE ? OR r.description ILIKE ?)")
		args = append(args, searchPattern, searchPattern, searchPattern)
		countArgs = append(countArgs, searchPattern, searchPattern, searchPattern)
	}

	// Admin filters
	if filters.CreatedBy != nil {
		conditions = append(conditions, "r.created_by = ?")
		args = append(args, *filters.CreatedBy)
		countArgs = append(countArgs, *filters.CreatedBy)
	}

	if filters.UpdatedBy != nil {
		conditions = append(conditions, "r.updated_by = ?")
		args = append(args, *filters.UpdatedBy)
		countArgs = append(countArgs, *filters.UpdatedBy)
	}

	if filters.WorkflowStatus != "" {
		conditions = append(conditions, "r.workflow_status = ?")
		args = append(args, filters.WorkflowStatus)
		countArgs = append(countArgs, filters.WorkflowStatus)
	}

	// Date range filters
	if filters.CreatedAfter != nil {
		conditions = append(conditions, "r.created_at >= ?")
		args = append(args, *filters.CreatedAfter)
		countArgs = append(countArgs, *filters.CreatedAfter)
	}
	if filters.CreatedBefore != nil {
		conditions = append(conditions, "r.created_at <= ?")
		args = append(args, *filters.CreatedBefore)
		countArgs = append(countArgs, *filters.CreatedBefore)
	}

	if filters.UpdatedAfter != nil {
		conditions = append(conditions, "r.updated_at >= ?")
		args = append(args, *filters.UpdatedAfter)
		countArgs = append(countArgs, *filters.UpdatedAfter)
	}
	if filters.UpdatedBefore != nil {
		conditions = append(conditions, "r.updated_at <= ?")
		args = append(args, *filters.UpdatedBefore)
		countArgs = append(countArgs, *filters.UpdatedBefore)
	}

	if filters.PublishedAfter != nil {
		conditions = append(conditions, "r.publish_date >= ?")
		args = append(args, *filters.PublishedAfter)
		countArgs = append(countArgs, *filters.PublishedAfter)
	}
	if filters.PublishedBefore != nil {
		conditions = append(conditions, "r.publish_date <= ?")
		args = append(args, *filters.PublishedBefore)
		countArgs = append(countArgs, *filters.PublishedBefore)
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count query
	countSQL := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM reports r
		LEFT JOIN categories c ON r.category_id = c.id
		WHERE %s
	`, whereClause)

	if err := r.db.Raw(countSQL, countArgs...).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	// Fetch query
	querySQL := fmt.Sprintf(`
		SELECT r.*
		FROM reports r
		LEFT JOIN categories c ON r.category_id = c.id
		WHERE %s
		ORDER BY COALESCE(r.publish_date, r.updated_at) DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, filters.Limit, offset)
	err := r.db.Raw(querySQL, args...).Scan(&reports).Error

	return reports, total, err
}

func (r *reportRepository) GetBySlug(slug string) (*report.ReportWithRelations, error) {
	var result report.ReportWithRelations

	// Fetch the main report with all JSONB fields using GORM
	err := r.db.Table("reports").
		Select(`
			reports.*,
			categories.name as category_name,
			COALESCE(sub_categories.name, '') as sub_category_name,
			COALESCE(market_segments.name, '') as market_segment_name
		`).
		Joins("INNER JOIN categories ON reports.category_id = categories.id AND categories.is_active = true").
		Joins("LEFT JOIN sub_categories ON reports.sub_category_id = sub_categories.id AND sub_categories.is_active = true").
		Joins("LEFT JOIN market_segments ON reports.market_segment_id = market_segments.id AND market_segments.is_active = true").
		Where("reports.slug = ? AND reports.status = 'published'", slug).
		First(&result).Error

	if err != nil {
		return nil, err
	}

	// Get charts
	charts, _ := r.GetChartsByReportID(result.ID)
	result.Charts = charts

	// Get versions
	versions, _ := r.GetVersionsByReportID(result.ID)
	result.Versions = versions

	return &result, nil
}

func (r *reportRepository) GetByCategorySlug(categorySlug string, page, limit int) ([]report.Report, int64, error) {
	var reports []report.Report
	var total int64

	offset := (page - 1) * limit

	countSQL := `
		SELECT COUNT(*)
		FROM reports r
		INNER JOIN categories c ON r.category_id = c.id
		WHERE c.slug = ? AND c.is_active = true AND r.status = 'published'
	`
	if err := r.db.Raw(countSQL, categorySlug).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	querySQL := `
		SELECT r.*
		FROM reports r
		INNER JOIN categories c ON r.category_id = c.id
		WHERE c.slug = ? AND c.is_active = true AND r.status = 'published'
		ORDER BY COALESCE(r.publish_date, r.updated_at) DESC
		LIMIT ? OFFSET ?
	`

	err := r.db.Raw(querySQL, categorySlug, limit, offset).Scan(&reports).Error

	return reports, total, err
}

func (r *reportRepository) Search(query string, page, limit int) ([]report.Report, int64, error) {
	var reports []report.Report
	var total int64

	offset := (page - 1) * limit
	searchPattern := "%" + query + "%"

	countSQL := `
		SELECT COUNT(*)
		FROM reports
		WHERE status = 'published'
		AND (title ILIKE ? OR description ILIKE ? OR summary ILIKE ?)
	`
	if err := r.db.Raw(countSQL, searchPattern, searchPattern, searchPattern).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	querySQL := `
		SELECT *
		FROM reports
		WHERE status = 'published'
		AND (title ILIKE ? OR description ILIKE ? OR summary ILIKE ?)
		ORDER BY COALESCE(publish_date, updated_at) DESC
		LIMIT ? OFFSET ?
	`

	err := r.db.Raw(querySQL, searchPattern, searchPattern, searchPattern, limit, offset).Scan(&reports).Error

	return reports, total, err
}

func (r *reportRepository) GetChartsByReportID(reportID uint) ([]report.ChartMetadata, error) {
	var charts []report.ChartMetadata

	// Use raw SQL for better performance
	querySQL := `
		SELECT id, report_id, title, chart_type, description,
		       data_points, "order", is_active, created_at, updated_at
		FROM chart_metadata
		WHERE report_id = ? AND is_active = true
		ORDER BY "order" ASC
	`

	err := r.db.Raw(querySQL, reportID).Scan(&charts).Error
	return charts, err
}

func (r *reportRepository) Create(rep *report.Report) error {
	return r.db.Create(rep).Error
}

func (r *reportRepository) Update(rep *report.Report) error {
	return r.db.Save(rep).Error
}

func (r *reportRepository) Delete(id uint) error {
	return r.db.Delete(&report.Report{}, id).Error
}

func (r *reportRepository) GetByID(id uint) (*report.Report, error) {
	var rep report.Report
	err := r.db.First(&rep, id).Error
	if err != nil {
		return nil, err
	}
	return &rep, nil
}

// Version history methods

func (r *reportRepository) CreateVersion(version *report.ReportVersion) error {
	return r.db.Create(version).Error
}

func (r *reportRepository) GetVersionsByReportID(reportID uint) ([]report.ReportVersion, error) {
	var versions []report.ReportVersion
	err := r.db.Where("report_id = ?", reportID).
		Order("version_number DESC").
		Find(&versions).Error
	return versions, err
}

func (r *reportRepository) GetLatestVersionNumber(reportID uint) (int, error) {
	var maxVersion int
	err := r.db.Model(&report.ReportVersion{}).
		Where("report_id = ?", reportID).
		Select("COALESCE(MAX(version_number), 0)").
		Scan(&maxVersion).Error
	return maxVersion, err
}
