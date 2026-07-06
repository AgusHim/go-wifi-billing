package repositories

import (
	"time"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ProvisioningJobRepository interface {
	Create(job *models.ProvisioningJob) error
	FindAll(limit int) ([]models.ProvisioningJob, error)
	FindByID(id uuid.UUID) (*models.ProvisioningJob, error)
	Update(job *models.ProvisioningJob) error
	FindByEntity(entityType string, entityID uuid.UUID, limit int) ([]models.ProvisioningJob, error)
	FindEligibleForRetry(now time.Time, maxAttempts int, limit int) ([]models.ProvisioningJob, error)
	CountByStatus(status string) (int64, error)
}

type provisioningJobRepository struct {
	db *gorm.DB
}

func NewProvisioningJobRepository(db *gorm.DB) ProvisioningJobRepository {
	return &provisioningJobRepository{db: db}
}

func (r *provisioningJobRepository) Create(job *models.ProvisioningJob) error {
	return r.db.Create(job).Error
}

func (r *provisioningJobRepository) FindAll(limit int) ([]models.ProvisioningJob, error) {
	if limit <= 0 {
		limit = 50
	}
	var jobs []models.ProvisioningJob
	err := r.db.Order("created_at desc").Limit(limit).Find(&jobs).Error
	return jobs, err
}

func (r *provisioningJobRepository) FindByID(id uuid.UUID) (*models.ProvisioningJob, error) {
	var job models.ProvisioningJob
	err := r.db.First(&job, "id = ?", id).Error
	return &job, err
}

func (r *provisioningJobRepository) Update(job *models.ProvisioningJob) error {
	return r.db.Omit(clause.Associations).Save(job).Error
}

func (r *provisioningJobRepository) FindByEntity(entityType string, entityID uuid.UUID, limit int) ([]models.ProvisioningJob, error) {
	if limit <= 0 {
		limit = 20
	}
	var jobs []models.ProvisioningJob
	err := r.db.
		Where("entity_type = ? AND entity_id = ?", entityType, entityID).
		Order("created_at desc").
		Limit(limit).
		Find(&jobs).Error
	return jobs, err
}

func (r *provisioningJobRepository) FindEligibleForRetry(now time.Time, maxAttempts int, limit int) ([]models.ProvisioningJob, error) {
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	if limit <= 0 {
		limit = 25
	}
	var jobs []models.ProvisioningJob
	err := r.db.
		Where("attempt_count < ?", maxAttempts).
		Where(`
			(LOWER(status) = 'pending' AND (scheduled_at IS NULL OR scheduled_at <= ?))
			OR
			(LOWER(status) = 'failed' AND scheduled_at IS NOT NULL AND scheduled_at <= ?)
		`, now, now).
		Order("scheduled_at asc, created_at asc").
		Limit(limit).
		Find(&jobs).Error
	return jobs, err
}

func (r *provisioningJobRepository) CountByStatus(status string) (int64, error) {
	var total int64
	err := r.db.Model(&models.ProvisioningJob{}).
		Where("LOWER(status) = LOWER(?)", status).
		Count(&total).Error
	return total, err
}
