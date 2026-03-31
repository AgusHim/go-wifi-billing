package repositories

import (
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProvisioningLogRepository interface {
	Create(log *models.ProvisioningLog) error
	FindAll(limit int) ([]models.ProvisioningLog, error)
	FindByID(id uuid.UUID) (*models.ProvisioningLog, error)
}

type provisioningLogRepository struct {
	db *gorm.DB
}

func NewProvisioningLogRepository(db *gorm.DB) ProvisioningLogRepository {
	return &provisioningLogRepository{db: db}
}

func (r *provisioningLogRepository) Create(log *models.ProvisioningLog) error {
	return r.db.Create(log).Error
}

func (r *provisioningLogRepository) FindAll(limit int) ([]models.ProvisioningLog, error) {
	if limit <= 0 {
		limit = 100
	}
	var logs []models.ProvisioningLog
	err := r.db.Preload("Router").Preload("ProvisioningJob").Order("created_at desc").Limit(limit).Find(&logs).Error
	return logs, err
}

func (r *provisioningLogRepository) FindByID(id uuid.UUID) (*models.ProvisioningLog, error) {
	var log models.ProvisioningLog
	err := r.db.Preload("Router").Preload("ProvisioningJob").First(&log, "id = ?", id).Error
	return &log, err
}
