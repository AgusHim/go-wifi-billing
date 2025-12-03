package repositories

import (
	"time"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SubscriptionRepository interface {
	Create(subscription *models.Subscription) error
	FindAll(page, limit int, search string, customerID *string, status *string) ([]models.Subscription, int64, error)
	FindForBill(customerID *string, status *string, isEndThisMonth bool) ([]models.Subscription, error)
	FindByID(id uuid.UUID) (*models.Subscription, error)
	FindByCustomerID(customerID uuid.UUID) (*models.Subscription, error)
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

func (r *subscriptionRepository) FindAll(page, limit int, search string, customerID *string, status *string) ([]models.Subscription, int64, error) {
	var (
		subscriptions []models.Subscription
		total         int64
	)

	query := r.db.Model(&models.Subscription{}).
		Preload("Customer").
		Preload("Customer.User").
		Preload("Package")

	// Filter by CustomerID
	if customerID != nil && *customerID != "" {
		query = query.Where("customer_id = ?", *customerID)
	}

	// Filter by Status
	if status != nil && *status != "" {
		query = query.Where("status = ?", *status)
	}

	// Search by related user name or email
	if search != "" {
		searchPattern := "%" + search + "%"
		query = query.Joins("JOIN customers ON customers.id = subscriptions.customer_id").
			Joins("JOIN users ON users.id = customers.user_id").
			Where("users.name ILIKE ? OR users.email ILIKE ?", searchPattern, searchPattern)
	}

	// Hitung total data (tanpa pagination)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Pagination
	offset := (page - 1) * limit
	if err := query.
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Find(&subscriptions).Error; err != nil {
		return nil, 0, err
	}

	return subscriptions, total, nil
}

func (r *subscriptionRepository) FindForBill(customerID *string, status *string, isEndThisMonth bool) ([]models.Subscription, error) {
	var subscriptions []models.Subscription
	query := r.db
	if customerID != nil && *customerID != "" {
		query = query.Where("customer_id = ?", customerID)
	}
	if status != nil && *status != "" {
		query = query.Where("status = ?", status)
	}
	if isEndThisMonth {
		currentMonth := int(time.Now().Month())
		currentYear := time.Now().Year()
		query = query.Where("EXTRACT(MONTH FROM end_date) = ? AND EXTRACT(YEAR FROM end_date) = ?", currentMonth, currentYear)
	}
	err := query.
		Preload("Customer").
		Preload("Customer.User").
		Preload("Package").
		Find(&subscriptions).Error
	return subscriptions, err
}

func (r *subscriptionRepository) FindByID(id uuid.UUID) (*models.Subscription, error) {
	var subscription models.Subscription
	err := r.db.
		Preload("Customer").
		Preload("Customer.User").
		Preload("Package").
		First(&subscription, "id = ?", id).Error
	return &subscription, err
}
func (r *subscriptionRepository) FindByCustomerID(customerID uuid.UUID) (*models.Subscription, error) {
	var subscription models.Subscription

	err := r.db.
		Preload("Customer").
		Preload("Customer.User").
		Preload("Package").
		Where("customer_id = ?", customerID).
		First(&subscription).Error

	if err != nil {
		return nil, err
	}

	return &subscription, nil
}

func (r *subscriptionRepository) Update(subscription *models.Subscription) error {
	return r.db.Omit("Customer", "Package").Save(subscription).Error
}

func (r *subscriptionRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Subscription{}, "id = ?", id).Error
}
