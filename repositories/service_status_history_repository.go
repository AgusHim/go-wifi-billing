package repositories

import (
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ServiceStatusHistoryRepository interface {
	Create(item *models.ServiceStatusHistory) error
	FindByServiceAccountID(serviceAccountID uuid.UUID, limit int) ([]models.ServiceStatusHistory, error)
}

type serviceStatusHistoryRepository struct {
	db *gorm.DB
}

func NewServiceStatusHistoryRepository(db *gorm.DB) ServiceStatusHistoryRepository {
	return &serviceStatusHistoryRepository{db: db}
}

func (r *serviceStatusHistoryRepository) Create(item *models.ServiceStatusHistory) error {
	return r.db.Create(item).Error
}

func (r *serviceStatusHistoryRepository) FindByServiceAccountID(serviceAccountID uuid.UUID, limit int) ([]models.ServiceStatusHistory, error) {
	if limit <= 0 {
		limit = 50
	}
	var items []models.ServiceStatusHistory
	err := r.db.
		Preload("ProvisioningJob").
		Preload("Bill").
		Preload("Payment").
		Where("service_account_id = ?", serviceAccountID).
		Order("created_at desc").
		Limit(limit).
		Find(&items).Error
	return items, err
}
