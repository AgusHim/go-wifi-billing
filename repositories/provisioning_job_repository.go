package repositories

import (
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProvisioningJobRepository interface {
	Create(job *models.ProvisioningJob) error
	FindAll(limit int) ([]models.ProvisioningJob, error)
	FindByID(id uuid.UUID) (*models.ProvisioningJob, error)
	Update(job *models.ProvisioningJob) error
	FindByEntity(entityType string, entityID uuid.UUID, limit int) ([]models.ProvisioningJob, error)
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
	return r.db.Save(job).Error
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
