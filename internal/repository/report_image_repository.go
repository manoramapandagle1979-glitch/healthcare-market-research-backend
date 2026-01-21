package repository

import (
	"time"

	"github.com/healthcare-market-research/backend/internal/domain/report"
	"gorm.io/gorm"
)

type ReportImageRepository interface {
	Create(image *report.ReportImage) error
	FindByID(id uint) (*report.ReportImage, error)
	FindByReportID(reportID uint) ([]report.ReportImage, error)
	FindActiveByReportID(reportID uint) ([]report.ReportImage, error)
	Update(image *report.ReportImage) error
	SoftDelete(id uint) error
	CountByReportID(reportID uint) (int64, error)
	CountActiveByReportID(reportID uint) (int64, error)
}

type reportImageRepository struct {
	db *gorm.DB
}

func NewReportImageRepository(db *gorm.DB) ReportImageRepository {
	return &reportImageRepository{db: db}
}

func (r *reportImageRepository) Create(image *report.ReportImage) error {
	return r.db.Create(image).Error
}

func (r *reportImageRepository) FindByID(id uint) (*report.ReportImage, error) {
	var image report.ReportImage
	err := r.db.First(&image, id).Error
	if err != nil {
		return nil, err
	}
	return &image, nil
}

func (r *reportImageRepository) FindByReportID(reportID uint) ([]report.ReportImage, error) {
	var images []report.ReportImage
	err := r.db.Where("report_id = ?", reportID).
		Order("created_at DESC").
		Find(&images).Error
	return images, err
}

func (r *reportImageRepository) FindActiveByReportID(reportID uint) ([]report.ReportImage, error) {
	var images []report.ReportImage
	err := r.db.Where("report_id = ? AND is_active = ?", reportID, true).
		Order("created_at DESC").
		Find(&images).Error
	return images, err
}

func (r *reportImageRepository) Update(image *report.ReportImage) error {
	return r.db.Save(image).Error
}

func (r *reportImageRepository) SoftDelete(id uint) error {
	return r.db.Model(&report.ReportImage{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_active":  false,
			"updated_at": time.Now(),
		}).Error
}

func (r *reportImageRepository) CountByReportID(reportID uint) (int64, error) {
	var count int64
	err := r.db.Model(&report.ReportImage{}).
		Where("report_id = ?", reportID).
		Count(&count).Error
	return count, err
}

func (r *reportImageRepository) CountActiveByReportID(reportID uint) (int64, error) {
	var count int64
	err := r.db.Model(&report.ReportImage{}).
		Where("report_id = ? AND is_active = ?", reportID, true).
		Count(&count).Error
	return count, err
}
