package repository

import (
	"time"

	"github.com/healthcare-market-research/backend/internal/domain/report"
	"gorm.io/gorm"
)

type ReportRepository interface {
	GetAll(page, limit int) ([]report.Report, int64, error)
	GetBySlug(slug string) (*report.ReportWithRelations, error)
	GetByCategorySlug(categorySlug string, page, limit int) ([]report.Report, int64, error)
	Search(query string, page, limit int) ([]report.Report, int64, error)
	GetChartsByReportID(reportID uint) ([]report.ChartMetadata, error)
	Create(report *report.Report) error
	Update(report *report.Report) error
	Delete(id uint) error
	GetByID(id uint) (*report.Report, error)
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

	// Use raw SQL for better performance
	countSQL := "SELECT COUNT(*) FROM reports WHERE is_published = true"
	if err := r.db.Raw(countSQL).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	// Raw SQL for fetching reports
	querySQL := `
		SELECT id, title, slug, description, summary, thumbnail_url, price, currency,
		       page_count, category_id, sub_category_id, market_segment_id,
		       is_published, is_featured, view_count, download_count, published_at,
		       meta_title, meta_description, meta_keywords, created_at, updated_at
		FROM reports
		WHERE is_published = true
		ORDER BY published_at DESC
		LIMIT ? OFFSET ?
	`

	err := r.db.Raw(querySQL, limit, offset).Scan(&reports).Error

	return reports, total, err
}

func (r *reportRepository) GetBySlug(slug string) (*report.ReportWithRelations, error) {
	// Use a single optimized query with JOINs to avoid N+1 queries
	type QueryResult struct {
		// Report fields
		ID               uint
		CategoryID       uint
		SubCategoryID    *uint
		MarketSegmentID  *uint
		Title            string
		Slug             string
		Description      string
		Summary          string
		ThumbnailURL     string
		Price            float64
		Currency         string
		PageCount        int
		PublishedAt      *time.Time
		IsPublished      bool
		IsFeatured       bool
		ViewCount        int
		DownloadCount    int
		MetaTitle        string
		MetaDescription  string
		MetaKeywords     string
		CreatedAt        time.Time
		UpdatedAt        time.Time
		// Related names
		CategoryName      string
		SubCategoryName   string
		MarketSegmentName string
	}

	var qr QueryResult

	// Single query with LEFT JOINs to get all related data
	querySQL := `
		SELECT
			r.id, r.category_id, r.sub_category_id, r.market_segment_id,
			r.title, r.slug, r.description, r.summary, r.thumbnail_url,
			r.price, r.currency, r.page_count, r.published_at,
			r.is_published, r.is_featured, r.view_count, r.download_count,
			r.meta_title, r.meta_description, r.meta_keywords,
			r.created_at, r.updated_at,
			c.name as category_name,
			COALESCE(sc.name, '') as sub_category_name,
			COALESCE(ms.name, '') as market_segment_name
		FROM reports r
		INNER JOIN categories c ON r.category_id = c.id AND c.is_active = true
		LEFT JOIN sub_categories sc ON r.sub_category_id = sc.id AND sc.is_active = true
		LEFT JOIN market_segments ms ON r.market_segment_id = ms.id AND ms.is_active = true
		WHERE r.slug = ? AND r.is_published = true
		LIMIT 1
	`

	err := r.db.Raw(querySQL, slug).Scan(&qr).Error
	if err != nil {
		return nil, err
	}

	// Build the result
	result := &report.ReportWithRelations{
		Report: report.Report{
			ID:              qr.ID,
			CategoryID:      qr.CategoryID,
			SubCategoryID:   qr.SubCategoryID,
			MarketSegmentID: qr.MarketSegmentID,
			Title:           qr.Title,
			Slug:            qr.Slug,
			Description:     qr.Description,
			Summary:         qr.Summary,
			ThumbnailURL:    qr.ThumbnailURL,
			Price:           qr.Price,
			Currency:        qr.Currency,
			PageCount:       qr.PageCount,
			PublishedAt:     qr.PublishedAt,
			IsPublished:     qr.IsPublished,
			IsFeatured:      qr.IsFeatured,
			ViewCount:       qr.ViewCount,
			DownloadCount:   qr.DownloadCount,
			MetaTitle:       qr.MetaTitle,
			MetaDescription: qr.MetaDescription,
			MetaKeywords:    qr.MetaKeywords,
			CreatedAt:       qr.CreatedAt,
			UpdatedAt:       qr.UpdatedAt,
		},
		CategoryName:      qr.CategoryName,
		SubCategoryName:   qr.SubCategoryName,
		MarketSegmentName: qr.MarketSegmentName,
	}

	// Get charts (single additional query)
	charts, _ := r.GetChartsByReportID(qr.ID)
	result.Charts = charts

	return result, nil
}

func (r *reportRepository) GetByCategorySlug(categorySlug string, page, limit int) ([]report.Report, int64, error) {
	var reports []report.Report
	var total int64

	offset := (page - 1) * limit

	// Use raw SQL for better performance
	countSQL := `
		SELECT COUNT(*)
		FROM reports r
		INNER JOIN categories c ON r.category_id = c.id
		WHERE c.slug = ? AND c.is_active = true AND r.is_published = true
	`
	if err := r.db.Raw(countSQL, categorySlug).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	// Raw SQL for fetching reports
	querySQL := `
		SELECT r.id, r.title, r.slug, r.description, r.summary, r.thumbnail_url,
		       r.price, r.currency, r.page_count, r.category_id, r.sub_category_id,
		       r.market_segment_id, r.is_published, r.is_featured, r.view_count,
		       r.download_count, r.published_at, r.meta_title, r.meta_description,
		       r.meta_keywords, r.created_at, r.updated_at
		FROM reports r
		INNER JOIN categories c ON r.category_id = c.id
		WHERE c.slug = ? AND c.is_active = true AND r.is_published = true
		ORDER BY r.published_at DESC
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

	// Use raw SQL for better performance
	countSQL := `
		SELECT COUNT(*)
		FROM reports
		WHERE is_published = true
		AND (title ILIKE ? OR description ILIKE ? OR summary ILIKE ?)
	`
	if err := r.db.Raw(countSQL, searchPattern, searchPattern, searchPattern).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	// Raw SQL for fetching reports
	querySQL := `
		SELECT id, title, slug, description, summary, thumbnail_url, price, currency,
		       page_count, category_id, sub_category_id, market_segment_id,
		       is_published, is_featured, view_count, download_count, published_at,
		       meta_title, meta_description, meta_keywords, created_at, updated_at
		FROM reports
		WHERE is_published = true
		AND (title ILIKE ? OR description ILIKE ? OR summary ILIKE ?)
		ORDER BY published_at DESC
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
