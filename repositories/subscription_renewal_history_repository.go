package repositories

import (
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SubscriptionRenewalHistoryRepository interface {
	Create(item *models.SubscriptionRenewalHistory) error
	FindBySubscriptionID(subscriptionID uuid.UUID, limit int) ([]models.SubscriptionRenewalHistory, error)
}

type subscriptionRenewalHistoryRepository struct {
	db *gorm.DB
}

func NewSubscriptionRenewalHistoryRepository(db *gorm.DB) SubscriptionRenewalHistoryRepository {
	return &subscriptionRenewalHistoryRepository{db: db}
}

func (r *subscriptionRenewalHistoryRepository) Create(item *models.SubscriptionRenewalHistory) error {
	return r.db.Create(item).Error
}

func (r *subscriptionRenewalHistoryRepository) FindBySubscriptionID(subscriptionID uuid.UUID, limit int) ([]models.SubscriptionRenewalHistory, error) {
	if limit <= 0 {
		limit = 20
	}
	var items []models.SubscriptionRenewalHistory
	err := r.db.
		Preload("Bill").
		Preload("Payment").
		Where("subscription_id = ?", subscriptionID).
		Order("executed_at desc").
		Limit(limit).
		Find(&items).Error
	return items, err
}
