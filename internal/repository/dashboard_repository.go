package repository

import (
	"time"

	"github.com/healthcare-market-research/backend/internal/domain/dashboard"
	"gorm.io/gorm"
)

// DashboardRepository defines the interface for dashboard data access
type DashboardRepository interface {
	GetReportStats(userRole string) (*dashboard.ReportStats, error)
	GetBlogStats() (*dashboard.BlogStats, error)
	GetPressReleaseStats() (*dashboard.PressReleaseStats, error)
	GetUserStats() (*dashboard.UserStats, error)
	GetLeadStats() (*dashboard.LeadStats, error)
	GetTopPerformingReports(limit int) ([]dashboard.TopReport, error)
	GetContentCreationStats() (*dashboard.ContentCreationStats, error)
	GetRecentActivities(limit int) ([]dashboard.Activity, error)
	GetTopCategories(limit int) ([]dashboard.TopCategory, error)
}

type dashboardRepository struct {
	db *gorm.DB
}

// NewDashboardRepository creates a new dashboard repository instance
func NewDashboardRepository(db *gorm.DB) DashboardRepository {
	return &dashboardRepository{db: db}
}

// GetReportStats retrieves report statistics
func (r *dashboardRepository) GetReportStats(userRole string) (*dashboard.ReportStats, error) {
	stats := &dashboard.ReportStats{}

	// Get counts with conditional aggregation
	type ReportCounts struct {
		Total     int64
		Published int64
		Draft     int64
		Archived  int64
	}

	var counts ReportCounts
	err := r.db.Table("reports").
		Select(`
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'published' THEN 1 END) as published,
			COUNT(CASE WHEN status = 'draft' THEN 1 END) as draft,
			COUNT(CASE WHEN deleted_at IS NOT NULL THEN 1 END) as archived
		`).
		Where("deleted_at IS NULL OR deleted_at IS NOT NULL").
		Scan(&counts).Error

	if err != nil {
		return nil, err
	}

	stats.Total = counts.Total
	stats.Published = counts.Published
	stats.Draft = counts.Draft
	stats.Archived = counts.Archived

	// Calculate change percentage (compare with last 30 days vs previous 30 days)
	now := time.Now()
	thirtyDaysAgo := now.AddDate(0, 0, -30)
	sixtyDaysAgo := now.AddDate(0, 0, -60)

	var currentCount, previousCount int64

	// Current period (last 30 days)
	r.db.Table("reports").
		Where("created_at >= ? AND created_at < ? AND deleted_at IS NULL", thirtyDaysAgo, now).
		Count(&currentCount)

	// Previous period (30-60 days ago)
	r.db.Table("reports").
		Where("created_at >= ? AND created_at < ? AND deleted_at IS NULL", sixtyDaysAgo, thirtyDaysAgo).
		Count(&previousCount)

	if previousCount > 0 {
		stats.Change = ((float64(currentCount) - float64(previousCount)) / float64(previousCount)) * 100
	} else if currentCount > 0 {
		stats.Change = 100.0
	}

	return stats, nil
}

// GetBlogStats retrieves blog statistics
func (r *dashboardRepository) GetBlogStats() (*dashboard.BlogStats, error) {
	stats := &dashboard.BlogStats{}

	type BlogCounts struct {
		Total     int64
		Published int64
		Draft     int64
		Review    int64
	}

	var counts BlogCounts
	err := r.db.Table("blogs").
		Select(`
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'published' THEN 1 END) as published,
			COUNT(CASE WHEN status = 'draft' THEN 1 END) as draft,
			COUNT(CASE WHEN status = 'review' THEN 1 END) as review
		`).
		Where("deleted_at IS NULL").
		Scan(&counts).Error

	if err != nil {
		return nil, err
	}

	stats.Total = counts.Total
	stats.Published = counts.Published
	stats.Draft = counts.Draft
	stats.Review = counts.Review

	// Calculate change percentage
	now := time.Now()
	thirtyDaysAgo := now.AddDate(0, 0, -30)
	sixtyDaysAgo := now.AddDate(0, 0, -60)

	var currentCount, previousCount int64

	r.db.Table("blogs").
		Where("created_at >= ? AND created_at < ? AND deleted_at IS NULL", thirtyDaysAgo, now).
		Count(&currentCount)

	r.db.Table("blogs").
		Where("created_at >= ? AND created_at < ? AND deleted_at IS NULL", sixtyDaysAgo, thirtyDaysAgo).
		Count(&previousCount)

	if previousCount > 0 {
		stats.Change = ((float64(currentCount) - float64(previousCount)) / float64(previousCount)) * 100
	} else if currentCount > 0 {
		stats.Change = 100.0
	}

	return stats, nil
}

// GetPressReleaseStats retrieves press release statistics
func (r *dashboardRepository) GetPressReleaseStats() (*dashboard.PressReleaseStats, error) {
	stats := &dashboard.PressReleaseStats{}

	type PressReleaseCounts struct {
		Total     int64
		Published int64
		Draft     int64
		Review    int64
	}

	var counts PressReleaseCounts
	err := r.db.Table("press_releases").
		Select(`
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'published' THEN 1 END) as published,
			COUNT(CASE WHEN status = 'draft' THEN 1 END) as draft,
			COUNT(CASE WHEN status = 'review' THEN 1 END) as review
		`).
		Where("deleted_at IS NULL").
		Scan(&counts).Error

	if err != nil {
		return nil, err
	}

	stats.Total = counts.Total
	stats.Published = counts.Published
	stats.Draft = counts.Draft
	stats.Review = counts.Review

	// Calculate change percentage
	now := time.Now()
	thirtyDaysAgo := now.AddDate(0, 0, -30)
	sixtyDaysAgo := now.AddDate(0, 0, -60)

	var currentCount, previousCount int64

	r.db.Table("press_releases").
		Where("created_at >= ? AND created_at < ? AND deleted_at IS NULL", thirtyDaysAgo, now).
		Count(&currentCount)

	r.db.Table("press_releases").
		Where("created_at >= ? AND created_at < ? AND deleted_at IS NULL", sixtyDaysAgo, thirtyDaysAgo).
		Count(&previousCount)

	if previousCount > 0 {
		stats.Change = ((float64(currentCount) - float64(previousCount)) / float64(previousCount)) * 100
	} else if currentCount > 0 {
		stats.Change = 100.0
	}

	return stats, nil
}

// GetUserStats retrieves user statistics (admin only)
func (r *dashboardRepository) GetUserStats() (*dashboard.UserStats, error) {
	stats := &dashboard.UserStats{
		ByRole: make(map[string]int64),
	}

	// Get total users
	r.db.Table("users").
		Count(&stats.Total)

	// Get active users (logged in within last 30 days)
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	r.db.Table("users").
		Where("last_login_at >= ?", thirtyDaysAgo).
		Count(&stats.Active)

	// Get users by role
	type RoleCount struct {
		Role  string
		Count int64
	}
	var roleCounts []RoleCount
	err := r.db.Table("users").
		Select("role, COUNT(*) as count").
		Group("role").
		Scan(&roleCounts).Error

	if err != nil {
		return nil, err
	}

	for _, rc := range roleCounts {
		stats.ByRole[rc.Role] = rc.Count
	}

	// Calculate change percentage
	now := time.Now()
	thirtyDaysAgo = now.AddDate(0, 0, -30)
	sixtyDaysAgo := now.AddDate(0, 0, -60)

	var currentCount, previousCount int64

	r.db.Table("users").
		Where("created_at >= ? AND created_at < ?", thirtyDaysAgo, now).
		Count(&currentCount)

	r.db.Table("users").
		Where("created_at >= ? AND created_at < ?", sixtyDaysAgo, thirtyDaysAgo).
		Count(&previousCount)

	if previousCount > 0 {
		stats.Change = ((float64(currentCount) - float64(previousCount)) / float64(previousCount)) * 100
	} else if currentCount > 0 {
		stats.Change = 100.0
	}

	return stats, nil
}

// GetLeadStats retrieves lead/form submission statistics
func (r *dashboardRepository) GetLeadStats() (*dashboard.LeadStats, error) {
	stats := &dashboard.LeadStats{
		ByCategory: make(map[string]int64),
		Recent:     &dashboard.RecentLeadStats{},
	}

	// Get total leads
	r.db.Table("form_submissions").Count(&stats.Total)

	// Get counts by status
	r.db.Table("form_submissions").
		Where("status = ?", "pending").
		Count(&stats.Pending)

	r.db.Table("form_submissions").
		Where("status = ?", "processed").
		Count(&stats.Processed)

	// Get leads by category
	type CategoryCount struct {
		Category string
		Count    int64
	}
	var categoryCounts []CategoryCount
	err := r.db.Table("form_submissions").
		Select("category, COUNT(*) as count").
		Group("category").
		Scan(&categoryCounts).Error

	if err != nil {
		return nil, err
	}

	for _, cc := range categoryCounts {
		stats.ByCategory[cc.Category] = cc.Count
	}

	// Get recent leads
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekStart := today.AddDate(0, 0, -int(today.Weekday()))
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	r.db.Table("form_submissions").
		Where("DATE(created_at) = CURRENT_DATE").
		Count(&stats.Recent.Today)

	r.db.Table("form_submissions").
		Where("created_at >= ?", weekStart).
		Count(&stats.Recent.ThisWeek)

	r.db.Table("form_submissions").
		Where("created_at >= ?", monthStart).
		Count(&stats.Recent.ThisMonth)

	return stats, nil
}

// GetTopPerformingReports retrieves top performing reports by view count
func (r *dashboardRepository) GetTopPerformingReports(limit int) ([]dashboard.TopReport, error) {
	var reports []dashboard.TopReport

	err := r.db.Table("reports").
		Select("id, title, slug, view_count, download_count").
		Where("status = ? AND deleted_at IS NULL", "published").
		Order("view_count DESC").
		Limit(limit).
		Scan(&reports).Error

	return reports, err
}

// GetContentCreationStats retrieves content creation statistics for current month
func (r *dashboardRepository) GetContentCreationStats() (*dashboard.ContentCreationStats, error) {
	stats := &dashboard.ContentCreationStats{}

	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	// Reports this month
	var reportsCount int64
	r.db.Table("reports").
		Where("created_at >= ? AND deleted_at IS NULL", monthStart).
		Count(&reportsCount)
	stats.ReportsThisMonth = reportsCount

	// Blogs this month
	var blogsCount int64
	r.db.Table("blogs").
		Where("created_at >= ? AND deleted_at IS NULL", monthStart).
		Count(&blogsCount)
	stats.BlogsThisMonth = blogsCount

	// Press releases this month
	var pressReleasesCount int64
	r.db.Table("press_releases").
		Where("created_at >= ? AND deleted_at IS NULL", monthStart).
		Count(&pressReleasesCount)
	stats.PressReleasesThisMonth = pressReleasesCount

	return stats, nil
}

// GetRecentActivities retrieves recent activities from audit logs
func (r *dashboardRepository) GetRecentActivities(limit int) ([]dashboard.Activity, error) {
	var activities []dashboard.Activity

	type AuditLogRow struct {
		ID         uint
		Action     string
		EntityType string
		EntityID   uint
		UserID     uint
		UserEmail  string
		CreatedAt  time.Time
	}

	var auditLogs []AuditLogRow
	err := r.db.Table("audit_logs").
		Select("id, action, entity_type, entity_id, user_id, user_email, created_at").
		Where("status = ?", "success").
		Order("created_at DESC").
		Limit(limit).
		Scan(&auditLogs).Error

	if err != nil {
		return nil, err
	}

	// Convert audit logs to activities
	for _, log := range auditLogs {
		activity := dashboard.Activity{
			ID:         log.ID,
			Type:       mapActionToActivityType(log.Action, log.EntityType),
			Title:      dashboard.GetActivityTitle(dashboard.ActivityType(mapActionToActivityType(log.Action, log.EntityType))),
			EntityType: log.EntityType,
			EntityID:   log.EntityID,
			Timestamp:  log.CreatedAt,
		}

		// Get entity description based on type
		activity.Description = getEntityDescription(r.db, log.EntityType, log.EntityID)

		// Add user info
		activity.User = &dashboard.UserInfo{
			ID:    log.UserID,
			Email: log.UserEmail,
		}

		// Try to get user name
		var userName string
		r.db.Table("users").
			Select("name").
			Where("id = ?", log.UserID).
			Scan(&userName)
		activity.User.Name = userName

		activities = append(activities, activity)
	}

	return activities, nil
}

// GetTopCategories retrieves top categories by report count
func (r *dashboardRepository) GetTopCategories(limit int) ([]dashboard.TopCategory, error) {
	var categories []dashboard.TopCategory

	err := r.db.Table("categories").
		Select("categories.id, categories.name, categories.slug, COUNT(reports.id) as report_count").
		Joins("LEFT JOIN reports ON reports.category_id = categories.id AND reports.deleted_at IS NULL").
		Where("categories.deleted_at IS NULL").
		Group("categories.id, categories.name, categories.slug").
		Order("report_count DESC").
		Limit(limit).
		Scan(&categories).Error

	return categories, err
}

// Helper function to map audit log action to activity type
func mapActionToActivityType(action, entityType string) string {
	switch entityType {
	case "report":
		switch action {
		case "create":
			return string(dashboard.ActivityReportCreated)
		case "update":
			return string(dashboard.ActivityReportUpdated)
		case "delete":
			return string(dashboard.ActivityReportDeleted)
		case "publish":
			return string(dashboard.ActivityReportPublished)
		}
	case "blog":
		switch action {
		case "create":
			return string(dashboard.ActivityBlogCreated)
		case "update":
			return string(dashboard.ActivityBlogUpdated)
		case "publish":
			return string(dashboard.ActivityBlogPublished)
		}
	case "press_release":
		switch action {
		case "create":
			return string(dashboard.ActivityPressReleaseCreated)
		case "publish":
			return string(dashboard.ActivityPressReleasePublished)
		}
	case "user":
		switch action {
		case "create":
			return string(dashboard.ActivityUserCreated)
		case "update":
			return string(dashboard.ActivityUserUpdated)
		}
	case "form_submission":
		switch action {
		case "create":
			return string(dashboard.ActivityLeadReceived)
		case "process":
			return string(dashboard.ActivityLeadProcessed)
		}
	}
	return action
}

// Helper function to get entity description
func getEntityDescription(db *gorm.DB, entityType string, entityID uint) string {
	var description string

	switch entityType {
	case "report":
		db.Table("reports").Select("title").Where("id = ?", entityID).Scan(&description)
	case "blog":
		db.Table("blogs").Select("title").Where("id = ?", entityID).Scan(&description)
	case "press_release":
		db.Table("press_releases").Select("title").Where("id = ?", entityID).Scan(&description)
	case "user":
		db.Table("users").Select("name").Where("id = ?", entityID).Scan(&description)
	case "form_submission":
		db.Table("form_submissions").Select("category").Where("id = ?", entityID).Scan(&description)
	}

	return description
}
