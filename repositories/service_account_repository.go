package repositories

import (
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ServiceAccountRepository interface {
	Create(account *models.ServiceAccount) error
	FindAll() ([]models.ServiceAccount, error)
	FindBySubscriptionID(subscriptionID string) ([]models.ServiceAccount, error)
	FindByID(id uuid.UUID) (*models.ServiceAccount, error)
	Update(account *models.ServiceAccount) error
	Delete(id uuid.UUID) error
}

type serviceAccountRepository struct {
	db *gorm.DB
}

func NewServiceAccountRepository(db *gorm.DB) ServiceAccountRepository {
	return &serviceAccountRepository{db: db}
}

func (r *serviceAccountRepository) Create(account *models.ServiceAccount) error {
	return r.db.Create(account).Error
}

func (r *serviceAccountRepository) FindAll() ([]models.ServiceAccount, error) {
	var accounts []models.ServiceAccount
	err := r.db.
		Preload("Router").
		Preload("Subscription").
		Preload("Subscription.Customer").
		Preload("Subscription.Customer.User").
		Preload("NetworkPlan").
		Preload("NetworkPlan.Router").
		Order("created_at desc").
		Find(&accounts).Error
	return accounts, err
}

func (r *serviceAccountRepository) FindBySubscriptionID(subscriptionID string) ([]models.ServiceAccount, error) {
	var accounts []models.ServiceAccount
	err := r.db.
		Preload("Router").
		Preload("Subscription").
		Preload("Subscription.Customer").
		Preload("Subscription.Customer.User").
		Preload("NetworkPlan").
		Preload("NetworkPlan.Router").
		Where("subscription_id = ?", subscriptionID).
		Order("created_at desc").
		Find(&accounts).Error
	return accounts, err
}

func (r *serviceAccountRepository) FindByID(id uuid.UUID) (*models.ServiceAccount, error) {
	var account models.ServiceAccount
	err := r.db.
		Preload("Router").
		Preload("Subscription").
		Preload("Subscription.Customer").
		Preload("Subscription.Customer.User").
		Preload("NetworkPlan").
		Preload("NetworkPlan.Router").
		First(&account, "id = ?", id).Error
	return &account, err
}

func (r *serviceAccountRepository) Update(account *models.ServiceAccount) error {
	return r.db.Omit("Router", "Subscription", "NetworkPlan").Save(account).Error
}

func (r *serviceAccountRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.ServiceAccount{}, "id = ?", id).Error
}
