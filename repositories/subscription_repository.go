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
	FindAutoRenewCandidates(threshold time.Time, renewalMode string) ([]models.Subscription, error)
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
		Preload("Package").
		Preload("NetworkPlan").
		Preload("NetworkPlan.Router").
		Preload("ServiceAccounts").
		Preload("ServiceAccounts.Router").
		Preload("RenewalHistories").
		Preload("RenewalHistories.Bill").
		Preload("RenewalHistories.Payment")

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
			Where("LOWER(users.name) LIKE LOWER(?) OR LOWER(users.email) LIKE LOWER(?)", searchPattern, searchPattern)
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
		now := time.Now()
		startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
		endOfMonth := startOfMonth.AddDate(0, 1, 0)
		query = query.Where("end_date >= ? AND end_date < ?", startOfMonth, endOfMonth)
	}
	err := query.
		Preload("Customer").
		Preload("Customer.User").
		Preload("Package").
		Preload("NetworkPlan").
		Preload("NetworkPlan.Router").
		Preload("ServiceAccounts").
		Preload("ServiceAccounts.Router").
		Preload("RenewalHistories").
		Preload("RenewalHistories.Bill").
		Preload("RenewalHistories.Payment").
		Find(&subscriptions).Error
	return subscriptions, err
}

func (r *subscriptionRepository) FindAutoRenewCandidates(threshold time.Time, renewalMode string) ([]models.Subscription, error) {
	var subscriptions []models.Subscription
	err := r.db.
		Where("auto_renew = ?", true).
		Where("LOWER(renewal_mode) = LOWER(?)", renewalMode).
		Where("LOWER(status) IN ?", []string{"active", "suspended"}).
		Where("end_date <= ?", threshold).
		Preload("Package").
		Preload("NetworkPlan").
		Preload("NetworkPlan.Router").
		Find(&subscriptions).Error
	return subscriptions, err
}

func (r *subscriptionRepository) FindByID(id uuid.UUID) (*models.Subscription, error) {
	var subscription models.Subscription
	err := r.db.
		Preload("Customer").
		Preload("Customer.User").
		Preload("Package").
		Preload("NetworkPlan").
		Preload("NetworkPlan.Router").
		Preload("ServiceAccounts").
		Preload("ServiceAccounts.Router").
		Preload("RenewalHistories").
		Preload("RenewalHistories.Bill").
		Preload("RenewalHistories.Payment").
		First(&subscription, "id = ?", id).Error
	return &subscription, err
}
func (r *subscriptionRepository) FindByCustomerID(customerID uuid.UUID) (*models.Subscription, error) {
	var subscription models.Subscription

	err := r.db.
		Preload("Customer").
		Preload("Customer.User").
		Preload("Package").
		Preload("NetworkPlan").
		Preload("NetworkPlan.Router").
		Preload("ServiceAccounts").
		Preload("ServiceAccounts.Router").
		Preload("RenewalHistories").
		Preload("RenewalHistories.Bill").
		Preload("RenewalHistories.Payment").
		Where("customer_id = ?", customerID).
		First(&subscription).Error

	if err != nil {
		return nil, err
	}

	return &subscription, nil
}

func (r *subscriptionRepository) Update(subscription *models.Subscription) error {
	return r.db.Omit("Customer", "Customer.User", "Package", "NetworkPlan", "RenewalHistories", "ServiceAccounts").Save(subscription).Error
}

func (r *subscriptionRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Subscription{}, "id = ?", id).Error
}
