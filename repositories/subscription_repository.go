package repositories

import (
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SubscriptionRepository interface {
	Create(subscription *models.Subscription) error
	FindAll() ([]models.Subscription, error)
	FindByID(id uuid.UUID) (*models.Subscription, error)
	Update(subscription *models.Subscription) error
	Delete(id uuid.UUID) error
}

type subscriptionRepository struct {
	db *gorm.DB
}

func NewSubscriptionRepository(db *gorm.DB) SubscriptionRepository {
	return &subscriptionRepository{db}
}

func (r *subscriptionRepository) Create(subscription *models.Subscription) error {
	return r.db.Create(subscription).Error
}

func (r *subscriptionRepository) FindAll() ([]models.Subscription, error) {
	var subscriptions []models.Subscription
	err := r.db.
		Preload("Customer").
		Preload("Package").
		Find(&subscriptions).Error
	return subscriptions, err
}

func (r *subscriptionRepository) FindByID(id uuid.UUID) (*models.Subscription, error) {
	var subscription models.Subscription
	err := r.db.
		Preload("Customer").
		Preload("Package").
		First(&subscription, "id = ?", id).Error
	return &subscription, err
}

func (r *subscriptionRepository) Update(subscription *models.Subscription) error {
	return r.db.Save(subscription).Error
}

func (r *subscriptionRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Subscription{}, "id = ?", id).Error
}
